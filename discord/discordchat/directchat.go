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

type Direct struct {
	Chat

	History []openai.ChatCompletionMessage
}

func (chat *Direct) OnMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	locked := chat.Mutex.TryLock()
	if !locked {
		s.ChannelMessageSendReply(m.ChannelID, "rate limit", m.Reference())
		return
	}
	defer chat.Mutex.Unlock()

	msg := chat.formatUsers(m.Content, m.Mentions)

	sender := &ChatParticipant{User: m.Author}

	system := chat.Brain.BuildSystemMessage([]string{sender.GetDisplayName()}, []string{sender.GetMentionString()})
	history := chat.History
	message := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: msg,
		Name:    sender.GetDisplayName(),
	}

	s.ChannelTyping(m.ChannelID)

	// todo: write a wrapper around
	// this that fullfills the same
	// interface as utils.NewStringPipe()
	// basically I want a single `Interface` that both of these can fullfil
	pipe := utils.NewBytePipe()

	//msgPipe := utils.NewStringPipe()

	group := errgroup.Group{}
	group.SetLimit(2)

	// writer routine will start reading in
	// openAI responses & return a final history
	group.Go(func() error {
		defer pipe.Close()

		new_history, err := chat.Brain.ProcessChunked(
			chat.Ctx,
			pipe,
			system,
			history,
			message,
			chat.getAvailableFunctions(s, m.Author),
			chat.getLanguageModel(m.Author.ID, ""),
			chat.getInternalArgs(s, m.Author, m.GuildID, m.ChannelID),
		)
		if err != nil {
			return fmt.Errorf("failed while processing in brain; %w", err)
		}
		if len(new_history) == 0 {
			return errors.New("blank history returned from brain.Process")
		}

		// update history
		history = new_history

		logrus.WithField("full_msg", history[len(history)-1].Content).Debugln("finished reading stream")
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

		// content exceeded buffer
		// send full message as a file :)
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
		s.ChannelMessageSend(m.ChannelID, err.Error())
		return
	}

	chat.History = history

	res := history[len(history)-1]

	// TODO: improve this log
	logrus.
		WithField("sender", sender.GetDisplayName()).
		WithField("message", msg).
		WithField("response", res.Content).
		Infoln("chat log")
}
