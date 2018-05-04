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

// Events constants
const (
	EventOriginate TargetEvent = iota + 1
	EventAnswer
	EventConnect
	EventFail
	EventSuccess
	EventHangup
	EventRemove
)

// Answered marks target as answered and sets uniqueID field.
// Returns error when target is not found.
func Answered(targetID, uniqueID string) error {
	return answered(targetID, uniqueID)
}

// Connected marks target with provided uniqueID as connected with operator and sets operatorID field.
// Also emits EventSuccess event with found target.
// Returns error when target is not found.
func Connected(uniqueID, operatorID string) error {
	return connected(uniqueID, operatorID)
}

// Failed marks target as failed. If no attempts remained, will remove target from campaign's target list.
// Returns error when target is not found.
func Failed(targetID string) error {
	return failed(targetID)
}

// Hanguped marks target with uniqueID provided as hanguped. If
// Returns error when target is not found.
func Hanguped(uniqueID string) error {
	return hanguped(uniqueID)
}

// Len returns targets count for campaign.
// It counts targets in target list, plus originating and answered targets.
// Connected targets are not included.
// Returns error if campaign is not found.
func Len(campaignID string) (int, error) {
	return targetsLen(campaignID)
}

// On registers event handler for provided e.
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

func emitSync(e TargetEvent, t Target) {
	eventHandlersLocker.RLock()
	defer eventHandlersLocker.RUnlock()

	for _, h := range eventHandlers[e] {
		h(t)
	}
}
