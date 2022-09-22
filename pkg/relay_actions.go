package webrtc_relay

import (
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
				log.Infof("Sending message to peer %s (via relay #%d)", peerConn.TargetPeerId, relayPeerNumber)
				err := peerConn.DataConnection.Send(msgBytes, false)
				if err != nil {
					log.Error("Error sending message to peer: ", peerConn.TargetPeerId, "err: ", err)
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

	// get the media source for the passed track name
	trackSrc := media.GetTrack(trackNames[0])

	peerConns := conn.getPeerConnections(targetPeerIds, relayPeerNumber)
	for _, peerConn := range peerConns {

		if peerConn.MediaConnection != nil {
			// if an open media channel exists between us and this peer...
			// abort if this track is already added to the media connection/channel with this peer
			relayMediaStream := peerConn.MediaConnection.GetLocalStream()
			relayMediaTracks := relayMediaStream.GetTracks()
			for _, track := range relayMediaTracks {
				if track.ID() == trackNames[0] {
					return
				}
			}

			// add the track to the peer media channel
			relayMediaStream.AddTrack(trackSrc.GetTrack())

		} else {

			// if a media channel doesn't exist with this peer, create one by calling that peer:
			_, err := peerConn.RelayPeer.CallPeer(peerConn.TargetPeerId, trackSrc.GetTrack(), peerjs.NewConnectionOptions(), exchangeId)
			if err != nil {
				log.Error("Error media calling remote peer: ", peerConn.TargetPeerId)
				errorType, ok := proto.PeerConnErrorTypes_value[err.Error()]
				if !ok {
					errorType = int32(proto.PeerConnErrorTypes_UNKNOWN_ERROR)
				}
				conn.sendPeerMediaConnErrorEvent(peerConn.RelayPeer.relayPeerNumber, peerConn.TargetPeerId, proto.PeerConnErrorTypes(errorType), "Error media calling remote peer")
			}

		}
	}
}

func (conn *WebrtcConnectionCtrl) stopMediaStream(media *media.MediaController, exchangeId uint32) {
	// TODO: implement
	log := conn.log
	log.Warn("TODO: Implement stopMediaStream()")

}
