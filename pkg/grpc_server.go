package webrtc_relay

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/kw-m/webrtc-relay/pkg/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RelayGRPCServer struct {
	proto.UnimplementedWebRTCRelayServer
	relay *WebrtcRelay
}

func (r *RelayGRPCServer) GetEventStream(req *proto.EventStreamRequest, stream proto.WebRTCRelay_GetEventStreamServer) error {
	eventStream := r.relay.GetEventStream()
	for {
		select {
		case event := <-eventStream:
			err := stream.Send(event)
			if err != nil {
				log.Printf("Failed to send event to client: %v", err)
				return status.Errorf(codes.Internal, fmt.Sprintf("Failed to send event to client: %v", err))
			}
		case <-stream.Context().Done():
			return status.Errorf(codes.OK, "done")
		case <-r.relay.stopRelaySignal.GetSignal():
			return status.Errorf(codes.OK, "relayExit")
		}
	}
}

func (r *RelayGRPCServer) ConnectToPeer(ctx context.Context, req *proto.ConnectionRequest) (*proto.ConnectionResponse, error) {
	r.relay.ConnectToPeer(req.GetPeerId(), req.GetRelayPeerNumber(), 0)
	// if err != nil {
	// 	return &proto.ConnectionResponse{
	// 		Status: proto.Status_ERROR,
	// 	}, status.Errorf(codes.Internal, fmt.Sprintf("Failed to connect to peer: %v", err))
	// } else {
	return &proto.ConnectionResponse{
		Status: proto.Status_OK,
	}, status.Errorf(codes.OK, "OKKKKK")
	// }
}

func (r *RelayGRPCServer) DisconnectFromPeer(context.Context, *proto.ConnectionRequest) (*proto.ConnectionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DisconnectFromPeer not implemented")
}

func (r *RelayGRPCServer) CallPeer(context.Context, *proto.CallRequest) (*proto.CallResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CallPeer not implemented")
}

func (r *RelayGRPCServer) HangupPeer(context.Context, *proto.ConnectionRequest) (*proto.CallResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method HangupPeer not implemented")
}

func (r *RelayGRPCServer) SendMsgStream(proto.WebRTCRelay_SendMsgStreamServer) error {
	return status.Errorf(codes.Unimplemented, "method SendMsgStream not implemented")
}

func (r *RelayGRPCServer) SendMsg(context.Context, *proto.SendMsgRequest) (*proto.SendMsgResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SendMsg not implemented")
}

func startRelayGRPCServer(relay *WebrtcRelay) {
	log.Println("Starting gRPC server")
	relayGrpcHandler := new(RelayGRPCServer)
	relayGrpcHandler.relay = relay

	// start grpc given the port
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", 9023))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	gServer := grpc.NewServer()
	proto.RegisterWebRTCRelayServer(gServer, relayGrpcHandler)
	err = gServer.Serve(lis)
	if err != nil {
		log.Fatalf("Failed to serve gRPC server %v", err)
	}
}
