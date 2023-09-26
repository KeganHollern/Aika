package transcoding

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMP3ToPCM(t *testing.T) {

	start := time.Now()
	data, err := mp3ToPCM("sample.mp3")
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.NotEmpty(t, data)
	assert.Len(t, data, 696)
	// expect PCM conversion happened INSTANTLY or something idk
	assert.Less(t, elapsed.Seconds(), 1.0)
}

func TestMP3ToOpus(t *testing.T) {

	start := time.Now()
	data, err := MP3ToOpus("sample.mp3")
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.NotEmpty(t, data)
	assert.Len(t, data, 696)
	// expect PCM conversion happened INSTANTLY or something idk
	assert.Less(t, elapsed.Seconds(), 1.0)
}

func TestDuration(t *testing.T) {
	data, err := mp3ToPCM("sample.mp3")
	assert.NoError(t, err)

	dur := pcmDuration(data)
	assert.Equal(t, "13.92s", dur.String())
}
