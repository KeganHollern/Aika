package discordchat

import (
	"aika/actions/discord"
	"aika/actions/math"
	action_openai "aika/actions/openai"
	"aika/actions/web"
	"aika/actions/youtube"
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
	ChatID string // TODO: find a purpose ?
	Brain  *discordai.AIBrain
	S3     *storage.S3
	Cfg    *storage.Disk
	Mutex  sync.Mutex
}

func (c *Chat) getLanguageModel(senderID string, guildID string) ai.LanguageModel {
	// RandalBot guild gets GPT4
	if c.isSubscriber(guildID) {
		return ai.LanguageModel_GPT4
	}

	// admins
	if c.isAdmin(senderID) {
		return ai.LanguageModel_GPT4
	}

	//TODO: premium chats?
	return ai.LanguageModel_GPT35
}

func (c *Chat) isSubscriber(guildID string) bool {
	data, ok := c.Cfg.Get("subscribers")
	if !ok {
		return false // no admins configred at all
	}
	array, ok := data.([]interface{})
	if !ok {
		logrus.WithField("data", data).Warnln("invalid 'subscribers' format in config.yaml")
		return false
	}

	for _, v := range array {
		str, ok := v.(string)
		if !ok {
			logrus.WithField("data", v).Warnln("invalid 'subscribers' entry in config.yaml")
			continue
		}
		if str == guildID {
			return true
		}
	}
	return false
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
		participant := &ChatParticipant{User: mention}
		formatted = strings.ReplaceAll(formatted, participant.GetMentionString(), participant.GetDisplayName())
	}
	return formatted
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

func (c *Chat) getAvailableFunctions(
	s *discordgo.Session,
	m *discordgo.MessageCreate,
) []discordai.Function {
	functions := []discordai.Function{
		web.Function_GetWaifuCateogires,
		web.Function_GetWaifuNsfw,
		web.Function_GetWaifuSfw,
		web.Function_SearchWeb,
		youtube.Function_SearchYoutube,
		math.Function_GenRandomNumber,
		web.Function_GetAnime,
	}

	// openAI stuff
	oai := &action_openai.DallE{
		Client: c.Brain.OpenAI,
		S3:     c.S3,
	}
	functions = append(functions, oai.GetFunction_DallE())

	// youtube stuff
	yt := &youtube.Youtube{
		S3: c.S3,
	}
	functions = append(functions, yt.GetFunction_DownloadYoutube())

	// admin commands
	if c.isAdmin(m.Author.ID) {
		g := &discord.Guilds{
			Session: s,
		}
		functions = append(functions, g.GetFunction_ListGuilds())
	}

	//TODO: add more functions to this
	return functions
}
