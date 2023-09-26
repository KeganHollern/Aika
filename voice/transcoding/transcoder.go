package transcoding

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"os/exec"
	"path"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"layeh.com/gopus"
)

const (
	channels  int = 2                   // 1 for mono, 2 for stereo
	frameRate int = 48000               // audio sampling rate
	frameSize int = 960                 // uint16 size of each audio frame
	maxBytes  int = (frameSize * 2) * 2 // max size of opus data
)

func GetDiscordDuration(packets []*discordgo.Packet) (time.Duration, error) {
	opus, err := discordToOpus(packets)
	if err != nil {
		return 0, fmt.Errorf("failed to exrtract opus frames; %w", err)
	}

	pcmData, err := opusToPCM(opus)
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

// convert Discord packets to raw Opus frames
func discordToOpus(packets []*discordgo.Packet) ([][]byte, error) {
	output := [][]byte{}
	for _, packet := range packets {
		if packet == nil {
			return nil, errors.New("nil discord packet")
		}
		output = append(output, packet.Opus)
	}
	return output, nil
}

// convert Opus frames to PCM data
func opusToPCM(opus [][]byte) ([][]int16, error) {
	decoder, err := gopus.NewDecoder(frameRate, channels)
	if err != nil {
		return nil, fmt.Errorf("failed to construct decoder; %w", err)
	}

	var output [][]int16
	for _, frame := range opus {
		pcm, err := decoder.Decode(frame, frameSize, false)
		if err != nil {
			return nil, fmt.Errorf("failed to decode opus frame; %w", err)
		}
		output = append(output, pcm)
	}

	return output, nil
}

// read mp3 file to PCM buffer
func mp3ToPCM(filename string) ([][]int16, error) {
	run := exec.Command("ffmpeg", "-i", filename, "-f", "s16le", "-ar", strconv.Itoa(frameRate), "-ac", strconv.Itoa(channels), "pipe:1")
	ffmpegout, err := run.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg error; %w", err)
	}

	ffmpegbuf := bufio.NewReaderSize(ffmpegout, 16384)

	err = run.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start ffmpeg; %w", err)
	}

	defer run.Process.Kill()

	// read ffmpeg output
	output := [][]int16{}
	for {
		audiobuf := make([]int16, frameSize*channels)
		err = binary.Read(ffmpegbuf, binary.LittleEndian, &audiobuf)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read ffmpeg output; %w", err)
		}

		output = append(output, audiobuf)
	}
	return output, nil
}

// convert PCM data to Opus frames
func pcmToOpus(pcmData [][]int16) ([][]byte, error) {
	encoder, err := gopus.NewEncoder(frameRate, channels, gopus.Audio)
	if err != nil {
		return nil, fmt.Errorf("failed to construct encoder; %w", err)
	}

	output := [][]byte{}
	for _, pcm := range pcmData {
		opus, err := encoder.Encode(pcm, frameSize, maxBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to encode opus frame; %w", err)
		}
		output = append(output, opus)
	}

	return output, nil
}

// convert PCM data to WAV
func pcmToWAV(pcmData [][]int16, output io.WriteSeeker) error {
	format := &audio.Format{SampleRate: frameRate, NumChannels: channels}     // assuming 48kHz stereo audio
	e := wav.NewEncoder(output, format.SampleRate, 16, format.NumChannels, 1) // 16 is the bit depth

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

// convert Mp3 file name to opus frames
func MP3ToOpus(filename string) ([][]byte, error) {
	pcm, err := mp3ToPCM(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to decode pcm data; %w", err)
	}

	opus, err := pcmToOpus(pcm)
	if err != nil {
		return nil, fmt.Errorf("failed to encode opus packets; %w", err)
	}

	return opus, nil
}

//TODO: possible to stream PCM data to a transcription service / interface ?

// write discord audio to a WAV file in the output directory
func DiscordToFile(packets []*discordgo.Packet, outdir string) (string, error) {
	if err := os.MkdirAll(outdir, 0755); err != nil {
		return "", fmt.Errorf("failed to create out dir; %w", err)
	}

	// discord->opus
	opus, err := discordToOpus(packets)
	if err != nil {
		return "", fmt.Errorf("failed to extract opus frames; %w", err)
	}

	// opus->pcm
	pcm, err := opusToPCM(opus)
	if err != nil {
		return "", fmt.Errorf("failed to decode opus frames; %w", err)
	}

	// create output file & open it for writing
	hash, err := hashPCMData(pcm)
	if err != nil {
		return "", err
	}
	output := path.Join(outdir, hash+".wav")
	outFile, err := os.Create(output)
	if err != nil {
		return "", fmt.Errorf("failed to create output wav file; %w", err)
	}
	defer outFile.Close()

	// pcm->wav
	err = pcmToWAV(pcm, outFile)
	if err != nil {
		return "", fmt.Errorf("failed to encode wav file; %w", err)
	}

	return output, nil
}

// --- helper functions for transcoding

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

// Convert []int16 to []int for IntBuffer
func convertToIntSlice(data []int16) []int {
	result := make([]int, len(data))
	for i, v := range data {
		result[i] = int(v)
	}
	return result
}
