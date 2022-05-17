package webrtc_relay

import (
	"encoding/json"
	"strings"

	peerjs "github.com/muka/peerjs-go"
)

func generateMessageMetadataForBackend(srcPeerId string, peerEvent string, err string) string {
	var metadata = new(DatachannelToRelayPipeMetadata)
	metadata.SrcPeerId = srcPeerId
	if len(peerEvent) > 0 {
		metadata.PeerEvent = peerEvent
	}
	if len(err) > 0 {
		metadata.Err = err
	}
	metaDataJson, _ := json.Marshal(metadata)
	return string(metaDataJson)
}

func parseMessageMetadataFromBackend(message string, msgMetadataSeparator string) (RelayPipeToDatachannelMetadata, string, error) {
	// split the message into the metadata and the actual message
	metaDataAndMessage := strings.SplitN(message, msgMetadataSeparator, 2)

	// parse the metadata json string into the RelayPipeToDatachannelMetadata struct type
	var metaData RelayPipeToDatachannelMetadata // empty struct
	err := json.Unmarshal([]byte(metaDataAndMessage[0]), &metaData)
	if err != nil {
		return metaData, "", err
	}

	if len(metaDataAndMessage) == 2 {
		actualMessage := metaDataAndMessage[1]
		return metaData, actualMessage, nil
	} else {
		return metaData, "", nil
	}
}

func runFunctionForTargetPeers(targetPeerIds []string, conn *WebrtcConnectionCtrl, funcToRun func(targetPeerId string, dataConn *peerjs.DataConnection)) {
	// Send the message out:
	if (len(targetPeerIds) == 0) || (targetPeerIds[0] == "*") {
		// If the action is meant for all peers, send it to all peers
		for peerId, dataConn := range conn.ActiveDataConnectionsToThisRelay {
			funcToRun(peerId, dataConn)
		}
	} else {
		// Otherwise send it to just the specified target peers:
		for _, peerId := range targetPeerIds {
			if dataConn := conn.GetActiveDataConnection(peerId); dataConn != nil {
				funcToRun(peerId, dataConn)
			} else {
				funcToRun(peerId, nil)
			}
		}
	}
}

func handleConnectToPeersMsg(metaData RelayPipeToDatachannelMetadata, conn *WebrtcConnectionCtrl) {
	// log := conn.log

	// // initiate connections to the target peers specified in the metadata
	// for _, targetPeerId := range metaData.TargetPeerIds {
	// 	// if conn.CurrentRelayPeer == nil || !conn.CurrentRelayPeer.open {
	// 	// 	log.Error("Cannot connect to peer: The relay peer has not yet been established (is nil).")
	// 	// 	return
	// 	// }
	// 	// dataConn, err := conn.CurrentRelayPeer.Connect(targetPeerId, peerjs.NewConnectionOptions())
	// 	// if err != nil {
	// 	// 	log.Error("Error connecting to peer: ", targetPeerId, "err: ", err)
	// 	// 	continue
	// 	// }

	// 	// dataConn.On("open", func(interface{}) {
	// 	// 	conn.peerConnectionOpenHandler(dataConn)
	// 	// })
	// }
}

func handleStartMediaStreamMsg(metaData RelayPipeToDatachannelMetadata, conn *WebrtcConnectionCtrl) {
	log := conn.log

	streamName := metaData.Params[0]
	mimeType := metaData.Params[1]
	pipeName := metaData.Params[2]

	// exit if the relay peer hasn't been initialized yet
	if conn.CurrentRelayPeer == nil {
		log.Error("Cannot start media call: The relay peer has not yet been established (is nil).")
		return
	}

	// create the named pipe to accept the encoded media stream bytes
	mediaSrc, err := CreateNamedPipeMediaSource(conn.Relay.config.NamedPipeFolder+pipeName, 10000, h264FrameDuration, mimeType, streamName)
	if err != nil {
		log.Error("Error creating named pipe media source: ", err)
		return
	}

	// initVideoTrack()
	// pipeVideoToStream(conn.Relay.stopRelaySignal)
	// targetPeerId := metaData.TargetPeerIds[0]

	// log.Debug("Video calling to peer: ", targetPeerId)

	// call all of the target peer ids:
	runFunctionForTargetPeers(metaData.TargetPeerIds, conn, func(targetPeerId string, dataConn *peerjs.DataConnection) {
		_, err := conn.CurrentRelayPeer.Call(targetPeerId, mediaSrc.WebrtcTrack, peerjs.NewConnectionOptions())
		if err != nil {
			log.Error("Error media calling client peer: ", targetPeerId)
		}
	})

	// start relaying bytes from the named pipe to the webrtc media channel
	go mediaSrc.StartMediaStream()
}

func handleStopMediaStreamMsg(metadata RelayPipeToDatachannelMetadata, conn *WebrtcConnectionCtrl) {
	// TODO: implement
	// log := conn.log

	// streamName := metadata.Params[0]

	// conn
}

func handleMessageFromBackend(message string, conn *WebrtcConnectionCtrl) {
	config := conn.Relay.config
	log := conn.log

	metaData, actualMsg, err := parseMessageMetadataFromBackend(message, config.MessageMetadataSeparator)
	if err != nil {
		log.Error("Could not parse message metadata. Err: " + err.Error() + " Message: " + message)
	}

	if len(metaData.Action) > 0 {
		log.Debug("Handling message action: " + metaData.Action)
		if metaData.Action == "Media_Call_Peer" {
			handleStartMediaStreamMsg(metaData, conn)
		} else if metaData.Action == "Stop_Media_Call" {
			handleStopMediaStreamMsg(metaData, conn)
		} else if metaData.Action == "Connect" {
			handleConnectToPeersMsg(metaData, conn)
		}
	}

	if len(actualMsg) != 0 {

		// Convert the message to byte array for transit through datachannel:
		actualMsgBytes := []byte(actualMsg)

		// Get the target peers from the metadata for this message
		targetPeerIds := metaData.TargetPeerIds

		// send the message to all target peers
		runFunctionForTargetPeers(targetPeerIds, conn, func(targetPeerId string, dataConn *peerjs.DataConnection) {
			if dataConn != nil {
				dataConn.Send(actualMsgBytes, false)
			} else {
				log.Error("Error sending message to peer: " + targetPeerId + " Err: " + err.Error())
			}
		})
	}
}

//  // parse the metadata json string into a map
//  metaData := make(map[string]interface{})
//  err := json.Unmarshal([]byte(metaDataAndMessage[0]), &metaData)
//  if err != nil {
// 	 log.Error("Could not parse meta data json string: " + metaDataAndMessage[0])
// 	 return
//  }

//  // get the target peers from the metadata
//  targetPeers := metaData["TargetPeers"].([]interface{})
//  if len(targetPeers) == 0 {
// 	 log.Error("No target peers specified in message: " + message)
// 	 return
//  }

//  // get the actual message
//  actualMessage := metaDataAndMessage[1]

//  // send the message to all target peers
//  for _, targetPeer := range targetPeers {
// 	 targetPeer := targetPeer.(string)
// 	 if targetPeer == "" {
// 		 log.Error("Target peer is empty in message: " + message)
// 		 continue
// 	 }

// 	 // get the webrtc connection for the target peer
// 	 targetConnection := conn.GetConnection(targetPeer)
// 	 if targetConnection == nil {
// 		 log.Error("Could not find webrtc connection for target peer: " + targetPeer)
// 		 continue
// 	 }

// 	 // send the message to the target peer
// 	 targetConnection.Send(actualMessage)
//  }
// }
// }
