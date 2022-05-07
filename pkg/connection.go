package webrtc_relay_core

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	peerjs "github.com/muka/peerjs-go"
	peerjsServer "github.com/muka/peerjs-go/server"
	log "github.com/sirupsen/logrus"
)

// WebrtcConnectionCtrl: This is the main controller in charge of maintaining an open peer and accepting/connecting to other peers.
// While the fields here are public, they are NOT meant to be modified by the user, do so at your own risk.
type WebrtcConnectionCtrl struct {
	// the Relay that this connection controller is associated with
	Relay *WebrtcRelay
	// To handle the case where multiple robots are running at the same time,
	// we make the PeerId of this ROBOT the BasePeerId plus this number tacked on
	// the end (eg: iROBOT-0) that we increment if the current peerId is already taken.
	RobotPeerIdEndingNum int
	// the current peer object associated with this WebrtcConnectionCtrl:
	CurrentRobotPeer *peerjs.Peer
	// map off all open peer datachannels connected to this robot (key is the peerId of the client)
	ActiveDataConnectionsToThisRobot map[string]*peerjs.DataConnection
	// the log for this WebrtcConnectionCtrl
	log *log.Entry
}

func CreateWebrtcConnectionCtrl(relay *WebrtcRelay) *WebrtcConnectionCtrl {
	return &WebrtcConnectionCtrl{
		Relay:                            relay,
		ActiveDataConnectionsToThisRobot: make(map[string]*peerjs.DataConnection),
		CurrentRobotPeer:                 nil, // &peerjs.Peer{}  nil equivalent of the peerjs.Peer struct
		log:                              relay.Log.WithFields(log.Fields{"src": "WebrtcConnectionCtrl"}),
	}
}

func (conn *WebrtcConnectionCtrl) RelayMessageFromDatachannel(message string) {
	select {
	case conn.Relay.FromDatachannelMessages <- message:
	default:
		conn.log.Error("RelayMessageFromDatachannel: Go channel is full! Msg:", message)
	}
}

func (conn *WebrtcConnectionCtrl) GetPeerId() string {
	return conn.Relay.config.BasePeerId + strconv.Itoa(conn.RobotPeerIdEndingNum)
}

func (conn *WebrtcConnectionCtrl) generateToIncomingChanMetadataMessage(srcPeerId string, peerEvent string, err string) string {
	var metadata = new(DatachannelToRelayPipeMetadata)
	metadata.SrcPeerId = srcPeerId
	if len(peerEvent) > 0 {
		metadata.PeerEvent = peerEvent
	}
	if len(err) > 0 {
		metadata.Err = err
	}
	mtaDataJson, _ := json.Marshal(metadata)
	return string(mtaDataJson)
}

/* forwards the passed message string (should come from the client/browser via the datachannel) to the named pipe */
func (conn *WebrtcConnectionCtrl) handleIncomingDatachannelMessage(message string, robotPeer *peerjs.Peer, clientPeerId string, clientPeerDataConnection *peerjs.DataConnection, log *log.Entry) {
	if conn.Relay.config.AddMetadataToPipeMessages {
		var metadata string = conn.generateToIncomingChanMetadataMessage(clientPeerId, "", "")
		message = metadata + conn.Relay.config.MessageMetadataSeparator + message
	}
	// send a message down the named message pipe with the message from the client peer
	conn.RelayMessageFromDatachannel(message)
}

/* handle forwarding messages from the named pipe to the client/browser (via the datachannel) */
func (conn *WebrtcConnectionCtrl) handleOutgoingDatachannelMessages() {
	for {
		select {
		case msgFromNamedPipe := <-conn.Relay.SendToDatachannelMessages:
			conn.log.Printf("msgFromNamedPipe GOT MESSAGE: %s", msgFromNamedPipe)
			var TargetPeerIds = make(map[string]bool)

			if conn.Relay.config.AddMetadataToPipeMessages {
				metadataAndMessage := strings.Split(msgFromNamedPipe, conn.Relay.config.MessageMetadataSeparator)
				if len(metadataAndMessage) == 2 {
					msgFromNamedPipe = metadataAndMessage[1]
					var metadataJson = metadataAndMessage[0]
					var metadata = new(RelayPipeToDatachannelMetadata)
					err := json.Unmarshal([]byte(metadataJson), &metadata)
					if err != nil {
						fmt.Printf("Error unmarshalling message metadata: %s\n", err)
					} else {
						// copy all of the target peer ids into the TargetPeerIds map
						for i := 0; i < len(metadata.TargetPeerIds); i++ {
							TargetPeerIds[metadata.TargetPeerIds[i]] = true
						}

						if metadata.Action == "Media_Call_Peer" {
							conn.log.Printf("msgFromNamedPipe GOT MESSAGE: %s", msgFromNamedPipe)

							streamName := metadata.Params[0]
							mimeType := metadata.Params[1]
							pipeName := metadata.Params[2]

							mediaSrc, err := CreateNamedPipeMediaSource(conn.Relay.config.NamedPipeFolder+pipeName, 10000, mimeType, streamName)
							if err != nil {
								conn.log.Error("Error creating named pipe media source: ", err)
								break
							}
							if conn.CurrentRobotPeer == nil {
								conn.log.Error("Error video calling: CurrentRobotPeer is nil")
							}
							mediaSrc.StartMediaStream(conn.Relay.stopRelaySignal)
							for _, peerId := range metadata.TargetPeerIds {
								_, err = conn.CurrentRobotPeer.Call(peerId, mediaSrc.WebrtcTrack, peerjs.NewConnectionOptions())
								if err != nil {
									conn.log.Error("Error video calling client peer: ", peerId)
								}
							}

							break
						}

						// handle other actions
						// ...
					}
				}
			}

			// send the message to all of the peers in the TargetPeerIds map (or all peers if TargetPeerIds is empty)
			var hasTargetPeerIds = len(TargetPeerIds) > 0
			for peerIdAndHost, dataChannel := range conn.ActiveDataConnectionsToThisRobot {
				var peerId string = strings.Split(peerIdAndHost, "::")[0]
				var host string = strings.Split(peerIdAndHost, "::")[1]
				if dataChannel != nil && (!hasTargetPeerIds || TargetPeerIds[peerId]) {
					conn.log.WithFields(log.Fields{
						"peerId": peerId,
						"host":   host,
					}).Println("Sending message to peer:", msgFromNamedPipe)
					dataChannel.Send([]byte(msgFromNamedPipe), false)
				}
			}
		// case <-time.After(time.Second * 5):
		// 	for peerIdAndHost, dataChannel := range conn.ActiveDataConnectionsToThisRobot {
		// 		var peerId string = strings.Split(peerIdAndHost, "::")[0]
		// 		var host string = strings.Split(peerIdAndHost, "::")[1]
		// 		if dataChannel != nil {
		// 			conn.log.WithFields(log.Fields{
		// 				"peerId": peerId,
		// 				"host":   host,
		// 			}).Println("Sending d message to peer:", time.Now().Format(time.RFC850))
		// 			dataChannel.Send([]byte(time.Now().Format(time.RFC850)), false)
		// 		}
		// 	}
		case <-conn.Relay.stopRelaySignal.GetSignal():
			conn.log.Println("Exiting handleOutgoingDatachannelMessages loop.")
			return
		}
	}
}

/* getNextPeerServerOptions (non-blocking function)
 * Given the parameter number of "tries" to sucessfully connect to a peerjs server, this function will return a new set of peerServerOptions, that can be used to try to establish a peer server connection
 * This function will return the next set of peerServerOptions immediately.
 */
func (conn *WebrtcConnectionCtrl) getNextPeerServerOptions(tries int) (*peerjs.Options, *peerjsServer.Options) {
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
		return &peerOptions, &peerServerOptions
	}

	return &peerOptions, &peerjsServer.Options{} // returns nil for peerServerOptions
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
 * This function sets up the loops that will keep restarting setupRobotPeer(), whenever it exits (exept if Relay.stopRelaySignal is triggered)
 * this loop also handles specific errors like offline robot state, by switching to offline mode, and peer id taken, by incrementing the peerid postfix number before trying again.
 * This function is blocking and will not return until the peer connection fails (with the error) or Relay.stopRelaySignal is triggered.
 */
func (conn *WebrtcConnectionCtrl) StartPeerServerConnectionLoop() {
	peerServerConnectionTries := 0
	peerOptions, peerServerOptions := conn.getNextPeerServerOptions(peerServerConnectionTries)

	go func() {
		for {
			select {
			case <-conn.Relay.stopRelaySignal.GetSignal():
				log.Println("Closing down webrtc connection loop.")
				return
			default:
				if peerServerOptions != nil {
					go conn.StartLocalPeerJsServer(*peerServerOptions)
				}
				conn.ActiveDataConnectionsToThisRobot = make(map[string]*peerjs.DataConnection)
				err := conn.setupRobotPeer(peerOptions, conn.Relay.stopRelaySignal)
				if e, ok := err.(*peerjs.PeerError); ok {
					errorType := e.Type
					if errorType == "unavailable-id" {
						log.Printf("Peer id is unavailable. Switching to next peer id end number...\n")
						conn.RobotPeerIdEndingNum++ // increment the peer id ending integer
					}
					if errorType == "network" {
						log.Printf("Peer Js server is unavailable, switching to next peer server\n")
						peerServerConnectionTries++ // increment the peer id ending integer
						peerOptions, peerServerOptions = conn.getNextPeerServerOptions(peerServerConnectionTries)
					}
				}
			}
		}
	}()

	// Relay all messages recived from the named message pipe to all connected peers (unless the message metadata dictates which peers to send the message to)
	go conn.handleOutgoingDatachannelMessages()

	conn.Relay.stopRelaySignal.Wait() // wait for the quitSignal channel to be triggered at which point this goroutine can exit
}

/* peerConnectionOpenHandler (non-blocking function)
 * This function sets up the event listeners for the robotPeer object that accept new webrtc peer connections to the robot and handle errors & sutch
 * This loop also handles specific errors like offline robot state, by switching to offline mode, and peer id taken, by incrementing the peerid postfix number before trying again.
 * This function should be called within the peer.On("open",) function of the robotPeer object.
 * This function DOES NOT block, BUT the passed robotPeer parameter MUST NOT GO OUT OF SCOPE, or the event listeners will be garbage collected and (maybe) closed.
 */
func (conn *WebrtcConnectionCtrl) peerConnectionOpenHandler(robotPeer *peerjs.Peer, peerId string, peerServerOpts peerjs.Options, robotConnLog *log.Entry) {
	robotPeer.On("connection", func(data interface{}) {
		conn.CurrentRobotPeer = robotPeer
		clientPeerDataConnection := data.(*peerjs.DataConnection) // typecast to DataConnection
		var clientPeerId string = clientPeerDataConnection.GetPeerID()

		log := robotConnLog.WithField("peer", robotPeer.ID)
		log.Info("Peer is connecting to rov... peer id: ", clientPeerDataConnection.GetPeerID())

		clientPeerDataConnection.On("open", func(interface{}) {
			log.Info("Peer connection established with Peer ID: ", clientPeerDataConnection.GetPeerID())
			// add this newly open peer connection to the map of active connections
			conn.ActiveDataConnectionsToThisRobot[clientPeerId+"::"+peerServerOpts.Host] = clientPeerDataConnection

			// send a metadata message down the named message pipe that a new peer has connected
			if conn.Relay.config.AddMetadataToPipeMessages {
				msg := conn.generateToIncomingChanMetadataMessage(clientPeerId, "Connected", "") + conn.Relay.config.MessageMetadataSeparator + "{}"
				conn.RelayMessageFromDatachannel(msg)
			}

			// handle incoming messages from this client peer
			clientPeerDataConnection.On("data", func(msgBytes interface{}) {
				var msgString string = string(msgBytes.([]byte))
				log.Debug("clientDataConnection 👩🏻‍✈️ GOT MESSAGE: ", msgString)
				conn.handleIncomingDatachannelMessage(msgString, robotPeer, clientPeerId, clientPeerDataConnection, log)
			})

			log.Info("VIDEO CALLING client peer: ", clientPeerId)
			_, err := robotPeer.Call(clientPeerId, cameraLivestreamVideoTrack, peerjs.NewConnectionOptions())
			if err != nil {
				log.Error("Error video calling client peer: ", clientPeerId)
				clientPeerDataConnection.Close()
				return
			}

		})

		clientPeerDataConnection.On("close", func(message interface{}) {
			log.Info("CLIENT PEER DATACHANNEL CLOSE EVENT", message)
			delete(conn.ActiveDataConnectionsToThisRobot, clientPeerId+"::"+peerServerOpts.Host) // remove this connection from the map of active connections

			// send a metadata message down the named message pipe that this peer connection has been closed
			if conn.Relay.config.AddMetadataToPipeMessages {
				msg := conn.generateToIncomingChanMetadataMessage(clientPeerId, "Closed", "") + conn.Relay.config.MessageMetadataSeparator + "{}"
				conn.RelayMessageFromDatachannel(msg)
			}
		})

		clientPeerDataConnection.On("disconnected", func(message interface{}) {
			log.Info("CLIENT PEER DATACHANNEL DISCONNECTED EVENT", message)

			// send a metadata message down the named message pipe that this peer has disconnected
			if conn.Relay.config.AddMetadataToPipeMessages {
				msg := conn.generateToIncomingChanMetadataMessage(clientPeerId, "Disconnected", "") + conn.Relay.config.MessageMetadataSeparator + "{}"
				conn.RelayMessageFromDatachannel(msg)
			}
		})

		clientPeerDataConnection.On("error", func(message interface{}) {
			errMessage := message.(error).Error()
			log.Error("CLIENT PEER DATACHANNEL ERROR EVENT: ", errMessage)
			if conn.Relay.config.AddMetadataToPipeMessages {
				msg := conn.generateToIncomingChanMetadataMessage(clientPeerId, "Error", errMessage) + conn.Relay.config.MessageMetadataSeparator + "{}"
				conn.RelayMessageFromDatachannel(msg)
			}
		})

	})
}

/* setupRobotPeer (blocking goroutine)
 * This function sets up the peerjs peer for the robot
 * Then it waits for the peerjs server to "Open" initilize the
 * robot peer which then passes controll to the peerConnectionOpenHandler function.
 * This function also handles the "error", "disconnected" and "closed" events for the peerjs server connection.
 * This function is blocking and will not return until the peer connection fails (with the error) or Relay.stopRelaySignal is triggered.
 */
func (conn *WebrtcConnectionCtrl) setupRobotPeer(peerOptions *peerjs.Options, stopRelaySignal *UnblockSignal) error {
	exitFuncSignal := NewUnblockSignal()

	log.Info("Setting up connection to peerjs server: " + peerOptions.Host + ":" + strconv.Itoa(peerOptions.Port))

	var robotPeerId string = conn.GetPeerId()

	// setup logrus logger
	robotConnLog := log.WithFields(log.Fields{"peer": robotPeerId, "peerServer": peerOptions.Host})

	// establish peer with peerjs server
	var robotPeer, err = peerjs.NewPeer(robotPeerId, *peerOptions)
	defer func() { // func to run when setupWebrtcConnection function exits (either normally or because of a panic)
		if robotPeer != nil && !robotPeer.GetDestroyed() {
			robotPeer.Close() // close this peer (including peer server connection)
		}
	}()

	if err != nil {
		robotConnLog.Error("Error creating robot peer: ", err)
		return err /// return and let the setupConnections loop take over
	}

	robotPeer.On("open", func(peerId interface{}) {
		var peerID string = peerId.(string) // typecast to string
		if peerID != robotPeerId {
			exitFuncSignal.Trigger() // signal to this goroutine to exit and let the setupConnections loop take over and rerun this function
		} else {
			robotConnLog.Info("Robot Peer Established!")
			conn.peerConnectionOpenHandler(robotPeer, robotPeerId, *peerOptions, robotConnLog)
		}
	})

	robotPeer.On("close", func(interface{}) {
		robotConnLog.Info("ROBOT PEER CLOSE EVENT")
		exitFuncSignal.Trigger() // signal to this goroutine to exit and let the setupConnections loop take over
	})

	robotPeer.On("disconnected", func(message interface{}) {
		robotConnLog.Info("ROBOT PEER DISCONNECTED EVENT", message)
		if !exitFuncSignal.HasTriggered {
			log.Debug("Reconnecting...")
			err = robotPeer.Reconnect()
			if err != nil {
				robotConnLog.Error("ERROR RECONNECTING TO DISCONNECTED PEER SERVER: ", err)
				exitFuncSignal.Trigger() // signal to this goroutine to exit and let the setupConnections loop take over
			}
		}
	})

	robotPeer.On("error", func(err interface{}) {
		errorMessage := err.(*peerjs.PeerError).Error()
		errorType := err.(*peerjs.PeerError).Type
		robotConnLog.Error("ROBOT PEER ERROR EVENT:", errorType, errorMessage)
		if contains(FATAL_PEER_ERROR_TYPES, errorType) {
			exitFuncSignal.TriggerWithError(err.(*peerjs.PeerError)) // signal to this goroutine to exit and let the setupConnections loop take over
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
