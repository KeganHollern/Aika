package voice

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestElevenLabs(t *testing.T) {
	var speaker TTS = &ElevenLabs{
		ApiKey:  os.Getenv("ELEVENLABS_APIKEY"),
		VoiceID: "BreKkXSwy4hr1vgm7ZqX",
	}
	test_speaker(speaker, t)
}

func test_speaker(speaker TTS, t *testing.T) {
	path, err := speaker.TextToSpeech("Wh-what? How can you say something so serious? B-baka!", "testout")
	assert.NoError(t, err)
	assert.NotEmpty(t, path)
}
