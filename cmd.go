package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	relayLib "github.com/kw-m/webrtc-relay/pkg"

	log "github.com/sirupsen/logrus"
)

// command line flag placeholder variables
var configFilePath string

// Parse the command line parameters passed to program in the shell eg "-a" in "ls -a"
func parseProgramCmdlineFlags() {
	flag.StringVar(&configFilePath, "config-file", "webrtc-relay-config.json", "Path to the config file. Default is webrtc-relay-config.json")
	flag.Parse()
}

func main() {

	parseProgramCmdlineFlags()
	println("------------ Starting WebRTC Relay ----------------|")

	// read the provided config file and set it to the config struct variable
	config, err := relayLib.ReadConfigFile(configFilePath)
	if err != nil {
		log.Fatal("Failed to read config file: ", err)
	}

	// Create a simple boolean "channel" that we can use to signal to go subroutine functions that they should stop cleanly:
	programShouldQuitSignal := *relayLib.NewUnblockSignal()
	defer programShouldQuitSignal.Trigger()

	// start the relay
	relay := relayLib.CreateWebrtcRelay(config)
	go relay.Start()
	defer relay.Stop()

	// Wait for a signal to stop the program
	systemExitCalled := make(chan os.Signal, 1)                                                     // Create a channel to listen for an interrupt signal from the OS.
	signal.Notify(systemExitCalled, os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGHUP) // tell the OS to send us a signal on the systemExitCalled go channel when it wants us to exit
	defer time.Sleep(time.Second)                                                                   // sleep a Second at very end to allow everything to finish and clean up.

	// wait until a signal on the done or systemExitCalled go channel variables is received.
	select {
	case <-programShouldQuitSignal.GetSignal():
		log.Println("Quit program channel triggered, exiting.")
		return
	case <-systemExitCalled:
		log.Println("ctrl+c or other system interrupt received, exiting.")
		programShouldQuitSignal.Trigger() // tell the go subroutines to exit by closing the programShouldQuitSignal channel
		return
	}
}

// func scheduleWrite(pipe *NamedPipeRelay) {
// 	for {
// 		<-time.After(time.Second * 1)
// 		print(".")
// 		pipe.SendMessagesToPipe <- "Hello" + time.Now().String()
// 	}
// }

// func readLoop(pipe *NamedPipeRelay) {
// 	for {
// 		message := <-pipe.GetMessagesFromPipe
// 		pipe.log.Info("Message from pipe: ", message)
// 	}
// }
