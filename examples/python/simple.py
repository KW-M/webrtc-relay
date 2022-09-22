from asyncio.log import logger
import datetime, asyncio, json, pathlib
import math
from random import random, randrange
from helpers.run_cmd import run_cmd_string
from protobuf.webrtcrelay import WebRtcRelayStub, EventStreamRequest, PeerConnectedEvent, MsgRecivedEvent, PeerCalledEvent, PeerDataConnErrorEvent, PeerDisconnectedEvent, PeerHungupEvent, PeerMediaConnErrorEvent, RelayConnectedEvent, RelayDisconnectedEvent, RelayErrorEvent, CallRequest, RtpCodecParams, TrackInfo
from grpclib.client import Channel
import betterproto

# These CONSTANTS must match the config options passed to webrtc-relay when started
RELAY_NAMED_PIPES_FOLDER = './webrtc-relay-pipes/'
RELAY_MSG_METADATA_SEPARATOR = "|\"|"

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

    # Use ffmpeg to send a test pattern video stream to the relay in h264 encoded video format:
    # NOTE that this requires the ffmpeg command to be installed and in the PATH
    ffmpegInstanceNum = len(ffmpeg_processes)
    if len(ffmpeg_processes) < 2:
        ffmpeg_processes.append(
            run_cmd_string(
                "ffmpeg -re -f lavfi -i testsrc=size=640x480:rate=30 -pix_fmt yuv420p -c:v libx264 -g 10 -preset ultrafast -tune zerolatency -f rtp 'rtp://127.0.0.1:182"
                + str(ffmpegInstanceNum) + "?pkt_size=1200'"))
        # alternatively replace the run_cmd_string line above with this use vp8 encoding (seems to run slower when run from python, not sure why):
        # run_cmd_string(  "ffmpeg -re -f lavfi -i testsrc=size=640x480:rate=30 -pix_fmt yuv420p -c:v libx264 -g 10 -preset ultrafast -tune zerolatency -f rtp 'rtp://127.0.0.1:122"  + str(len(ffmpegProcessies)) + "?pkt_size=1200'")

    # generate a random exchange id between 0 and 2^32 (max value of a 32 bit uint)
    # in a real application you could store this exchangeId and use it to match up the response events from the relay (will come as get_event_stream() events)  with the grpc request we are about to send
    exchange_id = randrange(4294967294)

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
                    rtp_source_url="udp://127.0.0.1:182" +
                    str(ffmpegInstanceNum),
                )
            ]))


async def start_grpc_client():
    global grpc_channel, relay_grpc_stub
    async with Channel(host="127.0.0.1", port=9023) as chan: # to use http/2 as the transport for grpc
    # async with Channel(path="./WebrtcRelayGrpc.sock") as chan: # to use unix domain sockets as the transport for grpc
        grpc_channel = chan
        relay_grpc_stub = WebRtcRelayStub(grpc_channel)
        eventStream = relay_grpc_stub.get_event_stream(
            event_stream_request=EventStreamRequest())
        async for event in eventStream:
            exchange_id = event.exchange_id
            (event_type, e) = betterproto.which_one_of(event, "event")
            print("PYTHON: Got GRPC Event: " + event_type)
            if event_type == "msg_recived":
                e: MsgRecivedEvent = e
                print("PYTHON: Got msgRecived event: " + str(e) + " \ exId: " +
                      str(exchange_id))
                await handle_msg_recived(e, exchange_id)
            if event_type == "relay_connected":
                e: RelayConnectedEvent = e
                print("PYTHON: Got relayConnected event: " + str(e) +
                      " \ exId: " + str(exchange_id))
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
                print("PYTHON: Got peerConnected event: " + str(e) +
                      " \ exId: " + str(exchange_id))
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


######## Main Program ###########
######################################
async def main():

    # let python know that these should be globally accesable variables (accessable outside of this function)):
    global webrtc_relay_cmd_process

    # Start the webrtc-relay in a seperate process:
    webrtc_relay_cmd_process = run_cmd_string(
        "webrtc-relay -config-file " + THIS_PYTHON_EXAMPLES_FOLDER +
        "/configs/webrtc-relay-config.json")

    # Configure the grpc client to communicate with the webrtc-relay:
    while True:
        try:
            await start_grpc_client()
        except (ConnectionRefusedError, ConnectionAbortedError,
                ConnectionResetError) as e:
            logger.info(
                "ERROR: Connection to webrtc-relay grpc server failed: " +
                str(e))
            logger.info("Retrying in 1 second...")

            await asyncio.sleep(1)

    # # Setup the asyncio loop to run each of these async functions aka "tasks" aka "coroutines" concurently
    # await asyncio.gather(duplex_relay.start_loops(),
    #                      read_messages(duplex_relay),
    #                      send_messages(duplex_relay))


##### Run the main program loop, and exit quietly if ctrl-c is pressed (KeyboardInterrupt)  #####
try:
    asyncio.run(main())
except KeyboardInterrupt:
    pass
finally:
    for ffmpeg_process in ffmpeg_processes:
        ffmpeg_process.kill()
    webrtc_relay_cmd_process.kill()
    grpc_channel.close()
