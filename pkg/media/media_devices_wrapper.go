package media

import (
	"fmt"

	"github.com/pion/mediadevices"

	wrConfig "github.com/kw-m/webrtc-relay/pkg/config"

	// If you don't like x264, you can also use vpx by importing as below
	// "github.com/pion/mediadevices/pkg/codec/vpx" // This is required to use VP8/VP9 video encoder
	// or you can also use openh264 for alternative h264 implementation
	// "github.com/pion/mediadevices/pkg/codec/openh264"
	// or if you use a raspberry pi like, you can use mmal for using its hardware encoder
	// "github.com/pion/mediadevices/pkg/codec/mmal"
	// "github.com/pion/mediadevices/pkg/codec/opus" // This is required to use opus audio encoder
	"github.com/pion/mediadevices/pkg/codec/vpx" // This is required to use h264 video encoder
	"github.com/pion/mediadevices/pkg/codec/x264"
	"github.com/pion/mediadevices/pkg/prop"

	// Note: If you don't have a camera or microphone or your adapters are not supported,
	//       you can always swap your adapters with our dummy adapters below.
	// _ "github.com/pion/mediadevices/pkg/driver/videotest"
	"github.com/pion/mediadevices/pkg/driver"
	"github.com/pion/mediadevices/pkg/driver/cmdsource"
	"github.com/pion/mediadevices/pkg/frame"
	// _ "github.com/pion/mediadevices/pkg/driver/audiotest"
	// _ "github.com/pion/mediadevices/pkg/driver/camera" // This is required to register camera adapter
	// _ "github.com/pion/mediadevices/pkg/driver/microphone" // This is required to register microphone adapter
)

var ffmpegFrameFormatMap = map[frame.Format]string{
	frame.FormatI420: "yuv420p",
	frame.FormatNV21: "nv21",
	frame.FormatNV12: "nv12",
	frame.FormatYUY2: "yuyv422",
	frame.FormatUYVY: "uyvy422",
	frame.FormatZ16:  "gray",
}

type mediaDevicesWrapper struct {
	CodecSelector *mediadevices.CodecSelector
	Streams       map[string]mediadevices.MediaStream
}

func getVideoCmdFfmpegTestpattern(input string, width int, height int, frameRate float32, frameFormat frame.Format) (string, prop.Media) {
	command := fmt.Sprintf("ffmpeg -f lavfi -i %s=size=%dx%d:rate=%f -vf realtime -f rawvideo -pix_fmt %s -", input, width, height, frameRate, ffmpegFrameFormatMap[frameFormat])
	mediaProps := prop.Media{
		DeviceID: "ffmpeg 1",
		Video: prop.Video{
			Width:       width,
			Height:      height,
			FrameFormat: frameFormat,
			FrameRate:   frameRate,
		},
	}
	return command, mediaProps
}

func newMediaDevicesWrapper() *mediaDevicesWrapper {
	mdw := &mediaDevicesWrapper{
		Streams: make(map[string]mediadevices.MediaStream),
	}

	// // configure source video
	// cmdString, mediaProps := getVideoCmdFfmpeg("testsrc", 640, 480, 30, frame.FormatI420)
	// mediaProps.DeviceID = "ffmpeg 1"
	// err := cmdsource.AddVideoCmdSource(cmdString, []prop.Media{mediaProps}, 10)
	// if err != nil {
	// 	panic(err)
	// }

	// // configure source video
	// cmdString2, mediaProps2 := getVideoCmdFfmpeg("testsrc2", 640, 480, 30, frame.FormatI420)
	// mediaProps2.DeviceID = "ffmpeg 2"
	// err = cmdsource.AddVideoCmdSource(cmdString2, []prop.Media{mediaProps2}, 10)
	// if err != nil {
	// 	panic(err)
	// }

	// configure h264 codec specific parameters
	x264Params, _ := x264.NewParams()
	x264Params.Preset = x264.PresetMedium
	x264Params.BitRate = 1_000_000 // 1mbps

	// configure vp8 codec specific parameters
	vp8Params, _ := vpx.NewVP8Params()
	vp8Params.BitRate = 1_000_000 // 1mbps
	vp8Params.ErrorResilient = vpx.ErrorResilientPartitions
	vp8Params.LagInFrames = 1

	mdw.CodecSelector = mediadevices.NewCodecSelector(
		mediadevices.WithVideoEncoders(&vp8Params, &x264Params),
	)

	return mdw
}

func (mdw *mediaDevicesWrapper) AddVideoCmdSource(config *wrConfig.MediaSourceConfig) error {
	// configure source video
	mediaProps :=
		prop.Media{
			Video: prop.Video{
				Width:       config.Width,
				Height:      config.Height,
				FrameFormat: config.PixelFormat,
				FrameRate:   config.FrameRate,
			},
		}
	err := cmdsource.AddVideoCmdSource(config.SourceLabel, config.SourceCmd, []prop.Media{mediaProps}, 10)
	return err
}

func (mdw *mediaDevicesWrapper) storeMediaStreamReference(deviceLabel string) mediadevices.MediaStream {
	drivers := driver.GetManager().Query(func(d driver.Driver) bool {
		return d.Info().DeviceType == driver.CmdSource && d.Info().Label == deviceLabel
	})
	if len(drivers) == 0 {
		panic("no driver found")
	}
	id := drivers[0].ID()
	mediaStream, err := mediadevices.GetUserMedia(mediadevices.MediaStreamConstraints{
		Video: func(c *mediadevices.MediaTrackConstraints) {
			c.DeviceID = prop.String(id)
		},
		Codec: mdw.CodecSelector, // let GetUsermedia know available codecs
	})

	// mediaStream.GetTracks()[0].(*mediadevices.VideoTrack).Source

	if err != nil {
		panic(err)
	}

	return mediaStream
}

func (mdw *mediaDevicesWrapper) GetMediaStream(deviceLabel string) mediadevices.MediaStream {
	if mdw.Streams[deviceLabel] == nil {
		mdw.Streams[deviceLabel] = mdw.storeMediaStreamReference(deviceLabel)
	}
	return mdw.Streams[deviceLabel]
}
