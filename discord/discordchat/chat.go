package discordchat

import (
	"aika/ai"
	"aika/discord/discordai"
	"context"
	"sync"
)

type Chat struct {
	Ctx    context.Context
	ChatID string
	Brain  *discordai.AIBrain
	Mutex  sync.Mutex
}

func (c *Chat) getLanguageModel() ai.LanguageModel {
	return ai.LanguageModel_GPT35
}
