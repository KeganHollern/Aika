package voice

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"layeh.com/gopus"
)

// encode the audio data into a file
// TODO: upload this to S3 for trolling ?
func EncodeAudio(packets []*discordgo.Packet) (string, error) {

	// decode packets to PCM data
	decoder, err := gopus.NewDecoder(48000, 2)
	if err != nil {
		return "", err
	}

	var pcmData [][]int16
	for _, packet := range packets {
		pcm, err := decoder.Decode(packet.Opus, 960, false)
		if err != nil {
			return "", err
		}
		pcmData = append(pcmData, pcm)
	}

	// hash PCM data for Unique ID
	hash, err := HashPCMData(pcmData)
	if err != nil {
		return "", err
	}

	// create file from UID
	filename := "assets/audio/" + hash + ".wav"

	// write PCM data to WAV file
	err = writePCMToWAV(pcmData, filename)
	if err != nil {
		return "", err
	}

	return filename, nil
}

// write PCM data to WAV file
func writePCMToWAV(pcmData [][]int16, filename string) error {
	outFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outFile.Close()

	format := &audio.Format{SampleRate: 48000, NumChannels: 2}                 // assuming 48kHz stereo audio
	e := wav.NewEncoder(outFile, format.SampleRate, 16, format.NumChannels, 1) // 16 is the bit depth

	for _, pcm := range pcmData {
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

// Convert []int16 to []int for IntBuffer
func convertToIntSlice(data []int16) []int {
	result := make([]int, len(data))
	for i, v := range data {
		result[i] = int(v)
	}
	return result
}

// HashPCMData generates a SHA-256 hash of the provided PCM data.
func HashPCMData(pcmData [][]int16) (string, error) {
	hasher := sha256.New()
	for _, pcm := range pcmData {
		for _, sample := range pcm {
			// Convert the int16 sample to byte array and write it to the hash.
			if err := writeInt16ToHasher(sample, hasher); err != nil {
				return "", fmt.Errorf("failed to hash PCM data: %w", err)
			}
		}
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// writeInt16ToHasher writes the int16 value to the given hash.Hash in little-endian format.
func writeInt16ToHasher(value int16, hasher hash.Hash) error {
	bytes := make([]byte, 2)
	bytes[0] = byte(value)
	bytes[1] = byte(value >> 8)
	_, err := hasher.Write(bytes)
	return err
}
