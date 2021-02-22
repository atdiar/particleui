// Package javascript defines the default set of Element constructors, native interfaces,
// events and event handlers, and animation properties used to build js-based UIs.
package javascript

import (
	"syscall/js"

	"github.com/atdiar/particleui"
)

type cancelable struct {
	js.Value
}

func (c cancelable) PreventDefault() {
	c.Value.Call("preventDefault")
}

func DefaultJSEventTranslator(evt js.Value) ui.Event {
	typ := evt.Get("type").String()
	bubbles := evt.Get("bubbles").Bool()
	cancancel := evt.Get("cancelable").Bool()
	targetid := evt.Get("target").Get("id").String()
	target := Elements.GetByID(targetid)
	var nativeEvent interface{}
	nativeEvent = evt
	if cancancel {
		nativeEvent = cancelable{evt}
	}
	return ui.NewEvent(typ, bubbles, cancancel, target, nativeEvent)
}

func DefaultGoEventTranslator(evt ui.Event) js.Value {
	var event = js.Global().Get("Event").New(evt.Type(), map[string]interface{}{
		"bubbles":    evt.Bubbles(),
		"cancelable": evt.Cancelable(),
	})
	return event
}

func LoadDefaultEventTable(evttbl EventTranslationTable) {
	evttbl.JSEventTranslator("popstate", "routechange", func(evt js.Value) ui.Event {
		targetid := evt.Get("target").Get("id").String()
		target := Elements.GetByID(targetid)
		return ui.NewRouteChangeEvent(js.Global().Get("location").String(), target)
	})
}
