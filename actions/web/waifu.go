package web

import (
	"aika/discord/discordai"
	"encoding/json"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

var (
	Function_GetWaifuCateogires = discordai.Function{
		Definition: definition_GetWaifuCategories,
		Handler:    handler_GetWaifuCategories,
	}
	Function_GetWaifuSfw = discordai.Function{
		Definition: definition_GetWaifuSfw,
		Handler:    handler_GetWaifuSfw,
	}
	Function_GetWaifuNsfw = discordai.Function{
		Definition: definition_GetWaifuNsfw,
		Handler:    handler_GetWaifuNsfw,
	}
)

var definition_GetWaifuCategories = openai.FunctionDefinition{
	Name:        "GetWaifuCategories",
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
				Description: "Category of anime image to return. Use GetWaifuCategories to get available categories.",
				Properties:  map[string]jsonschema.Definition{},
			},
		},
		Required: []string{"category"},
	},
}
var definition_GetWaifuNsfw = openai.FunctionDefinition{
	Name:        "GetWaifuOther",
	Description: "Returns an 'Other' anime waifu image from waifu.pics, an anime image API. Use GetWaifuCategories to get available categories.",
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

type waifuResponse struct {
	URL     *string `json:"url",omitempty`
	Message *string `json:"message",omitempty`
}

func handler_GetWaifuSfw(msgMap map[string]interface{}) (string, error) {
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
func handler_GetWaifuNsfw(msgMap map[string]interface{}) (string, error) {
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
func action_GetWaifu(waifu_type string, category string) (waifuResponse, error) {
	data, err := fetch("https://api.waifu.pics/" + waifu_type + "/" + strings.ReplaceAll(category, "*", "o"))
	if err != nil {
		return waifuResponse{}, err
	}
	var resp waifuResponse
	err = json.Unmarshal(data, &resp)
	return resp, err
}

type waifuCategories struct {
	Sfw  []string `json:"sfw_categories"`
	Nsfw []string `json:"other_categories"`
}

func handler_GetWaifuCategories(_ map[string]interface{}) (string, error) {
	categories := waifuCategories{
		Sfw:  waifu_categories_sfw,
		Nsfw: waifu_categories_nsfw,
	}

	data, err := json.Marshal(categories)
	if err != nil {
		return "", err
	}

	return string(data), err
}
