// +build js,wasm

// Package doc defines the default set of Element constructors, native interfaces,
// events and event handlers, and animation properties used to build js-based UIs.
package doc

import (
	"syscall/js"
	//"net/url"
	"github.com/atdiar/particleui"
)

type cancelable struct {
	js.Value
}

func (c cancelable) PreventDefault() {
	c.Value.Call("preventDefault")
}

func DefaultGoEventTranslator(evt ui.Event) js.Value {
	var event = js.Global().Get("Event").New(evt.Type(), map[string]interface{}{
		"bubbles":    evt.Bubbles(),
		"cancelable": evt.Cancelable(),
	})
	return event
}

var NativeEventBridge = func(NativeEventName string, target *ui.Element) {
	// Let's create the callback that will be called from the js side
	cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		evt := args[0]
		evt.Call("stopPropagation")

		// Time to create the corresponding GoEvent
		typ := evt.Get("type").String()
		bubbles := evt.Get("bubbles").Bool()
		cancancel := evt.Get("cancelable").Bool()
		var target *ui.Element
		targetid := evt.Get("target").Get("id")
		value := evt.Get("target").Get("value").String()
		if targetid.Truthy() {
			target = Elements.GetByID(targetid.String())
		} else {
			// this might be a stretch... but we assume that the only element without
			// a native side ID is the window in javascript.
			target = GetWindow().Element()
		}

		var nativeEvent interface{}
		nativeEvent = evt
		if cancancel {
			nativeEvent = cancelable{evt}
		}
		if typ == "popstate" || typ == "load" {
			//value = js.Global().Get("document").Get("URL").String()
			value = js.Global().Get("location").Get("pathname").String()
			/*u,err:= url.ParseRequestURI(value)
			if err!= nil{
				value = ""
			} else{
				value = u.Path
			}*/

		}
		goevt := ui.NewEvent(typ, bubbles, cancancel, target, nativeEvent, value)

		target.DispatchEvent(goevt, nil)
		return nil
	})

	if target.ID != GetWindow().Element().ID {
		js.Global().Get("document").Call("getElementById", target.ID).Call("addEventListener", NativeEventName, cb)
		if target.NativeEventUnlisteners.List == nil {
			target.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
		}
		target.NativeEventUnlisteners.Add(NativeEventName, func() {
			js.Global().Get("document").Call("getElementById", target.ID).Call("removeEventListener", NativeEventName, cb)
			cb.Release()
		})
	} else {
		js.Global().Call("addEventListener", NativeEventName, cb)
		if target.NativeEventUnlisteners.List == nil {
			target.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
		}
		target.NativeEventUnlisteners.Add(NativeEventName, func() {
			js.Global().Call("removeEventListener", NativeEventName, cb)
			cb.Release()
		})
	}

}
