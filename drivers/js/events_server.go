//go:build server

// Package doc defines the default set of Element constructors, native interfaces,
// events and event handlers, and animation properties used to build js-based UIs.
package doc

import (

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




// NativeDispatch allows for the propagation of a JS event created in Go.
func NativeDispatch(evt ui.Event){}


var NativeEventBridge = func(NativeEventName string, listener *ui.Element, capture bool) {}


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

func keyboardEventSerialized(o *ui.TempObject,e KeyboardEvent){
	o.Set("altKey",ui.Bool(e.altKey))
	o.Set("ctrlKey",ui.Bool(e.ctrlKey))
	o.Set("shiftKey", ui.Bool(e.shiftKey))
	o.Set("metaKey",ui.Bool(e.metaKey))

	o.Set("repeat",ui.Bool(e.repeat))
	o.Set("isComposing",ui.Bool(e.isComposing))

	o.Set("location",ui.Number(e.location))

	o.Set("code",ui.String(e.code))
	o.Set("key",ui.String(e.key))

	o.Commit()
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

func mouseEventSerialized(o *ui.TempObject,e MouseEvent){
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

	o.Commit()

}

func newMouseEvent(e ui.Event) MouseEvent{
	var k MouseEvent
	k.Event = e
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



