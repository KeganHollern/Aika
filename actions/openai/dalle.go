package actions

import (
	"aika/discord/discordai"
	"aika/storage"
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
	"github.com/sirupsen/logrus"
)

// because OpenAI actions require a client,
// the must be acquired via "get" functions.

type DallE struct {
	Client *openai.Client
	S3     *storage.S3
}

func (ai *DallE) GetFunction_DallE() discordai.Function {
	return discordai.Function{
		Definition: definition_DallE,
		Handler:    ai.handler_DallE,
	}
}

// === DALL-E

var definition_DallE = openai.FunctionDefinition{
	Name:        "GenerateImage",
	Description: "Generate an image using DallE, an AI image generator.",

	Parameters: jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"prompt": {
				Type:        jsonschema.String,
				Description: "AI image generation prompt",
				Properties:  map[string]jsonschema.Definition{},
			},
		},
		Required: []string{"prompt"},
	},
}

func (ai *DallE) handler_DallE(msgMap map[string]interface{}) (string, error) {
	// call function
	url, err := ai.action_DallE(msgMap["prompt"].(string))
	if err != nil {
		return "", err
	}

	// return parsed result
	// we marshal a map so the AI knows the response is the `image_url`
	result := map[string]string{
		"image_url": url,
	}

	data, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (ai *DallE) action_DallE(prompt string) (string, error) {
	reqUrl := openai.ImageRequest{
		Prompt:         prompt,
		Size:           openai.CreateImageSize1024x1024,
		ResponseFormat: openai.CreateImageResponseFormatURL,
		N:              1,
		Model:          openai.CreateImageModelDallE3,
		Quality:        openai.CreateImageQualityHD,
		//Style:  openai.CreateImageStyleVivid,
	}

	respUrl, err := ai.Client.CreateImage(context.Background(), reqUrl)
	if err != nil {
		return "", fmt.Errorf("failed to create image; %w", err)
	}

	oaiURL := respUrl.Data[0].URL
	if ai.S3 == nil {
		// no S3 configured
		return oaiURL, nil
	}

	path := "user-content/dalle/" + uuid.NewString() + ".png"
	err = ai.S3.DownloadAndUpload(oaiURL, path)
	if err != nil {
		logrus.WithError(err).Warnln("failed to upload to S3")
		return oaiURL, nil
	}

	// return the public URL for the uploaded image
	return fmt.Sprintf("%s/%s", ai.S3.PublicUrl, path), nil
}
