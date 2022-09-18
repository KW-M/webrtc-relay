package webrtc_relay

import (
	"fmt"
	"io"
	"time"

	"testing"

	context "context"

	proto "github.com/kw-m/webrtc-relay/pkg/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	log "github.com/sirupsen/logrus"
)

func TestGRPCRelay(t *testing.T) {
	// unixPipeFilePath := "./ipcPipeTest.pipe"
	go startRelayGRPCServer()

	<-time.After(5 * time.Second)

	// start the grpc client
	var conn *grpc.ClientConn
	conn, err := grpc.Dial(":9023", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Client: could not connect to grpc server: %s", err)
	}
	defer conn.Close()

	client := proto.NewWebRTCRelayClient(conn)
	response, err := client.ConnectToPeer(context.Background(), &proto.ConnectionRequest{
		PeerId: "test",
	})

	if err != nil {
		log.Fatalf("Client: could not call connect on grpc server: %s", err)
	}

	assert.Equal(t, proto.Status_OK, response.Status)
	print("Client: Response from server")
}

func printEventStream(client proto.WebRTCRelayClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
			fmt.Printf("message from peer %s (via relay #%d): %s, \n", event.MsgRecived.SrcPeerId, event.MsgRecived.RelayPeerNumber, string(event.MsgRecived.Payload))
		case *proto.RelayEventStream_PeerConnected:
			fmt.Printf("peer connected: %s (via relay #%d)\n", event.PeerConnected.SrcPeerId, event.PeerConnected.RelayPeerNumber)
		case *proto.RelayEventStream_PeerDisconnected:
			fmt.Printf("peer disconnected: %s (via relay #%d)\n", event.PeerDisconnected.SrcPeerId, event.PeerDisconnected.RelayPeerNumber)
		case *proto.RelayEventStream_PeerCalled:
			fmt.Printf("call from peer %s (via relay #%d)\n", event.PeerCalled.SrcPeerId, event.PeerCalled.RelayPeerNumber)
		case *proto.RelayEventStream_PeerHungup:
			fmt.Printf("hangup from peer %s (via relay #%d)\n", event.PeerHungup.SrcPeerId, event.PeerHungup.RelayPeerNumber)
		case *proto.RelayEventStream_PeerConnError:
			fmt.Printf("connection error with peer %s (via relay #%d): %s\n", event.PeerConnError.SrcPeerId, event.PeerConnError.RelayPeerNumber, event.PeerConnError.Error)
		case *proto.RelayEventStream_RelayError:
			fmt.Printf("relay error: %s\n", event.RelayError.Error)
		case *proto.RelayEventStream_RelayConnected:
			fmt.Printf("relay connected: %s\n", event.RelayConnected.RelayPeerNumber)
		case *proto.RelayEventStream_RelayDisconnected:
			fmt.Printf("relay disconnected: %s\n", event.RelayDisconnected.RelayPeerNumber)
		default:
			fmt.Println("No matching operations")
		}
	}
}
