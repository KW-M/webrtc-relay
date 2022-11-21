module github.com/kw-m/webrtc-relay

go 1.18

require (
	github.com/muka/peerjs-go v0.0.0-20221106184718-1f7e6f02ee86
	github.com/pion/webrtc/v3 v3.1.48
	github.com/sirupsen/logrus v1.9.0
	github.com/stretchr/testify v1.8.1
	golang.org/x/exp v0.0.0-20221031165847-c99f073a8326
	google.golang.org/grpc v1.50.1
	google.golang.org/protobuf v1.28.1
)

require github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect

require (
	github.com/chuckpreslar/emission v0.0.0-20170206194824-a7ddd980baf9 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/pion/datachannel v1.5.2 // indirect
	github.com/pion/dtls/v2 v2.1.5 // indirect
	github.com/pion/ice/v2 v2.2.11 // indirect
	github.com/pion/interceptor v0.1.12 // indirect
	github.com/pion/logging v0.2.2 // indirect
	github.com/pion/mdns v0.0.5 // indirect
	github.com/pion/mediadevices v0.3.11
	github.com/pion/randutil v0.1.0 // indirect
	github.com/pion/rtcp v1.2.10 // indirect
	github.com/pion/rtp v1.7.13 // indirect
	github.com/pion/sctp v1.8.3 // indirect
	github.com/pion/sdp/v3 v3.0.6 // indirect
	github.com/pion/srtp/v2 v2.0.10 // indirect
	github.com/pion/stun v0.3.5 // indirect
	github.com/pion/transport v0.13.1 // indirect
	github.com/pion/turn/v2 v2.0.8 // indirect
	github.com/pion/udp v0.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rs/cors v1.8.2 // indirect
	golang.org/x/crypto v0.1.0 // indirect
	golang.org/x/image v0.1.0 // indirect
	golang.org/x/net v0.1.0 // indirect
	golang.org/x/sys v0.1.0 // indirect
	golang.org/x/text v0.4.0 // indirect
	google.golang.org/genproto v0.0.0-20221018160656-63c7b68cfc55 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/muka/peerjs-go => github.com/kw-m/peerjs-go v0.0.0-20221026222843-ef6f3f7b9637

replace github.com/pion/mediadevices => github.com/kw-m/mediadevices v0.0.0-20221121030856-5ce9dff357c8
