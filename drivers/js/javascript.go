// +build js,wasm

// Package javascript defines the default set of Element constructors, native interfaces,
// events and event handlers, and animation properties used to build js-based UIs.
package javascript

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"syscall/js"

	"github.com/atdiar/particleui"
)

var (
	// DOCTYPE holds the document doctype.
	DOCTYPE = "html/js"
	// Elements stores wasm-generated HTML ui.Element constructors.
	Elements                      = ui.NewElementStore("default", DOCTYPE)
	EventTable                    = NewEventTranslationTable()
	DefaultWindowTitle            = "Powered by ParticleUI"
	EnablePropertyAutoInheritance = ui.EnablePropertyAutoInheritance
)

var NewID = ui.NewIDgenerator(56813256545869)

// mutationCaptureMode describes how a Go App may capture textarea value changes
// that happen in native javascript. For instance, when a blur event is dispatched
// or when any mutation is observed via the MutationObserver API.
type mutationCaptureMode int

const (
	onBlur mutationCaptureMode = iota
	onInput
)

/*
type Storage struct{
	Load func(key string) interface{}
	Store func(key string, value interface{})
}

func(s Storage) Get(key string) interface{}{
	return s.Load(key)
}

func(s Storage) Set(key string, value interface{}){
	s.store(key,value)
}

func NewStorage(load)

*/

type jsStore struct {
	store js.Value
}

func (s jsStore) Get(key string) (string, bool) {
	v := s.store.Call("getItem", key)
	if !v.Truthy() {
		return "", false
	}
	return v.String(), true
}

func (s jsStore) Set(key string, value string) {
	s.store.Call("setItem", key, value)
}

// Let's add sessionstorage and localstorage for Element properties.
// For example, an Element which would have been created with the sessionstorage option
// would have every set properties stored in sessionstorage, available for
// later recovery. It enables to have data that persists runs and loads of a
// web app.
/*
var sessionstorefn = func(element *ui.Element, category string, propname string, value interface{},flags ...bool){
	store:= jsStore{js.Global().Get("sessionStorage")}
	if category != "ui"{
		categoryExists := element.Properties.HasCategory()
		propertyExists := element.Properties.HasProperty()

		if  !categoryExists{
			categories := make([]string,0,len(element.Properties.Categories)+1)
			for k,_:= range element.Properties.Categories{
				categories = append(categories,k)
			}
			categories = append(categories, category)
			v,err:= json.Marshal(categories)
			if err!=nil{
				log.Print(err)
				return
			}
			store.Set(element.ID,string(v))
		}
		proptype:= "Local"
		if len(flags) > 0{
			if flags[0]{
				proptype = "Inheritable"
			}
		}
		if !propertyExists{
			props := make([]string,0,1)
			c,ok:=element.Properties[category]
			if !ok{
				props = append(props,proptype+"/"+propname)
				v,err:=json.Marshal(props)
				if err!=nil{
					log.Print(err)
					return
				}
				store.Set(element.ID+"/"+category,string(v))
			}
			for k,_:= range c.Default{
				props = append(props,"Default/"+k)
			}
			for k,_:= range c.Inherited{
				props = append(props,"Inherited/"+k)
			}
			for k,_:= range c.Local{
				props = append(props,"Local/"+k)
			}
			for k,_:= range c.Inheritable{
				props = append(props,"Inheritable/"+k)
			}

			props = append(props,proptype+"/"+propname)
			v,err:=json.Marshal(props)
			if err!=nil{
				log.Print(err)
				return
			}
			store.Set(element.ID+"/"+category,string(v))
		}
		val,err:= json.Marshal(value)
		if err!=nil{
			log.Print(err)
			return
		}
		store.Set(element.ID+"/"+category+"/"+propname, string(val))
		return
	}

	// Now in the case we want to persist a ui mutation

}


Elements.AddPersistenceMode("sessionstorage",loadfromsessionstore,sessionstorefn)
*/
// Window is a ype that represents a browser window
type Window struct {
	*ui.Element
}

func (w Window) SetTitle(title string) {
	w.Element.Set("ui", "title", ui.String(title))
}

// TODO see if can get height width of window view port, etc.

func getWindow() Window {
	e := ui.NewElement("window", DefaultWindowTitle, DOCTYPE)
	e.Native = NewNativeElementWrapper(js.Global())

	h := ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		target := evt.Origin()
		newtitle, ok := evt.NewValue().(ui.String)
		if !ok {
			return true
		}

		if target != e {
			return true
		}
		nat, ok := target.Native.(js.Wrapper)
		if !ok {
			return true
		}
		jswindow := nat.JSValue()
		jswindow.Get("document").Set("title", string(newtitle))
		return false
	})

	e.Watch("ui", "title", e, h)
	e.Set("ui", "title", ui.String(DefaultWindowTitle), false)
	return Window{e}
}

var DefaultWindow Window = getWindow()

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
	n.JSValue().Call("append", v.JSValue())
}

func (n NativeElement) PrependChild(child *ui.Element) {
	v, ok := child.Native.(NativeElement)
	if !ok {
		log.Print("wrong format for native element underlying objects.Cannot prepend " + child.Name)
		return
	}
	n.JSValue().Call("prepend", v.JSValue())
}

func (n NativeElement) InsertChild(child *ui.Element, index int) {
	v, ok := child.Native.(NativeElement)
	if !ok {
		log.Print("wrong format for native element underlying objects.Cannot insert " + child.Name)
		return
	}
	childlist := n.JSValue().Get("children")
	length := childlist.Get("length").Int()
	if index >= length {
		log.Print("insertion attempt out of bounds.")
		return
	}
	r := childlist.Call("item", index)
	n.JSValue().Call("insertBefore", v, r)
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
	n.JSValue().Call("replaceChild", nnew.JSValue(), nold.JSValue())
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

// AllowSessionPersistence is a constructor option. When passed as argument in
// the creation of a ui.Element constructor, it allows for ui.Element constructors to
// different options for property persistence.
var AllowSessionPersistence = ui.NewConstructorOption("sessionstorage", func(e *ui.Element) *ui.Element {
	ui.LoadElementProperty(e, "internals", "persistence", "default", ui.String("sessionstorage"))
	return e
})

var AllowAppLocalPersistence = ui.NewConstructorOption("localstorage", func(e *ui.Element) *ui.Element {
	ui.LoadElementProperty(e, "internals", "persistence", "default", ui.String("localstorage"))
	return e
})

func EnableSessionPeristence() string {
	return "sessionstorage"
}

func EnableAppLocalPersistence() string {
	return "localstorage"
}

// NewAppRoot creates a new app entry point. It is the top-most element
// in the tree of Elements that consitute the full document.
// It should be the element which is passed to a router to observe for route
// change.
// By default, it represents document.body. As such, it is different from the
// document which holds the head element for instance.
var NewAppRoot = Elements.NewConstructor("root", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)
	root := js.Global().Get("document").Get("body")
	n := NewNativeElementWrapper(root)
	e.Native = n
	return e
})

// NewDiv is a constructor for html div elements.
var NewDiv = Elements.NewConstructor("div", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)
	e = enableClasses(e)

	// TODO call getElementById first..; if element exist already, no need to call createElement
	// Also, need to try and load any corresponding properties that would have been persisted and retrigger ui.mutations to recover ui state.
	htmlDiv := js.Global().Get("document").Call("createElement", "div")
	n := NewNativeElementWrapper(htmlDiv)
	e.Native = n
	SetAttribute(e, "id", id)
	return e
}, AllowTooltip)

var tooltipConstructor = Elements.NewConstructor("tooltip", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)
	e = enableClasses(e)

	htmlTooltip := js.Global().Get("document").Call("createElement", "div")
	n := NewNativeElementWrapper(htmlTooltip)
	e.Native = n
	SetAttribute(e, "id", id)

	h := ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		content, ok := evt.NewValue().(*ui.Element)
		if ok {
			tooltip := evt.Origin()
			// tooltip.RemoveChildren()
			tooltip.Set("ui", "command", ui.RemoveChildrenCommand(), false)
			// tooltip.AppendChild(content)
			tooltip.Set("ui", "command", ui.AppendChildCommand(content), false)
			return false
		}
		strcontent, ok := evt.NewValue().(ui.String)
		if !ok {
			return true
		}

		tooltip := evt.Origin()
		// tooltip.RemoveChildren()
		tooltip.Set("ui", "command", ui.RemoveChildrenCommand(), false)
		tn := NewTextNode()
		tn.Set("data", "text", strcontent, false)
		//tooltip.AppendChild(tn)
		tooltip.Set("ui", "command", ui.AppendChildCommand(tn), false)
		return false
	})
	e.Watch("data", "content", e, h)

	return e
})

func TryRetrieveTooltip(target *ui.Element) *ui.Element {
	return target.ElementStore.GetByID(target.ID + "-tooltip")
}

// EnableTooltip, when passed to a constructor, creates a tootltip html div element (for a given target ui.Element)
// The content of the tooltip can be directly set by  specifying a value for
// the ("data","content") (category,propertyname) Element datastore entry.
// The content value can be a string or another ui.Element.
// The content of the tooltip can also be set by modifying the ("tooltip","content")
// property
func EnableTooltip() string {
	return "AllowTooltip"
}

var AllowTooltip = ui.NewConstructorOption("AllowTooltip", func(target *ui.Element) *ui.Element {
	e := tooltipConstructor(target.Name+"/tooltip", target.ID+"-tooltip")
	// Let's observe the target element which owns the tooltip too so that we can
	// change the tooltip automatically from there.
	h := ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		e.Set("data", "content", evt.NewValue(), false)
		return false
	})
	target.Watch("tooltip", "content", target, h)

	//target.AppendChild(e)
	target.Set("ui", "command", ui.AppendChildCommand(e), false)
	return target
})

// NewTextArea is a constructor for a textarea html element.
var NewTextArea = func(name string, id string, rows int, cols int, options ...string) *ui.Element {
	return Elements.NewConstructor("textarea", func(ename string, eid string) *ui.Element {
		e := ui.NewElement(ename, eid, Elements.DocType)
		e = enableClasses(e)

		htmlTextArea := js.Global().Get("document").Call("createElement", "textarea")

		e.Watch("data", "text", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			if s, ok := evt.NewValue().(ui.String); ok {
				old := htmlTextArea.Get("value").String()
				if string(s) != old {
					SetAttribute(evt.Origin(), "value", string(s))
				}
			}
			return false
		}))

		n := NewNativeElementWrapper(htmlTextArea)
		e.Native = n
		SetAttribute(e, "name", ename)
		SetAttribute(e, "id", eid)
		SetAttribute(e, "rows", strconv.Itoa(rows))
		SetAttribute(e, "cols", strconv.Itoa(cols))
		return e
	}, allowTextAreaDataBindingOnBlur, allowTextAreaDataBindingOnInput, AllowTooltip)(name, id, options...)
}

// allowTextAreaDataBindingOnBlur is a constructor option for TextArea UI elements enabling
// TextAreas to activate an option ofr two-way databinding.
var allowTextAreaDataBindingOnBlur = ui.NewConstructorOption("SyncOnBlur", func(e *ui.Element) *ui.Element {
	return enableDataBinding(onBlur)(e)
})

// allowTextAreaDataBindingOnInoput is a constructor option for TextArea UI elements enabling
// TextAreas to activate an option ofr two-way databinding.
var allowTextAreaDataBindingOnInput = ui.NewConstructorOption("SyncOnInput", func(e *ui.Element) *ui.Element {
	return enableDataBinding(onInput)(e)
})

// EnableTextAreaSyncOnBlur returns the name of the option that can be passed to
// textarea ui.Element constructor to trigger two-way databinding on textarea
// blur event.
func EnableTextAreaSyncOnBlur() string {
	return "SyncOnBlur"
}

// EnableTextAreaSyncOnInput returns the name of the option that can be passed to
// textarea ui.Element constructor to trigger two-way databinding on textarea
// input event.
func EnableTextAreaSyncOnInput() string {
	return "SyncOnInput"
}

func enableDataBinding(datacapturemode ...mutationCaptureMode) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		callback := ui.NewEventHandler(func(evt ui.Event) bool {
			if evt.Target().ID != e.ID {
				return false // we do not stop the event propagation but do not handle the event either
			}
			n, ok := e.Native.(NativeElement)
			if !ok {
				return true
			}
			nn := n.JSValue()
			v := nn.Get("value")
			ok = v.Truthy()
			if !ok {
				return true
			}
			s := v.String()
			e.Set("data", "text", ui.String(s), false)
			return false
		})

		if datacapturemode == nil || len(datacapturemode) > 1 {
			e.AddEventListener("blur", callback, EventTable.NativeEventBridge())
			return e
		}
		mode := datacapturemode[0]
		if mode == onInput {
			e.AddEventListener("input", callback, EventTable.NativeEventBridge())
			return e
		}

		// capture textarea value on blur by default
		e.AddEventListener("blur", callback, EventTable.NativeEventBridge())
		return e
	}
}

// TODO attribute setting functions such as Placeholder(val string) func(*ui.Element) *ui.Element to implement

// NewHeader is a constructor for a html header element.
var NewHeader = Elements.NewConstructor("header", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)
	e = enableClasses(e)

	htmlHeader := js.Global().Get("document").Call("createElement", "header")
	n := NewNativeElementWrapper(htmlHeader)
	e.Native = n
	SetAttribute(e, "id", id)
	return e
}, AllowTooltip)

// NewFooter is a constructor for an html footer element.
var NewFooter = Elements.NewConstructor("footer", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)
	e = enableClasses(e)

	htmlFooter := js.Global().Get("document").Call("createElement", "footer")
	n := NewNativeElementWrapper(htmlFooter)
	e.Native = n
	SetAttribute(e, "id", id)
	return e
}, AllowTooltip)

// NewSpan is a constructor for html div elements.
var NewSpan = Elements.NewConstructor("span", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)
	e = enableClasses(e)

	htmlSpan := js.Global().Get("document").Call("createElement", "span")
	n := NewNativeElementWrapper(htmlSpan)
	e.Native = n
	SetAttribute(e, "id", id)
	return e
}, AllowTooltip)

// NewDiv is a constructor for html div elements.
var NewParagraph = Elements.NewConstructor("paragraph", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)
	e = enableClasses(e)

	htmlParagraph := js.Global().Get("document").Call("createElement", "p")
	n := NewNativeElementWrapper(htmlParagraph)
	e.Native = n
	SetAttribute(e, "id", id)
	return e
}, AllowTooltip)

// NewNavMenu is a constructor for a html nav element.
var NewNavMenu = Elements.NewConstructor("nav", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)
	e = enableClasses(e)

	htmlNavMenu := js.Global().Get("document").Call("createElement", "nav")
	n := NewNativeElementWrapper(htmlNavMenu)
	e.Native = n
	SetAttribute(e, "id", id)
	return e
}, AllowTooltip)

// NewAnchor creates an html anchor element which points to the object whose id is
// being passed as argument.
// If the object does not exist, it points to itself.
var NewAnchor = Elements.NewConstructor("link", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)
	e = enableClasses(e)

	htmlAnchor := js.Global().Get("document").Call("createElement", "a")
	baseid := id
	id = id + "-link"
	// finds the element whose id has been passed as argument: if search returns nil
	// then the Link element references itself.
	lnkTarget := Elements.GetByID(baseid)
	if lnkTarget == nil {
		lnkTarget = e
		id = baseid
	}

	// Set a mutation Handler on lnkTarget which observes the tree insertion event (attach event)
	// At each attachment, we should rewrite href with the new route.
	lnkTarget.Watch("event", "attached", lnkTarget, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		if evt.ObservedKey() != "attached" || evt.Type() != "event" || evt.Origin() != lnkTarget {
			return true
		}

		SetAttribute(e, "href", e.Route())
		return false
	}))
	n := NewNativeElementWrapper(htmlAnchor)
	e.Native = n
	SetAttribute(e, "id", id)
	return e
}, AllowTooltip)

var NewButton = func(name string, id string, typ string, options ...string) *ui.Element {
	f := Elements.NewConstructor("button", func(elementname string, elementid string) *ui.Element {
		e := ui.NewElement(elementname, elementid, Elements.DocType)
		e = enableClasses(e)

		htmlButton := js.Global().Get("document").Call("createElement", "button")
		n := NewNativeElementWrapper(htmlButton)
		e.Native = n
		SetAttribute(e, "name", elementname)
		SetAttribute(e, "id", elementid)
		SetAttribute(e, "type", typ)
		return e
	}, AllowTooltip)
	return f(name, id, options...)
}

var NewInput = func(name string, id string, typ string, options ...string) *ui.Element {
	f := Elements.NewConstructor("input", func(elementname string, elementid string) *ui.Element {
		e := ui.NewElement(elementname, elementid, Elements.DocType)
		e = enableClasses(e)

		htmlInput := js.Global().Get("document").Call("createElement", "input")

		n := NewNativeElementWrapper(htmlInput)
		e.Native = n
		SetAttribute(e, "name", elementname)
		SetAttribute(e, "id", elementid)
		SetAttribute(e, "type", typ)
		return e
	})
	return f(name, id, options...)
}

var NewImage = func(src string, id string, altname string, options ...string) *ui.Element {
	return Elements.NewConstructor("image", func(name string, imgid string) *ui.Element {
		e := ui.NewElement(name, imgid, Elements.DocType)
		e = enableClasses(e)

		htmlImg := js.Global().Get("document").Call("createElement", "img")

		n := NewNativeElementWrapper(htmlImg)
		e.Native = n
		SetAttribute(e, "src", src)
		SetAttribute(e, "alt", name)
		SetAttribute(e, "id", imgid)
		return e
	}, AllowTooltip)(altname, id, options...)
}

var NewAudio = Elements.NewConstructor("audio", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)
	e = enableClasses(e)

	htmlAudio := js.Global().Get("document").Call("createElement", "audio")

	n := NewNativeElementWrapper(htmlAudio)
	e.Native = n
	SetAttribute(e, "name", name)
	SetAttribute(e, "id", id)
	return e
}, AllowTooltip)

var NewVideo = Elements.NewConstructor("video", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)
	e = enableClasses(e)

	htmlVideo := js.Global().Get("document").Call("createElement", "video")
	SetAttribute(e, "name", name)
	SetAttribute(e, "id", id)

	n := NewNativeElementWrapper(htmlVideo)
	e.Native = n
	return e
}, AllowTooltip)

var NewMediaSource = func(src string, typ string, options ...string) *ui.Element {
	return Elements.NewConstructor("source", func(name string, id string) *ui.Element {
		e := ui.NewElement(name, id, Elements.DocType)
		e = enableClasses(e)

		htmlVideo := js.Global().Get("document").Call("createElement", "video")

		n := NewNativeElementWrapper(htmlVideo)
		e.Native = n
		SetAttribute(e, "type", name)
		SetAttribute(e, "src", id)
		return e
	}, AllowTooltip)(typ, src, options...)
}

/* Convenience function

func WithSources(sources ...*ui.Element) func(*ui.Element) *ui.Element { // TODO
	return func(mediaplayer *ui.Element) *ui.Element {
		for _, source := range sources {
			if source.Name != "source" {
				log.Print("cannot append non media source element to mediaplayer")
				continue
			}
			mediaplayer.AppendChild(source)
		}
		return mediaplayer
	}
}
*/

// NewTextNode creates a text node for the Element whose id is passed as argument
// The id for the text Element is the id of its parent to which
// is suffixed "-txt-" and a random number.
// If the parent does not exist, a parent span is created whose id is the one
// passed as argument.
// To change the value of the text, one would Set the "text" property belonging
// to the "data" category/namespace. i.e. Set("data","text",value)
func NewTextNode() *ui.Element {
	var TextNode = Elements.NewConstructor("text", func(name string, id string) *ui.Element {
		e := ui.NewElement(name, id, Elements.DocType)
		htmlTextNode := js.Global().Get("document").Call("createTextNode", "")
		n := NewNativeElementWrapper(htmlTextNode)
		e.Native = n

		e.Watch("data", "text", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			if s, ok := evt.NewValue().(ui.String); ok { // if data.text is deleted, nothing happens, so no check for nil of  evt.NewValue()
				htmlTextNode.Set("nodeValue", s)
			}
			return false
		}))

		return e
	})
	return TextNode("textnode", NewID())
}

// NewTemplatedText returns either a textnode appended to the Element whose id
// is passed as argument, or a div wrapping a textnode if no ui.Element exists
// yet for the id.
// The template accepts a parameterized string as would be accepted by fmt.Sprint
// and the parameter should have their names passed as arguments.
// Done correctly, calling element.Set("data", paramname, stringvalue) will
// set the textnode with a new string value where the parameter whose name is
// `paramname` is set with the value `stringvalue`.
var NewTemplatedText = func(name string, id string, format string, paramsNames ...string) *ui.Element {
	nt := NewTextNode()

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
			s, ok := evt.NewValue().(ui.String)
			if ok {
				params[i] = string(s)
			} else {
				params[i] = "???"
			}

			nt.Set("data", "text", ui.String(formatter(format, params...)), false)
			return false
		}))
	}
	return nt
}

var NewList = func(name string, id string, options ...string) *ui.Element {
	return Elements.NewConstructor("ul", func(ename, eid string) *ui.Element {
		e := ui.NewElement(ename, eid, Elements.DocType)
		e = enableClasses(e)

		htmlList := js.Global().Get("document").Call("createElement", "ul")

		n := NewNativeElementWrapper(htmlList)
		e.Native = n
		SetAttribute(e, "name", ename)
		SetAttribute(e, "id", eid)
		return e
	}, AllowListAutoSync)(name, id, options...)
}

var NewOrderedList = func(name string, id string, typ string, numberingstart int, options ...string) *ui.Element {
	return Elements.NewConstructor("ol", func(ename, eid string) *ui.Element {
		e := ui.NewElement(ename, eid, Elements.DocType)
		e = enableClasses(e)

		htmlList := js.Global().Get("document").Call("createElement", "ol")

		n := NewNativeElementWrapper(htmlList)
		e.Native = n
		SetAttribute(e, "name", ename)
		SetAttribute(e, "id", eid)
		SetAttribute(e, "type", typ)
		SetAttribute(e, "start", strconv.Itoa(numberingstart))
		return e
	}, AllowListAutoSync)(name, id, options...)
}

var NewListItem = Elements.NewConstructor("listitem", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)
	e = enableClasses(e)

	htmlListItem := js.Global().Get("document").Call("createElement", "li")

	n := NewNativeElementWrapper(htmlListItem)
	e.Native = n
	SetAttribute(e, "name", name)
	SetAttribute(e, "id", id) // TODO define attribute setters optional functions

	ondatamutation := ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		cat := evt.Type()

		if cat != "data" {
			return false
		}
		propname := evt.ObservedKey()

		if propname != "content" {
			return false
		}
		evt.Origin().Set("ui", propname, evt.NewValue(), false)
		return false
	})
	e.Watch("data", "content", e, ondatamutation)

	onuimutation := ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		cat := evt.Type()
		if cat != "ui" {
			return true
		}

		propname := evt.ObservedKey()
		if propname != "content" {
			return true
		}

		// we apply the modifications to the UI
		v := evt.NewValue()
		item, ok := v.(*ui.Element)
		if !ok {
			str, ok := v.(ui.String)
			if !ok {
				return true
			}
			item = NewTextNode()
			item.Set("data", "text", str, false)
		}
		// evt.Origin().RemoveChildren().AppendChild(item)
		evt.Origin().Set("ui", "command", ui.RemoveChildrenCommand())
		evt.Origin().Set("ui", "command", ui.AppendChildCommand(item))
		return false
	})
	e.Watch("ui", "content", e, onuimutation)
	return e
}, AllowTooltip)

func newListValue(index int, value ui.Value) ui.Object {
	o := ui.NewObject()
	o.Set("index", ui.Number(index))
	o.Set("value", value)
	return o
}

func DataFromListChange(v ui.Value) (index int, newvalue ui.Value, ok bool) {
	res, ok := v.(ui.Object)
	i, ok := res.Get("index")
	if !ok {
		return -1, nil, false
	}
	idx, ok := i.(ui.Number)
	if !ok {
		return -1, nil, false
	}
	value, ok := res.Get("value")
	if !ok {
		return -1, nil, false
	}
	return int(idx), value, true
}

func ListAppend(list *ui.Element, values ...ui.Value) *ui.Element {
	var backinglist ui.List

	bkglist, ok := list.Get("internals", list.Name)
	if !ok {
		backinglist = ui.NewList()
	}
	backinglist, ok = bkglist.(ui.List)
	if !ok {
		backinglist = ui.NewList()
	}

	length := len(backinglist)

	backinglist = append(backinglist, values...)
	list.Set("internals", list.Name, backinglist, false)
	for i, value := range values {
		list.Set(list.Name, "append", newListValue(i+length, value), false)
	}
	return list
}

func ListPrepend(list *ui.Element, values ...ui.Value) *ui.Element {
	var backinglist ui.List

	bkglist, ok := list.Get("internals", list.Name)
	if !ok {
		backinglist = ui.NewList()
	}
	backinglist, ok = bkglist.(ui.List)
	if !ok {
		backinglist = ui.NewList()
	}

	backinglist = append(values, backinglist...)
	list.Set("internals", list.Name, backinglist, false)
	for i := len(values) - 1; i >= 0; i-- {
		list.Set(list.Name, "prepend", newListValue(i, values[i]), false)
	}
	return list
}

func ListInsertAt(list *ui.Element, offset int, values ...ui.Value) *ui.Element {
	var backinglist ui.List

	bkglist, ok := list.Get("internals", list.Name)
	if !ok {
		backinglist = ui.NewList()
	}
	backinglist, ok = bkglist.(ui.List)
	if !ok {
		backinglist = ui.NewList()
	}

	length := len(backinglist)
	if offset >= length || offset <= 0 {
		log.Print("Cannot insert element in list at that position.")
		return list
	}

	nel := ui.NewList(backinglist[:offset]...)
	nel = append(nel, values...)
	nel = append(nel, backinglist[offset:]...)
	backinglist = nel
	list.Set("internals", list.Name, backinglist, false)
	for i, value := range values {
		list.Set(list.Name, "insert", newListValue(offset+i, value), false)
	}
	return list
}

func ListDelete(list *ui.Element, offset int) *ui.Element {
	var backinglist ui.List

	bkglist, ok := list.Get("internals", list.Name)
	if !ok {
		backinglist = ui.NewList()
	}
	backinglist, ok = bkglist.(ui.List)
	if !ok {
		backinglist = ui.NewList()
	}

	length := len(backinglist)
	if offset >= length || offset <= 0 {
		log.Print("Cannot insert element in list at that position.")
		return list
	}
	backinglist = append(backinglist[:offset], backinglist[offset+1:])
	list.Set("internals", list.Name, backinglist, false)
	list.Set(list.Name, "delete", newListValue(offset, nil), false)
	return list
}

// EnableListAutoSync is passed as an optional Argument to a list constructor call in
// order to trigger list autosyncing.
// When a list is autosyncing, any modification to the list (item adjunction, deletion, modification)
// will propagate to the User Interface.
// This is a convenience function that enforces the argument list
func EnableListAutoSync() string {
	return "ListAutoSync"
}

// AutoSyncList enables to set a mutation handler which is called each time
// a change occurs in the chosen namespace/category of a list Element.
var AllowListAutoSync = ui.NewConstructorOption("ListAutoSync", func(e *ui.Element) *ui.Element {
	h := ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		i, v, ok := DataFromListChange(evt.NewValue())
		if !ok {
			return true
		}

		if evt.ObservedKey() == "append" {
			id := NewID()
			n := NewListItem(evt.Origin().Name+"-item", id)
			item, ok := v.(*ui.Element)
			if !ok {
				str, ok := v.(ui.String)
				if !ok {
					return true
				}
				item = NewTextNode()
				item.Set("data", "text", str, false)
			}
			n.Set("data", "content", item, false)

			// evt.Origin().AppendChild(n)
			evt.Origin().Set("ui", "command", ui.AppendChildCommand(n), false)
		}

		if evt.ObservedKey() == "prepend" {
			id := NewID()
			n := NewListItem(evt.Origin().Name+"-item", id)
			item, ok := v.(*ui.Element)
			if !ok {
				str, ok := v.(ui.String)
				if !ok {
					return true
				}
				item = NewTextNode()
				item.Set("data", "text", str, false)
			}
			n.Set("data", "content", item, false)

			// evt.Origin().PrependChild(n)
			evt.Origin().Set("ui", "command", ui.PrependChildCommand(n), false)
		}

		if evt.ObservedKey() == "insert" {
			id := NewID()
			n := NewListItem(evt.Origin().Name+"-item", id)
			item, ok := v.(*ui.Element)
			if !ok {
				str, ok := v.(ui.String)
				if !ok {
					return true
				}
				item = NewTextNode()
				item.Set("data", "text", str, false)
			}
			n.Set("data", "content", item, false)

			// evt.Origin().InsertChild(n, i)
			evt.Origin().Set("ui", "command", ui.InsertChildCommand(n, i), false)
		}

		if evt.ObservedKey() == "delete" {
			target := evt.Origin()
			deletee := target.Children.AtIndex(i)
			if deletee != nil {
				// target.RemoveChild(deletee)
				target.Set("ui", "command", ui.RemoveChildCommand(deletee))
			}
		}
		return false
	})

	e.WatchGroup(e.Name, e, h)
	return e
})

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
			cb.Release()
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
		nelmt.JSValue().Call("dispatchEvent", nativeevent)
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
		c, ok := classes.(ui.String)
		if !ok {
			target.Set(category, "class", ui.String(classname), false)
			return
		}
		sc := string(c)
		if !strings.Contains(sc, classname) {
			sc = sc + " " + classname
			target.Set(category, "class", ui.String(sc), false)
		}
		return
	}
	target.Set(category, "class", ui.String(classname), false)
}

func RemoveClass(target *ui.Element, classname string) {
	category := "css"
	classes, ok := target.Get(category, "class")
	if !ok {
		return
	}
	rc, ok := classes.(ui.String)
	if !ok {
		return
	}
	c := string(rc)
	c = strings.TrimPrefix(c, classname)
	c = strings.TrimPrefix(c, " ")
	c = strings.ReplaceAll(c, classname+" ", " ")
	target.Set(category, "class", ui.String(c), false)
}

func Classes(target *ui.Element) []string {
	category := "css"
	classes, ok := target.Get(category, "class")
	if !ok {
		return nil
	}
	c, ok := classes.(ui.String)
	if !ok {
		return nil
	}
	return strings.Split(string(c), " ")
}

func enableClasses(e *ui.Element) *ui.Element {
	h := ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		target := evt.Origin()
		native, ok := target.Native.(NativeElement)
		if !ok {
			log.Print("wrong type for native element or native element does not exist")
			return true
		}
		classes, ok := evt.NewValue().(ui.String)
		if !ok {
			log.Print("new value of non-string type. Unable to use as css class(es)")
			return true
		}
		native.JSValue().Call("setAttribute", "class", classes)
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
	return native.JSValue().Call("getAttribute", "name").String()
}

func SetAttribute(target *ui.Element, name string, value string) {
	target.Set("attrs", name, ui.String(value), false)
	native, ok := target.Native.(NativeElement)
	if !ok {
		log.Print("Cannot set Attribute on non-expected wrapper type")
		return
	}
	native.JSValue().Call("setAttribute", name, value)
}

func RemoveAttribute(target *ui.Element, name string) {
	target.Delete("attrs", name)
	native, ok := target.Native.(NativeElement)
	if !ok {
		log.Print("Cannot delete Attribute using non-expected wrapper type")
		return
	}
	native.JSValue().Call("removeAttribute", name)
}
