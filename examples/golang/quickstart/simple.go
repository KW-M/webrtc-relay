package main

import (
	"fmt"
	"time"

	webrtc_relay "github.com/kw-m/webrtc-relay/pkg"
	webrtc_relay_config "github.com/kw-m/webrtc-relay/pkg/config"
	webrtc_relay_proto "github.com/kw-m/webrtc-relay/pkg/proto"
)

func main() {

	// create a new config for the webrtc-relay, see src/consts.go for available options
	config := webrtc_relay_config.GetDefaultRelayConfig()
	config.StartGRPCServer = false // we don't need named pipes because this example is entirely in golang (also named pipes will compete with our for loop to read messages)

	// create and start the relay
	relay := webrtc_relay.NewWebrtcRelay(config)
	go relay.Start()
	defer relay.Stop() // stop the relay when the main function exits

	// every second send a message to all connected peers, note that the message is prefixed with a metadata
	// json string followed by the separator string specified in the relay config
	go func() {
		ticker := time.NewTicker(time.Second * 1)
		for {
			<-ticker.C // wait for ticker to trigger and then send the message
			relay.SendMsg([]string{"*"}, []byte("Relay, this is Relay do you copy? The time is "+time.Now().Local().Format(time.RFC850)+"\n"), 0, 123456789)
			// relay.RelayInputMessageChannel <- "{ \"TargetPeers\": [\"*\"] }|\"|Relay, this is Relay do you copy? The time is " + time.Now().Local().Format(time.RFC850) + "\n"
		}
	}()

	// listen for messages comming back from any connected peer (ie: from the browser client(s))
	// note these messages will also have a metadata json string and separator string prefixed to them
	eventStream := relay.GetEventStream()
	go func() {
		for {
			e := <-eventStream
			switch event := e.Event.(type) {
			case *webrtc_relay_proto.RelayEventStream_MsgRecived:
				fmt.Printf("Got message from peer %s, via relay #%d: %s\n", event.MsgRecived.SrcPeerId, event.MsgRecived.RelayPeerNumber, string(event.MsgRecived.Payload))
			}
		}
	}()

	// select loop keeps the main function from exiting (and thus the relay from stopping)
	select {}
}
