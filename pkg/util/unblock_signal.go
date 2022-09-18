package util

/* UnblockSignal
 * simple wrapper around a go channel to make it easier to block a goroutine from continuing and then let it continue when Trigger() is called
 * EXAMPLE USE: exiting a goroutine when some error happens in another goroutine
 */
type UnblockSignal struct {
	err          error // could be used to pass an error back to the blocked goroutine
	exitSignal   chan bool
	HasTriggered bool
}

func NewUnblockSignal() *UnblockSignal {
	p := UnblockSignal{exitSignal: make(chan bool), HasTriggered: false, err: nil}
	return &p
}

func (e *UnblockSignal) Trigger() {
	if !e.HasTriggered {
		e.HasTriggered = true
		close(e.exitSignal)
	}
}

func (e *UnblockSignal) TriggerWithError(err error) {
	if !e.HasTriggered {
		e.err = err
		e.HasTriggered = true
		close(e.exitSignal)
	}
}

func (e *UnblockSignal) Wait() error {
	<-e.exitSignal
	return e.err
}

func (e *UnblockSignal) GetSignal() chan bool {
	return e.exitSignal
}

func (e *UnblockSignal) GetError() error {
	return e.err
}
