package discordai

import "github.com/sashabaranov/go-openai"

type FunctionHandler func(map[string]interface{}) (string, error)

type Function struct {
	Definition openai.FunctionDefinition
	Handler    FunctionHandler
}
