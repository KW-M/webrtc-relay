package webrtc_relay

import (
	"time"

	peerjs "github.com/muka/peerjs-go"
	"github.com/pion/webrtc/v3"
	log "github.com/sirupsen/logrus"
)

// RelayPeer represents a peerjs instance used by this relay.
type RelayPeer struct {
	// the relay that this RelayPeer is associated with
	relay *WebrtcRelay
	// peerConfig: The peerjs PeerInitOptions to use for this peer
	peerConfig peerjs.Options
	// peerId: The peerId of this peer
	peerId string
	// Log: The logrus logger to use for debug logs within WebrtcRelay Code
	log *log.Entry
	// peer: The peerjs Peer instance
	peer *peerjs.Peer
	// currentState: The current state of this peer connection to the peer server (one of 'disconnected', 'connecting', 'connected', 'reconnecting', 'destroyed')
	currentState chan string
	// openDataConnections: A map of open data connections to this peer
	openDataConnections map[string]*peerjs.DataConnection
	// openMediaConnections: A map of open media connections to this peer
	openMediaConnections map[string]*peerjs.MediaConnection
	// connectionTimeout: The cancelable timeout timer. If the peer server connection (peer open) doesn't happen before the timeout the peer is destroyed and a new peer is created.
	connectionTimeout *time.Timer
	// onConnection: The callback to call when a new data connection is opened to this peer
	onConnection func(peerjs.DataConnection)
	// onCall: The callback to call when a new media connection is received
	onCall func(peerjs.MediaConnection)
}

// NewRelayPeer creates a new RelayPeer instance.
func NewRelayPeer(relay *WebrtcRelay, peerConfig peerjs.Options, peerId string) *RelayPeer {
	return &RelayPeer{
		relay:                relay,
		peerConfig:           peerConfig,
		peerId:               peerId,
		log:                  relay.Log.WithField("peerId", peerId),
		peer:                 nil,
		currentState:         make(chan string),
		openDataConnections:  make(map[string]*peerjs.DataConnection),
		openMediaConnections: make(map[string]*peerjs.MediaConnection),
		connectionTimeout:    nil,
		onConnection:         nil,
		onCall:               nil,
	}
}

func (p *RelayPeer) Start(onConnection func(peerjs.DataConnection), onCall func(peerjs.MediaConnection)) error {
	p.onConnection = onConnection
	p.onCall = onCall
	return p.createPeer()
}

func (p *RelayPeer) GetPeerId() string {
	return p.peerId
}

func (p *RelayPeer) GetPeer() *peerjs.Peer {
	return p.peer
}

func (p *RelayPeer) GetOpenDataConnections() map[string]*peerjs.DataConnection {
	return p.openDataConnections
}

func (p *RelayPeer) GetOpenMediaConnections() map[string]*peerjs.MediaConnection {
	return p.openMediaConnections
}

func (p *RelayPeer) CallPeer(peerId string, track webrtc.TrackLocal, opts *peerjs.ConnectionOptions) (*peerjs.MediaConnection, error) {
	mc, err := p.peer.Call(peerId, track, opts)
	if err != nil {
		return nil, err
	}
	p.addMediaConnection(*mc)
	return mc, nil
}

func (p *RelayPeer) ConnectToPeer(peerId string, opts *peerjs.ConnectionOptions) (*peerjs.DataConnection, error) {
	dc, err := p.peer.Connect(peerId, opts)
	if err != nil {
		return nil, err
	}
	p.addDataConnection(*dc)
	return dc, nil
}

// ----------- Private Methods -------------

func (p *RelayPeer) createPeer() error {
	var err error = nil
	p.peer, err = peerjs.NewPeer(p.peerId, p.peerConfig)
	if err != nil {
		return err
	}

	p.peer.On("open", func(id interface{}) {
		p.log.Info("Peer open")
		id = id.(string)
		if p.peerId != id.(string) && p.peerId != "" {
			p.log.Info("got new peerId from server")
		}
		p.peerId = id.(string)
		p.onConnected()
	})

	p.peer.On("connection", func(dataConn interface{}) {
		dataConnection := dataConn.(peerjs.DataConnection)
		p.addDataConnection(dataConnection)
		p.onConnection(dataConnection)
	})

	p.peer.On("call", func(mediaConn interface{}) {
		mediaConnection := mediaConn.(peerjs.MediaConnection)
		p.addMediaConnection(mediaConnection)
		p.onCall(mediaConnection)
	})

	p.peer.On("error", func(err interface{}) {
		pErr := err.(peerjs.PeerError)
		p.log.Error("Peer error", pErr)
		if p.peer.GetDestroyed() {
			p.recreatePeer()
		} else if p.peer.GetDisconnected() {
			p.onDisconnected()
		}
	})

	p.peer.On("disconnected", func(_ interface{}) {
		p.onDisconnected()
	})

	p.onConnecting()

	return nil
}

func (p *RelayPeer) addMediaConnection(mediaConn peerjs.MediaConnection) {
	p.openMediaConnections[mediaConn.GetID()] = &mediaConn
	mediaConn.On("close", func(_ interface{}) {
		p.log.Info("Media connection closed" + mediaConn.GetID())
		delete(p.openMediaConnections, mediaConn.GetID())
	})
}

func (p *RelayPeer) addDataConnection(dataConn peerjs.DataConnection) {
	p.openDataConnections[dataConn.GetID()] = &dataConn
	dataConn.On("close", func(_ interface{}) {
		p.log.Info("Data connection closed" + dataConn.GetID())
		delete(p.openDataConnections, dataConn.GetID())
	})
}

func (p *RelayPeer) onConnecting() {
	p.currentState <- "connecting"
	time.AfterFunc(10*time.Second, func() {
		p.recreatePeer()
	})
}

func (p *RelayPeer) onConnected() {
	p.currentState <- "connected"

}

func (p *RelayPeer) onDisconnected() {
	p.currentState <- "disconnected"

}

func (p *RelayPeer) onReconnecting() {
	p.currentState <- "reconnecting"

}

func (p *RelayPeer) onDestroyed() {
	p.currentState <- "destroyed"
}

func (p *RelayPeer) recreatePeer() {
	p.onDestroyed()
	p.createPeer()
}

func (p *RelayPeer) Cleanup() error {
	if p.peer != nil {
		p.peer.Destroy()
	}
	return nil
}

func (p *RelayPeer) BlockUntilPeerStateChange() string {
	return <-p.currentState
}
