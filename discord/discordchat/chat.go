package discordchat

import (
	"aika/ai"
	"aika/discord/discordai"
	"context"
	"regexp"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
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

func (c *Chat) formatUsers(message string, users []*discordgo.User) string {
	formatted := message
	for _, mention := range users {
		formatted = strings.ReplaceAll(formatted, "<@"+mention.ID+">", c.cleanUserName(mention.Username))
	}
	return formatted
}

func (c *Chat) cleanUserName(input string) string {
	// This regular expression matches any character that is not a letter or a number
	reg := regexp.MustCompile("[^a-zA-Z0-9]+")
	processedString := reg.ReplaceAllString(input, "")
	return processedString
}
