package discordchat

import (
	"aika/ai"
	"aika/discord/discordai"
	"aika/utils"
	"aika/voice"
	"aika/voice/transcoding"
	"errors"
	"fmt"
	"io"
	"os"
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

func (chat *Voice) streamResponse(speaker *discordgo.User, msg string, output chan string) error {

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

		return fmt.Errorf("failed to get chat participants; %w", err)
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

	pipe := utils.NewStringPipe()

	group := errgroup.Group{}
	group.SetLimit(2)

	// writer routine will start reading in
	// openAI responses & return a final history
	group.Go(func() error {
		defer pipe.Close()

		new_history, err := chat.Brain.ProcessChunked(
			chat.Ctx,
			pipe,
			system,
			history,
			message,
			funcs,
			ai.LanguageModel_GPT4, // voice needs to be fast
			chat.getInternalArgs(chat.Session, speaker, chat.ChatID, chat.Connection.ChannelID),
		)
		if err != nil {
			logrus.
				WithError(err).
				WithField("speaker", speaker.Username).
				WithField("message", msg).
				Errorln("failed while processing in brain")

			return fmt.Errorf("failed to process in brain; %w", err)
		}

		if len(new_history) == 0 {
			logrus.
				WithError(err).
				WithField("speaker", speaker.Username).
				WithField("message", msg).
				Errorln("blank history returned from brain.Process")

			return errors.New("no history returned from AI")
		}

		// update history
		history = new_history

		return nil
	})
	// stream chunked responses to output
	group.Go(func() error {
		for {
			line, err := pipe.Read()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return fmt.Errorf("failed to read pipe; %w", err)
			}
			if strings.TrimSpace(line) == "" {
				continue
			}

			output <- line
		}
		return nil
	})

	if err := group.Wait(); err != nil {
		return fmt.Errorf("failed streaming response; %w", err)
	}

	// update history
	chat.History = history

	return nil
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

	// clean wave file from disk so i don't leak
	os.Remove(waveFile)

	stt_latency := time.Since(stt_start)

	// if she cannot talk (for leaving chat) exit early
	if vc.Connection == nil {
		return
	}

	//
	// Filter out messages that don't mention AIKA (expensive as fuck!)
	//
	if !strings.Contains(strings.ToLower(text), "aika") {
		logrus.WithField("text", text).Debugln("dropped message not mentioning aika")
		return
	}
	if strings.Contains(strings.ToLower(text), "aika, an ai chatbot.") {
		logrus.WithField("text", text).Debugln("dropped message probably maltranslated")
		return
	}

	logrus.
		WithField("clip", duration.String()).
		WithField("input", text).
		WithField("sender", member.User.Username).
		Debug("processing new message")

	speakChan := make(chan string)

	group := errgroup.Group{}
	group.SetLimit(2)

	var ai_latency time.Duration
	var chat_latency time.Duration

	//
	// Text Generation
	//
	group.Go(func() error {
		defer close(speakChan)

		ai_start := time.Now()

		err := vc.streamResponse(member.User, text, speakChan)
		if err != nil {
			return fmt.Errorf("failed to stream response; %w", err)
		}

		ai_latency = time.Since(ai_start)
		return nil
	})

	//
	// Text To Speech
	//
	full_response := ""
	group.Go(func() error {
		chat_start := time.Now()

		for {
			response, ok := <-speakChan
			if !ok {
				break
			}
			full_response += response + "\n"
			err = vc.streamSpeech(response)
			if err != nil {
				return fmt.Errorf("failed to stream tts; %w", err)
			}
		}

		chat_latency = time.Since(chat_start)
		return nil
	})

	err = group.Wait()
	if err != nil {
		logrus.WithError(err).Errorln("failed to talk in chat")
		return
	}

	logrus.
		WithField("clip", duration.String()).
		WithField("input", text).
		WithField("output", full_response).
		WithField("sender", member.User.Username).
		WithField("latency_stt", stt_latency.String()).
		WithField("latency_ai", ai_latency.String()).
		WithField("latency_tts", chat_latency.String()).
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
		//TODO: panic when saying in voice "leave" - she tries to talk back lmfao
		err := transcoding.StreamMP3ToOpus(pr, vc.Connection.OpusSend)
		if err != nil {
			return fmt.Errorf("failed to transcode mp3 stream; %w", err)
		}

		return nil
	})

	return group.Wait()
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
