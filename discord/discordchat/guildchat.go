package discordchat

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
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

	history, err = chat.Brain.Process(
		chat.Ctx,
		system,
		history,
		message,
		chat.getAvailableFunctions(s, m.Author),
		chat.getLanguageModel(m.Author.ID, m.GuildID),
		chat.getInternalArgs(s, m.Author, m.GuildID, m.ChannelID),
	)
	if err != nil {
		logrus.WithError(err).Errorln("failed while processing in brain")
		s.ChannelMessageSend(m.ChannelID, err.Error())
		return
	}

	if len(history) == 0 {
		logrus.Errorln("blank history returned from brain.Process")
		s.ChannelMessageSend(m.ChannelID, "my brain is empty")
		return
	}

	chat.setHistory(m.ChannelID, history)

	res := history[len(history)-1]

	// TODO: improve this log
	logrus.
		WithField("sender", sender.GetDisplayName()).
		WithField("message", msg).
		WithField("response", res.Content).
		Infoln("chat log")

	response := chat.replaceMarkdownLinks(res.Content)
	if len(response) > 2000 {
		s.ChannelFileSendWithMessage(m.ChannelID, "*response too long - sent as file*", "response.txt", strings.NewReader(response))
	} else {
		s.ChannelMessageSend(m.ChannelID, response)
	}
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
