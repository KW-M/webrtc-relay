package webrtc_relay

import (
	"errors"
	"io"
	"time"

	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/h264reader"
	"github.com/pion/webrtc/v3/pkg/media/ivfreader"
	"github.com/pion/webrtc/v3/pkg/media/oggreader"
	log "github.com/sirupsen/logrus"
)

///https://github.com/edaniels/gostream/blob/master/codec/x264/encoder.go
// https://github.com/pion/mediadevices/blob/08a396571f87ee2888fc855964a5442f2a163879/track.go#L314

func read_h264_file(pipe *NamedPipeMediaSource) error {

	// NAIVE IMPLEMENTATION:
	// return read_raw_stream(pipe, 4096, h264FrameDuration)

	// SMARTER IMPLEMENTATION:
	// from https://github.com/ashellunts/ffmpeg-to-webrtc/blob/master/src/main.go
	// Send our video a frame at a time. Pace our sending so we send it at the same speed it should be played back as.
	// This isn't required since the video is timestamped, but we will such much higher loss if we send all at once.
	//
	// It is important to use a time.Ticker instead of time.Sleep because
	// * avoids accumulating skew, just calling time.Sleep didn't compensate for the time spent parsing the data
	// * works around latency issues with Sleep (see https://github.com/golang/go/issues/44343)

	h264, h264Err := h264reader.NewReader(pipe.pipeFile)
	if h264Err != nil {
		log.Error("h264reader Initilization Error", h264Err)
		return h264Err
	}

	spsAndPpsCache := []byte{}
	ticker := time.NewTicker(h264FrameDuration)
	for {
		select {
		case <-pipe.exitSignal.GetSignal():
			return nil
		case <-ticker.C:
			nal, h264Err := h264.NextNAL()
			if h264Err == io.EOF {
				log.Println("All video frames parsed and sent")
				// pipe.exitReadLoopSignal.Trigger()
				// return
				continue
			} else if h264Err != nil {
				log.Error("h264reader Decode Error: ", h264Err)
				return h264Err
			}
			nal.Data = append([]byte{0x00, 0x00, 0x00, 0x01}, nal.Data...)

			if nal.UnitType == h264reader.NalUnitTypeSPS || nal.UnitType == h264reader.NalUnitTypePPS {
				spsAndPpsCache = append(spsAndPpsCache, nal.Data...)
				continue
			} else if nal.UnitType == h264reader.NalUnitTypeCodedSliceIdr {
				nal.Data = append(spsAndPpsCache, nal.Data...)
				spsAndPpsCache = []byte{}
			}

			if err := pipe.WebrtcTrack.WriteSample(media.Sample{Data: nal.Data, Duration: time.Second}); err != nil {
				log.Println("Error writing h264 video track sample: ", err)
			}
		}
	}

}

func read_ivf_file(pipe *NamedPipeMediaSource) error {

	// from https://github.com/ashellunts/ffmpeg-to-webrtc/blob/master/src/main.go
	// Send our video a frame at a time. Pace our sending so we send it at the same speed it should be played back as.
	// This isn't required since the video is timestamped, but we will such much higher loss if we send all at once.
	//
	// It is important to use a time.Ticker instead of time.Sleep because
	// * avoids accumulating skew, just calling time.Sleep didn't compensate for the time spent parsing the data
	// * works around latency issues with Sleep (see https://github.com/golang/go/issues/44343)

	ivfReader, ivfHeader, ivfErr := ivfreader.NewWith(pipe.pipeFile)
	if ivfErr != nil {
		return errors.New("ivfReader Initilization Error" + ivfErr.Error())
	}
	print(ivfReader, ivfHeader)
	return errors.New("IVF READER NOT IMPLEMENTED")
}

func read_ogg_file(pipe *NamedPipeMediaSource) error {

	// only works with opus codec in the ogg container
	// https://github.com/pion/webrtc/issues/2181
	oggReader, _, oggErr := oggreader.NewWith(pipe.pipeFile)
	if oggErr != nil {
		return errors.New("oggReader Initilization Error: " + oggErr.Error())
	}

	ticker := time.NewTicker(33 * time.Millisecond)
	for {
		select {
		case <-pipe.exitSignal.GetSignal():
			return nil
		case <-ticker.C:
			oggPageBytes, _, Err := oggReader.ParseNextPage()
			if Err == io.EOF {
				log.Println("All video frames parsed and sent")
				// pipe.exitReadLoopSignal.Trigger()
				// return
				continue
			} else if Err != nil {
				log.Error("oggreader Decode Error: ", Err)
				return Err
			}

			if err := pipe.WebrtcTrack.WriteSample(media.Sample{Data: oggPageBytes, Duration: time.Second}); err != nil {
				log.Println("Error writing h264 video track sample: ", err)
			}
		}
	}
}

func read_file_raw_stream(pipe *NamedPipeMediaSource, readBufferSize int, readInterval time.Duration) error {
	// just keeps reading the named pipe bytes at a set intervals and pushing them onto the webrtc track
	tmpReadBuf := make([]byte, readBufferSize)
	ticker := time.NewTicker(readInterval)
	for {
		select {
		case <-pipe.exitSignal.GetSignal():
			return nil
		case <-ticker.C:
			numBytes, err := pipe.pipeFile.Read(tmpReadBuf) // read as much data as possible
			if err != nil {
				return errors.New("Error reading from media pipe source: " + err.Error())
			}
			if numBytes == 0 {
				log.Println("All video frames parsed and sent")
				return nil
			}

			if err := pipe.WebrtcTrack.WriteSample(media.Sample{Data: tmpReadBuf, Duration: readInterval}); err != nil {
				return errors.New("Error writing webrtc track sample: " + err.Error())
			}
		}
	}
}

// // https://developer.mozilla.org/en-US/docs/Web/Media/Formats/Video_codecs#avc_h.264
// // Find the H264 codec in the list of codecs supported by the remote peer (aka the pilot's browser)
// var h264PayloadType uint8 = 0
// for _, videoCodec := range mediaEngine.GetCodecsByKind(webrtc.RTPCodecTypeVideo) {
// 	if videoCodec.Name == "H264" {
// 		h264PayloadType = videoCodec.PayloadType
// 		break
// 	}
// }
// // if the payloadTypeNumber from never changed, the broswer doesn't support H264 (highly unlikely)
// if h264PayloadType == 0 {
// 	fmt.Println("Remote peer does not support H264")
// 	continue
// }
