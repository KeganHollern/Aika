package discordai

import (
	"aika/ai"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

//go:embed system.txt
var sys string

type AIBrain struct {
	OpenAI    *openai.Client
	Functions map[string]FunctionHandler
}

func (brain *AIBrain) AddFunction(name string, callback FunctionHandler) error {
	if _, exists := brain.Functions[name]; exists {
		return fmt.Errorf("function name %s already exists", name)
	}

	brain.Functions[name] = callback
	return nil
}

// process a message & return the new chat history
// response is *always* the last item in the chat history
func (brain *AIBrain) Process(
	ctx context.Context,
	system openai.ChatCompletionMessage,
	history []openai.ChatCompletionMessage,
	message openai.ChatCompletionMessage,
	functions []openai.FunctionDefinition,
	model ai.LanguageModel,
) ([]openai.ChatCompletionMessage, error) {

	// copy history to a new slice
	newHistory := []openai.ChatCompletionMessage{}
	newHistory = append(newHistory, history...)

	failedFuncCall := false
	for i := 0; i < 2; i++ {

		// get openai response
		req := ai.ChatRequest{
			Client:    brain.OpenAI,
			System:    system,
			History:   newHistory, // we use copied history here so function history is retained!
			Message:   message,
			Functions: functions,
			Model:     model,
		}
		res, err := req.Send(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to request openai; %w", err)
		}

		// push request message into history
		newHistory = append(newHistory, message)
		// push response into history
		newHistory = append(newHistory, res)

		// if FunctionCall is nil - then OpenAI sent us a human response :)
		if res.FunctionCall == nil {
			break
		}

		// !!! process function call !!!

		// find function handler
		name := res.FunctionCall.Name
		handler, exists := brain.Functions[name]
		if !exists {
			// hopefully the AI will correct itself and use a real function next time
			// if not - for loop will exit eventually
			logrus.WithField("func", name).Warnln("openai tried to call non-existant function")
			failedFuncCall = true
			continue
		}

		failedFuncCall = false

		// unmarshal args
		var args map[string]interface{}
		err = json.Unmarshal([]byte(res.FunctionCall.Arguments), &args)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal openai args; %w", err)
		}

		// call handler (runs function and gets result for openai!)
		result, err := handler(args)
		if err != nil {
			// functions only return ERR when a fatal error occurs
			// anything that OpenAI should process is returned as result
			return nil, fmt.Errorf("failed during function call; %w", err)
		}

		// update message for next iteration
		message = openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleFunction,
			Name:    name,
			Content: result,
		}
		// decrement i-- so we infinitely loop from this point
		i = 0
	}

	if failedFuncCall {
		return nil, fmt.Errorf("failed while calling functions")
	}

	return newHistory, nil
}

// build system message from format embedded system.txt
func (brain *AIBrain) BuildSystemMessage(
	chatMembers []string, // TODO: may change "string" to a more structured "ChatMember" struct which whill give aika both the @ identifier and the username
) openai.ChatCompletionMessage {
	return openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: fmt.Sprintf(sys, strings.Join([]string{}, ", ")),
	}
}
