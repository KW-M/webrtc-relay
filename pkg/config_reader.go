package webrtc_relay_core

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"strings"

	webrtc "github.com/pion/webrtc/v3"
	log "github.com/sirupsen/logrus"
)

func GetDefaultPeerInitOptions() PeerInitOptions {
	return PeerInitOptions{
		Key:          "peerjs",
		Host:         "0.peerjs.com",
		Port:         443,
		PingInterval: 5000,
		Path:         "/",

		// Secure: true if you're using SSL with this server.
		Secure: false,
		//Configuration hash passed to RTCPeerConnection. This hash contains any custom ICE/TURN server configuration. Defaults to { 'iceServers': [{ 'urls': 'stun:stun.l.google.com:19302' }], 'sdpSemantics': 'unified-plan' }
		Configuration: webrtc.Configuration{
			ICEServers: []webrtc.ICEServer{{
				URLs: []string{"stun:stun.l.google.com:19302"},
			}},
			SDPSemantics: webrtc.SDPSemanticsUnifiedPlan,
		},
		// Debug
		// Prints log messages depending on the debug level passed in. Defaults to 0.
		// 0 Prints no logs.
		// 1 Prints only errors.
		// 2 Prints errors and warnings.
		// 3 Prints all logs.
		Debug: 0,
		//Token a string to group peers
		Token: "",
		// Retry Count of times to retry connecting to this peer server before moving on to the next peer server in the PeerInitOptions list.
		RetryCount: 2,

		// -------------------------
		// StartLocalServer - if true, the peerjs-go module will start a local peerjs Server with the same config, and then connect to it.
		StartLocalServer: false,
		// (local peerjs Server only) Prints log messages from the local peer js server depending on the debug level passed in. Defaults to 0.
		ServerLogLevel: "error",
		// (local peerjs server only) How long to hold onto disconnected peer connections before releasing them & their peer ids.
		ExpireTimeout: 300000,
		// (local peerjs server only) How long to untill disconeected peer connections are marked as not alive.
		AliveTimeout: 300000,
		// (local peerjs server only) How many peers are allowed to be connected to this peer server at the same time.
		ConcurrentLimit: 100,
		// (local peerjs server only) Allow peerjs clients to get a list of connected peers from this server
		AllowDiscovery: false,
		// (local peerjs server only) How long the outgoing server websocket message queue can grow before dropping messages.
		CleanupOutMsgs: 100,
	}
}

func GetDefaultProgramConfig() ProgramConfig {
	peerInitOpts := GetDefaultPeerInitOptions()
	return ProgramConfig{
		BasePeerId:                  "go-robot-",
		PeerInitConfigs:             []*PeerInitOptions{&peerInitOpts},
		NamedPipeFolder:             "/tmp/webtrc-relay-pipes/",
		CreateDatachannelNamedPipes: true,
		MessageDelimiter:            "\n",
		MessageMetadataSeparator:    "|\"|", // intentionally an invalid json string
		AddMetadataToPipeMessages:   true,
	}
}

func StringToLogLevel(s string) (log.Level, error) {
	s = strings.ToLower(s)
	switch s {
	case "debug":
		return log.DebugLevel, nil
	case "info":
		return log.InfoLevel, nil
	case "warn":
		return log.WarnLevel, nil
	case "error":
		return log.ErrorLevel, nil
	case "fatal":
		return log.FatalLevel, nil
	case "panic":
		return log.PanicLevel, nil
	case "critical":
		return log.PanicLevel, nil
	default:
		return log.WarnLevel, errors.New("Invalid log level: " + s)
	}
}

func ReadConfigFile(configFilePath string) (*ProgramConfig, error) {
	config := GetDefaultProgramConfig()

	// Read the config file
	configFile, err := os.Open(configFilePath)
	if err != nil {
		return &config, err
	}
	defer configFile.Close()

	// read our opened jsonFile as a byte array.
	byteValues, err := ioutil.ReadAll(configFile)
	if err != nil {
		return &config, err
	}

	// we unmarshal our byteArray which contains our
	// jsonFile's content into 'users' which we defined above
	err = json.Unmarshal(byteValues, &config)
	if err != nil {
		return &config, err
	}

	return &config, nil
}

// if tries == 0 {
// 	// FOR CLOUD HOSTED PEERJS SERVER running on heroku (or wherever - you could use the default peerjs cloud server):
// 	peerServerOptions.Host = "0.peerjs.com"
// 	peerServerOptions.Port = 443
// 	peerServerOptions.Path = "/"
// 	peerServerOptions.Key = "peerjs"
// 	peerServerOptions.Secure = true
// 	peerServerOptions.PingInterval = 3000
// } else {
// 	// FOR LOCAL PEERJS SERVER RUNNING ON THIS raspberrypi (not heroku):
// 	peerServerOptions.Host = "localhost"
// 	peerServerOptions.Port = 9000
// 	peerServerOptions.Path = "/"
// 	peerServerOptions.Key = "peerjs"
// 	peerServerOptions.Secure = false
// 	peerServerOptions.PingInterval = 3000
// }
