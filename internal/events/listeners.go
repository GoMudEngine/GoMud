package events

import (
	"runtime/debug"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

type ListenerReturn int8

type ListenerId uint64

type ListenerWrapper struct {
	id       ListenerId
	listener Listener
	isFinal  bool
}

// Return false to stop further handling of this event.
type Listener func(Event) ListenerReturn
type QueueFlag int

var (
	listenerLock = sync.RWMutex{}
	// listeners that want to handle an event first.
	listenerCt          ListenerId = 0
	eventListeners      map[string][]ListenerWrapper
	hasWildcardListener bool = false

	eventsWithoutListeners map[string]int = map[string]int{}
)

const (
	NoListenerSampleSize = 20

	First QueueFlag = 1
	Last  QueueFlag = 2
	//
	// Event return codes
	//
	// Allows the event to continu to the next listener
	Continue ListenerReturn = 0b00000001
	// Cancels any further processing of the event
	Cancel ListenerReturn = 0b00000010
	// Cancels processing, but adds back into the queue for the next event loop.
	CancelAndRequeue ListenerReturn = 0b00000100
)

func ClearListeners() {
	listenerLock.Lock()
	defer listenerLock.Unlock()
	eventListeners = map[string][]ListenerWrapper{}
}

// Returns an ID for the listener which can be used to unregister later.
func RegisterListener(emptyEvent any, cbFunc Listener, qFlag ...QueueFlag) ListenerId {
	listenerLock.Lock()
	defer listenerLock.Unlock()

	if eventListeners == nil {
		eventListeners = map[string][]ListenerWrapper{}
	}

	listenerCt++

	eType := `*`

	if emptyEvent != nil {
		if evt, ok := emptyEvent.(Event); ok {
			eType = evt.Type()
		} else if evtString, ok := emptyEvent.(string); ok {
			eType = evtString
		}
	}

	if _, ok := eventListeners[eType]; !ok {
		eventListeners[eType] = []ListenerWrapper{}
	}

	listenerDetails := ListenerWrapper{
		id:       listenerCt,
		listener: cbFunc,
		isFinal:  len(qFlag) > 0 && qFlag[0] == Last,
	}

	frontOfQueue := len(qFlag) > 0 && qFlag[0] == First

	if frontOfQueue {
		eventListeners[eType] = append([]ListenerWrapper{listenerDetails}, eventListeners[eType]...)

	} else if listenerDetails.isFinal {
		eventListeners[eType] = append(eventListeners[eType], listenerDetails)

	} else { // end of the list, but before any "final" listeners

		insertPosition := 0

		for idx := len(eventListeners[eType]) - 1; idx >= 0; idx-- {
			// If we're looking at a "final" listener, we can't go any farther down the list
			if !eventListeners[eType][idx].isFinal {
				insertPosition = idx + 1
				break
			}
		}

		eventListeners[eType] = append(eventListeners[eType], ListenerWrapper{})
		copy(eventListeners[eType][insertPosition+1:], eventListeners[eType][insertPosition:])
		eventListeners[eType][insertPosition] = listenerDetails
	}

	// Write it to debug out
	//mudlog.Debug("Listener Registered", "Event", eType, "Function", runtime.FuncForPC(reflect.ValueOf(cbFunc).Pointer()).Name())

	if eType == `*` {
		hasWildcardListener = true
	}

	return listenerCt
}

// Returns true if listener found and removed.
func UnregisterListener(emptyEvent Event, id ListenerId) bool {

	listenerLock.Lock()
	defer listenerLock.Unlock()

	eType := `*`
	if emptyEvent != nil {
		eType = emptyEvent.Type()
	}

	if vals, ok := eventListeners[eType]; ok {

		for idx, wrapper := range vals {
			if wrapper.id == id {
				vals = append(vals[:idx], vals[idx+1:]...)
				eventListeners[eType] = vals
				return true
			}
		}
	}

	if eType == `*` {
		hasWildcardListener = len(eventListeners[eType]) > 0
	}

	return false

}

// invokeListenerSafely runs a single listener with panic recovery, so one
// misbehaving handler cannot take down the process.
//
// DoListeners is the sole dispatch point for combat rounds, quest events,
// command execution and mob AI — a surface spanning ~150 files — and neither it
// nor anything above it (ProcessEvents, EventLoop, MainWorker) recovered. A nil
// dereference or failed type assertion anywhere in that surface killed the
// server and disconnected every player.
//
// A panicking listener is treated as Continue: the event proceeds to the
// remaining listeners rather than being silently cancelled, so one broken
// handler degrades to "that handler did nothing this round".
//
// This mirrors the recovery already applied to individual callbacks elsewhere
// (behaviortree.invokePlannerSafely, goals.invokeContextScore).
func invokeListenerSafely(lw ListenerWrapper, e Event) (ret ListenerReturn) {

	defer func() {
		if r := recover(); r != nil {
			mudlog.Error(`DoListeners`,
				`error`, `listener panicked`,
				`event`, e.Type(),
				`panic`, r,
				`stack`, string(debug.Stack()))
			ret = Continue
		}
	}()

	return lw.listener(e)
}

func DoListeners(e Event) ListenerReturn {

	listenerLock.Lock()
	defer listenerLock.Unlock()

	if len(eventListeners) == 0 {
		return Continue
	}

	listenerFound := false
	// wildcard listener is really for debugging purpose
	if hasWildcardListener {
		if vals, ok := eventListeners[`*`]; ok {
			listenerFound = true
			for _, lw := range vals {
				if result := invokeListenerSafely(lw, e); result != Continue {
					return result
				}
			}
		}

	}

	if vals, ok := eventListeners[e.Type()]; ok {
		listenerFound = true
		for _, lw := range vals {
			if result := invokeListenerSafely(lw, e); result != Continue {
				return result
			}
		}
	}

	if !listenerFound {
		t := e.Type()
		eventsWithoutListeners[t] = eventsWithoutListeners[t] + 1
		if eventsWithoutListeners[t]%NoListenerSampleSize == 0 {
			mudlog.Error(`DoListeners`, "Event", t, "error", "no listener for event", "sample-size", NoListenerSampleSize)
		}
	}

	return Continue
}
