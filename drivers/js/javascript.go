// +build js,wasm

// Package javascript defines the default set of Element constructors, native interfaces,
// events and event handlers, and animation properties used to build js-based UIs.
package javascript

import (
	"fmt"
	"log"
	"syscall/js"

	"github.com/atdiar/particleui"
)

var (
	// DOCTYPE holds the document doctype.
	DOCTYPE = "html/js"
	// Elements stores wasm-generated HTML ui.Element constructors.
	Elements = ui.NewElementStore(DOCTYPE)
)

// NativeElement defines a wrapper around a js.Value that implements the
// ui.NativeElementWrapper interface.
type NativeElement struct {
	js.Value
}

func NewNativeElementWrapper(v js.Value) NativeElement {
	return NativeElement{v}
}

func (n NativeElement) AppendChild(child *ui.Element) {
	v, ok := child.Native.(NativeElement)
	if !ok {
		log.Print("wrong format for native element underlying objects.Cannot append " + child.Name)
		return
	}
	n.Value.Call("append", v.Value)
}

func (n NativeElement) PrependChild(child *ui.Element) {
	v, ok := child.Native.(NativeElement)
	if !ok {
		log.Print("wrong format for native element underlying objects.Cannot prepend " + child.Name)
		return
	}
	n.Value.Call("prepend", v.Value)
}

func (n NativeElement) InsertChild(child *ui.Element, index int) {
	v, ok := child.Native.(NativeElement)
	if !ok {
		log.Print("wrong format for native element underlying objects.Cannot insert " + child.Name)
		return
	}
	childlist := n.Value.Get("children")
	length := childlist.Get("length").Int()
	if index >= length {
		log.Print("insertion attempt out of bounds.")
		return
	}
	r := childlist.Call("item", index)
	n.Value.Call("insertBefore", v, r)
}

func (n NativeElement) ReplaceChild(old *ui.Element, new *ui.Element) {
	nold, ok := old.Native.(NativeElement)
	if !ok {
		log.Print("wrong format for native element underlying objects.Cannot replace " + old.Name)
		return
	}
	nnew, ok := new.Native.(NativeElement)
	if !ok {
		log.Print("wrong format for native element underlying objects.Cannot replace with " + new.Name)
		return
	}
	//nold.Call("replaceWith", nnew) also works
	n.Value.Call("replaceChild", nnew.Value, nold.Value)
}

func (n NativeElement) RemoveChild(child *ui.Element) {
	v, ok := child.Native.(NativeElement)
	if !ok {
		log.Print("wrong format for native element underlying objects.Cannot insert " + child.Name)
		return
	}
	n.JSValue().Call("removeChild", v.JSValue())
}

/*
//
//
// Element Constructors
//
//
//
*/

var NewAppRoot = Elements.NewConstructor("root", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)
	root := js.Global().Get("document")
	n := NewNativeElementWrapper(root)
	e.Native = n
	return e
})

// NewDiv is a constructor for html div elements.
var NewDiv = Elements.NewConstructor("div", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)

	htmlDiv := js.Global().Get("document").Call("createElement", "div")
	htmlDiv.Set("id", id)
	n := NewNativeElementWrapper(htmlDiv)
	e.Native = n
	return e
})

// NewSpan is a constructor for html div elements.
var NewSpan = Elements.NewConstructor("span", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)

	htmlSpan := js.Global().Get("document").Call("createElement", "span")
	htmlSpan.Set("id", id)
	n := NewNativeElementWrapper(htmlSpan)
	e.Native = n
	return e
})

// NewDiv is a constructor for html div elements.
var NewParagraph = Elements.NewConstructor("paragraph", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)

	htmlParagraph := js.Global().Get("document").Call("createElement", "p")
	htmlParagraph.Set("id", id)
	n := NewNativeElementWrapper(htmlParagraph)
	e.Native = n
	return e
})

// NewAnchor creates an html anchor element which points to the object whose id is
// being passed as argument.
// If the object does not exist, it points to itself.
var NewAnchor = Elements.NewConstructor("link", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)
	htmlAnchor := js.Global().Get("document").Call("createElement", "a")
	htmlAnchor.Set("id", id+"-link")
	// finds the element whose id has been passed as argument: if search returns nil
	// then the Link element references itself.
	lnkTarget := Elements.GetByID(id)
	if lnkTarget == nil {
		lnkTarget = e
		htmlAnchor.Set("id", id)
	}

	// TODO Set a mutation Handler on e which observes the tree insertion event (attach event)
	// At each attachment, we should rewrite href with the new route.
	lnkTarget.Watch("event", "attached", lnkTarget, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		if evt.ObservedKey() != "attached" || evt.Type() != "event" || evt.Origin() != lnkTarget {
			return true
		}
		htmlAnchor.Set("href", lnkTarget.Route())
		return false
	}))
	n := NewNativeElementWrapper(htmlAnchor)
	e.Native = n
	return e
})

// NewTextNode creates a text node for the Element whose id is passed as argument
// The id for the text Element is the id of its parent to which
// is suffixed "-txt-" and a random number.
// If the parent does not exist, a parent div is created whose id is the one
// passed as argument.
// To change the value of the text, one would Set the "text" property belonging
// to the "data" category/namespace. i.e. Set("data","text",value)
var NewTextNode = Elements.NewConstructor("text", func(name string, parentid string) *ui.Element {
	e := ui.NewElement(name, parentid+"-txt-"+ui.NewIDgenerator(789465)(), Elements.DocType)
	htmlTextNode := js.Global().Get("document").Call("createTextNode", "")
	n := NewNativeElementWrapper(htmlTextNode)
	e.Native = n

	target := Elements.GetByID(parentid)
	if target == nil {
		target = NewDiv(name, parentid)
	}

	e.Watch("data", "text", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		if evt.ObservedKey() != "text" || evt.Type() != "data" || evt.Origin() != target {
			return true
		}
		if s, ok := evt.NewValue().(string); ok {
			htmlTextNode.Set("nodeValue", s)
		}
		return false
	}))

	target.AppendChild(e)
	return e
})

// TODO NewTextTemplate : require to parse template ... template parameter format
// {{param}}

var NewTemplatedText = func(name string, id string, format string, paramsNames ...string) *ui.Element {
	nt := NewTextNode(name, id)

	formatter := func(tplt string, params ...string) string {
		v := make([]interface{}, len(params))
		for i, p := range params {
			val, ok := nt.Get("data", p)
			if ok {
				v[i] = val
			}
			continue
		}
		return fmt.Sprintf(tplt, v...)
	}
	params := make([]string, len(paramsNames))
	for i, p := range paramsNames {
		nt.Watch("data", p, nt, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			if evt.ObservedKey() != p || evt.Type() != "data" || evt.Origin() != nt {
				return true
			}
			s, ok := evt.NewValue().(string)
			if ok {
				params[i] = s
			}

			nt.Set("data", "text", formatter(format, params...), false)
			return false
		}))
	}
	return nt
}

type EventTranslationTable struct {
	List map[string]struct {
		NativeEventName string
		FromJS          func(evt js.Value) ui.Event
		ToJS            func(evt ui.Event) js.Value
	}
}

func NewEventTranslationTable() EventTranslationTable {
	return EventTranslationTable{make(map[string]struct {
		NativeEventName string
		FromJS          func(evt js.Value) ui.Event
		ToJS            func(evt ui.Event) js.Value
	})}
}

// Register enables the storage of an event translation function which is used
// by ui.Element to listen to events that are actually dispatched from the
// underlying javascript target.
func (e EventTranslationTable) Register(goEventName string, nativeEventName string, fromJS func(js.Value) ui.Event, toJS func(ui.Event) js.Value) {
	if e.List == nil {
		e.List = make(map[string]struct {
			NativeEventName string
			FromJS          func(evt js.Value) ui.Event
			ToJS            func(evt ui.Event) js.Value
		})
	}
	e.List[goEventName] = struct {
		NativeEventName string
		FromJS          func(evt js.Value) ui.Event
		ToJS            func(evt ui.Event) js.Value
	}{nativeEventName, fromJS, toJS}
}

func (e EventTranslationTable) NativeEventBridge() ui.NativeEventBridge {
	return func(evt string, target *ui.Element) {
		translation, ok := e.List[evt]
		if !ok {
			log.Print("Could not find translation fonction to convert go event into corresponding js event")
			return
		}
		// Let's create the callback that will be called from the js side
		cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			nativeEvent := args[0]
			nativeEvent.Call("stopPropagation")
			goevt := translation.FromJS(nativeEvent)
			target.DispatchEvent(goevt, nil)
			return nil
		})
		js.Global().Get("document").Call("getElementById", target.ID).Call("addEventListener", translation.NativeEventName, cb)
		if target.NativeEventUnlisteners.List == nil {
			target.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
		}
		target.NativeEventUnlisteners.Add(translation.NativeEventName, func() {
			js.Global().Get("document").Call("getElementById", target.ID).Call("removeEventListener", translation.NativeEventName, cb)
		})
	}
}

func (e EventTranslationTable) NativeDispatcher() ui.NativeDispatch {
	return func(evt ui.Event) {
		translation, ok := e.List[evt.Type()]
		if !ok {
			log.Print("Cannot dispatch event to underlying native target. Event translation object missing.")
			return
		}
		nativeevent := translation.ToJS(evt)
		nelmt, ok := evt.Target().Native.(NativeElement)
		if !ok {
			log.Print("Unable to dispatch event for non-javascript html element")
			return
		}
		nelmt.Value.Call("dispatchEvent", nativeevent)
	}
}
