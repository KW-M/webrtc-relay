package webrtc_relay

import (
	"github.com/kw-m/webrtc-relay/src/media"
)

// WebrtcConnectionCtrl: This is the main controller in charge of maintaining an open peer and accepting/connecting to other peers.
// While the fields here are public, they are NOT meant to be modified by the user, do so at your own risk.
type RelayApi struct {
	// MediaCtrl: The media controller for this connection.
	MediaCtrl *media.MediaController
	// ConnCtrl:
	ConnCtrl *WebrtcConnectionCtrl
}

// NewRelayApi: Creates a new WebrtcConnectionCtrl instance.
func NewRelayApi() *RelayApi {
	return &RelayApi{
		Media: media.NewMediaController(),
	}
}
