package webrtc_relay

import (
	"fmt"

	wrConfig "github.com/kw-m/webrtc-relay/pkg/config"
	"github.com/kw-m/webrtc-relay/pkg/media"
	"github.com/kw-m/webrtc-relay/pkg/proto"
	"github.com/kw-m/webrtc-relay/pkg/util"

	"github.com/pion/mediadevices"
	"github.com/pion/webrtc/v3"
	log "github.com/sirupsen/logrus"
)

type WebrtcRelay struct {

	// eventStream: The eventSub that all the events from the relay will get pushed to.
	eventStream *util.EventSub[proto.RelayEventStream]

	// inputMessageStream: Message requests pushed onto this EventSub will get sent to the client(s) via the datachannel(s) specified in the message request
	inputMessageStream *util.EventSub[proto.SendMsgRequest]

	// ConnCtrl: The connection controller to use for this webrtcRelay
	connCtrl *WebrtcConnectionCtrl

	// mediaCtrl: The media controller to use for this webrtcRelay
	mediaCtrl *media.MediaController

	// Config options for this WebrtcRelay
	config wrConfig.WebrtcRelayConfig

	// The signal used to stop the WebrtcRelay & all its sub-components
	stopRelaySignal util.UnblockSignal

	// Log: The logrus logger to use for debug logs within WebrtcRelay Code
	Log *log.Entry
}

var stream0, stream1 mediadevices.MediaStream

// Creates a new webrtc-relay instance. Call Start() to start the relay.
//
// Param config (WebrtcRelayConfig): The config options for the webrtc-relay including any relayPeers to start when Start() is run (peerInitOptions)
//
// example:
//
//	import relay "github.com/kw-m/webrtc-relay/pkg"
//	import relay_config "github.com/kw-m/webrtc-relay/pkg/config"
//	relay.NewWebrtcRelay(relay_config.GetDefaultRelayConfig())
func NewWebrtcRelay(config wrConfig.WebrtcRelayConfig) *WebrtcRelay {

	// Set up the logrus logger
	var rLog *log.Entry = log.WithField("mod", "webrtc-relay")
	level, err := wrConfig.StringToLogLevel(config.LogLevel)
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

	// add all the media sources from the config
	for _, mediaSourceConfig := range relay.config.MediaSources {
		relay.mediaCtrl.DevicesWrapper.AddVideoCmdSource(mediaSourceConfig)
	}

	// Start all of the initial peers specified in the config
	for _, initOptions := range relay.config.PeerInitConfigs {
		relay.connCtrl.AddRelayPeer(initOptions, 0)
	}

	if relay.config.StartGRPCServer {
		go startRelayGRPCServer(relay)
	}

	// // DEBUG
	// go func() {
	// 	t := time.NewTicker(5 * time.Millisecond)
	// 	for {
	// 		<-t.C
	// 		now := time.Now().GoString()
	// 		relay.connCtrl.sendMessageToPeers([]string{"*"}, 0, []byte(now), 123)
	// 		relay.Log.Info("Sending msg: ", now)
	// 	}
	// }()

	go func() {
		// handle logging events
		evtStream := relay.eventStream.Subscribe()
		for {
			select {
			case evt := <-evtStream:
				switch event := evt.Event.(type) {
				case *proto.RelayEventStream_MsgRecived:
					if relay.config.IncludeMessagesInLogs {
						relay.Log.Debugf("EVENT MSG: %s", event.MsgRecived.String())
						// relay.Log.Debugf("MSG EVENT->BKEND: %s, (peer %s via relay #%d) ", event.MsgRecived.GetSrcPeerId(), event.MsgRecived.GetRelayPeerNumber(), string(event.MsgRecived.GetPayload()))
						/* debug reply */
						// go func() {
						// 	relay.SendMsg([]string{event.MsgRecived.GetSrcPeerId()}, event.MsgRecived.GetPayload(), event.MsgRecived.GetRelayPeerNumber(), 123)
						// }()
					}
				case *proto.RelayEventStream_PeerConnected:
					relay.Log.Debugf("EVENT peer connected: %s (via relay #%d, exId %d)\n", event.PeerConnected.GetSrcPeerId(), evt.GetExchangeId(), event.PeerConnected.GetRelayPeerNumber())
					relay.AutoCall(event.PeerConnected.GetSrcPeerId())
				case *proto.RelayEventStream_PeerDisconnected:
					relay.Log.Debugf("EVENT peer disconnected: %s (via relay #%d, exId %d)\n", event.PeerDisconnected.GetSrcPeerId(), evt.GetExchangeId(), event.PeerDisconnected.GetRelayPeerNumber())
				case *proto.RelayEventStream_PeerCalled:
					relay.Log.Debugf("EVENT call from peer %s (via relay #%d, exId %d)\n", event.PeerCalled.GetSrcPeerId(), evt.GetExchangeId(), event.PeerCalled.GetRelayPeerNumber())
				case *proto.RelayEventStream_PeerHungup:
					relay.Log.Debugf("EVENT hangup from peer %s (via relay #%d, exId %d)\n", event.PeerHungup.GetSrcPeerId(), evt.GetExchangeId(), event.PeerHungup.GetRelayPeerNumber())
				case *proto.RelayEventStream_PeerDataConnError:
					relay.Log.Debugf("EVENT peer data connection error from peer: %s (via relay #%d, exId %d) type=%s %s\n", event.PeerDataConnError.GetSrcPeerId(), evt.GetExchangeId(), event.PeerDataConnError.GetRelayPeerNumber(), event.PeerDataConnError.GetType().String(), event.PeerDataConnError.GetMsg())
				case *proto.RelayEventStream_PeerMediaConnError:
					relay.Log.Debugf("EVENT peer media connection error from peer: %s (via relay #%d, exId %d) type=%s %s\n", event.PeerMediaConnError.GetSrcPeerId(), evt.GetExchangeId(), event.PeerMediaConnError.GetRelayPeerNumber(), event.PeerMediaConnError.GetType().String(), event.PeerMediaConnError.GetMsg())
				case *proto.RelayEventStream_RelayError:
					relay.Log.Debugf("EVENT relay error: [type=%s] %s (exId %d)\n", event.RelayError.GetType().String(), event.RelayError.GetMsg(), evt.GetExchangeId())
				case *proto.RelayEventStream_RelayConnected:
					relay.Log.Debugf("EVENT relay connected: %d (exId %d)\n", event.RelayConnected.GetRelayPeerNumber(), evt.GetExchangeId())
				case *proto.RelayEventStream_RelayDisconnected:
					relay.Log.Debugf("EVENT relay disconnected: %d (exId %d)\n", event.RelayDisconnected.GetRelayPeerNumber(), evt.GetExchangeId())
				default:
					fmt.Println("No matching operations")
				}
			case <-relay.stopRelaySignal.GetSignal():
				relay.Log.Debug("Stopping webrtc-relay...")
				return
			}
		}
	}()

	go func() {
		// handle passing input messages to the webrtc controller to get sent out to peers
		inputStream := relay.inputMessageStream.Subscribe()
		defer relay.inputMessageStream.UnSubscribe(&inputStream)
		for {
			select {
			case msg := <-inputStream:
				if relay.config.IncludeMessagesInLogs {
					relay.Log.Debugf("SENDING MSG (targetPeers=%v | via relay #%d | exId %d): %s", msg.GetTargetPeerIds(), msg.GetRelayPeerNumber(), msg.GetExchangeId(), string(msg.GetPayload()))
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

func (relay *WebrtcRelay) CloseEventStream(evtStreamChan *<-chan *proto.RelayEventStream) {
	relay.eventStream.UnSubscribe(evtStreamChan)
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

// AddMediaTrackRtpSource: Adds a new rtp-based media track source to the media controller to be used in media calls (does not start a call)
func (relay *WebrtcRelay) AddMediaTrackRtpSource(track *proto.TrackInfo) error {
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
	return nil
}

// AddMediaTrackRawSource: Add a new pion-webrtc media track source to the relay media contoller to be used in media calls (does not start a call)
// TODO Fix docs! can be called again with the same peerid and stream name to add aditional tracks (audio/video)
// can be called again with the same peerid, stream name and trackName to replace the track (pass an empty rtpSourceUrl to stop the track)
// Param targetPeerIds ([]string): The peerIds of the peers to call or []string{"*"} to call all peers
// Param relayPeerNumber (int): The relayPeerNumber of the relay peer you want to close the connection on (if 0, every RelayPeer will attempt to connect to the peer in parallel)
// Param trackName (string): The name of the track within the stream we are adding / replacing
// Param rtpSourceUrl (string): The url of the rtp source to use for the track (empty string will stop the track if it exists)
func (relay *WebrtcRelay) AddMediaTrackRawSource(trackName string) error {
	relay.Log.Warn("TODO: Implement AddMediaTrackDirect()")
	return nil
}

// ReplaceMediaTrackSource: Replaces a media track source with a new one in the media contoller
// TODO: will cause all peer connections using this media track to switch to the new track (possibly renegotiate?)
func (relay *WebrtcRelay) ReplaceMediaTrackSource(trackNameToReplace string, newTrack *proto.TrackInfo, exchangeId uint32) error {
	// if this track source is new, add it to the media controller:
	if relay.mediaCtrl.GetTrack(newTrack.GetName()) == nil && relay.mediaCtrl.GetTrack(trackNameToReplace) == nil {
		relay.AddMediaTrackRtpSource(newTrack)
	} else {
		err, oldTrack := relay.mediaCtrl.RemoveTrack(trackNameToReplace, false)
		if err != nil {
			relay.ReplaceMediaTrackInCalls([]string{"*"}, 0, trackNameToReplace, newTrack, exchangeId)
			oldTrack.Close()
		}
	}
	return nil
}

// CallPeer: Calls one or more peerjs peers with a media channel/stream containing one or more tracks (audio or video)
// Param targetPeerIds ([]string): The peerIds of the peers to call or []string{"*"} to call all peers
// Param relayPeerNumber (int): The relayPeerNumber of the relay peer you want to close the connection on (if 0, every RelayPeer will attempt to connect to the peer in parallel)
// DEPRECATED Param streamId (string): The name of the media channel/stream (must be unique per peer connection)
// Param tracks (trackInfo): The details of the tracks to put in the media call.
// Param exchangeId (uint32): The exchangeId to
func (relay *WebrtcRelay) CallPeers(targetPeerIds []string, relayPeerNumber uint32, tracks []*proto.TrackInfo, exchangeId uint32) error {
	return nil
	trackNames := make([]string, len(tracks))
	for i, track := range tracks {
		trackName := track.GetName()
		trackNames[i] = trackName
		// add all the new tracks to the media controller:
		if relay.mediaCtrl.GetTrack(trackName) == nil {
			relay.AddMediaTrackRtpSource(track)
		}
	}

	relay.connCtrl.streamTracksToPeers(targetPeerIds, relayPeerNumber, trackNames, relay.mediaCtrl, exchangeId)
	return nil
}

func (relay *WebrtcRelay) listenForRTCPPackets(rtpSender *webrtc.RTPSender) {
	// Read incoming RTCP packets
	// Before these packets are returned they are processed by interceptors.
	// For things like NACK this needs to be called.
	rtcpBuf := make([]byte, 1500)
	for {
		if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
			return
		}
	}
}

func (relay *WebrtcRelay) AutoCall(targetPeerId string) {
	// relay.AddMediaTrackRtpSource(track)
	log := relay.Log
	if len(relay.config.AutoStreamMediaSources) == 0 {
		return
	}

	var mediaSources []mediadevices.MediaStream
	for _, mediaSourceLabel := range relay.config.AutoStreamMediaSources {
		if src := relay.mediaCtrl.DevicesWrapper.GetMediaStream(mediaSourceLabel); src == nil {
			log.Warnf("Media source %s not found", mediaSourceLabel)
			continue
		} else {
			mediaSources = append(mediaSources, src)
		}
	}

	peerConns := relay.connCtrl.getPeerConnections([]string{targetPeerId}, 0)
	peerConn := peerConns[0]

	//https://www.cs.auckland.ac.nz/courses/compsci773s1c/lectures/YuY2.htm
	log.Error("Calling remote peer: ", peerConn.TargetPeerId)
	// if a media channel doesn't exist with this peer, create one by calling that peer:
	mediaConn, err := peerConn.RelayPeer.CallPeer(peerConn.TargetPeerId, mediaSources[0].GetVideoTracks()[0], relay.mediaCtrl.GetCallConnectionOptions(), 1)
	if err != nil {
		log.Error("Error media calling remote peer: ", peerConn.TargetPeerId)
		errorType, ok := proto.PeerConnErrorTypes_value[err.Error()]
		if !ok {
			errorType = int32(proto.PeerConnErrorTypes_UNKNOWN_ERROR)
		}
		log.Warn("Error media calling remote peer", proto.PeerConnErrorTypes(errorType))
		return
	}

	// go func() {
	// 	<-time.After(10 * time.Second)
	// 	if stream0 == nil {
	// 		stream0 = relay.mdw.getUserMediaStream("ffmpeg_0")
	// 	}
	// 	// trackSender := mediaConn.PeerConnection.GetSenders()[0]
	// 	// err := trackSender.ReplaceTrack(stream0.GetVideoTracks()[0])
	// 	// if err != nil {
	// 	// 	log.Error("Error replacing track: ", err)
	// 	// }

	// 	// log.Warn("Media: Remove track")
	// 	// err := mediaConn.PeerConnection.RemoveTrack(trackSender)
	// 	// if err != nil {
	// 	// 	log.Error("Error removing track: ", err)
	// 	// }
	// 	// mediaConn.PeerConnection.AddTransceiverFromTrack()
	// 	log.Warn("Media: Add track")
	// 	rtpSender, err := mediaConn.PeerConnection.AddTrack(stream0.GetVideoTracks()[0])
	// 	if err != nil {
	// 		log.Error("Error adding track: ", err)
	// 	}

	// 	log.Warn("Media: listenForRTCPPackets offer")
	// 	err = mediaConn.Renegotiate()
	// 	if err != nil {
	// 		log.Error("Error making offer: ", err)
	// 	}
	// 	go relay.listenForRTCPPackets(rtpSender)

	// }()

	mediaConn.PeerConnection.OnNegotiationNeeded(func() {
		log.Warn("Media: PeerConnection.OnNegotiationNeeded")
	})
}

// HangupPeer: Closes the media call with a peerjs peer
// Param peerId (string): The peerId of the peer to hangup
func (relay *WebrtcRelay) HangupPeer(peerId string, relayPeerNumber uint32, exchangeId uint32) {
	relay.connCtrl.stopMediaStream(relay.mediaCtrl, peerId, exchangeId)
}

// AddMediaTrackToCalls: Calls a peerjs peer with a pion media track object
func (relay *WebrtcRelay) AddMediaTrackToCalls(targetPeerIds []string, relayPeerNumber uint32, newTrack *proto.TrackInfo, exchangeId uint32) error {

	// if this track source is new, add it to the media controller:
	if relay.mediaCtrl.GetTrack(newTrack.GetName()) == nil {
		relay.AddMediaTrackRtpSource(newTrack)
	}

	// add the track to all the given peer media connections:
	connections := relay.connCtrl.getPeerConnections(targetPeerIds, relayPeerNumber)
	for _, conn := range connections {
		if conn.MediaConnection != nil {
			conn.MediaConnection.PeerConnection.AddTrack(relay.mediaCtrl.GetTrack(newTrack.GetName()).GetTrack())
		}
	}
	return nil
}

// ReplaceMediaTrackInCalls: Calls a peerjs peer with a pion media track object
func (relay *WebrtcRelay) ReplaceMediaTrackInCalls(targetPeerIds []string, relayPeerNumber uint32, trackNameToReplace string, newTrack *proto.TrackInfo, exchangeId uint32) error {

	// if this track source is new, add it to the media controller:
	if relay.mediaCtrl.GetTrack(newTrack.GetName()) == nil {
		relay.AddMediaTrackRtpSource(newTrack)
	}

	// replace the track in all the given peer media connections:
	connections := relay.connCtrl.getPeerConnections(targetPeerIds, relayPeerNumber)
	for _, conn := range connections {
		if conn.MediaConnection != nil {
			for _, trackSender := range conn.MediaConnection.PeerConnection.GetSenders() {
				trackName := trackSender.Track().ID()
				if trackName == trackNameToReplace {
					trackSender.ReplaceTrack(relay.mediaCtrl.GetTrack(newTrack.GetName()).GetTrack())
				}
			}
		}
	}
	return nil
}

// func (relay *WebrtcRelay) ReplaceMediaTrackInCalls(targetPeerIds []string, relayPeerNumber uint32, trackNamesToReplace []string, newTracks []*proto.TrackInfo, rtpSourceUrl string, exchangeId uint32) error {
// 	relay.Log.Warn("TODO: Implement ReplaceMediaTrackInCalls()")
// 	connections := relay.connCtrl.getPeerConnections(targetPeerIds, relayPeerNumber)
// 	for _, conn := range connections {
// 		if conn.MediaConnection != nil {
// 			for _, trackSender := range conn.MediaConnection.PeerConnection.GetSenders() {
// 				trackName := trackSender.Track().ID()
// 				for i, trackNameToReplace := range trackNamesToReplace {
// 					if trackName == trackNameToReplace {
// 						if i > len(newTracks) {
// 							log.Warnf("No replacement track given for track %s", trackNameToReplace)
// 							break
// 						}
// 						track := newTracks[i]
// 						trackSender.ReplaceTrack(relay.mediaCtrl.GetTrack(track.Name).GetTrack())
// 					}
// 				}
// 			}
// 		}
// 	}
// 	return nil
// }

// SendMsg: Sends a message to one or more peerjs peer(s)
func (relay *WebrtcRelay) SendMsg(targetPeerIds []string, msgPayload []byte, relayPeerNumber uint32, exchangeId uint32) {
	// relay.connCtrl.sendMessageToPeers(targetPeerIds, 0, msgPayload, exchangeId)
	log.Print("PUSHING MESSAGE TO STREAM")
	relay.inputMessageStream.Push(&proto.SendMsgRequest{
		Payload:         msgPayload,
		TargetPeerIds:   targetPeerIds,
		ExchangeId:      &exchangeId,
		RelayPeerNumber: &relayPeerNumber,
	})
}
