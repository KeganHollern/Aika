package ai

import (
	"context"
	"fmt"
	"io"

	"github.com/sashabaranov/go-openai"
)

type LanguageModel string

const (
	LanguageModel_GPT4  LanguageModel = "gpt-4-1106-preview" //openai.GPT40613
	LanguageModel_GPT35 LanguageModel = "gpt-3.5-turbo-1106" //openai.GPT3Dot5Turbo0613
)

type ChatRequest struct {
	Client *openai.Client

	System  openai.ChatCompletionMessage   // ai brain
	History []openai.ChatCompletionMessage // chat
	Message openai.ChatCompletionMessage   // requst

	Functions []openai.FunctionDefinition // chat - all definitions exist in AIBrain

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

func (request *ChatRequest) Stream(ctx context.Context, writer io.Writer) (openai.ChatCompletionMessage, error) {
	messages := []openai.ChatCompletionMessage{request.System}
	messages = append(messages, request.History...)
	messages = append(messages, request.Message)

	stream, err := request.Client.CreateChatCompletionStream(
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

	var message openai.ChatCompletionMessage

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			return message, nil
		}
		if err != nil {
			return message, fmt.Errorf("failed receiving openai chunks; %w", err)
		}

		delta := chunk.Choices[0].Delta

		if delta.FunctionCall != nil {
			// in case it's sent in chunks we'll do it this way :)
			if message.FunctionCall != nil {
				message.FunctionCall.Arguments += delta.FunctionCall.Arguments
				message.FunctionCall.Name += delta.FunctionCall.Name
			} else {
				message.FunctionCall = delta.FunctionCall
			}
		}
		// chunked role ?
		if delta.Role != "" {
			message.Role += delta.Role
		}
		// definitely chunked content
		if delta.Content != "" {
			message.Content += delta.Content
			_, err := writer.Write([]byte(delta.Content))
			if err != nil {
				return message, fmt.Errorf("failed to write chunk; %w", err)
			}
		}
	}
}
