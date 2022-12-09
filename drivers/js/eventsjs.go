// go:build js && wasm

// go:build client, !server

// +build client

// Package doc defines the default set of Element constructors, native interfaces,
// events and event handlers, and animation properties used to build js-based UIs.
package doc

import (
	"syscall/js"

	//"net/url"
	"encoding/json"
	"fmt"
	"log"
	"runtime"

	"github.com/atdiar/particleui"
)




/*
var dEBUGJS = func(v js.Value, isJsonString ...bool){
	if isJsonString!=nil{
		o:= js.Global().Get("JSON").Call("parse",v)
		js.Global().Get("console").Call("log",o)
		return
	}
	js.Global().Get("console").Call("log",v)
}


*/

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


var NativeEventBridge = func(NativeEventName string, listener *ui.Element, capture bool) {

	// Let's create the callback that will be called from the js side
	cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defer func() {
			if r := recover(); r != nil {
				body:= Document{ui.BasicElement{listener.Root()}}.Body().AsElement()
				msg:= Div("appfailure","appfailure")
				SetInlineCSS(msg.AsElement(),`all: initial;`)

				switch txt:= r.(type){
				case string:
					msg.SetText(txt)
				case ui.String:
					msg.SetText(string(txt))
				default:
					msg.SetText("Critical app failure. See console")
					DEBUG(r)
				}
				body.SetChildren(msg)
				GetWindow().SetTitle("Critical App Failure")
			}
		}()

		
		
		evt := args[0]
		//evt.Call("stopPropagation")

		// Let's create the corresponding GoEvent
		typ := evt.Get("type").String()
		bubbles := evt.Get("bubbles").Bool()
		cancancel := evt.Get("cancelable").Bool()
		phase := int(evt.Get("eventPhase").Float())

		var target *ui.Element
		jstarget := evt.Get("target")
		targetid := jstarget.Get("id")

		var currentTarget *ui.Element
		jscurrtarget := evt.Get("currentTarget")
		currtargetid := jscurrtarget.Get("id")

		rv:= ui.NewObject() //.Set("value",ui.String(jstarget.Get("value").String()))
	

		ui.Do(func(){
			if currtargetid.Truthy() {
				currentTarget= Elements.GetByID(currtargetid.String())
				if currentTarget == nil {
					DEBUG("no currenttarget found for this element despite a valid id")
					return 
				}
				if targetid.Truthy() {
					target = Elements.GetByID(targetid.String())	
					if target == nil {
						DEBUG("no target found for this element despite a valid id")
						return 
					}
				}else{
					if jstarget.Equal(js.Global().Get("document").Get("defaultView")) {
						target = GetWindow().AsElement()		
					} else{
						DEBUG("no target found for this element, no id and it's not the window")
						return
					}
				}
				
			} else {
				// this might be a stretch... but we assume that the only valid element without
				// a native side ID is the window in javascript.
				if jscurrtarget.Equal(js.Global().Get("document").Get("defaultView")) {
					currentTarget = GetWindow().AsElement()
					if targetid.Truthy() {
						target = Elements.GetByID(targetid.String())	
						if target == nil {
							DEBUG("currenttarget is window but no target found for this element despite a valid id")
							return 
						}
					}else{
						if jstarget.Equal(js.Global().Get("document").Get("defaultView")) {
							target = GetWindow().AsElement()		
						} else{
							//DEBUG("target seems to be #document")
							//dEBUGJS(jstarget)
							
						}
					}	
				} else {
					// the element has probably been deleted on the Go wasm sides
					DEBUG("no etarget element found for this event")
					return
				}
			}
	
			var nevt interface{}
			nevt = nativeEvent{evt}

			goevt := ui.NewEvent(typ, bubbles, cancancel, target, currentTarget, nevt, rv)
			goevt.SetPhase(phase)
	
			if typ == "popstate" {
				rv.Set("value",ui.String(js.Global().Get("location").Get("pathname").String()))

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
				//dEBUGJS(hstate,true)
				if hstate.Truthy() {
					hstateobj := ui.NewObject()
					err := json.Unmarshal([]byte(hstate.String()), &hstateobj)
					if err == nil {
						GetDocument().AsElement().SyncUISetData("history", hstateobj.Value())
					}
				}
			}
	
			/*if typ == "click"{
				button:= ui.Number(evt.Get("button").Float()) // TODO add other click event properties
				ctrlKey:= ui.Bool(evt.Get("ctrlKey").Bool())
				rv.Set("button",button)
				rv.Set("ctrlKey",ctrlKey)
			}*/

			if v:=jstarget.Get("value"); v.Truthy(){
				rv.Set("value",ui.String(v.String()))
			}

			jsUIEvent:= js.Global().Get("UIEvent")
			jsInputEvent:= js.Global().Get("InputEvent")
			jsHashChangeEvent:= js.Global().Get("HashChangeEvent")
			jsKeyboardEvent := js.Global().Get("KeyboardEvent")
			jsMouseEvent:= js.Global().Get("MouseEvent")
			
			if evt.InstanceOf(jsInputEvent){
				rv.Set("data",ui.String(evt.Get("data").String()))
				rv.Set("inputType", ui.String(evt.Get("inputType").String()))
			}

			if evt.InstanceOf(jsHashChangeEvent){
				rv.Set("newURL",ui.String(evt.Get("newURL").String()))
				rv.Set("oldURL",ui.String(evt.Get("oldURL").String()))
			}

			if evt.InstanceOf(jsUIEvent){
				rv.Set("detail",ui.Number(evt.Get("detail").Float()))
			}

			if evt.InstanceOf(jsKeyboardEvent){
				event:= newKeyboardEvent(goevt)
				keyboardEventSerialized(rv,event)
				goevt = event
			}

			if evt.InstanceOf(jsMouseEvent){
				event:= newMouseEvent(goevt)
				mouseEventSerialized(rv,event)
				goevt = event
			}
	
			
			currentTarget.Handle(goevt)

		})
		
		return nil
	})


	tgt:= JSValue(listener)
	if !tgt.Truthy(){
		panic("trying to add an event listener to non-existing HTML element on the JS side")
	}
	tgt.Call("addEventListener", NativeEventName, cb,capture)
	if listener.NativeEventUnlisteners.List == nil {
		listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
	}
	listener.NativeEventUnlisteners.Add(NativeEventName, func() {
		tgt.Call("removeEventListener", NativeEventName, cb)
		cb.Release()
	})

}



func newKeyboardEvent(e ui.Event) KeyboardEvent{
	var k KeyboardEvent
	k.Event = e
	evt:= e.Native().(js.Value)
	
	if v:=evt.Get("key"); v.Truthy(){
		k.key = v.String()
	}

	if v:=evt.Get("altKey"); v.Truthy(){
		k.altKey = v.Bool()
	}
	
	if v:= evt.Get("ctrlKey"); v.Truthy(){
		k.ctrlKey = v.Bool()
	}

	if v:= evt.Get("metaKey");v.Truthy(){
		k.metaKey = v.Bool()
	}

	if v:= evt.Get("shiftKey"); v.Truthy(){
		k.shiftKey = v.Bool()
	}

	if v:=evt.Get("code"); v.Truthy(){
		k.code = v.String()
	}

	if v:= evt.Get("isComposing"); v.Truthy(){
		k.isComposing = v.Bool()
	}

	if v:= evt.Get("repeat"); v.Truthy(){
		k.repeat = v.Bool()
	}

	if v:=evt.Get("location"); v.Truthy(){
		k.location = v.Float()
	}
	return k
}


func newMouseEvent(e ui.Event) MouseEvent{
	var k MouseEvent
	k.Event = e
	evt:= e.Native().(js.Value)
	
	if v:=evt.Get("button"); v.Truthy(){
		k.button = v.Float()
	}

	if v:=evt.Get("buttons"); v.Truthy(){
		k.buttons = v.Float()
	}

	if v:=evt.Get("altKey"); v.Truthy(){
		k.altKey = v.Bool()
	}
	
	if v:= evt.Get("ctrlKey"); v.Truthy(){
		k.ctrlKey = v.Bool()
	}

	if v:= evt.Get("metaKey");v.Truthy(){
		k.metaKey = v.Bool()
	}

	if v:= evt.Get("shiftKey"); v.Truthy(){
		k.shiftKey = v.Bool()
	}

	if v:=evt.Get("movementX"); v.Truthy(){
		k.movementX = v.Float()
	}

	if v:= evt.Get("movementY"); v.Truthy(){
		k.movementY = v.Float()
	}

	if v:= evt.Get("offsetX"); v.Truthy(){
		k.offsetX = v.Float()
	}

	if v:=evt.Get("offsetY"); v.Truthy(){
		k.offsetY = v.Float()
	}

	if v:= evt.Get("clientX"); v.Truthy(){
		k.clientX = v.Float()
	}

	if v:=evt.Get("clientY"); v.Truthy(){
		k.clientY = v.Float()
	}

	if v:= evt.Get("pageX"); v.Truthy(){
		k.pageX = v.Float()
	}

	if v:=evt.Get("pageX"); v.Truthy(){
		k.pageX = v.Float()
	}

	if v:= evt.Get("screenX"); v.Truthy(){
		k.screenX = v.Float()
	}

	if v:=evt.Get("screenY"); v.Truthy(){
		k.screenY = v.Float()
	}

	if v:=evt.Get("relatedTarget"); v.Truthy(){
		if id:= v.Get("id"); id.Truthy(){
			k.relatedTarget= Elements.GetByID(id.String())
		}
	}
	return k
}


