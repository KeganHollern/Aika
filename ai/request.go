package ai

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

type LanguageModel string

const (
	LanguageModel_GPT4  LanguageModel = openai.GPT40613
	LanguageModel_GPT35 LanguageModel = openai.GPT3Dot5Turbo0613
)

type ChatRequest struct {
	Client *openai.Client

	System  openai.ChatCompletionMessage   // ai brain
	History []openai.ChatCompletionMessage // chat
	Message openai.ChatCompletionMessage   // requst

	Functions []openai.FunctionDefinition // chat

	Model LanguageModel // chat
}

// Send a request to OpenAI and return the response
func (request *ChatRequest) Send(ctx context.Context) (openai.ChatCompletionMessage, error) {

	messages := []openai.ChatCompletionMessage{request.System}
	messages = append(messages, request.History...)
	messages = append(messages, request.Message)

	resp, err := request.Client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:     string(request.Model),
			Messages:  messages,
			Functions: request.Functions,
		},
	)
	if err != nil {
		return openai.ChatCompletionMessage{}, fmt.Errorf("failed to query openai; %w", err)
	}

	return resp.Choices[0].Message, nil
}
