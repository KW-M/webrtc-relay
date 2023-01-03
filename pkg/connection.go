package webrtc_relay

import (
	"time"

	relay_config "github.com/kw-m/webrtc-relay/pkg/config"
	"github.com/kw-m/webrtc-relay/pkg/proto"
	"github.com/kw-m/webrtc-relay/pkg/util"
	peerjs "github.com/muka/peerjs-go"
	peerjsServer "github.com/muka/peerjs-go/server"
	"github.com/pion/webrtc/v3"
	log "github.com/sirupsen/logrus"
)

// WebrtcConnectionCtrl: This is the main controller in charge of maintaining an open peer and accepting/connecting to other peers.
// While the fields here are public, they are NOT meant to be modified by the user, do so at your own risk.
type WebrtcConnectionCtrl struct {
	// map of relayPeers owned by this WebrtcConnectionCtrl (key is the RelayPeerNumber specified in the peerInitConfig or when calling addRelayPeer())
	RelayPeers map[uint32]*RelayPeer
	// the config for this WebrtcRelay
	config relay_config.WebrtcRelayConfig
	// The (json) store used to keep peer server tokens between sessions
	tokenStore *TokenPersistanceStore
	// eventStream is the eventSub that this controller should push events to
	eventStream *util.EventSub[proto.RelayEventStream]
	// the log for this WebrtcConnectionCtrl
	log *log.Entry
	// StopSignal is a signal that can be used to stop the WebrtcConnectionCtrl
	stopSignal util.UnblockSignal
}

func NewWebrtcConnectionCtrl(eventStream *util.EventSub[proto.RelayEventStream], config relay_config.WebrtcRelayConfig, logger *log.Logger) *WebrtcConnectionCtrl {
	return &WebrtcConnectionCtrl{
		config:      config,
		RelayPeers:  make(map[uint32]*RelayPeer),
		tokenStore:  NewTokenPersistanceStore(config.TokenPersistanceFile, logger),
		eventStream: eventStream,
		log:         logger.WithFields(log.Fields{"mod": "ConnCtrl"}),
		stopSignal:  util.NewUnblockSignal(),
	}
}

// AddRelayPeer creates a new peerjs peer and connects it to the specified peer server based on the passed config
// the peer will get added to the list of peers that this connection controller is managing.
// call stopRelayPeer() to stop the peer.
func (conn *WebrtcConnectionCtrl) AddRelayPeer(opts *relay_config.PeerInitOptions, exchangeId uint32) {
	if opts == nil || opts.RelayPeerNumber == 0 {
		conn.log.Error("AddRelayPeer: invalid config! Make sure the config has a unique RelayPeerNumber greater than 0.")
	}

	// start a local peerjs server if it is enabled for this PeerInitConfig
	if opts.StartLocalServer {
		peerServerOptions := relay_config.PeerServerOptsFromInitOpts(opts)
		go conn.startLocalPeerJsServer(peerServerOptions)
		<-time.After(time.Second * 2) // wait a second to let the server start up
		println("---2---")
	}

	// start the RelayPeer for this PeerInitConfig
	peerOptions := relay_config.PeerOptsFromInitOpts(opts)
	go conn.setupRelayPeer(&peerOptions, opts.RelayPeerNumber, exchangeId)
}

// StopRelayPeer: stops the relay peer with the specified relayPeerNumber and removes it from the list of peers this connection controller is managing.
func (conn *WebrtcConnectionCtrl) StopRelayPeer(relayPeerNumber uint32, exchangeId uint32) {
	relayPeer := conn.RelayPeers[relayPeerNumber]
	if relayPeer == nil {
		conn.log.Warnf("StopRelayPeer: no relay peer with number %d found!", relayPeerNumber)
	} else {
		relayPeer.Cleanup()
		delete(conn.RelayPeers, relayPeerNumber)
	}
	conn.log.Warn("TODO:!! - StopRelayPeer: stop relay server if it was started by this relay")
}

/* startLocalPeerJsServer starts up a local PeerJs SERVER on this computer. This can be used when no internet access is available or you don't want requests leaving the local network.
 */
func (conn *WebrtcConnectionCtrl) startLocalPeerJsServer(serverOptions peerjsServer.Options) *peerjsServer.PeerServer {
	var server *peerjsServer.PeerServer
	for {
		log.Debugf("Starting local peerjs server... ServerConfig: %+v", serverOptions)
		server = peerjsServer.New(serverOptions)
		if err := server.Start(); err != nil {
			log.Printf("Error starting local peerjs server: %s", err)
			time.Sleep(time.Second * 1)
			continue
		}
		break
	}

	return server
}

/* setupRelayPeer (blocking goroutine)
 * This function sets up the peerjs peer for the relay
 * Then it waits for the peerjs server to "Open" initilize the
 * relay peer which then passes controll to the peerConnectionOpenHandler function.
 * This function also handles the "error", "disconnected" and "closed" events for the peerjs server connection.
 * This function is blocking and will not return until the peer connection fails (with the error) or Relay.stopRelaySignal is triggered.
 */
func (conn *WebrtcConnectionCtrl) setupRelayPeer(peerOptions *peerjs.Options, relayPeerNumber uint32, exchangeId uint32) {
	// create a new RelayPeer class and add it to the map of relayPeers
	relayPeer := NewRelayPeer(conn, *peerOptions, 0, relayPeerNumber)
	conn.RelayPeers[relayPeerNumber] = relayPeer
	relayPeer.SetSavedExchangeId(exchangeId)

	go func() {
		for {
			select {
			case state := <-relayPeer.currentState:
				if state == RELAY_PEER_CONNECTING {
					conn.log.Infof("RelayPeer %d (%s) is now connecting.", relayPeerNumber, peerOptions.Host)
				} else if state == RELAY_PEER_CONNECTED {
					conn.log.Infof("RelayPeer %d (%s) is now connected.", relayPeerNumber, peerOptions.Host)
					conn.sendRelayConnectedEvent(relayPeerNumber)
				} else if state == RELAY_PEER_DISCONNECTED {
					conn.log.Infof("RelayPeer %d (%s) is now disconnected.", relayPeerNumber, peerOptions.Host)
					conn.sendRelayDisconnectedEvent(relayPeerNumber)
				} else if state == RELAY_PEER_RECONNECTING {
					conn.log.Infof("RelayPeer %d (%s) is now reconnecting.", relayPeerNumber, peerOptions.Host)
				} else if state == RELAY_PEER_DESTROYED {
					conn.log.Warnf("RelayPeer %d (%s) is now destroyed.", relayPeerNumber, peerOptions.Host)
					conn.sendRelayErrorEvent(relayPeerNumber, proto.RelayErrorTypes_RELAY_DESTROYED, "RelayPeer has been destroyed.")
				}
			case <-conn.stopSignal.GetSignal():
				conn.log.Debug("Exiting setupRelayPeer loop.")
				relayPeer.Cleanup()
				return
			}
		}
	}()

	// start the peer connection
	for {
		err := relayPeer.Start(conn.onConnection, conn.onCall, conn.onRelayError)
		if err == nil {
			break
		}
		conn.log.Warnf("Failed to start RelayPeer %d, retrying in %d seconds... %s", relayPeerNumber, relayPeer.expBackoffErrorCount, err.Error())
		time.Sleep(time.Second * time.Duration(relayPeer.expBackoffErrorCount))
	}

	conn.log.Infof("RelayPeer %d started.", relayPeerNumber)
}

/* peerConnectionOpenHandler (non-blocking function)
 * This function sets up the event listeners for the relayPeer object that accept new webrtc peer connections to the relay and handle errors & sutch
 * This loop also handles specific errors like offline relay state, by switching to offline mode, and peer id taken, by incrementing the peerid postfix number before trying again.
 * This function should be called within the peer.On("open",) function of the relayPeer object.
 * This function DOES NOT block, BUT the passed relayPeer parameter MUST NOT GO OUT OF SCOPE, or the event listeners will be garbage collected and (maybe) closed.
 */
func (conn *WebrtcConnectionCtrl) peerConnectionOpenHandler(dataConn *peerjs.DataConnection, relayPeerNumber uint32) {
	var clientPeerId string = dataConn.GetPeerID()
	log := conn.log

	// push out an event that a new peer has connected
	log.Info("Connection established with Peer: ", dataConn.GetPeerID())
	conn.sendPeerConnectedEvent(relayPeerNumber, dataConn.GetPeerID())

	// --- Handle Events on this datachannel

	dataConn.On("close", func(_ interface{}) {
		// push out an event that this peer connection has been closed
		conn.sendPeerDataConnErrorEvent(relayPeerNumber, dataConn.GetPeerID(), proto.PeerConnErrorTypes_CONNECTION_CLOSED, "Connection closed")
	})

	dataConn.On("disconnected", func(_ interface{}) {
		// push out an event that this peer connection has disconnected
		conn.sendPeerDisconnectedEvent(relayPeerNumber, dataConn.GetPeerID())
	})

	dataConn.On("error", func(message interface{}) {
		errMessage := message.(error).Error()
		// TODO: fix error message mapping
		errorType, ok := proto.PeerConnErrorTypes_value[errMessage]
		if !ok {
			errorType = int32(proto.PeerConnErrorTypes_UNKNOWN_ERROR)
		}
		// push out an event that this peer connection has had an error
		conn.sendPeerDataConnErrorEvent(relayPeerNumber, dataConn.GetPeerID(), proto.PeerConnErrorTypes(errorType), errMessage)
	})

	// handle incoming messages from this peer connection
	dataConn.On("data", func(msgBytes interface{}) {
		/* forwards the passed message string (coming from the client/browser via the datachannel) to the backend (named pipe or go code) */
		conn.sendMsgRecivedEvent(relayPeerNumber, clientPeerId, msgBytes.([]byte))
	})

}

func (conn *WebrtcConnectionCtrl) onRelayError(err peerjs.PeerError, relayPeerNumber uint32) {
	errorType, ok := proto.RelayErrorTypes_value[err.Type]
	if !ok {
		errorType = int32(proto.RelayErrorTypes_UNKNOWN)
	}
	conn.sendRelayErrorEvent(relayPeerNumber, proto.RelayErrorTypes(errorType), err.Error())
}

// TODO: implement this
func (conn *WebrtcConnectionCtrl) onCall(mediaConn *peerjs.MediaConnection, relayPeerNumber uint32) {
	conn.log.Warn("onCall: not fully implemented!")
	tracks := []*proto.TrackInfo{}
	if stream := mediaConn.GetRemoteStream(); stream != nil {
		for _, track := range stream.GetTracks() {
			trackInfo := peerjsTrackToTrackInfo(track.(*webrtc.TrackRemote))
			tracks = append(tracks, trackInfo)
		}
		conn.sendPeerCalledEvent(relayPeerNumber, mediaConn.GetPeerID(), mediaConn.Label, tracks)
	} else {
		conn.log.Warn("onCall: waiting for remote stream!")
		mediaConn.On("stream", func(stream interface{}) {
			rStream := stream.(*peerjs.MediaStream)
			for _, track := range rStream.GetTracks() {
				trackInfo := peerjsTrackToTrackInfo(track.(*webrtc.TrackRemote))
				tracks = append(tracks, trackInfo)
			}
			conn.sendPeerCalledEvent(relayPeerNumber, mediaConn.GetPeerID(), mediaConn.Label, tracks)
		})
	}
}

// onConnection: called when another peer connects to any of the relay peers
func (conn *WebrtcConnectionCtrl) onConnection(dataConn *peerjs.DataConnection, relayPeerNumber uint32) {
	if dataConn.Open {
		conn.peerConnectionOpenHandler(dataConn, relayPeerNumber)
	} else {
		dataConn.On("open", func(_ interface{}) {
			conn.peerConnectionOpenHandler(dataConn, relayPeerNumber)
		})
	}
}

func peerjsTrackToTrackInfo(track *webrtc.TrackRemote) *proto.TrackInfo {
	codec := track.Codec()
	RTCPFeedback := []*proto.RTCPFeedback{}
	for _, fb := range codec.RTCPFeedback {
		RTCPFeedback = append(RTCPFeedback, &proto.RTCPFeedback{
			Type:      fb.Type,
			Parameter: fb.Parameter,
		})
	}
	channelCount := uint32(codec.Channels)
	payloadType := uint32(codec.PayloadType)
	codecParams := &proto.RTPCodecParams{
		MimeType:     codec.MimeType,
		ClockRate:    &codec.ClockRate,
		Channels:     &channelCount,
		PayloadType:  &payloadType,
		RTCPFeedback: RTCPFeedback,
		SDPFmtpLine:  &codec.SDPFmtpLine,
	}
	return &proto.TrackInfo{
		Name:  track.ID(),
		Kind:  track.Kind().String(),
		Codec: codecParams,
	}
}
