package discordchat

import (
	"aika/actions/discord"
	"aika/actions/math"
	action_openai "aika/actions/openai"
	"aika/actions/web"
	"aika/discord/discordai"
	"aika/storage"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

type Direct struct {
	Chat

	History []openai.ChatCompletionMessage
}

func (chat *Direct) OnMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	locked := chat.Mutex.TryLock()
	if !locked {
		s.ChannelMessageSendReply(m.ChannelID, "rate limit", m.Reference())
		return
	}
	defer chat.Mutex.Unlock()

	msg := chat.formatUsers(m.Content, m.Mentions)

	system := chat.Brain.BuildSystemMessage([]string{m.Author.Username})
	history := chat.History
	message := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: msg, // TODO: prefix <USER>: <MSG> ?
		Name:    chat.cleanUserName(m.Author.Username),
	}

	var err error

	s.ChannelTyping(m.ChannelID)

	history, err = chat.Brain.Process(
		chat.Ctx,
		system,
		history,
		message,
		chat.getAvailableFunctions(s, m),
		chat.getLanguageModel(),
	)
	if err != nil {
		logrus.WithError(err).Errorln("failed while processing in brain")
		s.ChannelMessageSend(m.ChannelID, err.Error())
		return
	}
	if len(history) == 0 {
		logrus.Errorln("blank history returned from brain.Process")
		s.ChannelMessageSend(m.ChannelID, "my brain is empty")
		return
	}

	chat.History = history

	res := history[len(history)-1]

	// TODO: improve this log
	logrus.
		WithField("message", msg).
		WithField("response", res.Content).
		Infoln("chat log")

	response := chat.replaceMarkdownLinks(res.Content)
	if len(response) > 2000 {
		s.ChannelFileSendWithMessage(m.ChannelID, "*response too long - sent as file*", "response.txt", strings.NewReader(response))
	} else {
		s.ChannelMessageSend(m.ChannelID, response)
	}
}

func (chat *Direct) getAvailableFunctions(
	s *discordgo.Session,
	m *discordgo.MessageCreate,
) []discordai.Function {
	functions := []discordai.Function{
		web.Function_GetWaifuCateogires,
		web.Function_GetWaifuNsfw,
		web.Function_GetWaifuSfw,
		web.Function_SearchWeb,
		math.Function_GenRandomNumber,
	}
	s3, err := storage.NewS3FromEnv()
	if err != nil {
		s3 = nil // ensure this shit
		logrus.WithError(err).Warnln("no S3 configured for DallE action")
	}
	oai := &action_openai.DallE{
		Client: chat.Brain.OpenAI,
		S3:     s3,
	}

	functions = append(functions, oai.GetFunction_DallE())

	// kegan :)
	if m.Author.ID == "241370201222938626" {
		g := &discord.Guilds{
			Session: s,
		}
		functions = append(functions, g.GetFunction_ListGuilds())
	}

	//TODO: add more functions to this
	return functions
}
