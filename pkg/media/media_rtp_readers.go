package media

import (
	"errors"
	"io"
)

func read_raw_rtp_stream(rtpSource *RtpMediaSource) error {

	// Read RTP packets forever and send them to the WebRTC Client
	inboundRTPPacket := make([]byte, 1600) // UDP MTU
	for {
		n, _, err := rtpSource.listener.ReadFrom(inboundRTPPacket)
		if err != nil {
			rtpSource.log.Errorf("error during read: %s", err.Error())
			return err
		}

		if _, err = rtpSource.webrtcTrack.Write(inboundRTPPacket[:n]); err != nil {
			if errors.Is(err, io.ErrClosedPipe) {
				// The peerConnection has been closed.
				rtpSource.log.Warn("PeerConnection closed")
				return nil
			} else {
				rtpSource.log.Error(err.Error())
				return err
			}
		}

	}
}

func read_vp8_rtp_stream(rtpSource *RtpMediaSource) error {
	return read_raw_rtp_stream(rtpSource)
}

func read_h264_rtp_stream(rtpSource *RtpMediaSource) error {
	return read_raw_rtp_stream(rtpSource)
}

func read_ogg_rtp_stream(rtpSource *RtpMediaSource) error {
	return read_raw_rtp_stream(rtpSource)
}
