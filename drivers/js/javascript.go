// +build js,wasm

// Package doc defines the default set of Element constructors, native interfaces,
// events and event handlers, and animation properties used to build js-based UIs.
package doc

import (
	"encoding/json"
	"errors"
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
	DefaultWindow                 Window
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

var sessionstorefn = func(element *ui.Element, category string, propname string, value ui.Value, flags ...bool) {
	store := jsStore{js.Global().Get("sessionStorage")}
	categoryExists := element.Properties.HasCategory(category)
	propertyExists := element.Properties.HasProperty(category, propname)

	if !categoryExists {
		categories := make([]interface{}, 0, len(element.Properties.Categories)+1)
		for k := range element.Properties.Categories {
			categories = append(categories, k)
		}
		categories = append(categories, category)
		v := js.ValueOf(categories)
		store.Set(element.ID, v)
	}
	proptype := "Local"
	if len(flags) > 0 {
		if flags[0] {
			proptype = "Inheritable"
		}
	}
	if !propertyExists {
		props := make([]interface{}, 0, 1)
		c, ok := element.Properties.Categories[category]
		if !ok {
			props = append(props, proptype+"/"+propname)
			v := js.ValueOf(props)
			store.Set(element.ID+"/"+category, v)
		}
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
		v := js.ValueOf(props)
		store.Set(element.ID+"/"+category, v)
	}
	v := js.ValueOf(map[string]interface{}(value.RawValue()))
	store.Set(element.ID+"/"+category+"/"+propname, v)
	return
}

var localstoragefn = func(element *ui.Element, category string, propname string, value ui.Value, flags ...bool) {
	store := jsStore{js.Global().Get("localStorage")}
	categoryExists := element.Properties.HasCategory(category)
	propertyExists := element.Properties.HasProperty(category, propname)

	if !categoryExists {
		categories := make([]interface{}, 0, len(element.Properties.Categories)+1)
		for k := range element.Properties.Categories {
			categories = append(categories, k)
		}
		categories = append(categories, category)
		v := js.ValueOf(categories)
		store.Set(element.ID, v)
	}
	proptype := "Local"
	if len(flags) > 0 {
		if flags[0] {
			proptype = "Inheritable"
		}
	}
	if !propertyExists {
		props := make([]interface{}, 0, 1)
		c, ok := element.Properties.Categories[category]
		if !ok {
			props = append(props, proptype+"/"+propname)
			v := js.ValueOf(props)
			store.Set(element.ID+"/"+category, v)
		}
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
		v := js.ValueOf(props)
		store.Set(element.ID+"/"+category, v)
	}
	v := js.ValueOf(map[string]interface{}(value.RawValue()))
	store.Set(element.ID+"/"+category+"/"+propname, v)
	return
}

var loadfromsession = func(e *ui.Element) error {
	store := jsStore{js.Global().Get("sessionStorage")}
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
	for _, category := range categories {
		jsonproperties, ok := store.Get(e.ID + "/" + category)
		if !ok {
			return nil
		}
		err = json.Unmarshal([]byte(jsonproperties.String()), &properties)
		if err != nil {
			return err
		}
		for _, property := range properties {
			// let's retrieve the propname (it is suffixed by the proptype)
			// then we can retrieve the value
			proptypename := strings.SplitAfter(property, "/")
			proptype := proptypename[0]
			propname := proptypename[1]
			jsonvalue, ok := store.Get(e.ID + "/" + category + "/" + propname)
			if ok {
				rawvalue := ui.NewObject()
				err = json.Unmarshal([]byte(jsonvalue.String()), rawvalue)
				if err != nil {
					return err
				}
				if category != "ui" && propname != "mutationrecords" {
					ui.LoadProperty(e, category, propname, proptype, rawvalue.Value())
				}
				rawmutationrecords := rawvalue.Value()
				if rawmutationrecords.ValueType() != "List" {
					return errors.New("mutationrecords are not of type List")
				}
				mutationrecordlist, ok := rawmutationrecords.(ui.List)
				if !ok {
					return errors.New("mutationrecords are not of List type")
				}

				for _, mutationrecord := range mutationrecordlist {
					record, ok := mutationrecord.(ui.Object)
					if !ok {
						return errors.New("mutationrecord is not of expected type.")
					}
					vcategory, ok := record.Get("category")
					if !ok {
						return errors.New("mutationrecord bad encoding")
					}
					category, ok := vcategory.(ui.String)
					if !ok {
						return errors.New("mutationrecord bad encoding")
					}

					vpropname, ok := record.Get("property")
					if !ok {
						return errors.New("propname not found")
					}
					propname, ok := vpropname.(ui.String)
					if !ok {
						return errors.New("mutationrecord bad encoding")
					}
					value, ok := record.Get("value")
					if !ok {
						return errors.New("value not found")
					}
					e.Set(string(category), string(propname), value)
				}
			}
		}
	}
	return nil
}

var loadfromlocalstorage = func(e *ui.Element) error {
	store := jsStore{js.Global().Get("localStorage")}
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
	for _, category := range categories {
		jsonproperties, ok := store.Get(e.ID + "/" + category)
		if !ok {
			return nil
		}
		err = json.Unmarshal([]byte(jsonproperties.String()), &properties)
		if err != nil {
			return err
		}
		for _, property := range properties {
			// let's retrieve the propname (it is suffixed by the proptype)
			// then we can retrieve the value
			proptypename := strings.SplitAfter(property, "/")
			proptype := proptypename[0]
			propname := proptypename[1]
			jsonvalue, ok := store.Get(e.ID + "/" + category + "/" + propname)
			if ok {
				rawvalue := ui.NewObject()
				err = json.Unmarshal([]byte(jsonvalue.String()), rawvalue)
				if err != nil {
					return err
				}
				if category != "ui" && propname != "mutationrecords" {
					ui.LoadProperty(e, category, propname, proptype, rawvalue.Value())
				}
				rawmutationrecords := rawvalue.Value()
				if rawmutationrecords.ValueType() != "List" {
					return errors.New("mutationrecords are not of type List")
				}
				mutationrecordlist, ok := rawmutationrecords.(ui.List)
				if !ok {
					return errors.New("mutationrecords are not of List type")
				}

				for _, mutationrecord := range mutationrecordlist {
					record, ok := mutationrecord.(ui.Object)
					if !ok {
						return errors.New("mutationrecord is not of expected type.")
					}
					vcategory, ok := record.Get("category")
					if !ok {
						return errors.New("mutationrecord bad encoding")
					}
					category, ok := vcategory.(ui.String)
					if !ok {
						return errors.New("mutationrecord bad encoding")
					}

					vpropname, ok := record.Get("property")
					if !ok {
						return errors.New("propname not found")
					}
					propname, ok := vpropname.(ui.String)
					if !ok {
						return errors.New("mutationrecord bad encoding")
					}
					value, ok := record.Get("value")
					if !ok {
						return errors.New("value not found")
					}
					e.Set(string(category), string(propname), value)
				}
			}
		}
	}
	return nil
}

// Window is a ype that represents a browser window
type Window struct {
	UIElement *ui.Element
}

func (w Window) Element() *ui.Element {
	return w.UIElement
}

func (w Window) SetTitle(title string) {
	w.Element().Set("ui", "title", ui.String(title))
}

// TODO see if can get height width of window view port, etc.

func GetWindow() Window {
	e := ui.NewElement("window", DefaultWindowTitle, DOCTYPE)
	e.ElementStore = Elements
	wd := js.Global()
	if !wd.Truthy() {
		log.Print("unable to access windows")
		return Window{}
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
		nat, ok := target.Native.(js.Wrapper)
		if !ok {
			return true
		}
		jswindow := nat.JSValue()
		if !jswindow.Truthy() {
			log.Print("Unable to access native Window object")
			return true
		}
		jswindow.Get("document").Set("title", string(newtitle))
		return false
	})

	e.Watch("ui", "title", e, h)
	e.Set("ui", "title", ui.String(DefaultWindowTitle), false)

	return Window{e}
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

// NewAppRoot creates a new app entry point. It is the top-most element
// in the tree of Elements that consitute the full document.
// It should be the element which is passed to a router to observe for route
// change.
// By default, it represents document.body. As such, it is different from the
// document which holds the head element for instance.
var NewDocument = Elements.NewConstructor("root", func(name string, id string) *ui.Element {
	DefaultWindow = GetWindow()

	Elements.AddPersistenceMode("sessionstorage", loadfromsession, sessionstorefn)
	Elements.AddPersistenceMode("localstorage", loadfromlocalstorage, localstoragefn)

	e := Elements.NewAppRoot(id)

	root := js.Global().Get("document").Get("body")
	if !root.Truthy() {
		log.Print("failed to instantiate root element for the document")
		return e
	}
	n := NewNativeElementWrapper(root)
	e.Native = n
	return e
}, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

// Div is a concrete type that holds the common interface to Div *ui.Element objects.
// i.e. ui.Element whose constructor name is "div" and represents html div elements.
type Div struct {
	UIElement *ui.Element
}

func (d Div) Element() *ui.Element { return d.UIElement }
func (d Div) Contenteditable(b bool) Div {
	d.Element().SetDataSyncUI("contenteditable", ui.Bool(b))
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
		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

	return Div{tryLoad(newDiv(name, id, options...))}
}

func tryLoad(d *ui.Element) *ui.Element {
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
// The default ui.Element interface is reachable via a call to the Element() method.
type Tooltip struct {
	UIElement *ui.Element
}

func (t Tooltip) Element() *ui.Element {
	return t.UIElement
}

// SetContent sets the content of the tooltip. To pass some text, use a TextNode.
func (t Tooltip) SetContent(content *ui.Element) Tooltip {
	t.Element().SetData("content", content)
	return t
}

var tooltipConstructor = Elements.NewConstructor("tooltip", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)
	e = enableClasses(e)

	htmlTooltip := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlTooltip.IsNull()

	if !exist {
		htmlTooltip = js.Global().Get("document").Call("createElement", "div")
	}

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
		tn.SetValue(strcontent)
		//tooltip.AppendChild(tn)
		tooltip.Set("ui", "command", ui.AppendChildCommand(tn.Element()), false)
		return false
	})
	e.Watch("data", "content", e, h)

	return e
}, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func HasTooltip(target *ui.Element) (Tooltip, bool) {
	v := target.ElementStore.GetByID(target.ID + "-tooltip")
	if v == nil {
		return Tooltip{v}, false
	}
	return Tooltip{v}, true
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
	e := tryLoad(tooltipConstructor(target.Name+"/tooltip", target.ID+"-tooltip"))
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

type TextArea struct {
	UIElement *ui.Element
}

func (t TextArea) Element() *ui.Element {
	return t.UIElement
}

func (t TextArea) Text() string {
	v, ok := t.Element().GetData("text")
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
	t.Element().SetDataSyncUI("text", ui.String(text))
	return t
}

func (t TextArea) SetColumns(i int) TextArea {
	t.Element().SetDataSyncUI("cols", ui.Number(i))
	return t
}

func (t TextArea) SetRows(i int) TextArea {
	t.Element().SetDataSyncUI("rows", ui.Number(i))
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
		SetAttribute(e, "name", ename)
		SetAttribute(e, "id", eid)
		e.SetDataSyncUI("rows", ui.String(strconv.Itoa(rows)))
		e.SetDataSyncUI("cols", ui.String(strconv.Itoa(cols)))
		return e
	}, allowTextAreaDataBindingOnBlur, allowTextAreaDataBindingOnInput, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return TextArea{tryLoad(t(name, id, options...))}
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
			nn := n.JSValue()
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

type Header struct {
	UIElement *ui.Element
}

func (h Header) Element() *ui.Element {
	return h.UIElement
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
		}

		n := NewNativeElementWrapper(htmlHeader)
		e.Native = n
		if !exist {
			SetAttribute(e, "id", id)
		}
		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return Header{tryLoad(c(name, id, options...))}
}

type Footer struct {
	UIElement *ui.Element
}

func (f Footer) Element() *ui.Element {
	return f.UIElement
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
		}

		n := NewNativeElementWrapper(htmlFooter)
		e.Native = n

		if !exist {
			SetAttribute(e, "id", id)
		}

		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return Footer{tryLoad(c(name, id, options...))}
}

type Span struct {
	UIElement *ui.Element
}

func (s Span) Element() *ui.Element {
	return s.UIElement
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
		}

		n := NewNativeElementWrapper(htmlSpan)
		e.Native = n
		if !exist {
			SetAttribute(e, "id", id)
		}
		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return Span{tryLoad(c(name, id, options...))}
}

type Paragraph struct {
	UIElement *ui.Element
}

func (p Paragraph) Element() *ui.Element {
	return p.UIElement
}

// NewParagraph is a constructor for html paragraph elements.
func NewParagraph(name string, id string, options ...string) Paragraph {
	c := Elements.NewConstructor("paragraph", func(name string, id string) *ui.Element {
		e := ui.NewElement(name, id, Elements.DocType)
		e = enableClasses(e)

		htmlParagraph := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlParagraph.IsNull()
		if !exist {
			htmlParagraph = js.Global().Get("document").Call("createElement", "p")
		}

		n := NewNativeElementWrapper(htmlParagraph)
		e.Native = n
		if !exist {
			SetAttribute(e, "id", id)
		}
		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return Paragraph{tryLoad(c(name, id, options...))}
}

type Nav struct {
	UIElement *ui.Element
}

func (n Nav) Element() *ui.Element {
	return n.UIElement
}

func (n Nav) AppendAnchorLink(l Link) Nav {
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
	return Nav{tryLoad(c(name, id, options...))}
}

type Link struct {
	UIElement *ui.Element
}

func (l Link) Element() *ui.Element {
	return l.UIElement
}

// NewAnchor creates an html anchor element which points to the object
// passed as argument. The anchor id becomes the target Element ID prepended
// with the string "link_"
// If the object does not exist, it points to itself. The link uses the id passed
// as argument without prepending anything.
// TODO replace name,id string arguments by *ui.Element argument
func NewLink(target *ui.Element, options ...string) Link {
	var Name = "link"
	var ID = NewID()
	if target != nil {
		Name = target.Name + "_link"
		ID = target.ID + "_link"
	}
	c := Elements.NewConstructor("link", func(name string, id string) *ui.Element {
		e := ui.NewElement(name, id, Elements.DocType)
		e = enableClasses(e)

		htmlAnchor := js.Global().Get("document").Call("getElementbyId", id)
		exist := !htmlAnchor.IsNull()
		if !exist {
			htmlAnchor = js.Global().Get("document").Call("createElement", "a")
		}

		// finds the element whose id has been passed as argument: if search returns nil
		// then the Link element references itself.
		lnkTarget := target
		if target == nil {
			lnkTarget = e
		}

		// Set a mutation Handler on lnkTarget which observes the tree insertion event (attach event)
		// At each attachment, we should rewrite href with the new route.
		lnkTarget.Watch("event", "attached", lnkTarget, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			// SetAttribute(e, "href", e.Route())
			r, ok := e.GetData("href")
			if ok {
				oldroute, ok := r.(ui.String)
				if ok {
					if string(oldroute) == e.Route() {
						return false
					}
				}
			}
			e.SetDataSyncUI("href", ui.String(e.Route()))
			return false
		}))
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

		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return Link{tryLoad(c(Name, ID, options...))}
}

type Button struct {
	UIElement *ui.Element
}

func (b Button) Element() *ui.Element {
	return b.UIElement
}

func (b Button) Autofocus(t bool) Button {
	b.Element().SetDataSyncUI("autofocus", ui.Bool(t))
	return b
}

func (b Button) Disabled(t bool) Button {
	b.Element().SetDataSyncUI("disabled", ui.Bool(t))
	return b
}

// NewButton returns a button ui.Element.
// TODO (create the type interface for a form button element)
func NewButton(name string, id string, typ string, options ...string) Button {
	f := Elements.NewConstructor("button", func(elementname string, elementid string) *ui.Element {
		e := ui.NewElement(elementname, elementid, Elements.DocType)
		e = enableClasses(e)

		htmlButton := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlButton.IsNull()
		if !exist {
			htmlButton = js.Global().Get("document").Call("createElement", "button")
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

		SetAttribute(e, "name", elementname)
		SetAttribute(e, "id", elementid)
		SetAttribute(e, "type", typ)
		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return Button{tryLoad(f(name, id, options...))}
}

type Label struct {
	UIElement *ui.Element
}

func (l Label) Element() *ui.Element {
	return l.UIElement
}

func NewLabel(name string, id string, text string, options ...string) Label {
	c := Elements.NewConstructor("label", func(name string, id string) *ui.Element {
		e := ui.NewElement(name, id, Elements.DocType)
		e = enableClasses(e)

		htmlLabel := js.Global().Get("document").Call("getElmentByID", id)
		if htmlLabel.IsNull() {
			htmlLabel = js.Global().Get("document").Call("createElement", "label")
		}

		n := NewNativeElementWrapper(htmlLabel)
		e.Native = n
		target := Elements.GetByID(id)
		if target != nil {
			e.ID = "label_" + id
			SetAttribute(e, "for", id)
		}
		t := NewTextNode().SetValue(ui.String(text))
		e.Mutate(ui.AppendChildCommand(t.Element()))
		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return Label{tryLoad(c(name, id, options...))}
}

var NewInput = func(name string, id string, typ string, options ...string) *ui.Element {
	f := Elements.NewConstructor("input", func(elementname string, elementid string) *ui.Element {
		e := ui.NewElement(elementname, elementid, Elements.DocType)
		e = enableClasses(e)

		htmlInput := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlInput.IsNull()
		if !exist {
			htmlInput = js.Global().Get("document").Call("createElement", "input")
		}

		n := NewNativeElementWrapper(htmlInput)
		e.Native = n

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
			if bool(b) {
				SetAttribute(e, "checked", "")
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
			}
			RemoveAttribute(e, "multiple")
			return false
		}))

		SetAttribute(e, "name", elementname)
		SetAttribute(e, "id", elementid)
		SetAttribute(e, "type", typ)
		return e
	}, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return tryLoad(f(name, id, options...))
}

type Image struct {
	UIElement *ui.Element
}

func (i Image) Element() *ui.Element {
	return i.UIElement
}

func (i Image) Src(s string) Image {
	i.Element().SetDataSyncUI("src", ui.String(s))
	return i
}

func (i Image) Alt(s string) Image {
	i.Element().SetDataSyncUI("alt", ui.String(s))
	return i
}

func NewImage(name, id string, options ...string) Image {
	c := Elements.NewConstructor("image", func(name string, imgid string) *ui.Element {
		e := ui.NewElement(name, imgid, Elements.DocType)
		e = enableClasses(e)

		htmlImg := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlImg.IsNull()
		if !exist {
			htmlImg = js.Global().Get("document").Call("createElement", "img")
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

	return Image{tryLoad(c(name, id, options...))}
}

var NewAudio = Elements.NewConstructor("audio", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)
	e = enableClasses(e)

	htmlAudio := js.Global().Get("document").Call("getElmentByID", id)
	exist := !htmlAudio.IsNull()
	if !exist {
		htmlAudio = js.Global().Get("document").Call("createElement", "audio")
	}

	n := NewNativeElementWrapper(htmlAudio)
	e.Native = n
	SetAttribute(e, "name", name)
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
	}

	SetAttribute(e, "name", name)
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
	t.Element().SetDataSyncUI("text", s)
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
		htmlTextNode := js.Global().Get("document").Call("createTextNode", "") // DEBUG
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
	UIElement *ui.Element
}

func (t TemplatedTextNode) Element() *ui.Element {
	return t.UIElement
}

func (t TemplatedTextNode) SetParam(paramName string, value ui.String) TemplatedTextNode {
	params, ok := t.Element().GetData("listparams")
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
			t.Element().SetData(paramName, value)
		}
	}
	return t
}

func (t TemplatedTextNode) Value() ui.String {
	v, ok := t.UIElement.Get("data", "text")
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
			evt.Origin().SetDataSyncUI("text", ui.String(res))
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
	UIElement *ui.Element
}

func (l List) Element() *ui.Element {
	return l.UIElement
}

func (l List) Append(values ...ui.Value) List {
	listAppend(l.Element(), values...)
	return l
}

func (l List) Prepend(values ...ui.Value) List {
	listPrepend(l.Element(), values...)
	return l
}

func (l List) InsertAt(index int, values ...ui.Value) List {
	listInsertAt(l.Element(), index, values...)
	return l
}

func (l List) Delete(itemindex int) List {
	listDelete(l.Element(), itemindex)
	return l
}

func NewList(name string, id string, options ...string) List {
	c := Elements.NewConstructor("ul", func(ename, eid string) *ui.Element {
		e := ui.NewElement(ename, eid, Elements.DocType)
		e = enableClasses(e)

		htmlList := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlList.IsNull()
		if !exist {
			htmlList = js.Global().Get("document").Call("createElement", "ul")
		}

		n := NewNativeElementWrapper(htmlList)
		e.Native = n
		SetAttribute(e, "name", ename)
		SetAttribute(e, "id", eid)
		return e
	}, AllowListAutoSync, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return List{tryLoad(c(name, id, options...))}
}

type OrderedList struct {
	UIElement *ui.Element
}

func (l OrderedList) Element() *ui.Element {
	return l.UIElement
}

func (l OrderedList) Append(values ...ui.Value) OrderedList {
	listAppend(l.Element(), values...)
	return l
}

func (l OrderedList) Prepend(values ...ui.Value) OrderedList {
	listPrepend(l.Element(), values...)
	return l
}

func (l OrderedList) InsertAt(index int, values ...ui.Value) OrderedList {
	listInsertAt(l.Element(), index, values...)
	return l
}

func (l OrderedList) Delete(itemindex int) OrderedList {
	listDelete(l.Element(), itemindex)
	return l
}

func NewOrderedList(name string, id string, typ string, numberingstart int, options ...string) OrderedList {
	c := Elements.NewConstructor("ol", func(ename, eid string) *ui.Element {
		e := ui.NewElement(ename, eid, Elements.DocType)
		e = enableClasses(e)

		htmlList := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlList.IsNull()
		if !exist {
			htmlList = js.Global().Get("document").Call("createElement", "ol")
		}

		n := NewNativeElementWrapper(htmlList)
		e.Native = n
		SetAttribute(e, "name", ename)
		SetAttribute(e, "id", eid)
		SetAttribute(e, "type", typ)
		SetAttribute(e, "start", strconv.Itoa(numberingstart))
		return e
	}, AllowListAutoSync, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return OrderedList{tryLoad(c(name, id, options...))}
}

type ListItem struct {
	UIElement *ui.Element
}

func (li ListItem) Element() *ui.Element {
	return li.UIElement
}

func (li ListItem) SetValue(v ui.Value) ListItem {
	li.Element().SetDataSyncUI("content", v)
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
		}

		n := NewNativeElementWrapper(htmlListItem)
		e.Native = n
		SetAttribute(e, "name", name)
		SetAttribute(e, "id", id) // TODO define attribute setters optional functions

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
			var item *ui.Element
			switch t := v.(type) {
			case ui.String:
				item = NewTextNode().Element().Element()
				item.SetDataSyncUI("text", t, false)
			case ui.Bool:
				item = NewTextNode().Element()
				item.SetDataSyncUI("text", t, false)
			case ui.Number:
				item = NewTextNode().Element()
				item.SetDataSyncUI("text", t, false)
			case ui.Object:
				item = NewTextNode().Element()
				item.SetDataSyncUI("text", t, false)
			case *ui.Element:
				evt.Origin().Mutate(ui.RemoveChildrenCommand())
				evt.Origin().Mutate(ui.AppendChildCommand(t))
				return false
			default:
				return true
			}

			evt.Origin().Mutate(ui.RemoveChildrenCommand())
			evt.Origin().Mutate(ui.AppendChildCommand(item))
			return false
		})
		e.Watch("ui", "content", e, onuimutation)
		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)
	return ListItem{tryLoad(c(name, id, options...))}
}

func newListValue(index int, value ui.Value) ui.Object {
	o := ui.NewObject()
	o.Set("index", ui.Number(index))
	o.Set("value", value)
	return o
}

func ListValueInfo(v ui.Value) (index int, newvalue ui.Value, ok bool) {
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

func listAppend(list *ui.Element, values ...ui.Value) *ui.Element {
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

func listPrepend(list *ui.Element, values ...ui.Value) *ui.Element {
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

func listInsertAt(list *ui.Element, offset int, values ...ui.Value) *ui.Element {
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

func listDelete(list *ui.Element, offset int) *ui.Element {
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
		i, v, ok := ListValueInfo(evt.NewValue())
		if !ok {
			return true
		}

		if evt.ObservedKey() == "append" {
			id := NewID()
			n := NewListItem(evt.Origin().Name+"-item", id)
			n.Element().SetDataSyncUI("content", v, false)

			// evt.Origin().AppendChild(n)
			evt.Origin().Mutate(ui.AppendChildCommand(n.Element()))
		}

		if evt.ObservedKey() == "prepend" {
			id := NewID()
			n := NewListItem(evt.Origin().Name+"-item", id)
			n.Element().SetDataSyncUI("content", v, false)

			// evt.Origin().PrependChild(n)
			evt.Origin().Set("ui", "command", ui.PrependChildCommand(n.Element()))
		}

		if evt.ObservedKey() == "insert" {
			id := NewID()
			n := NewListItem(evt.Origin().Name+"-item", id)
			n.Element().SetDataSyncUI("content", v, false)

			// evt.Origin().InsertChild(n, i)
			evt.Origin().Mutate(ui.InsertChildCommand(n.Element(), i))
		}

		if evt.ObservedKey() == "delete" {
			target := evt.Origin()
			deletee := target.Children.AtIndex(i)
			if deletee != nil {
				// target.RemoveChild(deletee)
				target.Mutate(ui.RemoveChildCommand(deletee))
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
	native, ok := target.Native.(NativeElement)
	if !ok {
		log.Print("Cannot set Attribute on non-expected wrapper type")
		return
	}
	native.JSValue().Call("setAttribute", name, value)
}

func RemoveAttribute(target *ui.Element, name string) {
	native, ok := target.Native.(NativeElement)
	if !ok {
		log.Print("Cannot delete Attribute using non-expected wrapper type")
		return
	}
	native.JSValue().Call("removeAttribute", name)
}
