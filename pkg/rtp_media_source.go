package webrtc_relay

import (
	"net"
	"strconv"
	"strings"
	"time"

	// "os"

	webrtc "github.com/pion/webrtc/v3"
	log "github.com/sirupsen/logrus"
)

type RtpMediaSource struct {
	listener       *net.UDPConn
	udpAddress     *net.UDPAddr
	exitSignal     *UnblockSignal
	WebrtcTrack    *webrtc.TrackLocalStaticRTP
	readInterval   time.Duration
	readBufferSize int
	log            *log.Entry
}

func NewRtpMediaSource(url string, readBufferSize int, readInterval time.Duration, mediaMimeType string, trackName string) (*RtpMediaSource, error) {
	logger := log.WithField("rtp_media_src", url)
	addrParts := strings.Split(url, ":")
	ip := net.ParseIP(addrParts[0])
	port, err := strconv.Atoi(addrParts[1])
	if err != nil {
		logger.Error("Error parsing rtp url:", err.Error())
		return nil, err
	}
	var rtpSrc = RtpMediaSource{
		listener:       nil,
		udpAddress:     &net.UDPAddr{IP: ip, Port: port},
		exitSignal:     NewUnblockSignal(),
		readInterval:   readInterval,
		readBufferSize: readBufferSize,
		log:            logger,
	}

	rtpSrc.log.Print("Creating RTP Media Source ", rtpSrc.udpAddress.String())

	track, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: mediaMimeType}, trackName, "main-stream")
	if err != nil {
		rtpSrc.log.Error("Failed to create webrtc track: ", err)
		return nil, err
	}
	rtpSrc.WebrtcTrack = track

	return &rtpSrc, nil
}

func (rtpSrc *RtpMediaSource) Close() {
	if rtpSrc.listener != nil {
		rtpSrc.exitSignal.Trigger()
		if err := rtpSrc.listener.Close(); err != nil {
			panic(err)
		}
	}
}

func (rtpSrc *RtpMediaSource) GetTrack() *webrtc.TrackLocalStaticRTP {
	return rtpSrc.WebrtcTrack
}

//https://stackoverflow.com/questions/41739837/all-mime-types-supported-by-mediarecorder-in-firefox-and-chrome
func (rtpSrc *RtpMediaSource) StartMediaStream() error {
	defer rtpSrc.Close()
	for {

		// Open a UDP Listener for RTP Packets
		listener, err := net.ListenUDP("udp", rtpSrc.udpAddress)
		if err != nil {
			rtpSrc.log.Error("Error opening media source rtp:", err.Error())
			<-time.After(time.Second)
			continue
		}
		rtpSrc.listener = listener

		defer func() {
			if err = listener.Close(); err != nil {
				panic(err)
			}
		}()

		mimeType := rtpSrc.WebrtcTrack.Codec().MimeType
		if mimeType == "video/h264" {
			err = read_h264_rtp_stream(rtpSrc)
		} else if mimeType == "video/VP8" {
			err = read_vp8_rtp_stream(rtpSrc)
		} else if mimeType == "audio/ogg" {
			err = read_ogg_rtp_stream(rtpSrc)
		} else {
			rtpSrc.log.Debug("Unknow Media Source MimeType: " + mimeType + " sending raw stream as fallback")
			err = read_raw_rtp_stream(rtpSrc)
		}

		if err != nil {
			rtpSrc.log.Error("Error reading media source:", err.Error())
			continue
		}

		rtpSrc.exitSignal.Wait()
		return nil
	}
}
