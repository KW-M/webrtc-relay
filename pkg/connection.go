package webrtc_relay

import (
	"fmt"
	"strconv"
	"time"

	peerjs "github.com/muka/peerjs-go"
	peerjsServer "github.com/muka/peerjs-go/server"
	log "github.com/sirupsen/logrus"
)

type MediaTrackData struct {
	Track       *peerjs.MediaStreamTrack // the media track
	MediaSource *RtpMediaSource          // the source handler of the media track
	// ConsumerPeerIds []string                 // list of peer ids that are reciving this stream through a media channel
}

// WebrtcConnectionCtrl: This is the main controller in charge of maintaining an open peer and accepting/connecting to other peers.
// While the fields here are public, they are NOT meant to be modified by the user, do so at your own risk.
type WebrtcConnectionCtrl struct {
	// the Relay that this connection controller is associated with
	Relay *WebrtcRelay
	// To handle the case where multiple relays are running at the same time,
	// we make the PeerId of this ROBOT the BasePeerId plus this number tacked on
	// the end (eg: relay-0) that we increment if the current peerId is already taken (relay-1, relay-2, etc..)
	RelayPeerIdEndingNum int
	// the current peer object associated with this WebrtcConnectionCtrl:
	CurrentRelayPeer *peerjs.Peer
	// map of all open peer datachannels connected to this relay (key is the peerId of the client)
	ActiveDataConnectionsToThisRelay map[string]*peerjs.DataConnection
	// map of all open peer mediachannels connected to this relay (key is the peerId of the client)
	ActiveMediaConnectionsToThisRelay map[string]*peerjs.MediaConnection
	// map of media streams being broadcast to this relay (key is the stream name)
	MediaSources map[string]*RtpMediaSource
	// the log for this WebrtcConnectionCtrl
	log *log.Entry
}

func CreateWebrtcConnectionCtrl(relay *WebrtcRelay) *WebrtcConnectionCtrl {
	return &WebrtcConnectionCtrl{
		Relay:                            relay,
		MediaSources:                     make(map[string]*RtpMediaSource),
		ActiveDataConnectionsToThisRelay: make(map[string]*peerjs.DataConnection),
		CurrentRelayPeer:                 nil, // &peerjs.Peer{}  nil equivalent of the peerjs.Peer struct
		log:                              relay.Log.WithFields(log.Fields{"src": "WebrtcConnectionCtrl"}),
	}
}

func (conn *WebrtcConnectionCtrl) GetRelayPeerId() string {
	log.Infof("UseMemorablePeerIds: %t", conn.Relay.config.UseMemorablePeerIds)
	if conn.Relay.config.UseMemorablePeerIds {
		return conn.Relay.config.BasePeerId + getDailyName(uint64(conn.RelayPeerIdEndingNum))
	} else {
		return conn.Relay.config.BasePeerId + strconv.Itoa(conn.RelayPeerIdEndingNum)
	}
}

func (conn *WebrtcConnectionCtrl) GetActiveDataConnection(peerid string) *peerjs.DataConnection {
	if dataConn, found := conn.ActiveDataConnectionsToThisRelay[peerid+"::"+conn.Relay.config.PeerInitConfigs[0].Host]; found {
		return dataConn
	} else {
		return nil
	}
}

func (conn *WebrtcConnectionCtrl) SendMessageToBackend(message string) {
	select {
	case conn.Relay.RelayOutputMessageChannel <- message:
	default:
		conn.log.Error("SendMessageToBackend: Go channel is full! Msg:", message)
	}
}

/* forwards the passed message string (coming from the client/browser via the datachannel) to the backend (named pipe or go code) */
func (conn *WebrtcConnectionCtrl) handleIncomingDatachannelMessage(message string, relayPeer *peerjs.Peer, clientPeerId string, clientPeerDataConnection *peerjs.DataConnection, log *log.Entry) {
	if conn.Relay.config.AddMetadataToPipeMessages {
		var metadata string = generateMessageMetadataForBackend(clientPeerId, "", "")
		message = metadata + conn.Relay.config.MessageMetadataSeparator + message
	}
	// send a message down the named pipe containing the metadata plus the message from the client peer
	conn.SendMessageToBackend(message)
}

/* handle forwarding messages from the named pipe to the client/browser (via the datachannel) */
func (conn *WebrtcConnectionCtrl) handleMessagesFromBackend() {
	for {
		select {
		case msgFromBackend := <-conn.Relay.RelayInputMessageChannel:
			handleMessageFromBackend(msgFromBackend, conn)
		case <-conn.Relay.stopRelaySignal.GetSignal():
			conn.log.Debug("Exiting handleMessagesFromBackend loop.")
			return
		}
	}
}

/* getNextPeerServerOptions (non-blocking function)
 * Given the parameter number of "tries" to sucessfully connect to a peerjs server, this function will return a new set of peerServerOptions, that can be used to try to establish a peer server connection
 * This function will return the next set of peerServerOptions immediately.
 */
func (conn *WebrtcConnectionCtrl) getNextPeerServerOptions(tries int) (*peerjs.Options, *peerjsServer.Options, bool) {
	var peerOptionsConfig = conn.Relay.config.PeerInitConfigs[tries%len(conn.Relay.config.PeerInitConfigs)]

	log.Print("using peerjs Host: ", peerOptionsConfig.Host)

	var peerOptions = peerjs.NewOptions()
	peerOptions.Host = peerOptionsConfig.Host
	peerOptions.Port = peerOptionsConfig.Port
	peerOptions.Path = peerOptionsConfig.Path
	peerOptions.Secure = peerOptionsConfig.Secure
	peerOptions.Key = peerOptionsConfig.Key
	peerOptions.Debug = peerOptionsConfig.Debug
	// peerOptions.Token = peerOptionsConfig.Token
	peerOptions.Configuration = peerOptionsConfig.Configuration

	if peerOptionsConfig.StartLocalServer {
		var peerServerOptions = peerjsServer.NewOptions()
		peerServerOptions.LogLevel = peerOptionsConfig.ServerLogLevel
		peerServerOptions.Host = peerOptionsConfig.Host
		peerServerOptions.Port = peerOptionsConfig.Port
		peerServerOptions.Path = peerOptionsConfig.Path
		peerServerOptions.Key = peerOptionsConfig.Key
		peerServerOptions.ExpireTimeout = peerOptionsConfig.ExpireTimeout
		peerServerOptions.AliveTimeout = peerOptionsConfig.AliveTimeout
		peerServerOptions.AllowDiscovery = peerOptionsConfig.AllowDiscovery
		peerServerOptions.ConcurrentLimit = peerOptionsConfig.ConcurrentLimit
		peerServerOptions.CleanupOutMsgs = peerOptionsConfig.CleanupOutMsgs
		return &peerOptions, &peerServerOptions, true
	}

	return &peerOptions, &peerjsServer.Options{}, false // returns nil equivalent for peerServerOptions
}

/* startLocalPeerJsServer (blocking goroutine)
 * This function starts up a local PeerJs SERVER on this computer. This can be used when no external internet access is available.
 * This function is blocking and will not return until Relay.stopRelaySignal is triggered or a panic in the server occurs.
 */
func (conn *WebrtcConnectionCtrl) StartLocalPeerJsServer(serverOptions peerjsServer.Options) {
	for {
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

/* StartPeerServerConnectionLoop (blocking goroutine)
 * This function sets up the loops that will keep restarting setupRelayPeer(), whenever it exits (exept if Relay.stopRelaySignal is triggered)
 * this loop also handles specific errors like offline relay state, by switching to offline mode, and peer id taken, by incrementing the peerid postfix number before trying again.
 * This function is blocking and will not return until the peer connection fails (with the error) or Relay.stopRelaySignal is triggered.
 */
func (conn *WebrtcConnectionCtrl) StartPeerServerConnectionLoop() {
	peerServerConnectionTries := 0
	peerOptions, peerServerOptions, startLocalServer := conn.getNextPeerServerOptions(peerServerConnectionTries)

	go func() {
		for {
			select {
			case <-conn.Relay.stopRelaySignal.GetSignal():
				log.Println("Closing down webrtc connection loop.")
				return
			default:
				if startLocalServer {
					log.Debug("Starting local peerjs server... ServerConfig: ", peerServerOptions)
					go conn.StartLocalPeerJsServer(*peerServerOptions)
				}
				<-time.After(time.Second * 1)
				conn.ActiveDataConnectionsToThisRelay = make(map[string]*peerjs.DataConnection)
				log.Debug("Connecting to peerjs server... PeerConfig: ", peerOptions)
				err := conn.setupRelayPeer(peerOptions, conn.Relay.stopRelaySignal)
				if e, ok := err.(peerjs.PeerError); ok {
					errorType := e.Type
					if errorType == "unavailable-id" {
						log.Printf("Peer id is unavailable. Switching to next peer id end number...\n")
						conn.RelayPeerIdEndingNum++ // increment the peer id ending integer
					}
					if errorType == "network" {
						log.Printf("Peerjs server is unavailable, switching to next peer server\n")
						peerServerConnectionTries++ // increment the peer id ending integer
						peerOptions, peerServerOptions, startLocalServer = conn.getNextPeerServerOptions(peerServerConnectionTries)
					}
				}
			}
		}
	}()

	// Relay all messages recived from the named message pipe to all connected peers (unless the message metadata dictates which peers to send the message to)
	go conn.handleMessagesFromBackend()

	conn.Relay.stopRelaySignal.Wait() // wait for the quitSignal channel to be triggered at which point this goroutine can exit
}

/* peerConnectionOpenHandler (non-blocking function)
 * This function sets up the event listeners for the relayPeer object that accept new webrtc peer connections to the relay and handle errors & sutch
 * This loop also handles specific errors like offline relay state, by switching to offline mode, and peer id taken, by incrementing the peerid postfix number before trying again.
 * This function should be called within the peer.On("open",) function of the relayPeer object.
 * This function DOES NOT block, BUT the passed relayPeer parameter MUST NOT GO OUT OF SCOPE, or the event listeners will be garbage collected and (maybe) closed.
 */
func (conn *WebrtcConnectionCtrl) peerConnectionOpenHandler(clientPeerDataConnection *peerjs.DataConnection, peerServerOpts peerjs.Options) {
	var clientPeerId string = clientPeerDataConnection.GetPeerID()

	log := conn.log.WithField("peer", conn.CurrentRelayPeer.ID)
	log.Info("Peer is connecting to relay... peer id: ", clientPeerDataConnection.GetPeerID())

	clientPeerDataConnection.On("open", func(interface{}) {
		log.Info("Peer connection established with Peer ID: ", clientPeerDataConnection.GetPeerID())
		// add this newly open peer connection to the map of active connections
		conn.ActiveDataConnectionsToThisRelay[clientPeerId+"::"+peerServerOpts.Host] = clientPeerDataConnection

		// send a metadata message down the named message pipe that a new peer has connected
		if conn.Relay.config.AddMetadataToPipeMessages {
			msg := generateMessageMetadataForBackend(clientPeerId, "Connected", "")
			conn.SendMessageToBackend(msg)
		}

		// handle incoming messages from this client peer
		clientPeerDataConnection.On("data", func(msgBytes interface{}) {
			var msgString string = string(msgBytes.([]byte))
			log.Debug("clientDataConnection 🚘 GOT MESSAGE: ", msgString)
			conn.handleIncomingDatachannelMessage(msgString, conn.CurrentRelayPeer, clientPeerId, clientPeerDataConnection, log)
		})

	})

	clientPeerDataConnection.On("close", func(message interface{}) {
		log.Info("CLIENT PEER DATACHANNEL CLOSE EVENT", message)
		delete(conn.ActiveDataConnectionsToThisRelay, clientPeerId+"::"+peerServerOpts.Host) // remove this connection from the map of active connections

		// send a metadata message down the named message pipe that this peer connection has been closed
		if conn.Relay.config.AddMetadataToPipeMessages {
			msg := generateMessageMetadataForBackend(clientPeerId, "Closed", "")
			conn.SendMessageToBackend(msg)
		}
	})

	clientPeerDataConnection.On("disconnected", func(message interface{}) {
		log.Info("CLIENT PEER DATACHANNEL DISCONNECTED EVENT", message)

		// send a metadata message down the named message pipe that this peer has disconnected
		if conn.Relay.config.AddMetadataToPipeMessages {
			msg := generateMessageMetadataForBackend(clientPeerId, "Disconnected", "")
			conn.SendMessageToBackend(msg)
		}
	})

	clientPeerDataConnection.On("error", func(message interface{}) {
		errMessage := message.(error).Error()
		log.Error("CLIENT PEER DATACHANNEL ERROR EVENT: ", errMessage)
		if conn.Relay.config.AddMetadataToPipeMessages {
			msg := generateMessageMetadataForBackend(clientPeerId, "Error", errMessage)
			conn.SendMessageToBackend(msg)
		}
	})
}

/* setupRelayPeer (blocking goroutine)
 * This function sets up the peerjs peer for the relay
 * Then it waits for the peerjs server to "Open" initilize the
 * relay peer which then passes controll to the peerConnectionOpenHandler function.
 * This function also handles the "error", "disconnected" and "closed" events for the peerjs server connection.
 * This function is blocking and will not return until the peer connection fails (with the error) or Relay.stopRelaySignal is triggered.
 */
func (conn *WebrtcConnectionCtrl) setupRelayPeer(peerOptions *peerjs.Options, stopRelaySignal *UnblockSignal) error {
	exitFuncSignal := NewUnblockSignal()

	var relayPeerId string = conn.GetRelayPeerId()

	// setup logrus logger
	relayConnLog := log.WithFields(log.Fields{"peer": relayPeerId, "peerServer": peerOptions.Host})

	// establish peer with peerjs server
	var relayPeer, err = peerjs.NewPeer(relayPeerId, *peerOptions)
	log.Info("Setting up connection to peerjs server: " + peerOptions.Host + ":" + strconv.Itoa(peerOptions.Port) + peerOptions.Path)
	fmt.Printf("relayPeer.GetSocket(): %v\n", relayPeer.GetSocket())
	defer func() { // func to run when setupWebrtcConnection function exits (either normally or because of a panic)
		if relayPeer != nil && !relayPeer.GetDestroyed() {
			relayPeer.Close() // close this peer (including peer server connection)
		}
	}()

	if err != nil {
		relayConnLog.Error("Error creating relay peer: ", err)
		return err /// return and let the setupConnections loop take over
	}

	relayPeer.On("open", func(peerId interface{}) {
		var peerID string = peerId.(string) // typecast to string
		if peerID != relayPeerId {
			exitFuncSignal.Trigger() // signal to this goroutine to exit and let the setupConnections loop take over and rerun this function
		} else {
			relayConnLog.Info("Relay Peer Established!")
			relayPeer.On("connection", func(data interface{}) {
				conn.CurrentRelayPeer = relayPeer
				clientPeerDataConnection := data.(*peerjs.DataConnection) // typecast to DataConnection
				conn.peerConnectionOpenHandler(clientPeerDataConnection, *peerOptions)
			})
		}
	})

	relayPeer.On("close", func(interface{}) {
		relayConnLog.Info("ROBOT PEER CLOSE EVENT")
		exitFuncSignal.Trigger() // signal to this goroutine to exit and let the setupConnections loop take over
	})

	relayPeer.On("disconnected", func(message interface{}) {
		relayConnLog.Info("ROBOT PEER DISCONNECTED EVENT", message)
		if !exitFuncSignal.HasTriggered {
			log.Debug("Reconnecting...")
			err = relayPeer.Reconnect()
			if err != nil {
				relayConnLog.Error("ERROR RECONNECTING TO DISCONNECTED PEER SERVER: ", err)
				exitFuncSignal.Trigger() // signal to this goroutine to exit and let the setupConnections loop take over
			}
		}
	})

	relayPeer.On("error", func(err interface{}) {
		errorMessage := err.(peerjs.PeerError).Error()
		errorType := err.(peerjs.PeerError).Type
		relayConnLog.Error("ROBOT PEER ERROR EVENT:", errorType, errorMessage)
		if contains(FATAL_PEER_ERROR_TYPES, errorType) {
			exitFuncSignal.TriggerWithError(err.(peerjs.PeerError)) // signal to this goroutine to exit and let the setupConnections loop take over
		}
	})

	// ---------------------------------------------------------------------------------------------------------------------
	// block and wait for the exitFuncSignal or Relay.stopRelaySignal to be triggerd before exiting this function
	select {
	case <-exitFuncSignal.GetSignal():
		return exitFuncSignal.GetError()
	case <-conn.Relay.stopRelaySignal.GetSignal():
		exitFuncSignal.Trigger()
		return nil
	}
}
