package main

import (
	"time"

	webrtc_relay "github.com/kw-m/webrtc-relay/pkg"
)

func main() {

	// create a new config for the webrtc-relay, see pkg/consts.go for available options
	config := webrtc_relay.GetDefaultRelayConfig()

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
			relay.RelayInputMessageChannel <- "{ TargetPeers: [] }|\"|Hello World! Time=" + time.Now().String()
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
