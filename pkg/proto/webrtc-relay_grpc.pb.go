// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.21.6
// source: webrtc-relay.proto

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// WebRTCRelayClient is the client API for WebRTCRelay service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type WebRTCRelayClient interface {
	// Get a stream of events from the webrtc-relay including recived messages, peer dis/connection events & errors, relay dis/connection events & errors.
	// Events that happen as a result of an RPC call WILL BE sent on this stream with the exchangeId set to the.
	// see the RelayEventStream type for more details - This is a server-side streaming RPC
	// The grpc client should send an empty EventStreamRequest message to start the stream
	GetEventStream(ctx context.Context, in *EventStreamRequest, opts ...grpc.CallOption) (WebRTCRelay_GetEventStreamClient, error)
	// Tell the webrtc-relay to connect to a peer
	// If errors/events happen later because of DisconnectFromPeer(), they will get sent on the RelayEventStream with the same exchangeId as included in this rpc ConnectionRequest (not returned to this RPC call)
	ConnectToPeer(ctx context.Context, in *ConnectionRequest, opts ...grpc.CallOption) (*ConnectionResponse, error)
	// Tell the webrtc-relay to disconnect from a peer it has an open connection with
	// If errors/events happen later because of DisconnectFromPeer(), they will get sent on the RelayEventStream with the same exchangeId as included in this rpc ConnectionRequest (not returned to this RPC call)
	DisconnectFromPeer(ctx context.Context, in *ConnectionRequest, opts ...grpc.CallOption) (*ConnectionResponse, error)
	// Tell the webrtc-relay to call a peer with a given stream name and media track
	// can be used to initiate a call or to add a media track to an existing call (untested)
	// If errors/events happen later because of CallPeer(), they will get sent on the RelayEventStream with the same exchangeId as included in this rpc CallRequest (not returned to this RPC call)
	CallPeer(ctx context.Context, in *CallRequest, opts ...grpc.CallOption) (*CallResponse, error)
	// Tell the webrtc-relay to stop the media call with a peer (does not cause relay to disconnect any open datachannels with the peer)
	// If errors/events happen later because of HangupPeer(), they will get sent on the RelayEventStream with the same exchangeId as included in this rpc ConnectionRequest (not returned to this RPC call)
	HangupPeer(ctx context.Context, in *ConnectionRequest, opts ...grpc.CallOption) (*CallResponse, error)
	// Opens a stream to the webrtc-relay which can be used to send lots of messages to one or more connected peers.
	// If errors/events happen because of a sending a message, they will get sent on the RelayEventStream with the same exchangeId as included in this rpc SendMsgRequest (not returned to this RPC call)
	SendMsgStream(ctx context.Context, opts ...grpc.CallOption) (WebRTCRelay_SendMsgStreamClient, error)
	// Adds a new Relay Peer to the webrtc-relay instance, and starts it (not yet implemented)
	AddRelayPeer(ctx context.Context, in *AddRelayRequest, opts ...grpc.CallOption) (*RelayErrorEvent, error)
	// Stops a Relay Peer runnin in the webrtc-relay instance, and removes it from the instance (not yet implemented)
	CloseRelayPeer(ctx context.Context, in *RelayPeerNumber, opts ...grpc.CallOption) (*RelayErrorEvent, error)
	// Gets the config and actual peerId of the webrtc-relay instance (not yet implemented)
	GetRelayPeerConfig(ctx context.Context, in *RelayPeerNumber, opts ...grpc.CallOption) (*RelayConfig, error)
}

type webRTCRelayClient struct {
	cc grpc.ClientConnInterface
}

func NewWebRTCRelayClient(cc grpc.ClientConnInterface) WebRTCRelayClient {
	return &webRTCRelayClient{cc}
}

func (c *webRTCRelayClient) GetEventStream(ctx context.Context, in *EventStreamRequest, opts ...grpc.CallOption) (WebRTCRelay_GetEventStreamClient, error) {
	stream, err := c.cc.NewStream(ctx, &WebRTCRelay_ServiceDesc.Streams[0], "/webrtcrelay.WebRTCRelay/GetEventStream", opts...)
	if err != nil {
		return nil, err
	}
	x := &webRTCRelayGetEventStreamClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type WebRTCRelay_GetEventStreamClient interface {
	Recv() (*RelayEventStream, error)
	grpc.ClientStream
}

type webRTCRelayGetEventStreamClient struct {
	grpc.ClientStream
}

func (x *webRTCRelayGetEventStreamClient) Recv() (*RelayEventStream, error) {
	m := new(RelayEventStream)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *webRTCRelayClient) ConnectToPeer(ctx context.Context, in *ConnectionRequest, opts ...grpc.CallOption) (*ConnectionResponse, error) {
	out := new(ConnectionResponse)
	err := c.cc.Invoke(ctx, "/webrtcrelay.WebRTCRelay/ConnectToPeer", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *webRTCRelayClient) DisconnectFromPeer(ctx context.Context, in *ConnectionRequest, opts ...grpc.CallOption) (*ConnectionResponse, error) {
	out := new(ConnectionResponse)
	err := c.cc.Invoke(ctx, "/webrtcrelay.WebRTCRelay/DisconnectFromPeer", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *webRTCRelayClient) CallPeer(ctx context.Context, in *CallRequest, opts ...grpc.CallOption) (*CallResponse, error) {
	out := new(CallResponse)
	err := c.cc.Invoke(ctx, "/webrtcrelay.WebRTCRelay/CallPeer", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *webRTCRelayClient) HangupPeer(ctx context.Context, in *ConnectionRequest, opts ...grpc.CallOption) (*CallResponse, error) {
	out := new(CallResponse)
	err := c.cc.Invoke(ctx, "/webrtcrelay.WebRTCRelay/HangupPeer", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *webRTCRelayClient) SendMsgStream(ctx context.Context, opts ...grpc.CallOption) (WebRTCRelay_SendMsgStreamClient, error) {
	stream, err := c.cc.NewStream(ctx, &WebRTCRelay_ServiceDesc.Streams[1], "/webrtcrelay.WebRTCRelay/SendMsgStream", opts...)
	if err != nil {
		return nil, err
	}
	x := &webRTCRelaySendMsgStreamClient{stream}
	return x, nil
}

type WebRTCRelay_SendMsgStreamClient interface {
	Send(*SendMsgRequest) error
	CloseAndRecv() (*ConnectionResponse, error)
	grpc.ClientStream
}

type webRTCRelaySendMsgStreamClient struct {
	grpc.ClientStream
}

func (x *webRTCRelaySendMsgStreamClient) Send(m *SendMsgRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *webRTCRelaySendMsgStreamClient) CloseAndRecv() (*ConnectionResponse, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(ConnectionResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *webRTCRelayClient) AddRelayPeer(ctx context.Context, in *AddRelayRequest, opts ...grpc.CallOption) (*RelayErrorEvent, error) {
	out := new(RelayErrorEvent)
	err := c.cc.Invoke(ctx, "/webrtcrelay.WebRTCRelay/AddRelayPeer", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *webRTCRelayClient) CloseRelayPeer(ctx context.Context, in *RelayPeerNumber, opts ...grpc.CallOption) (*RelayErrorEvent, error) {
	out := new(RelayErrorEvent)
	err := c.cc.Invoke(ctx, "/webrtcrelay.WebRTCRelay/CloseRelayPeer", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *webRTCRelayClient) GetRelayPeerConfig(ctx context.Context, in *RelayPeerNumber, opts ...grpc.CallOption) (*RelayConfig, error) {
	out := new(RelayConfig)
	err := c.cc.Invoke(ctx, "/webrtcrelay.WebRTCRelay/GetRelayPeerConfig", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// WebRTCRelayServer is the server API for WebRTCRelay service.
// All implementations must embed UnimplementedWebRTCRelayServer
// for forward compatibility
type WebRTCRelayServer interface {
	// Get a stream of events from the webrtc-relay including recived messages, peer dis/connection events & errors, relay dis/connection events & errors.
	// Events that happen as a result of an RPC call WILL BE sent on this stream with the exchangeId set to the.
	// see the RelayEventStream type for more details - This is a server-side streaming RPC
	// The grpc client should send an empty EventStreamRequest message to start the stream
	GetEventStream(*EventStreamRequest, WebRTCRelay_GetEventStreamServer) error
	// Tell the webrtc-relay to connect to a peer
	// If errors/events happen later because of DisconnectFromPeer(), they will get sent on the RelayEventStream with the same exchangeId as included in this rpc ConnectionRequest (not returned to this RPC call)
	ConnectToPeer(context.Context, *ConnectionRequest) (*ConnectionResponse, error)
	// Tell the webrtc-relay to disconnect from a peer it has an open connection with
	// If errors/events happen later because of DisconnectFromPeer(), they will get sent on the RelayEventStream with the same exchangeId as included in this rpc ConnectionRequest (not returned to this RPC call)
	DisconnectFromPeer(context.Context, *ConnectionRequest) (*ConnectionResponse, error)
	// Tell the webrtc-relay to call a peer with a given stream name and media track
	// can be used to initiate a call or to add a media track to an existing call (untested)
	// If errors/events happen later because of CallPeer(), they will get sent on the RelayEventStream with the same exchangeId as included in this rpc CallRequest (not returned to this RPC call)
	CallPeer(context.Context, *CallRequest) (*CallResponse, error)
	// Tell the webrtc-relay to stop the media call with a peer (does not cause relay to disconnect any open datachannels with the peer)
	// If errors/events happen later because of HangupPeer(), they will get sent on the RelayEventStream with the same exchangeId as included in this rpc ConnectionRequest (not returned to this RPC call)
	HangupPeer(context.Context, *ConnectionRequest) (*CallResponse, error)
	// Opens a stream to the webrtc-relay which can be used to send lots of messages to one or more connected peers.
	// If errors/events happen because of a sending a message, they will get sent on the RelayEventStream with the same exchangeId as included in this rpc SendMsgRequest (not returned to this RPC call)
	SendMsgStream(WebRTCRelay_SendMsgStreamServer) error
	// Adds a new Relay Peer to the webrtc-relay instance, and starts it (not yet implemented)
	AddRelayPeer(context.Context, *AddRelayRequest) (*RelayErrorEvent, error)
	// Stops a Relay Peer runnin in the webrtc-relay instance, and removes it from the instance (not yet implemented)
	CloseRelayPeer(context.Context, *RelayPeerNumber) (*RelayErrorEvent, error)
	// Gets the config and actual peerId of the webrtc-relay instance (not yet implemented)
	GetRelayPeerConfig(context.Context, *RelayPeerNumber) (*RelayConfig, error)
	mustEmbedUnimplementedWebRTCRelayServer()
}

// UnimplementedWebRTCRelayServer must be embedded to have forward compatible implementations.
type UnimplementedWebRTCRelayServer struct {
}

func (UnimplementedWebRTCRelayServer) GetEventStream(*EventStreamRequest, WebRTCRelay_GetEventStreamServer) error {
	return status.Errorf(codes.Unimplemented, "method GetEventStream not implemented")
}
func (UnimplementedWebRTCRelayServer) ConnectToPeer(context.Context, *ConnectionRequest) (*ConnectionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ConnectToPeer not implemented")
}
func (UnimplementedWebRTCRelayServer) DisconnectFromPeer(context.Context, *ConnectionRequest) (*ConnectionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DisconnectFromPeer not implemented")
}
func (UnimplementedWebRTCRelayServer) CallPeer(context.Context, *CallRequest) (*CallResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CallPeer not implemented")
}
func (UnimplementedWebRTCRelayServer) HangupPeer(context.Context, *ConnectionRequest) (*CallResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method HangupPeer not implemented")
}
func (UnimplementedWebRTCRelayServer) SendMsgStream(WebRTCRelay_SendMsgStreamServer) error {
	return status.Errorf(codes.Unimplemented, "method SendMsgStream not implemented")
}
func (UnimplementedWebRTCRelayServer) AddRelayPeer(context.Context, *AddRelayRequest) (*RelayErrorEvent, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddRelayPeer not implemented")
}
func (UnimplementedWebRTCRelayServer) CloseRelayPeer(context.Context, *RelayPeerNumber) (*RelayErrorEvent, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CloseRelayPeer not implemented")
}
func (UnimplementedWebRTCRelayServer) GetRelayPeerConfig(context.Context, *RelayPeerNumber) (*RelayConfig, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetRelayPeerConfig not implemented")
}
func (UnimplementedWebRTCRelayServer) mustEmbedUnimplementedWebRTCRelayServer() {}

// UnsafeWebRTCRelayServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to WebRTCRelayServer will
// result in compilation errors.
type UnsafeWebRTCRelayServer interface {
	mustEmbedUnimplementedWebRTCRelayServer()
}

func RegisterWebRTCRelayServer(s grpc.ServiceRegistrar, srv WebRTCRelayServer) {
	s.RegisterService(&WebRTCRelay_ServiceDesc, srv)
}

func _WebRTCRelay_GetEventStream_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(EventStreamRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(WebRTCRelayServer).GetEventStream(m, &webRTCRelayGetEventStreamServer{stream})
}

type WebRTCRelay_GetEventStreamServer interface {
	Send(*RelayEventStream) error
	grpc.ServerStream
}

type webRTCRelayGetEventStreamServer struct {
	grpc.ServerStream
}

func (x *webRTCRelayGetEventStreamServer) Send(m *RelayEventStream) error {
	return x.ServerStream.SendMsg(m)
}

func _WebRTCRelay_ConnectToPeer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ConnectionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WebRTCRelayServer).ConnectToPeer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/webrtcrelay.WebRTCRelay/ConnectToPeer",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WebRTCRelayServer).ConnectToPeer(ctx, req.(*ConnectionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WebRTCRelay_DisconnectFromPeer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ConnectionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WebRTCRelayServer).DisconnectFromPeer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/webrtcrelay.WebRTCRelay/DisconnectFromPeer",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WebRTCRelayServer).DisconnectFromPeer(ctx, req.(*ConnectionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WebRTCRelay_CallPeer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CallRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WebRTCRelayServer).CallPeer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/webrtcrelay.WebRTCRelay/CallPeer",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WebRTCRelayServer).CallPeer(ctx, req.(*CallRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WebRTCRelay_HangupPeer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ConnectionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WebRTCRelayServer).HangupPeer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/webrtcrelay.WebRTCRelay/HangupPeer",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WebRTCRelayServer).HangupPeer(ctx, req.(*ConnectionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WebRTCRelay_SendMsgStream_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(WebRTCRelayServer).SendMsgStream(&webRTCRelaySendMsgStreamServer{stream})
}

type WebRTCRelay_SendMsgStreamServer interface {
	SendAndClose(*ConnectionResponse) error
	Recv() (*SendMsgRequest, error)
	grpc.ServerStream
}

type webRTCRelaySendMsgStreamServer struct {
	grpc.ServerStream
}

func (x *webRTCRelaySendMsgStreamServer) SendAndClose(m *ConnectionResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *webRTCRelaySendMsgStreamServer) Recv() (*SendMsgRequest, error) {
	m := new(SendMsgRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _WebRTCRelay_AddRelayPeer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AddRelayRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WebRTCRelayServer).AddRelayPeer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/webrtcrelay.WebRTCRelay/AddRelayPeer",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WebRTCRelayServer).AddRelayPeer(ctx, req.(*AddRelayRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WebRTCRelay_CloseRelayPeer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RelayPeerNumber)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WebRTCRelayServer).CloseRelayPeer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/webrtcrelay.WebRTCRelay/CloseRelayPeer",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WebRTCRelayServer).CloseRelayPeer(ctx, req.(*RelayPeerNumber))
	}
	return interceptor(ctx, in, info, handler)
}

func _WebRTCRelay_GetRelayPeerConfig_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RelayPeerNumber)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WebRTCRelayServer).GetRelayPeerConfig(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/webrtcrelay.WebRTCRelay/GetRelayPeerConfig",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WebRTCRelayServer).GetRelayPeerConfig(ctx, req.(*RelayPeerNumber))
	}
	return interceptor(ctx, in, info, handler)
}

// WebRTCRelay_ServiceDesc is the grpc.ServiceDesc for WebRTCRelay service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var WebRTCRelay_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "webrtcrelay.WebRTCRelay",
	HandlerType: (*WebRTCRelayServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ConnectToPeer",
			Handler:    _WebRTCRelay_ConnectToPeer_Handler,
		},
		{
			MethodName: "DisconnectFromPeer",
			Handler:    _WebRTCRelay_DisconnectFromPeer_Handler,
		},
		{
			MethodName: "CallPeer",
			Handler:    _WebRTCRelay_CallPeer_Handler,
		},
		{
			MethodName: "HangupPeer",
			Handler:    _WebRTCRelay_HangupPeer_Handler,
		},
		{
			MethodName: "AddRelayPeer",
			Handler:    _WebRTCRelay_AddRelayPeer_Handler,
		},
		{
			MethodName: "CloseRelayPeer",
			Handler:    _WebRTCRelay_CloseRelayPeer_Handler,
		},
		{
			MethodName: "GetRelayPeerConfig",
			Handler:    _WebRTCRelay_GetRelayPeerConfig_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "GetEventStream",
			Handler:       _WebRTCRelay_GetEventStream_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "SendMsgStream",
			Handler:       _WebRTCRelay_SendMsgStream_Handler,
			ClientStreams: true,
		},
	},
	Metadata: "webrtc-relay.proto",
}
