module github.com/kw-m/webrtc-relay

go 1.14

require (
	github.com/muka/peerjs-go v0.0.0-20220127055826-032344f03997
	github.com/pion/webrtc/v3 v3.1.43
	github.com/sirupsen/logrus v1.9.0
	github.com/stretchr/testify v1.8.0
	google.golang.org/grpc v1.49.0
	google.golang.org/protobuf v1.28.0
)

replace github.com/muka/peerjs-go => github.com/kw-m/peerjs-go v0.0.0-20220903010816-990600cd924f
