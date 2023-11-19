package youtube

import (
	"aika/discord/discordai"
	"aika/voice/transcoding"
	"errors"
	"fmt"
	"io"

	yt "github.com/kkdai/youtube/v2"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
	"github.com/sirupsen/logrus"
)

type Player struct {
	Mixer *transcoding.Mixer

	//TODO:
	// - enqueue songs if one is play
	// - skip action
	// - stop action
	// - get queue (including current playing) action
	// - remove X from queue action (-1 for clearing entire queue?)
}

func (player *Player) GetFunction_PlayAudio() discordai.Function {
	return discordai.Function{
		Definition: definition_PlayAudio,
		Handler:    player.handler_PlayAudio,
	}
}

var definition_PlayAudio = openai.FunctionDefinition{
	Name:        "PlayAudio",
	Description: "Play the audio or music of a youtube video over voice chat.",
	Parameters: jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"url": {
				Type:        jsonschema.String,
				Description: "Full Video URL.",
				Properties:  map[string]jsonschema.Definition{},
			},
		},
		Required: []string{"url"},
	},
}

func (player *Player) handler_PlayAudio(msgMap map[string]interface{}) (string, error) {
	err := player.action_PlayAudio(msgMap["url"].(string))
	if err != nil {
		return "", err
	}

	return "sound now playing", err
}

func (player *Player) action_PlayAudio(url string) error {
	if player.Mixer == nil {
		return errors.New("mixer not found")
	}

	c := yt.Client{}
	vid, err := c.GetVideo(url)
	if err != nil {
		return fmt.Errorf("failed to find youtube video; %w", err)
	}

	formats := vid.Formats.WithAudioChannels()
	formats.Sort()
	// this selects the _smallest_ format or so
	target := formats[len(formats)-1]

	stream, size, err := c.GetStream(vid, &target)
	if err != nil {
		return fmt.Errorf("failed to get stream; %w", err)
	}

	if size == 0 {
		return fmt.Errorf("stream size is 0; %w", err)
	}

	input := player.Mixer.Create()
	go func() {
		// TODO: make a way to 'stop' playing lmao
		//TODO: she doesn't send "start speaking" so idk whats up
		// might be able to close "stream" before we've reached EOF to kill it safely?

		defer close(input)
		err = transcoding.StreamMPEGToPCM(stream, 0.3, input)
		if err != nil {
			logrus.WithError(err).Errorln("failed to stream youtube video to PCM audio")
		}
		buffer := make([]byte, 1)
		_, err := stream.Read(buffer)
		if errors.Is(err, io.EOF) {
			logrus.Debug("done playing youtube audio")
		} else {
			logrus.Warn("something strange happened while playing youtube audio - download stream isn't dead!")
		}

	}()

	return nil
}
