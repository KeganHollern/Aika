package youtube

import (
	"aika/discord/discordai"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/buger/jsonparser"

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
	Title string `json:"title"`
	URL   string `json:"url"`
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

	results, err := search(input, 5)
	if err != nil {
		return searchResults, fmt.Errorf("failed to search youtube; %w", err)
	}

	for _, result := range results[0:maxSearchResults] {
		searchResults.Results = append(searchResults.Results, youtubeResult{
			Title: result.Title,
			URL:   result.URL,
		})
	}

	return searchResults, nil
}

type SearchResult struct {
	Title, Uploader, URL, Duration, ID string
	Live                               bool
	SourceName                         string
	Extra                              []string
}

func getContent(data []byte, index int) []byte {
	id := fmt.Sprintf("[%d]", index)
	contents, _, _, _ := jsonparser.Get(data, "contents", "twoColumnSearchResultsRenderer", "primaryContents", "sectionListRenderer", "contents", id, "itemSectionRenderer", "contents")
	return contents
}

var httpClient = &http.Client{}

func search(searchTerm string, limit int) ([]*SearchResult, error) {
	results := []*SearchResult{}
	url := fmt.Sprintf("https://www.youtube.com/results?search_query=%s", url.QueryEscape(searchTerm))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build GET request; %w", err)
	}
	req.Header.Add("Accept-Language", "en")
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot get youtube page; %w", err)
	}

	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	buffer, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot read body; %w", err)
	}

	body := string(buffer)
	splittedScript := strings.Split(body, `window["ytInitialData"] = `)
	if len(splittedScript) != 2 {
		splittedScript = strings.Split(body, `var ytInitialData = `)
	}

	if len(splittedScript) != 2 {
		return nil, fmt.Errorf("too many splitted scripts")
	}
	splittedScript = strings.Split(splittedScript[1], `window["ytInitialPlayerResponse"] = null;`)
	jsonData := []byte(splittedScript[0])

	index := 0
	var contents []byte

	for {
		contents = getContent(jsonData, index)
		_, _, _, err = jsonparser.Get(contents, "[0]", "carouselAdRenderer")

		if err == nil {
			index++
		} else {
			break
		}
	}

	_, err = jsonparser.ArrayEach(contents, func(value []byte, t jsonparser.ValueType, i int, err error) {
		if err != nil { // error so just skip this item
			return
		}

		if limit > 0 && len(results) >= limit {
			return
		}

		id, err := jsonparser.GetString(value, "videoRenderer", "videoId")
		if err != nil {
			return
		}

		title, err := jsonparser.GetString(value, "videoRenderer", "title", "runs", "[0]", "text")
		if err != nil {
			return
		}

		uploader, err := jsonparser.GetString(value, "videoRenderer", "ownerText", "runs", "[0]", "text")
		if err != nil {
			return
		}

		live := false
		duration, err := jsonparser.GetString(value, "videoRenderer", "lengthText", "simpleText")

		if err != nil {
			duration = ""
			live = true
		}

		results = append(results, &SearchResult{
			Title:      title,
			Uploader:   uploader,
			Duration:   duration,
			ID:         id,
			URL:        fmt.Sprintf("https://youtube.com/watch?v=%s", id),
			Live:       live,
			SourceName: "youtube",
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse results; %w", err)
	}

	return results, nil
}
