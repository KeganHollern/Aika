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

	msgPipe := utils.NewStringPipe()

	group := errgroup.Group{}
	group.SetLimit(2)

	// writer routine will start reading in
	// openAI responses & return a final history
	group.Go(func() error {
		defer msgPipe.Close()

		new_hisory, err := chat.Brain.ProcessChunked(
			chat.Ctx,
			msgPipe,
			system,
			history,
			message,
			chat.getAvailableFunctions(s, m.Author),
			chat.getLanguageModel(m.Author.ID, ""),
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

		for {
			line, err := msgPipe.Read()
			if errors.Is(err, io.EOF) {
				break
			}

			// process line from AI
			if content != "" {
				content += "\n"
			}
			content += line
			content := chat.replaceMarkdownLinks(content)

			// if content is too large we can't
			// put it in a single message
			// so just keep processing chunks
			// we'll handle it elsewhere
			if len(content) > 2000 {
				continue
			}

			var msg *discordgo.Message
			if msgId == "" {
				msg, err = s.ChannelMessageSend(m.ChannelID, content)
				if err != nil {
					// TODO: ??? idk
					continue
				}
			} else {
				msg, err = s.ChannelMessageEdit(m.ChannelID, msgId, content)
				if err != nil {
					// TODO: ??? idk
					continue
				}
			}
			msgId = msg.ID
		}

		// content exceeded buffer
		// send full message as a file :)
		if len(content) > 2000 {
			s.ChannelFileSendWithMessage(m.ChannelID, "*response too long - sent as file*", "response.txt", strings.NewReader(content))
		}
		return nil
	})

	if err := group.Wait(); err != nil {
		logrus.WithError(err).Errorln("failed to send message")
	}

	chat.History = history

	res := history[len(history)-1]

	logrus.
		WithField("sender", sender.GetDisplayName()).
		WithField("message", msg).
		WithField("response", res.Content).
		Infoln("chat log")
}
