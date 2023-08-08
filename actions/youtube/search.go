package youtube

import (
	"aika/discord/discordai"
	"encoding/json"
	"fmt"

	yt "github.com/lithdew/youtube"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

const maxSearchResults = 5

var (
	Function_SearchYoutube = discordai.Function{
		Definition: definition_SearchYoutube,
		Handler:    handler_SearchYoutube,
	}
)

type youtubeResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
}
type youtubeResults struct {
	Results []youtubeResult `json:"results"`
}

var definition_SearchYoutube = openai.FunctionDefinition{
	Name:        "SearchYoutube",
	Description: "Search youtube for a video. Returns the top 5 results for the search query.",
	Parameters: jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"query": {
				Type:        jsonschema.String,
				Description: "Search query.",
				Properties:  map[string]jsonschema.Definition{},
			},
		},
		Required: []string{"query"},
	},
}

func handler_SearchYoutube(msgMap map[string]interface{}) (string, error) {
	results, err := action_SearchYoutube(msgMap["query"].(string))
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(results)
	if err != nil {
		return "", err
	}

	return string(data), err
}

func action_SearchYoutube(input string) (youtubeResults, error) {
	searchResults := youtubeResults{}

	results, err := yt.Search("fuck you", 0)
	if err != nil {
		return searchResults, fmt.Errorf("failed to search youtube; %w", err)
	}

	for _, result := range results.Items[0:maxSearchResults] {
		searchResults.Results = append(searchResults.Results, youtubeResult{
			Title:       result.Title,
			URL:         fmt.Sprintf("https://youtu.be/%s", result.ID),
			Description: result.Description[0:128],
		})
	}

	return searchResults, nil
}
