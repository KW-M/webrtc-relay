<p align="center">
  <img src="Docs/Images/Webrtc-Relay Logo.svg" alt="drawing" width="200"/>
  <h1 align="center">WebRTC Relay</h1>
</p>

Relay low latency audio/video and data between remote computers and any web browser over WebRTC. This tool is programing language independent (using named pipes) and should compile on \*any os where golang is supported (\*except windows for now).

## Inspiration

I made this tool as an oss alternative to UV4L-WebRTC with fewer webrtc-related constraints. My orignal use case was for controlling underwater drones over the internet, but this library could be used for any purpose where low latency video and/or data transport are needed. This project is similar to [Bot Box](https://github.com/roboportal/bot_box) but with more flexibility, at the cost of less being implemented for you: bring your own frontend / backend etc...

## Details

Behind the scenes, this tool uses the [Pion WebRTC](https://github.com/pion/webrtc) stack and the [Peerjs-go](https://github.com/muka/peerjs-go) library for all the WebRTC-related stuff. This means you can use the PeerJS browser library and any peer.js signaling server (including the inbuilt peerjs-go signaling server) to make the webrtc connection easier. For the back-end interface, you can uses named pipes to send and recieve datachannel messages and video streams from your backend language of choice (so long as it supports reading and writing to named pipes)

![Webrtc-Relay-Overview](Docs/Images/Webrtc-Relay-Overview.drawio.svg)

## Standalone Install

1. Install [Golang](https://go.dev) for your platform.

   > **NOTE** on some linux distros like RaspberryPi os, apt ships an old version of go, so I recommend installing the latest version from the [Go website](https://go.dev/dl) - [Tutorial](https://www.jeremymorgan.com/tutorials/raspberry-pi/install-go-raspberry-pi)

2. In a terminal run: **`git clone github.com/kw-m/webrtc-relay.git`**,
3. **`cd webrtc-relay`** into the folder
4. Run **`go install .`**
   > This should give you an executable called `webrtc-relay` in the folder `<your home folder>/go/bin/` (or wherever $GOPATH is set to).
5. To be able to run the webrtc-relay command from anywhere (required for the examples) add `~/go/bin` to your path by running **`echo "PATH=$PATH:$HOME/go/bin" >> ~/.profile`** then **`source ~/.profile`**
   > **NOTE** if your shell dosn't use `~/.profile`, you'll want to replace `~/.profile` with `~/.bash_profile`, or `~/.zshrc` if either of those files exist.

## Use with any programing langauge

1. To start the relay run the command: **`webrtc-relay -config-file "path/to/your/webrtc-relay-config.json"`**
   > **NOTE:** The python examples start the relay (run this command) as part of the example code.
2. See the [examples/python](examples/python) for a simple python interface as well as example webrtc-relay-config.json files.
3. Basic idea for sending messages to/from browsers through the webrtc data channel:
   1. The relay will create two named pipe files in the "NamedPipeFolder" (specified in your webrtc-relay-config.json) named "from_datachannel_relay.pipe" and "to_datachannel_relay.pipe"
   2. Your program can open these pipe and read/write to them like normal files.
      > **Warning** Opening named pipes for writing without an aready open reader will block your program and/or the relay, so make sure to let relay open and create all named pipes before opening both the "to" and "from" named pipe in your program.
   3. Each browser should connect to the relay peer id using peer JS.
      - To accept media streams, you should have on "Call" listener and answer any call with a null reply stream (browser to peer streams are not currently supported)
        > **NOTE**: The browser side should be a pretty basic peerjs connection. No special api or messages exist for the browser to control the relay, all commands for the relay itself must come from your backend through metadata messages.
   4. Your backend can send specially formatted messages with JSON metadata prepended to tell the relay where to send a message or to perform certain actions - like initiating a media call with a peer.
   - See the python example for the currently supported commands / metadata

## Use in a Go program

1. Add the program to your go.mod with **`go get github.com/kw-m/webrtc-relay`**

2. See the [examples/golang](/examples/golang) folder

## Relay Config Options

See the example configs in [examples/python/configs](examples/python/configs)

For an updated list of config options available see the WebrtcRelayConfig struct in [pkg/consts.go](pkg/consts.go).

## Getting Media from Devices

Most use cases involve getting video or audio from a device attached to the computer. In the examples I use the FFMPEG command line program which can get an h264 encoded video stream from just about any source (or to convert a raw video stream format to h264 encoding for the relay). Hardware encoding or any accelerated encoding can be great here for acheiving sub-second latency. The python examples have a simple class for sending the output of a command-line program like ffmpeg to a media named pipe created by the relay.

**Things to note with FFMPEG:**

1.  You must have the ffmpeg command line program installed on your system.
2.  ffmpeg command line option order MATTERS!
    - `ffmpeg [global options] [input options] -i input [output options] output`
3.  `-re` input option is needed to read input video files or test source at the right rate.
4.  You must have the `-pix-format` output option set to `yuv420p` and `-f` output option set to `h264` (for widest browser support)
5.  The video encoder/codec parameter `-c:v` or `-vcodec`:
    - `libx264` is supported on almost all ffmpeg installs, but probably won't use available hardware encoders.
    - `-vcodec h264_v4l2m2m` should be used on Raspberry Pi's as it takes advantage of the hardware encoding of the Broadcom videocore.
    - (both output h264 video)

### Raspberry Pi:

- You can use raspicam-vid (older raspi OS) or libcamera-vid (Raspberry Pi os buster or later) for hardware encoded video stream using the broadcom video core).

#### -- TEST PATTERNS ---

##### Basic 1280x720 resolution 30 FPS Test Pattern:

```sh
ffmpeg -hide_banner -f lavfi -i "testsrc=size=1280x720:rate=30" -r 30 -vcodec h264_v4l2m2m -f h264 -y pipe:1
```

##### Lower latency with the same test pattern:

```sh
ffmpeg -hide_banner -f lavfi -rtbufsize 1M -use_wallclock_as_timestamps 1 -i "testsrc=size=1280x720:rate=30" -r 30 -vcodec h264_v4l2m2m -preset ultrafast -tune zerolatency  -use_wallclock_as_timestamps 1 -fflags nobuffer -b:v 900k -f h264 -y pipe:1
```

- use_wallclock_as_timestamps 1 | (I think) add timestamps to keep framerate consistant
- fflags nobuffer | Do not buffer the input (testpattern) video at all - may suffer more stuttering/quality loss as a result
- b:v 900k | Target output bitrate 900,000 bits per second. lower number results in lower quality & lower bandwith requirement.
- preset ultrafast | Encoder preset - goes faster?
- tune zerolatency | Encoder tune - also reduces latency?

#### Low latency test pattern overlayed with miliseccond clock:

```sh
ffmpeg -hide_banner -f lavfi -rtbufsize 50M -use_wallclock_as_timestamps 1 -i "testsrc=size=1280x720:rate=30" -r 30 -vf "settb=AVTB,setpts='trunc(PTS/1K)*1K+st(1,trunc(RTCTIME/1K))-1K*trunc(ld(1)/1K)',drawtext=text='%{localtime}.%{eif\:1M*t-1K*trunc(t*1K)\:d}':fontcolor=black@1:fontsize=(h/10):x=(w-text_w)/2:y=10" -vcodec h264_v4l2m2m -preset ultrafast -tune zerolatency   -use_wallclock_as_timestamps 1 -fflags nobuffer -b:v 9k -f h264 -y pipe:1
```

(useful for quickly testing baseline media latency - take a screenshot with real clock and clock in video)
Source: https://stackoverflow.com/questions/47543426/ffmpeg-embed-current-time-in-milliseconds-into-video

### --- Raspberry Pi Camera (or third party camera) Feed ---

h264 raspi camera feed w/o ffmpeg (raspi os buster or later)

```sh
libcamera-vid --width 960 --height 720 --codec h264 --profile high --level 4.2 --bitrate 800000 --framerate 30 --inline 1 --flush 1 --timeout 0 --nopreview 1 --output -
```

Get raw raspi camera output feed using libcamera

```sh
libcamera-vid --width 960 --height 720 --codec yuv420 --framerate 30 --flush 1 --timeout 0 --nopreview 1 --output -
```

pipe raw camera feed into ffmpeg and add timestamp

```sh
libcamera-vid --width 960 --height 720 --codec yuv420 --framerate 20 --flush 1 --timeout 0 --nopreview 1 --output - | ffmpeg -hide_banner -f rawvideo -pix_fmt yuv420p -s 960x720 -framerate 20 -rtbufsize 1M -use_wallclock_as_timestamps 1 -i "pipe:" -vf "settb=AVTB,setpts='trunc(PTS/1K)*1K+st(1,trunc(RTCTIME/1K))-1K*trunc(ld(1)/1K)',drawtext=text='%{localtime}.%{eif\:1M*t-1K*trunc(t\*1K)\:d}':fontcolor=black@1:fontsize=(h/10):x=(w-text_w)/2:y=10" -vcodec h264_v4l2m2m -preset ultrafast -tune zerolatency -use_wallclock_as_timestamps 1 -fflags nobuffer -b:v 100k -f h264 -y pipe:1
```

- NOTE: That the output parameters of libcamera vid and before -i in the ffmpeg commmand must match

## Mac OS:

```sh

```
