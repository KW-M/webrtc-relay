package media

import (
	"errors"
	"strings"

	"github.com/pion/webrtc/v3"
	log "github.com/sirupsen/logrus"
)

type MediaSource interface {
	GetTrack() *webrtc.TrackLocal // the webrtc track to hold the media
	StartMediaStream()
	AddConsumer(peerId string)
	RemoveConsumer(peerId string)
	GetConsumerPeerIds() []string
	Close()
}

type MediaController struct {
	// map of media streams being sent to this relay from the backend or frontend clients (key is the track name)
	MediaSources map[string]*RtpMediaSource
}

func NewMediaController() *MediaController {
	return &MediaController{
		MediaSources: make(map[string]*RtpMediaSource),
	}
}

func (mediaCtrl *MediaController) GetTrack(trackName string) *RtpMediaSource {
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
	_ = mediaCtrl.RemoveTrack(trackName)

	// make sure the  metadata is a valid media track udp (rtp) url;
	sourceParts := strings.Split(rtpSrcUrl, "/")
	if sourceParts[0] != "udp:" {
		return nil, errors.New("Cannot start media call: The media source rtp url must start with 'udp://'")
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

// close the media source and remove it from the map
func (mediaCtrl *MediaController) RemoveTrack(trackName string) error {
	if track := mediaCtrl.GetTrack(trackName); track != nil {
		track.Close()
		delete(mediaCtrl.MediaSources, trackName)
		return nil
	} else {
		return errors.New("Cannot remove track: The track name does not exist: " + trackName)
	}
}
