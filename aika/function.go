package aika

import (
	"github.com/sashabaranov/go-openai"
)

type FunctionHandler func(string) (string, error)

type Function struct {
	Definition openai.FunctionDefinition
	Handler    FunctionHandler
}

// see `actions` package for example functions
