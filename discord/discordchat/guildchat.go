package discordchat

import (
	"github.com/bwmarrin/discordgo"
	"github.com/sashabaranov/go-openai"
)

type Guild struct {
	Chat

	// chat history
	History   []openai.ChatCompletionMessage
	Functions []openai.FunctionDefinition
}

func (chat *Guild) OnMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	locked := chat.Mutex.TryLock()
	if !locked {
		s.ChannelMessageSendReply(m.ChannelID, "rate limit", m.Reference())
		return
	}
	defer chat.Mutex.Unlock()

	//	history := chat.getHistory(m.ChannelID)
	// 	model := chat.getLanguageModel()

	response := chat.Brain.DummyRequest(chat.Ctx)

	s.ChannelMessageSend(m.ChannelID, response)
}

func (chat *Guild) getHistory(channel string) []openai.ChatCompletionMessage {
	return chat.History
}
