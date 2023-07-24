package discordchat

import (
	"aika/ai"
	"aika/discord/discordai"
	"aika/storage"
	"context"
	"regexp"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

type Chat struct {
	Ctx    context.Context
	ChatID string
	Brain  *discordai.AIBrain
	S3     *storage.S3
	Cfg    *storage.Disk
	Mutex  sync.Mutex
}

// TODO: need to feed this function channel
// and guild IDs so the model can be determined better
func (c *Chat) getLanguageModel() ai.LanguageModel {
	// RandalBot guild gets GPT4
	if c.ChatID == "1092965539346907156" {
		return ai.LanguageModel_GPT4
	}
	// kegan DM channel
	if c.ChatID == "1132494588330901524" {
		return ai.LanguageModel_GPT4
	}

	//TODO: premium chats?
	return ai.LanguageModel_GPT35
}

// isAdmin reads "admins" from the config file
// if the provided user ID is in the list it returns true
func (c *Chat) isAdmin(userID string) bool {
	data, ok := c.Cfg.Get("admins")
	if !ok {
		return false // no admins configred at all
	}
	array, ok := data.([]interface{})
	if !ok {
		logrus.WithField("data", data).Warnln("invalid 'admins' format in config.yaml")
		return false
	}

	for _, v := range array {
		str, ok := v.(string)
		if !ok {
			logrus.WithField("data", v).Warnln("invalid 'admins' entry in config.yaml")
			continue
		}
		if str == userID {
			return true
		}
	}
	return false
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
