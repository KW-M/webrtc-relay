package webrtc_relay

import (
	"strconv"
	"time"

	peerjs "github.com/muka/peerjs-go"
	"github.com/pion/webrtc/v3"
	log "github.com/sirupsen/logrus"

	util "github.com/kw-m/webrtc-relay/pkg/util"
)

const (
	RELAY_PEER_CONNECTED    = "connected"
	RELAY_PEER_CONNECTING   = "connecting"
	RELAY_PEER_RECONNECTING = "reconnecting"
	RELAY_PEER_DISCONNECTED = "disconnected"
	RELAY_PEER_DESTROYED    = "destroyed"
)

type openDataConnection struct {
	exchangeId uint32
	conn       *peerjs.DataConnection
}

type openMediaConnection struct {
	exchangeId uint32
	conn       *peerjs.MediaConnection
}

// RelayPeer represents a peerjs instance used by this relay.
type RelayPeer struct {
	//
	relayPeerNumber uint32
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
	// openDataConnections: A map of open data connections to this peer (keyed by the peerId of the connected (remote) peer)
	openDataConnections map[string]openDataConnection
	// openMediaConnections: A map of open media connections to this peer (keyed by the peerId of the connected (remote) peer)
	openMediaConnections map[string]openMediaConnection
	// connectionTimeout: The cancelable timeout timer. If the peer server connection (peer open) doesn't happen before the timeout the peer is destroyed and a new peer is created.
	connectionTimeout *time.Timer
	// onConnection: The callback to call when a new data connection is opened to this peer
	onConnection func(*peerjs.DataConnection, uint32)
	// onCall: The callback to call when a new media connection is received
	onCall func(*peerjs.MediaConnection, uint32)
	// onError: The callback to call when an error occurs
	onError func(peerjs.PeerError, uint32)
	// expBackoffErrorCount: The number of consecutive failed attempts to connect to the peer server. Used to determine the exponential backoff timeout.
	expBackoffErrorCount uint
	// savedExchangeId: The exchangeId sent with the last action associated with this relay peer, used to help the webrtc-relay user correlate errors or events with the action that caused them
	savedExchangeId uint32
}

// NewRelayPeer creates a new RelayPeer instance.
func NewRelayPeer(connCtrl *WebrtcConnectionCtrl, peerConfig peerjs.Options, startingEndNumber uint32, relayPeerNumber uint32) *RelayPeer {
	// stopRelaySignal: a signal that can be used to stop the relay
	var p = RelayPeer{
		peer:                 nil,
		connCtrl:             connCtrl,
		peerConfig:           peerConfig,
		relayPeerNumber:      relayPeerNumber,
		peerIdEndingNum:      startingEndNumber,
		currentState:         make(chan string),
		openDataConnections:  make(map[string]openDataConnection),
		openMediaConnections: make(map[string]openMediaConnection),
		connectionTimeout:    nil,
		onConnection:         nil,
		onCall:               nil,
		expBackoffErrorCount: 0,
	}
	p.peerId = p.GetRelayPeerId()
	p.log = connCtrl.log.WithField("peerId", p.peerId)
	return &p
}

func (p *RelayPeer) Start(onConnection func(*peerjs.DataConnection, uint32), onCall func(*peerjs.MediaConnection, uint32), onError func(peerjs.PeerError, uint32)) error {
	p.onConnection = onConnection
	p.onCall = onCall
	p.onError = onError
	return p.createPeer()
}

func (p *RelayPeer) GetPeerId() string {
	return p.peerId
}

func (p *RelayPeer) GetCurrentPeer() *peerjs.Peer {
	return p.peer
}

func (p *RelayPeer) GetSavedExchangeId() uint32 {
	return p.savedExchangeId
}

func (p *RelayPeer) SetSavedExchangeId(exchangeId uint32) {
	p.savedExchangeId = exchangeId
}

func (p *RelayPeer) GetOpenDataConnections() map[string]openDataConnection {
	return p.openDataConnections
}

func (p *RelayPeer) GetOpenMediaConnections() map[string]openMediaConnection {
	return p.openMediaConnections
}

func (p *RelayPeer) GetDataConnection(peerId string) *peerjs.DataConnection {
	if dc, ok := p.openDataConnections[peerId]; ok {
		return dc.conn
	}
	return nil
}

func (p *RelayPeer) GetMediaConnection(peerId string) *peerjs.MediaConnection {
	if mc, ok := p.openMediaConnections[peerId]; ok {
		return mc.conn
	}
	return nil
}

func (p *RelayPeer) CallPeer(peerId string, track webrtc.TrackLocal, opts *peerjs.ConnectionOptions, exchangeId uint32) (*peerjs.MediaConnection, error) {
	if mc, ok := p.openMediaConnections[peerId]; ok && mc.conn.Open {
		// return mc.conn, nil
		mc.conn.Close()
		println("!!!!!!!!!!!!  closed existing media connection")
	}
	mc, err := p.peer.Call(peerId, track, opts)
	if err != nil {
		return nil, err
	}
	p.addMediaConnection(mc, exchangeId)
	return mc, nil
}

func (p *RelayPeer) ConnectToPeer(peerId string, opts *peerjs.ConnectionOptions, exchangeId uint32) (*peerjs.DataConnection, error) {
	if dc, ok := p.openDataConnections[peerId]; ok && dc.conn.Open {
		// return dc.conn, nil
		dc.conn.Close()
	}
	dc, err := p.peer.Connect(peerId, opts)
	if err != nil {
		return nil, err
	}
	p.addDataConnection(dc, exchangeId)
	return dc, nil
}

func (p *RelayPeer) DisconnectFromPeer(peerId string) (error, error) {
	dc, dcOk := p.openDataConnections[peerId]
	mc, mcOk := p.openMediaConnections[peerId]
	var dcErr error
	var mcErr error
	if dcOk && dc.conn.Open {
		dcErr = dc.conn.Close()
	}
	if mcOk && mc.conn.Open {
		mcErr = mc.conn.Close()
	}
	return dcErr, mcErr
}

// ----------- Private Methods -------------

func (p *RelayPeer) GetRelayPeerId() string {
	config := p.connCtrl.config
	if config.UseMemorablePeerIds {
		return config.BasePeerId + util.GetUniqueName(p.peerIdEndingNum, config.MemorablePeerIdOffset)
	} else {
		return config.BasePeerId + strconv.FormatInt(int64(p.peerIdEndingNum+config.MemorablePeerIdOffset), 10)
	}
}

func (rp *RelayPeer) createPeer() error {
	var err error
	rp.peerConfig.Token = rp.connCtrl.tokenStore.GetToken(rp.peerId + "|" + rp.peerConfig.Host)
	rp.peer, err = peerjs.NewPeer(rp.peerId, rp.peerConfig)
	if err != nil {
		rp.expBackoffErrorCount += 1
		return err
	}

	rp.peer.On("open", func(id interface{}) {
		rp.log.Info("Peer open")
		id = id.(string)
		if rp.peerId != id.(string) && rp.peerId != "" {
			rp.log.Infof("Relay #%d (%s) got new/different peerId from server: %s", rp.relayPeerNumber, rp.peerConfig.Host, id)
		}
		rp.peerId = id.(string)
		rp.onConnected()
	})

	rp.peer.On("connection", func(dataConn interface{}) {
		dataConnection := dataConn.(*peerjs.DataConnection)
		rp.addDataConnection(dataConnection, rp.savedExchangeId)
		rp.onConnection(dataConnection, rp.relayPeerNumber)
	})

	rp.peer.On("call", func(mediaConn interface{}) {
		mediaConnection := mediaConn.(*peerjs.MediaConnection)
		rp.addMediaConnection(mediaConnection, rp.savedExchangeId)
		rp.onCall(mediaConnection, rp.relayPeerNumber)
	})

	rp.peer.On("error", func(err interface{}) {
		pErr := err.(peerjs.PeerError)
		rp.log.Errorf("RelayPeer error (type %s): %s", pErr.Type, pErr.Error())
		rp.onError(pErr, rp.relayPeerNumber)
		if pErr.Type == "unavailable-id" {
			rp.peerIdEndingNum++
			newId := rp.GetRelayPeerId()
			rp.log.Info("Peer id unavailable, trying new id: ", rp.peerId)
			if err := rp.connCtrl.tokenStore.DiscardToken(rp.peerId + "|" + rp.peerConfig.Host); err != nil {
				rp.log.Errorf("Error discarding token: %s", err)
			}
			rp.peerId = newId
			rp.recreatePeer()
		} else if rp.peer.GetDestroyed() {
			rp.expBackoffErrorCount += 1
			rp.recreatePeer()
		} else if rp.peer.GetDisconnected() {
			rp.expBackoffErrorCount += 1
			rp.onDisconnected()
		}
	})

	rp.peer.On("disconnected", func(_ interface{}) {
		rp.onDisconnected()
	})

	rp.onConnecting()

	return nil
}

func (p *RelayPeer) addMediaConnection(mediaConn *peerjs.MediaConnection, exchangeId uint32) {
	p.openMediaConnections[mediaConn.GetPeerID()] = openMediaConnection{conn: mediaConn, exchangeId: exchangeId}
	// for _, sender := range mediaConn.PeerConnection.GetSenders() {
	// 	pkts, attribtes, err := sender.ReadRTCP()
	// 	attribtes
	// }
	mediaConn.On("close", func(_ interface{}) {
		p.log.Info("Media connection closed" + mediaConn.GetPeerID())
		delete(p.openMediaConnections, mediaConn.GetPeerID())
	})
}

func (p *RelayPeer) addDataConnection(dataConn *peerjs.DataConnection, exchangeId uint32) {
	p.openDataConnections[dataConn.GetPeerID()] = openDataConnection{conn: dataConn, exchangeId: exchangeId}
	dataConn.On("close", func(_ interface{}) {
		p.log.Info("Data connection closed" + dataConn.GetPeerID())
		delete(p.openDataConnections, dataConn.GetPeerID())
	})
}

func (p *RelayPeer) onConnecting() {
	p.currentState <- RELAY_PEER_CONNECTING
	p.connectionTimeout = time.AfterFunc(time.Duration(8+p.expBackoffErrorCount)*time.Second, func() {
		p.expBackoffErrorCount += 1
		p.recreatePeer()
	})
}

func (p *RelayPeer) onConnected() {
	p.currentState <- RELAY_PEER_CONNECTED
	p.expBackoffErrorCount = 0
	if p.connectionTimeout != nil {
		p.connectionTimeout.Stop()
		p.connectionTimeout = nil
	}
}

func (p *RelayPeer) onDisconnected() {
	p.currentState <- RELAY_PEER_DISCONNECTED
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
	p.currentState <- RELAY_PEER_RECONNECTING
}

func (p *RelayPeer) onDestroyed() {
	p.currentState <- RELAY_PEER_DESTROYED
}

func (p *RelayPeer) recreatePeer() {
	p.log.Debug("Recreating peer")
	p.onDestroyed()
	if err := p.createPeer(); err != nil {
		p.log.Error("Error (re)creating peer: ", err.Error())
	}
}

func (p *RelayPeer) Cleanup() {
	if p.peer != nil {
		p.peer.Destroy()
	}
}

func (p *RelayPeer) BlockUntilPeerStateChange() string {
	return <-p.currentState
}
