// +build js,wasm

// Package doc defines the default set of Element constructors, native interfaces,
// events and event handlers, and animation properties used to build js-based UIs.
package doc

import (
	"syscall/js"
	//"net/url"
	"encoding/json"
	"log"

	"github.com/atdiar/particleui"
)

var DEBUG = log.Print // DEBUG

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
		var target ui.BasicElement
		jstarget := evt.Get("currentTarget")
		value := jstarget.Get("value").String()
		targetid := jstarget.Get("id")

		if targetid.Truthy() {
			element := Elements.GetByID(targetid.String())
			if element != nil {
				target = ui.BasicElement{element}
			} else {
				return nil
			}
		} else {
			// this might be a stretch... but we assume that the only element without
			// a native side ID is the window in javascript.
			if jstarget.Equal(js.Global().Get("document").Get("defaultView")) {
				target = GetWindow().AsBasicElement()
			} else {
				// the element has probably been deleted on the Go wasm side
				return nil
			}
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

			// TGiven that events are not handled concurrently
			// but triggered sequentially, we can Set the value of the history state
			// on the target *ui.Element, knowing that it will be visible before
			// the event dispatch.
			hstate := js.Global().Get("history").Get("state")

			if hstate.Truthy() {
				hstateobj := ui.NewObject()
				err := json.Unmarshal([]byte(hstate.String()), &hstateobj)
				if err == nil {
					target.AsElement().SetUI("history", hstateobj.Value())
				}
			}
		}

		if typ == "keyup" || typ == "keydown" || typ == "keypress" {
			value = evt.Get("key").String()
		}
		goevt := ui.NewEvent(typ, bubbles, cancancel, target.AsElement(), nativeEvent, value)
		target.AsElement().DispatchEvent(goevt, nil)
		return nil
	})

	if target.ID != GetWindow().AsElement().ID {
		js.Global().Get("document").Call("getElementById", target.ID).Call("addEventListener", NativeEventName, cb)
		if target.NativeEventUnlisteners.List == nil {
			target.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
		}
		target.NativeEventUnlisteners.Add(NativeEventName, func() {
			v := js.Global().Get("document").Call("getElementById", target.ID)
			if !v.IsNull() {
				v.Call("removeEventListener", NativeEventName, cb)
			} else {
				// DEBUG("Call for event listener removal on ", target.ID)
			}
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
