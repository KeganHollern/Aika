package discordchat

import (
	"github.com/bwmarrin/discordgo"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

type Direct struct {
	Chat

	History   []openai.ChatCompletionMessage
	Functions []openai.FunctionDefinition
}

func (chat *Direct) OnMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	locked := chat.Mutex.TryLock()
	if !locked {
		s.ChannelMessageSendReply(m.ChannelID, "rate limit", m.Reference())
		return
	}
	defer chat.Mutex.Unlock()

	msg := chat.formatUsers(m.Content, m.Mentions)

	system := chat.Brain.BuildSystemMessage([]string{m.Author.Username, "Aika"})
	history := chat.History
	message := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: msg, // TODO: prefix <USER>: <MSG> ?
		Name:    chat.cleanUserName(m.Author.Username),
	}

	var err error

	history, err = chat.Brain.Process(
		chat.Ctx,
		system,
		history,
		message,
		chat.Functions,
		chat.getLanguageModel(),
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
		WithField("message", msg).
		WithField("response", res.Content).
		Infoln("chat log")

	// TODO: parse out weird openAI markdown
	s.ChannelMessageSend(m.ChannelID, res.Content)
}
