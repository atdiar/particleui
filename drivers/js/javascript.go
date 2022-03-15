// +build js,wasm

// Package doc defines the default set of Element constructors, native interfaces,
// events and event handlers, and animation properties used to build js-based UIs.
package doc

import (
	"encoding/json"
	//"errors"
	"io"
	"log"
	"strconv"
	"strings"
	"syscall/js"
	"time"

	"github.com/atdiar/particleui"
	"golang.org/x/net/html"
)

var (
	// DOCTYPE holds the document doctype.
	DOCTYPE = "html/js"
	// Elements stores wasm-generated HTML ui.Element constructors.
	Elements                      = ui.NewElementStore("default", DOCTYPE).AddPersistenceMode("sessionstorage", loadfromsession, sessionstorefn).AddPersistenceMode("localstorage", loadfromlocalstorage, localstoragefn)
	EnablePropertyAutoInheritance = ui.EnablePropertyAutoInheritance
)

var NewID = ui.NewIDgenerator(time.Now().UnixNano())

// mutationCaptureMode describes how a Go App may capture textarea value changes
// that happen in native javascript. For instance, when a blur event is dispatched
// or when any mutation is observed via the MutationObserver API.
type mutationCaptureMode int

const (
	onBlur mutationCaptureMode = iota
	onInput
)

type jsStore struct {
	store js.Value
}

func (s jsStore) Get(key string) (js.Value, bool) {
	v := s.store.Call("getItem", key)
	if !v.Truthy() {
		return v, false
	}
	return v, true
}

func (s jsStore) Set(key string, value js.Value) {
	JSON := js.Global().Get("JSON")
	res := JSON.Call("stringify", value)
	s.store.Call("setItem", key, res)
}

// Let's add sessionstorage and localstorage for Element properties.
// For example, an Element which would have been created with the sessionstorage option
// would have every set properties stored in sessionstorage, available for
// later recovery. It enables to have data that persists runs and loads of a
// web app.

func storer(s string) func(element *ui.Element, category string, propname string, value ui.Value, flags ...bool) {
	return func(element *ui.Element, category string, propname string, value ui.Value, flags ...bool) {
		store := jsStore{js.Global().Get(s)}
		categoryExists := element.Properties.HasCategory(category)
		propertyExists := element.Properties.HasProperty(category, propname)

		// log.Print("CALL TO STORE: ", category, categoryExists, propname, propertyExists) // DEBUG

		// Let's check whether the element exists ins store. In the negative case,
		// we can act as if no category has been registered.
		// Every indices need to be generated and stored.
		var indexed bool

		storedcategories, ok := element.Get("index", "categories")
		if ok {
			sc, ok := storedcategories.(ui.List)
			if ok {
				for _, cat := range sc {
					catstr, ok := cat.(ui.String)
					if !ok {
						indexed = false
						break
					}
					if string(catstr) != category {
						continue
					}
					indexed = true
					break
				}
			}
		}
		if !(category == "index" && propname == "categories") {
			if !indexed {
				catlist := ui.NewList()
				for k := range element.Properties.Categories {
					catlist = append(catlist, ui.String(k))
				}
				if !categoryExists {
					catlist = append(catlist, ui.String(category))
				}
				catlist = append(catlist, ui.String("index"))
				// log.Print("indexed catlist", catlist) // DEBUG
				element.Set("index", "categories", catlist)
			}
		}

		if !categoryExists || !indexed {
			categories := make([]interface{}, 0, len(element.Properties.Categories)+1)

			for k := range element.Properties.Categories {
				categories = append(categories, k)
			}
			if !categoryExists {
				categories = append(categories, category)
			}
			v := js.ValueOf(categories)
			store.Set(element.ID, v)
		}
		proptype := "Local"
		if len(flags) > 0 {
			if flags[0] {
				proptype = "Inheritable"
			}
		}

		if !propertyExists || !indexed {
			props := make([]interface{}, 0, 1)
			c, ok := element.Properties.Categories[category]
			if !ok {
				props = append(props, proptype+"/"+propname)
				v := js.ValueOf(props)
				store.Set(element.ID+"/"+category, v)
			} else {
				for k := range c.Default {
					props = append(props, "Default/"+k)
				}
				for k := range c.Inherited {
					props = append(props, "Inherited/"+k)
				}
				for k := range c.Local {
					props = append(props, "Local/"+k)
				}
				for k := range c.Inheritable {
					props = append(props, "Inheritable/"+k)
				}

				props = append(props, proptype+"/"+propname)
				// log.Print("all props stored...", props) // DEBUG
				v := js.ValueOf(props)
				store.Set(element.ID+"/"+category, v)
			}
		}
		item := value.RawValue()
		v := stringify(item)
		store.Set(element.ID+"/"+category+"/"+propname, js.ValueOf(v))
		return
	}
}

/*func stringify(v interface{}) js.Value {
	defer func() {
		if r := recover(); r != nil {
			log.Print(v)
			log.Print(r)
		}
	}()
	res := js.ValueOf(v)
	return res
}*/
func stringify(v interface{}) string {
	res, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(res)
}

var sessionstorefn = storer("sessionStorage")
var localstoragefn = storer("localStorage")

func loader(s string) func(e *ui.Element) error {
	return func(e *ui.Element) error {
		store := jsStore{js.Global().Get(s)}
		id := e.ID

		// Let's retrieve the category index for this element, if it exists in the sessionstore
		jsoncategories, ok := store.Get(id)
		if !ok {
			return nil // Not necessarily an error in the general case. element just does not exist in store
		}

		categories := make([]string, 0)
		properties := make([]string, 0)
		err := json.Unmarshal([]byte(jsoncategories.String()), &categories)
		if err != nil {
			return err
		}
		//log.Print(categories, properties) //DEBUG
		for _, category := range categories {
			jsonproperties, ok := store.Get(e.ID + "/" + category)
			if !ok {
				continue
			}
			err = json.Unmarshal([]byte(jsonproperties.String()), &properties)
			if err != nil {
				log.Print(err)
				return err
			}

			for _, property := range properties {
				// let's retrieve the propname (it is suffixed by the proptype)
				// then we can retrieve the value
				// log.Print("debug...", category, property) // DEBUG
				proptypename := strings.Split(property, "/")
				proptype := proptypename[0]
				propname := proptypename[1]
				jsonvalue, ok := store.Get(e.ID + "/" + category + "/" + propname)
				if ok {
					var rawvaluemapstring string
					err = json.Unmarshal([]byte(jsonvalue.String()), &rawvaluemapstring)
					if err != nil {
						return err
					}
					rawvalue := ui.NewObject()
					err = json.Unmarshal([]byte(rawvaluemapstring), &rawvalue)
					if err != nil {
						return err
					}
					ui.LoadProperty(e, category, propname, proptype, rawvalue.Value())
					//log.Print("LOADED PROPMAP: ", e.Properties, category, propname, rawvalue.Value()) // DEBUG
				}
			}
		}
		return nil
	}
}

var loadfromsession = loader("sessionStorage")
var loadfromlocalstorage = loader("localStorage")

// Window is a ype that represents a browser window
type Window struct {
	UIElement ui.BasicElement
}

func (w Window) AsBasicElement() ui.BasicElement {
	return w.UIElement
}

func (w Window) AsElement() *ui.Element {
	return w.UIElement.AsElement()
}

func (w Window) SetTitle(title string) {
	w.AsBasicElement().AsElement().Set("ui", "title", ui.String(title))
}

// TODO see if can get height width of window view port, etc.

func newWindow(title string, options ...string) Window {
	c := Elements.NewConstructor("window", func(name string, id string) *ui.Element {
		e := ui.NewElement("window", name, DOCTYPE)
		e.Set("event", "mounted", ui.Bool(true))
		e.Set("event", "attached", ui.Bool(true))
		e.ElementStore = Elements
		wd := js.Global().Get("document").Get("defaultView")
		if !wd.Truthy() {
			panic("unable to access windows")
		}
		e.Native = NewNativeElementWrapper(wd)

		h := ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			target := evt.Origin()
			newtitle, ok := evt.NewValue().(ui.String)
			if !ok {
				return true
			}

			if target != e {
				return true
			}
			nat, ok := target.Native.(NativeElement)
			if !ok {
				return true
			}
			jswindow := nat.Value
			if !jswindow.Truthy() {
				log.Print("Unable to access native Window object")
				return true
			}
			jswindow.Get("document").Set("title", string(newtitle))
			return false
		})

		e.Watch("ui", "title", e, h)
		e.Set("ui", "title", ui.String(title), false)

		return e
	})

	return Window{ui.BasicElement{LoadElement(c("window", "window", options...))}}
}

func GetWindow(options ...string) Window {
	w := Elements.GetByID("window")
	if w == nil {
		return newWindow("Powered by ParticleUI", options...)
	}
	cname, ok := w.Get("internals", "constructor")
	if !ok {
		return newWindow("Powered by ParticleUI", options...)
	}
	nname, ok := cname.(ui.String)
	if !ok {
		return newWindow("Powered by ParticleUI", options...)
	}
	if string(nname) != "window" {
		log.Print("There is a UI Element whose id is similar to the Window name. This is incorrect.")
		return Window{}
	}
	return Window{ui.BasicElement{w}}
}

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
		log.Print("wrong format for native element underlying objects.Cannot remove ", child.Native)
		return
	}
	v.Value.Call("remove")
	//n.Value.Call("removeChild", v.Value)

}

// JSValue retrieves the js.Value corresponding to the Element submmitted as
// argument.
func JSValue(e *ui.Element) js.Value {
	n, ok := e.Native.(NativeElement)
	if !ok {
		panic("js.Value not wrapped in NativeElement type")
	}
	return n.Value
}

// SetInnerHTML sets the innerHTML property of HTML elements.
// Please note that it is unsafe to sets client submittd HTML inputs.
func SetInnerHTML(e *ui.Element, html string) *ui.Element {
	jsv := JSValue(e)
	jsv.Set("innerHTML", html)
	return e
}

/*
//
//
// Element Constructors
//
//
//
*/

// AllowSessionStoragePersistence is a constructor option. When passed as argument in
// the creation of a ui.Element constructor, it allows for ui.Element constructors to
// different options for property persistence.
var AllowSessionStoragePersistence = ui.NewConstructorOption("sessionstorage", func(e *ui.Element) *ui.Element {
	e.Set("internals", "persistence", ui.String("sessionstorage"))
	return e
})

var AllowAppLocalStoragePersistence = ui.NewConstructorOption("localstorage", func(e *ui.Element) *ui.Element {
	e.Set("internals", "persistence", ui.String("localstorage"))
	return e
})

func EnableSessionPersistence() string {
	return "sessionstorage"
}

func EnableLocalPersistence() string {
	return "localstorage"
}

type Document struct {
	ui.BasicElement
}

func (d Document) Render(w io.Writer) error {
	return html.Render(w, NewHTMLTree(d))
}

// NewDocument returns the root of new js app. It is the top-most element
// in the tree of Elements that consitute the full document.
// It should be the element which is passed to a router to observe for route
// change.
// By default, it represents document.body. As such, it is different from the
// DOM which holds the head element for instance.
func NewDocument(id string, options ...string) Document {
	var newDocument = Elements.NewConstructor("root", func(name string, id string) *ui.Element {

		e := Elements.NewAppRoot(id).AsElement()

		root := js.Global().Get("document").Get("body")
		if !root.Truthy() {
			log.Print("failed to instantiate root element for the document")
			return e
		}
		n := NewNativeElementWrapper(root)
		e.Native = n
		SetAttribute(e, "id", id)

		e.Watch("ui", "redirectroute", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			v := evt.NewValue()
			nroute, ok := v.(ui.String)
			if !ok {
				panic(nroute)
			}
			route := string(nroute)
			js.Global().Get("history").Call("replaceState", "{}", "", route)

			e.SyncUISetData("currentroute", v)
			return false
		}))

		e.Watch("ui", "currentroute", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			v := evt.NewValue()
			nroute, ok := v.(ui.String)
			if !ok {
				panic(nroute)
			}
			route := string(nroute)
			js.Global().Get("history").Call("pushState", "{}", "", route)
			return false
		}))

		e.Watch("navigation", "ready", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			route := js.Global().Get("location").Get("pathname").String()
			log.Println("init", route) //DEBUG
			e.Set("navigation", "routechangerequest", ui.String(route))
			return false
		}))

		return e
	}, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

	return Document{ui.BasicElement{LoadElement(newDocument(id, id, options...))}}
}

// reset is used to delete all eventlisteners froman Element
func reset(element js.Value) js.Value {
	clone := element.Call("cloneNode")
	parent := element.Get("parentNode")
	if !parent.IsNull() {
		element.Call("replaceWith", clone)
	}
	return clone
}

// Div is a concrete type that holds the common interface to Div *ui.Element objects.
// i.e. ui.Element whose constructor name is "div" and represents html div elements.
type Div struct {
	ui.BasicElement
}

func (d Div) Contenteditable(b bool) Div {
	d.AsElement().SetDataSetUI("contenteditable", ui.Bool(b))
	return d
}

func (d Div) SetText(str string) Div {
	d.AsElement().SetDataSetUI("text", ui.String(str))
	return d
}

// NewDiv is a constructor for html div elements.
// The name constructor argument is used by the framework for automatic route
// and automatic link generation.
func NewDiv(name string, id string, options ...string) Div {
	var newDiv = Elements.NewConstructor("div", func(name string, id string) *ui.Element {
		e := ui.NewElement(name, id, Elements.DocType)
		e = enableClasses(e)

		htmlDiv := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlDiv.IsNull()

		// Also, need to try and load any corresponding properties that would have been persisted and retrigger ui.mutations to recover ui state.
		// Let's defer this to the persistence option so that the loading function of the right persistent storage is used.
		if !exist {
			htmlDiv = js.Global().Get("document").Call("createElement", "div")
		} else {
			htmlDiv = reset(htmlDiv)
		}

		n := NewNativeElementWrapper(htmlDiv)
		e.Native = n
		if !exist {
			SetAttribute(e, "id", id)
		}
		e.Watch("ui", "contenteditable", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			b, ok := evt.NewValue().(ui.Bool)
			if !ok {
				return true
			}
			if bool(b) {
				SetAttribute(evt.Origin(), "contenteditable", "")
			}
			return false
		}))

		e.Watch("ui", "text", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			str, ok := evt.NewValue().(ui.String)
			if !ok {
				return true
			}
			htmlDiv.Set("textContent", string(str))

			return false
		}))

		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

	return Div{ui.BasicElement{LoadElement(newDiv(name, id, options...))}}
}

// LoadElement will load Element properties.
func LoadElement(d *ui.Element) *ui.Element {
	pmode := ui.PersistenceMode(d)
	storage, ok := d.ElementStore.PersistentStorer[pmode]
	if ok {
		err := storage.Load(d)
		if err != nil {
			log.Print(err)
		}
	}
	return d
}

// Tooltip defines the type implementing the interface of a tooltip ui.Element.
// The default ui.Element interface is reachable via a call to the   AsBasicElement() method.
type Tooltip struct {
	ui.BasicElement
}

// SetContent sets the content of the tooltip.
func (t Tooltip) SetContent(content ui.BasicElement) Tooltip {
	t.AsElement().SetData("content", content.AsElement())
	return t
}

// SetContent sets the content of the tooltip.
func (t Tooltip) SetText(content string) Tooltip {
	t.AsElement().SetData("content", ui.String(content))
	return t
}

var tooltipConstructor = Elements.NewConstructor("tooltip", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)
	e.Set("internals", "tag", ui.String("div"))
	e = enableClasses(e)

	htmlTooltip := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlTooltip.IsNull()

	if !exist {
		htmlTooltip = js.Global().Get("document").Call("createElement", "div")
	} else {
		htmlTooltip = reset(htmlTooltip)
	}

	n := NewNativeElementWrapper(htmlTooltip)
	e.Native = n
	SetAttribute(e, "id", id)
	AddClass(e, "tooltip")

	h := ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		content, ok := evt.NewValue().(*ui.Element)
		if ok {
			tooltip := evt.Origin()

			tooltip.AsElement().SetChildren(ui.BasicElement{content})

			return false
		}
		strcontent, ok := evt.NewValue().(ui.String)
		if !ok {
			return true
		}

		tooltip := evt.Origin()
		tooltip.RemoveChildren()

		htmlTooltip.Set("textContent", strcontent)

		return false
	})
	e.Watch("data", "content", e, h)

	return e
}, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func HasTooltip(target *ui.Element) (Tooltip, bool) {
	v := target.ElementStore.GetByID(target.ID + "-tooltip")
	if v == nil {
		return Tooltip{ui.BasicElement{v}}, false
	}
	return Tooltip{ui.BasicElement{v}}, true
}

// EnableTooltip, when passed to a constructor which has the AllowTooltip option,
// creates a tootltip html div element (for a given target ui.Element)
// The content of the tooltip can be directly set by  specifying a value for
// the ("data","content") (category,propertyname) Element datastore entry.
// The content value can be a string or another ui.Element.
// The content of the tooltip can also be set by modifying the ("tooltip","content")
// property
func EnableTooltip() string {
	return "AllowTooltip"
}

var AllowTooltip = ui.NewConstructorOption("AllowTooltip", func(target *ui.Element) *ui.Element {
	e := LoadElement(tooltipConstructor(target.Name+"/tooltip", target.ID+"-tooltip"))
	// Let's observe the target element which owns the tooltip too so that we can
	// change the tooltip automatically from there.
	h := ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		e.Set("data", "content", evt.NewValue(), false)
		return false
	})
	target.Watch("tooltip", "content", target, h)

	target.AppendChild(ui.BasicElement{e})

	return target
})

type TextArea struct {
	ui.BasicElement
}

func (t TextArea) Text() string {
	v, ok := t.AsElement().GetData("text")
	if !ok {
		return ""
	}
	text, ok := v.(ui.String)
	if !ok {
		return ""
	}
	return string(text)
}

func (t TextArea) SetText(text string) TextArea {
	t.AsElement().SetDataSetUI("text", ui.String(text))
	return t
}

func (t TextArea) SetColumns(i int) TextArea {
	t.AsElement().SetDataSetUI("cols", ui.Number(i))
	return t
}

func (t TextArea) SetRows(i int) TextArea {
	t.AsElement().SetDataSetUI("rows", ui.Number(i))
	return t
}

// NewTextArea is a constructor for a textarea html element.
func NewTextArea(name string, id string, rows int, cols int, options ...string) TextArea {
	t := Elements.NewConstructor("textarea", func(ename string, eid string) *ui.Element {
		e := ui.NewElement(ename, eid, Elements.DocType)
		e = enableClasses(e)

		htmlTextArea := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlTextArea.IsNull()

		if !exist {
			htmlTextArea = js.Global().Get("document").Call("createElement", "textarea")
		} else {
			htmlTextArea = reset(htmlTextArea)
		}

		e.Watch("ui", "text", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			if s, ok := evt.NewValue().(ui.String); ok {
				old := htmlTextArea.Get("value").String()
				if string(s) != old {
					SetAttribute(evt.Origin(), "value", string(s))
				}
			}
			return false
		}))

		e.Watch("ui", "rows", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			if n, ok := evt.NewValue().(ui.Number); ok {
				SetAttribute(e, "rows", strconv.Itoa(int(n)))
				return false
			}
			return true
		}))

		e.Watch("ui", "cols", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			if n, ok := evt.NewValue().(ui.Number); ok {
				SetAttribute(e, "rows", strconv.Itoa(int(n)))
				return false
			}
			return true
		}))

		n := NewNativeElementWrapper(htmlTextArea)
		e.Native = n
		//SetAttribute(e, "name", ename)
		SetAttribute(e, "id", eid)
		e.SetDataSetUI("rows", ui.String(strconv.Itoa(rows)))
		e.SetDataSetUI("cols", ui.String(strconv.Itoa(cols)))
		return e
	}, allowTextAreaDataBindingOnBlur, allowTextAreaDataBindingOnInput, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return TextArea{ui.BasicElement{LoadElement(t(name, id, options...))}}
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

// EnableaSyncOnBlur returns the name of the option that can be passed to
// textarea ui.Element constructor to trigger two-way databinding on textarea
// blur event.
func EnableSyncOnBlur() string {
	return "SyncOnBlur"
}

// EnableSyncOnInput returns the name of the option that can be passed to
// textarea ui.Element constructor to trigger two-way databinding on textarea
// input event.
func EnableSyncOnInput() string {
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
			nn := n.Value
			v := nn.Get("value")
			ok = v.Truthy()
			if !ok {
				return true
			}
			s := v.String()
			e.SyncUISetData("text", ui.String(s), false)
			return false
		})

		if datacapturemode == nil || len(datacapturemode) > 1 {
			e.AddEventListener("blur", callback, NativeEventBridge)
			return e
		}
		mode := datacapturemode[0]
		if mode == onInput {
			e.AddEventListener("input", callback, NativeEventBridge)
			return e
		}

		// capture textarea value on blur by default
		e.AddEventListener("blur", callback, NativeEventBridge)
		return e
	}
}

type Header struct {
	ui.BasicElement
}

// NewHeader is a constructor for a html header element.
func NewHeader(name string, id string, options ...string) Header {
	c := Elements.NewConstructor("header", func(name string, id string) *ui.Element {
		e := ui.NewElement(name, id, Elements.DocType)
		e = enableClasses(e)

		htmlHeader := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlHeader.IsNull()

		if !exist {
			htmlHeader = js.Global().Get("document").Call("createElement", "header")
		} else {
			htmlHeader = reset(htmlHeader)
		}

		n := NewNativeElementWrapper(htmlHeader)
		e.Native = n
		if !exist {
			SetAttribute(e, "id", id)
		}
		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return Header{ui.BasicElement{LoadElement(c(name, id, options...))}}
}

type Footer struct {
	ui.BasicElement
}

// NewFooter is a constructor for an html footer element.
func NewFooter(name string, id string, options ...string) Footer {
	c := Elements.NewConstructor("footer", func(name string, id string) *ui.Element {
		e := ui.NewElement(name, id, Elements.DocType)
		e = enableClasses(e)

		htmlFooter := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlFooter.IsNull()

		if !exist {
			htmlFooter = js.Global().Get("document").Call("createElement", "footer")
		} else {
			htmlFooter = reset(htmlFooter)
		}

		n := NewNativeElementWrapper(htmlFooter)
		e.Native = n

		if !exist {
			SetAttribute(e, "id", id)
		}

		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return Footer{ui.BasicElement{LoadElement(c(name, id, options...))}}
}

type Section struct {
	ui.BasicElement
}

// NewSection is a constructor for html section elements.
func NewSection(name string, id string, options ...string) Section {
	c := Elements.NewConstructor("section", func(name string, id string) *ui.Element {
		e := ui.NewElement(name, id, Elements.DocType)
		e = enableClasses(e)

		htmlSection := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlSection.IsNull()
		if !exist {
			htmlSection = js.Global().Get("document").Call("createElement", "section")
		} else {
			htmlSection = reset(htmlSection)
		}

		n := NewNativeElementWrapper(htmlSection)
		e.Native = n
		if !exist {
			SetAttribute(e, "id", id)
		}
		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return Section{ui.BasicElement{LoadElement(c(name, id, options...))}}
}

type H1 struct {
	ui.BasicElement
}

func (h H1) SetText(s string) H1 {
	h.AsElement().SetDataSetUI("text", ui.String(s))
	return h
}

// NewH1 is a constructor for html heading H1 elements.
func NewH1(name string, id string, options ...string) H1 {
	c := Elements.NewConstructor("h1", func(name string, id string) *ui.Element {
		e := ui.NewElement(name, id, Elements.DocType)
		e = enableClasses(e)

		htmlH1 := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlH1.IsNull()
		if !exist {
			htmlH1 = js.Global().Get("document").Call("createElement", "h1")
		} else {
			htmlH1 = reset(htmlH1)
		}

		n := NewNativeElementWrapper(htmlH1)
		e.Native = n
		if !exist {
			SetAttribute(e, "id", id)
		}

		e.Watch("ui", "text", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			str, ok := evt.NewValue().(ui.String)
			if !ok {
				return true
			}
			htmlH1.Set("innerHTML", string(str))

			return false
		}))
		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return H1{ui.BasicElement{LoadElement(c(name, id, options...))}}
}

type H2 struct {
	ui.BasicElement
}

func (h H2) SetText(s string) H2 {
	h.AsElement().SetDataSetUI("text", ui.String(s))
	return h
}

// NewH2 is a constructor for html heading H2 elements.
func NewH2(name string, id string, options ...string) H2 {
	c := Elements.NewConstructor("h2", func(name string, id string) *ui.Element {
		e := ui.NewElement(name, id, Elements.DocType)
		e = enableClasses(e)

		htmlH2 := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlH2.IsNull()
		if !exist {
			htmlH2 = js.Global().Get("document").Call("createElement", "h2")
		} else {
			htmlH2 = reset(htmlH2)
		}

		n := NewNativeElementWrapper(htmlH2)
		e.Native = n
		if !exist {
			SetAttribute(e, "id", id)
		}

		e.Watch("ui", "text", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			str, ok := evt.NewValue().(ui.String)
			if !ok {
				return true
			}
			htmlH2.Set("innerHTML", string(str))

			return false
		}))
		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return H2{ui.BasicElement{LoadElement(c(name, id, options...))}}
}

type H3 struct {
	ui.BasicElement
}

func (h H3) SetText(s string) H3 {
	h.AsElement().SetDataSetUI("text", ui.String(s))
	return h
}

// NewH3 is a constructor for html heading H3 elements.
func NewH3(name string, id string, options ...string) H3 {
	c := Elements.NewConstructor("h3", func(name string, id string) *ui.Element {
		e := ui.NewElement(name, id, Elements.DocType)
		e = enableClasses(e)

		htmlH3 := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlH3.IsNull()
		if !exist {
			htmlH3 = js.Global().Get("document").Call("createElement", "h3")
		} else {
			htmlH3 = reset(htmlH3)
		}

		n := NewNativeElementWrapper(htmlH3)
		e.Native = n
		if !exist {
			SetAttribute(e, "id", id)
		}

		e.Watch("ui", "text", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			str, ok := evt.NewValue().(ui.String)
			if !ok {
				return true
			}
			htmlH3.Set("innerHTML", string(str))

			return false
		}))
		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return H3{ui.BasicElement{LoadElement(c(name, id, options...))}}
}

type H4 struct {
	ui.BasicElement
}

func (h H4) SetText(s string) H4 {
	h.AsElement().SetDataSetUI("text", ui.String(s))
	return h
}

// NewH4 is a constructor for html heading H4 elements.
func NewH4(name string, id string, options ...string) H4 {
	c := Elements.NewConstructor("h4", func(name string, id string) *ui.Element {
		e := ui.NewElement(name, id, Elements.DocType)
		e = enableClasses(e)

		htmlH4 := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlH4.IsNull()
		if !exist {
			htmlH4 = js.Global().Get("document").Call("createElement", "h4")
		} else {
			htmlH4 = reset(htmlH4)
		}

		n := NewNativeElementWrapper(htmlH4)
		e.Native = n
		if !exist {
			SetAttribute(e, "id", id)
		}

		e.Watch("ui", "text", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			str, ok := evt.NewValue().(ui.String)
			if !ok {
				return true
			}
			htmlH4.Set("innerHTML", string(str))

			return false
		}))
		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return H4{ui.BasicElement{LoadElement(c(name, id, options...))}}
}

type H5 struct {
	ui.BasicElement
}

func (h H5) SetText(s string) H5 {
	h.AsElement().SetDataSetUI("text", ui.String(s))
	return h
}

// NewH5 is a constructor for html heading H5 elements.
func NewH5(name string, id string, options ...string) H5 {
	c := Elements.NewConstructor("h5", func(name string, id string) *ui.Element {
		e := ui.NewElement(name, id, Elements.DocType)
		e = enableClasses(e)

		htmlH5 := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlH5.IsNull()
		if !exist {
			htmlH5 = js.Global().Get("document").Call("createElement", "h5")
		} else {
			htmlH5 = reset(htmlH5)
		}

		n := NewNativeElementWrapper(htmlH5)
		e.Native = n
		if !exist {
			SetAttribute(e, "id", id)
		}

		e.Watch("ui", "text", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			str, ok := evt.NewValue().(ui.String)
			if !ok {
				return true
			}
			htmlH5.Set("innerHTML", string(str))

			return false
		}))
		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return H5{ui.BasicElement{LoadElement(c(name, id, options...))}}
}

type H6 struct {
	ui.BasicElement
}

func (h H6) SetText(s string) H6 {
	h.AsElement().SetDataSetUI("text", ui.String(s))
	return h
}

// NewH6 is a constructor for html heading H6 elements.
func NewH6(name string, id string, options ...string) H6 {
	c := Elements.NewConstructor("h6", func(name string, id string) *ui.Element {
		e := ui.NewElement(name, id, Elements.DocType)
		e = enableClasses(e)

		htmlH6 := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlH6.IsNull()
		if !exist {
			htmlH6 = js.Global().Get("document").Call("createElement", "h6")
		} else {
			htmlH6 = reset(htmlH6)
		}

		n := NewNativeElementWrapper(htmlH6)
		e.Native = n
		if !exist {
			SetAttribute(e, "id", id)
		}

		e.Watch("ui", "text", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			str, ok := evt.NewValue().(ui.String)
			if !ok {
				return true
			}
			htmlH6.Set("innerHTML", string(str))

			return false
		}))
		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return H6{ui.BasicElement{LoadElement(c(name, id, options...))}}
}

type Span struct {
	ui.BasicElement
}

func (s Span) SetText(str string) Span {
	s.AsElement().SetDataSetUI("text", ui.String(str))
	return s
}

// NewSpan is a constructor for html span elements.
func NewSpan(name string, id string, options ...string) Span {
	c := Elements.NewConstructor("span", func(name string, id string) *ui.Element {
		e := ui.NewElement(name, id, Elements.DocType)
		e = enableClasses(e)

		htmlSpan := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlSpan.IsNull()
		if !exist {
			htmlSpan = js.Global().Get("document").Call("createElement", "span")
		} else {
			htmlSpan = reset(htmlSpan)
		}

		n := NewNativeElementWrapper(htmlSpan)
		e.Native = n

		e.Watch("ui", "text", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			rawstr, ok := evt.NewValue().(ui.String)
			if !ok {
				return true
			}
			htmlSpan.Set("textContent", string(rawstr))
			return false
		}))

		if !exist {
			SetAttribute(e, "id", id)
		}
		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return Span{ui.BasicElement{LoadElement(c(name, id, options...))}}
}

type Paragraph struct {
	ui.BasicElement
}

func (p Paragraph) SetText(s string) Paragraph {
	p.AsElement().SetDataSetUI("text", ui.String(s))
	return p
}

// NewParagraph is a constructor for html paragraph elements.
func NewParagraph(name string, id string, options ...string) Paragraph {
	c := Elements.NewConstructor("p", func(name string, id string) *ui.Element {
		e := ui.NewElement(name, id, Elements.DocType)
		e = enableClasses(e)

		htmlParagraph := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlParagraph.IsNull()
		if !exist {
			htmlParagraph = js.Global().Get("document").Call("createElement", "p")
		} else {
			htmlParagraph = reset(htmlParagraph)
		}

		n := NewNativeElementWrapper(htmlParagraph)
		e.Native = n
		if !exist {
			SetAttribute(e, "id", id)
		}

		e.Watch("ui", "text", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			rawstr, ok := evt.NewValue().(ui.String)
			if !ok {
				return true
			}
			htmlParagraph.Set("innerText", string(rawstr))
			return false
		}))
		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return Paragraph{ui.BasicElement{LoadElement(c(name, id, options...))}}
}

/*type Nav struct {
	UIElement *ui.Element
}

func (n Nav) Element() *ui.Element {
	return n.UIElement
}

func (n Nav) AppendAnchorLink(l Anchor) Nav {
	// TODO append link element
	n.Element().AppendChild(l.Element())
	return n
}

// NewNavMenu is a constructor for a html nav element.
func NewNavMenu(name string, id string, options ...string) Nav {
	c := Elements.NewConstructor("nav", func(name string, id string) *ui.Element {
		e := ui.NewElement(name, id, Elements.DocType)
		e = enableClasses(e)

		htmlNavMenu := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlNavMenu.IsNull()
		if !exist {
			htmlNavMenu = js.Global().Get("document").Call("createElement", "nav")
		}

		n := NewNativeElementWrapper(htmlNavMenu)
		e.Native = n

		if !exist {
			SetAttribute(e, "id", id)
		}

		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return Nav{LoadElement(c(name, id, options...))}
}
*/

type Anchor struct {
	ui.BasicElement
}

func (a Anchor) SetHREF(target string) Anchor {
	a.AsElement().SetDataSetUI("href", ui.String(target))
	return a
}

func (a Anchor) FromLink(link ui.Link) Anchor {
	// Check if link is already verified
	_, ok := link.AsElement().Get("event", "verified")
	if ok {
		a.SetHREF(link.URI())
		log.Print(link.URI(), " test") // DEBUG
	}
	a.AsElement().Watch("event", "verified", link, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		a.SetHREF(link.URI())
		log.Print(link.URI(), " test") // DEBUG
		return false
	}), true)

	a.AsElement().Watch("data", "active", link, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		a.AsElement().SetDataSetUI("active", evt.NewValue())
		return false
	}), true)

	a.AsElement().AddEventListener("click", ui.NewEventHandler(func(evt ui.Event) bool {
		evt.PreventDefault()
		link.Activate()
		return false
	}), NativeEventBridge)

	return a
}

func (a Anchor) SetText(text string) Anchor {
	a.AsElement().SetDataSetUI("text", ui.String(text))
	return a
}

// NewAnchor creates an html anchor element.
func NewAnchor(name string, id string, options ...string) Anchor {
	c := Elements.NewConstructor("a", func(name string, id string) *ui.Element {
		e := ui.NewElement(name, id, Elements.DocType)
		e = enableClasses(e)

		htmlAnchor := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlAnchor.IsNull()
		if !exist {
			htmlAnchor = js.Global().Get("document").Call("createElement", "a")
		} else {
			htmlAnchor = reset(htmlAnchor)
		}

		n := NewNativeElementWrapper(htmlAnchor)
		e.Native = n
		if !exist {
			SetAttribute(e, "id", id)
		}

		e.Watch("ui", "href", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			r, ok := evt.NewValue().(ui.String)
			if !ok {
				return true
			}
			SetAttribute(e, "href", string(r))
			return false
		}))

		e.Watch("ui", "text", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			s, ok := evt.NewValue().(ui.String)
			if !ok {
				return true
			}
			SetInnerHTML(e, string(s))
			return false
		}))

		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return Anchor{ui.BasicElement{LoadElement(c(name, id, options...))}}
}

type Button struct {
	ui.BasicElement
}

func (b Button) Autofocus(t bool) Button {
	b.AsElement().SetDataSetUI("autofocus", ui.Bool(t))
	return b
}

func (b Button) Disabled(t bool) Button {
	b.AsElement().SetDataSetUI("disabled", ui.Bool(t))
	return b
}

func (b Button) SetText(str string) Button {
	b.AsElement().SetDataSetUI("content", ui.String(str))
	return b
}

// NewButton returns a button ui.BasicElement.
// TODO (create the type interface for a form button element)
func NewButton(name string, id string, typ string, options ...string) Button {
	f := Elements.NewConstructor("button", func(elementname string, elementid string) *ui.Element {
		e := ui.NewElement(elementname, elementid, Elements.DocType)
		e = enableClasses(e)

		htmlButton := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlButton.IsNull()
		if !exist {
			htmlButton = js.Global().Get("document").Call("createElement", "button")
		} else {
			htmlButton = reset(htmlButton)
		}

		n := NewNativeElementWrapper(htmlButton)
		e.Native = n

		e.Watch("ui", "autofocus", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			b, ok := evt.NewValue().(ui.Bool)
			if !ok {
				return true
			}
			if bool(b) {
				SetAttribute(e, "autofocus", "")
				return false
			}
			RemoveAttribute(e, "autofocus")
			return false
		}))

		e.Watch("ui", "disabled", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			b, ok := evt.NewValue().(ui.Bool)
			if !ok {
				return true
			}
			if bool(b) {
				SetAttribute(e, "disabled", "")
			}
			RemoveAttribute(e, "disabled")
			return false
		}))

		e.Watch("ui", "content", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			s, ok := evt.NewValue().(ui.String)
			if !ok {
				return true
			}
			htmlButton.Set("innerHTML", string(s))
			return false
		}))

		//SetAttribute(e, "name", elementname)
		SetAttribute(e, "id", elementid)
		SetAttribute(e, "type", typ)
		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return Button{ui.BasicElement{LoadElement(f(name, id, options...))}}
}

type Label struct {
	ui.BasicElement
}

func (l Label) SetText(s string) Label {
	l.AsElement().SetUI("content", ui.String(s))
	return l
}

func (l Label) For(e *ui.Element) Label {
	SetAttribute(l.AsElement(), "for", e.ID)
	return l
}

func NewLabel(name string, id string, options ...string) Label {
	c := Elements.NewConstructor("label", func(name string, id string) *ui.Element {
		e := ui.NewElement(name, id, Elements.DocType)
		e = enableClasses(e)

		htmlLabel := js.Global().Get("document").Call("getElementById", id)
		if htmlLabel.IsNull() {
			htmlLabel = js.Global().Get("document").Call("createElement", "label")
		} else {
			htmlLabel = reset(htmlLabel)
		}

		n := NewNativeElementWrapper(htmlLabel)
		e.Native = n

		SetAttribute(e, "id", id)
		e.Watch("ui", "content", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			c, ok := evt.NewValue().(ui.String)
			if !ok {
				return true
			}
			htmlLabel.Set("innerHTML", string(c))
			return false
		}))
		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return Label{ui.BasicElement{LoadElement(c(name, id, options...))}}
}

type Input struct {
	ui.BasicElement
}

func (i Input) Value() ui.String {
	v, ok := i.AsElement().GetData("value")
	if !ok {
		return ui.String("")
	}
	val, ok := v.(ui.String)
	if !ok {
		return ui.String("")
	}
	return val
}

func (i Input) Blur() {
	native, ok := i.AsElement().Native.(NativeElement)
	if !ok {
		panic("native element should be of doc.NativeELement type")
	}
	native.Value.Call("blur")
}

func (i Input) Focus() {
	native, ok := i.AsElement().Native.(NativeElement)
	if !ok {
		panic("native element should be of doc.NativeELement type")
	}
	native.Value.Call("focus")
}

func (i Input) Clear() {
	native, ok := i.AsElement().Native.(NativeElement)
	if !ok {
		panic("native element should be of doc.NativeELement type")
	}
	native.Value.Set("value", "")
}

func NewInput(typ string, name string, id string, options ...string) Input {
	f := Elements.NewConstructor("input", func(elementname string, elementid string) *ui.Element {
		e := ui.NewElement(elementname, elementid, Elements.DocType)
		e = enableClasses(e)

		htmlInput := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlInput.IsNull()
		if !exist {
			htmlInput = js.Global().Get("document").Call("createElement", "input")
		} else {
			htmlInput = reset(htmlInput)
		}

		n := NewNativeElementWrapper(htmlInput)
		e.Native = n

		e.Watch("ui", "value", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			s, ok := evt.NewValue().(ui.String)
			if !ok {
				return true
			}
			//SetAttribute(e, "value", string(s))
			htmlInput.Set("value", string(s))
			return false
		}))

		e.Watch("ui", "accept", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			s, ok := evt.NewValue().(ui.String)
			if !ok {
				return true
			}
			SetAttribute(e, "accept", string(s))
			return false
		}))

		e.Watch("ui", "autocomplete", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			b, ok := evt.NewValue().(ui.Bool)
			if !ok {
				return true
			}
			if bool(b) {
				SetAttribute(e, "autocomplete", "")
				return false
			}
			RemoveAttribute(e, "autocomplete")
			return false
		}))

		e.Watch("ui", "capture", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			s, ok := evt.NewValue().(ui.String)
			if !ok {
				return true
			}
			SetAttribute(e, "capture", string(s))
			return false
		}))

		e.Watch("ui", "checked", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			b, ok := evt.NewValue().(ui.Bool)
			if !ok {
				return true
			}
			if b {
				SetAttribute(e, "checked", "")
				return false
			}
			RemoveAttribute(e, "checked")
			return false
		}))

		e.Watch("ui", "disabled", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			b, ok := evt.NewValue().(ui.Bool)
			if !ok {
				return true
			}
			if bool(b) {
				SetAttribute(e, "disabled", "")
				return false
			}
			RemoveAttribute(e, "disabled")
			return false
		}))

		e.Watch("ui", "inputmode", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			s, ok := evt.NewValue().(ui.String)
			if !ok {
				return true
			}
			SetAttribute(e, "inputmode", string(s))
			return false
		}))

		e.Watch("ui", "maxlength", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			i, ok := evt.NewValue().(ui.Number)
			if !ok {
				return true
			}
			if int(i) > 0 {
				SetAttribute(e, "maxlength", strconv.Itoa(int(i)))
				return false
			}
			RemoveAttribute(e, "maxlength")
			return false
		}))

		e.Watch("ui", "minlength", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			i, ok := evt.NewValue().(ui.Number)
			if !ok {
				return true
			}
			if int(i) > 0 {
				SetAttribute(e, "minlength", strconv.Itoa(int(i)))
				return false
			}
			RemoveAttribute(e, "minlength")
			return false
		}))

		e.Watch("ui", "step", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			i, ok := evt.NewValue().(ui.Number)
			if !ok {
				return true
			}
			if int(i) > 0 {
				SetAttribute(e, "step", strconv.Itoa(int(i)))
				return false
			}
			RemoveAttribute(e, "step")
			return false
		}))

		e.Watch("ui", "min", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			i, ok := evt.NewValue().(ui.Number)
			if !ok {
				return true
			}
			SetAttribute(e, "min", strconv.Itoa(int(i)))
			return false
		}))

		e.Watch("ui", "max", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			i, ok := evt.NewValue().(ui.Number)
			if !ok {
				return true
			}
			SetAttribute(e, "max", strconv.Itoa(int(i)))
			return false
		}))

		e.Watch("ui", "multiple", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			b, ok := evt.NewValue().(ui.Bool)
			if !ok {
				return true
			}
			if bool(b) {
				SetAttribute(e, "multiple", "")
				return false
			}
			RemoveAttribute(e, "multiple")
			return false
		}))

		//SetAttribute(e, "name", elementname)
		SetAttribute(e, "id", elementid)
		SetAttribute(e, "type", typ)

		return e
	}, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return Input{ui.BasicElement{LoadElement(f(name, id, options...))}}
}

type Img struct {
	ui.BasicElement
}

func (i Img) Src(s string) Img {
	i.AsElement().SetDataSetUI("src", ui.String(s))
	return i
}

func (i Img) Alt(s string) Img {
	i.AsElement().SetDataSetUI("alt", ui.String(s))
	return i
}

func NewImage(name, id string, options ...string) Img {
	c := Elements.NewConstructor("img", func(name string, imgid string) *ui.Element {
		e := ui.NewElement(name, imgid, Elements.DocType)
		e = enableClasses(e)

		htmlImg := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlImg.IsNull()
		if !exist {
			htmlImg = js.Global().Get("document").Call("createElement", "img")
		} else {
			htmlImg = reset(htmlImg)
		}

		n := NewNativeElementWrapper(htmlImg)
		e.Native = n
		SetAttribute(e, "id", imgid)
		SetAttribute(e, "alt", name)

		e.Watch("ui", "src", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			src, ok := evt.NewValue().(ui.String)
			if !ok {
				return true
			}
			SetAttribute(e, "src", string(src))
			return false
		}))

		e.Watch("ui", "alt", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			alt, ok := evt.NewValue().(ui.String)
			if !ok {
				return true
			}
			SetAttribute(e, "alt", string(alt))
			return false
		}))
		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

	return Img{ui.BasicElement{LoadElement(c(name, id, options...))}}
}

var NewAudio = Elements.NewConstructor("audio", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)
	e = enableClasses(e)

	htmlAudio := js.Global().Get("document").Call("getElmentByID", id)
	exist := !htmlAudio.IsNull()
	if !exist {
		htmlAudio = js.Global().Get("document").Call("createElement", "audio")
	} else {
		htmlAudio = reset(htmlAudio)
	}

	n := NewNativeElementWrapper(htmlAudio)
	e.Native = n
	//SetAttribute(e, "name", name)
	SetAttribute(e, "id", id)
	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var NewVideo = Elements.NewConstructor("video", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)
	e = enableClasses(e)

	htmlVideo := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlVideo.IsNull()
	if !exist {
		htmlVideo = js.Global().Get("document").Call("createElement", "video")
	} else {
		htmlVideo = reset(htmlVideo)
	}

	//SetAttribute(e, "name", name)
	SetAttribute(e, "id", id)

	n := NewNativeElementWrapper(htmlVideo)
	e.Native = n
	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var NewMediaSource = func(src string, typ string, options ...string) *ui.Element { // TODO review arguments, create custom type interface
	return Elements.NewConstructor("source", func(name string, id string) *ui.Element {
		e := ui.NewElement(name, id, Elements.DocType)
		e = enableClasses(e)

		htmlSource := js.Global().Get("document").Call("getElmentByID", id)
		exist := !htmlSource.IsNull()
		if !exist {
			htmlSource = js.Global().Get("document").Call("createElement", "source")
		} else {
			htmlSource = reset(htmlSource)
		}

		n := NewNativeElementWrapper(htmlSource)
		e.Native = n
		SetAttribute(e, "type", name)
		SetAttribute(e, "src", id)
		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)(typ, src, options...)
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

type TextNode struct {
	UIElement *ui.Element
}

func (t TextNode) Element() *ui.Element {
	return t.UIElement
}
func (t TextNode) SetValue(s ui.String) TextNode {
	t.Element().SetDataSetUI("text", s)
	return t
}

func (t TextNode) Value() ui.String {
	v, ok := t.Element().Get("data", "text")
	if !ok {
		return ""
	}
	s, ok := v.(ui.String)
	if !ok {
		return ""
	}
	return s
}

// NewTextNode creates a text node.
//
func NewTextNode() TextNode {
	var NewNode = Elements.NewConstructor("text", func(name string, id string) *ui.Element {
		e := ui.NewElement(name, id, Elements.DocType)
		htmlTextNode := js.Global().Get("document").Call("createTextNode", "")
		n := NewNativeElementWrapper(htmlTextNode)
		e.Native = n

		e.Watch("ui", "text", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			if s, ok := evt.NewValue().(ui.String); ok { // if data.text is deleted, nothing happens, so no check for nil of  evt.NewValue() TODO handkle all the Value types
				htmlTextNode.Set("nodeValue", string(s))
			}

			return false
		}))

		return e
	}, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return TextNode{NewNode("textnode", NewID())}
}

type TemplatedTextNode struct {
	*ui.Element
}

func (t TemplatedTextNode) AsElement() *ui.Element {
	return t.Element
}

func (t TemplatedTextNode) SetParam(paramName string, value ui.String) TemplatedTextNode {
	params, ok := t.AsElement().GetData("listparams")
	if !ok {
		return t
	}
	paramslist, ok := params.(ui.List)
	if !ok {
		return t
	}
	for _, pname := range paramslist {
		p, ok := pname.(ui.String)
		if !ok {
			continue
		}
		if paramName == string(p) {
			t.Element.SetData(paramName, value)
		}
	}
	return t
}

func (t TemplatedTextNode) Value() ui.String {
	v, ok := t.Element.Get("data", "text")
	if !ok {
		return ""
	}
	s, ok := v.(ui.String)
	if !ok {
		return ""
	}
	return s
}

// NewTemplatedText returns a templated textnode.
// Using SetParam allows to specify a value for the string parameters.
// Checking that all the parameters have been set before appending the textnode
//  is left at the discretion of the user.
func NewTemplatedText(template string) TemplatedTextNode {
	nt := NewTextNode()
	// nt.Element().Set("internals","template", ui.String(template))

	strmuthandlerFn := func(name string) *ui.MutationHandler {
		m := ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			o, ok := evt.Origin().GetData("listparams")
			if !ok {
				return true
			}
			listparams, ok := o.(ui.Object)
			if !ok {
				return true
			}
			var res string

			for k := range listparams {
				if k != "typ" {
					s, ok := evt.Origin().GetData(k)
					if !ok {
						continue
					}
					str, ok := s.(ui.String)
					if !ok {
						continue
					}
					param := "$(" + k + "}"
					res = strings.ReplaceAll(template, param, string(str))
				}
			}
			evt.Origin().SetDataSetUI("text", ui.String(res))
			return false
		})
		return m
	}

	paramnames := parse(template, "${", "}")
	nt.Element().SetData("listparams", paramnames)

	for paramname := range paramnames {
		nt.Element().Watch("data", paramname, nt.Element(), strmuthandlerFn(paramname))
	}

	return TemplatedTextNode{nt.Element()}
}

func parse(input string, tokenstart string, tokenend string) ui.Object {
	result := ui.NewObject()
	ns := input

	startcursor := strings.Index(ns, tokenstart)
	if startcursor == -1 {
		return result
	}
	ns = ns[startcursor:]
	ns = strings.TrimPrefix(ns, tokenstart)

	endcursor := strings.Index(ns, tokenend)
	if endcursor < 1 {
		return result
	}
	tail := ns[endcursor:]
	p := strings.TrimSuffix(ns, tail)
	_, ok := result.Get(p)
	if !ok {
		result.Set(p, nil)
	}

	subresult := parse(strings.TrimPrefix(tail, tokenend), tokenstart, tokenend)
	for k, v := range subresult {
		_, ok := result.Get(k)
		if !ok {
			str, ok := v.(ui.String)
			if !ok {
				continue
			}
			result.Set(k, str)
		}
	}
	return result
}

type List struct {
	ui.BasicElement
}

func (l List) FromValues(values ...ui.Value) List {
	l.AsElement().SetDataSetUI("list", ui.NewList(values...))
	return l
}

func (l List) Values() ui.List {
	v, ok := l.AsElement().GetData("list")
	if !ok {
		return ui.NewList()
	}
	list, ok := v.(ui.List)
	if !ok {
		panic("data/list got overwritten with wrong type or something bad has happened")
	}
	return list
}

func NewUl(name string, id string, options ...string) List {
	c := Elements.NewConstructor("ul", func(ename, eid string) *ui.Element {
		e := ui.NewElement(ename, eid, Elements.DocType)
		e = enableClasses(e)

		htmlList := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlList.IsNull()
		if !exist {
			htmlList = js.Global().Get("document").Call("createElement", "ul")
		} else {
			htmlList = reset(htmlList)
		}

		n := NewNativeElementWrapper(htmlList)
		e.Native = n
		//SetAttribute(e, "name", ename)
		SetAttribute(e, "id", eid)

		h := ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			list, ok := evt.NewValue().(ui.List)
			if !ok {
				return true
			}

			for i, v := range list {
				item := Elements.GetByID(eid + "-item-" + strconv.Itoa(i))
				if item != nil {
					ListItem{ui.BasicElement{item}}.SetValue(v)
				} else {
					item = NewListItem(ename+"-item", eid+"-item-"+strconv.Itoa(i)).SetValue(v).AsBasicElement().AsElement()
				}

				evt.Origin().AppendChild(ui.BasicElement{item})
			}
			return false
		})
		e.Watch("ui", "list", e, h)

		return e
	}, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return List{ui.BasicElement{LoadElement(c(name, id, options...))}}
}

type OrderedList struct {
	ui.BasicElement
}

func (l OrderedList) SetValue(lobjs ui.ListofObjects) OrderedList {
	l.AsElement().Set("data", "value", lobjs)
	return l
}

func NewOl(name string, id string, typ string, numberingstart int, options ...string) OrderedList {
	c := Elements.NewConstructor("ol", func(ename, eid string) *ui.Element {
		e := ui.NewElement(ename, eid, Elements.DocType)
		e = enableClasses(e)

		htmlList := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlList.IsNull()
		if !exist {
			htmlList = js.Global().Get("document").Call("createElement", "ol")
		} else {
			htmlList = reset(htmlList)
		}

		n := NewNativeElementWrapper(htmlList)
		e.Native = n
		//SetAttribute(e, "name", ename)
		SetAttribute(e, "id", eid)
		SetAttribute(e, "type", typ)
		SetAttribute(e, "start", strconv.Itoa(numberingstart))
		return e
	}, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return OrderedList{ui.BasicElement{LoadElement(c(name, id, options...))}}
}

type ListItem struct {
	ui.BasicElement
}

func (li ListItem) SetValue(v ui.Value) ListItem {
	li.AsElement().SetDataSetUI("value", v)
	return li
}

func NewListItem(name string, id string, options ...string) ListItem {
	c := Elements.NewConstructor("li", func(name string, id string) *ui.Element {
		e := ui.NewElement(name, id, Elements.DocType)
		e = enableClasses(e)

		htmlListItem := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlListItem.IsNull()
		if !exist {
			htmlListItem = js.Global().Get("document").Call("createElement", "li")
		} else {
			htmlListItem = reset(htmlListItem)
		}

		n := NewNativeElementWrapper(htmlListItem)
		e.Native = n
		//SetAttribute(e, "name", name)
		SetAttribute(e, "id", id) // TODO define attribute setters optional functions

		onuimutation := ui.NewMutationHandler(func(evt ui.MutationEvent) bool {

			// we apply the modifications to the UI
			v := evt.NewValue()
			var item *ui.Element
			switch t := v.(type) {
			case ui.String:
				item = NewTextNode().SetValue(t).Element()
			case ui.Bool:
				item = NewTextNode().Element()
				item.SetDataSetUI("text", t, false)
			case ui.Number:
				item = NewTextNode().Element()
				item.SetDataSetUI("text", t, false)
			case ui.Object:
				item = NewTextNode().Element()
				item.SetDataSetUI("text", t, false)
			case *ui.Element:
				if t != nil {
					item = t
				} else {
					return true
				}

			default:
				log.Print("not the type we want") // DEBUG
				return true
			}

			evt.Origin().SetChildren(ui.BasicElement{item})
			return false
		})
		e.Watch("ui", "value", e, onuimutation)
		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return ListItem{ui.BasicElement{LoadElement(c(name, id, options...))}}
}

type Table struct {
	ui.BasicElement
}

type Thead struct {
	ui.BasicElement
}

type Tbody struct {
	ui.BasicElement
}

type Tr struct {
	ui.BasicElement
}

type Td struct {
	ui.BasicElement
}

type Th struct {
	ui.BasicElement
}

type TableCell struct {
	ui.BasicElement
}

func NewThead(name string, id string, options ...string) Thead {
	c := Elements.NewConstructor("thead", func(name string, id string) *ui.Element {
		e := ui.NewElement(name, id, Elements.DocType)
		e = enableClasses(e)

		htmlThead := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlThead.IsNull()
		if !exist {
			htmlThead = js.Global().Get("document").Call("createElement", "thead")
		} else {
			htmlThead = reset(htmlThead)
		}

		n := NewNativeElementWrapper(htmlThead)
		e.Native = n
		//SetAttribute(e, "name", name)
		SetAttribute(e, "id", id) // TODO define attribute setters optional functions

		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return Thead{ui.BasicElement{LoadElement(c(name, id, options...))}}
}

func (t Thead) AddRow(rows ...Tr) Thead {
	for _, row := range rows {
		t.AsElement().AppendChild(row)
	}
	return t
}

func (row Tr) AppendThChild(th Th) Tr {
	row.AsElement().AppendChild(th)
	return row
}

func (row Tr) AppendTdChild(td Td) Tr {
	row.AsElement().AppendChild(td)
	return row
}

func NewTr(name string, id string, options ...string) Tr {
	c := Elements.NewConstructor("tr", func(name string, id string) *ui.Element {
		e := ui.NewElement(name, id, Elements.DocType)
		e = enableClasses(e)

		htmlTr := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlTr.IsNull()
		if !exist {
			htmlTr = js.Global().Get("document").Call("createElement", "tr")
		} else {
			htmlTr = reset(htmlTr)
		}

		n := NewNativeElementWrapper(htmlTr)
		e.Native = n
		//SetAttribute(e, "name", name)
		SetAttribute(e, "id", id) // TODO define attribute setters optional functions

		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return Tr{ui.BasicElement{LoadElement(c(name, id, options...))}}
}

func NewTd(name string, id string, options ...string) Td {
	c := Elements.NewConstructor("td", func(name string, id string) *ui.Element {
		e := ui.NewElement(name, id, Elements.DocType)
		e = enableClasses(e)

		htmlTableData := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlTableData.IsNull()
		if !exist {
			htmlTableData = js.Global().Get("document").Call("createElement", "td")
		} else {
			htmlTableData = reset(htmlTableData)
		}

		n := NewNativeElementWrapper(htmlTableData)
		e.Native = n
		//SetAttribute(e, "name", name)
		SetAttribute(e, "id", id) // TODO define attribute setters optional functions

		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return Td{ui.BasicElement{LoadElement(c(name, id, options...))}}
}

func NewTh(name string, id string, options ...string) Th {
	c := Elements.NewConstructor("td", func(name string, id string) *ui.Element {
		e := ui.NewElement(name, id, Elements.DocType)
		e = enableClasses(e)

		htmlTableDataHeader := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlTableDataHeader.IsNull()
		if !exist {
			htmlTableDataHeader = js.Global().Get("document").Call("createElement", "th")
		} else {
			htmlTableDataHeader = reset(htmlTableDataHeader)
		}

		n := NewNativeElementWrapper(htmlTableDataHeader)
		e.Native = n
		//SetAttribute(e, "name", name)
		SetAttribute(e, "id", id) // TODO define attribute setters optional functions

		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return Th{ui.BasicElement{LoadElement(c(name, id, options...))}}
}

func NewTable(name string, id string, options ...string) Table {
	c := Elements.NewConstructor("table", func(name string, id string) *ui.Element {
		e := ui.NewElement(name, id, Elements.DocType)
		e = enableClasses(e)

		htmlTable := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlTable.IsNull()
		if !exist {
			htmlTable = js.Global().Get("document").Call("createElement", "table")
		} else {
			htmlTable = reset(htmlTable)
		}

		n := NewNativeElementWrapper(htmlTable)
		e.Native = n
		//SetAttribute(e, "name", name)
		SetAttribute(e, "id", id) // TODO define attribute setters optional functions

		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return Table{ui.BasicElement{LoadElement(c(name, id, options...))}}
}

/*
func (t Table) AddRow(values ...ui.Value) error { // TODO
	var rowcount int
	c, ok := t.Element().GetData("rowcount")
	if ok {
		count, ok := c.(ui.Number)
		if ok {
			rowcount = int(count)
		}
	}

}
*/
func (t Table) SetColumns(cols ...ColumnDesc) Table {
	c := make([]ui.Value, 0)
	for _, col := range cols {
		c = append(c, ui.Value(col.RawObject()))
	}
	l := ui.NewList(c...)
	t.AsElement().Set("data", "columndesc", l)
	return t
}

type ColumnDesc ui.Object

func (c ColumnDesc) RawObject() ui.Object { return ui.Object(c) }
func (c ColumnDesc) Name() string {
	n, ok := c.RawObject().Get("name")
	if !ok {
		return ""
	}
	name, ok := n.(ui.String)
	if !ok {
		panic("columndesc expects a string for name")
	}
	return string(name)
}
func (c ColumnDesc) DataType() string {
	n, ok := c.RawObject().Get("datatype")
	if !ok {
		return ""
	}
	dt, ok := n.(ui.String)
	if !ok {
		panic("columndesc expects a string for datatype")
	}
	return string(dt)
}
func (c ColumnDesc) Sortable() bool {
	n, ok := c.RawObject().Get("sortable")
	if !ok {
		return false
	}
	sort, ok := n.(ui.Bool)
	if !ok {
		panic("columndesc expects a boolean for sortable")
	}
	return bool(sort)
}
func (c ColumnDesc) Editablel() bool {
	n, ok := c.RawObject().Get("editable")
	if !ok {
		return false
	}
	edit, ok := n.(ui.Bool)
	if !ok {
		panic("columndesc expects a boolean for editable")
	}
	return bool(edit)
}
func NewColumnDesc(name string, datatype string, sortable bool, editable bool) ColumnDesc {
	c := ui.NewObject()
	c.Set("name", ui.String(name))
	c.Set("datatype", ui.String(datatype))
	c.Set("sortable", ui.Bool(sortable))
	c.Set("editable", ui.Bool(editable))
	return ColumnDesc(c)
}

// Code tag TODO

type Code struct {
	ui.BasicElement
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
			sc = strings.TrimSpace(sc + " " + classname)
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

		if len(strings.TrimSpace(string(classes))) != 0 {
			native.Value.Call("setAttribute", "class", string(classes))
			return false
		}
		native.Value.Call("removeAttribute", "class")

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

func AppendInlineCSS(target *ui.Element, str string) { // TODO space separated?
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
	var attrmap ui.Object
	m, ok := target.Get("data", "attrs")
	if !ok {
		attrmap = ui.NewObject()
	} else {
		attrmap, ok = m.(ui.Object)
		if !ok {
			panic("data/attrs should be stored as a ui.Object")
			//log.Print(m,attrmap) // SEBUG
		}
	}

	attrmap.Set(name, ui.String(value))
	target.SetData("attrs", attrmap)

	native, ok := target.Native.(NativeElement)
	if !ok {
		log.Print("Cannot set Attribute on non-expected wrapper type")
		return
	}
	native.Value.Call("setAttribute", name, value)
}

func RemoveAttribute(target *ui.Element, name string) {
	m, ok := target.Get("data", "attrs")
	if !ok {
		return
	}
	attrmap, ok := m.(ui.Object)
	if !ok {
		panic("data/attrs should be stored as a ui.Object")
	}
	delete(attrmap, name)
	target.SetData("attrs", attrmap)

	native, ok := target.Native.(NativeElement)
	if !ok {
		log.Print("Cannot delete Attribute using non-expected wrapper type ", target.ID)
		return
	}
	native.Value.Call("removeAttribute", name)
}

// Buttonify turns an Element into a clickable link
func Buttonify(any ui.AnyElement, link ui.Link) {
	callback := ui.NewEventHandler(func(evt ui.Event) bool {
		link.Activate()
		return false
	})
	any.AsElement().AddEventListener("click", callback, NativeEventBridge)
}

/*
 HTML rendering

*/

func NewHTMLNode(e *ui.Element) *html.Node {
	if e.DocType != Elements.DocType {
		panic("Bad Element doctype")
	}
	v, ok := e.Get("internals", "constructor")
	if !ok {
		return nil
	}
	tag, ok := v.(ui.String)
	if !ok {
		panic("constructor name should be a string")
	}
	data := string(tag)
	nodetype := html.RawNode
	if string(tag) == "root" {
		data = "body"
	}
	n := &html.Node{}
	n.Type = nodetype
	n.Data = data

	attrs, ok := e.GetData("attrs")
	if !ok {
		return n
	}
	tattrs, ok := attrs.(ui.Object)
	if !ok {
		panic("attributes is supossed to be a ui.Object type")
	}
	for k, v := range tattrs {
		val, ok := v.(ui.String)
		if !ok {
			continue // should panic probably instead
		}
		a := html.Attribute{"", k, string(val)}
		n.Attr = append(n.Attr, a)
	}
	return n
}

func NewHTMLTree(document Document) *html.Node {
	doc := document.AsBasicElement()
	return newHTMLTree(doc.AsElement())
}

func newHTMLTree(e *ui.Element) *html.Node {
	d := NewHTMLNode(e)
	if e.Children != nil && e.Children.List != nil {
		for _, child := range e.Children.List {
			c := newHTMLTree(child)
			d.AppendChild(c)
		}
	}
	return d
}
