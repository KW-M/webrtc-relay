// for converting text to/from binary for messages sent/received through the webrtc datachannel
const messageEncoder = new TextEncoder(); // always utf-8
const messageDecoder = new TextDecoder(); // always utf-8
const messageDisplayElement = document.getElementById("messages")
var messagePingIntervalId = null
var relayWebrtcDatachannel = null


// !!!! ------ set this somewhere in your code (see where it is used below): -------!!!!!
// const PEERJS_CONNECTION_OPTIONS = peerServerCloudOptions;

// setup callback function that will be setup to run when the connection to the relay is open:
// see the connectToRelay() function
var relayConectionOpenCallback = function (relayDatachannel) {


    // Receive messages
    relayDatachannel.on('data', (data) => {
        msg = messageDecoder.decode(data);
        // add the message to the page:
        messageDisplayElement.appendChild(document.createTextNode(String(msg) + "\n"));

        relayDatachannel.send(messageEncoder.encode("B" + String(msg)));
    });

    console.log("DC", relayDatachannel)
    // send a message to the relay every second with the current time.
    messagePingIntervalId = setInterval(() => {
        relayDatachannel.send(messageEncoder.encode("BR_TIME: " + Date.now()));
    }, 100);

    relayWebrtcDatachannel = relayDatachannel
    startVideoButton.disabled = false
}

// make a function that will be setup to run when the peer connection to the relay is closed, or failed:
// see the connectToRelay() function
var relayConectionFailedCallback = function (error) {
    // do something w error message:
    console.log("Error connecting to relay: ", error)
    // close and cleanup the connection
    cleanupConnection();
    // set the datachannel to null so other functions know it's  Not open
    relayWebrtcDatachannel = null
    // stop the "send a ping message to the relay every second" interval
    clearInterval(messagePingIntervalId)
    // re-enable the connect button:
    connectButton.disabled = false
    // disable the start video button:
    startVideoButton.disabled = true
}

// make a function that will be setup to run when a remote peer (presumably the relay, but not guaranteed) sends a media stream:
// see the connectToRelay() function
function mediaStreamRecivedCallback(remoteStream) {
    console.log("Got remote stream from relay", remoteStream)
    remoteStream.onaddtrack = function (event) {
        console.log("Got remote track from relay", event.track)
        createVideoPlayer(new MediaStream([event.track]));
    }
    createVideoPlayer(remoteStream)
}

function createVideoPlayer(MediaStream) {
    // Add the stream to a new html video element and append it to the page:
    var video = document.createElement('video');
    messageDisplayElement.appendChild(video);
    video.srcObject = MediaStream;
    video.autoplay = true
    video.controls = false
    video.play();
}

// Setup the connect button
var connectButton = document.getElementById('connect_btn')
connectButton.addEventListener('click', () => {
    // prompt the user for the peer id of the relay
    relayPeerId = window.prompt("Enter relay peer id to connect to (set in the webrtc-relay-config.json when you started the relay, followed by zero unless that peer id was already taken)", "go-relay-0")
    // disable the connect button:
    connectButton.disabled = true
    // connect to the relay passing the callbacks defined above:
    connectToRelay(relayPeerId, PEERJS_CONNECTION_OPTIONS, relayConectionOpenCallback, relayConectionFailedCallback, mediaStreamRecivedCallback)
});

// Setup the start video button
var startVideoButton = document.getElementById('start_video_btn')
startVideoButton.addEventListener('click', () => {
    if (relayDatachannel == null || relayDatachannel.open == false) {
        alert("No Open Datachannel");
    } else {
        // tell the PYTHON backend to start the video:
        relayDatachannel.send(messageEncoder.encode("begin_video_stream"));
    }
});

// allow the user to send their own messages to the relay
window.addEventListener('keypress', () => {
    if (relayDatachannel == null || relayDatachannel.open == false) {
        alert("No Open Datachannel");
    } else {
        var msg = window.prompt("Message to send to relay:");
        if (msg) relayDatachannel.send(messageEncoder.encode(msg));
    }
});

// when the user closes their browser / tab, close & cleanup the peer connection:
// window.onbeforeunload = () => {
//     cleanupConnection();
// }
