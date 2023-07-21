package discord

import (
	"aika/aika"
	"aika/premium"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

type AikaBot struct {
	ApiKey    string
	API       aika.OpenAI
	PremiumDB *premium.Servers

	channels map[string]*aika.Chat
}

func (bot *AikaBot) Start(wg *sync.WaitGroup) error {
	wg.Add(1)

	// todo: monitor for shutdown or something idk
	dg, err := discordgo.New("Bot " + bot.ApiKey)
	if err != nil {
		return fmt.Errorf("failed to init discord bot; %w", err)
	}

	dg.AddHandler(bot.messageCreate)

	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.IntentsAll

	dg.StateEnabled = true

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		return fmt.Errorf("error opening connection; %w", err)
	}

	// log all guild names
	for _, rootGd := range dg.State.Guilds {
		gd, err := dg.Guild(rootGd.ID)
		if err != nil {
			logrus.
				WithError(err).
				WithField("id", rootGd.ID).
				Warnln("could not find guild data")
		} else {
			logrus.
				WithField("name", gd.Name).
				WithField("id", gd.ID).
				WithField("premium", bot.PremiumDB.IsPremium(gd.ID)).
				Info("in guild")
		}
	}

	go func() {
		fmt.Println("Aika is now running.  Press CTRL-C to exit.")
		sc := make(chan os.Signal, 1)
		signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
		<-sc

		logrus.Info("SHUTDOWN STARTED....")

		// Cleanly close down the Discord session.
		dg.Close()
		wg.Done()
	}()
	return nil
}

func (bot *AikaBot) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages from bots (including itself)
	if m.Author.Bot {
		return
	}

	// ignore all messages not mentioning Aika (if they're in guilds)
	if m.GuildID != "" {
		hasMention := false
		for _, mention := range m.Mentions {
			if mention.ID == s.State.User.ID {
				hasMention = true
				break
			}
		}
		if !hasMention {
			return
		}
	}

	logrus.WithField("msg", m.Content).Debugln("DISCORD")

	if bot.channels == nil {
		bot.channels = make(map[string]*aika.Chat)
	}

	participants := []string{}

	// probably not needed?
	/*
		err := s.RequestGuildMembers(m.GuildID, "", 0, "0", true) // update all presense of guild members
		if err != nil {
			logrus.WithError(err).Errorln("failed to request guild member update")
			return
		}
	*/

	chat_name := fmt.Sprintf("DM @%s", m.Author.Username)

	if m.GuildID != "" {
		gd, err := s.State.Guild(m.GuildID)
		if err != nil {
			logrus.WithError(err).WithField("guild_id", m.GuildID).Errorln("failed to get guild from state")
			return
		}
		chat_name = fmt.Sprintf("Guild: %s [%s]", gd.Name, gd.ID)

		for _, member := range gd.Members {
			// aika can't see other bots (only herself)
			if member.User.Bot &&
				member.User.ID != s.State.User.ID {
				continue
			}

			presence, err := s.State.Presence(m.GuildID, member.User.ID)
			if errors.Is(err, discordgo.ErrStateNotFound) {
				continue // user likely offline or some shit
			}
			if err != nil {
				logrus.
					WithError(err).
					WithField("username", member.User.Username).
					WithField("userid", member.User.ID).
					Warnln("failed to get presence")
				continue
			}

			// ??? what the fuck?
			// we don't get presence info when they're offline so what the fuck?
			if presence.Status == discordgo.StatusOffline ||
				presence.Status == discordgo.StatusInvisible {
				continue
			}

			// TODO: filter for channel permissions
			perms, err := s.State.UserChannelPermissions(member.User.ID, m.ChannelID)
			if err != nil {
				logrus.
					WithError(err).
					WithField("username", member.User.Username).
					WithField("userid", member.User.ID).
					Warnln("failed to get user permissions")
			}
			logrus.
				WithField("username", member.User.Username).
				WithField("userid", member.User.ID).
				WithField("permissions", perms).
				Debugln("user channel permission value")

			participants = append(participants, member.User.Username)
		}
	} else {
		participants = []string{m.Author.Username, s.State.User.Username}
	}

	chat, exists := bot.channels[m.ChannelID]
	if !exists {
		chat = &aika.Chat{
			CTX: aika.ChatContext{
				PremiumDB: bot.PremiumDB,
				CID:       m.ChannelID,
				GID:       m.GuildID,
			},
			API:     bot.API,
			Members: participants, // TODO: channel participant usernames
			History: []openai.ChatCompletionMessage{},
			Mutex:   sync.Mutex{},
		}
		bot.channels[m.ChannelID] = chat
	} else {
		chat.Members = participants // update participants
	}

	// clean mentions
	msgToSend := m.Content
	for _, mention := range m.Mentions {
		msgToSend = strings.ReplaceAll(msgToSend, "<@"+mention.ID+">", mention.Username)
	}

	// send typing
	s.ChannelTyping(m.ChannelID)
	response, err := chat.Send(m.Author.Username, msgToSend)

	logrus.
		WithField("sender", m.Author.Username).
		WithField("chat", chat_name).
		WithField("message", msgToSend).
		WithField("response", response).
		Info("conversation log")

	if err != nil {
		if errors.Is(err, aika.ErrChatInUse) {
			s.ChannelMessageSendReply(m.ChannelID, "I am busy with another request. Please try again later.", m.Reference())
			return
		}
		s.ChannelMessageSend(m.ChannelID, fmt.Errorf("sorry aika failed to respond; %w", err).Error())
	}
	if len(response) > 2000 {
		s.ChannelFileSendWithMessage(m.ChannelID, "*response too long - sent as file*", "response.txt", strings.NewReader(response))
	} else {
		s.ChannelMessageSend(m.ChannelID, response)
	}
}
