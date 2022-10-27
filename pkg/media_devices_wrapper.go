package webrtc_relay

import (
	"github.com/pion/mediadevices"

	// If you don't like x264, you can also use vpx by importing as below
	// "github.com/pion/mediadevices/pkg/codec/vpx" // This is required to use VP8/VP9 video encoder
	// or you can also use openh264 for alternative h264 implementation
	// "github.com/pion/mediadevices/pkg/codec/openh264"
	// or if you use a raspberry pi like, you can use mmal for using its hardware encoder
	// "github.com/pion/mediadevices/pkg/codec/mmal"
	// "github.com/pion/mediadevices/pkg/codec/opus" // This is required to use opus audio encoder
	"github.com/pion/mediadevices/pkg/codec/x264" // This is required to use h264 video encoder

	// Note: If you don't have a camera or microphone or your adapters are not supported,
	//       you can always swap your adapters with our dummy adapters below.
	_ "github.com/pion/mediadevices/pkg/driver/videotest"
	// _ "github.com/pion/mediadevices/pkg/driver/audiotest"
	_ "github.com/pion/mediadevices/pkg/driver/camera" // This is required to register camera adapter
	// _ "github.com/pion/mediadevices/pkg/driver/microphone" // This is required to register microphone adapter
)

type mediaDevicesWrapper struct {
	CodecSelector *mediadevices.CodecSelector
}

func newMediaDevicesWrapper() *mediaDevicesWrapper {
	mdw := &mediaDevicesWrapper{}

	// configure codec specific parameters
	x264Params, _ := x264.NewParams()
	x264Params.Preset = x264.PresetMedium
	x264Params.BitRate = 1_000_000 // 1mbps

	mdw.CodecSelector = mediadevices.NewCodecSelector(
		mediadevices.WithVideoEncoders(&x264Params),
	)

	return mdw
}

func (mdw *mediaDevicesWrapper) getUserMediaStream() mediadevices.MediaStream {
	mediaStream, err := mediadevices.GetUserMedia(mediadevices.MediaStreamConstraints{
		Video: func(c *mediadevices.MediaTrackConstraints) {},
		Codec: mdw.CodecSelector, // let GetUsermedia know available codecs
	})

	if err != nil {
		println(err)
		return nil
	}

	return mediaStream
}
