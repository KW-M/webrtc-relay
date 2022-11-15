package media

// DEPRICATED

// import (
// 	"os"
// 	"syscall"
// 	"time"

// 	// "os"

// 	"github.com/kw-m/webrtc-relay/pkg/util"
// 	webrtc "github.com/pion/webrtc/v3"
// 	log "github.com/sirupsen/logrus"
// )

// type NamedPipeMediaSource struct {
// 	pipeFile       *os.File
// 	pipeFilePath   string
// 	exitSignal     util.UnblockSignal
// 	WebrtcTrack    *webrtc.TrackLocalStaticSample
// 	readInterval   time.Duration
// 	readBufferSize int
// 	log            *log.Entry
// }

// func CreateNamedPipeMediaSource(pipeFilePath string, readBufferSize int, readInterval time.Duration, mediaMimeType string, trackName string) (*NamedPipeMediaSource, error) {
// 	var pipe = NamedPipeMediaSource{
// 		pipeFile:       nil,
// 		pipeFilePath:   pipeFilePath,
// 		exitSignal:     util.NewUnblockSignal(),
// 		readInterval:   readInterval,
// 		readBufferSize: readBufferSize,
// 		log:            log.WithField("media_pipe", pipeFilePath),
// 	}

// 	track, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: mediaMimeType}, trackName, trackName+"-stream")
// 	if err != nil {
// 		pipe.log.Error("Failed to create webrtc track: ", err.Error())
// 		return nil, err
// 	}
// 	pipe.WebrtcTrack = track

// 	// attempt to create the pipe file if doesn't already exist:
// 	if _, err := os.Stat(pipeFilePath); err != nil {
// 		err := syscall.Mkfifo(pipeFilePath, 0666)
// 		if err != nil {
// 			pipe.log.Error("Make named pipe file error:", err.Error())
// 			return nil, err
// 		}
// 	}

// 	return &pipe, nil
// }

// func (pipe *NamedPipeMediaSource) Close() {
// 	if pipe.pipeFile != nil {
// 		pipe.exitSignal.Trigger()
// 		pipe.pipeFile.Close()
// 	}
// }

// func (pipe *NamedPipeMediaSource) GetTrack() *webrtc.TrackLocalStaticSample {
// 	return pipe.WebrtcTrack
// }

// // https://stackoverflow.com/questions/41739837/all-mime-types-supported-by-mediarecorder-in-firefox-and-chrome
// func (pipe *NamedPipeMediaSource) StartMediaStream() error {
// 	defer pipe.Close()
// 	for {

// 		// open the media source pipe file for reading:
// 		var err error
// 		pipe.pipeFile, err = os.OpenFile(pipe.pipeFilePath, os.O_RDONLY, os.ModeNamedPipe|0666)
// 		if err != nil {
// 			pipe.log.Error("Error opening media source named pipe:", err.Error())
// 			<-time.After(time.Second)
// 			continue
// 		}

// 		mimeType := pipe.WebrtcTrack.Codec().MimeType
// 		if mimeType == "video/h264" {
// 			err = read_h264_file(pipe)
// 		} else if mimeType == "video/x-ivf" || mimeType == "video/x-indeo" {
// 			err = read_ivf_file(pipe)
// 		} else if mimeType == "audio/ogg" {
// 			err = read_ogg_file(pipe)
// 		} else {
// 			log.Debug("Unknow Media Source MimeType: " + mimeType + " sending raw stream as fallback")
// 			err = read_file_raw_stream(pipe, pipe.readBufferSize, pipe.readInterval)
// 		}

// 		if err != nil {
// 			pipe.log.Error("Error reading media source:", err.Error())
// 			continue
// 		}

// 		pipe.exitSignal.Wait()
// 		return nil
// 	}
// }
