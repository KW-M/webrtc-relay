package webrtc_relay

import (
	"fmt"
	"testing"
	"time"

	relay_config "github.com/kw-m/webrtc-relay/pkg/config"
	"github.com/kw-m/webrtc-relay/pkg/proto"
	peer "github.com/muka/peerjs-go"
	"github.com/muka/peerjs-go/server"
	"github.com/pion/webrtc/v3"
	"github.com/stretchr/testify/assert"
)

func createCloudTestConfig(createPipes bool) relay_config.WebrtcRelayConfig {
	// FOR Connecting to a LOCAL PEERJS SERVER RUNNING ON THIS COMPUTER:
	programConfig := relay_config.GetDefaultRelayConfig()
	programConfig.LogLevel = "debug"
	programConfig.StartGRPCServer = false
	programConfig.PeerInitConfigs[0].StartLocalServer = false
	return programConfig
}

func createLocalTestConfig(startLocalServer bool) relay_config.WebrtcRelayConfig {
	// FOR Connecting to a LOCAL PEERJS SERVER RUNNING ON THIS COMPUTER:
	localConfig := relay_config.GetLocalServerPeerInitOptions()
	programConfig := relay_config.GetDefaultRelayConfig()
	programConfig.LogLevel = "debug"
	programConfig.PeerInitConfigs[0] = &localConfig
	// programConfig.PeerInitConfigs[0].Port = 9129
	// programConfig.PeerInitConfigs[0].ServerLogLevel = "debug"
	// programConfig.PeerInitConfigs[0].Debug = 3
	// programConfig.PeerInitConfigs[0].ConcurrentLimit = 0 // uncomment to restore segfault
	return programConfig
}

func getPeerjsGoTestOpts(peerInitConfig *relay_config.PeerInitOptions) peer.Options {
	opts := peer.NewOptions()
	opts.Path = peerInitConfig.Path
	opts.Host = peerInitConfig.Host
	opts.Port = peerInitConfig.Port
	opts.Key = peerInitConfig.Key
	opts.Secure = peerInitConfig.Secure
	opts.Debug = peerInitConfig.Debug
	opts.Configuration = peerInitConfig.Configuration
	return opts
}

func TestRelayStartup(t *testing.T) {
	// create a new relay
	programConfig := createLocalTestConfig(true)
	relay := NewWebrtcRelay(programConfig)
	go relay.Start()

	<-time.After(time.Second * 20)
	relay.Stop()
}

func TestCloudMsgRelay(t *testing.T) {
	// create a new relay
	cloudProgramConfig := createCloudTestConfig(true)
	testMsgRelay(t, cloudProgramConfig)
}

func TestLocalMsgRelay(t *testing.T) {
	// create a new relay
	localProgramConfigWithServer := createLocalTestConfig(true)
	testMsgRelay(t, localProgramConfigWithServer)
}

func createFrontendPeer(t *testing.T, peerInitConfig *relay_config.PeerInitOptions) *peer.Peer {
	opts := getPeerjsGoTestOpts(peerInitConfig)
	frontendPeer, err := peer.NewPeer("!WR_Client_Peer!", opts)
	assert.NoError(t, err)
	return frontendPeer
}

func testWithFrontendPeer(t *testing.T, relayId string, peerInitConfig *relay_config.PeerInitOptions, connectToRelay bool, mediaCallRelay bool) (*peer.Peer, [4]string) {
	frontendPeer := createFrontendPeer(t, peerInitConfig)

	frontendMessagesToSend := [...]string{
		"from frontend to relay msg 1",
		"from frontend to relay msg 2",
		"from frontend to relay msg 3",
		"begin test media call",
	}

	frontendPeer.On("open", func(id interface{}) {
		clientId := id.(string)
		opts := peer.NewConnectionOptions()
		println("Frontend Peer Open ", clientId)

		if connectToRelay {
			// connect to the relay
			println("Frontend peer: now connecting to ", relayId, " (Relay)")
			dataConn, err := frontendPeer.Connect(relayId, opts)
			assert.NoError(t, err)
			assert.NotNil(t, dataConn)
			dataConn.On("open", func(none interface{}) {
				println("Frontend peer: connection to relay open!")
				for _, msg := range frontendMessagesToSend {
					println("Frontend peer: Sending message to relay:", msg)
					assert.NoError(t, dataConn.Send([]byte(msg), false))
				}
				dataConn.On("data", func(msgBytes interface{}) {
					msg := string(msgBytes.([]byte))
					println("Frontend peer:Got msg from relay:", msg)
				})
			})
		} else {
			// wait for relay to connect to us
			println("Frontend peer: waiting for ", relayId, " (Relay) to connect")
			frontendPeer.On("connection", func(dataConn interface{}) {
				dc := dataConn.(*peer.DataConnection)
				fmt.Printf("Frontend peer: got peer data connection! %+v\n", dc)
				dc.On("data", func(msgBytes interface{}) {
					msg := string(msgBytes.([]byte))
					println("Frontend peer: Got msg from relay:", msg)
				})
			})
		}

		if mediaCallRelay {
			// media call the relay
			println("Frontend peer: media calling ", relayId, " (Relay)")
			mediaConn, err := frontendPeer.Call(relayId, &webrtc.TrackLocalStaticSample{}, opts)
			assert.NoError(t, err)
			assert.NotNil(t, mediaConn)
			mediaConn.On("stream", func(relayStream interface{}) {
				println("Frontend peer: got media stream from relay!")
			})
		} else {
			// wait for relay to media call us
			println("Frontend peer: waiting for ", relayId, " (Relay) to media call")
			frontendPeer.On("call", func(mediaConn interface{}) {
				mc := mediaConn.(*peer.MediaConnection)
				fmt.Printf("Frontend peer: got peer call! %+v\n", mc)
			})
		}
	})

	frontendPeer.On("error", func(err interface{}) {
		t.Error("Frontend Peer Error: ", err.(error).Error())
	})

	return frontendPeer, frontendMessagesToSend
}

func testMsgRelay(t *testing.T, config relay_config.WebrtcRelayConfig) {
	// print out the passed config:
	fmt.Printf("test webrtcRelayConfig: %+v\n", config)

	// create and start the webrtc_relay:
	relay := NewWebrtcRelay(config)
	relay.Start()
	defer relay.Stop()

	// give the relay time to start up
	<-time.After(time.Second * 5)
	println("--------- Starting Client Peer ---------")

	// create a test "frontend" peer:
	peerInitConfig := config.PeerInitConfigs[0]
	relayPeer, ok := relay.connCtrl.RelayPeers[peerInitConfig.RelayPeerNumber]
	if !ok {
		t.Error("Relay peer not found with number: ", peerInitConfig.RelayPeerNumber)
	}
	relayId := relayPeer.GetPeerId()
	frontendPeer, frontendMessagesToSend := testWithFrontendPeer(t, relayId, peerInitConfig, true, false)
	defer frontendPeer.Destroy()

	// get the relay peer's data connection:
	msgIndex := 0
	relayEvents := relay.GetEventStream()
	for {
		select {
		case evt := <-relayEvents:
			switch event := evt.Event.(type) {
			case *proto.RelayEventStream_PeerConnected:
				println("Relay: Peer Connected: ", event.PeerConnected.SrcPeerId)
			case *proto.RelayEventStream_PeerDisconnected:
				println("Relay: Peer Disconnected: ", event.PeerDisconnected.SrcPeerId)
			case *proto.RelayEventStream_MsgRecived:
				msg := string(event.MsgRecived.Payload)
				println("relay1 received: " + msg)
				assert.Equal(t, msg, frontendMessagesToSend[msgIndex])
				if msg != frontendMessagesToSend[msgIndex] {
					t.Logf("Expected message '%s' but got '%s'", frontendMessagesToSend[msgIndex], msg)
				}
				msgIndex++
				if msgIndex == len(frontendMessagesToSend) {
					return
				}
			}
		case <-time.After(time.Second * 15):
			t.Error("Timeout waiting for message to be recived on relay")
		}
	}
}

// func TestTwoRelay(t *testing.T) {
// 	// create a new relay
// 	localProgramConfigWithServer := createLocalTestConfig(true)
// 	relay1 := CreateWebrtcRelay(&localProgramConfigWithServer)
// 	go relay1.Start()

// 	localProgramConfigNoServer := createLocalTestConfig(false)
// 	relay2 := CreateWebrtcRelay(&localProgramConfigNoServer)
// 	go relay2.Start()

// 	go func() {
// 		relay1.RelayInputMessageChannel <- "from relay1 to relay2"
// 	}()

// 	go func() {
// 		relay2.RelayInputMessageChannel <- "from relay1 to relay2"
// 	}()

// 	<-time.After(time.Second * 10)
// 	relay1.Stop()
// 	relay2.Stop()
// }

/// peerjs tests:

func getTestOpts(serverOpts server.Options) peer.Options {
	opts := peer.NewOptions()
	opts.Path = serverOpts.Path
	opts.Host = serverOpts.Host
	opts.Port = serverOpts.Port
	opts.Secure = false
	opts.Debug = 3
	return opts
}

func startServer() (*server.PeerServer, server.Options) {
	opts := server.NewOptions()
	opts.Port = 9000
	opts.Host = "localhost"
	opts.Path = "/myapp"
	return server.New(opts), opts
}

func TestHellodWorld(t *testing.T) {

	peer1Name := "peer1____d"
	peer2Name := "peer2____d"

	peerServer, serverOpts := startServer()
	err := peerServer.Start()
	if err != nil {
		t.Logf("Server error: %s", err)
		t.FailNow()
	}
	defer assert.NoError(t, peerServer.Stop())

	<-time.After(10 * time.Second)
	println("STARTING PEERS")

	peer1, err := peer.NewPeer(peer1Name, getTestOpts(serverOpts))
	assert.NoError(t, err)
	defer peer1.Close()

	peer2, err := peer.NewPeer(peer2Name, getTestOpts(serverOpts))
	assert.NoError(t, err)
	defer peer2.Close()

	// done := false
	done := false
	peer2.On("connection", func(data interface{}) {
		conn2 := data.(*peer.DataConnection)
		conn2.On("data", func(data interface{}) {
			// Will print 'hi!'
			println("Received:", string(data.([]byte)))
			if string(data.([]byte)) == "hi!" {
				done = true
			}
		})
	})

	conn1, err := peer1.Connect(peer2Name, nil)
	assert.NoError(t, err)
	conn1.On("open", func(data interface{}) {
		for {
			assert.NoError(t, conn1.Send([]byte("hi!"), false))
			<-time.After(time.Millisecond * 1000)
		}
	})

	<-time.After(time.Second * 2)
	assert.True(t, done)
}

func TestHelloWorld(t *testing.T) {

	peer1Name := "peer1____d"
	peer2Name := "peer2____d"

	peerServer, serverOpts := startServer()
	err := peerServer.Start()
	if err != nil {
		t.Logf("Server error: %s", err)
		t.FailNow()
	}
	defer assert.NoError(t, peerServer.Stop())

	peer1, err := peer.NewPeer(peer1Name, getTestOpts(serverOpts))
	assert.NoError(t, err)
	defer peer1.Close()

	peer2, err := peer.NewPeer(peer2Name, getTestOpts(serverOpts))
	assert.NoError(t, err)
	defer peer2.Close()

	// done := false
	done := false
	peer2.On("connection", func(data interface{}) {
		conn2 := data.(*peer.DataConnection)
		conn2.On("data", func(data interface{}) {
			// Will print 'hi!'
			println("Received:", string(data.([]byte)))
			if string(data.([]byte)) == "hi!" {
				done = true
			}
		})
	})

	conn1, err := peer1.Connect(peer2Name, nil)
	assert.NoError(t, err)
	conn1.On("open", func(data interface{}) {
		for {
			assert.NoError(t, conn1.Send([]byte("hi!"), false))
			<-time.After(time.Millisecond * 1000)
		}
	})

	<-time.After(time.Second * 2)
	assert.True(t, done)
}
