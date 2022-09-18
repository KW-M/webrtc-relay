package webrtc_relay

import (
	wr_config "github.com/kw-m/webrtc-relay/pkg/config"
	"github.com/kw-m/webrtc-relay/pkg/media"
	"github.com/kw-m/webrtc-relay/pkg/proto"
	"github.com/kw-m/webrtc-relay/pkg/util"
	log "github.com/sirupsen/logrus"
)

type WebrtcRelay struct {

	// // RelayInputMessageChannel: Push a message onto this channel to send that message to any or all open datachannels (ie: to the client(s))
	// // NOTE: mesages sent are expected to have metadata json & separtor string before the actual message (if config.AddMetadataToBackendMessages is true)
	// RelayInputMessageChannel chan string
	// // RelayOutputMessageChannel: Whenever a message is recived from any open datachannel, it is pushed onto this channel.
	// // NOTE: messages from this channel will contain prepended metadata json & separtor string before the actual message (if config.AddMetadataToBackendMessages is true)
	// RelayOutputMessageChannel chan string

	// eventStream: The eventSub that all the events from the relay will get pushed to.
	eventStream *util.EventSub[proto.RelayEventStream]

	// inputMessageStream: Message requests pushed onto this EventSub will get sent to the client(s) via the datachannel(s) specified in the message request
	inputMessageStream *util.EventSub[proto.SendMsgRequest]

	// ConnCtrl: The connection controller to use for this webrtcRelay
	connCtrl *WebrtcConnectionCtrl

	// mediaCtrl: The media controller to use for this webrtcRelay
	mediaCtrl *media.MediaController

	// Config options for this WebrtcRelay
	config wr_config.WebrtcRelayConfig

	// The signal used to stop the WebrtcRelay & all its sub-components
	stopRelaySignal *util.UnblockSignal

	// Log: The logrus logger to use for debug logs within WebrtcRelay Code
	Log *log.Entry
}

// Creates a new webrtc-relay instance. Call Start() to start the relay.
//
// Param config (WebrtcRelayConfig): The config options for the webrtc-relay including any relayPeers to start when Start() is run (peerInitOptions)
//
// example:
//
//	import relay "github.com/kw-m/webrtc-relay/pkg"
//	import relay_config "github.com/kw-m/webrtc-relay/pkg/config"
//	relay.NewWebrtcRelay(relay_config.GetDefaultRelayConfig())
func NewWebrtcRelay(config wr_config.WebrtcRelayConfig) *WebrtcRelay {

	// Set up the logrus logger
	var rLog *log.Entry = log.WithField("mod", "webrtc-relay")
	level, err := wr_config.StringToLogLevel(config.LogLevel)
	if err != nil {
		rLog.Warn(err.Error())
	}
	rLog.Logger.SetLevel(level)
	rLog.Logger.SetFormatter(&log.TextFormatter{
		// DisableColors:    true,
		DisableTimestamp: true,
		DisableQuote:     true,
	})

	eventStream := util.NewEventSub[proto.RelayEventStream](1)
	inputMessageStream := util.NewEventSub[proto.SendMsgRequest](1)

	// Create the webrtc-relay
	return &WebrtcRelay{
		Log:                rLog,
		config:             config,
		eventStream:        eventStream,
		inputMessageStream: inputMessageStream,
		mediaCtrl:          media.NewMediaController(),
		connCtrl:           NewWebrtcConnectionCtrl(*eventStream, config, rLog.Logger),
		stopRelaySignal:    util.NewUnblockSignal(),
	}
}

// Starts the webrtc-relay (blocking - should be called as a goroutine)
func (relay *WebrtcRelay) Start() {
	defer func() {
		if r := recover(); r != nil {
			relay.Log.Println("Panicked in WebrtcRelay.Start(), stopping webrtc-relay...", r)
			relay.stopRelaySignal.Trigger()
		}
	}()

	// Start all of the initial peers specified in the config
	for _, initOptions := range relay.config.PeerInitConfigs {
		relay.connCtrl.AddRelayPeer(initOptions)
	}

	// (blocking) handle logging events / input messages and sending input messages to the webrtc controller
	evtStream := relay.eventStream.Subscribe()
	inputStream := relay.inputMessageStream.Subscribe()
	for {
		select {
		case evt := <-evtStream:
			switch event := evt.Event.(type) {
			case *proto.RelayEventStream_MsgRecived:
				if relay.config.IncludeMessagesInLogs {
					relay.Log.Debug("RELAY->BKEND: ", event)
				}
			default:
				relay.Log.Debug("RELAY EVENT: ", event)
			}
		case msg := <-inputStream:
			if relay.config.IncludeMessagesInLogs {
				relay.Log.Debug("BKEND->RELAY: ", msg)
			}
			relay.connCtrl.SendMsg(msg)
		case <-relay.stopRelaySignal.GetSignal():
			relay.Log.Debug("Stopping webrtc-relay...")
			return
		}
	}
}

// Stops & cleans up the webrtc-relay (non-blocking)
func (relay *WebrtcRelay) Stop() {
	relay.stopRelaySignal.Trigger()
}

func (relay *WebrtcRelay) GetEventStream() <-chan proto.RelayEventStream {
	return relay.eventStream.Subscribe()
}

// ConnectToPeer: Attempts to connect to a peerjs peer
// Param peerId (string): The peerId of the peer to connect to
// Param relayPeerNumber (int): The relayPeerNumber to use for the connection (if 0, every RelayPeer will attempt to connect to the peer in parallel)
func (relay *WebrtcRelay) ConnectToPeer(peerId string, relayPeerNumber uint32) error {
	relay.Log.Warn("TODO: Implement ConnectToPeer()")
	return nil
}

// DisconnectFromPeer: Closes the connection with a peer
// Param peerId (string): The peerId of the peer to disconnect from
// Param relayPeerNumber (int): The relayPeerNumber of the relay peer you want to close the connection on (if 0, every RelayPeer will attempt to connect to the peer in parallel)
func (relay *WebrtcRelay) DisconnectFromPeer(peerId string) error {
	relay.Log.Warn("TODO: Implement DisconnectFromPeer()")
	return nil
}

// CallPeer: Calls a peerjs peer with a media stream and one track (audio/video)
// Param peerId (string): The peerId of the peer to call
// Param streamName (string): The name of the media channel stream to use
// Param trackName (string): The name of the track within the stream to add (empty string will not add a track)
// Param rtpSourceUrl (string): The url of the rtp source to use for the track (empty string will not add a track)
func (relay *WebrtcRelay) CallPeer(peerId string, streamName string, trackName string, mimeType string, rtpSourceUrl string) error {
	relay.mediaCtrl.AddRtpTrack(trackName, mimeType, rtpSourceUrl)
	streamTrackToPeers(streamName, trackName, peerId, relay.connCtrl)
	return nil
}

// AddMediaTrack: add aditional tracks to an open media call
// can be called again with the same peerid and stream name to  (audio/video)
// can be called again with the same peerid, stream name and trackName to replace the track (pass an empty rtpSourceUrl to stop the track)
// Param peerId (string): The peerId of the peer to call
// Param streamName (string): The name of the media channel stream to use
// Param trackName (string): The name of the track within the stream we are adding / replacing
// Param rtpSourceUrl (string): The url of the rtp source to use for the track (empty string will stop the track if it exists)
func (relay *WebrtcRelay) AddMediaTrack(peerId string, streamName string, trackName string, mimeType string, rtpSourceUrl string) error {
	relay.Log.Warn("TODO: Implement AddMediaTrack()")
	return nil
}

// AddMediaTrackDirect: Calls a peerjs peer with a pion media track object
// can be called again with the same peerid and stream name to add aditional tracks (audio/video)
// can be called again with the same peerid, stream name and trackName to replace the track (pass an empty rtpSourceUrl to stop the track)
// Param peerId (string): The peerId of the peer to call
// Param streamName (string): The name of the media channel stream to use
// Param trackName (string): The name of the track within the stream we are adding / replacing
// Param rtpSourceUrl (string): The url of the rtp source to use for the track (empty string will stop the track if it exists)
func (relay *WebrtcRelay) AddMediaTrackDirect(peerId string, streamName string, trackName string, rtpSourceUrl string) error {
	relay.Log.Warn("TODO: Implement AddMediaTrackDirect()")
	return nil
}

// HangupPeer: Closes the media call with a peerjs peer
// Param peerId (string): The peerId of the peer to hangup
func (relay *WebrtcRelay) HangupPeer(peerId string) error {
	relay.Log.Warn("TODO: Implement HangupPeer()")
	return nil
}

// SendMsg: Sends a message to one or more peerjs peer(s)
func (relay *WebrtcRelay) SendMsg(targetPeerIds string, msgPayload []byte) error {
	relay.Log.Warn("TODO: Implement SendMsg()")
	return nil
}
