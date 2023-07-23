package web

import (
	"aika/discord/discordai"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

var (
	Function_GetAnime = discordai.Function{
		Definition: definition_GetAnime,
		Handler:    handler_FindAnime,
	}
)

type SearchResult struct {
	Data []struct {
		MalID  int    `json:"mal_id"`
		URL    string `json:"url"`
		Images struct {
			Jpg struct {
				ImageURL      string `json:"image_url"`
				SmallImageURL string `json:"small_image_url"`
				LargeImageURL string `json:"large_image_url"`
			} `json:"jpg"`
			Webp struct {
				ImageURL      string `json:"image_url"`
				SmallImageURL string `json:"small_image_url"`
				LargeImageURL string `json:"large_image_url"`
			} `json:"webp"`
		} `json:"images"`
		Trailer struct {
			YoutubeID string `json:"youtube_id"`
			URL       string `json:"url"`
			EmbedURL  string `json:"embed_url"`
			Images    struct {
				ImageURL        string `json:"image_url"`
				SmallImageURL   string `json:"small_image_url"`
				MediumImageURL  string `json:"medium_image_url"`
				LargeImageURL   string `json:"large_image_url"`
				MaximumImageURL string `json:"maximum_image_url"`
			} `json:"images"`
		} `json:"trailer"`
		Approved bool `json:"approved"`
		Titles   []struct {
			Type  string `json:"type"`
			Title string `json:"title"`
		} `json:"titles"`
		Title         string   `json:"title"`
		TitleEnglish  string   `json:"title_english"`
		TitleJapanese string   `json:"title_japanese"`
		TitleSynonyms []string `json:"title_synonyms"`
		Type          string   `json:"type"`
		Source        string   `json:"source"`
		Episodes      int      `json:"episodes"`
		Status        string   `json:"status"`
		Airing        bool     `json:"airing"`
		Aired         struct {
			From time.Time `json:"from"`
			To   time.Time `json:"to"`
			Prop struct {
				From struct {
					Day   int `json:"day"`
					Month int `json:"month"`
					Year  int `json:"year"`
				} `json:"from"`
				To struct {
					Day   int `json:"day"`
					Month int `json:"month"`
					Year  int `json:"year"`
				} `json:"to"`
			} `json:"prop"`
			String string `json:"string"`
		} `json:"aired"`
		Duration   string  `json:"duration"`
		Rating     string  `json:"rating"`
		Score      float64 `json:"score"`
		ScoredBy   int     `json:"scored_by"`
		Rank       int     `json:"rank"`
		Popularity int     `json:"popularity"`
		Members    int     `json:"members"`
		Favorites  int     `json:"favorites"`
		Synopsis   string  `json:"synopsis"`
		Background string  `json:"background"`
		Season     string  `json:"season"`
		Year       int     `json:"year"`
		Broadcast  struct {
			Day      string `json:"day"`
			Time     string `json:"time"`
			Timezone string `json:"timezone"`
			String   string `json:"string"`
		} `json:"broadcast"`
		Producers []struct {
			MalID int    `json:"mal_id"`
			Type  string `json:"type"`
			Name  string `json:"name"`
			URL   string `json:"url"`
		} `json:"producers"`
		Licensors []struct {
			MalID int    `json:"mal_id"`
			Type  string `json:"type"`
			Name  string `json:"name"`
			URL   string `json:"url"`
		} `json:"licensors"`
		Studios []struct {
			MalID int    `json:"mal_id"`
			Type  string `json:"type"`
			Name  string `json:"name"`
			URL   string `json:"url"`
		} `json:"studios"`
		Genres []struct {
			MalID int    `json:"mal_id"`
			Type  string `json:"type"`
			Name  string `json:"name"`
			URL   string `json:"url"`
		} `json:"genres"`
		ExplicitGenres []any `json:"explicit_genres"`
		Themes         []struct {
			MalID int    `json:"mal_id"`
			Type  string `json:"type"`
			Name  string `json:"name"`
			URL   string `json:"url"`
		} `json:"themes"`
		Demographics []struct {
			MalID int    `json:"mal_id"`
			Type  string `json:"type"`
			Name  string `json:"name"`
			URL   string `json:"url"`
		} `json:"demographics"`
	} `json:"data"`
}

type AnimeInfo struct {
	Title    string `json:"title"`
	URL      string `json:"info_url"`
	ImageUrl string `json:"image_url"`
	Synopsis string `json:"synopsis"`
}
type AnimeResult struct {
	Animes []AnimeInfo `json:"animes"`
}

var definition_GetAnime = openai.FunctionDefinition{
	Name:        "GetAnime",
	Description: "Search for information on an anime.",
	Parameters: jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"query": {
				Type:        jsonschema.String,
				Description: "Anime search query. Best kept short and concise, such as the title of the anime.",
				Properties:  map[string]jsonschema.Definition{},
			},
		},
		Required: []string{"query"},
	},
}

func handler_FindAnime(msgMap map[string]interface{}) (string, error) {
	tags, err := action_FindAnime(msgMap["query"].(string))
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(tags)
	if err != nil {
		return "", err
	}

	return string(data), err
}

func action_FindAnime(query string) (AnimeResult, error) {
	data, err := fetch("https://api.jikan.moe/v4/anime?q=" + url.QueryEscape(query) + "&sfw&limit=3")
	if err != nil {
		return AnimeResult{}, fmt.Errorf("fetch anime failed; %w", err)
	}

	var result SearchResult
	err = json.Unmarshal(data, &result)
	if err != nil {
		return AnimeResult{}, fmt.Errorf("failed to parse animes; %w", err)
	}

	response := AnimeResult{}
	for _, entry := range result.Data {
		title := entry.TitleEnglish
		if title == "" {
			title = entry.Title
		}
		response.Animes = append(response.Animes, AnimeInfo{
			Title:    title,
			URL:      entry.URL,
			ImageUrl: entry.Images.Jpg.ImageURL,
			Synopsis: entry.Synopsis,
		})
	}

	return response, nil
}
