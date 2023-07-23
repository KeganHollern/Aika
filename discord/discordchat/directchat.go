package discordchat

import (
	"github.com/bwmarrin/discordgo"
	"github.com/sashabaranov/go-openai"
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

	//	history := chat.History
	// 	model := chat.getLanguageModel()

	response := chat.Brain.DummyRequest(chat.Ctx)

	s.ChannelMessageSend(m.ChannelID, response)
}
