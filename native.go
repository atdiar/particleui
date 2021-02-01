// Package ui is a library of functions for simple, generic gui development.
package ui

type NativeElement interface {
	AppendChild(child *Element)
	PrependChild(child *Element)
	InsertChild(child *Element, index int)
	ReplaceChild(old *Element, new *Element)
	RemoveChild(child *Element)
}

type NativeDispatch func(evt Event)

type NativeEventBridge func(event string, target *Element) // TODO might need to turn this in a stuct with func and map fields, map needed to map the event names from Go to native Host's

type NativeEventUnlisteners struct {
	List map[string]func()
}

func NewNativeEventUnlisteners() NativeEventUnlisteners {
	return NativeEventUnlisteners{make(map[string]func(), 0)}
}

func (n NativeEventUnlisteners) Add(event string, f func()) {
	_, ok := n.List[event]
	if ok {
		return
	}
	n.List[event] = f
}

func (n NativeEventUnlisteners) Apply(event string) {
	removeNativeEventListener, ok := n.List[event]
	if !ok {
		return
	}
	removeNativeEventListener()
}
