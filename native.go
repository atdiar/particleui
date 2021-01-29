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

/* Example of Generic JS Event bridging
(It does not handle the specifics of the event, such as event target value so a library of specific event bridges should be built for each target platform
and incorporated to the JSEventbridge function (switch over event types and dispatch of a more specific event type on the go side))

func JSEventBridge() NativeEventBridge {
	return func(event Event, target *Element) {
		cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			nativeEvent := args[0]
			nativeEvent.Call("stopPropagation")
			evt := NewEvent(event.Type(), true, target, nativeEvent)
      if evt,ok := event.(RouteChangeEvent);ok{
        evt = NewRouteChangeEvent(event.Type(),true,target,nativeEvent, evt.target.Value) // this is pseudo code for an example of how to switch on the event type
      }
			target.DispatchEvent(evt,nil)
			return nil
		})
		js.Global().Get("document").Call("getElementById", target.ID).Call("addEventListener", event.Type(), cb)
    if target.NativeEventUnlisteners.List == nil {
			target.NativeEventUnlisteners = NewNativeEventUnlisteners()
		}
		target.NativeEventUnlisteners.Add(func() { js.Global("document").Call("getElementById", target.ID).Call("removeEventListener", event.Type(), cb) })
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
