package transcoding

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"time"

	"github.com/bwmarrin/discordgo"
)

func GetDiscordDuration(packets []*discordgo.Packet) (time.Duration, error) {
	opus, err := DiscordToOpus(packets)
	if err != nil {
		return 0, fmt.Errorf("failed to exrtract opus frames; %w", err)
	}

	pcmData, err := OpusToPCM(opus)
	if err != nil {
		return 0, fmt.Errorf("failed to decode opus frames; %w", err)
	}

	return pcmDuration(pcmData), nil

}

func pcmDuration(pcmData [][]int16) time.Duration {
	totalSamples := 0
	for _, pcm := range pcmData {
		totalSamples += len(pcm)
	}

	seconds := float64(totalSamples) / (float64(frameRate) * float64(channels))
	return time.Duration(seconds * float64(time.Second))
}

// Convert []int16 to []int for IntBuffer
func convertToIntSlice(data []int16) []int {
	result := make([]int, len(data))
	for i, v := range data {
		result[i] = int(v)
	}
	return result
}

// hashPCMData generates a SHA-256 hash of the provided PCM data.
func hashPCMData(pcmData [][]int16) (string, error) {
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
