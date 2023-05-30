//go:build !server && js && wasm


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



var DEBUG = log.Print // DEBUG


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


type NativeEvent struct {
	Value js.Value
}

func (e NativeEvent) PreventDefault() {
	e.Value.Call("preventDefault")
}

func(e NativeEvent) StopPropagation(){
	e.Value.Call("stopPropagation")
}

func(e NativeEvent) StopImmediatePropagation(){
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
	t,ok:= JSValue(evt.Target())
	if !ok{
		return
	}
	if t.Truthy(){
		t.Call("dispatchEvent",e)
	}
}


var NativeEventBridge = func(NativeEventName string, listener *ui.Element, capture bool) {

	// Let's create the callback that will be called from the js side
	cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defer func() {
			if r := recover(); r != nil {
				doc:= GetDocument(listener)
				body:= doc.Body().AsElement()
				msg:= doc.Div.WithID("appfailure")
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
				GetDocument(listener).Window().SetTitle("Critical App Failure")
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

		
		ui.DoSync(func(){
			if currtargetid.Truthy() {
				currentTarget= GetDocument(listener).GetElementById(currtargetid.String())
				if currentTarget == nil {
					DEBUG("no currenttarget found for this element despite a valid id")
					return 
				}
				if targetid.Truthy() {
					target = GetDocument(listener).GetElementById(targetid.String())	
					if target == nil {
						DEBUG("no target found for this element despite a valid id")
						return 
					}
				}else{
					if jstarget.Equal(js.Global().Get("document").Get("defaultView")) {
						target = GetDocument(listener).Window().AsElement()
					} else{
						DEBUG("no target found for this element, no id and it's not the window")
						return
					}
				}
				
			} else {
				// this might be a stretch... but we assume that the only valid element without
				// a native side ID is the window in javascript.
				if jscurrtarget.Equal(js.Global().Get("document").Get("defaultView")) {
					currentTarget = GetDocument(listener).Window().AsElement()
					if targetid.Truthy() {
						target = GetDocument(listener).GetElementById(targetid.String())	
						if target == nil {
							DEBUG("currenttarget is window but no target found for this element despite a valid id")
							return 
						}
					}else{
						if jstarget.Equal(js.Global().Get("document").Get("defaultView")) {
							target = GetDocument(listener).Window().AsElement()		
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
			nevt = NativeEvent{evt}

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
						GetDocument(listener).AsElement().SyncUISyncData("history", hstateobj.Value())
					}
				}
			}
	

			if v:=jstarget.Get("value"); v.Truthy(){
				rv.Set("value",ui.String(v.String()))
			}

			jsUIEvent:= js.Global().Get("UIEvent")
			jsInputEvent:= js.Global().Get("InputEvent")
			jsHashChangeEvent:= js.Global().Get("HashChangeEvent")
			jsKeyboardEvent := js.Global().Get("KeyboardEvent")
			jsMouseEvent:= js.Global().Get("MouseEvent")

			if evt.InstanceOf(jsUIEvent){
				rv.Set("detail",ui.Number(evt.Get("detail").Float()))
				rv.Set("which",ui.Number(evt.Get("which").Float()))
			}
			
			if evt.InstanceOf(jsInputEvent){
				rv.Set("data",ui.String(evt.Get("data").String()))
				rv.Set("inputType", ui.String(evt.Get("inputType").String()))

			} else if evt.InstanceOf(jsKeyboardEvent){
				event:= newKeyboardEvent(goevt)
				keyboardEventSerialized(rv,event)
				goevt = event

			} else if evt.InstanceOf(jsMouseEvent){
				event:= newMouseEvent(goevt)
				mouseEventSerialized(rv,event)
				goevt = event

			}else if evt.InstanceOf(jsHashChangeEvent){
				rv.Set("newURL",ui.String(evt.Get("newURL").String()))
				rv.Set("oldURL",ui.String(evt.Get("oldURL").String()))
			}
			
			currentTarget.Handle(goevt)

		})
		
		return nil
	})


	tgt,ok:= JSValue(listener)
	if !ok{

	}
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

type KeyboardEvent struct{
	ui.Event

	altKey bool
	code string
	ctrlKey bool
	isComposing bool
	key string
	location float64
	metaKey bool
	repeat bool
	shiftKey bool

}

func keyboardEventSerialized(o ui.Object,e KeyboardEvent){
	o.Set("altKey",ui.Bool(e.altKey))
	o.Set("ctrlKey",ui.Bool(e.ctrlKey))
	o.Set("shiftKey", ui.Bool(e.shiftKey))
	o.Set("metaKey",ui.Bool(e.metaKey))

	o.Set("repeat",ui.Bool(e.repeat))
	o.Set("isComposing",ui.Bool(e.isComposing))

	o.Set("location",ui.Number(e.location))

	o.Set("code",ui.String(e.code))
	o.Set("key",ui.String(e.key))
}

func(k KeyboardEvent) GetModifierState()bool{
	return k.altKey || k.ctrlKey || k.metaKey || k.shiftKey
}

func(k KeyboardEvent) AltKey() bool{
	return k.altKey
}

func(k KeyboardEvent) CtrlKey() bool{
	return k.ctrlKey
}

func(k KeyboardEvent) MetaKey() bool{
	return k.metaKey
}

func(k KeyboardEvent) ShiftKey() bool{
	return k.shiftKey
}

func(k KeyboardEvent) Code() string{
	return k.code
}

func(k KeyboardEvent) Composing() bool{
	return k.isComposing
}

func(k KeyboardEvent) Key() string{
	return k.key
}

func(k KeyboardEvent) Location() float64{
	return k.location
}

func(k KeyboardEvent) Repeat() bool{
	return k.repeat
}


func newKeyboardEvent(e ui.Event) KeyboardEvent{
	var k KeyboardEvent
	k.Event = e
	evt:= e.Native().(NativeEvent).Value
	
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



type MouseEvent struct{
	ui.Event

	altKey bool
	button float64
	buttons float64
	clientX float64
	clientY float64
	ctrlKey bool
	metaKey bool
	movementX float64
	movementY float64
	offsetX float64
	offsetY float64
	pageX float64
	pageY float64
	relatedTarget *ui.Element
	screenX float64
	screenY float64
	shiftKey bool
}

func mouseEventSerialized(o ui.Object,e MouseEvent){
	o.Set("altKey",ui.Bool(e.altKey))
	o.Set("ctrlKey",ui.Bool(e.ctrlKey))
	o.Set("shiftKey", ui.Bool(e.shiftKey))
	o.Set("metaKey",ui.Bool(e.metaKey))

	o.Set("button",ui.Number(e.button))
	o.Set("buttons",ui.Number(e.buttons))
	o.Set("clientX",ui.Number(e.clientX))
	o.Set("clientY",ui.Number(e.clientY))
	o.Set("movementX",ui.Number(e.movementX))
	o.Set("movemnentY",ui.Number(e.movementY))
	o.Set("offsetX",ui.Number(e.offsetX))
	o.Set("offsetY",ui.Number(e.offsetY))
	o.Set("pageX",ui.Number(e.pageX))
	o.Set("pageY",ui.Number(e.pageY))
	o.Set("screenX",ui.Number(e.screenX))
	o.Set("screenY",ui.Number(e.screenY))

	if e.relatedTarget != nil{
		o.Set("relatedTarget",ui.String(e.relatedTarget.ID))
	}

}

func newMouseEvent(e ui.Event) MouseEvent{
	var k MouseEvent
	k.Event = e
	evt:= e.Native().(NativeEvent).Value
	
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
			k.relatedTarget= GetDocument(e.Target()).GetElementById(id.String())
		}
	}
	return k
}



func(k MouseEvent) GetModifierState()bool{
	return k.altKey || k.ctrlKey || k.metaKey || k.shiftKey
}

func(k MouseEvent) AltKey() bool{
	return k.altKey
}

func(k MouseEvent) CtrlKey() bool{
	return k.ctrlKey
}

func(k MouseEvent) MetaKey() bool{
	return k.metaKey
}

func(k MouseEvent) ShiftKey() bool{
	return k.shiftKey
}

func(k MouseEvent) Button() float64{
	return k.button
}

func(k MouseEvent) Buttons() float64{
	return k.buttons
}

func(k MouseEvent) ClientX() float64{
	return k.clientX
}

func(k MouseEvent) X() float64{
	return k.clientX
}

func(k MouseEvent) ClientY() float64{
	return k.clientY
}

func(k MouseEvent) Y() float64{
	return k.clientY
}

func(k MouseEvent) MovementX() float64{
	return k.movementX
}

func(k MouseEvent) MovementY() float64{
	return k.movementY
}

func(k MouseEvent) OffsetX() float64{
	return k.offsetX
}

func(k MouseEvent) OffsetY() float64{
	return k.offsetY
}

func(k MouseEvent) PageX() float64{
	return k.pageX
}

func(k MouseEvent) PageY() float64{
	return k.pageY
}

func(k MouseEvent) ScreenX() float64{
	return k.screenX
}

func(k MouseEvent) ScreenY() float64{
	return k.screenY
}

func(k MouseEvent) RelatedTarget() *ui.Element{
	return k.RelatedTarget()
}

