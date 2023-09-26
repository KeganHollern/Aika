package voice

import (
	"errors"
	"fmt"
	"io"
	"os"

	htgotts "github.com/hegedustibor/htgo-tts"
	"github.com/sirupsen/logrus"
)

type Google struct{}

func (api *Google) TextToSpeech(text string, outdir string) (string, error) {
	speech := htgotts.Speech{Folder: outdir, Language: "en"}
	path, err := speech.CreateSpeechFile(text, hashString(text))
	if err != nil {
		return "", err
	}

	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.Size() == 1685 {
		logrus.WithField("line", text).Infoln("htgotts returned bad MP3file")
		return "", errors.New("failed to gen speech - line too long")
	}
	return path, nil
}

// not real-time streaming but fullfills the interface
func (api *Google) TextToSpeechStream(text string, writer io.Writer) error {
	// Generate the audio file from the given text.
	filepath, err := api.TextToSpeech(text, "assets/audio")
	if err != nil {
		return fmt.Errorf("failed to gen tts file; %w", err)
	}

	// Open the generated audio file.
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to open tts file; %w", err)
	}
	defer file.Close()

	// Stream the contents of the file to the provided writer.
	_, err = io.Copy(writer, file)
	if err != nil {
		return fmt.Errorf("failed to stream tts file; %w", err)
	}

	// Optionally, delete the audio file after streaming.
	err = os.Remove(filepath)
	if err != nil {
		return fmt.Errorf("failed to delete tts file; %w", err)
	}

	return nil
}
