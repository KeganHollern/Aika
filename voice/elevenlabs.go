package voice

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/haguro/elevenlabs-go"
)

type ElevenLabs struct {
	ApiKey string
	// voice ID
	VoiceID string
}

// convert text to speech & save the output in the provided directory
func (api *ElevenLabs) TextToSpeech(text string, outdir string) (string, error) {
	if err := os.MkdirAll(outdir, 0755); err != nil {
		return "", fmt.Errorf("failed to create out dir; %w", err)
	}

	file := path.Join(outdir, hashString(text)+".mp3")

	_, err := os.Stat(file)
	if err == nil {
		return file, nil // this voice line was already spoken!
	}

	// not spoken - gen voice lines
	client := elevenlabs.NewClient(context.Background(), api.ApiKey, 30*time.Second)
	ttsReq := elevenlabs.TextToSpeechRequest{
		Text:    text,
		ModelID: "eleven_monolingual_v1",
	}
	// BreKkXSwy4hr1vgm7ZqX -- Janiah
	audio, err := client.TextToSpeech(api.VoiceID, ttsReq)
	if err != nil {
		return "", fmt.Errorf("failed tts; %w", err)
	}

	if err := os.WriteFile(file, audio, 0644); err != nil {
		return "", fmt.Errorf("failed to write file to disk; %w", err)
	}

	return file, nil
}

func (api *ElevenLabs) TextToSpeechStream(text string, writer io.Writer) error {
	client := elevenlabs.NewClient(context.Background(), api.ApiKey, 30*time.Second)
	ttsReq := elevenlabs.TextToSpeechRequest{
		Text:    text,
		ModelID: "eleven_monolingual_v1",
	}
	return client.TextToSpeechStream(writer, api.VoiceID, ttsReq)
}

func (api *ElevenLabs) GetVoices() ([]Voice, error) {

	client := elevenlabs.NewClient(context.Background(), api.ApiKey, 30*time.Second)
	voices, err := client.GetVoices()
	if err != nil {
		return nil, fmt.Errorf("failed to get voices; %w", err)
	}

	output := []Voice{}
	for _, voice := range voices {
		// only show cloned voices & the normal aika voice
		if voice.Category != "cloned" && voice.VoiceId != "BreKkXSwy4hr1vgm7ZqX" {
			continue
		}
		output = append(output, Voice{
			Name: voice.Name,
			Id:   voice.VoiceId,
		})
	}
	return output, nil
}

func (api *ElevenLabs) SetVoice(nameOrId string) error {
	voices, err := api.GetVoices()
	if err != nil {
		return fmt.Errorf("failed to get voices; %w", err)
	}

	for _, voice := range voices {
		if strings.EqualFold(voice.Id, nameOrId) ||
			strings.EqualFold(voice.Name, nameOrId) {

			api.VoiceID = voice.Id
			return nil
		}
	}

	return errors.New("incorrect voice ID or name provided")
}
