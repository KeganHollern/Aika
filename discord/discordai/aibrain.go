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

const (
	failAttempts = 2
)

type AIBrain struct {
	OpenAI *openai.Client

	HistorySize int
	Character   string
}

// process a message & return the new chat history
// response is *always* the last item in the chat history
func (brain *AIBrain) Process(
	ctx context.Context,
	system openai.ChatCompletionMessage,
	history []openai.ChatCompletionMessage,
	message openai.ChatCompletionMessage,
	functions []Function,
	model ai.LanguageModel,
	internalArgs map[string]interface{},
) ([]openai.ChatCompletionMessage, error) {

	// copy history to a new slice
	newHistory := []openai.ChatCompletionMessage{}
	newHistory = append(newHistory, history...)

	functionHandlers := make(map[string]FunctionHandler)
	functionDefinitions := []openai.FunctionDefinition{}
	for _, fnc := range functions {
		functionDefinitions = append(functionDefinitions, fnc.Definition)
		functionHandlers[fnc.Definition.Name] = fnc.Handler
	}

	failedFuncCall := false
	for i := 0; i < failAttempts; i++ {

		// get openai response
		req := ai.ChatRequest{
			Client:    brain.OpenAI,
			System:    system,
			History:   newHistory, // we use copied history here so function history is retained!
			Message:   message,
			Functions: functionDefinitions,
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

		// trim history
		// TODO: trim based on TOKEN COUNT
		if len(newHistory) > brain.HistorySize {
			newHistory = newHistory[len(newHistory)-brain.HistorySize:]
		}

		// if FunctionCall is nil - then OpenAI sent us a human response :)
		if res.FunctionCall == nil {
			break
		}

		// !!! process function call !!!

		// find function handler
		name := res.FunctionCall.Name
		handler, exists := functionHandlers[name]
		var result string
		if !exists {
			// hopefully the AI will correct itself and use a real function next time
			// if not - for loop will exit eventually
			logrus.WithField("call", res.FunctionCall).Warnln("openai tried to call non-existant function")
			failedFuncCall = true
			result = fmt.Sprintf("the function '%s' does not exist.", name)
		} else {

			failedFuncCall = false

			// unmarshal args
			var args map[string]interface{}
			err = json.Unmarshal([]byte(res.FunctionCall.Arguments), &args)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal openai args; %w", err)
			}

			// append internal args
			for k, v := range internalArgs {
				args[k] = v
			}

			// call handler (runs function and gets result for openai!)
			result, err = handler(args)
			if err != nil {
				// functions only return ERR when a fatal error occurs
				// anything that OpenAI should process is returned as result
				return nil, fmt.Errorf("failed during function call; %w", err)
			}
			logrus.WithField("call", res.FunctionCall).WithField("result", result).Debugln("executed function")
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
	displayNames []string,
	mentions []string,
) openai.ChatCompletionMessage {
	memberNames := strings.Join(displayNames, ", ")
	memberMentions := strings.Join(mentions, ", ")

	return openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: fmt.Sprintf(sys, memberNames, memberMentions, brain.Character),
	}
}
