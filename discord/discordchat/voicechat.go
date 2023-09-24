package discordchat

import (
	"aika/discord/discordai"
	"aika/voice"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/dgvoice"
	"github.com/bwmarrin/discordgo"
	htgotts "github.com/hegedustibor/htgo-tts"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
	"github.com/sirupsen/logrus"
)

var (
	ErrNotConnected = errors.New("not in voice chat")
)

type Voice struct {
	Chat

	History []openai.ChatCompletionMessage

	// voice stuff
	Connection *discordgo.VoiceConnection
	Session    *discordgo.Session
	SsrcUsers  map[uint32]string

	Receiver *voice.Receiver
}

func (chat *Voice) OnMessage(speaker *discordgo.User, msg string) {

	// convert sender to "chat participant"
	sender := &ChatParticipant{User: speaker}

	// get everyone in the voice chat
	members, err := chat.getChatMembers()
	if err != nil {
		logrus.
			WithError(err).
			WithField("speaker", speaker.Username).
			WithField("message", msg).
			Errorln("failed to get chat members")

		return
	}

	// convert members to Display names
	memberNames := []string{}
	for _, member := range members {
		memberNames = append(memberNames, member.GetDisplayName())
	}

	// TODO: need a better AI brain for voice!
	// will need an interface & multiple brain implementations?
	system := chat.Brain.BuildVoiceSystemMessage(memberNames)
	history := chat.History
	message := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: msg,
		Name:    sender.GetDisplayName(),
	}

	// append voice leave functionality
	funcs := chat.getAvailableFunctions(chat.Session, speaker)
	funcs = append(funcs, chat.GetFunction_LeaveChannel())

	// TODO: update called functions in
	// chat to handle non-text message processing
	start := time.Now()
	history, err = chat.Brain.Process(
		chat.Ctx,
		system,
		history,
		message,
		funcs,
		chat.getLanguageModel(speaker.ID, chat.ChatID),
		chat.getInternalArgs(chat.Session, speaker, chat.ChatID, chat.Connection.ChannelID),
	)
	if err != nil {
		logrus.
			WithError(err).
			WithField("speaker", speaker.Username).
			WithField("message", msg).
			Errorln("failed while processing in brain")

		return
	}

	if len(history) == 0 {
		logrus.
			WithError(err).
			WithField("speaker", speaker.Username).
			WithField("message", msg).
			Errorln("blank history returned from brain.Process")

		return
	}

	// update history
	chat.History = history

	// get response message
	res := history[len(history)-1]

	// TODO: improve this log
	logrus.
		WithField("sender", sender.GetDisplayName()).
		WithField("message", msg).
		WithField("response", res.Content).
		WithField("latency", time.Since(start)).
		Infoln("voice chat log")

	response := chat.replaceMarkdownLinks(res.Content)

	// if she just left the chat for w/e reason she can't talk back
	if chat.Connection == nil {
		return
	}

	// convert response message to audio files
	files, err := chat.genSpeech(response)
	if err != nil {
		logrus.
			WithError(err).
			WithField("speaker", speaker.Username).
			WithField("message", msg).
			Errorln("failed to generate TTS audio")

		return
	}

	// play audio in sequence
	for _, file := range files {
		err = chat.play(file)
		if err != nil {
			logrus.
				WithError(err).
				WithField("speaker", speaker.Username).
				WithField("message", msg).
				Errorln("failed to play TTS audio")

			break
		}
	}
}

// get voice chat members by scanning the voicestates
func (chat *Voice) getChatMembers() ([]*ChatParticipant, error) {

	participants := []*ChatParticipant{}

	gd, err := chat.Session.State.Guild(chat.ChatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild details; %w", err)
	}

	for _, state := range gd.VoiceStates {
		member, err := chat.Session.State.Member(chat.ChatID, state.UserID)
		if err != nil {
			logrus.WithField("state", state).Warnln("failed to get member")
			continue
		}
		if member.User == nil {
			logrus.WithField("member", member).Warnln("nil user in member")
			continue
		}
		// aika can't see other bots (or herself)
		if member.User.Bot {
			continue
		}

		// aika can't talk to people in other channels
		if state.ChannelID != chat.Connection.ChannelID {
			continue
		}

		// aika can't talk to deaf members
		if state.Deaf {
			continue
		}

		// check if the user wants to be visible or not
		presence, err := chat.Session.State.Presence(chat.ChatID, state.UserID)
		if errors.Is(err, discordgo.ErrStateNotFound) {
			continue // user likely offline or some shit
		}
		if err != nil {
			logrus.
				WithError(err).
				WithField("username", member.User.Username).
				WithField("userid", state.UserID).
				Warnln("failed to get presence")
			continue
		}

		// ??? what the fuck?
		// we don't get presence info when they're offline so what the fuck?
		if presence.Status == discordgo.StatusOffline ||
			presence.Status == discordgo.StatusInvisible {
			continue
		}

		participants = append(participants, &ChatParticipant{User: member.User})
	}

	return participants, nil
}

// join voice chat & start voice conversation
func (vc *Voice) JoinVoice(guild string, channel string) error {
	if vc.Connection != nil && vc.Connection.GuildID != guild {
		return errors.New("invalid guild when joining voice")
	}

	// change channel if we're already connected
	if vc.Connection != nil {
		vc.Connection.ChangeChannel(channel, false, false)
		return nil
	}

	// set up to handle recieving communication in 2 second bursts of voice
	vc.Receiver = voice.NewReceiver(time.Second, vc.onSpeakingStop)

	// connect & setup systems
	conn, err := vc.Session.ChannelVoiceJoin(guild, channel, false, false)
	if err != nil {
		return err
	}
	vc.Connection = conn
	// keep ssrc<->user maps synced while connected
	// this does _not_ need reset after changing channel ?
	vc.Connection.AddHandler(vc.speakingHandler)

	// start listening
	go vc.listener()

	return nil
}
func (vc *Voice) LeaveVoice() error {
	if vc.Connection == nil {
		return ErrNotConnected
	}

	vc.Connection.Speaking(false)
	// TODO: this is crashing
	close(vc.Connection.OpusRecv)
	err := vc.Connection.Disconnect()
	vc.Connection = nil

	return err
}

func (vc *Voice) listener() {
	for packet := range vc.Connection.OpusRecv {
		user, ok := vc.SsrcUsers[packet.SSRC]
		if !ok { // drop packets we can't identify
			continue
		}

		// push packet into receiver
		// logrus.WithField("timestamp", packet.Timestamp).Info("recieved")
		vc.Receiver.Push(user, packet)

	}
	logrus.Infoln("no longer listening")
}

// this function will ensure we can convert SSRC ids to discord user IDs
func (vc *Voice) speakingHandler(_ *discordgo.VoiceConnection, vs *discordgo.VoiceSpeakingUpdate) {
	vc.SsrcUsers[uint32(vs.SSRC)] = vs.UserID
}

// called when the speaker has finished speaking (delay configured in receiver init)
// recieves all packets and the speaker
func (vc *Voice) onSpeakingStop(speakerID string, packets []*discordgo.Packet) {
	// if speakerID != "241370201222938626" {
	//	 return // only kegan
	// }

	// lock here rather than OnMessage()
	locked := vc.Mutex.TryLock()
	if !locked {
		// she's processing some other spoken message
		logrus.
			WithField("speaker", speakerID).
			Warnln("missed spoken message due to processing")

		return
	}
	defer vc.Mutex.Unlock()

	// encode voice snippet to wave file
	waveFile, err := voice.EncodeAudio(packets)
	if err != nil {
		logrus.WithError(err).Errorln("failed to encode audio message")
		return
	}

	// transcribe
	start := time.Now()
	text, err := vc.Brain.SpeechToText(vc.Ctx, waveFile)
	if err != nil {
		logrus.WithError(err).Errorln("failed whisper transcription")
		return
	}

	/*
		logrus.
			WithField("text", text).
			Debug("transcribed")*/

	member, err := vc.Session.State.Member(vc.ChatID, speakerID)
	if err != nil {
		logrus.WithError(err).Errorln("failed to get member")
	}

	logrus.
		WithField("text", text).
		WithField("sender", member.User.Username).
		WithField("latency", time.Since(start)).
		Debug("transcribed voice message")

	vc.OnMessage(member.User, text)
	return

	/*
		// this will play back all the packets
		// maintaining the natural delay between them
		logrus.
			WithField("count", len(packets)).
			Infoln("trying to speak!")

		vc.Connection.Speaking(true)
		for i := 0; i < len(packets); i++ {
			packetToSend := packets[i]
			vc.Connection.OpusSend <- packetToSend.Opus
			// delay until the next packet
			if i != (len(packets) - 1) {
				nextPacket := packets[i+1]
				delay := nextPacket.Timestamp - packetToSend.Timestamp
				time.Sleep(time.Duration(delay) * time.Nanosecond)
			}
		}
		vc.Connection.Speaking(false)

	*/

	logrus.Infoln("done speaking")

}

// generate AUDIO files with speech in sequence
func (vc *Voice) genSpeech(content string) ([]string, error) {

	lines := strings.Split(content, "\n")
	files := []string{}
	for idx, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue // skip blank lines!
		}
		// generate TTS for the line
		// TODO: find a better TTS API
		speech := htgotts.Speech{Folder: "assets/audio", Language: "en"}
		path, err := speech.CreateSpeechFile(line, hashString(line))
		if err != nil {
			return nil, err
		}

		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if info.Size() == 1685 {
			logrus.WithField("line", line).WithField("idx", idx).Infoln("htgotts returned bad MP3file")
			return nil, errors.New("failed to gen speech - line too long")
		}

		files = append(files, path)
	}

	return files, nil
}

// play an AUDIO file in the current voice chat
func (vc *Voice) play(file string) error {
	if vc.Connection == nil {
		return ErrNotConnected
	}

	// TODO:
	// rewrite this from scratch
	// so its not doggers
	stop := make(chan bool)
	dgvoice.PlayAudioFile(vc.Connection, file, stop)
	close(stop)

	return nil
}

// hashString takes an input string and returns its SHA-256 hash.
func hashString(input string) string {
	hash := sha256.New()
	hash.Write([]byte(input))
	return hex.EncodeToString(hash.Sum(nil))
}

// ------------- FUNCTIONS for AI to call which can call CONNECT and DISCONNECT

// SCUFFED

func (vc *Voice) GetFunction_JoinChannel() discordai.Function {
	return discordai.Function{
		Definition: definition_joinChannel,
		Handler:    vc.handle_joinChannel,
	}
}
func (vc *Voice) GetFunction_LeaveChannel() discordai.Function {
	return discordai.Function{
		Definition: definition_leaveChannel,
		Handler:    vc.handle_leaveChannel,
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
	err := v.JoinVoice(msgMap["internal_sender_guildid"].(string), msgMap["internal_sender_author_vc"].(string))
	if err != nil {
		return "", err
	}

	return "connected successfully", nil
}

func (v *Voice) handle_leaveChannel(msgMap map[string]interface{}) (string, error) {
	// call function
	err := v.LeaveVoice()
	if err != nil {
		return "", err
	}

	return "disconnected successfully", nil
}
