package discord

import (
	"aika/aika"
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
	ApiKey string
	API    aika.OpenAI

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
	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		return fmt.Errorf("error opening connection; %w", err)
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

	chat, exists := bot.channels[m.ChannelID]
	if !exists {
		chat = &aika.Chat{
			API:     bot.API,
			Members: []string{}, // TODO: channel participant usernames
			History: []openai.ChatCompletionMessage{},
			Mutex:   sync.Mutex{},
		}
		bot.channels[m.ChannelID] = chat
	}

	// clean mentions
	msgToSend := m.Content
	for _, mention := range m.Mentions {
		msgToSend = strings.ReplaceAll(msgToSend, "<@"+mention.ID+">", mention.Username)
	}

	// send typing
	s.ChannelTyping(m.ChannelID)
	response, err := chat.Send(m.Author.Username, msgToSend)

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
