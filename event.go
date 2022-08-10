// Package ui is a library of functions for simple, generic gui development.
package ui

type Event interface {
	Type() string
	Target() *Element
	CurrentTarget() *Element
	Value() Value // TODO make this a Value type (DEBUG)

	PreventDefault()
	StopPropagation()          // the phase is stil 1,2,or 3 but Stopped returns true
	StopImmediatePropagation() // sets the Phase to 0 and Stopped to true
	SetPhase(int)
	SetCurrentTarget(*Element)

	Phase() int
	Bubbles() bool
	DefaultPrevented() bool
	Cancelable() bool
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
	cancelable       bool
	phase            int

	nativeObject interface{}
	value        Value
}

type defaultPreventer interface {
	PreventDefault()
}

type propagationStopper interface{
	StopPropagation()
}

type propagationImmediateStopper interface{
	StopImmediatePropagation()
}

func (e *eventObject) Type() string            { return e.typ }
func (e *eventObject) Target() *Element        { return e.target }
func (e *eventObject) CurrentTarget() *Element { return e.currentTarget }
func (e *eventObject) PreventDefault() {
	if !e.Cancelable() {
		return
	}
	if v, ok := e.nativeObject.(defaultPreventer); ok {
		v.PreventDefault()
	}
	e.defaultPrevented = true
}
func (e *eventObject) StopPropagation() {
	if v, ok := e.nativeObject.(propagationStopper); ok {
		v.StopPropagation()
	} 
	e.stopped = true 
}
func (e *eventObject) StopImmediatePropagation() {
	if v, ok := e.nativeObject.(propagationImmediateStopper); ok {
		v.StopImmediatePropagation()
	} 
	e.stopped = true
	e.phase = 0
}
func (e *eventObject) SetPhase(i int)              { e.phase = i }
func (e *eventObject) SetCurrentTarget(t *Element) { e.currentTarget = t }
func (e *eventObject) Phase() int                  { return e.phase }
func (e *eventObject) Bubbles() bool               { return e.bubbles }
func (e *eventObject) DefaultPrevented() bool      { return e.defaultPrevented }
func (e *eventObject) Stopped() bool               { return e.stopped }
func (e *eventObject) Cancelable() bool            { return e.cancelable }
func (e *eventObject) Native() interface{}         { return e.nativeObject }
func (e *eventObject) Value() Value               { return e.value }

func NewEvent(typ string, bubbles bool, cancelable bool, target *Element, currentTarget *Element, nativeEvent interface{}, value Value) Event {
	return &eventObject{typ, target, currentTarget, false, bubbles, false, cancelable, 0, nativeEvent, value}
}

type EventListeners struct {
	list map[string]*eventHandlers
}

func NewEventListenerStore() EventListeners {
	return EventListeners{make(map[string]*eventHandlers, 0)}
}

func (e EventListeners) AddEventHandler(event string, handler *EventHandler) {
	eh, ok := e.list[event]
	if !ok {
		eh = newEventHandlers()
		e.list[event] = eh
	}
	eh.Add(handler)
}

func (e EventListeners) RemoveEventHandler(event string, handler *EventHandler) {
	eh, ok := e.list[event]
	if !ok {
		return
	}
	eh.Remove(handler)
}

func (e EventListeners) Handle(evt Event) bool {
	evh, ok := e.list[evt.Type()]
	if !ok {
		return false
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
		return false
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
		return false
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
		return false
	}
	return false
}

type eventHandlers struct {
	List []*EventHandler
}

func newEventHandlers() *eventHandlers {
	return &eventHandlers{make([]*EventHandler, 0, 20)}
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
	Capture bool // propagation mode: if false bubbles up, otherwise captured by the top most element and propagates down.
	
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

