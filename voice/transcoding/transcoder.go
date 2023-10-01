package transcoding

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/sync/errgroup"
	"layeh.com/gopus"
)

// write discord audio to a WAV file in the output directory
func DiscordToFile(packets []*discordgo.Packet, outdir string) (string, error) {
	if err := os.MkdirAll(outdir, 0755); err != nil {
		return "", fmt.Errorf("failed to create out dir; %w", err)
	}

	// discord->opus
	opus, err := DiscordToOpus(packets)
	if err != nil {
		return "", fmt.Errorf("failed to extract opus frames; %w", err)
	}

	// opus->pcm
	pcm, err := OpusToPCM(opus)
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
	err = PCMToWav(pcm, outFile)
	if err != nil {
		return "", fmt.Errorf("failed to encode wav file; %w", err)
	}

	return output, nil
}

func NewOpusEncoder() (*gopus.Encoder, error) {
	encoder, err := gopus.NewEncoder(frameRate, channels, gopus.Audio)
	if err != nil {
		return nil, fmt.Errorf("failed to construct encoder; %w", err)
	}
	return encoder, nil
}

// blocking function to read MP3 data from the io.Reader and
// return opus frames on the opus channel
func StreamMP3ToOpus(reader io.Reader, opusChan chan []byte) error {
	pcmChan := make(chan []int16)

	// PCM->Opus encoder
	encoder, err := gopus.NewEncoder(frameRate, channels, gopus.Audio)
	if err != nil {
		return fmt.Errorf("failed to construct encoder; %w", err)
	}

	group := errgroup.Group{}
	group.SetLimit(2)

	// decode mp3 to pcm
	group.Go(func() error {
		defer close(pcmChan) // we close the PCM channel here to signify MP3 streaming is done

		err := StreamMP3ToPCM(reader, pcmChan)
		if err != nil {
			return fmt.Errorf("error decoding mp3; %w", err)
		}
		return nil
	})

	// encode pcm to opus
	group.Go(func() error {
		err := StreamPCMToOpus(encoder, pcmChan, opusChan)
		if err != nil {
			return fmt.Errorf("error encoding to opus; %w", err)
		}
		return nil
	})

	return group.Wait()
}

// convert Mp3 file name to opus frames
func MP3ToOpus(filename string) ([][]byte, error) {
	pcm, err := MP3ToPCM(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to decode pcm data; %w", err)
	}

	opus, err := PCMToOpus(pcm)
	if err != nil {
		return nil, fmt.Errorf("failed to encode opus packets; %w", err)
	}

	return opus, nil
}
