package voice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/haguro/elevenlabs-go"
)

type ElevenLabs struct {
	ApiKey string
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

	/*
		models, err := client.GetModels()
		if err != nil {
			return "", fmt.Errorf("could not get models; %w", err)
		}
	*/

	/*
		voices, err := client.GetVoices()
		if err != nil {
			return "", fmt.Errorf("could not get voices; %w", err)
		}
		fmt.Println(voices[0])
	*/

	ttsReq := elevenlabs.TextToSpeechRequest{
		Text:    text,
		ModelID: "eleven_monolingual_v1",
		/*
			VoiceSettings: &elevenlabs.VoiceSettings{
				Stability:       .35,
				SimilarityBoost: .4,
			},*/
	}
	// todo: stream response directly to voice chat ?
	// elevenlabs.TextToSpeechStream()

	// MF3mGyEYCl7XYWbV9V6O -- Elli
	// HoS8AiGDOzZCrk4VTbQl -- Valley Girl
	// BreKkXSwy4hr1vgm7ZqX -- Janiah (also good)
	// cfB6lViBdEjgGRi26uBV -- Sexy Female (like)
	// heILIY4H8yX0oMI0iwte -- Sally realistic
	// kOFQK5H1ZWrZR03j1rvh -- Shelby
	audio, err := client.TextToSpeech("BreKkXSwy4hr1vgm7ZqX", ttsReq)
	if err != nil {
		return "", fmt.Errorf("failed tts; %w", err)
	}

	if err := os.WriteFile(file, audio, 0644); err != nil {
		return "", fmt.Errorf("failed to write file to disk; %w", err)
	}

	return file, nil
}

func hashString(input string) string {
	hash := sha256.New()
	hash.Write([]byte(input))
	return hex.EncodeToString(hash.Sum(nil))
}
