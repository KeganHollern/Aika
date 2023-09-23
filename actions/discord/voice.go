package discord

import (
	"aika/discord/discordai"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/bwmarrin/dgvoice"
	"github.com/bwmarrin/discordgo"

	htgotts "github.com/hegedustibor/htgo-tts"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
	"github.com/sirupsen/logrus"
)

type Voice struct {
	Session *discordgo.Session
	// TODO: store this in some guild details or something idk
	Connection *discordgo.VoiceConnection
}

func (v *Voice) GetFunction_JoinChannel() discordai.Function {
	return discordai.Function{
		Definition: definition_joinChannel,
		Handler:    v.handle_joinChannel,
	}
}
func (v *Voice) GetFunction_LeaveChannel() discordai.Function {
	return discordai.Function{
		Definition: definition_leaveChannel,
		Handler:    v.handle_leaveChannel,
	}
}

// admin command
func (v *Voice) GetFunction_ForceSay() discordai.Function {
	return discordai.Function{
		Definition: definition_forceSay,
		Handler:    v.handle_forceSay,
	}
}

var definition_forceSay = openai.FunctionDefinition{
	Name:        "forceSpeakMessage",
	Description: "Force a message to be spoken in the current voice chat.",

	Parameters: jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"message": {
				Type:        jsonschema.String,
				Description: "message to speak in voice chat.",
				Properties:  map[string]jsonschema.Definition{},
			},
		},
		Required: []string{"message"},
	},
}
var definition_leaveChannel = openai.FunctionDefinition{
	Name:        "leaveVoiceChat",
	Description: "Disconnect from the voice chat channel.",

	Parameters: jsonschema.Definition{
		Type:       jsonschema.Object,
		Properties: map[string]jsonschema.Definition{},
		Required:   []string{},
	},
}
var definition_joinChannel = openai.FunctionDefinition{
	Name:        "joinVoiceChat",
	Description: "Connect to the provide voice chat channel.",

	Parameters: jsonschema.Definition{
		Type:       jsonschema.Object,
		Properties: map[string]jsonschema.Definition{},
		Required:   []string{},
	},
}

func (v *Voice) handle_joinChannel(msgMap map[string]interface{}) (string, error) {
	// if sender is not in a voice chat then we can't join anything
	if msgMap["internal_sender_author_vc"].(string) == "" {
		return "user is not in a voice chat", nil
	}

	// call function
	err := v.action_joinChannel(msgMap["internal_sender_guildid"].(string), msgMap["internal_sender_author_vc"].(string))
	if err != nil {
		return "", err
	}

	return "connected successfully", nil
}

func (v *Voice) handle_forceSay(msgMap map[string]interface{}) (string, error) {
	// call function
	err := v.action_forceSay(msgMap["message"].(string))
	if err != nil {
		return "", err
	}

	return "spoken successfully", nil
}

func (v *Voice) handle_leaveChannel(msgMap map[string]interface{}) (string, error) {
	// call function
	err := v.action_leaveChannel()
	if err != nil {
		return "", err
	}

	return "disconnected successfully", nil
}

func (v *Voice) action_joinChannel(guildID string, channelID string) error {
	//v.Session.LogLevel = discordgo.LogDebug

	var err error
	if v.Connection != nil {
		if v.Connection.GuildID != guildID {
			err = v.Connection.ChangeChannel(channelID, false, false)
		} else {
			return errors.New("connected in a different discord guild")
		}
	} else {
		v.Connection, err = v.Session.ChannelVoiceJoin(guildID, channelID, false, false)
		v.Connection.LogLevel = discordgo.LogDebug
	}
	if err != nil {
		return fmt.Errorf("failed to join voice chat; %w", err)
	}

	//logrus.Infoln("JOINED VOICE")

	// TODO: dump channel participants for aika ?
	// TODO: maybe start sending/recieve loops and just pipe those into chat message?
	// literally no idea
	/*

		so my thought is let aika "connect" and "disconnect" from voice
		then once she is connected she gets a special 'guild' and 'channel' populated with VC participants
		where she treats that "voice channel" like a chat channel
		but all in/out values are speech translated
		this is kinda scuffed

	*/

	v.Connection.AddHandler(func(vc *discordgo.VoiceConnection, vs *discordgo.VoiceSpeakingUpdate) {
		if vs.UserID != "241370201222938626" {
			return
		}
		// from here I can convert USER->SSRC

		logrus.WithField("update", vs).Info("kegan void update")
	})

	// TODO: figure out how to pipe "OpusRecv" into per-ssrc channels :)
	go func() {
		err := v.Connection.Speaking(true)
		if err != nil {
			logrus.WithError(err).Errorln("failed to set speaking")
			return
		}
		// listen and echo back voices
		for p := range v.Connection.OpusRecv {
			// from here I can identify SSRC->USER and filter packets by sender
			// allowing Aika to build "voice audio" on a per-sender basis
			v.Connection.OpusSend <- p.Opus
		}

		v.Connection.Speaking(false)
	}()

	return nil
}
func (v *Voice) action_leaveChannel() error {
	//logrus.Println("LEAVE CHAN")
	if v.Connection == nil {
		return nil // not connected
	}

	close(v.Connection.OpusRecv)
	err := v.Connection.Disconnect()
	v.Connection = nil
	if err != nil {
		return fmt.Errorf("failed disconnect voice chat; %w", err)
	}
	return nil
}

func (v *Voice) action_forceSay(message string) error {
	if v.Connection == nil {
		return nil // not connected
	}

	// generate TTS
	speech := htgotts.Speech{Folder: "assets/audio", Language: "en"}
	path, err := speech.CreateSpeechFile(message, hashString(message))
	if err != nil {
		return err
	}

	stop := make(chan bool)
	dgvoice.OnError = func(str string, err error) {
		logrus.WithError(err).Errorln(str)
	}
	dgvoice.PlayAudioFile(v.Connection, path, stop)
	close(stop)

	return nil
}

// hashString takes an input string and returns its SHA-256 hash.
func hashString(input string) string {
	hash := sha256.New()
	hash.Write([]byte(input))
	return hex.EncodeToString(hash.Sum(nil))
}
