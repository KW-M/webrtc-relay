// for converting text to/from binary for messages sent/received through the webrtc datachannel
const messageEncoder = new TextEncoder(); // always utf-8
const messageDecoder = new TextDecoder(); // always utf-8
var messagePingIntervalId = null
var relayWebrtcDatachannel = null

// setup callback function that will be setup to run when the connection to the relay is open:
// see the connectToRelay() function
var relayConectionOpenCallback = function (relayDatachannel) {
    // Receive messages
    relayDatachannel.on('data', (data) => {
        msg = messageDecoder.decode(data);
        // add the message to the page:
        document.body.appendChild(document.createTextNode(String(msg)));
    });

    // send a message to the relay every second with the current time.
    // messagePingIntervalId = setInterval(() => {
    //     relayDatachannel.send(messageEncoder.encode("Current time from BROWSER: " + Date.now()));
    // }, 1000);

    relayWebrtcDatachannel = relayDatachannel
    startVideoButton.disabled = false
}

// setup callback function that will be setup to run when the peer connection to the relay is closed, or failed:
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

// Setup the connect button
var connectButton = document.getElementById('connect_btn')
connectButton.addEventListener('click', () => {
    // prompt the user for the peer id of the relay
    relayPeerId = window.prompt("Enter relay peer id to connect to (set in the webrtc-relay-config.json when you started the relay, followed by zero unless that peer id was already taken)", "go-relay-0")
    // show a help message on the page:
    document.body.appendChild(document.createTextNode("Open Browser Console For Progress. Type any key to send a message once connected."))
    // disable the connect button:
    connectButton.disabled = true
    // connect to the relay passing the callbacks defined above:
    connectToRelay(relayPeerId, relayConectionOpenCallback, relayConectionFailedCallback)
});

// Setup the start video button
var startVideoButton = document.getElementById('start_video_btn')
startVideoButton.addEventListener('click', () => {
    if (relayDatachannel == null || relayDatachannel.open == false) {
        alert("No Open Datachannel");
    } else {
        // tell the PYTHON backend to start the video:
        relayDatachannel.send(messageEncoder.encode("start video stream"));
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
window.onbeforeunload = () => {
    cleanupConnection();
}
