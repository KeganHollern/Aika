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
	"aika/voice"
	"context"
	"errors"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

//var global_voice_functions *discord.Voice

type Chat struct {
	Ctx    context.Context
	ChatID string // unique identifier for this chat (DM/Guild/ect)
	Brain  *discordai.AIBrain
	S3     *storage.S3
	Cfg    *storage.Disk
	Mutex  sync.Mutex

	// internal voice chat connection for this
	voice *Voice
}

func (c *Chat) getInternalArgs(
	s *discordgo.Session,
	user *discordgo.User,
	guildid string,
	channelid string,
) map[string]interface{} {

	// get authors voice channel
	voiceChannel := ""
	state, err := s.State.VoiceState(guildid, user.ID)
	if err != nil && !errors.Is(err, discordgo.ErrStateNotFound) {
		logrus.WithError(err).Errorln("failed to get sender voice state")
	} else if err == nil {
		voiceChannel = state.ChannelID
	}

	// attach discord information from sender
	return map[string]interface{}{
		"internal_sender_guildid":   guildid,
		"internal_sender_channelid": channelid,
		"internal_sender_author_id": guildid,
		"internal_sender_author_vc": voiceChannel,
	}
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
	user *discordgo.User,
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
	if c.isAdmin(user.ID) {
		g := &discord.Guilds{
			Session: s,
		}
		functions = append(functions, g.GetFunction_ListGuilds())
	}

	//-- add functions to tell aika to leave/join voice chat
	if c.voice != nil {
		// admin OR subscriber
		// chatID is always guildID when voice is non nil
		// this is implicit and retarded but o wel
		if c.isAdmin(user.ID) || c.isSubscriber(c.ChatID) {
			functions = append(functions, c.voice.GetFunction_JoinChannel())
			functions = append(functions, c.voice.GetFunction_LeaveChannel())
		}
	}

	//TODO: add more functions to this
	return functions
}

// support a voice chat connection
// only supports GUILD chats (afaik)
func (chat *Chat) InitVoiceChat(s *discordgo.Session) {
	chat.voice = &Voice{
		Chat: Chat{
			Ctx:    chat.Ctx,
			ChatID: chat.ChatID,
			Mutex:  sync.Mutex{},
			Brain:  chat.Brain,
			S3:     chat.S3,
			Cfg:    chat.Cfg,
		},
		History:    make([]openai.ChatCompletionMessage, 0),
		SsrcUsers:  make(map[uint32]string),
		Connection: nil,
		Session:    s, // ???? TODO can we get rid of this?

		Speaker: &voice.ElevenLabs{
			ApiKey: os.Getenv("ELEVENLABS_APIKEY"),
		},

		// google free-to-use TTS
		// Speaker: &voice.Google{},
	}
}
