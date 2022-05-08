package webrtc_relay_core

import (
	"testing"
	"time"

	peer "github.com/muka/peerjs-go"
	"github.com/muka/peerjs-go/server"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func createLocalTestConfig(startLocalServer bool, createPipes bool) ProgramConfig {

	// FOR Connecting to a LOCAL PEERJS SERVER RUNNING ON THIS COMPUTER:
	programConfig := GetDefaultProgramConfig()
	programConfig.CreateDatachannelNamedPipes = createPipes
	programConfig.PeerInitConfigs[0].Host = "localhost"
	programConfig.PeerInitConfigs[0].Port = 9129
	programConfig.PeerInitConfigs[0].Path = "/myapp"
	programConfig.PeerInitConfigs[0].Key = "peerjs"
	programConfig.PeerInitConfigs[0].Secure = false
	programConfig.PeerInitConfigs[0].StartLocalServer = startLocalServer

	return programConfig
}

func getPeerjsGoTestOpts(peerInitConfig *PeerInitOptions) peer.Options {
	opts := peer.NewOptions()
	opts.Path = peerInitConfig.Path
	opts.Host = peerInitConfig.Host
	opts.Port = peerInitConfig.Port
	opts.Key = peerInitConfig.Key
	opts.Secure = peerInitConfig.Secure
	opts.Debug = 0
	return opts
}

func TestRelayStartup(t *testing.T) {
	// create a new relay
	programConfig := createLocalTestConfig(true, false)
	relay := CreateWebrtcRelay(&programConfig)
	go relay.Start()

	<-time.After(time.Second * 50)
	relay.Stop()
}

func TestMsgRelay(t *testing.T) {

	// create a new relay
	localProgramConfigWithServer := createLocalTestConfig(true, false)
	relay1 := CreateWebrtcRelay(&localProgramConfigWithServer)
	go relay1.Start()

	// wait for the relay to start:
	<-time.After(time.Second * 2)

	// create a "client" peer:
	opts := getPeerjsGoTestOpts(localProgramConfigWithServer.PeerInitConfigs[0])
	clientPeer, err := peer.NewPeer("!Client_Peer!", opts)
	assert.NoError(t, err)

	// go func() {
	// 	select {
	// 	case relay1.SendToDatachannelMessages <- "from relay to client":
	// 	case <-time.After(time.Second * 5):
	// 		t.Error("Timeout waiting for message to be sent from relay to client")
	// 	}
	// }()

	expectedMessages := [...]string{
		"{\"SrcPeerId\":\"!Client_Peer!\",\"PeerEvent\":\"Connected\"}",
		"{\"SrcPeerId\":\"!Client_Peer!\"}|\"|from relay to client_msg1",
		"{\"SrcPeerId\":\"!Client_Peer!\"}|\"|from relay to client_msg2",
		"{\"SrcPeerId\":\"!Client_Peer!\",\"PeerEvent\":\"Disconnected\"}",
	}
	go func() {
		msgIndex := 0
		select {
		case msg := <-relay1.FromDatachannelMessages:
			println("relay1 received: " + msg)
			// assert.Equal(t, msg, expectedMessages[msgIndex])
			if msg != expectedMessages[msgIndex] {
				t.Logf("Expected message '%s' but got '%s'", expectedMessages[msgIndex], msg)
			}
			msgIndex++
			if msgIndex == len(expectedMessages) {
				return
			}
		case <-time.After(time.Second * 10):
			t.Error("Timeout waiting for message to be recived on relay")
		}
	}()

	sendingMessages := [...]string{
		"from relay to client_msg1",
		"from relay to client_msg2",
	}
	println("Connecting from client to relay", relay1.ConnCtrl.GetPeerId())
	dataConn, err := clientPeer.Connect(relay1.ConnCtrl.GetPeerId(), peer.NewConnectionOptions())
	assert.NoError(t, err)
	assert.NotNil(t, dataConn)
	clientPeer.On("open", func(id interface{}) {
		println("Client peer opened connection!")
		for _, msg := range sendingMessages {
			println("Sending message from client to relay:", msg)
			dataConn.Send([]byte(msg), false)
		}
	})

	<-time.After(time.Second * 14)
	clientPeer.Destroy()
	<-time.After(time.Second * 14)
	relay1.Stop()
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
// 		relay1.SendToDatachannelMessages <- "from relay1 to relay2"
// 	}()

// 	go func() {
// 		relay2.SendToDatachannelMessages <- "from relay1 to relay2"
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
	opts.Debug = 0
	return opts
}

func startServer() (*server.PeerServer, server.Options) {
	opts := server.NewOptions()
	opts.Port = 9000
	opts.Host = "localhost"
	opts.Path = "/myapp"
	return server.New(opts), opts
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
	defer peerServer.Stop()

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
			log.Println("Received")
			done = true
		})
	})

	conn1, err := peer1.Connect(peer2Name, nil)
	assert.NoError(t, err)
	conn1.On("open", func(data interface{}) {
		for {
			conn1.Send([]byte("hi!"), false)
			<-time.After(time.Millisecond * 1000)
		}
	})

	<-time.After(time.Second * 2)
	assert.True(t, done)
}
