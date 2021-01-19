// Package ui is a library of functions for simple, generic gui development.
package ui

type Event interface {
	Type() string
	Target() *Element
	CurrentTarget() *Element

	PreventDefault()
	StopPropagation()          // the phase is stil 1,2,or 3 but Stopped returns true
	StopImmediatePropagation() // sets the Phase to 0 and Stopped to true
	SetPhase(int)
	SetCurrentTarget(*Element)

	Phase() int
	Bubbles() bool
	DefaultPrevented() bool
	Stopped() bool

	Native() interface{} // returns the native event object
}

type eventObject struct {
	typ           string
	target        *Element
	currentTarget *Element

	defaultPrevented bool
	bubbles          bool
	stopped          bool
	phase            int

	nativeObject interface{}
}

type defaultPreventer interface {
	PrventDefault()
}

func (e *eventObject) Type() string            { return e.typ }
func (e *eventObject) Target() *Element        { return e.target }
func (e *eventObject) CurrentTarget() *Element { return e.currentTarget }
func (e *eventObject) PreventDefault() {
	if v, ok := e.nativeObject.(defaultPreventer); ok {
		v.PrventDefault()
	}
	e.defaultPrevented = true
}
func (e *eventObject) StopPropagation() { e.stopped = true }
func (e *eventObject) StopImmediatePropagation() {
	e.stopped = true
	e.phase = 0
}
func (e *eventObject) SetPhase(i int)              { e.phase = i }
func (e *eventObject) SetCurrentTarget(t *Element) { e.currentTarget = t }
func (e *eventObject) Phase() int                  { return e.phase }
func (e *eventObject) Bubbles() bool               { return e.bubbles }
func (e *eventObject) DefaultPrevented() bool      { return e.defaultPrevented }
func (e *eventObject) Stopped() bool               { return e.stopped }
func (e *eventObject) Native() interface{}         { return e.nativeObject }

func NewEvent(typ string, bubbles bool, target *Element, nativeEvent interface{}) Event {
	return &eventObject{typ, target, target, false, bubbles, false, 0, nativeEvent}
}

type EventListeners struct {
	list map[string]*eventHandlers
}

func NewEventListenerStore() EventListeners {
	return EventListeners{make(map[string]*eventHandlers, 0)}
}

func (e EventListeners) AddEventHandler(event Event, handler *EventHandler) {
	eh, ok := e.list[event.Type()]
	if !ok {
		e.list[event.Type()] = newEventHandlers().Add(handler)
	}
	eh.Add(handler)
}

func (e EventListeners) RemoveEventHandler(event Event, handler *EventHandler) {
	eh, ok := e.list[event.Type()]
	if !ok {
		return
	}
	eh.Remove(handler)
}

func (e EventListeners) Handle(evt Event) bool {
	evh, ok := e.list[evt.Type()]
	if !ok {
		return true
	}
	switch evt.Phase() {
	// capture
	case 0:
		return true
	case 1:
		for _, h := range evh.List {
			if !h.Capture {
				continue
			}
			done := h.Handle(evt)
			if h.Once {
				evh.Remove(h)
			}

			if done {
				return done
			}
			if evt.Stopped() && (evt.Phase() == 0) {
				return true
			}
		}
	case 2:
		for _, h := range evh.List {
			done := h.Handle(evt)
			if h.Once {
				evh.Remove(h)
			}
			if done {
				return done
			}
		}
	case 3:
		if !evt.Bubbles() {
			return true
		}
		for _, h := range evh.List {
			if h.Capture {
				continue
			}
			done := h.Handle(evt)
			if h.Once {
				evh.Remove(h)
			}
			if done {
				return done
			}
			if evt.Stopped() && (evt.Phase() == 0) {
				return true
			}
		}
	}
	return false // not supposed to be reached anyway
}

type eventHandlers struct {
	List []*EventHandler
}

func newEventHandlers() *eventHandlers {
	return &eventHandlers{make([]*EventHandler, 0)}
}

func (e *eventHandlers) Add(h *EventHandler) *eventHandlers {
	e.List = append(e.List, h)
	return e
}

func (e *eventHandlers) Remove(h *EventHandler) *eventHandlers {
	var index int
	for k, v := range e.List {
		if v != h {
			continue
		}
		index = k
		break
	}
	if index >= 0 {
		e.List = append(e.List[:index], e.List[index+1:]...)
	}
	return e
}

type EventHandler struct {
	Fn      func(Event) bool
	Capture bool // propagation mode: if false bubbles up, otherwise captured by the top most element and propagate down .

	Once bool
}

func (e EventHandler) Handle(evt Event) bool {
	return e.Fn(evt)
}

func NewEventHandler(fn func(Event) bool) *EventHandler {
	return &EventHandler{fn, false, false}
}
func (e *EventHandler) ForCapture() *EventHandler {
	e.Capture = true
	return e
}

func (e *EventHandler) TriggerOnce() *EventHandler {
	e.Once = true
	return e
}
