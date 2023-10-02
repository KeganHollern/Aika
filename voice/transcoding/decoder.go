package transcoding

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"layeh.com/gopus"
)

const (
	channels  int = 2                   // 1 for mono, 2 for stereo
	frameRate int = 48000               // audio sampling rate
	frameSize int = 960                 // uint16 size of each audio frame
	maxBytes  int = (frameSize * 2) * 2 // max size of opus data
)

// Stream decode MP3 reader to PCM frame channel
func StreamMPEGToPCM(reader io.Reader, volume float64, ch chan []int16) error {
	// Create a shell command "object" to run.
	// We set up ffmpeg to read from its stdin (the provided PipeReader) by passing "-" as the input file.
	run := exec.Command(
		"ffmpeg",
		"-i", "-",
		"-f", "s16le",
		"-ar", strconv.Itoa(frameRate),
		"-ac", strconv.Itoa(channels),
		"-vn", // This flag ensures that only audio is processed
		"-filter:a", "volume="+strconv.FormatFloat(volume, 'f', 2, 64),
		"pipe:1")
	run.Stdin = reader

	ffmpegout, err := run.StdoutPipe()
	if err != nil {
		return fmt.Errorf("StdoutPipe error: %w", err)
	}

	ffmpegbuf := bufio.NewReaderSize(ffmpegout, 16384)

	err = run.Start()
	if err != nil {
		return fmt.Errorf("failed to start ffmpeg; %w", err)
	}

	defer run.Process.Kill()

	for {
		// Read data from ffmpeg stdout
		audiobuf := make([]int16, frameSize*channels)
		err := binary.Read(ffmpegbuf, binary.LittleEndian, &audiobuf)
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading from ffmpeg stdout: %w", err)
		}

		// Send received PCM frame to the provided channel
		ch <- audiobuf
	}

	return nil
}

// Decode Opus frames to PCM frames
func OpusToPCM(opus [][]byte) ([][]int16, error) {
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

// Extract Opus frames from Discord packets
func DiscordToOpus(packets []*discordgo.Packet) ([][]byte, error) {
	output := [][]byte{}
	for _, packet := range packets {
		if packet == nil {
			return nil, errors.New("nil discord packet")
		}
		output = append(output, packet.Opus)
	}
	return output, nil
}

// read mp3 file to PCM buffer
func MP3ToPCM(filename string) ([][]int16, error) {
	run := exec.Command("ffmpeg", "-i", filename, "-f", "s16le", "-ar", strconv.Itoa(frameRate), "-ac", strconv.Itoa(channels), "pipe:1")

	ffmpegout, err := run.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("StdoutPipe error: %w", err)
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
