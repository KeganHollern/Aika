package main

import (
	"aika/actions"
	"aika/aika"
	"aika/discord"
	"aika/premium"
	"aika/s3"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/peterbourgon/diskv/v3"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

func main() {
	// randomness and logger setup
	rand.Seed(time.Now().UnixMilli())
	//logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.JSONFormatter{
		PrettyPrint:      true,
		DisableTimestamp: true,
	})

	// yoink env vars
	apiKey, exists := os.LookupEnv("OPENAI_KEY")
	if !exists {
		logrus.Fatalln("missing OPENAI_KEY environment variable")
	}

	// S3 configuration
	var s3cfg *s3.S3Config
	secret, exists := os.LookupEnv("S3_SECRET")
	if exists {
		access, exists := os.LookupEnv("S3_ACCESS")
		if exists {
			hostname, exists := os.LookupEnv("S3_HOSTNAME")
			if exists {
				bucket, exists := os.LookupEnv("S3_BUCKET")
				if exists {
					s3cfg = &s3.S3Config{
						Endpoint:  hostname,
						Bucket:    bucket,
						SecretKey: secret,
						AccessKey: access,
					}

				} else {
					logrus.Warnln("missing S3_BUCKET environment variable")
				}
			} else {
				logrus.Warnln("missing S3_HOSTNAME environment variable")
			}
		} else {
			logrus.Warnln("missing S3_ACCESS environment variable")
		}
	} else {
		logrus.Warnln("missing S3_SECRET environment variable")
	}

	// init OpenAI Client
	client := openai.NewClient(apiKey)

	// init Aika
	ai := aika.OpenAI{
		Client:  client,
		Timeout: time.Second * 60,
	}

	// add some functionality to aika
	err := ai.AddFunction(actions.Function_GenRandomNumber)
	if err != nil {
		logrus.WithError(err).Fatalln("failed to add random function")
	}
	err = ai.AddFunction(actions.Function_GetWaifuCategories)
	if err != nil {
		logrus.WithError(err).Fatalln("failed to add waifu category function")
	}
	err = ai.AddFunction(actions.Function_GetWaifuSfw)
	if err != nil {
		logrus.WithError(err).Fatalln("failed to add sfw waifu function")
	}
	err = ai.AddFunction(actions.Function_GetWaifuNsfw)
	if err != nil {
		logrus.WithError(err).Fatalln("failed to add nsfw waifu function")
	}
	err = ai.AddFunction(actions.Function_GetAnime)
	if err != nil {
		logrus.WithError(err).Fatalln("failed to add anime function")
	}
	err = ai.AddFunction(actions.Function_SearchWeb)
	if err != nil {
		logrus.WithError(err).Fatalln("failed to add anime function")
	}
	openAIActions := actions.OpenAI{
		Client: client,
		S3:     s3cfg,
	}
	err = ai.AddFunction(openAIActions.GetFunction_DallE())
	if err != nil {
		logrus.WithError(err).Fatalln("failed to add DallE function")
	}

	// init db
	flatTransform := func(s string) []string { return []string{} }
	db := &premium.Servers{
		DataStore: diskv.New(diskv.Options{
			BasePath:     "premium-servers",
			Transform:    flatTransform,
			CacheSizeMax: 1024 * 1024,
		}),
	}

	// start chat connections
	var wg sync.WaitGroup

	// init discord
	startDiscordChat(&wg, ai, db)

	// block on chats
	wg.Wait()
}

func startDiscordChat(
	wg *sync.WaitGroup,
	ai aika.OpenAI,
	db *premium.Servers,
) {

	// yoink env vars
	discordKey, exists := os.LookupEnv("AIKA_DISCORD_KEY")
	if !exists {
		logrus.Fatalln("missing AIKA_DISCORD_KEY environment variable")
	}

	bot := discord.AikaBot{
		PremiumDB: db,
		ApiKey:    discordKey,
		API:       ai,
	}

	err := bot.Start(wg)
	if err != nil {
		logrus.WithError(err).Fatalln("failed to discordgo")
	}
}
