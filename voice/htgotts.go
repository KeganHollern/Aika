package voice

import (
	"errors"
	"os"

	htgotts "github.com/hegedustibor/htgo-tts"
	"github.com/sirupsen/logrus"
)

type TTS interface {
	TextToSpeech(text string, outdir string) (string, error)
}

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
