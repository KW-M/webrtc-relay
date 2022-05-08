package webrtc_relay_core

import (
	"os"

	log "github.com/sirupsen/logrus"
)

type WebrtcRelay struct {
	// Log: The logrus logger to use for debug logs within WebrtcRelay Code
	Log *log.Entry
	// SendToDatachannelMessages: Push a message onto this channel to send that message to any or all open datachannels (ie: to the client(s))
	// NOTE: mesages sent are expected to have metadata json & separtor string before the actual message (if config.AddMetadataToPipeMessages is true)
	SendToDatachannelMessages chan string
	// FromDatachannelMessages: Whenever a message is recived from any open datachannel, it is pushed onto this channel.
	// NOTE: messages from this channel will contain prepended metadata json & separtor string before the actual message (if config.AddMetadataToPipeMessages is true)
	FromDatachannelMessages chan string
	// ConnCtrl: The connection controller to use for this webrtcRelay
	ConnCtrl *WebrtcConnectionCtrl

	// --- Private Fields ---
	// Config options for this WebrtcRelay
	config *ProgramConfig
	// The signal to stop the WebrtcRelay
	stopRelaySignal *UnblockSignal
}

func CreateWebrtcRelay(configOptions *ProgramConfig) *WebrtcRelay {

	// Set up the logrus logger
	var lo *log.Entry = log.WithField("|", "webrtc-relay")
	level, err := StringToLogLevel(configOptions.LogLevel)
	if err != nil {
		lo.Warn(err)
	}
	lo.Logger.SetLevel(level)
	lo.Logger.SetFormatter(&log.TextFormatter{
		DisableColors:    true,
		DisableTimestamp: true,
		DisableQuote:     true,
	})

	return &WebrtcRelay{
		Log:                       log.WithField("|", "webrtc-relay"),
		stopRelaySignal:           NewUnblockSignal(),
		config:                    configOptions,
		FromDatachannelMessages:   make(chan string),
		SendToDatachannelMessages: make(chan string),
		ConnCtrl:                  nil,
	}
}

// Starts the webrtc-relay
// should be called as a goroutine (blocking)
func (relay *WebrtcRelay) Start() {
	config := relay.config
	log := relay.Log

	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered from panic, stopping webrtc-relay...", r)
			relay.stopRelaySignal.Trigger()
		}
	}()

	relay.Log.Debug("Creating named pipe relay folder: ", config.NamedPipeFolder)
	os.MkdirAll(config.NamedPipeFolder, os.ModePerm)

	if relay.config.CreateDatachannelNamedPipes {
		// Create the two named pipes to send and receive data to / from the webrtc-relay user's backend code
		var msgPipe, err = CreateDuplexNamedPipeRelay(config.NamedPipeFolder+"to_datachannel_relay.pipe", config.NamedPipeFolder+"from_datachannel_relay.pipe", 4096)
		if err != nil {
			log.Fatal("Failed to create message relay named pipe: ", err)
		}
		defer msgPipe.Close()

		relay.FromDatachannelMessages = msgPipe.SendMessagesToPipe
		relay.SendToDatachannelMessages = msgPipe.GetMessagesFromPipe
		go msgPipe.RunPipeLoops(relay.stopRelaySignal)
	}

	mediaSrc, err := CreateNamedPipeMediaSource(config.NamedPipeFolder+"vido.pipe", 10000, "video/h264", "my-stream")
	if err != nil {
		log.Error("Error creating named pipe media source: ", err)
		return
	}
	cameraLivestreamVideoTrack = mediaSrc.WebrtcTrack
	go mediaSrc.StartMediaStream(relay.stopRelaySignal)

	// Setup the peerjs client to accept webrtc connections
	relay.ConnCtrl = CreateWebrtcConnectionCtrl(relay)
	go relay.ConnCtrl.StartPeerServerConnectionLoop()

	relay.stopRelaySignal.Wait()
}

// Stops & cleans up the webrtc-relay (non-blocking)
func (relay *WebrtcRelay) Stop() {
	relay.stopRelaySignal.Trigger()
}
