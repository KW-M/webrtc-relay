package webrtc_relay

import (
	"fmt"

	"github.com/kw-m/webrtc-relay/pkg/media"
	"github.com/kw-m/webrtc-relay/pkg/proto"
	peerjs "github.com/muka/peerjs-go"
	"golang.org/x/exp/maps"
)

const ALL_RELAY_PEERS uint32 = 0

type ConnectionInfo struct {
	RelayPeer       *RelayPeer
	DataConnection  *peerjs.DataConnection
	MediaConnection *peerjs.MediaConnection
	TargetPeerId    string
}

func (conn *WebrtcConnectionCtrl) getRelayPeers(targetRelayPeer uint32) []*RelayPeer {
	if targetRelayPeer == ALL_RELAY_PEERS {
		// return all the relay peers:
		return maps.Values(conn.RelayPeers)
	} else {
		// If the action is meant for one relay return that one:
		return []*RelayPeer{conn.RelayPeers[targetRelayPeer]}
	}
}

func (conn *WebrtcConnectionCtrl) getPeerConnections(targetPeerIds []string, targetRelayPeer uint32) []ConnectionInfo {
	outConns := make([]ConnectionInfo, 0)
	if targetPeerIds[0] == "*" {
		// If the action is meant for all peers, return all the peer data and/or media connections
		for _, RelayPeer := range conn.RelayPeers {
			if targetRelayPeer == ALL_RELAY_PEERS || RelayPeer.relayPeerNumber == targetRelayPeer {
				for peerId := range RelayPeer.openDataConnections {
					outConns = append(outConns, ConnectionInfo{
						RelayPeer:       RelayPeer,
						TargetPeerId:    peerId,
						DataConnection:  RelayPeer.GetDataConnection(peerId),
						MediaConnection: RelayPeer.GetMediaConnection(peerId),
					})
				}
			}
		}
	} else {
		// Otherwise return just the data and/or media connections for the specified target peers:
		for _, peerId := range targetPeerIds {
			for _, RelayPeer := range conn.RelayPeers {
				if targetRelayPeer == ALL_RELAY_PEERS || RelayPeer.relayPeerNumber == targetRelayPeer {
					outConns = append(outConns, ConnectionInfo{
						RelayPeer:       RelayPeer,
						TargetPeerId:    peerId,
						DataConnection:  RelayPeer.GetDataConnection(peerId),
						MediaConnection: RelayPeer.GetMediaConnection(peerId),
					})
				}
			}
		}
	}
	return outConns
}

func (conn *WebrtcConnectionCtrl) connectToPeer(peerId string, relayPeerNumber uint32, exchangeId uint32) {
	log := conn.log

	// initiate connections to the target peers specified in the metadata
	for _, relayPeer := range conn.getRelayPeers(relayPeerNumber) {

		// connect to the peer
		log.Infof("Connecting to peer %s (via relay #%d)", peerId, relayPeer.relayPeerNumber)
		dataConn, err := relayPeer.ConnectToPeer(peerId, peerjs.NewConnectionOptions(), exchangeId)
		if err != nil {
			log.Error("Error connecting to peer: ", peerId, "err: ", err)
			conn.sendPeerDataConnErrorEvent(relayPeer.relayPeerNumber, peerId, proto.PeerConnErrorTypes_UNKNOWN_ERROR, err.Error())
			continue
		}

		// wait for the connection to open:
		dataConn.On("open", func(interface{}) {
			conn.peerConnectionOpenHandler(dataConn, relayPeerNumber)
		})
	}
}

func (conn *WebrtcConnectionCtrl) disconnectFromPeer(peerId string, relayPeerNumber uint32, exchangeId uint32) {
	log := conn.log

	// disconnect from the target peers specified in the metadata
	for _, relayPeer := range conn.getRelayPeers(relayPeerNumber) {
		dcErr, mcErr := relayPeer.DisconnectFromPeer(peerId)
		if dcErr != nil {
			log.Errorf("Error closing data connection with peer %s (via relay #%d): %v", peerId, relayPeer.relayPeerNumber, mcErr)
		}
		if mcErr != nil {
			log.Errorf("Error closing media connection with peer %s (via relay #%d): %v", peerId, relayPeer.relayPeerNumber, mcErr)
		}
	}
}

// sendMessageToPeers: Send a message (as bytes) to the specified peers
// targetPeerIds: The peer IDs to send the message to. If the first element is "*", send to all peers.
// relayPeerNumber: The relay peer number to send the message through. If 0, send through all relay peers.
// msgBytes: The message to send, as bytes.
func (conn *WebrtcConnectionCtrl) sendMessageToPeers(targetPeerIds []string, relayPeerNumber uint32, msgBytes []byte, exchangeId uint32) {
	log := conn.log

	sentCount := 0
	peerConns := conn.getPeerConnections(targetPeerIds, relayPeerNumber)
	for _, peerConn := range peerConns {
		if peerConn.DataConnection != nil {
			if relayPeerNumber == 0 || peerConn.RelayPeer.relayPeerNumber == relayPeerNumber {
				err := peerConn.DataConnection.Send(msgBytes, false)
				if err != nil {
					log.Error("Error sending message to peer: ", peerConn.TargetPeerId, " err: ", err)
					conn.disconnectFromPeer(peerConn.TargetPeerId, peerConn.RelayPeer.relayPeerNumber, exchangeId)
				} else {
					sentCount++
				}
			}
		} else {
			log.Warnf("Cannot send message to peer %s (via relay #%d): The data connection has not yet been established (is nil).", peerConn.TargetPeerId, relayPeerNumber)
			conn.sendPeerDataConnErrorEvent(peerConn.RelayPeer.relayPeerNumber, peerConn.TargetPeerId, proto.PeerConnErrorTypes_CONNECTION_NOT_OPEN, "The data connection to this peer has not yet been established.")
		}
	}
}

func (conn *WebrtcConnectionCtrl) streamTracksToPeers(targetPeerIds []string, relayPeerNumber uint32, trackNames []string, media *media.MediaController, exchangeId uint32) {
	log := conn.log
	peerConns := conn.getPeerConnections(targetPeerIds, relayPeerNumber)

	for _, trackName := range trackNames {

		// get the media track source for this track name
		trackSrc := media.GetTrack(trackName)
		if trackSrc == nil {
			log.Errorf("Cannot stream track %s to peers: The track source is nil.", trackNames[0])
			return
		}

	peerConnLoop:
		for _, peerConn := range peerConns {

			if peerConn.MediaConnection != nil {
				// if an open media channel exists between us and this peer...

				// abort if this track is already added to the media connection/channel with this peer
				relayMediaStream := peerConn.MediaConnection.GetLocalStream()
				relayMediaTracks := relayMediaStream.GetTracks()
				for _, track := range relayMediaTracks {
					if track.ID() == trackNames[0] {
						continue peerConnLoop
					}
				}

				peerConn.MediaConnection.PeerConnection.OnNegotiationNeeded(func() {
					print("Negotiation needed")
				})

				// // add the track to the peer media channel
				// // _, err := peerConn.MediaConnection.PeerConnection.AddTrack(trackSrc.GetTrack())
				// // if err != nil {
				// // 	log.Errorf("Error adding track %s to media connection with peer %s (via relay #%d): %v", trackNames[0], peerConn.TargetPeerId, relayPeerNumber, err)
				// // 	continue
				// // }
				// // peerConn.MediaConnection.PeerConnection.RemoveTrack(peerConn.MediaConnection.PeerConnection.GetSenders()[0])
				// for _, trackSender := range peerConn.MediaConnection.PeerConnection.GetSenders() {
				// 	// trackName := trackSender.Track().ID()
				// 	err := trackSender.ReplaceTrack(trackSrc.GetTrack())
				// 	if err != nil {
				// 		log.Errorf("Error replacing track %s to media connection with peer %s (via relay #%d): %v", trackSrc.GetTrack().ID(), peerConn.TargetPeerId, relayPeerNumber, err)
				// 		continue
				// 	} else {
				// 		log.Infof("Replaced track %s to media connection with peer %s (via relay #%d)", trackSrc.GetTrack().ID(), peerConn.TargetPeerId, relayPeerNumber)
				// 	}

				// }
				// // relayMediaStream.AddTrack(trackSrc.GetTrack())

				// relayMediaStream = peerConn.MediaConnection.GetLocalStream()
				// relayMediaTracks = relayMediaStream.GetTracks()
				// print("Tracks: ")
				// for _, track := range relayMediaTracks {
				// 	print(track.ID(), ", ")
				// }
				// println()

				trueMediaSenders := peerConn.MediaConnection.PeerConnection.GetSenders()
				print("Senders: ")
				for _, sender := range trueMediaSenders {
					fmt.Printf("%v, ", sender.GetParameters())
				}
				println()
			} else {

				// if a media channel doesn't exist with this peer, create one by calling that peer:
				mediaConn, err := peerConn.RelayPeer.CallPeer(peerConn.TargetPeerId, trackSrc.GetTrack(), peerjs.NewConnectionOptions(), exchangeId)
				if err != nil {
					log.Error("Error media calling remote peer: ", peerConn.TargetPeerId)
					errorType, ok := proto.PeerConnErrorTypes_value[err.Error()]
					if !ok {
						errorType = int32(proto.PeerConnErrorTypes_UNKNOWN_ERROR)
					}
					conn.sendPeerMediaConnErrorEvent(peerConn.RelayPeer.relayPeerNumber, peerConn.TargetPeerId, proto.PeerConnErrorTypes(errorType), "Error media calling remote peer")
					return
				}

				mediaConn.PeerConnection.OnNegotiationNeeded(func() {
					conn.log.Warn("Media: PeerConnection.OnNegotiationNeeded")
				})
			}
		}
	}
}

func (conn *WebrtcConnectionCtrl) stopMediaStream(media *media.MediaController, peerId string, exchangeId uint32) {
	// TODO: implement
	log := conn.log
	log.Warn("TODO: Implement stopMediaStream()")

}
