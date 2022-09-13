package webrtc_relay

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"strings"

	webrtc "github.com/pion/webrtc/v3"
	log "github.com/sirupsen/logrus"
)

func GetPeerjsCloudPeerInitOptions() PeerInitOptions {
	return PeerInitOptions{
		Key:          "peerjs",
		Host:         "0.peerjs.com",
		Port:         443,
		PingInterval: 5000,
		Path:         "/",
		Secure:       false,
		Configuration: webrtc.Configuration{
			ICEServers: []webrtc.ICEServer{{
				URLs: []string{"stun:stun.l.google.com:19302"},
			}},
			SDPSemantics: webrtc.SDPSemanticsUnifiedPlan,
		},
		Debug:            2,
		Token:            "",
		RetryCount:       2,
		StartLocalServer: false,
	}
}

func GetLocalServerPeerInitOptions() PeerInitOptions {
	return PeerInitOptions{
		Key:          "peerjs",
		Host:         "localhost",
		Port:         9000,
		PingInterval: 5000,
		Path:         "/",
		Secure:       false,
		Debug:        2,
		Token:        "",
		RetryCount:   2,
		Configuration: webrtc.Configuration{
			ICEServers:   []webrtc.ICEServer{},
			SDPSemantics: webrtc.SDPSemanticsUnifiedPlan,
		},
		// -------------------------
		StartLocalServer: true,
		ServerLogLevel:   "warn",
		ExpireTimeout:    300000,
		AliveTimeout:     300000,
		ConcurrentLimit:  100,
		AllowDiscovery:   false,
		CleanupOutMsgs:   100,
	}
}

func GetDefaultRelayConfig() WebrtcRelayConfig {
	peerInitOpts := GetPeerjsCloudPeerInitOptions()
	return WebrtcRelayConfig{
		BasePeerId:                     "go-relay-",
		initialRelayPeerIdEndingNumber: 0,
		UseMemorablePeerIds:            false,
		MemorablePeerIdOffset:          0,
		PeerInitConfigs:                []*PeerInitOptions{&peerInitOpts},
		NamedPipeFolder:                "/tmp/webtrc-relay-pipes/",
		TokenPersistanceFile:           "./webtrc-relay-tokens.json",
		CreateDatachannelNamedPipes:    true,
		MessageDelimiter:               "\n",
		MessageMetadataSeparator:       "|\"|", // intentionally an invalid json string
		AddMetadataToBackendMessages:   true,
		IncludeMessagesInLogs:          false,
		LogLevel:                       "info",
		GoProfilingServerEnabled:       false,
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

func ReadConfigFile(configFilePath string) (WebrtcRelayConfig, error) {
	config := GetDefaultRelayConfig()

	// Read the config file
	configFile, err := os.Open(configFilePath)
	if err != nil {
		return config, err
	}
	defer configFile.Close()

	// read our opened json file as a byte array.
	jsonConfigBytes, err := ioutil.ReadAll(configFile)
	if err != nil {
		return config, err
	}

	// we unmarshal our byteArray which contains our
	// jsonFile's content into 'config' which we defined above
	err = json.Unmarshal(jsonConfigBytes, &config)
	if err != nil {
		return config, err
	}

	return config, nil
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
