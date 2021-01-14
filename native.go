// Package ui is a library of functions for simple, generic gui development.
package ui

type NativeElementWrapper interface {
	AppendChild(child *Element)
	PrependChild(child *Element)
	InsertChild(child *Element, position int)
	ReplaceChild(old *Element, new *Element)
	RemoveChild(child *Element)
}

type NativeEventBridge func(event string, target *Element)

/* Example of Generic JS Event bridging
(It does not handle the specifics of the event, such as event target value so a library of specific event bridges should be built for each target platform
and incorporated to the JSEventbridge function (switch over event types and dispatch of a more specific event type on the go side))

func JSEventBridge() NativeEventBridge {
	return func(event string, target *Element) {
		cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			nativeEvent := args[0]
			nativeEvent.Call("stopPropagation")
			evt := NewEvent(event, true, target, nativeEvent)
      if event == "routechange"{
        evt = NewRouteChangeEvent(event,true,target,nativeEvent, evt.target.Value) // this is pseudo code for an example of how to switch on the event type
      }
			target.DispatchEvent(evt)
			return nil
		})
		js.Global().Get("document").Call("getElementById", target.ID).Call("addEventListener", event, cb)
    if target.NativeEventUnlisteners.List == nil {
			target.NativeEventUnlisteners = NewNativeEventUnlisteners()
		}
		target.NativeEventUnlisteners.Add(func() { js.Global("document").Call("getElementById", target.ID).Call("removeEventListener", event, cb) })
	}
}
*/

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

/*

  AddEventListener: how to link to the native event:
  the native event should be available from the callback function.
  1. Extract native event...
  2. Build Go event
  3. Dispatch Go event.

  By default, we stop propagation on the js side events as the handling logic occurs
  on the Go side.

  E.g. with native JS element
  EvtDispatcherBridge := func(e *Element, event string) (removebridge func()){

    cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
      nativeEvent := args[0]
      nativeEvent.Call("stopPropagation")
      evt := NewEvent(event,true,e,nativeEvent)
      e.DispatchEvent(evt)
      return nil
    })
    js.Global().Get("document").Call("getElementById", e.ID).Call("addEventListener", event, cb)
    return func(){js.Global("document")Call("getElementById", e.ID).Call("removeEventListener", event, cb)}
  }

*/
