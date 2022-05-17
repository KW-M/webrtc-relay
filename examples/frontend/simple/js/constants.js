// For Connecting to the Peerjs Cloud Signalling Server
const peerServerCloudOptions = {
    host: '0.peerjs.com',
    secure: true,
    path: '/',
    port: 443,
}

// for converting text to/from binary for messages sent/received through the webrtc datachannel
const messageEncoder = new TextEncoder(); // always utf-8
const messageDecoder = new TextDecoder(); // always utf-8
