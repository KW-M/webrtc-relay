package webrtc_relay

import (
	"context"
	"fmt"
	"io"
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

func (r *RelayGRPCServer) CallPeer(ctx context.Context, req *proto.CallRequest) (*proto.CallResponse, error) {
	r.relay.CallPeers(req.GetTargetPeerIds(), req.GetRelayPeerNumber(), req.GetTracks(), req.GetExchangeId())
	// return nil, status.Errorf(codes.Unimplemented, "method CallPeer not implemented")
	return &proto.CallResponse{
		Status: proto.Status_OK,
	}, status.Errorf(codes.OK, "OKKKKK")
}

func (r *RelayGRPCServer) HangupPeer(context.Context, *proto.ConnectionRequest) (*proto.CallResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method HangupPeer not implemented")
}

func (r *RelayGRPCServer) SendMsgStream(msgStream proto.WebRTCRelay_SendMsgStreamServer) error {
	for {
		msg, err := msgStream.Recv()
		if err == io.EOF {
			return status.Errorf(codes.OK, "done")
		} else if err != nil {
			log.Printf("Failed to receive message from client: %v", err)
			return status.Errorf(codes.Internal, fmt.Sprintf("Failed to receive message from client: %v", err))
		}
		r.relay.SendMsg(msg.GetTargetPeerIds(), msg.GetPayload(), msg.GetRelayPeerNumber(), msg.GetExchangeId())
	}
}

func (r *RelayGRPCServer) SendMsg(context.Context, *proto.SendMsgRequest) (*proto.SendMsgResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SendMsg not implemented")
}

func startRelayGRPCServer(relay *WebrtcRelay) {
	relayGrpcHandler := new(RelayGRPCServer)
	relayGrpcHandler.relay = relay
	serverTransport := relay.config.GRPCServerAddress[0:7] // "http://" or "unix://"
	serverAddress := relay.config.GRPCServerAddress[7:]

	log.Println("Starting gRPC server, transport:", serverTransport, " address:", serverAddress)
	// start grpc given the port
	var lis net.Listener
	var err error
	if serverTransport == "http://" {
		lis, err = net.Listen("tcp", serverAddress)
		if err != nil {
			log.Fatalf("failed to listen on tcp address: %s err: %v", serverAddress, err)
		}
	} else if serverTransport == "unix://" {
		lis, err = net.Listen("unix", serverAddress)
		if err != nil {
			log.Fatalf("failed to listen on unix socket: %s err: %v", serverAddress, err)
		}
	} else {
		log.Fatalf("invalid server transport in webrtc-relay config: %s", serverTransport)
		return
	}

	gServer := grpc.NewServer()
	proto.RegisterWebRTCRelayServer(gServer, relayGrpcHandler)
	err = gServer.Serve(lis)
	if err != nil {
		log.Fatalf("Failed to serve gRPC server %v", err)
	}
}
