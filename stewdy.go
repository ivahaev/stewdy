//go:generate stringer -type=Event
package stewdy

import (
	"fmt"
	"sync"
)

var (
	eventHandlers       = map[Event][]EventHandler{}
	eventHandlersLocker sync.RWMutex
)

// EventHandler is a hook func to handle target's events.
type EventHandler func(t Target)

// Event is a type for target's events types.
type Event int

const (
	Originate Event = iota + 1
	Answer
	Connect
	Fail
)

func On(e Event, f EventHandler) {
	eventHandlersLocker.Lock()
	defer eventHandlersLocker.Unlock()

	hs, ok := eventHandlers[e]
	if !ok {
		hs = []EventHandler{}
	}

	f1 := fmt.Sprintf("%p", f)
	for _, v := range hs {
		f2 := fmt.Sprintf("%p", v)
		if f1 == f2 {
			return
		}
	}

	eventHandlers[e] = append(hs, f)
}

func emit(e Event, t Target) {
	eventHandlersLocker.RLock()
	defer eventHandlersLocker.RUnlock()

	for _, h := range eventHandlers[e] {
		go h(t)
	}
}

func Setup() {

}

func Status() {}
