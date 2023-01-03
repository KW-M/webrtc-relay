package webrtc_relay

import "github.com/kw-m/webrtc-relay/pkg/proto"

func (conn *WebrtcConnectionCtrl) getRelayExchangeId(relayPeerNumber uint32) uint32 {
	relayPeer, ok := conn.RelayPeers[relayPeerNumber]
	if ok {
		return relayPeer.GetSavedExchangeId()
	} else {
		return 0
	}
}

func (conn *WebrtcConnectionCtrl) getDataConnectionExchangeId(relayPeerNumber uint32, srcPeerId string) uint32 {
	relayPeer := conn.RelayPeers[relayPeerNumber]
	openDc, ok := relayPeer.GetOpenDataConnections()[srcPeerId]
	if ok {
		return openDc.exchangeId
	} else {
		return relayPeer.GetSavedExchangeId()
	}
}

func (conn *WebrtcConnectionCtrl) getMediaConnectionExchangeId(relayPeerNumber uint32, srcPeerId string) uint32 {
	relayPeer := conn.RelayPeers[relayPeerNumber]
	openDc, ok := relayPeer.GetOpenDataConnections()[srcPeerId]
	if ok {
		return openDc.exchangeId
	} else {
		return relayPeer.GetSavedExchangeId()
	}
}

func (conn *WebrtcConnectionCtrl) sendMsgRecivedEvent(relayPeerNumber uint32, srcPeerId string, payload []byte) {
	exchangeId := conn.getDataConnectionExchangeId(relayPeerNumber, srcPeerId)
	conn.eventStream.Push(&proto.RelayEventStream{
		ExchangeId: &exchangeId,
		Event: &proto.RelayEventStream_MsgRecived{
			MsgRecived: &proto.MsgRecivedEvent{
				SrcPeerId:       srcPeerId,
				RelayPeerNumber: relayPeerNumber,
				Payload:         payload,
			},
		},
	})
}

func (conn *WebrtcConnectionCtrl) sendRelayConnectedEvent(relayPeerNumber uint32) {
	exchangeId := conn.getRelayExchangeId(relayPeerNumber)
	conn.eventStream.Push(&proto.RelayEventStream{
		ExchangeId: &exchangeId,
		Event: &proto.RelayEventStream_RelayConnected{
			RelayConnected: &proto.RelayConnectedEvent{
				RelayPeerNumber: relayPeerNumber,
			},
		},
	})
}

func (conn *WebrtcConnectionCtrl) sendRelayDisconnectedEvent(relayPeerNumber uint32) {
	exchangeId := conn.getRelayExchangeId(relayPeerNumber)
	conn.eventStream.Push(&proto.RelayEventStream{
		ExchangeId: &exchangeId,
		Event: &proto.RelayEventStream_RelayDisconnected{
			RelayDisconnected: &proto.RelayDisconnectedEvent{
				RelayPeerNumber: relayPeerNumber,
			},
		},
	})
}

func (conn *WebrtcConnectionCtrl) sendRelayErrorEvent(relayPeerNumber uint32, errType proto.RelayErrorTypes, msg string) {
	exchangeId := conn.getRelayExchangeId(relayPeerNumber)
	conn.eventStream.Push(&proto.RelayEventStream{
		ExchangeId: &exchangeId,
		Event: &proto.RelayEventStream_RelayError{
			RelayError: &proto.RelayErrorEvent{
				RelayPeerNumber: relayPeerNumber,
				Type:            errType,
				Msg:             msg,
			},
		},
	})
}

func (conn *WebrtcConnectionCtrl) sendPeerConnectedEvent(relayPeerNumber uint32, srcPeerId string) {
	exchangeId := conn.getDataConnectionExchangeId(relayPeerNumber, srcPeerId)
	conn.eventStream.Push(&proto.RelayEventStream{
		ExchangeId: &exchangeId,
		Event: &proto.RelayEventStream_PeerConnected{
			PeerConnected: &proto.PeerConnectedEvent{
				RelayPeerNumber: relayPeerNumber,
				SrcPeerId:       srcPeerId,
			},
		},
	})
}

func (conn *WebrtcConnectionCtrl) sendPeerDisconnectedEvent(relayPeerNumber uint32, srcPeerId string) {
	exchangeId := conn.getDataConnectionExchangeId(relayPeerNumber, srcPeerId)
	conn.eventStream.Push(&proto.RelayEventStream{
		ExchangeId: &exchangeId,
		Event: &proto.RelayEventStream_PeerDisconnected{
			PeerDisconnected: &proto.PeerDisconnectedEvent{
				RelayPeerNumber: relayPeerNumber,
				SrcPeerId:       srcPeerId,
			},
		},
	})
}

func (conn *WebrtcConnectionCtrl) sendPeerCalledEvent(relayPeerNumber uint32, srcPeerId string, streamName string, tracks []*proto.TrackInfo) {
	exchangeId := conn.getMediaConnectionExchangeId(relayPeerNumber, srcPeerId)
	conn.eventStream.Push(&proto.RelayEventStream{
		ExchangeId: &exchangeId,
		Event: &proto.RelayEventStream_PeerCalled{
			PeerCalled: &proto.PeerCalledEvent{
				RelayPeerNumber: relayPeerNumber,
				SrcPeerId:       srcPeerId,
				StreamName:      streamName,
				Tracks:          tracks,
			},
		},
	})
}

func (conn *WebrtcConnectionCtrl) sendPeerHungupEvent(relayPeerNumber uint32, srcPeerId string) {
	exchangeId := conn.getMediaConnectionExchangeId(relayPeerNumber, srcPeerId)
	conn.eventStream.Push(&proto.RelayEventStream{
		ExchangeId: &exchangeId,
		Event: &proto.RelayEventStream_PeerHungup{
			PeerHungup: &proto.PeerHungupEvent{
				RelayPeerNumber: relayPeerNumber,
				SrcPeerId:       srcPeerId,
			},
		},
	})
}

func (conn *WebrtcConnectionCtrl) sendPeerDataConnErrorEvent(relayPeerNumber uint32, srcPeerId string, errType proto.PeerConnErrorTypes, msg string) {
	exchangeId := conn.getDataConnectionExchangeId(relayPeerNumber, srcPeerId)
	conn.eventStream.Push(&proto.RelayEventStream{
		ExchangeId: &exchangeId,
		Event: &proto.RelayEventStream_PeerDataConnError{
			PeerDataConnError: &proto.PeerDataConnErrorEvent{
				RelayPeerNumber: relayPeerNumber,
				SrcPeerId:       srcPeerId,
				Type:            errType,
				Msg:             msg,
			},
		},
	})
}

func (conn *WebrtcConnectionCtrl) sendPeerMediaConnErrorEvent(relayPeerNumber uint32, srcPeerId string, errType proto.PeerConnErrorTypes, msg string) {
	exchangeId := conn.getMediaConnectionExchangeId(relayPeerNumber, srcPeerId)
	conn.eventStream.Push(&proto.RelayEventStream{
		ExchangeId: &exchangeId,
		Event: &proto.RelayEventStream_PeerMediaConnError{
			PeerMediaConnError: &proto.PeerMediaConnErrorEvent{
				RelayPeerNumber: relayPeerNumber,
				SrcPeerId:       srcPeerId,
				Type:            errType,
				Msg:             msg,
			},
		},
	})
}
