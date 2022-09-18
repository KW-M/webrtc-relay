package webrtc_relay

import (
	"time"

	relay_config "github.com/kw-m/webrtc-relay/pkg/config"
	media "github.com/kw-m/webrtc-relay/pkg/media"
	"github.com/kw-m/webrtc-relay/pkg/proto"
	"github.com/kw-m/webrtc-relay/pkg/util"
	peerjs "github.com/muka/peerjs-go"
	peerjsServer "github.com/muka/peerjs-go/server"
	log "github.com/sirupsen/logrus"
)

type MediaTrackData struct {
	Track       *peerjs.MediaStreamTrack // the media track
	MediaSource *media.RtpMediaSource    // the source handler of the media track
}

// WebrtcConnectionCtrl: This is the main controller in charge of maintaining an open peer and accepting/connecting to other peers.
// While the fields here are public, they are NOT meant to be modified by the user, do so at your own risk.
type WebrtcConnectionCtrl struct {
	// the config for this WebrtcRelay
	config relay_config.WebrtcRelayConfig
	// map of relayPeers owned by this WebrtcConnectionCtrl (key is peer server hostname that the relayPeer is connected to)
	RelayPeers map[string]*RelayPeer
	// The (json) store used to keep peer server tokens between sessions
	TokenStore *TokenPersistanceStore
	// eventStream is the eventSub that this controller should push events to
	eventStream util.EventSub[proto.RelayEventStream]
	// the log for this WebrtcConnectionCtrl
	log *log.Entry
	// StopSignal is a signal that can be used to stop the WebrtcConnectionCtrl
	stopSignal util.UnblockSignal
}

func NewWebrtcConnectionCtrl(eventStream util.EventSub[proto.RelayEventStream], config relay_config.WebrtcRelayConfig, logger *log.Logger) *WebrtcConnectionCtrl {
	return &WebrtcConnectionCtrl{
		config:      config,
		RelayPeers:  make(map[string]*RelayPeer),
		TokenStore:  NewTokenPersistanceStore(config.TokenPersistanceFile, logger),
		eventStream: eventStream,
		log:         logger.WithFields(log.Fields{"mod": "ConnCtrl"}),
		stopSignal:  util.NewUnblockSignal(),
	}
}

// AddRelayPeer creates a new peerjs peer and connects it to the specified peer server based on the passed config
// the peer will get added to the list of peers that this connection controller is managing.
// call stopRelayPeer to stop the peer.
func (conn *WebrtcConnectionCtrl) AddRelayPeer(opts *relay_config.PeerInitOptions) {
	if opts == nil || opts.RelayPeerNumber == 0 {
		conn.log.Error("AddRelayPeer: invalid config! Make sure the config has a unique RelayPeerNumber greater than 0.")
	}

	// start a local peerjs server if it is enabled for this PeerInitConfig
	if opts.StartLocalServer {
		peerServerOptions := relay_config.PeerServerOptsFromInitOpts(opts)
		go conn.startLocalPeerJsServer(peerServerOptions)
		<-time.After(time.Second * 1) // wait a second to let the server start up
	}

	// start the RelayPeer for this PeerInitConfig
	peerOptions := relay_config.PeerOptsFromInitOpts(opts)
	go conn.setupRelayPeer(&peerOptions, opts.RelayPeerNumber)
}

/* setupRelayPeer (blocking goroutine)
 * This function sets up the peerjs peer for the relay
 * Then it waits for the peerjs server to "Open" initilize the
 * relay peer which then passes controll to the peerConnectionOpenHandler function.
 * This function also handles the "error", "disconnected" and "closed" events for the peerjs server connection.
 * This function is blocking and will not return until the peer connection fails (with the error) or Relay.stopRelaySignal is triggered.
 */
func (conn *WebrtcConnectionCtrl) setupRelayPeer(peerOptions *peerjs.Options, RelayPeerNumber uint32) error {
	// create a new RelayPeer class and add it to the map of relayPeers
	relayPeer := NewRelayPeer(conn, *peerOptions, 0, RelayPeerNumber)
	conn.RelayPeers[peerOptions.Host] = relayPeer

	go func() {
		for {
			select {
			case state := <-relayPeer.currentState:
				if state == RELAY_PEER_CONNECTING {
					conn.log.Debugf("RelayPeer %s is now connecting.", peerOptions.Host)
				} else if state == RELAY_PEER_CONNECTED {
					conn.log.Infof("RelayPeer %s is now connected.", peerOptions.Host)
				} else if state == RELAY_PEER_DISCONNECTED {
					conn.log.Warnf("RelayPeer %s is now disconnected.", peerOptions.Host)
				} else if state == RELAY_PEER_RECONNECTING {
					conn.log.Warnf("RelayPeer %s is now reconnecting.", peerOptions.Host)
				} else if state == RELAY_PEER_DESTROYED {
					conn.log.Warnf("RelayPeer %s is now destroyed.", peerOptions.Host)
				}
			case <-conn.stopSignal.GetSignal():
				conn.log.Debug("Exiting setupRelayPeer loop.")
				relayPeer.Cleanup()
				return
			}
		}
	}()

	// start the peer connection
	for err := relayPeer.Start(conn.onConnection, conn.onCall); err != nil; {
		conn.log.Warnf("Failed to start RelayPeer %s, retrying in %d seconds... %s", peerOptions.Host, relayPeer.expBackoffErrorCount, err.Error())
		time.Sleep(time.Second * time.Duration(relayPeer.expBackoffErrorCount))
	}

	conn.log.Infof("RelayPeer %s started.", peerOptions.Host)

	return nil
}

/* forwards the passed message string (coming from the client/browser via the datachannel) to the backend (named pipe or go code) */
func (conn *WebrtcConnectionCtrl) handleIncomingDatachannelMessage(msgBytes []byte, srcPeerId string, srcRelayPeerNumber uint32) {
	conn.eventStream.Push(proto.RelayEventStream{
		Event: &proto.RelayEventStream_MsgRecived{
			MsgRecived: &proto.MsgRecivedEvent{
				SrcPeerId:       srcPeerId,
				RelayPeerNumber: srcRelayPeerNumber,
				Payload:         msgBytes,
			},
		},
	})
}

// TODO: implement this
func (conn *WebrtcConnectionCtrl) onCall(mediaConn *peerjs.MediaConnection, relayPeerNumber uint32) {
	conn.log.Warn("onCall: not implemented!")
	trackNames := []string{}
	for _, track := range mediaConn.GetRemoteStream().GetTracks() {
		trackNames = append(trackNames, track.ID())
	}
	conn.eventStream.Push(proto.RelayEventStream{
		Event: &proto.RelayEventStream_PeerCalled{
			PeerCalled: &proto.PeerCalledEvent{
				SrcPeerId:       mediaConn.GetPeerID(),
				RelayPeerNumber: relayPeerNumber,
				StreamName:      mediaConn.Label,
				TrackNames:      trackNames,
			},
		},
	})
}

// onConnection: called when another peer connects to any of the relay peers
func (conn *WebrtcConnectionCtrl) onConnection(dataConn *peerjs.DataConnection, relayPeerNumber uint32) {
	if dataConn.Open {
		conn.peerConnectionOpenHandler(dataConn)
	} else {
		dataConn.On("open", func(_ interface{}) {
			conn.peerConnectionOpenHandler(dataConn)
		})
	}
}

/* startLocalPeerJsServer (blocking goroutine)
 * This function starts up a local PeerJs SERVER on this computer. This can be used when no external internet access is available.
 * This function is blocking and will not return until Relay.stopRelaySignal is triggered or a panic in the server occurs.
 */
func (conn *WebrtcConnectionCtrl) startLocalPeerJsServer(serverOptions peerjsServer.Options) {
	for {
		log.Debug("Starting local peerjs server... ServerConfig: ", serverOptions)
		server := peerjsServer.New(serverOptions)
		defer server.Stop()
		if err := server.Start(); err != nil {
			log.Printf("Error starting local peerjs server: %s", err)
			time.Sleep(time.Second * 1)
			continue
		}
	}

	// wait for the Relay.stopRelaySignal channel to be closed at which point this function will exit and the local peerjs server will stop beacuse of the defer server.stop() function
	conn.stopSignal.Wait()
	return

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

	log.Info("Connection established with Peer: ", dataConn.GetPeerID())

	// push out an event that a new peer has connected
	conn.eventStream.Push(proto.RelayEventStream{
		Event: &proto.RelayEventStream_PeerConnected{
			PeerConnected: &proto.PeerConnectedEvent{
				SrcPeerId:       dataConn.GetPeerID(),
				RelayPeerNumber: relayPeerNumber,
			},
		},
	})

	// --- Handle Events on this datachannel

	dataConn.On("close", func(_ interface{}) {
		log.Infof("CLIENT PEER %s DATACHANNEL CLOSED", dataConn.GetPeerID())
		// push out an event that this peer connection has been closed
		conn.eventStream.Push(proto.RelayEventStream{
			Event: &proto.RelayEventStream_PeerConnected{
				PeerConnected: &proto.PeerConnectedEvent{
					SrcPeerId:       dataConn.GetPeerID(),
					RelayPeerNumber: relayPeerNumber,
				},
			},
		})
	})

	dataConn.On("disconnected", func(_ interface{}) {
		log.Infof("CLIENT PEER %s DATACHANNEL DISCONNECTED", dataConn.GetPeerID())
		// push out an event that this peer has disconnected
		conn.eventStream.Push(proto.RelayEventStream{
			Event: &proto.RelayEventStream_PeerDisconnected{
				PeerDisconnected: &proto.PeerDisconnectedEvent{
					SrcPeerId:       dataConn.GetPeerID(),
					RelayPeerNumber: relayPeerNumber,
				},
			},
		})
	})

	dataConn.On("error", func(message interface{}) {
		errMessage := message.(error).Error()
		log.Errorf("CLIENT PEER %s DATACHANNEL ERROR EVENT: %s", dataConn.GetPeerID(), errMessage)

		msg := generateMessageMetadataForBackend(clientPeerId, "Error", errMessage)
		conn.pushRelayEvent(msg)
	})

	// handle incoming messages from this client peer
	dataConn.On("data", func(msgBytes interface{}) {
		var msgString string = string(msgBytes.([]byte))
		conn.handleIncomingDatachannelMessage(msgString, clientPeerId)
	})

}

func (conn *WebrtcConnectionCtrl) getPeerConnections(targetPeerIds []string) []ConnectionInfo {
	outConns := make([]ConnectionInfo, 0)
	if targetPeerIds[0] == "*" {
		// If the action is meant for all peers, return all the peer data and/or media connections
		for _, RelayPeer := range conn.RelayPeers {
			for peerId, _ := range RelayPeer.openDataConnections {
				outConns = append(outConns, ConnectionInfo{
					RelayPeer:       RelayPeer,
					TargetPeerId:    peerId,
					DataConnection:  RelayPeer.GetDataConnection(peerId),
					MediaConnection: RelayPeer.GetMediaConnection(peerId),
				})
			}
		}
	} else {
		// Otherwise return just the data and/or media connections for the specified target peers:
		for _, peerId := range targetPeerIds {
			for _, RelayPeer := range conn.RelayPeers {
				outConns = append(outConns, ConnectionInfo{
					RelayPeer:       RelayPeer,
					TargetPeerId:    peerId,
					DataConnection:  RelayPeer.GetDataConnection(peerId),
					MediaConnection: RelayPeer.GetMediaConnection(peerId),
				})
			}
		}
	}
	return outConns
}
