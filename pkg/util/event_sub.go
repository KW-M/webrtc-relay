package util

import (
	"sync"
)

// EventSub is a simple fan-out event subscription system using go channels.
// based on PubSub from https://eli.thegreenplace.net/2020/pubsub-using-channels-in-go/
type EventSub[Typ any] struct {
	mu        sync.RWMutex
	subs      []chan *Typ
	closed    bool
	bufferAmt uint
}

// NewEventSub creates a new EventSub struct of the passed type with the given buffer amount
// bufferAmt is the number of messages that can be pushed onto the event sub without subscribers reading the message(s) before the channel blocks (1 or greater is recommended)
func NewEventSub[Typ any](bufferAmt uint) *EventSub[Typ] {
	return &EventSub[Typ]{
		subs:      make([]chan *Typ, 0),
		closed:    false,
		bufferAmt: bufferAmt,
	}
}

func (es *EventSub[Typ]) Subscribe() <-chan *Typ {
	es.mu.Lock()
	defer es.mu.Unlock()

	ch := make(chan *Typ, es.bufferAmt)
	es.subs = append(es.subs, ch)
	return ch
}

func (es *EventSub[Typ]) UnSubscribe(c *chan *Typ) {
	es.mu.Lock()
	defer es.mu.Unlock()

	chanelIndex := -1
	for i, ch := range es.subs {
		if ch == *c {
			chanelIndex = i
			close(ch)
		}
	}
	if chanelIndex != -1 {
		// remove the channel from the list: https://stackoverflow.com/questions/37334119/how-to-delete-an-element-from-a-slice-in-golang
		es.subs[chanelIndex] = es.subs[len(es.subs)-1]
		es.subs = es.subs[:len(es.subs)-1]
	}
}

func (es *EventSub[Typ]) Push(data *Typ) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	if es.closed {
		return
	}

	for _, ch := range es.subs {
		ch <- data
	}
}

func (es *EventSub[Typ]) Close() {
	es.mu.Lock()
	defer es.mu.Unlock()

	if !es.closed {
		es.closed = true
		for _, ch := range es.subs {
			close(ch)
		}
	}
}
