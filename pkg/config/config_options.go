package config

import (
	peerjs "github.com/muka/peerjs-go"
	peerjsServer "github.com/muka/peerjs-go/server"
	webrtc "github.com/pion/webrtc/v3"
)

// configuration for webrtc-relay
type WebrtcRelayConfig struct {

	// a list of peer servers to attempt to connect to in parallel when the relay starts up (see PeerInitOptions type for details)
	// Default: !!empty list!!
	PeerInitConfigs []*PeerInitOptions

	// the webrtc-relay will try to aquire a peerjs peer id that is this string with an int tacked on the end.
	// if that peer id is taken, it will increment the ending int and try again.
	// Default: "go-relay-"
	BasePeerId string

	// To handle the case where multiple relays are running at the same time,
	// we make the PeerId of this ROBOT the BasePeerId plus this number tacked on
	// the end (eg: iROBOT-0) that we increment if the current peerId is already taken.
	// Default: 0
	initialRelayPeerIdEndingNumber int

	// Use a longer but more memorable name in place of the ending number to distiguish webrtc-relay peer ids (name is deterministically based on the end number and MemorablePeerIdOffset, see uid_generation.go).
	// Default: false
	UseMemorablePeerIds bool

	// if UseMemorablePeerIds is true, this number rotates the name indecies for a given peer end number to make name collisions even less likely. Choose any random number that fits in the positive int range.
	// Default: 0
	MemorablePeerIdOffset uint32

	// File path to use for temporarilly storing the token used by peerjs server to verify this client (webrtc-relay) really is the peer id it says it is.
	TokenPersistanceFile string

	// The folder path (w trailing slash) where the named pipes should be created to act as a relay for messages and media streams sent from your prefered programming language (eg: python)
	// Default: "/tmp/webtrc-relay-pipes/"
	NamedPipeFolder string

	// Whether or not the webrtc-relay should attempt to create the named pipes for data communication (set to false if you wish to send messages directly from go code)
	// (note that regardless of this value, the datachannel message string format is the same also media channel named pipes will always be created).
	// Default: true
	StartGRPCServer bool

	// The string that goes between each message when sent through the named pipe:
	// Default: "\n"
	MessageDelimiter string

	// the string that goes between the message metadata json and the actual message when sent through the named pipe
	// Default: |"|  (an intentionally an invalid json string)
	MessageMetadataSeparator string

	// if true, when a datachannel message is recived or a peer connects, metadata like the sender's peer id will be prepended to all message as json before being sent to the named pipe.
	// if true, the webrtc-relay will expect json to be prepended to messages comming back from the named pipe in the same format (json metadata, then seperator, then message)
	AddMetadataToBackendMessages bool

	// LogLevel: The log verbosity to use for the webrtc-relay. Must be one of: critical, error, warn, info, debug. (debug is most verbose)
	// Default: "warn"
	LogLevel string

	// IncludeMessagesInLogs: If true, messages sent and recived from the backend will be included in the logs, careful with using this in production.
	// Default: "warn"
	IncludeMessagesInLogs bool

	// Go Profiling Server Enabled: If true, the webrtc-relay will start a go pprof profiling server on port 6060, careful with using this in production.
	// see: https://go.dev/blog/pprof
	// Default: false
	GoProfilingServerEnabled bool
}

type PeerInitOptions struct {
	// RelayPeerNumber (required): A unique number you must provide that identifies this relay peer within webrtc-relay and grpc calls. Whenever some event happens, like a message recived, you will recive this number to indicate which Relay peer the event originated from)
	// This number is *NOT* the peer id of the peerjs peer. It is only used between the relay go code & grpc-backend side.
	RelayPeerNumber uint32

	// Server host. Defaults to 0.peerjs.com. Also accepts '/' to signify relative hostname.
	Host string

	//Port Server port. Defaults to 443.
	Port int

	//Key: API key for the cloud PeerServer. This is not used for servers other than peerjs cloud (0.peerjs.com).
	Key string

	//PingInterval: Heartbeat ping interval in ms  to check /ensure the socket connection to the peer server is still open. Defaults to 5000.
	PingInterval int

	//Path The path where your self-hosted PeerServer is running. Defaults to '/'.
	Path string

	//Secure: true if the server supports using SSL (and you want to use it).
	Secure bool

	//Configuration: struct passed to pion RTCPeerConnection. This contains any custom ICE/TURN server configuration. Defaults to { 'iceServers': [{ 'urls': 'stun:stun.l.google.com:19302' }], 'sdpSemantics': 'unified-plan' }
	Configuration webrtc.Configuration

	// Debug: Prints log messages depending on the debug level passed in. Defaults to 0.
	// 0 Prints no logs.
	// 1 Prints only errors.
	// 2 Prints errors and warnings.
	// 3 Prints all logs (verbose).
	Debug int8

	// ----------- (local peerjs server options) --------------
	// StartLocalServer - if true, the peerjs-go module will start a local peerjs Server with the same config, and then connect to it.
	StartLocalServer bool

	// (local peerjs Server only) Prints log messages from the local peer js server depending on the debug level passed in. Defaults to 0.
	// Options: critical, error, warn, info, debug. (in order of verbosity least to greatest)
	ServerLogLevel string

	// (local peerjs server only) How long to hold onto disconnected peer connections before releasing them & their peer ids.
	ExpireTimeout int64

	// (local peerjs server only) How long to untill disconeected peer connections are marked as not alive.
	AliveTimeout int64

	// (local peerjs server only) How many peers are allowed to be connected to this peer server at the same time.
	ConcurrentLimit int

	// (local peerjs server only) Allow peerjs clients to get a list of connected peers from this server
	AllowDiscovery bool

	// (local peerjs server only) How long the outgoing server websocket message queue can grow before dropping messages.
	CleanupOutMsgs int
}

func GetPeerjsCloudPeerInitOptions() PeerInitOptions {
	return PeerInitOptions{
		RelayPeerNumber: 1,
		Host:            "0.peerjs.com",
		Port:            443,
		Key:             "peerjs",
		PingInterval:    4000,
		Path:            "/",
		Secure:          true,
		Configuration: webrtc.Configuration{
			ICEServers: []webrtc.ICEServer{
				{
					URLs: []string{"stun:stun.l.google.com:19302"},
				},
				{
					URLs:           []string{"turn:eu-0.turn.peerjs.com:3478", "turn:us-0.turn.peerjs.com:3478"},
					Username:       "peerjs",
					Credential:     "peerjsp",
					CredentialType: webrtc.ICECredentialTypePassword,
				},
			},
			SDPSemantics: webrtc.SDPSemanticsUnifiedPlan,
		},
		Debug:            2,
		StartLocalServer: false,
	}
}

func GetLocalServerPeerInitOptions() PeerInitOptions {
	return PeerInitOptions{
		RelayPeerNumber: 2,
		Key:             "peerjs",
		Host:            "localhost",
		Port:            9000,
		PingInterval:    5000,
		Path:            "/",
		Secure:          false,
		Debug:           2,
		Configuration: webrtc.Configuration{
			ICEServers: []webrtc.ICEServer{
				{
					URLs: []string{"stun:stun.l.google.com:19302"},
				},
			},
			SDPSemantics: webrtc.SDPSemanticsUnifiedPlan,
		},
		// Configuration: webrtc.Configuration{
		// 	ICEServers:   []webrtc.ICEServer{}, // empty list means don't use any ice servers (use only local network / ip)
		// 	SDPSemantics: webrtc.SDPSemanticsUnifiedPlan,
		// },
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
		TokenPersistanceFile:           "/tmp/webtrc-relay-tokens.json",
		StartGRPCServer:                true,
		MessageDelimiter:               "\n",
		MessageMetadataSeparator:       "|\"|", // intentionally an invalid json string
		AddMetadataToBackendMessages:   true,
		IncludeMessagesInLogs:          false,
		LogLevel:                       "info",
		GoProfilingServerEnabled:       false,
	}
}

func PeerOptsFromInitOpts(config *PeerInitOptions) peerjs.Options {
	var peerOptions = peerjs.NewOptions()
	peerOptions.Host = config.Host
	peerOptions.Port = config.Port
	peerOptions.Path = config.Path
	peerOptions.Secure = config.Secure
	peerOptions.Key = config.Key
	peerOptions.Debug = config.Debug
	peerOptions.Configuration = config.Configuration
	return peerOptions
}

func PeerServerOptsFromInitOpts(config *PeerInitOptions) peerjsServer.Options {
	var peerServerOptions = peerjsServer.NewOptions()
	peerServerOptions.LogLevel = config.ServerLogLevel
	peerServerOptions.Host = config.Host
	peerServerOptions.Port = config.Port
	peerServerOptions.Path = config.Path
	peerServerOptions.Key = config.Key
	peerServerOptions.ExpireTimeout = config.ExpireTimeout
	peerServerOptions.AliveTimeout = config.AliveTimeout
	peerServerOptions.AllowDiscovery = config.AllowDiscovery
	peerServerOptions.ConcurrentLimit = config.ConcurrentLimit
	peerServerOptions.CleanupOutMsgs = config.CleanupOutMsgs
	return peerServerOptions
}
