package media

// DEPRICATED

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/kw-m/webrtc-relay/pkg/namedpipe"
	"github.com/stretchr/testify/assert"
)

// func Test_Interceptor_BindUnbind(t *testing.T) {
// 	var (
// 		cntBindRTCPReader     uint32
// 		cntBindRTCPWriter     uint32
// 		cntBindLocalStream    uint32
// 		cntUnbindLocalStream  uint32
// 		cntBindRemoteStream   uint32
// 		cntUnbindRemoteStream uint32
// 		cntClose              uint32
// 	)
// 	mockInterceptor := &interceptor.mock_interceptor{
// 		BindRTCPReaderFn: func(reader interceptor.RTCPReader) interceptor.RTCPReader {
// 			atomic.AddUint32(&cntBindRTCPReader, 1)
// 			return reader
// 		},
// 		BindRTCPWriterFn: func(writer interceptor.RTCPWriter) interceptor.RTCPWriter {
// 			atomic.AddUint32(&cntBindRTCPWriter, 1)
// 			return writer
// 		},
// 		BindLocalStreamFn: func(i *interceptor.StreamInfo, writer interceptor.RTPWriter) interceptor.RTPWriter {
// 			atomic.AddUint32(&cntBindLocalStream, 1)
// 			return writer
// 		},
// 		UnbindLocalStreamFn: func(i *interceptor.StreamInfo) {
// 			atomic.AddUint32(&cntUnbindLocalStream, 1)
// 		},
// 		BindRemoteStreamFn: func(i *interceptor.StreamInfo, reader interceptor.RTPReader) interceptor.RTPReader {
// 			atomic.AddUint32(&cntBindRemoteStream, 1)
// 			return reader
// 		},
// 		UnbindRemoteStreamFn: func(i *interceptor.StreamInfo) {
// 			atomic.AddUint32(&cntUnbindRemoteStream, 1)
// 		},
// 		CloseFn: func() error {
// 			atomic.AddUint32(&cntClose, 1)
// 			return nil
// 		},
// 	}
// 	ir := &interceptor.Registry{}
// 	ir.Add(&mock_interceptor.Factory{
// 		NewInterceptorFn: func(_ string) (interceptor.Interceptor, error) { return mockInterceptor, nil },
// 	})

// 	sender, receiver, err := NewAPI(WithMediaEngine(m), WithInterceptorRegistry(ir)).newPair(Configuration{})
// 	assert.NoError(t, err)
// }

func TestNamedPipeMediaSource(t *testing.T) {
	pipeFilePath := "./namedPipeMediaSourceTest.pipe"

	// create the media source named pipe to recive the fake data:
	mediaSrc, err := CreateNamedPipeMediaSource(pipeFilePath, 1024, time.Millisecond*33, "video/unknown", "testMediaTrack")
	assert.NoError(t, err)
	go mediaSrc.StartMediaStream()

	// create the named pipe to write fake data too:
	mediaGeneratorPipe, err := namedpipe.CreateNamedPipeRelay(pipeFilePath, 0666, os.O_WRONLY, 0)
	assert.NoError(t, err)
	go mediaGeneratorPipe.RunPipeLoops()

	go func() {
		<-time.After(time.Millisecond * 10)
		var i int = 0
		for i < 1000 {
			mediaGeneratorPipe.SendBytesToPipe([]byte(fmt.Sprint(i % 9)))
		}
	}()

	time.Sleep(time.Millisecond * 100)
	// var i int = 0

	mediaGeneratorPipe.Close()
	mediaSrc.Close()
	os.Remove(pipeFilePath)
}
