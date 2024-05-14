package actions

import (
	"aika/actions/web"
	"aika/ai"
	"aika/discord/discordai"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

// because OpenAI actions require a client,
// the must be acquired via "get" functions.

type Vision struct {
	Client *openai.Client
}

func (vis *Vision) GetFunction_DescribeImage() discordai.Function {
	return discordai.Function{
		Definition: definition_DescribeImage,
		Handler:    vis.handler_DescribeImage,
	}
}

var definition_DescribeImage = openai.FunctionDefinition{
	Name: "DescribeImage",
	Description: `Answer a question about an image.
Can be used to get a description or to answer specific questions.
Returns a short answer to the query.
Supports: PNG (.png), JPEG (.jpeg and .jpg), WEBP (.webp), and non-animated GIF (.gif).`,

	Parameters: jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"image": {
				Type:        jsonschema.String,
				Description: "Image URL. Format: https://example.com/image.jpg",
				Properties:  map[string]jsonschema.Definition{},
			},
			"query": {
				Type:        jsonschema.String,
				Description: "Query. Question or task for inspecting the image.",
				Properties:  map[string]jsonschema.Definition{},
			},
		},
		Required: []string{"image", "query"},
	},
}

func (vis *Vision) handler_DescribeImage(msgMap map[string]interface{}) (string, error) {
	// call function
	answer, err := vis.action_DescribeImage(msgMap["image"].(string), msgMap["query"].(string))
	if err != nil {
		return "", err
	}

	// return parsed result
	// we marshal a map so the AI knows the response is the `image_url`
	result := map[string]string{
		"answer": answer,
	}

	data, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (vis *Vision) action_DescribeImage(image string, query string) (string, error) {
	// correct tenor URLs so aika can process them
	if strings.Index(image, "https://tenor.com/view") == 0 {
		newUrl, err := web.TenorToGif(image)
		if err != nil {
			return "", fmt.Errorf("failed to extract tenor gif; %w", err)
		}
		image = newUrl
	}

	// send image to OAI
	req := &ai.VisionRequest{
		Client: vis.Client,
		System: openai.ChatCompletionMessage{
			Role: openai.ChatMessageRoleSystem,
			Content: `You are an image inspection utility.
Please be descriptive when answering the users question.
It's important to provide any details the user might need.
Keep your reply condensed and avoid lists.`,
		},
		Message:  query,
		ImageURL: image,
		Model:    ai.VisionModel_GPT4o,
	}
	response, err := req.Send(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to query image; %w", err)
	}

	// return ai response
	return response.Content, nil
}
