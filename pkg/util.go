package webrtc_relay

/* contains
 * checks if a string is present in a slice (aka an array)
 * PARAM s: the list/slice of strings to check
 * PARAM str: the string to check for
 * RETURNS: true if the string is present in the slice, false otherwise
 * from: https://freshman.tech/snippets/go/check-if-slice-contains-element/
 */
func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

/* removeMatching (UNTESTED)
 * removes all elements from a slice that match elements from another slice
 * PARAM a: the slice to remove elements from
 * PARAM b: the slice of match elements to remove from a
 * RETURNS: the slice a with all matching elements removed
 */
func removeMatching(a []string, b []string) []string {
	newMap := map[string]bool{}

	// mask the elements in b with a map
	for _, value := range b {
		newMap[value] = true
	}

	//
	output := make([]string, 0)
	for _, value := range a {
		if _, exists := newMap[value]; !exists {
			output = append(output, value)
		}
	}

	return output
}

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
