package webrtc_relay

import (
	"encoding/json"
	"strings"

	peerjs "github.com/muka/peerjs-go"
	webrtc "github.com/pion/webrtc/v3"
)

type MediaSource interface {
	GetTrack() *webrtc.TrackLocalStaticRTP
	StartMediaStream() float64
	Close()
}

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

	trackName := metaData.Params[0]
	mimeType := metaData.Params[1]
	sourcePath := metaData.Params[2]

	// exit if the relay peer hasn't been initialized yet
	if conn.CurrentRelayPeer == nil {
		log.Error("Cannot start media call: The relay peer has not yet been established (is nil).")
		return
	}

	// make sure the  metadata is a valid media track udp (rtp) url;
	sourceParts := strings.Split(sourcePath, "/")
	if sourceParts[0] != "udp:" {
		log.Error("Cannot start media call: The media source rtp url must start with 'udp://'")
	}

	// check if the passed track name refers to an already in use track source and if so, use the existing track;
	TrackSrc, trackSrcFound := conn.MediaSources[trackName]

	// if no track is ready, create a new track source from the passed source url
	if !trackSrcFound {

		// create a new media stream rtp reciver and webrtc track
		ipAndPort := sourceParts[2]
		mediaSrc, err := CreateRtpMediaSource(ipAndPort, 10000, h264FrameDuration, mimeType, trackName)
		if err != nil {
			log.Error("Error creating named pipe media source: ", err)
			return
		}
		conn.MediaSources[trackName] = mediaSrc
		TrackSrc = mediaSrc

		// start relaying bytes from the udp rtp to the webrtc media channel
		go mediaSrc.StartMediaStream()
	}

	runFunctionForTargetPeers(metaData.TargetPeerIds, conn, func(targetPeerId string, dataConn *peerjs.DataConnection) {

		// get the media channel between us and the target peer (if one exists)
		mediaChann, exists := conn.ActiveMediaConnectionsToThisRelay[targetPeerId]

		// if a media channel exists with this peer...
		if exists == true {

			// abort if this track is already added to the media channel  with this peer
			relayMediaStream := mediaChann.GetLocalStream()
			relayMediaTracks := relayMediaStream.GetTracks()
			for _, track := range relayMediaTracks {
				if track.ID() == trackName {
					// if the track is already being sent to this peer, don't add it again
					return
				}
			}

			// add the track to the media channel if it isn't in the list of tracks for this media stream channel already
			relayMediaStream.AddTrack(TrackSrc.GetTrack())

		} else {

			// if a media channel doesn't exist with this peer,
			// create one by calling that peer:
			_, err := conn.CurrentRelayPeer.Call(targetPeerId, TrackSrc.GetTrack(), peerjs.NewConnectionOptions())
			if err != nil {
				log.Error("Error media calling client peer: ", targetPeerId)
			}

		}
	})

}

func handleStopMediaStreamMsg(metadata RelayPipeToDatachannelMetadata, conn *WebrtcConnectionCtrl) {
	// TODO: implement
	// log := conn.log

	// trackName := metadata.Params[0]

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
