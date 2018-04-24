//go:generate stringer -type=TargetEvent
package stewdy

import (
	"fmt"
	"sync"
)

var (
	eventHandlers       = map[TargetEvent][]EventHandler{}
	eventHandlersLocker sync.RWMutex
)

// EventHandler is a hook func to handle target's events.
type EventHandler func(t Target)

// TargetEvent is a type for target's events types.
type TargetEvent int

const (
	EventOriginate TargetEvent = iota + 1
	EventAnswer
	EventConnect
	EventFail
)

// Answered marks target as answered and sets uniqueID field.
func Answered(campaignID, targetID, uniqueID string) error {
	return answered(campaignID, targetID, uniqueID)
}

func Connected(campaignID, uniqueID, operatorID string) error {
	return connected(campaignID, uniqueID, operatorID)
}

func Failed(campaignID, targetID string) error {
	return failed(campaignID, targetID)
}

func On(e TargetEvent, f EventHandler) {
	eventHandlersLocker.Lock()
	defer eventHandlersLocker.Unlock()

	hs, ok := eventHandlers[e]
	if !ok {
		hs = []EventHandler{}
	}

	// dirty hack to comppare addresses of funcs
	f1 := fmt.Sprintf("%p", f)
	for _, v := range hs {
		f2 := fmt.Sprintf("%p", v)
		if f1 == f2 {
			return
		}
	}

	eventHandlers[e] = append(hs, f)
}

func emit(e TargetEvent, t Target) {
	eventHandlersLocker.RLock()
	defer eventHandlersLocker.RUnlock()

	for _, h := range eventHandlers[e] {
		go h(t)
	}
}

func Setup() {

}

func Status() {}
