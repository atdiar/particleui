// +build js,wasm

// Package javascript defines the default set of Element constructors, native interfaces,
// events and event handlers, and animation properties used to build js-based UIs.
package javascript

import (
	"fmt"
	"log"
	"strings"
	"syscall/js"

	"github.com/atdiar/particleui"
)

var (
	// DOCTYPE holds the document doctype.
	DOCTYPE = "html/js"
	// Elements stores wasm-generated HTML ui.Element constructors.
	Elements   = ui.NewElementStore(DOCTYPE)
	EventTable = NewEventTranslationTable()
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
	e = enableClasses(e)

	htmlDiv := js.Global().Get("document").Call("createElement", "div")
	htmlDiv.Set("id", id)
	n := NewNativeElementWrapper(htmlDiv)
	e.Native = n
	return e
})

// NewSpan is a constructor for html div elements.
var NewSpan = Elements.NewConstructor("span", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)
	e = enableClasses(e)

	htmlSpan := js.Global().Get("document").Call("createElement", "span")
	htmlSpan.Set("id", id)
	n := NewNativeElementWrapper(htmlSpan)
	e.Native = n
	return e
})

// NewDiv is a constructor for html div elements.
var NewParagraph = Elements.NewConstructor("paragraph", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)
	e = enableClasses(e)

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
	e = enableClasses(e)

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

// NewTemplatedText returns either a textnode appended to the Element whose id
// is passed as argument, or a div wrapping a textnode if no ui.Element exists
// yet for the id.
// The template accepts a parameterized string as would be accepted by fmt.Sprint
// and the parameter should have their names passed as arguments.
// Done correctly, calling element.Set("data", paramname, stringvalue) will
// set the textnode with a new string value where the parameter whose name is
// `paramname` is set with the value `stringvalue`.
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
	FromJS          map[string]func(evt js.Value) ui.Event
	ToJS            map[string]func(evt ui.Event) js.Value
	nameTranslation map[nameTranslation]string
}

type nameTranslation struct {
	Event  string
	Native bool
}

func translationKey(evtname string, js bool) nameTranslation {
	return nameTranslation{evtname, js}
}

func NewEventTranslationTable() EventTranslationTable {
	return EventTranslationTable{make(map[string]func(evt js.Value) ui.Event), make(map[string]func(evt ui.Event) js.Value), make(map[nameTranslation]string)}
}

// Register enables the storage of an event translation function which is used
// by ui.Element to listen to events that are actually dispatched from the
// underlying javascript target.
func (e EventTranslationTable) GoEventTranslator(goEventName string, nativeEventName string, toJS func(ui.Event) js.Value) {
	e.ToJS[goEventName] = toJS
	e.nameTranslation[translationKey(goEventName, false)] = nativeEventName
}

func (e EventTranslationTable) JSEventTranslator(nativeEventName string, goEventName string, fromJS func(js.Value) ui.Event) {
	e.FromJS[nativeEventName] = fromJS
	e.nameTranslation[translationKey(nativeEventName, true)] = goEventName
}

func (e EventTranslationTable) TranslateEventName(evt string, jsNative bool) string {
	res, ok := e.nameTranslation[translationKey(evt, jsNative)]
	if !ok {
		return evt
	}
	return res
}

func (e EventTranslationTable) NativeEventBridge() ui.NativeEventBridge {
	return func(evt string, target *ui.Element) {
		translate, ok := e.FromJS[evt]
		NativeEventName := e.nameTranslation[translationKey(evt, false)]
		if !ok {
			translate = DefaultJSEventTranslator
			NativeEventName = evt
		}
		// Let's create the callback that will be called from the js side
		cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			nativeEvent := args[0]
			nativeEvent.Call("stopPropagation")
			goevt := translate(nativeEvent)
			target.DispatchEvent(goevt, nil)
			return nil
		})
		js.Global().Get("document").Call("getElementById", target.ID).Call("addEventListener", NativeEventName, cb)
		if target.NativeEventUnlisteners.List == nil {
			target.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
		}
		target.NativeEventUnlisteners.Add(NativeEventName, func() {
			js.Global().Get("document").Call("getElementById", target.ID).Call("removeEventListener", NativeEventName, cb)
		})
	}
}

func (e EventTranslationTable) NativeDispatcher() ui.NativeDispatch {
	return func(evt ui.Event) {
		translate, ok := e.ToJS[evt.Type()]
		if !ok {
			translate = DefaultGoEventTranslator
		}
		nativeevent := translate(evt)
		nelmt, ok := evt.Target().Native.(NativeElement)
		if !ok {
			log.Print("Unable to dispatch event for non-javascript html element")
			return
		}
		nelmt.Value.Call("dispatchEvent", nativeevent)
	}
}

func (e EventTranslationTable) EventFromJS(evt js.Value) ui.Event {
	typ := evt.Get("type").String()
	translate, ok := e.FromJS[typ]
	if !ok {
		translate = DefaultJSEventTranslator
	}
	return translate(evt)
}

func (e EventTranslationTable) EventToJS(evt ui.Event) js.Wrapper {
	translate, ok := e.ToJS[evt.Type()]
	if !ok {
		translate = DefaultGoEventTranslator
	}
	return translate(evt)
}

func AddClass(target *ui.Element, classname string) {
	category := "css"
	classes, ok := target.Get(category, "class")
	if ok {
		c, ok := classes.(string)
		if !ok {
			target.Set(category, "class", classname, false)
			return
		}
		if !strings.Contains(c, classname) {
			c = c + " " + classname
			target.Set(category, "class", c, false)
		}
		return
	}
	target.Set(category, "class", classname, false)
}

func RemoveClass(target *ui.Element, classname string) {
	category := "css"
	classes, ok := target.Get(category, "class")
	if !ok {
		return
	}
	c, ok := classes.(string)
	if !ok {
		return
	}
	c = strings.TrimPrefix(c, classname)
	c = strings.TrimPrefix(c, " ")
	c = strings.ReplaceAll(c, classname+" ", " ")
	target.Set(category, "class", c, false)
}

func Classes(target *ui.Element) []string {
	category := "css"
	classes, ok := target.Get(category, "class")
	if !ok {
		return nil
	}
	c, ok := classes.(string)
	if !ok {
		return nil
	}
	return strings.Split(c, " ")
}

func enableClasses(e *ui.Element) *ui.Element {
	h := ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		target := evt.Origin()
		native, ok := target.Native.(NativeElement)
		if !ok {
			log.Print("wrong type for native element or native element does not exist")
			return true
		}
		classes, ok := evt.NewValue().(string)
		if !ok {
			log.Print("new value of non-string type. Unable to use as css class(es)")
			return true
		}
		native.Value.Set("class", classes)
		return false
	})
	e.Watch("css", "class", e, h)
	return e
}

// TODO check that the string is well formatted style
func SetInlineCSS(target *ui.Element, str string) {
	SetAttribute(target, "style", str)
}

func GetInlineCSS(target *ui.Element) string {
	return GetAttribute(target, "style")
}

func AppendInlineCSS(target *ui.Element, str string) {
	css := GetInlineCSS(target)
	css = css + str
	SetInlineCSS(target, css)
}

func GetAttribute(target *ui.Element, name string) string {
	native, ok := target.Native.(NativeElement)
	if !ok {
		log.Print("Cannot retrieve Attribute on non-expected wrapper type")
		return ""
	}
	return native.Value.Call("getAttribute", "name").String()
}

func SetAttribute(target *ui.Element, name string, value string) {
	target.Set("attrs", name, value, false)
	native, ok := target.Native.(NativeElement)
	if !ok {
		log.Print("Cannot set Attribute on non-expected wrapper type")
		return
	}
	native.Value.Call("setAttribute", name, value)
}

func RemoveAttribute(target *ui.Element, name string) {
	target.Delete("attrs", name)
	native, ok := target.Native.(NativeElement)
	if !ok {
		log.Print("Cannot delete Attribute using non-expected wrapper type")
		return
	}
	native.Value.Call("removeAttribute", name)
}
