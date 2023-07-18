package aika

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

const (
	maxFailedCalls = 3
)

type OpenAI struct {
	Client  *openai.Client
	Timeout time.Duration

	functions   map[string]Function
	definitions []openai.FunctionDefinition
}

func (ai *OpenAI) AddFunction(fnc Function) error {
	if ai.functions == nil {
		ai.functions = make(map[string]Function)
	}

	_, exists := ai.functions[fnc.Definition.Name]
	if exists {
		return errors.New("function already exists")
	}
	ai.definitions = append(ai.definitions, fnc.Definition)
	ai.functions[fnc.Definition.Name] = fnc
	return nil
}

// GetResponse takes in a user message and returns
// a chain of messages including function calls which
// answers the users query.
func (ai *OpenAI) GetResponse(
	system openai.ChatCompletionMessage,
	history []openai.ChatCompletionMessage,
	request openai.ChatCompletionMessage,
) ([]openai.ChatCompletionMessage, error) {

	messages := []openai.ChatCompletionMessage{system}
	messages = append(messages, history...)
	messages = append(messages, request)

	var newMessages []openai.ChatCompletionMessage

	// loop call functions until the ai generates a response
	// itr is used to track successive invalid function calls
	// if itr exceeds maxFailedCalls we break out with error
	itr := 0
	for itr = 0; itr < maxFailedCalls; itr++ {
		message, err := ai.Query(messages)
		if err != nil {
			return nil, fmt.Errorf("failed to get response; %w", err)
		}
		newMessages = append(newMessages, message)

		// this is a real response message so stop looping !
		if message.FunctionCall == nil {
			break
		}

		// this must be a function call - so we'll handle that & iterate!

		fnc, exists := ai.functions[message.FunctionCall.Name]
		if !exists {
			continue // hopefully the AI will correct itself and use a real function next time
		}

		result, err := fnc.Handler(message.FunctionCall.Arguments)
		if err != nil {
			return nil, fmt.Errorf("fatal function call; %w", err)
		}

		// because a real function was called
		// we reset itr to 0
		itr = 0
		// push function call details to openai and iterate
		resultMsg := openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleFunction,
			Name:    message.FunctionCall.Name,
			Content: result,
		}
		newMessages = append(newMessages, resultMsg) // must save this to `newMessages`` as well!

		messages = append(messages, message)
		messages = append(messages, resultMsg)
	}
	if itr == maxFailedCalls {
		return nil, errors.New("ai called invalid function")
	}

	return newMessages, nil
}

func (ai *OpenAI) Query(
	messages []openai.ChatCompletionMessage,
) (openai.ChatCompletionMessage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ai.Timeout)
	defer cancel()

	logrus.WithField("messages", messages).Debugln("querying OpenAI")

	resp, err := ai.Client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:     openai.GPT40613,
			Messages:  messages,
			Functions: ai.definitions,
		},
	)
	if err != nil {
		return openai.ChatCompletionMessage{}, fmt.Errorf("failed to query openai; %w", err)
	}

	return resp.Choices[0].Message, nil
}
