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

type DataConnectionInfo struct {
	RelayPeer      *RelayPeer
	DataConnection *peerjs.DataConnection
	TargetPeerId   string
}

type MediaConnectionInfo struct {
	MediaConnection *peerjs.MediaConnection
	TargetPeerId    string
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

	// set default struct values
	metaData := RelayPipeToDatachannelMetadata{
		Action:        "",
		Params:        []string{},
		TargetPeerIds: []string{"*"},
	}

	// parse the metadata json string into the RelayPipeToDatachannelMetadata struct type
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

func getMediaConnections(targetPeerIds []string, conn *WebrtcConnectionCtrl) []MediaConnectionInfo {
	outConns := make([]MediaConnectionInfo, 0)
	if targetPeerIds[0] == "*" {
		// If the action is meant for all peers, return all the peer data connections
		for _, RelayPeer := range conn.RelayPeers {
			for peerId, mediaConn := range RelayPeer.openMediaConnections {
				outConns = append(outConns, MediaConnectionInfo{
					MediaConnection: mediaConn,
					TargetPeerId:    peerId,
				})
			}
		}
	} else {
		// Otherwise send it to just the specified target peers:
		for _, peerId := range targetPeerIds {
			for _, RelayPeer := range conn.RelayPeers {
				if mediaConn := RelayPeer.GetMediaConnection(peerId); mediaConn != nil {
					outConns = append(outConns, MediaConnectionInfo{
						MediaConnection: mediaConn,
						TargetPeerId:    peerId,
					})
				}
			}
		}
	}
	return outConns
}

func getDataConnections(targetPeerIds []string, conn *WebrtcConnectionCtrl) []DataConnectionInfo {
	outConns := make([]DataConnectionInfo, 0)
	if targetPeerIds[0] == "*" {
		// If the action is meant for all peers, return all the peer data connections
		for _, RelayPeer := range conn.RelayPeers {
			for peerId, dataConn := range RelayPeer.openDataConnections {
				outConns = append(outConns, DataConnectionInfo{
					DataConnection: dataConn,
					TargetPeerId:   peerId,
				})
			}
		}
	} else {
		// Otherwise send it to just the specified target peers:
		for _, peerId := range targetPeerIds {
			for _, RelayPeer := range conn.RelayPeers {
				if dataConn := RelayPeer.GetDataConnection(peerId); dataConn != nil {
					outConns = append(outConns, DataConnectionInfo{
						DataConnection: dataConn,
						TargetPeerId:   peerId,
					})
				}
			}
		}
	}
	return outConns
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

	// make sure the  metadata is a valid media track udp (rtp) url;
	sourceParts := strings.Split(sourcePath, "/")
	if sourceParts[0] != "udp:" {
		log.Error("Cannot start media call: The media source rtp url must start with 'udp://'")
	}

	// check if the passed track name refers to an already in use track source;
	TrackSrc, ok := conn.MediaSources[trackName]
	if !ok { // No existing track under this name

		// Create a new media stream rtp reciver and webrtc track from the passed source url
		ipAndPort := sourceParts[2]
		mediaSrc, err := NewRtpMediaSource(ipAndPort, 10000, h264FrameDuration, mimeType, trackName)
		if err != nil {
			log.Error("Error creating named pipe media source: ", err.Error())
			return
		}

		// Add the new media track back to the connection's media sources map
		conn.MediaSources[trackName] = mediaSrc
		TrackSrc = mediaSrc

		// start relaying bytes from the rtp udp url to the webrtc media track for this track
		go mediaSrc.StartMediaStream()
	}

	dataConns := getDataConnections(metaData.TargetPeerIds, conn)
	for _, dataConn := range dataConns {

		// get the media channel between us and the target peer (if one exists based on the datachannel)
		mediaChann, ok := dataConn.RelayPeer.openMediaConnections[dataConn.TargetPeerId]

		// if a media channel exists with this peer...
		if ok == true {

			// abort if this track is already added to the media connection/channel with this peer
			relayMediaStream := mediaChann.GetLocalStream()
			relayMediaTracks := relayMediaStream.GetTracks()
			for _, track := range relayMediaTracks {
				if track.ID() == trackName {
					return
				}
			}

			// add the track to the peer media channel
			relayMediaStream.AddTrack(TrackSrc.GetTrack())

		} else {

			// if a media channel doesn't exist with this peer, create one by calling that peer:
			_, err := dataConn.RelayPeer.CallPeer(dataConn.TargetPeerId, TrackSrc.GetTrack(), peerjs.NewConnectionOptions())
			if err != nil {
				log.Error("Error media calling client peer: ", dataConn.TargetPeerId)
			}

		}
	}

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

	metadata, mainMsg, err := parseMessageMetadataFromBackend(message, config.MessageMetadataSeparator)
	if err != nil {
		log.Error("Could not parse message metadata. Err: " + err.Error() + " Message: " + message)
	}

	if len(metadata.Action) > 0 {
		log.Debug("Handling message action: " + metadata.Action)
		if metadata.Action == "Media_Call_Peer" {
			handleStartMediaStreamMsg(metadata, conn)
		} else if metadata.Action == "Stop_Media_Call" {
			handleStopMediaStreamMsg(metadata, conn)
		} else if metadata.Action == "Connect" {
			handleConnectToPeersMsg(metadata, conn)
		}
	}

	if len(mainMsg) != 0 {

		// Convert the message to byte array for transit through datachannel:
		mainMsgBytes := []byte(mainMsg)

		// send the message to all target peers
		dataConns := getDataConnections(metadata.TargetPeerIds, conn)
		for _, dataConn := range dataConns {
			dataConn.DataConnection.Send(mainMsgBytes, false)
		}
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
