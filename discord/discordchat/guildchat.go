package discordchat

import (
	"github.com/bwmarrin/discordgo"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

type Guild struct {
	Chat

	// chat history
	History   map[string][]openai.ChatCompletionMessage
	Functions []openai.FunctionDefinition
}

func (chat *Guild) OnMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	locked := chat.Mutex.TryLock()
	if !locked {
		s.ChannelMessageSendReply(m.ChannelID, "rate limit", m.Reference())
		return
	}
	defer chat.Mutex.Unlock()

	// TODO: get member list
	system := chat.Brain.BuildSystemMessage([]string{"Kegan", "Aika"})
	history := chat.getHistory(m.ChannelID)
	message := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: m.Content, // TODO: prefix <USER>: <MSG> ?
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

	chat.setHistory(m.ChannelID, history)

	res := history[len(history)-1]

	// TODO: improve this log
	logrus.
		WithField("message", m.Content).
		WithField("response", res.Content).
		Infoln("chat log")

	// TODO: parse out weird openAI markdown
	s.ChannelMessageSend(m.ChannelID, res.Content)
}

func (chat *Guild) getHistory(channel string) []openai.ChatCompletionMessage {
	return chat.History[channel]
}
func (chat *Guild) setHistory(channel string, history []openai.ChatCompletionMessage) {
	chat.History[channel] = history
}
