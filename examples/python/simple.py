import datetime, asyncio, json, pathlib

from helpers.named_pipe import Duplex_Named_Pipe_Relay, Command_Output_To_Named_Pipe
from helpers.run_cmd import run_cmd_string

# These CONSTANTS must match the config options passed to webrtc-relay when started
RELAY_NAMED_PIPES_FOLDER = './webrtc-relay-pipes/'
RELAY_MSG_METADATA_SEPARATOR = "|\"|"

# Get the folder containing this python file:
THIS_PYTHON_EXAMPLES_FOLDER = pathlib.Path(
    __file__).parent.resolve().as_posix()

# keeps track of the ffmpeg media tracks we have spawned, to show that we can have multiple media streams or send the same media stream to multiple peers:
ffmpegProcessies = []


async def read_messages(duplex_relay):
    while True:
        raw_message = await duplex_relay.get_next_message()
        print("PYTHON: Got Message: " + raw_message)

        # seperate the metadata part of the message from the actual message from the browser:
        msg_parts = raw_message.split(RELAY_MSG_METADATA_SEPARATOR)

        # check if the message from the browser is the "begin_video_stream" message:
        if (len(msg_parts) > 1):
            metadata = msg_parts[0]
            msg = msg_parts[1]
            if (msg == "begin_video_stream"):
                print(
                    "PYTHON: Got \"begin_video_stream\" message from browser, now telling the relay to video call the peer that sent the message"
                )

                # get the SrcPeerId from the message metadata (the peer id of the sender of the begin_video_stream message):
                metadata = json.loads(metadata)
                if "SrcPeerId" in metadata:
                    await start_test_pattern_video_stream(metadata["SrcPeerId"]
                                                          )
                else:
                    print("PYTHON: ERROR: No SrcPeerId in msg metadata: " +
                          raw_message)


async def send_messages(duplex_relay):
    while True:

        # send this message to all connected peers (star symbol means all peers):
        outgoing_msg_metadata = json.dumps({"TargetPeers": ["*"]})
        current_time = datetime.datetime.now().strftime("%H:%M:%S:%f")
        message = outgoing_msg_metadata + RELAY_MSG_METADATA_SEPARATOR + "The python time is now: " + current_time

        print("PYTHON: Sending Message: " + message)
        await duplex_relay.write_message(message)
        await asyncio.sleep(1)  # wait a second before sending the next message


async def start_test_pattern_video_stream(peer_id_to_video_call):
    global ffmpegProcessies

    # Use ffmpeg to send a test pattern video stream to the relay in h264 encoded video format:
    # NOTE that this requires the ffmpeg command to be installed and in the PATH
    if len(ffmpegProcessies) < 2:
        ffmpegProcessies.append(
            run_cmd_string(
                "ffmpeg -re -f lavfi -i testsrc=size=640x480:rate=30 -pix_fmt yuv420p -c:v libx264 -g 10 -preset ultrafast -tune zerolatency -f rtp 'rtp://127.0.0.1:182"
                + str(len(ffmpegProcessies)) + "?pkt_size=1200'"))
        # alternatively replace the run_cmd_string line above with this use vp8 encoding (seems to run slower when run from python, not sure why):
        # run_cmd_string(  "ffmpeg -re -f lavfi -i testsrc=size=640x480:rate=30 -pix_fmt yuv420p -c:v libx264 -g 10 -preset ultrafast -tune zerolatency -f rtp 'rtp://127.0.0.1:122"  + str(len(ffmpegProcessies)) + "?pkt_size=1200'")

    # make a metadata-only message to tell the relay to create a new named pipe for acceping video bytes and then media call the given peer id with that media stream:
    outgoing_msg_metadata = json.dumps({
        "TargetPeerIds": [peer_id_to_video_call],
        "Action":
        "Media_Call_Peer",
        "Params": [
            "This_is_trackid_" + str(len(ffmpegProcessies) - 1),
            "video/H264",  #"video/VP8", specify vp8 mime time to use VP8 video codec instead of H264
            "udp://127.0.0.1:182" + str(len(ffmpegProcessies) - 1),
        ]
    })

    # send the message to the relay:
    await duplex_relay.write_message(outgoing_msg_metadata)


######## Main Program ###########
######################################
async def main():

    # let python know that these should be globally accesable variables (accessable outside of this function)):
    global duplex_relay, webrtc_relay_cmd_process

    # Start the webrtc-relay in a seperate process:
    webrtc_relay_cmd_process = run_cmd_string(
        "webrtc-relay -config-file " + THIS_PYTHON_EXAMPLES_FOLDER +
        "/configs/webrtc-relay-config.json")

    # Configure the named pipes to communicate with the webrtc-relay (ie: send/recive datachannel messages):
    duplex_relay = Duplex_Named_Pipe_Relay(
        RELAY_NAMED_PIPES_FOLDER + 'from_webrtc_relay.pipe',
        RELAY_NAMED_PIPES_FOLDER + 'to_webrtc_relay.pipe',
        create_pipes=True)

    # Setup the asyncio loop to run each of these async functions aka "tasks" aka "coroutines" concurently
    await asyncio.gather(duplex_relay.start_loops(),
                         read_messages(duplex_relay),
                         send_messages(duplex_relay))


##### Run the main program loop, and exit quietly if ctrl-c is pressed (KeyboardInterrupt)  #####
try:
    asyncio.run(main())
except KeyboardInterrupt:
    pass
finally:
    webrtc_relay_cmd_process.kill()
    duplex_relay.cleanup()
