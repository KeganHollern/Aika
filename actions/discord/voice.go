package discord

import (
	"aika/discord/discordai"
	"errors"
	"fmt"

	"github.com/bwmarrin/discordgo"
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

	logrus.Infoln("JOINED VOICE")

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
	/*
		v.Connection.AddHandler(func(vc *discordgo.VoiceConnection, vs *discordgo.VoiceSpeakingUpdate) {
			if vs.UserID != "241370201222938626" {
				return
			}

			v.Connection.Speaking(vs.Speaking)
		})
	*/

	go func() {
		err := v.Connection.Speaking(true)
		if err != nil {
			logrus.WithError(err).Errorln("failed to set speaking")
			return
		}
		// listen and echo back voices
		for p := range v.Connection.OpusRecv {
			v.Connection.OpusSend <- p.Opus
		}

		v.Connection.Speaking(false)
	}()

	return nil
}
func (v *Voice) action_leaveChannel() error {
	logrus.Println("LEAVE CHAN")
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
