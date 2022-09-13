// For Connecting to the Peerjs Cloud Signalling Server
const peerServerCloudOptions = {
    host: '0.peerjs.com',
    secure: true,
    path: '/',
    port: 443,
    key: 'peerjs',
}


// For Connecting to the Peer Server run by the Relay Locally (if you have the relay running with a startLocalServer config set to true (see the golang peerServerFallbacks example))
const localPeerServerOptions = {
    host: '127.0.0.1',
    secure: false,
    path: '/',
    port: 9000,
    key: 'peerjs',
}
