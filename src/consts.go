package webrtc_relay

import webrtc "github.com/pion/webrtc/v3"

// FATAL_PEER_ERROR_TYPES is a list of peer error types that the peerjs-go module can throw which should be handled by closing the peer and restarting the whole peer init -> peer server -> peer conection process.
var FATAL_PEER_ERROR_TYPES = []string{"network", "unavailable-id", "invalid-id", "invalid-key", "browser-incompatible", "webrtc", "server-error", "ssl-unavailable", "socket-error", "socket-closed"}

type PeerInitOptions struct {
	// Key API key for the cloud PeerServer. This is not used for servers other than 0.peerjs.com.
	Key string
	// Server host. Defaults to 0.peerjs.com. Also accepts '/' to signify relative hostname.
	Host string
	//Port Server port. Defaults to 443.
	Port int
	//PingInterval Ping interval in ms. Defaults to 5000.
	PingInterval int
	//Path The path where your self-hosted PeerServer is running. Defaults to '/'.
	Path string
	//Secure true if you're using SSL.
	Secure bool
	//Configuration struct passed to pion RTCPeerConnection. This contains any custom ICE/TURN server configuration. Defaults to { 'iceServers': [{ 'urls': 'stun:stun.l.google.com:19302' }], 'sdpSemantics': 'unified-plan' }
	Configuration webrtc.Configuration
	// Debug: Prints log messages depending on the debug level passed in. Defaults to 0.
	// 0 Prints no logs.
	// 1 Prints only errors.
	// 2 Prints errors and warnings.
	// 3 Prints all logs (verbose).
	Debug int8
	// Retry Count of times to retry connecting to this peer server before moving on to the next peer server in the PeerInitOptions list.
	RetryCount int
	// -------------------------
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

// configuration for webrtc-relay
type WebrtcRelayConfig struct {

	// a list of peer servers to attempt to connect to, in order of preference
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
	CreateDatachannelNamedPipes bool

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

// DatachannelToRelayPipeMetadata is prepended (as a JSON string) to messages sent to your program through the named pipe message relay (when AddMetadataToBackendMessages config is True)
type DatachannelToRelayPipeMetadata struct {

	// PeerId is the peerjs ID of the sender browser peer.
	SrcPeerId string `json:"SrcPeerId,omitempty"`
	// Whenever a peer connects or disconnects this will be "connect" or "disconnect" with the connected or disconnected peer set in SrcPeerId
	PeerEvent string `json:"PeerEvent,omitempty"`
	// Err is the error message if there was an error with the previous metadata action command recived on the unix socket.
	Err string `json:"Err,omitempty"`
}

// RelayPipeToDatachannelMetadata is what is expected to be prepended (as JSON followed by the MessageMetadataSeparator string) to messages recived from your program through the named pipe message relay (when AddMetadataToBackendMessages config is True)
type RelayPipeToDatachannelMetadata struct {
	// TargetPeerIds is the list of peerjs peers (by peer id) this mesage should be sent to. An empty list means broadcast mesage to all connected peers.
	TargetPeerIds []string `json:"TargetPeerIds,omitempty"`
	// An action to be performed by this go code. Options are currently: "Media_Call_Peer"
	Action string `json:"Action,omitempty"`
	// Parameters used by the Action.
	// - When Action is "Media_Call_Peer": This param is an array in the following format ["stream name", "media MIME type eg: video/h264", "filename of media source pipe in the configured NamedPipeFolder eg: lowRezFrontCameraH264.pipe"]
	Params []string `json:"Params,omitempty"`
}
