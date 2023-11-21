package ai

import (
	"context"
	"fmt"
	"io"

	"github.com/sashabaranov/go-openai"
)

type LanguageModel string
type VisionModel string

const (
	LanguageModel_GPT4  LanguageModel = "gpt-4-1106-preview" //openai.GPT40613
	LanguageModel_GPT35 LanguageModel = "gpt-3.5-turbo-1106" //openai.GPT3Dot5Turbo0613
)

const (
	VisionModel_GPT4 VisionModel = "gpt-4-vision-preview"
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

// since vision does not support functions, we needed a unique request for hitting the VISION model and extracting a description of the provided image
// we can use something like "actions" to let the AI grab the description of an attached image or we can automatically attach an image description to
// the provided AI request like:
//
//	What do you see?
//
//	*Note: the user has attached an image to their message. This is a description:*
//	> Description of the image here
//
// By combining the two, "action" and "description", the AI could potentially send a _different_ request to the Vision model by
// calling an action like "RequestImageDetails: 'what text is in the image?'"- so they have a generic description like 'a paper w/ writing on it'
// and can ask for more details of that image like 'what is written on the paper?' ect.

type VisionRequest struct {
	Client *openai.Client

	System openai.ChatCompletionMessage // vision brain

	//TODO: do we need history in this request?

	Message  string // request - "gen desc", "what text is"
	ImageURL string // image url - see https://platform.openai.com/docs/guides/vision

	Model VisionModel
}

func (request *VisionRequest) Send(ctx context.Context) (openai.ChatCompletionMessage, error) {
	messages := []openai.ChatCompletionMessage{
		request.System,
		{
			Role: openai.ChatMessageRoleUser,
			MultiContent: []openai.ChatMessagePart{
				{
					Type: openai.ChatMessagePartTypeText,
					Text: request.Message,
				},
				{
					Type: openai.ChatMessagePartTypeImageURL,
					ImageURL: &openai.ChatMessageImageURL{
						URL:    request.ImageURL,
						Detail: openai.ImageURLDetailAuto,
					},
				},
			},
		},
	}

	resp, err := request.Client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:    string(request.Model),
			Messages: messages,
		},
	)
	if err != nil {
		return openai.ChatCompletionMessage{}, fmt.Errorf("failed to query openai; %w", err)

	}

	return resp.Choices[0].Message, nil
}
