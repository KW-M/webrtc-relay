var thisPeer = null

/*
 peerjs api reference: https://peerjs.com/docs/
 */

function cleanupConnection() {
    if (thisPeer) thisPeer.destroy()
    thisPeer = null
}

function connectToRelay(relayPeerId, connectionOpenCallback, connectionFailedCallback) {

    // Create the peerjs peer for this browser. You can pass a set peerid as the first argument, null will give us a random unique one.
    console.info("Creating Peer...");
    thisPeer = new Peer(null, {
        host: '0.peerjs.com',
        secure: true,
        path: '/',
        port: 443,
        key: 'peerjs',
    });

    // This event is called when the peer server to acknowledge that it knows about us and give us a unique peer id.
    thisPeer.on('open', (realPeerId) => {
        console.info("Connection to peer server established! Our PeerID:", realPeerId);

        // ---------------------------------------------------------------------------------------------------------------------

        // Actually connect to the webrtc-relay peer:
        console.info("Connecting to relay peer: ", relayPeerId)
        relayDatachannel = thisPeer.connect(relayPeerId, {
            reliable: true, //  True if we want datachannel messages to be guaranteed to arrive in order at the cost of some overhead. (This usually works, but not always)
            serialization: 'none' // webrtc-relay doesn't support js binarypack serialization so we must set this to none.
        })

        // Handle when the datachannel is open (ie the connection to the rov sucecceded):
        relayDatachannel.on('open', () => {
            console.info("Relay connection (ie: datachannel) is open!")
            connectionOpenCallback(relayDatachannel)
        });

        // Handle the case where the connection to the webrtc-relay has an error:
        relayDatachannel.on('error', (err) => {
            console.error('Relay connection (ie: datachannel) error: ', err);
            connectionFailedCallback(err)
        });

        // Handle the case where the connection to the webrtc-relay has disconnected (eg: because their wifi went down) but might be able to connect again when internet is regained:
        relayDatachannel.on('disconnected', () => {
            console.warn('Relay (ie: datachannel) has disconnected.');
            connectionFailedCallback('datachannel-disconnected')
        });

        // Handle the case where the connection to the webrtc-relay has closed (eg: because the relay closed the connection or we closed the connection, or the relay went offline for too long):
        relayDatachannel.on('close', () => {
            console.warn('Relay (ie: datachannel) has closed');
            connectionFailedCallback('datachannel-close')
        });
    });

    // Handle if the peer is closed, either because of an error or we call close on it.
    thisPeer.on('disconnected', () => {
        console.info("Got disconnected from peer server")
        console.info("Attempting to reconnect to peer server...")
        thisPeer.reconnect();
    });

    // Handle if the peer is closed, either because of an error or we call close on it.
    thisPeer.on('close', () => {
        console.info('Peer server connection closed.');
        connectionFailedCallback('closed')
    });

    // Handle if an error occurs in the future (including errors connecting to the peer server or with connections to other peers):
    thisPeer.on('error', (err) => {
        if (err.type == 'browser-incompatible') {
            alert('Your browser does not support some WebRTC features, please use a newer / different browser.');
        } else if (err.type == "peer-unavailable" && thisPeer.open) {
            console.warn("Relay is not yet online")
        } else if (err.type == "webrtc") {
            console.alert("Webrtc browser error, please reload page...")
        } else {
            console.error("Peerjs error:", err)
        }
        connectionFailedCallback(err)
    });

    // Handle when we receive a media call from the webrtc-relay (or any peer, so you should check the peerid in a real app):
    thisPeer.on('call', (call) => {

        console.info('Got media channel from PeerId: ' + call.peer)

        // Answer the call. Webrtc-relay currently doesn't support receving media, so we can't answer with user media (PRs welcome though!).
        call.answer(null);

        // Handle when the stream handshake is finished (or another stream is added to the mediachannel)
        call.on('stream', (remoteStream) => {
            console.info('Got media stream!');

            // Add the stream to a new html video element and append it to the page:
            var video = document.createElement('video');
            video.srcObject = remoteStream;
            video.muted = true
            video.autoplay = true
            video.controls = false
            document.body.appendChild(video);
            video.play();

        });
    });

}