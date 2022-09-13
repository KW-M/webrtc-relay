package webrtc_relay

import (
	"strconv"
	"time"

	peerjs "github.com/muka/peerjs-go"
	"github.com/pion/webrtc/v3"
	log "github.com/sirupsen/logrus"
)

// RelayPeer represents a peerjs instance used by this relay.
type RelayPeer struct {
	// the WebrtcConnectionCtrl that this RelayPeer is associated with
	connCtrl *WebrtcConnectionCtrl
	// peerConfig: The peerjs PeerInitOptions to use for this peer
	peerConfig peerjs.Options
	// To handle the case where multiple relays are running at the same time on the same peer server,
	// we make the PeerId of this relay the BasePeerId plus this number tacked on the end
	// that we increment if the current peerId is already taken (relay-1, relay-2, etc..)
	// (if config.UseMemorablePeerIds is true this number will be the input to the getUniqueName function)
	peerIdEndingNum uint32
	// peerId: The full peerId of this peer
	peerId string
	// Log: The logrus logger to use for debug logs within WebrtcRelay Code
	log *log.Entry
	// peer: The current peerjs Peer instance for this RelayPeer
	peer *peerjs.Peer
	// currentState: The current state of this peer connection to the peer server (one of 'disconnected', 'connecting', 'connected', 'reconnecting', 'destroyed')
	currentState chan string
	// openDataConnections: A map of open data connections to this peer (keyed by the peerId of the connected peer)
	openDataConnections map[string]*peerjs.DataConnection
	// openMediaConnections: A map of open media connections to this peer (keyed by the peerId of the connected peer)
	openMediaConnections map[string]*peerjs.MediaConnection
	// connectionTimeout: The cancelable timeout timer. If the peer server connection (peer open) doesn't happen before the timeout the peer is destroyed and a new peer is created.
	connectionTimeout *time.Timer
	// onConnection: The callback to call when a new data connection is opened to this peer
	onConnection func(*peerjs.DataConnection)
	// onCall: The callback to call when a new media connection is received
	onCall func(*peerjs.MediaConnection)
	// expBackoffErrorCount: The number of consecutive errors that have occurred when trying to connect to the peer server or when the connection fails
	expBackoffErrorCount uint
}

// NewRelayPeer creates a new RelayPeer instance.
func NewRelayPeer(connCtrl *WebrtcConnectionCtrl, peerConfig peerjs.Options, startingEndNumber uint32) *RelayPeer {
	var p = RelayPeer{
		peer:                 nil,
		connCtrl:             connCtrl,
		peerConfig:           peerConfig,
		peerIdEndingNum:      startingEndNumber,
		currentState:         make(chan string),
		openDataConnections:  make(map[string]*peerjs.DataConnection),
		openMediaConnections: make(map[string]*peerjs.MediaConnection),
		connectionTimeout:    nil,
		onConnection:         nil,
		onCall:               nil,
		expBackoffErrorCount: 0,
	}
	p.peerId = p.GetRelayPeerId()
	p.log = connCtrl.log.WithField("peerId", p.peerId)
	return &p
}

func (p *RelayPeer) Start(onConnection func(*peerjs.DataConnection), onCall func(*peerjs.MediaConnection)) error {
	p.onConnection = onConnection
	p.onCall = onCall
	return p.createPeer()
}

func (p *RelayPeer) GetPeerId() string {
	return p.peerId
}

func (p *RelayPeer) GetCurrentPeer() *peerjs.Peer {
	return p.peer
}

func (p *RelayPeer) GetOpenDataConnections() map[string]*peerjs.DataConnection {
	return p.openDataConnections
}

func (p *RelayPeer) GetOpenMediaConnections() map[string]*peerjs.MediaConnection {
	return p.openMediaConnections
}

func (p *RelayPeer) GetDataConnection(peerId string) *peerjs.DataConnection {
	if dc, ok := p.openDataConnections[peerId]; ok {
		return dc
	}
	return nil
}

func (p *RelayPeer) GetMediaConnection(peerId string) *peerjs.MediaConnection {
	if mc, ok := p.openMediaConnections[peerId]; ok {
		return mc
	}
	return nil
}

func (p *RelayPeer) CallPeer(peerId string, track webrtc.TrackLocal, opts *peerjs.ConnectionOptions) (*peerjs.MediaConnection, error) {
	if mc, ok := p.openMediaConnections[peerId]; ok {
		return mc, nil
	}
	mc, err := p.peer.Call(peerId, track, opts)
	if err != nil {
		return nil, err
	}
	p.addMediaConnection(mc)
	return mc, nil
}

func (p *RelayPeer) ConnectToPeer(peerId string, opts *peerjs.ConnectionOptions) (*peerjs.DataConnection, error) {
	if dc, ok := p.openDataConnections[peerId]; ok {
		return dc, nil
	}
	dc, err := p.peer.Connect(peerId, opts)
	if err != nil {
		return nil, err
	}
	p.addDataConnection(dc)
	return dc, nil
}

// ----------- Private Methods -------------

func (p *RelayPeer) GetRelayPeerId() string {
	config := p.connCtrl.Relay.config
	if config.UseMemorablePeerIds {
		return config.BasePeerId + getUniqueName(p.peerIdEndingNum, config.MemorablePeerIdOffset)
	} else {
		return config.BasePeerId + strconv.FormatInt(int64(p.peerIdEndingNum+config.MemorablePeerIdOffset), 10)
	}
}

func (p *RelayPeer) createPeer() error {
	var err error = nil
	p.peerConfig.Token = p.connCtrl.TokenStore.GetToken(p.peerId + "|" + p.peerConfig.Host)
	p.peer, err = peerjs.NewPeer(p.peerId, p.peerConfig)
	if err != nil {
		p.expBackoffErrorCount += 1
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
		dataConnection := dataConn.(*peerjs.DataConnection)
		p.addDataConnection(dataConnection)
		p.onConnection(dataConnection)
	})

	p.peer.On("call", func(mediaConn interface{}) {
		mediaConnection := mediaConn.(*peerjs.MediaConnection)
		p.addMediaConnection(mediaConnection)
		p.onCall(mediaConnection)
	})

	p.peer.On("error", func(err interface{}) {
		pErr := err.(peerjs.PeerError)
		p.log.Errorf("Peer error (type %s): %s", pErr.Type, pErr.Error())
		if pErr.Type == "unavailable-id" {
			p.connCtrl.TokenStore.DiscardToken(p.peerId + "|" + p.peerConfig.Host)
			p.peerIdEndingNum++
			p.peerId = p.GetRelayPeerId()
			p.log.Info("Peer id unavailable, trying new id: ", p.peerId)
			p.recreatePeer()
		} else if p.peer.GetDestroyed() {
			p.expBackoffErrorCount += 1
			p.recreatePeer()
		} else if p.peer.GetDisconnected() {
			p.expBackoffErrorCount += 1
			p.onDisconnected()
		}
	})

	p.peer.On("disconnected", func(_ interface{}) {
		p.onDisconnected()
	})

	p.onConnecting()

	return nil
}

func (p *RelayPeer) addMediaConnection(mediaConn *peerjs.MediaConnection) {
	p.openMediaConnections[mediaConn.GetPeerID()] = mediaConn
	mediaConn.On("close", func(_ interface{}) {
		p.log.Info("Media connection closed" + mediaConn.GetPeerID())
		delete(p.openMediaConnections, mediaConn.GetPeerID())
	})
}

func (p *RelayPeer) addDataConnection(dataConn *peerjs.DataConnection) {
	p.openDataConnections[dataConn.GetPeerID()] = dataConn
	dataConn.On("close", func(_ interface{}) {
		p.log.Info("Data connection closed" + dataConn.GetPeerID())
		delete(p.openDataConnections, dataConn.GetPeerID())
	})
}

func (p *RelayPeer) onConnecting() {
	p.currentState <- "connecting"
	p.connectionTimeout = time.AfterFunc(time.Duration(8+p.expBackoffErrorCount)*time.Second, func() {
		p.expBackoffErrorCount += 1
		p.recreatePeer()
	})
}

func (p *RelayPeer) onConnected() {
	p.currentState <- "connected"
	p.expBackoffErrorCount = 0
	if p.connectionTimeout != nil {
		p.connectionTimeout.Stop()
		p.connectionTimeout = nil
	}
}

func (p *RelayPeer) onDisconnected() {
	p.currentState <- "disconnected"
	err := p.peer.Reconnect()
	if err != nil {
		p.expBackoffErrorCount += 1
		log.Error("ERROR RECONNECTING TO DISCONNECTED PEER SERVER: ", err.Error())
		p.recreatePeer()
	} else {
		p.onReconnecting()
	}
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
