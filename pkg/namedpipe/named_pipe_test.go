package namedpipe

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNamedPipeRelay(t *testing.T) {
	pipeFilePath := "./namedPipeTest.pipe"

	// create the named pipe to write to
	pipeIn, err := CreateNamedPipeRelay(pipeFilePath, 0666, os.O_WRONLY, 0)
	assert.NoError(t, err)
	go pipeIn.RunPipeLoops()

	// open the same pipe to read from
	pipeOut, err := CreateNamedPipeRelay(pipeFilePath, 0666, os.O_RDONLY, 0)
	assert.NoError(t, err)
	go pipeOut.RunPipeLoops()

	go func() {
		<-time.After(time.Millisecond * 100)
		var i int = 0
		for i < 10 {
			println("Sending: ", i)
			pipeIn.SendMessageToPipe(fmt.Sprint(i))
			i++
			<-time.After(time.Millisecond * 100)
		}
	}()

	var i int = 0
	for i < 10 {
		select {
		case msg := <-pipeOut.MessagesFromPipeChannel:
			println("gotMsg:", msg)
			msgInt, err := strconv.Atoi(msg)
			assert.NoError(t, err)
			assert.Equal(t, i, msgInt)
		case <-time.After(time.Millisecond * 200):
			assert.Fail(t, "read timeout")
		}
		i++
	}

	pipeOut.Close()
	pipeIn.Close()
	os.Remove(pipeFilePath)
}

func TestDuplexNamedPipeRelay(t *testing.T) {
	pipeFilePath := "./namedPipeDuplexTest.pipe"

	// create the named pipe to write to
	duplexPipe, err := CreateDuplexNamedPipeRelay(pipeFilePath, pipeFilePath, 0666, 0)
	assert.NoError(t, err)
	go duplexPipe.RunPipeLoops()

	go func() {
		var i int = 0
		for i < 10 {
			select {
			case msg := <-duplexPipe.MessagesFromPipeChannel:
				println("gotMsg:", msg)
				msgInt, err := strconv.Atoi(msg)
				assert.NoError(t, err)
				assert.Equal(t, i, msgInt)
			case <-time.After(time.Millisecond * 200):
				assert.Fail(t, "read timeout")
			}
			i++
		}
	}()

	<-time.After(time.Millisecond * 10)
	var i int = 0
	for i < 10 {
		duplexPipe.SendMessageToPipe(fmt.Sprint(i))
		i++
		<-time.After(time.Millisecond * 100)
	}

	duplexPipe.Close()
	os.Remove(pipeFilePath)
}

func TestDuplexNamedPipeRelay2(t *testing.T) {
	outgoingPipeFilePath := "./namedPipeDuplexTestOutgoing.pipe"
	incomingPipeFilePath := "./namedPipeDuplexIncoming.pipe"

	// create the named pipe to write to
	duplexPipe, err := CreateDuplexNamedPipeRelay(incomingPipeFilePath, outgoingPipeFilePath, 0666, 0)
	assert.NoError(t, err)
	go duplexPipe.RunPipeLoops()

	go func() {
		var i int = 0
		for i < 10 {
			select {
			case msg := <-duplexPipe.MessagesFromPipeChannel:
				println("gotMsg:", msg)
				msgInt, err := strconv.Atoi(msg)
				assert.NoError(t, err)
				assert.Equal(t, i, msgInt)
			case <-time.After(time.Millisecond * 200):
				assert.Fail(t, "read timeout")
			}
			i++
		}
	}()

	<-time.After(time.Millisecond * 10)
	var i int = 0
	for i < 10 {
		duplexPipe.SendMessageToPipe(fmt.Sprint(i))
		i++
		<-time.After(time.Millisecond * 100)
	}

	duplexPipe.Close()
	os.Remove(outgoingPipeFilePath)
	os.Remove(incomingPipeFilePath)
}
