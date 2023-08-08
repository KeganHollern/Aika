package youtube

import (
	"aika/discord/discordai"
	"aika/storage"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"

	yt "github.com/kkdai/youtube/v2"
)

type Youtube struct {
	S3 *storage.S3
}

func (ytwrapper *Youtube) GetFunction_DownloadYoutube() discordai.Function {
	return discordai.Function{
		Definition: definition_DownloadYoutube,
		Handler:    ytwrapper.handler_DownloadYoutube,
	}
}

var definition_DownloadYoutube = openai.FunctionDefinition{
	Name:        "DownloadYoutube",
	Description: "Convert a youtube video URL to a downloadable MP4 URL.",
	Parameters: jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"url": {
				Type:        jsonschema.String,
				Description: "Full Video URL.",
				Properties:  map[string]jsonschema.Definition{},
			},
		},
		Required: []string{"url"},
	},
}

func (ytwrapper *Youtube) handler_DownloadYoutube(msgMap map[string]interface{}) (string, error) {
	results, err := ytwrapper.action_DownloadYoutube(msgMap["url"].(string))
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(results)
	if err != nil {
		return "", err
	}

	return string(data), err
}

func (ytwrapper *Youtube) action_DownloadYoutube(url string) (string, error) {
	c := yt.Client{}
	vid, err := c.GetVideo(url)
	if err != nil {
		return "", fmt.Errorf("failed to find youtube video; %w", err)
	}

	path := "user-content/youtube/" + vid.ID + ".mp4"

	// if already exists just give the user the existing video
	exists, err := ytwrapper.S3.KeyExists(path)
	if err != nil {
		return "", fmt.Errorf("failed to check s3; %w", err)
	}
	if exists {
		return fmt.Sprintf("%s/%s", ytwrapper.S3.PublicUrl, path), nil
	}

	formats := vid.Formats.WithAudioChannels()
	// TODO: download smallest fucking format
	// to save me money lmfao
	stream, _, err := c.GetStream(vid, &formats[0])
	if err != nil {
		return "", fmt.Errorf("failed to get stream; %w", err)
	}

	// stream video directly to S3
	// retry if 0 data transfers (idk? bug?)
	err = storage.ErrNoDataTransfered
	i := 0
	for errors.Is(err, storage.ErrNoDataTransfered) && i < 2 {
		err = ytwrapper.S3.StreamUpload(stream, path)
		i++
	}
	if err != nil {
		return "", fmt.Errorf("failed to upload stream to s3; %w", err)
	}

	return fmt.Sprintf("%s/%s", ytwrapper.S3.PublicUrl, path), nil
}
