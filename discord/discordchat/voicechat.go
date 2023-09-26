package discordchat

import (
	"aika/ai"
	"aika/discord/discordai"
	"aika/voice"
	"aika/voice/transcoding"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
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
	Speaker  voice.TTS
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
		ai.LanguageModel_GPT35, // voice needs to be fast
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
		WithField("latency", time.Since(start).String()).
		Infoln("voice chat log")

	response := chat.replaceMarkdownLinks(res.Content)

	// if she just left the chat for w/e reason she can't talk back
	if chat.Connection == nil {
		return
	}

	// WIP
	err = chat.streamSpeech(response)
	if err != nil {
		logrus.WithError(err).Errorln("failed to stream tts")
		return
	}

	return

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
	full_start := time.Now()

	duration, err := transcoding.GetDiscordDuration(packets)
	if err != nil {
		logrus.WithError(err).Errorln("failed to decode speaking duration")
		return
	}

	// drop audio clips too short to process
	if duration < (time.Millisecond * 500) {
		logrus.WithField("duration", duration.String()).Debugln("audio clip too short")
		return
	}

	// lock here since aika is processing an existing message
	locked := vc.Mutex.TryLock()
	if !locked {
		// she's processing some other spoken message
		logrus.
			WithField("speaker", speakerID).
			Warnln("missed spoken message due to processing")

		return
	}
	defer vc.Mutex.Unlock()

	member, err := vc.Session.State.Member(vc.ChatID, speakerID)
	if err != nil {
		logrus.WithError(err).Errorln("failed to get member")
	}

	//
	// Speech to text
	//

	stt_start := time.Now()
	// encode voice snippet to wave file
	waveFile, err := transcoding.DiscordToFile(packets, "assets/audio")
	if err != nil {
		logrus.WithError(err).Errorln("failed to encode audio message")
		return
	}

	text, err := vc.Brain.SpeechToText(vc.Ctx, waveFile)
	if err != nil {
		logrus.WithError(err).Errorln("failed whisper transcription")
		return
	}
	stt_latency := time.Since(stt_start)

	//
	// Text Generation
	//

	// TODO: split this out for more accurate metrics

	//
	// Text To Speech
	//
	chat_start := time.Now()
	vc.OnMessage(member.User, text)

	logrus.
		WithField("clip", duration.String()).
		WithField("text", text).
		WithField("sender", member.User.Username).
		WithField("latency_stt", stt_latency.String()).
		WithField("latency_tts", time.Since(chat_start).String()).
		WithField("latency_full", time.Since(full_start).String()).
		Debug("audio chat handling done")
}

// stream the content to voice via TTS
func (vc *Voice) streamSpeech(content string) error {
	pr, pw := io.Pipe()

	group := errgroup.Group{}
	group.SetLimit(2)

	// routine for streaming MP3 content down
	group.Go(func() error {
		defer pw.Close() // close the writer here so the transcoder knows when it's done

		err := vc.Speaker.TextToSpeechStream(content, pw)
		if err != nil {
			return fmt.Errorf("failed to stream tts; %w", err)
		}

		return nil
	})
	// routine for transcoding MP3 to discord send
	group.Go(func() error {
		err := transcoding.StreamMP3ToOpus(pr, vc.Connection.OpusSend)
		if err != nil {
			return fmt.Errorf("failed to transcode mp3 stream; %w", err)
		}

		return nil
	})

	return group.Wait()
}

// generate AUDIO files with speech in sequence
func (vc *Voice) genSpeech(content string) ([]string, error) {
	// break content into spoken lines
	lines := strings.Split(content, "\n")
	text_lines := []string{}
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue // skip blank lines!
		}
		text_lines = append(text_lines, strings.TrimSpace(line))
	}

	// download in parallel on 2 threads
	// https://help.elevenlabs.io/hc/en-us/articles/14312733311761-How-many-requests-can-I-make-and-can-I-increase-it-
	// I can only download 2 in parallel at free teir
	group := errgroup.Group{}
	group.SetLimit(2)

	// populate files with spoken line audio files
	// in ORDER
	files := make([]string, len(text_lines))
	for idx, line := range text_lines {
		// gen tts
		text := line
		entry := idx
		group.Go(func() error {
			path, err := vc.Speaker.TextToSpeech(text, "assets/audio")
			if err != nil {
				return fmt.Errorf("failed tts gen; %w", err)
			}

			// safely push path to files list
			files[entry] = path
			return nil
		})
	}
	// wait for all files to be downloaded
	err := group.Wait()
	if err != nil {
		return nil, err
	}

	// return spoken files in order of lines
	return files, nil
}

// play an AUDIO file in the current voice chat
func (vc *Voice) play(file string) error {
	if vc.Connection == nil {
		return ErrNotConnected
	}

	// convert Mp3 file to Opus frames
	opus, err := transcoding.MP3ToOpus(file)
	if err != nil {
		return err
	}

	// send frames
	vc.Connection.Speaking(true)
	for _, packet := range opus {
		vc.Connection.OpusSend <- packet
		// sleep sampleRate ?
	}
	vc.Connection.Speaking(false)

	/*
		// TODO:
		// rewrite this from scratch
		// so its not doggers
		// we should actually look to pipe in a READER interface
		// this way we can feed it an MP3 stream (for improved efficiency)
		stop := make(chan bool)
		dgvoice.OnError = func(str string, err error) {
			if err != nil {
				logrus.WithError(err).Errorln(str)
			}
		}
		dgvoice.PlayAudioFile(vc.Connection, file, stop)
		close(stop)

	*/
	return nil
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
	Description: "Disconnect from the voice chat.",

	Parameters: jsonschema.Definition{
		Type:       jsonschema.Object,
		Properties: map[string]jsonschema.Definition{},
		Required:   []string{},
	},
}
var definition_joinChannel = openai.FunctionDefinition{
	Name:        "joinVoiceChat",
	Description: "Connect to the sender's voice chat.",

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
