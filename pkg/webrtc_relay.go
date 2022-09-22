package webrtc_relay

import (
	"errors"
	"fmt"

	wr_config "github.com/kw-m/webrtc-relay/pkg/config"
	"github.com/kw-m/webrtc-relay/pkg/media"
	"github.com/kw-m/webrtc-relay/pkg/proto"
	"github.com/kw-m/webrtc-relay/pkg/util"
	"github.com/pion/webrtc/v3"
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
	stopRelaySignal util.UnblockSignal

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
		connCtrl:           NewWebrtcConnectionCtrl(eventStream, config, rLog.Logger),
		stopRelaySignal:    util.NewUnblockSignal(),
	}
}

// Starts the webrtc-relay
func (relay *WebrtcRelay) Start() {
	defer func() {
		if r := recover(); r != nil {
			relay.Log.Println("Panicked in WebrtcRelay.Start(), stopping webrtc-relay...", r)
			relay.stopRelaySignal.Trigger()
		}
	}()

	// Start all of the initial peers specified in the config
	for _, initOptions := range relay.config.PeerInitConfigs {
		relay.connCtrl.AddRelayPeer(initOptions, 0)
	}

	if relay.config.StartGRPCServer {
		go startRelayGRPCServer(relay)
	}

	go func() {
		// handle logging events / input messages and sending input messages to the webrtc controller
		evtStream := relay.eventStream.Subscribe()
		inputStream := relay.inputMessageStream.Subscribe()
		for {
			select {
			case evt := <-evtStream:
				switch event := evt.Event.(type) {
				case *proto.RelayEventStream_MsgRecived:
					if relay.config.IncludeMessagesInLogs {
						relay.Log.Debugf("EVENT MSG: %s", event.MsgRecived.String())
						// relay.Log.Debugf("MSG EVENT->BKEND: %s, (peer %s via relay #%d) ", event.MsgRecived.SrcPeerId, event.MsgRecived.RelayPeerNumber, string(event.MsgRecived.Payload))
					}
				case *proto.RelayEventStream_PeerConnected:
					relay.Log.Debugf("EVENT peer connected: %s (via relay #%d, exId %d)\n", event.PeerConnected.SrcPeerId, evt.ExchangeId, event.PeerConnected.RelayPeerNumber)
				case *proto.RelayEventStream_PeerDisconnected:
					relay.Log.Debugf("EVENT peer disconnected: %s (via relay #%d, exId %d)\n", event.PeerDisconnected.SrcPeerId, evt.ExchangeId, event.PeerDisconnected.RelayPeerNumber)
				case *proto.RelayEventStream_PeerCalled:
					relay.Log.Debugf("EVENT call from peer %s (via relay #%d, exId %d)\n", event.PeerCalled.SrcPeerId, evt.ExchangeId, event.PeerCalled.RelayPeerNumber)
				case *proto.RelayEventStream_PeerHungup:
					relay.Log.Debugf("EVENT hangup from peer %s (via relay #%d, exId %d)\n", event.PeerHungup.SrcPeerId, evt.ExchangeId, event.PeerHungup.RelayPeerNumber)
				case *proto.RelayEventStream_PeerDataConnError:
					relay.Log.Debugf("EVENT peer data connection error from peer: %s (via relay #%d, exId %d) type=%s %s\n", event.PeerDataConnError.SrcPeerId, evt.ExchangeId, event.PeerDataConnError.RelayPeerNumber, event.PeerDataConnError.Type.String(), event.PeerDataConnError.Msg)
				case *proto.RelayEventStream_PeerMediaConnError:
					relay.Log.Debugf("EVENT peer media connection error from peer: %s (via relay #%d, exId %d) type=%s %s\n", event.PeerMediaConnError.SrcPeerId, evt.ExchangeId, event.PeerMediaConnError.RelayPeerNumber, event.PeerMediaConnError.Type.String(), event.PeerMediaConnError.Msg)
				case *proto.RelayEventStream_RelayError:
					relay.Log.Debugf("EVENT relay error: [type=%s] %s (exId %d)\n", event.RelayError.Type.String(), event.RelayError.Msg, evt.ExchangeId)
				case *proto.RelayEventStream_RelayConnected:
					relay.Log.Debugf("EVENT relay connected: %d (exId %d)\n", event.RelayConnected.RelayPeerNumber, evt.ExchangeId)
				case *proto.RelayEventStream_RelayDisconnected:
					relay.Log.Debugf("EVENT relay disconnected: %d (exId %d)\n", event.RelayDisconnected.RelayPeerNumber, evt.ExchangeId)
				default:
					fmt.Println("No matching operations")
				}
			case msg := <-inputStream:
				if relay.config.IncludeMessagesInLogs {
					// relay.Log.Debug("SENDING MSG (targetPeers=%v | via relay #%d): %s", msg)
					relay.Log.Debugf("SENDING MSG (targetPeers=%v | via relay #%d | exId %d): %s", msg.TargetPeerIds, msg.RelayPeerNumber, msg.GetExchangeId(), string(msg.Payload))
				}
				relay.connCtrl.sendMessageToPeers(msg.GetTargetPeerIds(), msg.GetRelayPeerNumber(), msg.GetPayload(), msg.GetExchangeId())
			case <-relay.stopRelaySignal.GetSignal():
				relay.Log.Debug("Stopping webrtc-relay...")
				return
			}
		}
	}()
}

// Stops & cleans up the webrtc-relay (non-blocking)
func (relay *WebrtcRelay) Stop() {
	relay.stopRelaySignal.Trigger()
}

func (relay *WebrtcRelay) GetEventStream() <-chan *proto.RelayEventStream {
	return relay.eventStream.Subscribe()
}

// ConnectToPeer: Attempts to connect to a peerjs peer
// Param peerId (string): The peerId of the peer to connect to
// Param relayPeerNumber (int): The relayPeerNumber to use for the connection (if 0, every RelayPeer will attempt to connect to the peer in parallel)
func (relay *WebrtcRelay) ConnectToPeer(peerId string, relayPeerNumber uint32, exchangeId uint32) {
	relay.connCtrl.connectToPeer(peerId, relayPeerNumber, exchangeId)
}

// DisconnectFromPeer: Closes the connection with a peer
// Param peerId (string): The peerId of the peer to disconnect from
// Param relayPeerNumber (int): The relayPeerNumber of the relay peer you want to close the connection on (if 0, every RelayPeer will attempt to connect to the peer in parallel)
func (relay *WebrtcRelay) DisconnectFromPeer(peerId string, relayPeerNumber uint32, exchangeId uint32) {
	relay.connCtrl.disconnectFromPeer(peerId, relayPeerNumber, exchangeId)
}

// CallPeer: Calls a peerjs peer with a media stream and one track (audio/video)
// Param targetPeerIds ([]string): The peerIds of the peers to call or []string{"*"} to call all peers
// Param relayPeerNumber (int): The relayPeerNumber of the relay peer you want to close the connection on (if 0, every RelayPeer will attempt to connect to the peer in parallel)
// Param streamName (string): The name of the media channel stream to use
// Param trackName (string): The name of the track within the stream to add (empty string will not add a track)
// Param rtpSourceUrl (string): The url of the rtp source to use for the track (empty string will not add a track)
func (relay *WebrtcRelay) CallPeers(targetPeerIds []string, relayPeerNumber uint32, streamName string, tracks []*proto.TrackInfo, exchangeId uint32) error {
	if len(tracks) > 1 {
		return errors.New("more than one track isn't supported for now")
	}

	for _, track := range tracks {
		params := webrtc.RTPCodecParameters{
			RTPCodecCapability: webrtc.RTPCodecCapability{
				MimeType:    track.Codec.GetMimeType(),
				ClockRate:   track.Codec.GetClockRate(),
				Channels:    uint16(track.Codec.GetChannels()),
				SDPFmtpLine: track.Codec.GetSDPFmtpLine(),
			},
			PayloadType: webrtc.PayloadType(track.Codec.GetPayloadType()),
		}
		_, err := relay.mediaCtrl.AddRtpTrack(track.Name, track.Kind, track.GetRtpSourceUrl(), params)
		if err != nil {
			return err
		}
	}

	// TODO: Support multiple tracks

	relay.connCtrl.streamTracksToPeers(targetPeerIds, relayPeerNumber, []string{tracks[0].Name}, relay.mediaCtrl, exchangeId)
	return nil
}

// AddMediaTrack: add aditional tracks to an open media call
// can be called again with the same peerid and stream name to  (audio/video)
// can be called again with the same peerid, stream name and trackName to replace the track (pass an empty rtpSourceUrl to stop the track)
// Param targetPeerIds ([]string): The peerIds of the peers to call or []string{"*"} to call all peers
// Param relayPeerNumber (int): The relayPeerNumber of the relay peer you want to close the connection on (if 0, every RelayPeer will attempt to connect to the peer in parallel)
// Param streamName (string): The name of the media channel stream to use
// Param trackName (string): The name of the track within the stream we are adding / replacing
// Param rtpSourceUrl (string): The url of the rtp source to use for the track (empty string will stop the track if it exists)
func (relay *WebrtcRelay) AddMediaTrack(targetPeerIds []string, relayPeerNumber uint32, streamName string, trackName string, mimeType string, rtpSourceUrl string, exchangeId uint32) error {
	relay.Log.Warn("TODO: Implement AddMediaTrack()")
	// _, err := relay.mediaCtrl.AddRtpTrack(trackName, mimeType, rtpSourceUrl)
	// if err != nil {
	// 	return err
	// }
	// relay.connCtrl.streamTrackToPeers(targetPeerIds, relayPeerNumber, trackName, relay.mediaCtrl, exchangeId)
	return nil
}

// AddMediaTrackDirect: Calls a peerjs peer with a pion media track object
// can be called again with the same peerid and stream name to add aditional tracks (audio/video)
// can be called again with the same peerid, stream name and trackName to replace the track (pass an empty rtpSourceUrl to stop the track)
// Param targetPeerIds ([]string): The peerIds of the peers to call or []string{"*"} to call all peers
// Param relayPeerNumber (int): The relayPeerNumber of the relay peer you want to close the connection on (if 0, every RelayPeer will attempt to connect to the peer in parallel)
// Param streamName (string): The name of the media channel stream to use
// Param trackName (string): The name of the track within the stream we are adding / replacing
// Param rtpSourceUrl (string): The url of the rtp source to use for the track (empty string will stop the track if it exists)
func (relay *WebrtcRelay) AddMediaTrackDirect(targetPeerIds []string, relayPeerNumber uint32, streamName string, trackName string, rtpSourceUrl string, exchangeId uint32) error {
	relay.Log.Warn("TODO: Implement AddMediaTrackDirect()")
	return nil
}

// HangupPeer: Closes the media call with a peerjs peer
// Param peerId (string): The peerId of the peer to hangup
func (relay *WebrtcRelay) HangupPeer(peerId string, relayPeerNumber uint32, exchangeId uint32) {
	relay.connCtrl.stopMediaStream(relay.mediaCtrl, exchangeId)
}

// SendMsg: Sends a message to one or more peerjs peer(s)
func (relay *WebrtcRelay) SendMsg(targetPeerIds []string, msgPayload []byte, exchangeId uint32) {
	// relay.connCtrl.sendMessageToPeers(targetPeerIds, 0, msgPayload, exchangeId)
	relay.inputMessageStream.Push(&proto.SendMsgRequest{
		TargetPeerIds: targetPeerIds,
		ExchangeId:    &exchangeId,
		Payload:       msgPayload,
	})
}
