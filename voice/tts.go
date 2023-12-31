package voice

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
)

type TTS interface {
	// TextToSpeech converts text to speech and saves the
	// generated audio to an MP3 file in the out directory.
	TextToSpeech(text string, outdir string) (string, error)
	// TextToSpeechStream converts text to speech and
	// writes the generated audio to the io.Writer in MP3
	// format
	TextToSpeechStream(text string, writer io.Writer) error
	// GetVoices returns a list of possible voices for this
	// TTS engine. If applicable, a voice Identifier is included.
	GetVoices() ([]Voice, error)
	// SetVoice changes the voice used for text to speech.
	SetVoice(nameOrId string) error
}

type Voice struct {
	Name string
	Id   string
}

// --- utilities for this package

func hashString(input string) string {
	hash := sha256.New()
	hash.Write([]byte(input))
	return hex.EncodeToString(hash.Sum(nil))
}
