module github.com/kw-m/webrtc-relay

go 1.14

require (
	github.com/muka/peerjs-go v0.0.0-20220127055826-032344f03997
	github.com/pion/webrtc/v3 v3.1.34
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.7.1
)

replace github.com/muka/peerjs-go => github.com/kw-m/peerjs-go v0.0.0-20220509175640-d26eb77cf736
