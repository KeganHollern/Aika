package discordai

import (
	"aika/ai"
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

//go:embed system.txt
var sys string

type AIBrain struct {
	OpenAI *openai.Client
}

func (brain *AIBrain) DummyRequest(ctx context.Context) string {
	req := ai.ChatRequest{
		Client: brain.OpenAI,

		System: openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: "You are a chat bot impersonating a tsundere anime girl. Respond as an anime character would.",
		},

		History: []openai.ChatCompletionMessage{},

		Message: openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: "Hi :)",
		},

		Functions: []openai.FunctionDefinition{},

		Model: ai.LanguageModel_GPT35,
	}

	res, err := req.Send(ctx)
	if err != nil {
		logrus.WithError(err).Errorln("failed while requesting openai")
		panic(err)
	}

	logrus.WithField("res", res.Content).Infoln("ai response")

	return res.Content
}

// build system message from format embedded system.txt
func (brain *AIBrain) buildSystemMessage(
	chatMembers []string, // TODO: may change "string" to a more structured "ChatMember" struct which whill give aika both the @ identifier and the username
) openai.ChatCompletionMessage {
	return openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: fmt.Sprintf(sys, strings.Join([]string{}, ", ")),
	}
}

func THIS_FUNCTION_HANDLES_USER_INPUT_AND_RETURNS_CHANNEL_OUTPUT() {

}
