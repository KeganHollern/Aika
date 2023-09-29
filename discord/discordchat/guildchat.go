package discordchat

import (
	"aika/utils"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

type Guild struct {
	Chat
	// chat history
	History map[string][]openai.ChatCompletionMessage
}

func (chat *Guild) OnMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	locked := chat.Mutex.TryLock()
	if !locked {
		s.ChannelMessageSendReply(m.ChannelID, "rate limit", m.Reference())
		return
	}
	defer chat.Mutex.Unlock()

	msg := chat.formatUsers(m.Content, m.Mentions)

	members, err := chat.getChatMembers(s, m.ChannelID)
	if err != nil {
		logrus.WithError(err).Errorln("failed to get chat members")
		s.ChannelMessageSend(m.ChannelID, err.Error())
		return
	}
	memberNames := []string{}
	memberMentions := []string{}
	for _, member := range members {
		memberNames = append(memberNames, member.GetDisplayName())
		memberMentions = append(memberMentions, member.GetMentionString())
	}

	sender := &ChatParticipant{User: m.Author}

	system := chat.Brain.BuildSystemMessage(memberNames, memberMentions)
	history := chat.getHistory(m.ChannelID)
	message := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: msg,
		Name:    sender.GetDisplayName(),
	}

	s.ChannelTyping(m.ChannelID)

	// todo: write a wrapper around
	// this that fullfills the same
	// interface as utils.NewStringPipe()
	pipe := utils.NewBytePipe()

	group := errgroup.Group{}
	group.SetLimit(2)

	// writer routine will start reading in
	// openAI responses & return a final history
	group.Go(func() error {
		defer pipe.Close()

		new_hisory, err := chat.Brain.ProcessChunked(
			chat.Ctx,
			pipe,
			system,
			history,
			message,
			chat.getAvailableFunctions(s, m.Author),
			chat.getLanguageModel(m.Author.ID, m.GuildID),
			chat.getInternalArgs(s, m.Author, m.GuildID, m.ChannelID),
		)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, err.Error())
			return fmt.Errorf("failed while processing in brain; %w", err)
		}
		if len(new_hisory) == 0 {
			s.ChannelMessageSend(m.ChannelID, "my brain is empty")
			return errors.New("blank history returned from brain.Process")
		}

		// update history
		history = new_hisory

		return nil
	})
	// reader routine will create & continuously edit
	// the discord message with content
	// as it's streamed in
	group.Go(func() error {
		// process chunks into a message
		content := ""
		msgId := ""

		buffer := make([]byte, 255)

		rl := rate.NewLimiter(1, 1)

		for {
			n, err := pipe.Read(buffer)
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return fmt.Errorf("failed to read pipe; %w", err)
			}
			if n == 0 {
				continue // no new data
			}
			line := string(buffer[:n])

			content += line
			content = chat.replaceMarkdownLinks(content)

			// if content is too large we can't
			// put it in a single message
			// so just keep processing chunks
			// we'll handle it elsewhere
			if len(content) > 2000 {
				continue
			}

			// discord throttles our requests if we make them too fast
			// this will ensure we don't make them too fast
			// writes are buffered so this won't slow down OpenAI response
			if !rl.Allow() {
				continue
			}

			var msg *discordgo.Message
			if msgId == "" {
				msg, err = s.ChannelMessageSend(m.ChannelID, content)
				if err != nil {
					logrus.WithError(err).Errorln("failed to send message")
					continue
				}
			} else {
				msg, err = s.ChannelMessageEdit(m.ChannelID, msgId, content)
				if err != nil {
					logrus.WithError(err).Errorln("failed to update message")
					continue
				}
			}
			msgId = msg.ID
		}

		if len(content) > 2000 {
			s.ChannelFileSendWithMessage(m.ChannelID, "*response too long - sent as file*", "response.txt", strings.NewReader(content))
		} else {
			if msgId == "" {
				s.ChannelMessageSend(m.ChannelID, content)
			} else {
				s.ChannelMessageEdit(m.ChannelID, msgId, content)
			}
		}
		return nil
	})

	if err := group.Wait(); err != nil {
		logrus.WithError(err).Errorln("failed to send message")
	}

	chat.setHistory(m.ChannelID, history)

	res := history[len(history)-1]

	// TODO: improve this log
	logrus.
		WithField("sender", sender.GetDisplayName()).
		WithField("message", msg).
		WithField("response", res.Content).
		Infoln("chat log")
}

func (chat *Guild) getHistory(channel string) []openai.ChatCompletionMessage {
	return chat.History[channel]
}
func (chat *Guild) setHistory(channel string, history []openai.ChatCompletionMessage) {
	chat.History[channel] = history
}

func (chat *Guild) getChatMembers(s *discordgo.Session, channel string) ([]*ChatParticipant, error) {

	participants := []*ChatParticipant{}

	gd, err := s.State.Guild(chat.ChatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild details; %w", err)
	}

	for _, member := range gd.Members {
		// aika can't see other bots (or herself)
		if member.User.Bot {
			continue
		}

		presence, err := s.State.Presence(chat.ChatID, member.User.ID)
		if errors.Is(err, discordgo.ErrStateNotFound) {
			continue // user likely offline or some shit
		}
		if err != nil {
			logrus.
				WithError(err).
				WithField("username", member.User.Username).
				WithField("userid", member.User.ID).
				Warnln("failed to get presence")
			continue
		}

		// ??? what the fuck?
		// we don't get presence info when they're offline so what the fuck?
		if presence.Status == discordgo.StatusOffline ||
			presence.Status == discordgo.StatusInvisible {
			continue
		}

		// filter people who can't view channel
		perms, err := s.State.UserChannelPermissions(member.User.ID, channel)
		if err != nil {
			logrus.
				WithError(err).
				WithField("username", member.User.Username).
				WithField("userid", member.User.ID).
				Warnln("failed to get user permissions")
		}
		if perms&discordgo.PermissionViewChannel != discordgo.PermissionViewChannel {
			continue
		}

		participants = append(participants, &ChatParticipant{User: member.User})
	}

	return participants, nil
}
