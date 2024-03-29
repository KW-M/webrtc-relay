package namedpipe

// DEPRECATED

// import (
// 	"bufio"
// 	"errors"
// 	"io/fs"
// 	"os"
// 	"syscall"
// 	"time"

// 	util "github.com/kw-m/webrtc-relay/pkg/util"
// 	log "github.com/sirupsen/logrus"
// )

// type NamedPipeRelay struct {
// 	pipeFile                *os.File
// 	pipeFilePath            string
// 	pipeFilePermissions     uint32
// 	pipeFileOpenMode        int
// 	MessagesFromPipeChannel chan string
// 	readChannelBufferCount  int
// 	lastErr                 error
// 	exitSignal              util.UnblockSignal
// 	log                     *log.Entry
// }

// func CreateNamedPipeRelay(pipeFilePath string, pipeFilePermissions uint32, pipeOpenMode int, readChannelBufferCount int) (*NamedPipeRelay, error) {
// 	var pipe = NamedPipeRelay{
// 		pipeFile:                nil,
// 		pipeFilePath:            pipeFilePath,
// 		pipeFilePermissions:     pipeFilePermissions,
// 		pipeFileOpenMode:        pipeOpenMode,
// 		exitSignal:              util.NewUnblockSignal(),
// 		MessagesFromPipeChannel: make(chan string, readChannelBufferCount),
// 		readChannelBufferCount:  readChannelBufferCount,
// 		log:                     log.WithFields(log.Fields{"pipe": pipeFilePath, "fileOpenMode": pipeOpenMode}),
// 		lastErr:                 nil,
// 	}

// 	// attempt to create the named pipe file if doesn't already exist:
// 	if _, err := os.Stat(pipeFilePath); err != nil {
// 		err := syscall.Mkfifo(pipeFilePath, pipeFilePermissions)
// 		if err != nil {
// 			pipe.log.Error("Create named pipe file error:", err.Error())
// 			return nil, err
// 		}
// 	}

// 	return &pipe, nil
// }

// func (pipe *NamedPipeRelay) GetLastError() error {
// 	err := pipe.lastErr
// 	pipe.lastErr = nil
// 	return err
// }

// func (pipe *NamedPipeRelay) Close() {
// 	if pipe.pipeFile != nil {
// 		pipe.pipeFile.Close()
// 		pipe.pipeFile = nil
// 	}
// 	pipe.exitSignal.Trigger()
// }

// func (pipe *NamedPipeRelay) SendBytesToPipe(bytes []byte) {
// 	if pipe.pipeFile != nil {
// 		_, err := pipe.pipeFile.Write(bytes)
// 		if err != nil {
// 			pipe.lastErr = err
// 			pipe.log.Error("Error writing bytes to pipe:", err.Error())
// 		}
// 	} else {
// 		pipe.log.Error("SendBytesToPipe() called when pipe file was not open or not writable")
// 		pipe.lastErr = errors.New("ErrPipeNotReady")
// 	}
// }

// func (pipe *NamedPipeRelay) SendMessageToPipe(msg string) {
// 	if pipe.pipeFile != nil {
// 		_, err := pipe.pipeFile.WriteString(msg + "\n")
// 		if err != nil {
// 			pipe.lastErr = err
// 			pipe.log.Error("Error writing message to pipe:", err.Error())
// 		}
// 	} else {
// 		pipe.log.Error("SendMessageToPipe() called when pipe file was not open or not writable")
// 		pipe.lastErr = errors.New("ErrPipeNotReady")
// 	}
// }

// func (pipe *NamedPipeRelay) RunPipeLoops() error {
// 	defer pipe.Close()
// openloop:
// 	for {

// 		var err error
// 		//https://medium.com/@cpuguy83/non-blocking-i-o-in-go-bc4651e3ac8d
// 		pipe.pipeFile, err = os.OpenFile(pipe.pipeFilePath, os.O_RDWR|syscall.O_CLOEXEC|syscall.O_NONBLOCK, os.ModeNamedPipe|fs.FileMode(pipe.pipeFilePermissions))
// 		if err != nil {
// 			pipe.log.Error("Error opening named pipe:", err.Error())
// 			<-time.After(time.Second)
// 			continue
// 		}
// 		pipe.log.Debug("Pipe file open: ", pipe.pipeFilePath)

// 		if pipe.pipeFileOpenMode == os.O_RDONLY || pipe.pipeFileOpenMode == os.O_RDWR {
// 			// read messages from pipe loop
// 			scanner := bufio.NewScanner(pipe.pipeFile)
// 			for scanner.Scan() {
// 				if pipe.exitSignal.HasTriggered {
// 					pipe.log.Debug("pipe exit: ")
// 					return nil
// 				} else if err := scanner.Err(); err != nil {
// 					pipe.log.Printf("Error reading message from pipe: %v", err.Error())
// 					pipe.lastErr = err
// 					continue openloop
// 				}
// 				msg := scanner.Text()
// 				pipe.log.Debug("Message received from pipe: ", msg)
// 				pipe.MessagesFromPipeChannel <- msg
// 			}
// 		}

// 		pipe.exitSignal.Wait()
// 		return nil
// 	}
// }

// type DuplexNamedPipeRelay struct {
// 	incomingPipe            *NamedPipeRelay
// 	outgoingPipe            *NamedPipeRelay
// 	MessagesFromPipeChannel chan string
// 	log                     *log.Entry
// 	exitSignal              util.UnblockSignal
// }

// func CreateDuplexNamedPipeRelay(incomingPipeFilePath string, outgoingPipeFilePath string, pipeFilePermissions uint32, readChannelBufferCount int) (*DuplexNamedPipeRelay, error) {
// 	var duplexPipe = DuplexNamedPipeRelay{
// 		exitSignal: util.NewUnblockSignal(),
// 		log:        log.WithField("mod", "webrtc_relay/duplex_pipe_pair"),
// 	}

// 	var err error
// 	duplexPipe.incomingPipe, err = CreateNamedPipeRelay(incomingPipeFilePath, pipeFilePermissions, os.O_RDONLY, readChannelBufferCount)
// 	if err != nil {
// 		duplexPipe.log.Error("Error creating incoming pipe:", err.Error())
// 		return nil, err
// 	}

// 	duplexPipe.outgoingPipe, err = CreateNamedPipeRelay(outgoingPipeFilePath, pipeFilePermissions, os.O_WRONLY, readChannelBufferCount)
// 	if err != nil {
// 		duplexPipe.log.Error("Error creating outgoing pipe:", err.Error())
// 		return nil, err
// 	}

// 	// setup duplex channel forwarding
// 	duplexPipe.MessagesFromPipeChannel = duplexPipe.incomingPipe.MessagesFromPipeChannel

// 	return &duplexPipe, nil
// }

// func (pipe *DuplexNamedPipeRelay) Close() {
// 	pipe.exitSignal.Trigger()
// 	pipe.incomingPipe.Close()
// 	pipe.outgoingPipe.Close()
// }

// func (pipe *DuplexNamedPipeRelay) SendMessageToPipe(msg string) {
// 	pipe.outgoingPipe.SendMessageToPipe(msg)
// }

// func (pipe *DuplexNamedPipeRelay) RunPipeLoops() {
// 	defer pipe.Close()
// 	go pipe.incomingPipe.RunPipeLoops()
// 	go pipe.outgoingPipe.RunPipeLoops()
// 	pipe.exitSignal.Wait()
// }
