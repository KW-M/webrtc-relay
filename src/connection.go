package webrtc_relay

import (
	"time"

	media "github.com/kw-m/webrtc-relay/src/media"
	util "github.com/kw-m/webrtc-relay/src/util"
	peerjs "github.com/muka/peerjs-go"
	peerjsServer "github.com/muka/peerjs-go/server"
	log "github.com/sirupsen/logrus"
)

type MediaTrackData struct {
	Track       *peerjs.MediaStreamTrack // the media track
	MediaSource *media.RtpMediaSource    // the source handler of the media track
	// ConsumerPeerIds []string                 // list of peer ids that are reciving this stream through a media channel
}

// WebrtcConnectionCtrl: This is the main controller in charge of maintaining an open peer and accepting/connecting to other peers.
// While the fields here are public, they are NOT meant to be modified by the user, do so at your own risk.
type WebrtcConnectionCtrl struct {
	// the Relay that this connection controller is associated with
	Relay *WebrtcRelay
	// map of relayPeers owned by this WebrtcConnectionCtrl (key is peer server hostname that the relayPeer is connected to)
	RelayPeers map[string]*RelayPeer
	// map of media streams being broadcast to this relay from the backend (key is the stream name)
	MediaSources map[string]*media.RtpMediaSource
	// The (json) store used to keep peer server tokens between sessions
	TokenStore *TokenPersistanceStore
	// the log for this WebrtcConnectionCtrl
	log *log.Entry
}

func NewWebrtcConnectionCtrl(relay *WebrtcRelay) *WebrtcConnectionCtrl {
	return &WebrtcConnectionCtrl{
		Relay:        relay,
		RelayPeers:   make(map[string]*RelayPeer),
		MediaSources: make(map[string]*media.RtpMediaSource),
		TokenStore:   NewTokenPersistanceStore(relay.config.TokenPersistanceFile, relay.Log.Logger),
		log:          relay.Log.WithFields(log.Fields{"mod": "ConnCtrl"}),
	}
}

func (conn *WebrtcConnectionCtrl) Start(stopRelaySignal *util.UnblockSignal) {
	// Start all of the peers specified in the config
	for _, config := range conn.Relay.config.PeerInitConfigs {

		// start a local peerjs server if it is enabled for this PeerInitConfig
		if config.StartLocalServer {
			peerServerOptions := peerServerOptsFromConfig(config)
			go conn.startLocalPeerJsServer(peerServerOptions)
			<-time.After(time.Second * 1) // wait a second to let the server start up
		}

		// start the RelayPeer for this PeerInitConfig
		peerOptions := peerOptsFromConfig(config)
		go conn.setupRelayPeer(&peerOptions, stopRelaySignal)
	}

	/* (blocking loop): handle forwarding messages from the named pipe to the client/browser (via the datachannels)  */
	for {
		select {
		case msgFromBackend := <-conn.Relay.RelayInputMessageChannel:
			handleMessageFromBackend(msgFromBackend, conn)
		case <-stopRelaySignal.GetSignal():
			conn.log.Debug("Exiting handleMessagesFromBackend loop.")
			return
		}
	}
}

/* setupRelayPeer (blocking goroutine)
 * This function sets up the peerjs peer for the relay
 * Then it waits for the peerjs server to "Open" initilize the
 * relay peer which then passes controll to the peerConnectionOpenHandler function.
 * This function also handles the "error", "disconnected" and "closed" events for the peerjs server connection.
 * This function is blocking and will not return until the peer connection fails (with the error) or Relay.stopRelaySignal is triggered.
 */
func (conn *WebrtcConnectionCtrl) setupRelayPeer(peerOptions *peerjs.Options, stopRelaySignal *util.UnblockSignal) error {

	// create a new RelayPeer class and add it to the map of relayPeers
	relayPeer := NewRelayPeer(conn, *peerOptions, 0)
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
			case <-stopRelaySignal.GetSignal():
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

func (conn *WebrtcConnectionCtrl) sendMessageToBackend(message string) {
	select {
	case conn.Relay.RelayOutputMessageChannel <- message:
	default:
		conn.log.Warn("sendMessageToBackend: Go channel is full!")
	}
}

/* forwards the passed message string (coming from the client/browser via the datachannel) to the backend (named pipe or go code) */
func (conn *WebrtcConnectionCtrl) handleIncomingDatachannelMessage(message string, clientPeerId string) {
	if conn.Relay.config.AddMetadataToBackendMessages {
		var metadata string = generateMessageMetadataForBackend(clientPeerId, "", "")
		message = metadata + conn.Relay.config.MessageMetadataSeparator + message
	}
	// send a message down the named pipe containing the metadata plus the message from the client peer
	conn.sendMessageToBackend(message)
}

// TODO: implement this
func (conn *WebrtcConnectionCtrl) onCall(mediaConn *peerjs.MediaConnection) {
	conn.log.Warn("onCall: not implemented!")
}

// onConnection: called when another peer connects to any of the relay peers
func (conn *WebrtcConnectionCtrl) onConnection(dataConn *peerjs.DataConnection) {
	if dataConn.Open {
		conn.peerConnectionOpenHandler(dataConn)
	} else {
		dataConn.On("open", func(_ interface{}) {
			conn.peerConnectionOpenHandler(dataConn)
		})
	}
}

func peerOptsFromConfig(config *PeerInitOptions) peerjs.Options {
	var peerOptions = peerjs.NewOptions()
	peerOptions.Host = config.Host
	peerOptions.Port = config.Port
	peerOptions.Path = config.Path
	peerOptions.Secure = config.Secure
	peerOptions.Key = config.Key
	peerOptions.Debug = config.Debug
	peerOptions.Configuration = config.Configuration
	return peerOptions
}

func peerServerOptsFromConfig(config *PeerInitOptions) peerjsServer.Options {
	var peerServerOptions = peerjsServer.NewOptions()
	peerServerOptions.LogLevel = config.ServerLogLevel
	peerServerOptions.Host = config.Host
	peerServerOptions.Port = config.Port
	peerServerOptions.Path = config.Path
	peerServerOptions.Key = config.Key
	peerServerOptions.ExpireTimeout = config.ExpireTimeout
	peerServerOptions.AliveTimeout = config.AliveTimeout
	peerServerOptions.AllowDiscovery = config.AllowDiscovery
	peerServerOptions.ConcurrentLimit = config.ConcurrentLimit
	peerServerOptions.CleanupOutMsgs = config.CleanupOutMsgs
	return peerServerOptions
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

		// wait for the Relay.stopRelaySignal channel to be closed at which point this function will exit and the local peerjs server will stop beacuse of the defer server.stop() function
		conn.Relay.stopRelaySignal.Wait()
		return
	}
}

/* peerConnectionOpenHandler (non-blocking function)
 * This function sets up the event listeners for the relayPeer object that accept new webrtc peer connections to the relay and handle errors & sutch
 * This loop also handles specific errors like offline relay state, by switching to offline mode, and peer id taken, by incrementing the peerid postfix number before trying again.
 * This function should be called within the peer.On("open",) function of the relayPeer object.
 * This function DOES NOT block, BUT the passed relayPeer parameter MUST NOT GO OUT OF SCOPE, or the event listeners will be garbage collected and (maybe) closed.
 */
func (conn *WebrtcConnectionCtrl) peerConnectionOpenHandler(dataConn *peerjs.DataConnection) {
	var clientPeerId string = dataConn.GetPeerID()
	log := conn.log

	log.Info("Connection established with Peer: ", dataConn.GetPeerID())

	// send a metadata message down the named message pipe that a new peer has connected
	if conn.Relay.config.AddMetadataToBackendMessages {
		msg := generateMessageMetadataForBackend(clientPeerId, "Connected", "")
		conn.sendMessageToBackend(msg)
	}

	dataConn.On("close", func(_ interface{}) {
		log.Infof("CLIENT PEER %s DATACHANNEL CLOSED", dataConn.GetPeerID())
		// send a metadata message down the named message pipe that this peer connection has been closed
		if conn.Relay.config.AddMetadataToBackendMessages {
			msg := generateMessageMetadataForBackend(clientPeerId, "Closed", "")
			conn.sendMessageToBackend(msg)
		}
	})

	dataConn.On("disconnected", func(_ interface{}) {
		log.Infof("CLIENT PEER %s DATACHANNEL DISCONNECTED", dataConn.GetPeerID())
		// send a metadata message down the named message pipe that this peer has disconnected
		if conn.Relay.config.AddMetadataToBackendMessages {
			msg := generateMessageMetadataForBackend(clientPeerId, "Disconnected", "")
			conn.sendMessageToBackend(msg)
		}
	})

	dataConn.On("error", func(message interface{}) {
		errMessage := message.(error).Error()
		log.Errorf("CLIENT PEER %s DATACHANNEL ERROR EVENT: %s", dataConn.GetPeerID(), errMessage)
		if conn.Relay.config.AddMetadataToBackendMessages {
			msg := generateMessageMetadataForBackend(clientPeerId, "Error", errMessage)
			conn.sendMessageToBackend(msg)
		}
	})

	// handle incoming messages from this client peer
	dataConn.On("data", func(msgBytes interface{}) {
		var msgString string = string(msgBytes.([]byte))
		conn.handleIncomingDatachannelMessage(msgString, clientPeerId)
	})

}
