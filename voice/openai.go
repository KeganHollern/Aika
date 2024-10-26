package voice

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/sashabaranov/go-openai"
)

type OpenAI struct {
	Client *openai.Client
	// voice ID
	VoiceID openai.SpeechVoice
}

func (api *OpenAI) TextToSpeechStream(text string, writer io.Writer) error {
	resp, err := api.Client.CreateSpeech(context.Background(), openai.CreateSpeechRequest{
		Model:          openai.TTSModel1,
		Input:          text,
		Voice:          api.VoiceID,
		ResponseFormat: openai.SpeechResponseFormatMp3,
		Speed:          1.0,
	})
	if err != nil {
		return fmt.Errorf("failed to create speech; %w", err)
	}
	defer resp.Close()

	// write speech to
	_, err = io.Copy(writer, resp)
	return err
}

func (api *OpenAI) GetVoices() ([]Voice, error) {
	return []Voice{
		{
			Name: "Alloy",
			Id:   string(openai.VoiceAlloy),
		},
		{
			Name: "Echo",
			Id:   string(openai.VoiceEcho),
		},
		{
			Name: "Fable",
			Id:   string(openai.VoiceFable),
		},
		{
			Name: "Onyx",
			Id:   string(openai.VoiceOnyx),
		},
		{
			Name: "Nova",
			Id:   string(openai.VoiceNova),
		},
		{
			Name: "Shimmer",
			Id:   string(openai.VoiceShimmer),
		},
	}, nil
}

func (api *OpenAI) SetVoice(nameOrId string) error {
	voices, err := api.GetVoices()
	if err != nil {
		return fmt.Errorf("failed to get voices; %w", err)
	}

	for _, voice := range voices {
		if strings.EqualFold(voice.Id, nameOrId) ||
			strings.EqualFold(voice.Name, nameOrId) {

			api.VoiceID = openai.SpeechVoice(voice.Id)
			return nil
		}
	}

	return errors.New("incorrect voice ID or name provided")
}
