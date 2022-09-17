package util

import "sync"

// EventSub is a simple fan-out event subscription system using go channels.
// based on PubSub from https://eli.thegreenplace.net/2020/pubsub-using-channels-in-go/

type EventSub struct {
	mu     sync.RWMutex
	subs   []chan interface{}
	closed bool
}

func (es *EventSub) Subscribe() <-chan interface{} {
	es.mu.Lock()
	defer es.mu.Unlock()

	ch := make(chan interface{}, 1)
	es.subs = append(es.subs, ch)
	return ch
}

func (es *EventSub) UnSubscribe(c *chan interface{}) {
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

func (es *EventSub) Publish(data interface{}) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	if es.closed {
		return
	}

	for _, ch := range es.subs {
		ch <- data
	}
}

func (es *EventSub) Close() {
	es.mu.Lock()
	defer es.mu.Unlock()

	if !es.closed {
		es.closed = true
		for _, ch := range es.subs {
			close(ch)
		}
	}
}
