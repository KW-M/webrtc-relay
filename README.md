<p align="center">
  <img src="Docs/Images/Webrtc-Relay Logo.svg" alt="drawing" width="200"/>
  <h1 align="center">WebRTC Relay</h1>
</p>

Relay low latency video and data between remote computers and any web browser over WebRTC. This tool is programing language independent (using named pipes) and should compile on \*any os where golang is supported (\*except windows for now).

## Inspiration

I made this tool as an alternative to UV4L-WebRTC with fewer webrtc-related constraints. My orignal use case was for controlling underwater drones over the internet, but this library could be used for any purpose where low latency video and/or data transport are needed. This project is a lot like [Bot Box](https://github.com/roboportal/bot_box) but with more flexibility, at the cost of less being implemented for you: bring your own frontend / backend etc...

## Details

Behind the scenes, this tool uses the [Pion WebRTC](https://github.com/pion/webrtc) stack and the Peerjs-go library for all the WebRTC-related stuff. This means you can use the PeerJS browser library and any peer.js signaling server (including the inbuilt peerjs-go signaling server) to make the webrtc connection easier. For the back-end interface, you can uses named pipes to send and recieve datachannel messages and video streams from your backend language of choice (so long as it supports reading and writing to named pipes)

![Webrtc-Relay-Overview](Docs/Images/Webrtc-Relay-Overview.svg)

## Standalone Install

1. Install [Golang](https://go.dev) for your platform.

   > Note On raspberry pi, apt ships an old version of go, so I recommend installing the latest version from the [Go website](https://go.dev/dl) - [Tutorial](https://www.jeremymorgan.com/tutorials/raspberry-pi/install-go-raspberry-pi)

2. In a terminal run: **`go install github.com/kw-m/webrtc-relay/cmd/webrtc-relay@latest`**

## Use in a Go program

1. Add the program to your go.mod with **`go get github.com/kw-m/webrtc-relay`**

2. See the [examples/golang](/examples/golang) folder

## Use with another programming language

1. To start the relay run the command: **`webrtc-relay -config-file "path/to/your/webrtc-relay-config.json"`**
   > **NOTE:** The python examples start the relay (run this command) as part of the example code.
2. See the [examples/python](examples/python) for a simple python interface as well as example webrtc-relay-config.json files.
3. Basic idea for sending messages to browsers through the webrtc data channel:
   1. The relay will create two named pipe files in the "relayPipeFolder" (specified in your webrtcRelayConfig.json) named "
   2. One

**Compressed video stream could come from:**

- ffmpeg can get you a hardware encoded stream on most devices.

Raspberry Pi:

- Use Raspicam or Libcamera-vid (raspi os buster or later) for hardware encoded video stream (broadcom video core).

```sh
# raspberry broadcom videocore TEST PATTERN WORKS!
ffmpeg -hide_banner -f lavfi -i "testsrc=size=1280x720:rate=30" -r 30 -vcodec h264_v4l2m2m -f h264 -y pipe:1 > /tmp/webrtc-relay/vido.pipe
## raspberry pi hardware-encoded lower latency test pattern:
ffmpeg -hide_banner -f lavfi -rtbufsize 1M -use_wallclock_as_timestamps 1 -i "testsrc=size=1280x720:rate=30" -r 30 -vcodec h264_v4l2m2m -preset "ultrafast" -tune zerolatency  -use_wallclock_as_timestamps 1 -fflags nobuffer -b:v 200k -f h264 -y pipe:1 > /tmp/webrtc-relay/vido.pipe
## raspi test pattern with clock:
ffmpeg -hide_banner -f lavfi -rtbufsize 50M -use_wallclock_as_timestamps 1 -i "testsrc=size=1280x720:rate=30" -r 30 -vf "settb=AVTB,setpts='trunc(PTS/1K)*1K+st(1,trunc(RTCTIME/1K))-1K*trunc(ld(1)/1K)',drawtext=text='%{localtime}.%{eif\:1M*t-1K*trunc(t*1K)\:d}':fontcolor=black@1:fontsize=(h/10):x=(w-text_w)/2:y=10" -vcodec h264_v4l2m2m -preset "ultrafast" -tune zerolatency   -use_wallclock_as_timestamps 1 -fflags nobuffer -b:v 9k -f h264 -y pipe:1 > /tmp/webrtc-relay/vido.pipe
## raspi camera feed (raspi os buster or later)
libcamera-vid --width 960 --height 720 --codec h264 --profile high --level 4.2 --bitrate 800000 --framerate 30 --inline 1 --flush 1 --timeout 0 --nopreview 1 --output -
# get raw libcamera (raspicamera output feed)
libcamera-vid --width 960 --height 720 --codec yuv420 --framerate 30 --flush 1 --timeout 0 --nopreview 1 --output -
## pipe raw feed into ffmpeg and add timestamp (note that the output parameters of libcamera vid and before -i in the ffmpeg cmd must mach)
libcamera-vid --width 960 --height 720 --codec yuv420 --framerate 20 --flush 1 --timeout 0 --nopreview 1 --output - | ffmpeg -hide_banner -f rawvideo -pix_fmt yuv420p -s 960x720 -framerate 20 -rtbufsize 1M -use_wallclock_as_timestamps 1 -i "pipe:" -vf "settb=AVTB,setpts='trunc(PTS/1K)*1K+st(1,trunc(RTCTIME/1K))-1K*trunc(ld(1)/1K)',drawtext=text='%{localtime}.%{eif\:1M*t-1K*trunc(t*1K)\:d}':fontcolor=black@1:fontsize=(h/10):x=(w-text_w)/2:y=10" -vcodec h264_v4l2m2m -preset "ultrafast" -tune zerolatency   -use_wallclock_as_timestamps 1 -fflags nobuffer -b:v 100k -f h264 -y pipe:1 > /tmp/webrtc-relay/vido.pipe
```
