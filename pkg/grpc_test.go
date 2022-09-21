package webrtc_relay

import (
	"fmt"
	"io"
	"time"

	"testing"

	context "context"

	"github.com/kw-m/webrtc-relay/pkg/config"
	proto "github.com/kw-m/webrtc-relay/pkg/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	log "github.com/sirupsen/logrus"
)

func TestGRPCRelay(t *testing.T) {
	// unixPipeFilePath := "./ipcPipeTest.pipe"

	// create and start the webrtc_relay:
	config := config.GetDefaultRelayConfig()
	config.StartGRPCServer = true
	relay := NewWebrtcRelay(config)
	relay.Start()
	defer relay.Stop()

	<-time.After(3 * time.Second)
	println("------- relay started -------")

	// start the grpc backend client
	var conn *grpc.ClientConn
	conn, err := grpc.Dial(":9023", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Backend: could not connect to grpc server: %s", err)
	} else {
		println("Backend: connected to grpc server")
	}
	defer conn.Close()
	backend := proto.NewWebRTCRelayClient(conn)
	go printEventStream(backend)

	<-time.After(2 * time.Second)
	println("------- grpc backend started -------")

	// create a test "frontend" peer:
	peerInitConfig := config.PeerInitConfigs[0]
	relayPeer, ok := relay.connCtrl.RelayPeers[peerInitConfig.RelayPeerNumber]
	if !ok {
		t.Error("Relay peer not found with number: ", peerInitConfig.RelayPeerNumber)
	}
	relayId := relayPeer.GetPeerId()
	frontendPeer, _ := testWithFrontendPeer(t, relayId, peerInitConfig, false, false)
	defer frontendPeer.Destroy()

	<-time.After(2 * time.Second)
	println("------- frontend peer started -------")

	// tell the relay to connect to the frontend peer using grpc calls:
	fmt.Printf("Backend GRPC: connecting to frontend peer... (%s) \n", frontendPeer.ID)
	response, err := backend.ConnectToPeer(context.Background(), &proto.ConnectionRequest{
		PeerId: frontendPeer.ID,
	})
	if err != nil {
		t.Errorf("Backend: could not connect to peer: %v", err)
	}
	assert.Equal(t, proto.Status_OK, response.Status)
	fmt.Printf("Backend: Response from grpc connect call: %+v", response)
	<-time.After(6 * time.Second)
}

func printEventStream(client proto.WebRTCRelayClient) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := &proto.EventStreamRequest{}
	stream, err := client.GetEventStream(ctx, req)
	if err != nil {
		log.Fatal("cannot GetEventStream: ", err)
	}
	for {
		evt, err := stream.Recv()
		if err == io.EOF {
			return // end of event stream
		} else if err != nil {
			log.Fatal("cannot receive event stream: ", err)
		}

		switch event := evt.Event.(type) {
		case *proto.RelayEventStream_MsgRecived:
			fmt.Printf("EVENT MSG: %s\n", event.MsgRecived.String())
		case *proto.RelayEventStream_PeerConnected:
			fmt.Printf("EVENT peer connected: %s (via relay #%d, exId %d)\n", event.PeerConnected.SrcPeerId, evt.ExchangeId, event.PeerConnected.RelayPeerNumber)
		case *proto.RelayEventStream_PeerDisconnected:
			fmt.Printf("EVENT peer disconnected: %s (via relay #%d, exId %d)\n", event.PeerDisconnected.SrcPeerId, evt.ExchangeId, event.PeerDisconnected.RelayPeerNumber)
		case *proto.RelayEventStream_PeerCalled:
			fmt.Printf("EVENT call from peer %s (via relay #%d, exId %d)\n", event.PeerCalled.SrcPeerId, evt.ExchangeId, event.PeerCalled.RelayPeerNumber)
		case *proto.RelayEventStream_PeerHungup:
			fmt.Printf("EVENT hangup from peer %s (via relay #%d, exId %d)\n", event.PeerHungup.SrcPeerId, evt.ExchangeId, event.PeerHungup.RelayPeerNumber)
		case *proto.RelayEventStream_PeerDataConnError:
			fmt.Printf("EVENT peer data connection error from peer: %s (via relay #%d, exId %d) type=%s %s\n", event.PeerDataConnError.SrcPeerId, evt.ExchangeId, event.PeerDataConnError.RelayPeerNumber, event.PeerDataConnError.Type.String(), event.PeerDataConnError.Msg)
		case *proto.RelayEventStream_PeerMediaConnError:
			fmt.Printf("EVENT peer media connection error from peer: %s (via relay #%d, exId %d) type=%s %s\n", event.PeerMediaConnError.SrcPeerId, evt.ExchangeId, event.PeerMediaConnError.RelayPeerNumber, event.PeerMediaConnError.Type.String(), event.PeerMediaConnError.Msg)
		case *proto.RelayEventStream_RelayError:
			fmt.Printf("EVENT relay error: [type=%s] %s (exId %d)\n", event.RelayError.Type.String(), event.RelayError.Msg, evt.ExchangeId)
		case *proto.RelayEventStream_RelayConnected:
			fmt.Printf("EVENT relay connected: %d (exId %d)\n", event.RelayConnected.RelayPeerNumber, evt.ExchangeId)
		case *proto.RelayEventStream_RelayDisconnected:
			fmt.Printf("EVENT relay disconnected: %d (exId %d)\n", event.RelayDisconnected.RelayPeerNumber, evt.ExchangeId)
		default:
			fmt.Println("No matching operations")
		}
	}
}
