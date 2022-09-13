package webrtc_relay

import (
	"os"

	log "github.com/sirupsen/logrus"
)

type WebrtcRelay struct {
	// Log: The logrus logger to use for debug logs within WebrtcRelay Code
	Log *log.Entry
	// RelayInputMessageChannel: Push a message onto this channel to send that message to any or all open datachannels (ie: to the client(s))
	// NOTE: mesages sent are expected to have metadata json & separtor string before the actual message (if config.AddMetadataToBackendMessages is true)
	RelayInputMessageChannel chan string
	// RelayOutputMessageChannel: Whenever a message is recived from any open datachannel, it is pushed onto this channel.
	// NOTE: messages from this channel will contain prepended metadata json & separtor string before the actual message (if config.AddMetadataToBackendMessages is true)
	RelayOutputMessageChannel chan string
	// ConnCtrl: The connection controller to use for this webrtcRelay
	ConnCtrl *WebrtcConnectionCtrl

	// --- Private Fields ---
	// Config options for this WebrtcRelay
	config WebrtcRelayConfig
	// The signal to stop the WebrtcRelay
	stopRelaySignal *UnblockSignal
}

func CreateWebrtcRelay(config WebrtcRelayConfig) *WebrtcRelay {

	// Set up the logrus logger
	var lo *log.Entry = log.WithField("mod", "webrtc-relay")
	level, err := StringToLogLevel(config.LogLevel)
	if err != nil {
		lo.Warn(err)
	}
	lo.Logger.SetLevel(level)
	lo.Logger.SetFormatter(&log.TextFormatter{
		// DisableColors:    true,
		DisableTimestamp: true,
		DisableQuote:     true,
	})

	return &WebrtcRelay{
		Log:                       lo,
		stopRelaySignal:           NewUnblockSignal(),
		config:                    config,
		RelayOutputMessageChannel: make(chan string),
		RelayInputMessageChannel:  make(chan string),
		ConnCtrl:                  nil,
	}
}

// Starts the webrtc-relay
// should be called as a goroutine (blocking)
func (relay *WebrtcRelay) Start() {
	config := relay.config

	defer func() {
		if r := recover(); r != nil {
			relay.Log.Println("Panicked in Start(), stopping webrtc-relay...", r)
			relay.stopRelaySignal.Trigger()
		}
	}()

	relay.Log.Debug("Creating named pipe relay folder: ", config.NamedPipeFolder)
	os.MkdirAll(config.NamedPipeFolder, os.ModePerm)

	if relay.config.CreateDatachannelNamedPipes {
		// Create the two named pipes to send and receive data to / from the webrtc-relay user's backend code
		var toDcPipePath string = config.NamedPipeFolder + "to_webrtc_relay.pipe"
		var fromDcPipePath string = config.NamedPipeFolder + "from_webrtc_relay.pipe"
		relay.Log.Debug("Making Named pipes: " + toDcPipePath + " & " + fromDcPipePath)
		var msgPipe, err = CreateDuplexNamedPipeRelay(toDcPipePath, fromDcPipePath, 0666, 3)
		if err != nil {
			relay.Log.Fatal("Failed to create message relay named pipe: ", err)
		}
		defer msgPipe.Close()

		go msgPipe.RunPipeLoops()
		go func() {
			for {
				select {
				case msg := <-relay.RelayOutputMessageChannel:
					if relay.config.IncludeMessagesInLogs {
						relay.Log.Debug("RELAY->BKEND: ", msg)
					}
					msgPipe.SendMessageToPipe(msg)
				case msg := <-msgPipe.MessagesFromPipeChannel:
					if relay.config.IncludeMessagesInLogs {
						relay.Log.Debug("BKEND->RELAY: ", msg)
					}
					// relay.RelayInputMessageChannel <- msg
				case <-relay.stopRelaySignal.GetSignal():
					return
				}
			}
		}()
	}

	// Setup the peerjs client to accept webrtc connections
	relay.ConnCtrl = NewWebrtcConnectionCtrl(relay)
	go relay.ConnCtrl.Start(relay.stopRelaySignal)

	// Wait for the stop signal
	relay.stopRelaySignal.Wait()
}

// Stops & cleans up the webrtc-relay (non-blocking)
func (relay *WebrtcRelay) Stop() {
	relay.stopRelaySignal.Trigger()
}
