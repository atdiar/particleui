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

type nativeEventObject struct{
	Event
}

func(nativeEventObject) DispatchNative(){}

func MakeDispatchNative(e Event) nativeEventObject{
	return nativeEventObject{e}
}

// DispatchNative is the interface implemented by events that should be dispatched on the native
// platform.
type DispatchNative interface{
	DispatchNative()
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
	list:= e.List[:0]
	for _, v := range e.List {
		if v != h {
			list = append(list,v)
			index++
		}
	}

	for i:= index; i<len(e.List); i++{ // cleanup to avoid dangling pointer
		e.List[i] = nil
	}
	e.List = list[:index]
	return e
}

type EventHandler struct {
	Fn      func(Event) bool
	Capture bool // propagation mode: if false bubbles up, otherwise captured by the top most element and propagates down.
	
	Once bool
	Bubble bool
}

func (e EventHandler) Handle(evt Event) bool {
	if !e.Bubble{
		if evt.Target() == nil || evt.CurrentTarget().ID != evt.Target().ID{
			return false
		}
	}
	return e.Fn(evt)
}

func NewEventHandler(fn func(Event) bool) *EventHandler {
	return &EventHandler{fn, false, false, true}
}
func (e *EventHandler) ForCapture() *EventHandler {
	if e.Capture {
		return e
	}
	n:= NewEventHandler(e.Fn)
	n.Capture = true

	if e.Once{
		n.Once = true
	}

	if !e.Bubble{
		n.Bubble = false
	}

	return n
}

func (e *EventHandler) TriggerOnce() *EventHandler {
	if e.Once{
		return e
	}
	n:= NewEventHandler(e.Fn)
	n.Capture = true

	if e.Capture{
		n.Capture = true
	}

	if !e.Bubble{
		n.Bubble = false
	}

	return n
}

// NoBubble disallows the handling of bubbling events. Essentially, if the event target is not the current target
// the event handler on the current target does not run. (the event probably got triggered on a child element)
func(e *EventHandler) NoBubble() *EventHandler{
	if !e.Bubble{
		return e
	}
	n:= NewEventHandler(e.Fn)
	n.Capture = true

	if e.Capture{
		n.Capture = true
	}

	if e.Once{
		n.Once = true
	}

	n.Bubble = false
	return n
}


// Transition  events
// These events can be composed since it is merely function composition of MutationHandler callbacks
// One would typically create a new transition event and manage the different subtransitions within
// the transitionstart mutation event handler.

// DefineTransition defines a long-form even that can go through different phases (start to end,
// possibly cause by an error or a cancellation).
func(e *Element) DefineTransition(name string, onstart, onerror, oncancel, onend *MutationHandler){
	e.WatchEvent(transition(name, "start"), e, NewMutationHandler(func(evt MutationEvent)bool{
		// Once the event starts, the element  should watch out for cancellation
		/*_,ok:= e.Get("event",transition(name,"cancel"))
		if ok{
			 e.Properties.Delete("event",transition(name, "cancel"))
		}
		*/

		e.CancelTransition(name)

		if onerror != nil{
			onerror = onerror.RunOnce()
			evt.Origin().OnTransitionError(name, onerror)
		}
		


		if oncancel != nil{
			oncancel = oncancel.RunOnce()
			evt.Origin().OnTransitionCancel(name, oncancel)
		}
		
		if onend != nil{
			onend = onend.RunOnce()
			evt.Origin().OnTransitionError(name, onend)
		}
	

		cancelall:= NewMutationHandler(func(evt MutationEvent)bool{
			e.TriggerEvent(transition(name, "cancel"))
			return false
		}).RunOnce()

		// After the transition start, upon failure, the element should be able to trigger the transition end.
		evt.Origin().OnTransitionError(name,NewMutationHandler(func(ev MutationEvent)bool{
			ev.Origin().TriggerEvent(transition(name, "end"),String("error"))
			return false
		}).RunOnce())

		// After the transition start, upon cancellation, the element should be able to trigger the transition end.
		e.OnTransitionCancel(name,NewMutationHandler(func(evt MutationEvent)bool{
			// TODO check that it does not create problems
			evt.Origin().TriggerEvent(transition(name, "end"),String("cancelled"))
			return false
		}).RunOnce())

		e.WatchEvent("cancelalltransitions", e, cancelall)

		e.WatchEvent(transition(name, "end"), e, NewMutationHandler(func(evt MutationEvent)bool{
			evt.Origin().Unwatch("event","cancelalltransitions", e)
			evt.Origin().Unwatch("event",transition(name,"cancel"), e)
			evt.Origin().Unwatch("event",transition(name,"error"), e)
			e.WatchEvent(transition(name,"end"),e,NewMutationHandler(func(evt MutationEvent)bool{
				evt.Origin().Unwatch("event",transition(name,"end"), e)
				return false
			}).RunOnce())
			return false
		}).RunOnce())

		return onstart.Handle(evt)
	}))
}

func(e *Element) OnTransitionStart(name string, h *MutationHandler){
	e.WatchEvent(transition(name, "start"), e, NewMutationHandler(func(evt MutationEvent)bool{
		evt.Origin().WatchEvent(transition(name, "start"), evt.Origin(), h.RunOnce())
		return false
	}))
}

func(e *Element) OnTransitionError(name string, h *MutationHandler){
	e.WatchEvent(transition(name, "start"), e, NewMutationHandler(func(evt MutationEvent)bool{
		evt.Origin().WatchEvent(transition(name, "error"), evt.Origin(), h.RunOnce())
		return false
	}))
}

func(e *Element) OnTransitionCancel(name string, h *MutationHandler){
	e.WatchEvent(transition(name, "start"), e, NewMutationHandler(func(evt MutationEvent)bool{
		evt.Origin().WatchEvent(transition(name, "cancel"), evt.Origin(), h.RunOnce())
		return false
	}))
}

func(e *Element) OnTransitionEnd(name string, h *MutationHandler){
	e.WatchEvent(transition(name, "start"), e, NewMutationHandler(func(evt MutationEvent)bool{
		evt.Origin().WatchEvent(transition(name, "end"), evt.Origin(), h.RunOnce())
		return false
	}))
}

func(e *Element) StartTransition(name string,values ...Value){
	e.TriggerEvent(transition(name,"start"),values...)
}

func(e *Element) ErrorTransition(name string, values ...Value){
	e.TriggerEvent(transition(name,"error"),values...)
}

func(e *Element) CancelTransition(name string,values ...Value){
	e.TriggerEvent(transition(name, "cancel"), values...)
}

func (e *Element) CancelAllTransitions(){
	e.TriggerEvent("cancelalltransitions")
}

func(e *Element) EndTransition(name string, values ...Value){
	e.TriggerEvent(transition(name,"end"),values...)
}

func transitionCancelled(e *Element, transitionname string) bool{
	v,ok:= e.Get("event",transition(transitionname, "end"))
	if !ok{
		return false
	}
	vv,ok:= v.(String)
	if !ok{
		return false
	}
	return vv.String() == "cancelled"
}


// NewTransitionChain creates a new transition chain. The transition chain is a sequence of 
// transitions that are triggered synchronously.
func(e *Element) NewTransitionChain(name string, transitionevents ...string) func(onstart, onerror, oncancel, onend *MutationHandler){


	h:= NewMutationHandler(func(evt MutationEvent)bool{
		// When a transition ends, the next one should start unless cancellation was triggered for one of the transitions
		// in the chain.
		l:= len(transitionevents)-1
		for i,t:= range transitionevents{
			if  i == 0{
				e.OnTransitionEnd(name, NewMutationHandler(func(evt MutationEvent)bool{
					// check cancellation status first
					if transitionCancelled(e, name){
						return false
					}
					e.TriggerEvent(transition(t, "start"))
					return false
				}))
			}

			if 0 < i &&  i < l{
				e.OnTransitionEnd(t, NewMutationHandler(func(evt MutationEvent)bool{
					// check cancellation status first
					if transitionCancelled(e, t){
						return false
					}
					
					
					e.TriggerEvent(transition(transitionevents[i+1], "start"))
					return false
				}))
			} 
			if i == l {
				e.OnTransitionEnd(t, NewMutationHandler(func(evt MutationEvent)bool{
					// check cancellation status first
					if transitionCancelled(e, t){
						return false
					}

					e.TriggerEvent(transition(name, "end"))
					return false
				}))
			}

			e.OnTransitionCancel(name, NewMutationHandler(func(evt MutationEvent)bool{
				e.TriggerEvent(transition(t, "cancel"))
				return false
			}))
		}
		if len(transitionevents) > 0{
			e.TriggerEvent(transition(transitionevents[0], "start"))
		}

		return false
	})


	return func(onstart, onerror, oncancel, onend *MutationHandler){
		g:= NewMutationHandler(func(evt MutationEvent)bool{
			if onstart != nil{
				b:= onstart.Handle(evt)
				if b {
					return true
				}
			}
			return h.Handle(evt)
		})
		e.DefineTransition(name, g, onerror, oncancel, onend)
	}
	
}

func transition(name string, phase string) string{
	return "tr-"+ name + "-" + phase
}