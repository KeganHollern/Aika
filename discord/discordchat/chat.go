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

	// internal command structers
	actions chatActions
}

type chatActions struct {
	downloader *youtube.Downloader
	player     *youtube.Player
	dalle      *action_openai.DallE
	vision     *action_openai.Vision
	guilds     *discord.Guilds
}

// initializes chatActions
// kinda scuffed but this'll help ensure actions
// with structs maintain some data between
// api calls / chats
func (c *Chat) initActions(s *discordgo.Session) {
	if c.actions.dalle == nil {
		c.actions.dalle = &action_openai.DallE{
			Client: c.Brain.OpenAI,
			S3:     c.S3,
		}
	}
	if c.actions.vision == nil {
		c.actions.vision = &action_openai.Vision{
			Client: c.Brain.OpenAI,
		}
	}

	if c.actions.downloader == nil {
		c.actions.downloader = &youtube.Downloader{
			S3: c.S3,
		}
	}

	if c.actions.guilds == nil && s != nil {
		c.actions.guilds = &discord.Guilds{
			Session: s,
		}
	}

	// if voice is enabled init the player actions
	if c.voice != nil && c.actions.player == nil {
		c.actions.player = &youtube.Player{
			Mixer: c.voice.Mixer,
		}
	}

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
	// premium chats get GPT4
	if c.isSubscriber(guildID) {
		return ai.LanguageModel_GPT4T
	}

	// admins
	if c.isAdmin(senderID) {
		return ai.LanguageModel_GPT4T
	}

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

	// initialize any uninitialized actions
	c.initActions(s)

	// add action functions
	functions = append(functions, c.actions.vision.GetFunction_DescribeImage())
	functions = append(functions, c.actions.dalle.GetFunction_DallE())
	functions = append(functions, c.actions.downloader.GetFunction_SaveYoutube())

	// admin commands
	if c.isAdmin(user.ID) {
		functions = append(functions, c.actions.guilds.GetFunction_ListGuilds())
	}

	//-- add functions to tell aika to leave/join voice chat

	// this is non-nill when C is a voice chat or has a voice chat associated
	// if c is a voice chat then c.voice == c
	if c.voice != nil {
		// idea of how to check if this command
		// is coming from a voice speaker / voice chat
		/* if c.voice == c {
			// this is a voice chat
		} */

		functions = append(functions, c.actions.player.GetFunction_PlayAudio())

		// admin OR subscriber
		// chatID is always guildID when voice is non nil
		// this is implicit and retarded but o wel
		if c.isAdmin(user.ID) || c.isSubscriber(c.ChatID) {
			functions = append(functions, c.voice.GetFunction_JoinChannel())
			functions = append(functions, c.voice.GetFunction_LeaveChannel())
			functions = append(functions, c.voice.GetFunction_GetVoices())
			functions = append(functions, c.voice.GetFunction_SetVoice())
		}
	}

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
			ApiKey:  os.Getenv("ELEVENLABS_APIKEY"),
			VoiceID: "BreKkXSwy4hr1vgm7ZqX",
		},

		// google free-to-use TTS
		// Speaker: &voice.Google{},
	}

	// this is scuffed but it will ensure that the _voice_ chat points to itself
	// this will make voice chat functions available to both
	// the source chat and the voice channel
	chat.voice.voice = chat.voice
}
