import argparse, distutils, grpclib
from asyncio.log import logger
import asyncio, pathlib
import random
from helpers.run_cmd import run_cmd_string
from protobuf.webrtcrelay import WebRtcRelayStub, EventStreamRequest, PeerConnectedEvent, MsgRecivedEvent, PeerCalledEvent, PeerDataConnErrorEvent, PeerDisconnectedEvent, PeerHungupEvent, PeerMediaConnErrorEvent, RelayConnectedEvent, RelayDisconnectedEvent, RelayErrorEvent, CallRequest, RtpCodecParams, TrackInfo, SendMsgRequest
from grpclib.client import Channel
import betterproto

logger.setLevel("INFO")

# Get the folder containing this python file:
THIS_PYTHON_EXAMPLES_FOLDER = pathlib.Path(
    __file__).parent.resolve().as_posix()

# keeps track of the ffmpeg media tracks we have spawned, to show that we can have multiple media streams or send the same media stream to multiple peers:
ffmpeg_processes = []


async def handle_msg_recived(event: MsgRecivedEvent, exchangeId: int):
    msg = str(event.payload, "utf-8")
    if (msg == "begin_video_stream"):
        print(
            "PYTHON: Got \"begin_video_stream\" message from browser, now telling the relay to video call the peer that sent the message"
        )
        await start_test_pattern_video_stream(event.src_peer_id)


async def start_test_pattern_video_stream(peer_id_to_video_call):
    global ffmpeg_processes

    MAX_FFMPEG_PROCESSES = 3  # arbitrary limit to prevent too show how you can send new or existing video streams to multiple peers
    # Use ffmpeg to send a test pattern video stream to the relay in h264 encoded video format:
    # NOTE that this requires the ffmpeg command to be installed and in the PATH
    ffmpegInstanceNum = len(ffmpeg_processes)
    # space out the next rtp port number to allow every other port to be used for RTCP (RTP Control Protocol)
    rtpPort = 7870 + (ffmpegInstanceNum * 2)
    rtcpPort = rtpPort + 1  # set every other port to be used for RTCP (RTP Control Protocol)
    if len(ffmpeg_processes) < MAX_FFMPEG_PROCESSES:
        inputSrc1 = "lavfi -i testsrc2=size=640x480:rate=30"  # generate a test pattern video stream
        inputSrc2 = "lavfi -i testsrc=size=640x480:rate=30"
        inputSrc = inputSrc1 if len(ffmpeg_processes) % 2 == 0 else inputSrc2
        ffmpeg_processes.append(
            # -x264-params intra-refresh=1,fast-pskip=0 -profile:v baseline -level:v 3.1 -threads 3 -minrate 500K -maxrate 1.3M -bufsize 500K -g 10
            run_cmd_string(
                "ffmpeg -fflags +genpts -protocol_whitelist pipe,tls,file,http,https,tcp,rtp -f {inputSrc} -pix_fmt yuv420p -vf realtime -c:v libx264 -profile:v baseline -level:v 3.1 -preset ultrafast -tune zerolatency -f rtp -sdp_file stream{}.sdp 'rtp://127.0.0.1:{}?rtcpport={}&localrtcpport={}&pkt_size=1200'"
                .format(ffmpegInstanceNum,
                        rtpPort,
                        rtcpPort,
                        rtcpPort,
                        inputSrc=inputSrc)))

    # generate a random exchange id between 0 and 2^32 (max value of a 32 bit uint)
    # in a real application you could store this exchangeId and use it to match up the response events from the relay (will come as get_event_stream() events)  with the grpc request we are about to send
    exchange_id = random.randrange(4294967294)

    # tell the relay to media call the given peer id with the video stream we just created:
    await relay_grpc_stub.call_peer(
        CallRequest(
            target_peer_ids=[peer_id_to_video_call],
            stream_name="test_video_stream",
            relay_peer_number=
            0,  # 0 means use all relay peers that are online within the webrtc-relay instance (in this case there would only be one relay peer because there's only one peerInitConfig in the config file passed to webrtc-relay)
            exchange_id=exchange_id,
            tracks=[
                TrackInfo(
                    name="This_is_trackid_" + str(ffmpegInstanceNum),
                    kind="video",
                    codec=RtpCodecParams(
                        mime_type="video/H264"
                    ),  #specify "video/VP8" mime time to use VP8 video codec instead of H264 (also change the ffmpeg command above to use VP8 (or any webrtc-supported) encoding)
                    rtp_source_url="rtp://127.0.0.1:" + str(rtpPort),
                )
            ]))


async def listen_to_event_stream():
    eventStream = relay_grpc_stub.get_event_stream(
        event_stream_request=EventStreamRequest())
    async for event in eventStream:
        exchange_id = event.exchange_id
        (event_type, e) = betterproto.which_one_of(event, "event")
        if event_type == "msg_recived":
            e: MsgRecivedEvent = e
            print("PYTHON: Got msgRecived event: " + str(e) + " \ exId: " +
                  str(exchange_id))
            await handle_msg_recived(e, exchange_id)
        if event_type == "relay_connected":
            e: RelayConnectedEvent = e
            print("PYTHON: Got relayConnected event: " + str(e) + " \ exId: " +
                  str(exchange_id))
        if event_type == "relay_disconnected":
            e: RelayDisconnectedEvent = e
            print("PYTHON: Got relayDisconnected event: " + str(e) +
                  " \ exId: " + str(exchange_id))
        if event_type == "relay_error":
            e: RelayErrorEvent = e
            print("PYTHON: Got relayError event: " + str(e) + " \ exId: " +
                  str(exchange_id))
        if event_type == "peer_connected":
            e: PeerConnectedEvent = e
            print("PYTHON: Got peerConnected event: " + str(e) + " \ exId: " +
                  str(exchange_id))
        if event_type == "peer_disconnected":
            e: PeerDisconnectedEvent = e
            print("PYTHON: Got peerDisconnected event: " + str(e) +
                  " \ exId: " + str(exchange_id))
        if event_type == "peer_called":
            e: PeerCalledEvent = e
            print("PYTHON: Got peerCalled event: " + str(e) + " \ exId: " +
                  str(exchange_id))
        if event_type == "peer_hungup":
            e: PeerHungupEvent = e
            print("PYTHON: Got peerHungup event: " + str(e) + " \ exId: " +
                  str(exchange_id))
        if event_type == "peer_data_conn_error":
            e: PeerDataConnErrorEvent = e
            print("PYTHON: Got peerDataConnError event: " + str(e) +
                  " \ exId: " + str(exchange_id))
        if event_type == "peer_media_conn_error":
            e: PeerMediaConnErrorEvent = e
            print("PYTHON: Got peerMediaConnError event: " + str(e) +
                  " \ exId: " + str(exchange_id))
        else:
            print("PYTHON: Got unknown GRPC event: " + str(event))


async def send_update_messages():
    while True:
        await asyncio.sleep(2)
        emoji = random.choice(list("🏔🏎🚃🕤🐔🛤🚖🎿🐼🙏🏨💞🐺👽🎯🏊🍘🍕🎡"))
        yield SendMsgRequest(
            exchange_id=random.randrange(4294967294),
            target_peer_ids=["*"],
            payload=bytes("Hello from python! Here's an emoji " + emoji + "\n",
                          "utf-8"))


async def start_grpc_client():
    global relay_grpc_stub
    async with Channel(
            host="127.0.0.1", port=9718
    ) as grpc_channel:  # to use http/2 as the transport for grpc
        # async with Channel(path="./WebrtcRelayGrpc.sock") as chan: # to use unix domain sockets as the transport for grpc

        print("PYTHON: GRPC Channel Created")
        relay_grpc_stub = WebRtcRelayStub(grpc_channel)

        # run the listen_to_event_stream() and send_msg_stream() functions in parallel:
        try:
            await asyncio.gather(
                listen_to_event_stream(),
                relay_grpc_stub.send_msg_stream(send_update_messages()))
        except asyncio.CancelledError:
            pass
        finally:
            # close the grpc channel when the program exits
            grpc_channel.close()


######## Main Program ###########
######################################
async def main():

    # let python know that these should be globally accesable variables (accessable outside of this function)):
    global webrtc_relay_cmd_process
    webrtc_relay_cmd_process = None

    # parse command line arguments:
    parser = argparse.ArgumentParser(
        description='Run demo python webrtc-relay client')
    parser.add_argument(
        '--start-relay',
        type=distutils.util.strtobool,
        required=False,
        default=True,
        help=
        'Start a webrtc-relay process for this demo, if false it will assume you have started a webrtc-relay process already'
    )
    args = parser.parse_args()

    # Start the webrtc-relay in a seperate process:
    webrtc_relay_cmd = "webrtc-relay -config-file " + THIS_PYTHON_EXAMPLES_FOLDER + "/configs/webrtc-relay-config.json"
    # print(webrtc_relay_cmd) # uncomment this line to see the command that will be run
    if (args.start_relay):
        webrtc_relay_cmd_process = run_cmd_string(
            webrtc_relay_cmd)  # start the webrtc-relay process
        await asyncio.sleep(1)  # wait a second for webrtc-relay to start up

    # Configure the grpc client to communicate with the webrtc-relay:
    while True:
        try:
            await start_grpc_client()
            break
        except asyncio.CancelledError:
            break
        except (ConnectionRefusedError, ConnectionAbortedError,
                ConnectionResetError,
                grpclib.exceptions.StreamTerminatedError) as e:
            print(
                "ERROR: Connection to webrtc-relay grpc server lost or failed: "
                + str(e))
            print("Retrying in 5 seconds...")
            await asyncio.sleep(5)


##### Run the main program loop, and exit quietly if ctrl-c is pressed (KeyboardInterrupt)  #####
# loop = asyncio.get_event_loop()
try:
    mainCoroutine = asyncio.run(main())
except KeyboardInterrupt:
    # loop.close()
    print("PYTHON: Ctrl-C pressed, exiting...")
    pass
finally:
    for ffmpeg_process in ffmpeg_processes:
        ffmpeg_process.kill()
    if webrtc_relay_cmd_process:
        webrtc_relay_cmd_process.kill()
