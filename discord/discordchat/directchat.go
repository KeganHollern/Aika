package discordchat

import (
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
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

	var err error

	s.ChannelTyping(m.ChannelID)

	history, err = chat.Brain.Process(
		chat.Ctx,
		system,
		history,
		message,
		chat.getAvailableFunctions(s, m.Author),
		chat.getLanguageModel(m.Author.ID, ""),
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

	chat.History = history

	res := history[len(history)-1]

	// TODO: improve this log
	logrus.
		WithField("sender", sender.GetDisplayName()).
		WithField("message", msg).
		WithField("response", res.Content).
		Infoln("chat log")

	response := chat.replaceMarkdownLinks(res.Content)
	if len(response) > 2000 {
		// TODO: do we need this fixed msg ?
		s.ChannelFileSendWithMessage(m.ChannelID, "*response too long - sent as file*", "response.txt", strings.NewReader(response))
	} else {
		s.ChannelMessageSend(m.ChannelID, response)
	}
}
