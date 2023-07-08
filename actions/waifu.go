package actions

import (
	"aika/aika"
	"encoding/json"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

var (
	Function_GetWaifuCategories = aika.Function{
		Definition: definition_GetWaifuCategories,
		Handler:    handler_GetWaifuCategories,
	}
	Function_GetWaifuSfw = aika.Function{
		Definition: definition_GetWaifuSfw,
		Handler:    handler_GetWaifuSfw,
	}
	Function_GetWaifuNsfw = aika.Function{
		Definition: definition_GetWaifuNsfw,
		Handler:    handler_GetWaifuNsfw,
	}
)

var waifu_categories_sfw = []string{
	"waifu",
	"neko",
	"shinobu",
	"megumin",
	"bully",
	"cuddle",
	"cry",
	"hug",
	"awoo",
	"kiss",
	"lick",
	"pat",
	"smug",
	"bonk",
	"yeet",
	"blush",
	"smile",
	"wave",
	"highfive",
	"handhold",
	"nom",
	"bite",
	"glomp",
	"slap",
	"kill",
	"kick",
	"happy",
	"wink",
	"poke",
	"dance",
	"cringe",
	"waifu",
	"neko",
}

var waifu_categories_nsfw = []string{
	"waifu",
	"neko",
	"trap",
	"bl*wj*b",
}

var definition_GetWaifuCategories = openai.FunctionDefinition{
	Name:        "definition_GetWaifuCategories",
	Description: "Returns available categories from waifu.pics, an anime image API.",
	Parameters: jsonschema.Definition{
		Type:       jsonschema.Object,
		Properties: map[string]jsonschema.Definition{},
		Required:   []string{},
	},
}

var definition_GetWaifuSfw = openai.FunctionDefinition{
	Name:        "GetWaifuSfw",
	Description: "Returns an anime waifu image from waifu.pics, an anime image API.",
	Parameters: jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"category": {
				Type:        jsonschema.String,
				Enum:        waifu_categories_sfw,
				Description: "Category of anime image to return.",
				Properties:  map[string]jsonschema.Definition{},
			},
		},
		Required: []string{"category"},
	},
}
var definition_GetWaifuNsfw = openai.FunctionDefinition{
	Name:        "GetWaifuOther",
	Description: "Returns an 'Other' anime waifu image from waifu.pics, an anime image API.",
	Parameters: jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"category": {
				Type:        jsonschema.String,
				Enum:        waifu_categories_nsfw,
				Description: "Category of 'Other' anime image to return.",
				Properties:  map[string]jsonschema.Definition{},
			},
		},
		Required: []string{"category"},
	},
}

type WaifuResponse struct {
	URL     *string `json:"url",omitempty`
	Message *string `json:"message",omitempty`
}

func handler_GetWaifuSfw(args string) (string, error) {
	msgMap, err := argsToMap(args)
	if err != nil {
		return "", err
	}

	tags, err := action_GetWaifu("sfw", msgMap["category"].(string))
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(tags)
	if err != nil {
		return "", err
	}

	return string(data), err
}
func handler_GetWaifuNsfw(args string) (string, error) {
	msgMap, err := argsToMap(args)
	if err != nil {
		return "", err
	}

	tags, err := action_GetWaifu("nsfw", msgMap["category"].(string))
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(tags)
	if err != nil {
		return "", err
	}

	return string(data), err
}
func action_GetWaifu(waifu_type string, category string) (WaifuResponse, error) {
	data, err := fetch("https://api.waifu.pics/" + waifu_type + "/" + strings.ReplaceAll(category, "*", "o"))
	if err != nil {
		return WaifuResponse{}, err
	}
	var resp WaifuResponse
	err = json.Unmarshal(data, &resp)
	return resp, err
}

type WaifuCategories struct {
	Sfw  []string `json:"sfw_categories"`
	Nsfw []string `json:"other_categories"`
}

func handler_GetWaifuCategories(_ string) (string, error) {
	categories := WaifuCategories{
		Sfw:  waifu_categories_sfw,
		Nsfw: waifu_categories_nsfw,
	}

	data, err := json.Marshal(categories)
	if err != nil {
		return "", err
	}

	return string(data), err
}
