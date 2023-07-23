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
	// RandalBot guild gets GPT4
	if c.ChatID == "1092965539346907156" {
		return ai.LanguageModel_GPT4
	}

	//TODO: premium chats?
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

func (c *Chat) replaceMarkdownLinks(md string) string {
	re := regexp.MustCompile(`!?\]\((https?.*?)\)`)

	// Find all markdown links in the text
	matches := re.FindAllStringSubmatch(md, -1)

	// Replace markdown links with their URLs
	for _, match := range matches {
		if len(match) > 1 {
			md = regexp.MustCompile(`!?\[[^\]]+\]\(`+regexp.QuoteMeta(match[1])+`[\)]`).ReplaceAllString(md, match[1])
		}
	}

	return md
}
