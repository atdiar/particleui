// Package ui is a library of functions for simple, generic gui development.
package ui

import (
	"errors"
	"fmt"
	"strings"
)

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

type nativeEventObject struct {
	Event
}

func (nativeEventObject) DispatchNative() {}

func MakeDispatchNative(e Event) nativeEventObject {
	return nativeEventObject{e}
}

// DispatchNative is the interface implemented by events that should be dispatched on the native
// platform.
type DispatchNative interface {
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

type propagationStopper interface {
	StopPropagation()
}

type propagationImmediateStopper interface {
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
func (e *eventObject) Value() Value                { return e.value }

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
	list := e.List[:0]
	for _, v := range e.List {
		if v != h {
			list = append(list, v)
			index++
		}
	}

	for i := index; i < len(e.List); i++ { // cleanup to avoid dangling pointer
		e.List[i] = nil
	}
	e.List = list[:index]
	return e
}

type EventHandler struct {
	Fn      func(Event) bool
	Capture bool // propagation mode: if false bubbles up, otherwise captured by the top most element and propagates down.

	Once   bool
	Bubble bool
}

func (e EventHandler) Handle(evt Event) bool {
	if !e.Bubble {
		if evt.Target() == nil || evt.CurrentTarget().ID != evt.Target().ID {
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
	n := NewEventHandler(e.Fn)
	n.Capture = true

	if e.Once {
		n.Once = true
	}

	if !e.Bubble {
		n.Bubble = false
	}

	return n
}

func (e *EventHandler) TriggerOnce() *EventHandler {
	if e.Once {
		return e
	}
	n := NewEventHandler(e.Fn)
	n.Capture = true

	if e.Capture {
		n.Capture = true
	}

	if !e.Bubble {
		n.Bubble = false
	}

	return n
}

// NoBubble disallows the handling of bubbling events. Essentially, if the event target is not the current target
// the event handler on the current target does not run. (the event probably got triggered on a child element)
func (e *EventHandler) NoBubble() *EventHandler {
	if !e.Bubble {
		return e
	}
	n := NewEventHandler(e.Fn)
	n.Capture = true

	if e.Capture {
		n.Capture = true
	}

	if e.Once {
		n.Once = true
	}

	n.Bubble = false
	return n
}

// Transition  events
// These events can be composed since it is merely function composition of MutationHandler callbacks
// One would typically create a new transition event and manage the different subtransitions within
// the transitionstart mutation event handler.

// DefineTransition defines a long-form even that can go through different phases (from a start to an end,
// the end possibly caused by an error or a cancellation).
func (e *Element) DefineTransition(name string, onstart, onerror, oncancel, onend *MutationHandler) {
	if onstart == nil {
		panic("onstart transition handler cannot be nil")
	}

	e.WatchEvent(TransitionPhase(name, "start"), e, NewMutationHandler(func(evnt MutationEvent) bool {
		// cancel previous in-flight transitions and reset transition state to not failed, not cancelled)
		//evt.Origin().CancelTransition(name,String("transition restarted"))
		if TransitionStarted(e, name) {
			evnt.Origin().CancelTransition(name, String("transition restarted"))
		}

		evnt.Origin().Set("transition", name, String("started"))

		if onerror != nil {
			onerror = onerror.RunOnce()
			evnt.Origin().WatchEvent(TransitionPhase(name, "error"), evnt.Origin(), NewMutationHandler(func(event MutationEvent) bool {
				evnt.Origin().Set("transition", name, String("error"))
				return false
			}).RunOnce().RunASAP())
			evnt.Origin().WatchEvent(TransitionPhase(name, "error"), evnt.Origin(), onerror)
		}

		if oncancel != nil {
			oncancel = oncancel.RunOnce()
			evnt.Origin().WatchEvent(TransitionPhase(name, "cancel"), evnt.Origin(), NewMutationHandler(func(event MutationEvent) bool {
				evnt.Origin().Set("transition", name, String("cancelled"))
				return false
			}).RunOnce().RunASAP())
			evnt.Origin().WatchEvent(TransitionPhase(name, "cancel"), evnt.Origin(), oncancel)
		}

		if onend != nil {
			onend = onend.RunOnce()
			evnt.Origin().AfterEvent(TransitionPhase(name, "end"), evnt.Origin(), NewMutationHandler(func(event MutationEvent) bool {
				evnt.Origin().TriggerEvent(TransitionPhase(name, "ended"))
				evnt.Origin().Set("transition", name, String("ended"))
				return false
			}).RunOnce())
			evnt.Origin().WatchEvent(TransitionPhase(name, "end"), evnt.Origin(), onend)
		}

		cancelall := NewMutationHandler(func(ev MutationEvent) bool {
			ev.Origin().CancelTransition(name, String("all transitions cancelled"))
			return false
		}).RunOnce()

		evnt.Origin().WatchEvent("cancelalltransitions", evnt.Origin(), cancelall)

		// After the transition start, upon failure, the element should be able to trigger the transition end.
		evnt.Origin().WatchEvent(TransitionPhase(name, "error"), evnt.Origin(), NewMutationHandler(func(ev MutationEvent) bool {
			ev.Origin().Set("transition", name, String("error"))

			ev.Origin().WatchEvent(TransitionPhase(name, "error"), ev.Origin(), NewMutationHandler(func(event MutationEvent) bool {
				event.Origin().EndTransition(name, event.NewValue())
				return false
			}).RunOnce())

			return false
		}).RunOnce())

		// After the transition start, upon cancellation, the element should be able to trigger the transition end.
		evnt.Origin().WatchEvent(TransitionPhase(name, "cancel"), evnt.Origin(), NewMutationHandler(func(ev MutationEvent) bool {
			ev.Origin().Set("transition", name, String("cancelled"))

			ev.Origin().WatchEvent(TransitionPhase(name, "cancel"), ev.Origin(), NewMutationHandler(func(event MutationEvent) bool {
				event.Origin().TriggerEvent(TransitionPhase(name, "ended"))
				event.Origin().Set("transition", name, String("ended"))
				return false
			}).RunOnce())
			return false
		}).RunOnce())

		// Upon transition end, we should  cleanup the transition mutation handlers which didn't get
		// called (e.g. error, cancel)
		evnt.Origin().WatchEvent(TransitionPhase(name, "ended"), evnt.Origin(), NewMutationHandler(func(ev MutationEvent) bool {
			ev.Origin().Unwatch(Namespace.Event, "cancelalltransitions", ev.Origin())
			ev.Origin().Unwatch(Namespace.Event, TransitionPhase(name, "cancel"), ev.Origin())
			ev.Origin().Unwatch(Namespace.Event, TransitionPhase(name, "error"), ev.Origin())
			ev.Origin().Unwatch(Namespace.Event, TransitionPhase(name, "end"), ev.Origin())
			return false
		}).RunOnce())

		evnt.Origin().TriggerEvent(onendedRegistrationHook(name))
		evnt.Origin().TriggerEvent(onendRegistrationHook(name))
		evnt.Origin().TriggerEvent(oncancelRegistrationHook(name))
		evnt.Origin().TriggerEvent(onerrorRegistrationHook(name))
		evnt.Origin().TriggerEvent(onstartRegistrationHook(name))

		if !onstart.Handle(evnt) {
			evnt.Origin().EndTransition(name, String("transition ended"))
			return false
		}
		return true
	}))

	e.TriggerEvent(strings.Join([]string{name, "transition", "defined"}, "-"))
}

func onerrorRegistrationHook(name string) string {
	return strings.Join([]string{TransitionPhase(name, "start"), "register", "onerror"}, "-")
}
func oncancelRegistrationHook(name string) string {
	return strings.Join([]string{TransitionPhase(name, "start"), "register", "oncancel"}, "-")
}
func onendRegistrationHook(name string) string {
	return strings.Join([]string{TransitionPhase(name, "start"), "register", "onend"}, "-")
}

func onendedRegistrationHook(name string) string {
	return strings.Join([]string{TransitionPhase(name, "end"), "register", "onended"}, "-")
}

func onstartRegistrationHook(name string) string {
	return strings.Join([]string{TransitionPhase(name, "start"), "register", "onstart"}, "-")
}

func (e *Element) OnTransitionStart(name string, h *MutationHandler) {
	e.WatchEvent(onstartRegistrationHook(name), e, NewMutationHandler(func(evnt MutationEvent) bool {
		evnt.Origin().WatchEvent(TransitionPhase(name, "start"), evnt.Origin(), h.RunOnce())
		return false
	}).RunASAP().RunOnce())
}

/*
func (e *Element) OnTransitionStart(name string, h *MutationHandler) {
	e.WatchEvent(strings.Join([]string{name, "transition", "defined"}, "-"), e, NewMutationHandler(func(evnt MutationEvent) bool {
		evnt.Origin().WatchEvent(TransitionPhase(name, "start"), evnt.Origin(), NewMutationHandler(func(event MutationEvent) bool {
			event.Origin().WatchEvent(TransitionPhase(name, "start"), event.Origin(), h.RunOnce())
			return false
		}).RunOnce())
		return false
	}).RunASAP().RunOnce())
}
*/

func (e *Element) OnTransitionError(name string, h *MutationHandler) {
	e.WatchEvent(onerrorRegistrationHook(name), e, NewMutationHandler(func(evnt MutationEvent) bool {
		evnt.Origin().WatchEvent(TransitionPhase(name, "error"), evnt.Origin(), h.RunOnce())
		return false
	}).RunOnce().RunASAP())
}

func (e *Element) OnTransitionCancel(name string, h *MutationHandler) {
	e.WatchEvent(oncancelRegistrationHook(name), e, NewMutationHandler(func(evnt MutationEvent) bool {
		evnt.Origin().WatchEvent(TransitionPhase(name, "cancel"), evnt.Origin(), h.RunOnce())
		return false
	}).RunOnce().RunASAP())
}

func (e *Element) OnTransitionEnd(name string, h *MutationHandler) {
	e.WatchEvent(onendRegistrationHook(name), e, NewMutationHandler(func(evnt MutationEvent) bool {
		evnt.Origin().WatchEvent(TransitionPhase(name, "end"), evnt.Origin(), h.RunOnce())
		return false
	}).RunOnce().RunASAP())
}

func (e *Element) AfterTransition(name string, h *MutationHandler) {
	e.WatchEvent(onendedRegistrationHook(name), e, NewMutationHandler(func(evt MutationEvent) bool {
		evt.Origin().WatchEvent(TransitionPhase(name, "ended"), evt.Origin(), h)
		return false
	}).RunOnce().RunASAP())
}

func (e *Element) transitionIsDefined(name string) bool {
	_, ok := e.GetEventValue(strings.Join([]string{name, "transition", "defined"}, "-"))
	return ok
}

func (e *Element) StartTransition(name string, values ...Value) {
	if !e.transitionIsDefined(name) {
		panic(fmt.Sprint(name, " transition is not defined for element ", e.ID))
	}
	e.TriggerEvent(TransitionPhase(name, "start"), values...)
}

func (e *Element) ErrorTransition(name string, values ...Value) {
	if !e.transitionIsDefined(name) {
		panic(fmt.Sprint(name, " transition is not defined"))
	}
	e.TriggerEvent(TransitionPhase(name, "error"), values...)
}

func (e *Element) CancelTransition(name string, values ...Value) {
	if !e.transitionIsDefined(name) {
		panic(fmt.Sprint(name, " transition is not defined"))
	}
	e.TriggerEvent(TransitionPhase(name, "cancel"), values...)
}

func (e *Element) CancelAllTransitions() {
	e.TriggerEvent("cancelalltransitions")
}

func (e *Element) EndTransition(name string, values ...Value) {
	if !e.transitionIsDefined(name) {
		panic(fmt.Sprint(name, " transition is not defined"))
	}
	e.TriggerEvent(TransitionPhase(name, "end"), values...)
}

// TransitionCancelled returns true if the transition was cancelled.
//
// Note: Transition cancellation is not an error. It should still go thorugh the end process which
// also deals with cleaning up the transition state.
// The onend handler should take into account that it has to deal with faile or cancelled transitions.
// But it is always called as we need to arrive to a known state, whichever way.
func TransitionCancelled(e *Element, transitionname string) bool {
	v, ok := e.Get("transition", transitionname)
	if !ok {
		return false
	}
	vv, ok := v.(String)
	if !ok {
		return false
	}
	return vv.String() == "cancelled"
}

func TransitionError(e *Element, transitionname string) bool {
	v, ok := e.Get("transition", transitionname)
	if !ok {
		return false
	}
	vv, ok := v.(String)
	if !ok {
		return false
	}
	return vv.String() == "error"
}

func TransitionEnded(e *Element, transitionname string) bool {
	v, ok := e.Get("transition", transitionname)
	if !ok {
		return false
	}
	vv, ok := v.(String)
	if !ok {
		return false
	}
	return vv.String() == "ended"
}

func TransitionStarted(e *Element, transitionname string) bool {
	v, ok := e.Get("transition", transitionname)
	if !ok {
		return false
	}
	vv, ok := v.(String)
	if !ok {
		return false
	}
	return vv.String() == "started"
}

func TransitionEndValue(e *Element, transitionname string) (Value, error) {
	v, ok := e.Get(Namespace.Event, TransitionPhase(transitionname, "end"))
	if !ok {
		return nil, errors.New("transition doesn't seem to have completed yet")
	}
	o, ok := v.(Object).Get("value")
	if !ok {
		panic("unexpected event object format for transition end event")
	}
	return o, nil
}

/* DEBUG rewrite this

// NewTransitionChain creates a new transition chain. The transition chain is a sequence of
// transitions that are triggered synchronously.These transitions belongs to a same element by construction.
func (e *Element) NewTransitionChain(name string, transitionevents ...string) func(onstart, onerror, oncancel, onend *MutationHandler) {

	h := NewMutationHandler(func(evt MutationEvent) bool {
		// When a transition ends, the next one should start unless cancellation was triggered for one of the transitions
		// in the chain.
		l := len(transitionevents) - 1
		for i, t := range transitionevents {
			if i == 0 {
				e.OnTransitionEnd(name, NewMutationHandler(func(evt MutationEvent) bool {
					// check cancellation status first
					if TransitionCancelled(e, name) {
						return false
					}
					e.TriggerEvent(TransitionPhase(t, "start"))
					return false
				}))
			}

			if 0 < i && i < l {
				e.OnTransitionEnd(t, NewMutationHandler(func(evt MutationEvent) bool {
					// check cancellation status first
					if TransitionCancelled(e, t) || TransitionError(e, t) {
						return false
					}

					e.TriggerEvent(TransitionPhase(transitionevents[i+1], "start"))
					return false
				}))
			}
			if i == l {
				e.OnTransitionEnd(t, NewMutationHandler(func(evt MutationEvent) bool {
					if TransitionCancelled(e, t) || TransitionError(e, t) {
						return false
					}

					e.TriggerEvent(TransitionPhase(name, "end"))
					return false
				}))
			}

			e.OnTransitionCancel(name, NewMutationHandler(func(evt MutationEvent) bool {
				e.TriggerEvent(TransitionPhase(t, "cancel"))
				return false
			}))
		}
		if len(transitionevents) > 0 {
			e.TriggerEvent(TransitionPhase(transitionevents[0], "start"))
		}

		return false
	})

	return func(onstart, onerror, oncancel, onend *MutationHandler) {
		g := NewMutationHandler(func(evt MutationEvent) bool {
			if onstart != nil {
				b := onstart.Handle(evt)
				if b {
					return true
				}
			}
			return h.Handle(evt)
		})
		e.DefineTransition(name, g, onerror, oncancel, onend)
	}

}

*/

// TransitionPhase returns the name for the event that is triggered when a transition reaches
// a given phase.
func TransitionPhase(name string, phase string) string {
	return strings.Join([]string{"tr", name, phase}, "-")
}

type LifecycleHandlers struct {
	root *Element
}

func NewLifecycleHandlers(root AnyElement) LifecycleHandlers {
	return LifecycleHandlers{root.AsElement()}
}

func (l LifecycleHandlers) OnReady(h *MutationHandler) {
	l.root.WatchEvent("ui-ready", l.root, h)
}

func (l LifecycleHandlers) OnIdle(h *MutationHandler) {
	l.root.WatchEvent("ui-idle", l.root, h)
}

func (l LifecycleHandlers) OnLoad(h *MutationHandler) {
	l.root.WatchEvent(TransitionPhase("load", "start"), l.root, h)
}

func (l LifecycleHandlers) OnLoaded(h *MutationHandler) {
	l.root.WatchEvent("ui-loaded", l.root, h)
}

// SetReady is used to signal that the UI tree has been built and data/resources have been loaded on both the Go/wasm side
// and the native side.
func (l LifecycleHandlers) SetReady() {
	l.root.TriggerEvent("ui-ready")
}

// MutationShouldReplay
func (l LifecycleHandlers) MutationShouldReplay(b bool) {
	_, ok := l.root.Get(Namespace.Internals, "mutation-should-replay")
	if !ok {
		l.root.OnTransitionStart("load", NewMutationHandler(func(evt MutationEvent) bool {
			if l.MutationWillReplay() {
				evt.Origin().StartTransition("replay")
			}
			return false
		}))
	}
	l.root.Set(Namespace.Internals, "mutation-should-replay", Bool(b))
}

// MutationWillReplay
func (l LifecycleHandlers) MutationWillReplay() bool {
	v, ok := l.root.Get(Namespace.Internals, "mutation-should-replay")
	if !ok {
		return false
	}
	return bool(v.(Bool))
}

// MutationReplaying
func (l LifecycleHandlers) MutationReplaying() bool {
	return MutationReplaying(l.root)
}

func (l LifecycleHandlers) OnMutationsReplayed(h *MutationHandler) {
	l.root.WatchEvent(TransitionPhase("replay", "ended"), l.root, h.RunASAP())
}
