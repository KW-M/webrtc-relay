package webrtc_relay

import (
	context "context"
	"fmt"
	"log"
	"net"

	proto "github.com/kw-m/webrtc-relay/pkg/proto"
	"google.golang.org/grpc"
)

type RelayGRPCServer struct {
	proto.UnimplementedWebRTCRelayServer
}

func (s *RelayGRPCServer) GetEventStream(*proto.EventStreamRequest, proto.WebRTCRelay_GetEventStreamServer) error {
	return nil
}

func (s *RelayGRPCServer) ConnectToPeer(ctx context.Context, req *proto.ConnectionRequest) (*proto.ConnectionResponse, error) {
	return &proto.ConnectionResponse{
		Status: proto.Status_OK,
	}, nil
}

func (s *RelayGRPCServer) DisconnectFromPeer(ctx context.Context, req *proto.ConnectionRequest) (*proto.ConnectionResponse, error) {
	return &proto.ConnectionResponse{
		Status: proto.Status_OK,
	}, nil
}

func (s *RelayGRPCServer) CallPeer(ctx context.Context, req *proto.ConnectionRequest) (*proto.ConnectionResponse, error) {
	return &proto.ConnectionResponse{
		Status: proto.Status_OK,
	}, nil
}

func (s *RelayGRPCServer) HangupPeer(ctx context.Context, req *proto.ConnectionRequest) (*proto.ConnectionResponse, error) {
	return &proto.ConnectionResponse{
		Status: proto.Status_OK,
	}, nil
}

func (s *RelayGRPCServer) SendMsgStream(proto.WebRTCRelay_SendMsgStreamServer) error {
	return nil
}

func startRelayGRPCServer() {
	log.Println("Starting gRPC server")

	// start grpc given the port
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", 9023))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	gServer := grpc.NewServer()
	proto.RegisterWebRTCRelayServer(gServer, &RelayGRPCServer{})
	err = gServer.Serve(lis)
	if err != nil {
		log.Fatalf("Failed to serve gRPC server %v", err)
	}
}
