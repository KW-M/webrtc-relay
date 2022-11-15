package media

import (
	"errors"
	"strings"
	"time"

	peerjs "github.com/muka/peerjs-go"
	"github.com/pion/webrtc/v3"
	log "github.com/sirupsen/logrus"
)

const (
	H264FrameDuration = time.Millisecond * 33
)

type MediaSource interface {
	GetTrack() *webrtc.TrackLocalStaticRTP // the webrtc track to hold the media .TrackLocal
	StartMediaStream()
	AddConsumer(peerId string)
	RemoveConsumer(peerId string)
	GetConsumerPeerIds() []string
	Close()
}

type MediaController struct {
	// map of media streams being sent to this relay from the backend or frontend clients (key is the track name)
	MediaSources   map[string]MediaSource
	DevicesWrapper *mediaDevicesWrapper
	MediaEngine    webrtc.MediaEngine
}

func NewMediaController() *MediaController {

	mdw := newMediaDevicesWrapper()
	mediaEngine := webrtc.MediaEngine{}
	mdw.CodecSelector.Populate(&mediaEngine)

	return &MediaController{
		MediaSources:   make(map[string]MediaSource),
		DevicesWrapper: mdw,
		MediaEngine:    mediaEngine,
	}
}

func (mediaCtrl *MediaController) GetTrack(trackName string) MediaSource {
	// check if the passed track name refers to an already in use track source;
	TrackSrc, ok := mediaCtrl.MediaSources[trackName]
	if ok {
		return TrackSrc
	}
	return nil
}

// AddRtpTrack: add a new rtp track to the media controller and start listening for incoming rtp packets
func (mediaCtrl *MediaController) AddRtpTrack(trackName string, kind string, rtpSrcUrl string, codecParams webrtc.RTPCodecParameters) (*RtpMediaSource, error) {

	// check if the passed track name refers to an already in use track source, in which case we will remove it and replace it.
	// _ = mediaCtrl.RemoveTrack(trackName)

	// Check if the passed track name refers to an already in use track source:
	if track := mediaCtrl.GetTrack(trackName); track != nil {
		return nil, errors.New("Cannot AddRawTrack: The media source track name is already in use")
	}

	// make sure the  metadata is a valid media track udp (rtp) url;
	sourceParts := strings.Split(rtpSrcUrl, "/")
	if sourceParts[0] != "rtp:" {
		return nil, errors.New("Cannot start media call: The media source rtp url must start with 'rtp://'")
	}

	hostAndPort := sourceParts[2]

	// Create a new media stream rtp reciver and webrtc track from the passed source url
	mediaSrc, err := NewRtpMediaSource(hostAndPort, 10000, H264FrameDuration, codecParams.MimeType, trackName)
	if err != nil {
		log.Error("Error creating rtp media source: ", err.Error())
		return nil, err
	}

	// Add the new media track back to the connection's media sources map
	mediaCtrl.MediaSources[trackName] = mediaSrc

	// start relaying bytes from the rtp udp url to the webrtc media track for this track
	go mediaSrc.StartMediaStream()

	// return the new media source
	return mediaSrc, nil
}

func (mediaCtrl *MediaController) GetCallConnectionOptions() *peerjs.ConnectionOptions {
	connOpts := peerjs.NewConnectionOptions()
	connOpts.MediaEngine = &mediaCtrl.MediaEngine
	return connOpts
}

//// AddRawTrack: add a new raw track to the media controller and start listening for incoming samples
// func (mediaCtrl *MediaController) AddRawTrack(trackName string, kind string, rtpSrcUrl string, codecParams webrtc.RTPCodecParameters) (*MediaSource, error) {

// 	// check if the passed track name refers to an already in use track source, in which case we will remove it and replace it.
// 	// _, trackName = mediaCtrl.RemoveTrack(trackName)
// 	if track := mediaCtrl.GetTrack(trackName); track != nil {
// 		return nil, errors.New("Cannot AddRawTrack: The media source track name is already in use")
// 	}

// 	// Create a new media stream rtp reciver and webrtc track from the passed source url
// 	mediaSrc, err := NewRawMediaSource(track, H264FrameDuration, codecParams.MimeType, trackName)
// 	if err != nil {
// 		log.Error("Error creating raw media source: ", err.Error())
// 		return nil, err
// 	}

// 	// Add the new media track back to the connection's media sources map
// 	mediaCtrl.MediaSources[trackName] = mediaSrc

// 	// start relaying bytes from the rtp udp url to the webrtc media track for this track
// 	go mediaSrc.StartMediaStream()

// 	// return the new media source
// 	return mediaSrc, nil
// }

// close the media source and remove it from the map
func (mediaCtrl *MediaController) RemoveTrack(trackName string, closeTrack bool) (error, MediaSource) {
	if track := mediaCtrl.GetTrack(trackName); track != nil {
		if closeTrack {
			track.Close()
		}
		delete(mediaCtrl.MediaSources, trackName)
		return nil, track
	} else {
		return errors.New("Cannot remove track: The track name does not exist: " + trackName), nil
	}
}
