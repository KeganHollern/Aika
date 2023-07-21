package actions

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/gocolly/colly"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

const maxSearchResults = 5

var (
	Function_SearchWeb = Function{
		Definition: definition_SearchWeb,
		Handler:    handler_SearchWeb,
	}
)

type WebResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
}
type WebResults struct {
	Results []WebResult `json:"results"`
}

var definition_SearchWeb = openai.FunctionDefinition{
	Name:        "SearchWeb",
	Description: "Search the internet. Returns the top 5 results for the search query.",
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

func handler_SearchWeb(args string) (string, error) {
	msgMap, err := argsToMap(args)
	if err != nil {
		return "", err
	}

	results, err := action_SearchWeb(msgMap["query"].(string))
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(results)
	if err != nil {
		return "", err
	}

	return string(data), err
}

func action_SearchWeb(input string) (WebResults, error) {
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36"),
	)
	searchResults := WebResults{}
	c.OnHTML("body > form > div > table:nth-of-type(3) > tbody", func(e *colly.HTMLElement) {
		resultTitle := ""
		resultDesc := ""
		resultURL := ""
		e.ForEachWithBreak("tr", func(i int, row *colly.HTMLElement) bool {
			switch (i + 1) % 4 {
			case 0:
				searchResults.Results = append(searchResults.Results, WebResult{
					Title:       resultTitle,
					Description: resultDesc,
					URL:         "https://" + resultURL,
				})

				return true
			case 1:
				//title
				resultTitle = strings.TrimSpace(row.ChildText("td:nth-child(2)"))
			case 2:
				// description
				resultDesc = strings.TrimSpace(row.ChildText("td:nth-child(2)"))
			case 3:
				//URL
				resultURL = strings.TrimSpace(row.ChildText("td:nth-child(2)"))
			}

			return len(searchResults.Results) < maxSearchResults
		})
	})

	err := c.Visit("https://lite.duckduckgo.com/lite/?q=" + url.QueryEscape(input))
	if err != nil {
		return searchResults, fmt.Errorf("failed to visit site; %w", err)
	}
	c.Wait()

	return searchResults, nil
}
