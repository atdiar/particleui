// go:build js && wasm

// Package doc defines the default set of Element constructors, native interfaces,
// events and event handlers, and animation properties used to build js-based UIs.
package doc

import (
	"syscall/js"
	//"net/url"
	"encoding/json"
	"github.com/atdiar/particleui"
	"log"
	"runtime"
	"fmt"
)

ui.NativeEventBridge = NativeEventBridge
ui.NativeDispatch = NativeDispatch

var DEBUG = log.Print // DEBUG
var DEBUGJS = func(v js.Value, isJsonString ...bool){
	if isJsonString!=nil{
		o:= js.Global().Get("JSON").Call("parse",v)
		js.Global().Get("console").Call("log",o)
		return
	}
	js.Global().Get("console").Call("log",v)
}

func SDEBUG(){
	pc := make([]uintptr, 30)
	n := runtime.Callers(0, pc)
	DEBUG(n)
	if n == 0{
		return
	}
	pc = pc[:n] // pass only valid pcs to runtime.CallersFrames
	frames := runtime.CallersFrames(pc)

	for {
		frame, more := frames.Next()

		fmt.Printf("%s\n", frame.Function)

		// Check whether there are more frames to process after this one.
		if !more {
			break
		}
	}


}

type nativeEvent struct {
	js.Value
}

func (e nativeEvent) PreventDefault() {
	e.Value.Call("preventDefault")
}

func(e nativeEvent) StopPropagation(){
	e.Value.Call("stopPropagation")
}

func(e nativeEvent) StopImmediatePropagation(){
	e.Value.Call("stopImmediatePropagation")
}


func defaultGoEventTranslator(evt ui.Event) js.Value {
	var event = js.Global().Get("Event").New(evt.Type(), map[string]interface{}{
		"bubbles":    evt.Bubbles(),
		"cancelable": evt.Cancelable(),
	})
	return event
}

// NativeDispatch allows for the propagation of a JS event created in Go.
func NativeDispatch(evt ui.Event){
	e:= defaultGoEventTranslator(evt)
	t:= JSValue(evt.Target())
	if t.Truthy(){
		t.Call("dispatchEvent",e)
	}
}


var NativeEventBridge = func(NativeEventName string, target *ui.Element, capture bool) {

	// Let's create the callback that will be called from the js side
	cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		
		evt := args[0]
		//evt.Call("stopPropagation")

		// Let's create the corresponding GoEvent
		typ := evt.Get("type").String()
		bubbles := evt.Get("bubbles").Bool()
		cancancel := evt.Get("cancelable").Bool()

		var target *ui.Element
		jstarget := evt.Get("Target")
		targetid := jstarget.Get("id")

		var currentTarget *ui.Element
		jscurrtarget := evt.Get("currentTarget")
		currtargetid := jscurrtarget.Get("id")

		var value ui.Value
		rawvalue := ui.String(jstarget.Get("value").String())
		value = rawvalue
		

		b:= ui.Lock.TryLock()
		defer func(){
			if b{
				ui.Lock.Unlock()
			}
		}()


		if currtargetid.Truthy() && targetid.Truthy() {
			currentTarget = Elements.GetByID(currtargetid.String())
			target= Elements.GetByID(targetid.String())
			if target == nil || currentTarget==nil{
				return nil
			}

		} else {
			// this might be a stretch... but we assume that the only valid element without
			// a native side ID is the window in javascript.
			if targetid.Truthy() && jscurrtarget.Equal(js.Global().Get("document").Get("defaultView")) {
				currentTarget = GetWindow().AsElement()
				target= Elements.GetByID(targetid.String())
			} else {
				// the element has probably been deleted on the Go wasm sides
				DEBUG("no etarget element found for this event")
				return nil
			}
		}

		var nevt interface{}
		nevt = nativeEvent{evt}
		if typ == "popstate" {
			value = ui.String(js.Global().Get("location").Get("pathname").String())
			/*u,err:= url.ParseRequestURI(value)
			if err!= nil{
				value = ""
			} else{
				value = u.Path
			}*/

			// Given that events are not handled concurrently
			// but triggered sequentially, we can Set the value of the history state
			// on the target *ui.Element, knowing that it will be visible before
			// the event dispatch.
			hstate := js.Global().Get("history").Get("state")
			//DEBUGJS(hstate,true)

			if hstate.Truthy() {
				hstateobj := ui.NewObject()
				err := json.Unmarshal([]byte(hstate.String()), &hstateobj)
				if err == nil {
					currentTarget.AsElement().SetUI("history", hstateobj.Value())
				}
			}
		}

		if typ == "keyup" || typ == "keydown" || typ == "keypress" {
			value = ui.String(evt.Get("key").String())
		}

		if typ == "click"{
			button:= ui.Number(evt.Get("button").Float()) // TODO add other click event properties
			ctrlKey:= ui.Bool(evt.Get("ctrlKey").Bool())
			v:= ui.NewObject()
			v.Set("button",button)
			v.Set("ctrlKey",ctrlKey)
			value = v
		}

		goevt := ui.NewEvent(typ, bubbles, cancancel, target, currentTarget, nevt, value)
		goevt.SetPhase(2)
		target.AsElement().Handle(goevt)
		return nil
	})


	tgt:= JSValue(target)
	if !tgt.Truthy(){
		panic("trying to add an event listener to non-existing HTML element on the JS side")
	}
	tgt.Call("addEventListener", NativeEventName, cb,capture)
	if target.NativeEventUnlisteners.List == nil {
		target.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
	}
	target.NativeEventUnlisteners.Add(NativeEventName, func() {
		tgt.Call("removeEventListener", NativeEventName, cb)
		cb.Release()
	})

}
