package math

import (
	"aika/discord/discordai"
	"fmt"
	"math"
	"math/rand"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
	"github.com/sirupsen/logrus"
)

// because no struct is necessary for math functions
// they can be inline defined

var (
	Function_GenRandomNumber = discordai.Function{
		Definition: definition_getRandomNumber,
		Handler:    handler_GetRandomNumber,
	}
)

// definition for getRandomNumber
// TODO: learn to generate this from reflection
var definition_getRandomNumber = openai.FunctionDefinition{
	Name:        "getRandomNumber",
	Description: "generate a random number with decimals.",

	Parameters: jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"min": {
				Type:        jsonschema.Number,
				Description: "inclusive minimum random value",
				Properties:  map[string]jsonschema.Definition{},
			},
			"max": {
				Type:        jsonschema.Number,
				Description: "exclusive maximum random value",
				Properties:  map[string]jsonschema.Definition{},
			},
			"round": {
				Type:        jsonschema.Boolean,
				Description: "true to round the random number to the nearest whole number.",
				Properties:  map[string]jsonschema.Definition{},
			},
		},
		Required: []string{"min", "max", "round"},
	},
}

// handler for getRandomNumber
func handler_GetRandomNumber(msgMap map[string]interface{}) (string, error) {
	value := action_GetRandomNumber(msgMap["min"].(float64), msgMap["max"].(float64), msgMap["round"].(bool))
	return fmt.Sprintf("%f", value), nil
}

// raw function implementation
func action_GetRandomNumber(min float64, max float64, round bool) float64 {
	result := min + ((max - min) * rand.Float64())
	if round {
		result = math.Round(result)
	}
	logrus.WithFields(logrus.Fields{
		"min":    min,
		"max":    max,
		"result": result,
	}).Debug("generating random number")
	return result
}
