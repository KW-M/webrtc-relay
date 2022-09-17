package webrtc_relay

import (
	"time"

	"testing"

	context "context"

	proto "github.com/kw-m/webrtc-relay/src/proto"
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
