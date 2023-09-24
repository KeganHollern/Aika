package discord

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"

	"aika/discord/discordai"
	"aika/discord/discordchat"
	"aika/storage"
)

var (
	ErrInvalidHistoryConfiguration   = errors.New("invalid history configuration value")
	ErrInvalidCharacterConfiguration = errors.New("invalid character configuration value")
)

type ChatBot struct {
	Ctx         context.Context
	Session     *discordgo.Session
	Brain       *discordai.AIBrain
	GuildChats  map[string]*discordchat.Guild
	DirectChats map[string]*discordchat.Direct

	S3  *storage.S3
	Cfg *storage.Disk
}

func StartChatbot(
	ctx context.Context,
	apiKey string,
	client *openai.Client,
	s3 *storage.S3,
	cfg *storage.Disk,
) (*ChatBot, error) {
	// create session object
	dg, err := discordgo.New("Bot " + apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to start session; %w", err)
	}

	// extract configuration
	historySize, ok := cfg.Get("history")
	if !ok {
		return nil, ErrInvalidHistoryConfiguration
	}

	historyLen, ok := historySize.(int)
	if !ok {
		return nil, ErrInvalidHistoryConfiguration
	}

	if historyLen <= 0 {
		return nil, ErrInvalidHistoryConfiguration
	}

	characterCfg, ok := cfg.Get("character")
	if !ok {
		return nil, ErrInvalidCharacterConfiguration
	}
	character, ok := characterCfg.(string)
	if !ok {
		return nil, ErrInvalidCharacterConfiguration
	}

	// create bot object
	bot := &ChatBot{
		Ctx:     ctx,
		Session: dg,
		Brain: &discordai.AIBrain{
			OpenAI:      client,
			HistorySize: historyLen,
			Character:   character,
		},
		GuildChats:  make(map[string]*discordchat.Guild),
		DirectChats: make(map[string]*discordchat.Direct),
		S3:          s3,
		Cfg:         cfg,
	}

	// add OnMessage handler
	dg.AddHandler(bot.onMessage)

	// intents & enable state tracking
	dg.Identify.Intents = discordgo.IntentsAll
	dg.StateEnabled = true

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		return nil, fmt.Errorf("error opening connection; %w", err)
	}

	// wait for ctx done to close discord connection safely
	go func() {
		<-ctx.Done()

		err := dg.Close()
		if err != nil {
			logrus.WithError(err).Errorln("failed to close discord connection")
		}
	}()

	return bot, nil
}

// onMessage handles when a message is recieved
func (bot *ChatBot) onMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages from bots (including itself)
	if m.Author.Bot {
		return
	}

	if m.GuildID == "" {
		// direct message
		dchat, exists := bot.DirectChats[m.ChannelID]
		if !exists {
			dchat = bot.newDirectChat(m.ChannelID)
			bot.DirectChats[m.ChannelID] = dchat
		}
		dchat.OnMessage(s, m)
		return
	}

	// ignore all messages not mentioning Aika (if they're in guilds)
	hasMention := false

	for _, mention := range m.Mentions {
		//TODO: this doesn't work if a user "copies" thei @aika from another message
		// why trhe fuck is discord like this?
		// what the fuck when I copy @Aika i get a MENTIONROLES but not a Mention ?!
		// NOTE: if you copy an @ in discord
		//		you can't click the @ and see their profile
		//		this is a discord fuckery
		if mention.ID == s.State.User.ID {
			hasMention = true
			break
		}
	}
	if !hasMention {
		return
	}

	// guild message
	gchat, exists := bot.GuildChats[m.ChannelID]
	if !exists {
		gchat = bot.newGuildChat(m.GuildID)
		bot.GuildChats[m.ChannelID] = gchat
	}
	gchat.OnMessage(s, m)
}

// --- chat constructors

func (bot *ChatBot) newGuildChat(guildId string) *discordchat.Guild {
	chat := &discordchat.Guild{
		Chat: discordchat.Chat{
			Ctx:    bot.Ctx,
			ChatID: guildId,
			Mutex:  sync.Mutex{},
			Brain:  bot.Brain,
			S3:     bot.S3,
			Cfg:    bot.Cfg,
		},
		History: make(map[string][]openai.ChatCompletionMessage),
	}
	// enable voice chat for this guild
	// TODO: setting for this so i can monetize ?
	chat.InitVoiceChat(bot.Session)

	return chat
}

func (bot *ChatBot) newDirectChat(channelId string) *discordchat.Direct {
	return &discordchat.Direct{
		Chat: discordchat.Chat{
			Ctx:    bot.Ctx,
			ChatID: channelId,
			Mutex:  sync.Mutex{},
			Brain:  bot.Brain,
			S3:     bot.S3,
			Cfg:    bot.Cfg,
		},
		History: []openai.ChatCompletionMessage{},
	}
}
