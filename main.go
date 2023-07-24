package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"aika/discord"
	"aika/storage"

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
	dev := flag.Bool("dev", false, "developer mode")
	flag.Parse()

	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.JSONFormatter{PrettyPrint: true})

	ctx, cancel := newInterruptContext(context.Background())
	defer cancel()
	logrus.Infoln("starting aika")

	envVar := "AIKA_DISCORD_KEY"
	if *dev {
		envVar = "AIKA_DEV_DISCORD_KEY"
	}
	discordKey, exists := os.LookupEnv(envVar)
	if !exists {
		logrus.Fatalf("missing %s from env\n", envVar)
	}

	openaiKey, exists := os.LookupEnv("OPENAI_KEY")
	if !exists {
		logrus.Fatalln("missing OPENAI_KEY from env")
	}

	s3, err := storage.NewS3FromEnv()
	if err != nil {
		logrus.WithError(err).Fatalln("failed to init S3 store")
	}

	cfg, err := storage.NewDisk("./data/config.yaml")
	if err != nil {
		logrus.WithError(err).Fatalln("error reading config.yaml")
	}

	logrus.WithField("discord_key", discordKey[0:3]).Debugln("starting chatbot...")
	_, err = discord.StartChatbot(
		ctx,
		discordKey,
		openai.NewClient(openaiKey),
		s3,
		cfg,
	)
	if err != nil {
		logrus.WithError(err).Fatalln("failed to init discord bot")
	}

	<-ctx.Done()
	logrus.Infoln("shutdown")
	// do things that exit w/ ctx cancellation
}
