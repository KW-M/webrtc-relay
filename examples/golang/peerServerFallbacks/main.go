package main

import (
	"time"

	webrtc_relay "github.com/kw-m/webrtc-relay/pkg"
)

func main() {

	// create a new config for the webrtc-relay, see pkg/consts.go for available options
	config := webrtc_relay.GetDefaultRelayConfig()
	config.CreateDatachannelNamedPipes = false // we don't need named pipes because this example is entirely in golang (also named pipes will compete with our loop to read messages)

	// set up the options that will be used to connect to the peerjs server
	cloudPeerServerOptions := webrtc_relay.GetPeerjsCloudPeerInitOptions()
	localPeerServerOptions := webrtc_relay.GetLocalServerPeerInitOptions()
	localPeerServerOptions.Port = 9001 // change the port the local peer server will run on. You can change any of the other PeerInitOptions too (before creating the peer relay).

	// set the peer init configs in the order they should start up:
	// each RelayPeer with a unique hostname option should run concurrently and all datachannel messages will be routed to the correct client peer through whichever RelayPeer(s) that client peer is connected to.
	config.PeerInitConfigs = []*webrtc_relay.PeerInitOptions{
		&cloudPeerServerOptions,
		&localPeerServerOptions,
	}

	// create and start the relay
	relay := webrtc_relay.CreateWebrtcRelay(config)
	go relay.Start()
	defer relay.Stop() // stop the relay when the main function exits

	// every second send a message to all connected peers, note that the message is prefixed with a metadata
	// json string followed by the separator string specified in the relay config
	go func() {
		ticker := time.NewTicker(time.Second * 1)
		for {
			<-ticker.C // wait for ticker to trigger and then send the message
			relay.RelayInputMessageChannel <- "{ \"TargetPeers\": [\"*\"] }|\"|Hello World! Time=" + time.Now().String()
		}
	}()

	// listen for messages comming back from any connected peer (ie: from the browser client(s))
	// note these messages will also have a metadata json string and separator string prefixed to them
	go func() {
		for {
			msg := <-relay.RelayOutputMessageChannel
			println("Got Message: " + msg)
		}
	}()

	// select loop keeps the main function from exiting (and thus the relay from stopping)
	select {}
}
