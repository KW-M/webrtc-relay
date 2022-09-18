package webrtc_relay

import (
	"github.com/kw-m/webrtc-relay/pkg/media"
	peerjs "github.com/muka/peerjs-go"
)

func connectToPeer(peerId string, relayPeerNumber uint32, conn *WebrtcConnectionCtrl) {
	log := conn.log

	// initiate connections to the target peers specified in the metadata
	for _, relayPeer := range conn.RelayPeers {
		if relayPeerNumber == 0 || relayPeer.relayPeerNumber == relayPeerNumber {
			log.Infof("Connecting to peer %s", relayPeer.peerId)

			dataConn, err := relayPeer.ConnectToPeer(peerId, peerjs.NewConnectionOptions())
			if err != nil {
				log.Error("Error connecting to peer: ", peerId, "err: ", err)
				continue
			}

			dataConn.On("open", func(interface{}) {
				conn.peerConnectionOpenHandler(dataConn)
			})
		}
	}

	if relayPeerNumber != 0 {
		log.Warnf("Cannot connect using relayPeer number %d: The relay peer has not yet been established (is nil).", relayPeerNumber)
	}
}

func streamTrackToPeers(targetPeerIds []string, trackName string, conn *WebrtcConnectionCtrl, media *media.MediaController) {
	log := conn.log

	// get the media source for the passed track name
	trackSrc := media.GetTrack(trackName)

	peerConns := conn.getPeerConnections(targetPeerIds)
	for _, peerConn := range peerConns {

		if peerConn.MediaConnection != nil {
			// if an open media channel exists between us and this peer...
			// abort if this track is already added to the media connection/channel with this peer
			relayMediaStream := peerConn.MediaConnection.GetLocalStream()
			relayMediaTracks := relayMediaStream.GetTracks()
			for _, track := range relayMediaTracks {
				if track.ID() == trackName {
					return
				}
			}

			// add the track to the peer media channel
			relayMediaStream.AddTrack(trackSrc.GetTrack())

		} else {

			// if a media channel doesn't exist with this peer, create one by calling that peer:
			_, err := peerConn.RelayPeer.CallPeer(peerConn.TargetPeerId, trackSrc.GetTrack(), peerjs.NewConnectionOptions())
			if err != nil {
				log.Error("Error media calling client peer: ", peerConn.TargetPeerId)
			}

		}
	}
}

// func stopMediaStream(conn *WebrtcConnectionCtrl) {
// 	// TODO: implement
// 	// log := conn.log
// }

// func handleMessageFromBackend(message string, conn *WebrtcConnectionCtrl) {
// 	config := conn.Relay.config
// 	log := conn.log

// 	metadata, mainMsg, err := parseMessageMetadataFromBackend(message, config.MessageMetadataSeparator)
// 	if err != nil {
// 		log.Error("Could not parse message metadata. Err: " + err.Error() + " Message: " + message)
// 	}

// 	if len(metadata.Action) > 0 {
// 		log.Debug("Handling message action: " + metadata.Action)
// 		if metadata.Action == "Media_Call_Peer" {
// 			handleStartMediaStreamMsg(metadata, conn)
// 		} else if metadata.Action == "Stop_Media_Call" {
// 			handleStopMediaStreamMsg(metadata, conn)
// 		} else if metadata.Action == "Connect" {
// 			handleConnectToPeersMsg(metadata, conn)
// 		}
// 	}

// 	if len(mainMsg) != 0 {

// 		// Convert the message to byte array for transit through datachannel:
// 		mainMsgBytes := []byte(mainMsg)

// 		// send the message to all target peers
// 		dataConns := getDataConnections(metadata.TargetPeerIds, conn)
// 		for _, dataConn := range dataConns {
// 			dataConn.DataConnection.Send(mainMsgBytes, false)
// 		}
// 	}
// }

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

// // NewRelayApi: Creates a new WebrtcConnectionCtrl instance.
// func NewRelayApi() *RelayApi {
// 	return &RelayApi{
// 		relayEventStream: util.NewEventSub[proto.RelayEventStream](1),
// 		// MediaCtrl: media.NewMediaController(),
// 		// ConnCtrl:  NewWebrtcConnectionCtrl(),
// 		log: log.WithField("module", "relay_api"),
// 	}
// }

// func (api *RelayApi) GetEventStream(*proto.EventStreamRequest) (<-chan proto.RelayEventStream, error) {
// 	return api.relayEventStream.Subscribe(), nil
// }

// func (api *RelayApi) ConnectToPeer(req *proto.ConnectionRequest) (*proto.ConnectionResponse, error) {
// 	return &proto.ConnectionResponse{
// 		Status: proto.Status_OK,
// 	}, nil
// }

// func (api *RelayApi) DisconnectFromPeer(req *proto.ConnectionRequest) (*proto.ConnectionResponse, error) {
// 	return &proto.ConnectionResponse{
// 		Status: proto.Status_OK,
// 	}, nil
// }

// func (api *RelayApi) CallPeer(req *proto.ConnectionRequest) (*proto.ConnectionResponse, error) {
// 	return &proto.ConnectionResponse{
// 		Status: proto.Status_OK,
// 	}, nil
// }

// func (api *RelayApi) HangupPeer(req *proto.ConnectionRequest) (*proto.ConnectionResponse, error) {
// 	return &proto.ConnectionResponse{
// 		Status: proto.Status_OK,
// 	}, nil
// }

// func (api *RelayApi) SendMsgStream(proto.WebRTCRelay_SendMsgStreamServer) error {
// 	return nil
// }
