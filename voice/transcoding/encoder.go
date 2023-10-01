package transcoding

import (
	"fmt"
	"io"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"layeh.com/gopus"
)

// Stream encode PCM frames into Opus frames. Requires a GOPUS encoder
func StreamPCMToOpus(encoder *gopus.Encoder, pcm chan []int16, opusChan chan []byte) error {
	for pcm := range pcm {
		opus, err := encoder.Encode(pcm, frameSize, maxBytes)
		if err != nil {
			return fmt.Errorf("failed to encode opus frame; %w", err)
		}
		opusChan <- opus
	}
	return nil
}

// Encode PCM frames to Opus frames
func PCMToOpus(pcm [][]int16) ([][]byte, error) {
	encoder, err := gopus.NewEncoder(frameRate, channels, gopus.Audio)
	if err != nil {
		return nil, fmt.Errorf("failed to construct encoder; %w", err)
	}

	var output [][]byte
	for _, pcm := range pcm {
		opus, err := encoder.Encode(pcm, frameSize, maxBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to encode opus frame; %w", err)
		}
		output = append(output, opus)
	}

	return output, nil
}

func PCMToWav(pcm [][]int16, output io.WriteSeeker) error {
	format := &audio.Format{SampleRate: frameRate, NumChannels: channels}     // assuming 48kHz stereo audio
	e := wav.NewEncoder(output, format.SampleRate, 16, format.NumChannels, 1) // 16 is the bit depth

	for _, pcm := range pcm {
		intBuffer := &audio.IntBuffer{
			Format:         format,
			Data:           convertToIntSlice(pcm),
			SourceBitDepth: 16,
		}
		if err := e.Write(intBuffer); err != nil {
			return err
		}
	}

	return e.Close()
}
