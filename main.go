package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"aika/discord"

	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

func newInterruptContext(parent context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// probably not the right way to say "whenever anything happens on this channel"
	go func() {
		select {
		case <-c:
			logrus.Infoln("ctrl+c detected")
			cancel()
		case <-ctx.Done():
		}
	}()

	return ctx, cancel
}

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.JSONFormatter{PrettyPrint: true})

	ctx, cancel := newInterruptContext(context.Background())
	defer cancel()
	logrus.Infoln("starting aika")

	discordKey, exists := os.LookupEnv("AIKA_DEV_DISCORD_KEY")
	if !exists {
		logrus.Fatalln("missing AIKA_DEV_DISCORD_KEY from env")
	}

	openaiKey, exists := os.LookupEnv("OPENAI_KEY")
	if !exists {
		logrus.Fatalln("missing OPENAI_KEY from env")
	}

	logrus.WithField("discord_key", discordKey[0:3]).Debugln("starting chatbot...")
	_, err := discord.StartChatbot(
		ctx,
		discordKey,
		openai.NewClient(openaiKey),
	)
	if err != nil {
		logrus.WithError(err).Fatalln("failed to init discord bot")
	}

	<-ctx.Done()
	logrus.Infoln("shutdown")
	// do things that exit w/ ctx cancellation
}
