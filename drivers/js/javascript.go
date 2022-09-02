// go:build js && wasm

// Package doc defines the default set of Element constructors, native interfaces,
// events and event handlers, and animation properties used to build js-based UIs.

package doc

import (
	"encoding/json"
	//"errors"
	"log"
	"strconv"
	"strings"
	"syscall/js"
	"time"
	"github.com/atdiar/particleui"
	"net/url"
)

func init(){
	ui.NativeEventBridge = NativeEventBridge
	ui.NativeDispatch = NativeDispatch
}

var (
	// DOCTYPE holds the document doctype.
	DOCTYPE = "html/js"
	// Elements stores wasm-generated HTML ui.Element constructors.
	Elements                      = ui.NewElementStore("default", DOCTYPE).AddPersistenceMode("sessionstorage", loadfromsession, sessionstorefn, clearfromsession).AddPersistenceMode("localstorage", loadfromlocalstorage, localstoragefn, clearfromlocalstorage).ApplyGlobalOption(cleanStorageOnDelete)
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

func(s jsStore) Delete(key string){
	s.store.Call("removeItem",key)
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

		// Let's check whether the element exists in store. In the negative case,
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
			props := make([]interface{}, 0, 4)
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
		element.Set("event","storesynced",ui.Bool(false))
		return
	}
}

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

		categories := make([]string, 0, 50)
		properties := make([]string, 0, 50)
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

func clearer(s string) func(element *ui.Element){
	return func(element *ui.Element){
		store := jsStore{js.Global().Get(s)}
		id := element.ID

		// Let's retrieve the category index for this element, if it exists in the sessionstore
		jsoncategories, ok := store.Get(id)
		if !ok {
			return
		}

		categories := make([]string, 0, 20)
		properties := make([]string, 0, 50)
		err := json.Unmarshal([]byte(jsoncategories.String()), &categories)
		if err != nil {
			return 
		}
		
		for _, category := range categories {
			jsonproperties, ok := store.Get(id + "/" + category)
			if !ok {
				continue
			}
			err = json.Unmarshal([]byte(jsonproperties.String()), &properties)
			if err != nil {
				store.Delete(id)
				panic("An error occured when removing an element from storage. It's advised to reinitialize " + s)
			}

			for _, property := range properties {
				// let's retrieve the propname (it is suffixed by the proptype)
				// then we can retrieve the value
				// log.Print("debug...", category, property) // DEBUG
				proptypename := strings.Split(property, "/")
				//proptype := proptypename[0]
				propname := proptypename[1]
				store.Delete(id + "/" + category + "/" + propname)
			}
			store.Delete(id + "/" + category)
		}
		store.Delete(id)
		element.Set("event","storesynced",ui.Bool(false))
	}
}

var clearfromsession = clearer("sessionStorage")
var clearfromlocalstorage = clearer("localStorage")

var cleanStorageOnDelete = ui.NewConstructorOption("cleanstorageondelete",func(e *ui.Element)*ui.Element{
	e.OnDeleted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		ClearFromStorage(evt.Origin())
		j:= JSValue(e)
		if j.Truthy(){
			j.Call("remove")
		}
		return false
	}))
	return e
})

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

var newWindowConstructor= Elements.NewConstructor("window", func(id string) *ui.Element {
	e := ui.NewElement("window", DOCTYPE)
	e.Set("event", "mounted", ui.Bool(true))
	e.Set("event", "mountable", ui.Bool(true))
	e.Set("event", "attached", ui.Bool(true))
	e.Set("event", "firstmount", ui.Bool(true))
	e.Set("event", "firsttimemounted", ui.Bool(true))
	e.ElementStore = Elements
	e.Parent = e
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

	return e
})

func newWindow(title string, options ...string) Window {
	e:= newWindowConstructor("window", options...)
	e.Set("ui", "title", ui.String(title), false)
	return Window{ui.BasicElement{LoadFromStorage(e)}}
}

func GetWindow(options ...string) Window {
	w := Elements.GetByID("window")
	if w == nil {
		return newWindow("Created with ParticleUI", options...)
	}
	cname, ok := w.Get("internals", "constructor")
	if !ok {
		return newWindow("Created with ParticleUI", options...)
	}
	nname, ok := cname.(ui.String)
	if !ok {
		return newWindow("Created with ParticleUI", options...)
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
		log.Print("wrong format for native element underlying objects.Cannot append " + child.ID)
		return
	}
	n.Value.Call("append", v.Value)
}

func (n NativeElement) PrependChild(child *ui.Element) {
	v, ok := child.Native.(NativeElement)
	if !ok {
		log.Print("wrong format for native element underlying objects.Cannot prepend " + child.ID)
		return
	}
	n.Value.Call("prepend", v.Value)
}

func (n NativeElement) InsertChild(child *ui.Element, index int) {
	v, ok := child.Native.(NativeElement)
	if !ok {
		log.Print("wrong format for native element underlying objects.Cannot insert " + child.ID)
		return
	}
	childlist := n.Value.Get("children")
	length := childlist.Get("length").Int()
	if index > length {
		log.Print("insertion attempt out of bounds.")
		return
	}

	if index == length {
		n.Value.Call("append", v.Value)
		return
	}
	r := childlist.Call("item", index)
	n.Value.Call("insertBefore", v.Value, r)
}

func (n NativeElement) ReplaceChild(old *ui.Element, new *ui.Element) {
	nold, ok := old.Native.(NativeElement)
	if !ok {
		log.Print("wrong format for native element underlying objects.Cannot replace " + old.ID)
		return
	}
	nnew, ok := new.Native.(NativeElement)
	if !ok {
		log.Print("wrong format for native element underlying objects.Cannot replace with " + new.ID)
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

func (n NativeElement) SetChildren(children ...*ui.Element) {
	fragment := js.Global().Get("document").Call("createDocumentFragment")
	for _, child := range children {
		v, ok := child.Native.(NativeElement)
		if !ok {
			panic("wrong format for native element underlying objects.Cannot append " + child.ID)
		}
		fragment.Call("append", v.Value)
	}
	n.Value.Call("append", fragment)
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
// NOTE: the element constructor functions are stored in unexported top-level variables so that 
// when reconstructing an element from its serialized representation, we are sure that the constructor exists.
// If the constructor was defined within a function, it would require for that function to have been called first.
// This might not have happened and maybe navigation/path-dependent.
*/

// AllowSessionStoragePersistence is a constructor option.
// A constructor option allows us to add custom optional behaviors to Element constructors.
// If made available to a constructor function, the coder may decide to enable
//  session storage of the properties of an Element  created with said constructor.
var AllowSessionStoragePersistence = ui.NewConstructorOption("sessionstorage", func(e *ui.Element) *ui.Element {
	e.Set("internals", "persistence", ui.String("sessionstorage"))
	if isPersisted(e){
		return LoadFromStorage(e)
	}
	return PutInStorage(e)
})

var AllowAppLocalStoragePersistence = ui.NewConstructorOption("localstorage", func(e *ui.Element) *ui.Element {
	e.Set("internals", "persistence", ui.String("localstorage"))
	if isPersisted(e){
		return LoadFromStorage(e)
	}
	return PutInStorage(e)
})

func EnableSessionPersistence() string {
	return "sessionstorage"
}

func EnableLocalPersistence() string {
	return "localstorage"
}

func isScrollable(property string) bool {
	switch property {
	case "auto":
		return true
	case "scroll":
		return true
	default:
		return false
	}
}

var AllowScrollRestoration = ui.NewConstructorOption("scrollrestoration", func(e *ui.Element) *ui.Element {
	e.OnFirstTimeMounted(ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		e.Watch("navigation", "ready", e.Root(), ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			router := ui.GetRouter()

			ejs := JSValue(e)
			wjs := js.Global().Get("document").Get("defaultView")

			stylesjs := wjs.Call("getComputedStyle", ejs)
			overflow := stylesjs.Call("getPropertyValue", "overflow").String()
			overflowx := stylesjs.Call("getPropertyValue", "overflowX").String()
			overflowy := stylesjs.Call("getPropertyValue", "overflowY").String()

			scrollable := isScrollable(overflow) || isScrollable(overflowx) || isScrollable(overflowy)

			if scrollable {
				if js.Global().Get("history").Get("scrollRestoration").Truthy() {
					js.Global().Get("history").Set("scrollRestoration", "manual")
				}
				e.SetDataSetUI("scrollrestore", ui.Bool(true))
				e.AddEventListener("scroll", ui.NewEventHandler(func(evt ui.Event) bool {
					scrolltop := ui.Number(ejs.Get("scrollTop").Float())
					scrollleft := ui.Number(ejs.Get("scrollLeft").Float())
					router.History.Set(e.ID, "scrollTop", scrolltop)
					router.History.Set(e.ID, "scrollLeft", scrollleft)
					return false
				}))

				h := ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
					b, ok := e.Get("event", "shouldscroll")
					if !ok {
						return false
					}
					if scroll := b.(ui.Bool); scroll {
						t, ok := router.History.Get(e.ID, "scrollTop")
						if !ok {
							ejs.Set("scrollTop", 0)
							ejs.Set("scrollLeft", 0)
							return false
						}
						l, ok := router.History.Get(e.ID, "scrollLeft")
						if !ok {
							ejs.Set("scrollTop", 0)
							ejs.Set("scrollLeft", 0)
							return false
						}
						top := t.(ui.Number)
						left := l.(ui.Number)
						ejs.Set("scrollTop", float64(top))
						ejs.Set("scrollLeft", float64(left))
						if e.ID != e.Root().ID {
							e.Set("event", "shouldscroll", ui.Bool(false)) //always scroll root
						}
					}
					return false
				})
				e.Watch("event", "navigationend", e.Root(), h)
			} else {
				e.SetDataSetUI("scrollrestore", ui.Bool(false))
			}
			return false
		}))
		return false
	}))

	e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent) bool { // TODO DEBUG Mounted is not the appopriate event

		sc, ok := e.Get("ui", "scrollrestore")
		if !ok {
			return false
		}
		if scrollrestore := sc.(ui.Bool); scrollrestore {
			e.Set("event", "shouldscroll", ui.Bool(true))
		}
		return false
	}))
	return e
})

func EnableScrollRestoration() string {
	return "scrollrestoration"
}

var RouterConfig = func(r *ui.Router) *ui.Router{

	ns:= func(id string) ui.Observable{
		o:= NewObservable(id,EnableSessionPersistence())
		//PutInStorage(o.AsElement()) DEBUG
		return o
	}

	rs:= func(o ui.Observable) ui.Observable{
		e:= LoadFromStorage(o.AsElement())
		return ui.Observable{e}
	}

	r.History.NewState = ns
	r.History.RecoverState = rs
	

	// Add default navigation error handlers
	// notfound:
	pnf:= Div(r.Outlet.AsElement().Root().ID+"-notfound").SetText("Page Not Found.")
	SetAttribute(pnf.AsElement(),"role","alert")
	SetInlineCSS(pnf.AsElement(),`all: initial;`)
	r.OnNotfound(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		v,ok:= r.Outlet.AsElement().Root().Get("navigation", "targetview")
		if !ok{
			panic("targetview should have been set")
		}
		tv:= ui.ViewElement{v.(*ui.Element)}
		if tv.HasStaticView("notfound"){
			tv.ActivateView("notfound")
			return false
		}
		if r.Outlet.HasStaticView("notfound"){
			r.Outlet.ActivateView("notfound")
			return false
		}
		document:=  Document{ui.BasicElement{r.Outlet.AsElement().Root()}}
		body:= document.Body().AsElement()
		body.SetChildren(pnf)
		GetWindow().SetTitle("Page Not Found")

		return false
	}))

	// unauthorized
	ui.AddView("unauthorized",Div(r.Outlet.AsElement().ID+"-unauthorized").SetText("Unauthorized"))(r.Outlet.AsElement())
	r.OnUnauthorized(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		v,ok:= r.Outlet.AsElement().Root().Get("navigation", "targetview")
		if !ok{
			panic("targetview should have been set")
		}
		tv:= ui.ViewElement{v.(*ui.Element)}
		if tv.HasStaticView("unauthorized"){
			tv.ActivateView("unauthorized")
			return false // DEBUG TODO return true?
		}
		r.Outlet.ActivateView("unauthorized")
		return false
	}))

	// appfailure
	afd:= Div("ParticleUI-appfailure").SetText("App Failure")
	r.OnUnauthorized(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		r.Outlet.AsElement().Root().SetChildren(afd)
		return false
	}))

	return r
}


var newObservable = Elements.NewConstructor("observable",func(id string) *ui.Element{
	e:= Elements.GetByID(id)
	if e != nil{
		ui.Delete(e)
	}
	o:= ui.NewObservable(id)
	return o.AsElement()

}, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func NewObservable(id string, options ...string) ui.Observable{
	return ui.Observable{newObservable(id,options...)}
}

type Document struct {
	ui.BasicElement
}

func(d Document) Head() *ui.Element{
	b,ok:= d.AsElement().Get("ui","head")
	if !ok{ return nil}
	return b.(*ui.Element)
}

func(d Document) Body() *ui.Element{
	b,ok:= d.AsElement().Get("ui","body")
	if !ok{ return nil}
	return b.(*ui.Element)
}

func(d Document) SetLang(lang string) Document{
	d.AsElement().SetUI("lang", ui.String(lang))
	return d
}

// ListenAndServe is used to start listening to state changes to the document (aka navigation)
// coming from the browser such as popstate.
// It needs to run at the end, after the UI tree has been built.
func(d Document) ListenAndServe(){
	ui.GetRouter().ListenAndServe("popstate", GetWindow().AsElement())
}

func GetDocumentContaining(e ui.AnyElement) (Document, bool){
	el:= e.AsElement()
	if !el.Mounted(){
		return Document{},false
	}
	return Document{ui.BasicElement{el.Root()}},true
}

var newDocument = Elements.NewConstructor("root", func(id string) *ui.Element {

	e := Elements.NewAppRoot(id).AsElement()

	root := js.Global().Get("document").Get("documentElement")
	if !root.Truthy() {
		log.Print("failed to instantiate root element for the document")
		return e
	}
	n := NewNativeElementWrapper(root)
	e.Native = n
	SetAttribute(e, "id", id)

	e.Watch("ui", "history", GetWindow().AsElement(), ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		e.SyncUI("history", evt.NewValue())
		return false
	}).RunASAP())

	e.Watch("ui","lang",e,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		SetAttribute(evt.Origin(),"lang",string(evt.NewValue().(ui.String)))
		return false
	}).RunASAP())

	e.Watch("ui", "redirectroute", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		v := evt.NewValue()
		nroute, ok := v.(ui.String)
		if !ok {
			panic(nroute)
		}
		route := string(nroute)

		history, ok := e.Get("data", "history")
		if !ok {
			panic("missing history entry")
		} else {
			s := stringify(history.RawValue())
			js.Global().Get("history").Call("replaceState", js.ValueOf(s), "", route)
			e.SetUI("history", history)
		}

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
		history, ok := e.Get("data", "history")
		if !ok {
			panic("missing history entry")
		} else {
			browserhistory, ok := e.Get("ui", "history")
			if !ok {
				s := stringify(history.RawValue())
				js.Global().Get("history").Call("pushState", js.ValueOf(s), "", route)
				e.SetUI("history", history)
				return false
			}
			if ui.Equal(browserhistory, history) {
				return false
			}
			// TODO check if cursors are the same: if they are, state should be updated (use replaceState)
			bhc:= browserhistory.(ui.Object)["cursor"].(ui.Number)
			hc:= history.(ui.Object)["cursor"].(ui.Number)
		
			if bhc==hc {
				s := stringify(history.RawValue())
				js.Global().Get("history").Call("replaceState", js.ValueOf(s), "", route)
				e.SetUI("history", history)
				return false
			}

			s := stringify(history.RawValue())
			js.Global().Get("history").Call("pushState", js.ValueOf(s), "", route)
			e.SetUI("history", history)
		}
		return false
	}))

	// makes ViewElements focusable (focus management support)
	e.Watch("internals", "views",e.Global,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		l:= evt.NewValue().(ui.List)
		view:= l[len(l)-1].(*ui.Element)
		SetAttribute(view,"tabindex","-1")
		e.Watch("ui","activeview",view,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			e.SetUI("focus",view)
			return false
		}))

		return false
	}))

	ui.UseRouter(e,func(r *ui.Router){
		e.AddEventListener("focusin",ui.NewEventHandler(func(evt ui.Event)bool{
			r.History.Set("ui","focus",evt.Target())
			return false
		}))
		
	})

	e.Watch("navigation", "ready", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		r:= ui.GetRouter()
		baseURI:= JSValue(evt.Origin()).Get("baseURI").String() // this is absolute by default
		u,err:= url.ParseRequestURI(baseURI)
		if err!= nil{
			DEBUG(err)
			return false
		}
		r.BasePath = u.Path
		return false
	}))
		
	
	e.Watch("navigation", "ready", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		hstate := js.Global().Get("history").Get("state")
		
		if hstate.Truthy() {
			hstateobj := ui.NewObject()
			err := json.Unmarshal([]byte(hstate.String()), &hstateobj)
			if err == nil {
				GetWindow().AsElement().SetUI("history", hstateobj.Value())
			}
		}

		route := js.Global().Get("location").Get("pathname").String()
		e.Set("navigation", "routechangerequest", ui.String(route))
		return false
	}))
	

	if js.Global().Get("history").Get("scrollRestoration").Truthy() {
		js.Global().Get("history").Set("scrollRestoration", "manual")
	}


	// Adding scrollrestoration support
	e.Watch("navigation", "ready", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		router := ui.GetRouter()

		ejs := js.Global().Get("document").Get("scrollingElement")

		e.SetDataSetUI("scrollrestore", ui.Bool(true))

		GetWindow().AsElement().AddEventListener("scroll", ui.NewEventHandler(func(evt ui.Event) bool {
			scrolltop := ui.Number(ejs.Get("scrollTop").Float())
			scrollleft := ui.Number(ejs.Get("scrollLeft").Float())
			router.History.Set(e.ID, "scrollTop", scrolltop)
			router.History.Set(e.ID, "scrollLeft", scrollleft)
			return false
		}))

		h := ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			newpageaccess:= router.History.CurrentEntryIsNew()
			t, oktop := router.History.Get(e.ID, "scrollTop")
			l, okleft := router.History.Get(e.ID, "scrollLeft")

			if !oktop || !okleft {
				ejs.Set("scrollTop", 0)
				ejs.Set("scrollLeft", 0)
			} else{
				top := t.(ui.Number)
				left := l.(ui.Number)

				ejs.Set("scrollTop", float64(top))
				ejs.Set("scrollLeft", float64(left))
			}
			
			// focus restoration if applicable
			v,ok:= router.History.Get("ui","focus")
			if !ok{
				v,ok= e.Get("ui","focus")
				if !ok{
					DEBUG("expected focus element to exist. Not sure it always does but should check. ***DEBUG***")
					return false
				}
				el:=v.(*ui.Element)
				if el != nil && el.Mounted(){
					focus(JSValue(el))
					if newpageaccess{
						if !partiallyVisible(JSValue(el)){
							DEBUG("focused element not in view...scrolling")
							n.Call("scrollIntoView")
						}
					}
						
				}
			} else{
				el:=v.(*ui.Element)
				if el != nil && el.Mounted(){
					focus(JSValue(el))
					if newpageaccess{
						if !partiallyVisible(JSValue(el)){
							DEBUG("focused element not in view...scrolling")
							n.Call("scrollIntoView")
						}
					}
						
				}
			}
			
			return false
		})
		e.Watch("event", "navigationend", e, h)

		return false
	}))

	e.AppendChild(NewHead("head"))
	e.AppendChild(Body("body"))
	return e
}, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Focus(e ui.AnyElement, scrollintoview bool){
	if !e.AsElement().Mounted(){
		return
	}
	n:= JSValue(e.AsElement())
	focus(n)
	if scrollintoview{
		if !partiallyVisible(n){
			n.Call("scrollIntoView")
		}
	}
}

func focus(e js.Value){
	e.Call("focus",map[string]interface{}{"preventScroll": true})
}

func Autofocus(e *ui.Element) *ui.Element{
	e.OnFirstTimeMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		evt.Origin().Watch("event","navigationend",evt.Origin().Root(),ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			r:= ui.GetRouter()
			if !r.History.CurrentEntryIsNew(){
				return false
			}
			Focus(e,true) // only applies if element is mounted
			return false
		}))
		return false
	}))
	return e
}


func IsInViewPort(n js.Value) bool{
	bounding:= n.Call("getBoundingClientRect")
	top:= int(bounding.Get("top").Float())
	bottom:= int(bounding.Get("bottom").Float())
	left:= int(bounding.Get("left").Float())
	right:= int(bounding.Get("right").Float())

	w:= JSValue(GetWindow().AsElement())
	var ih int
	var iw int
	innerHeight := w.Get("innerHeight")
	if innerHeight.Truthy(){
		ih = int(innerHeight.Float())
		iw = int(w.Get("innerWidth").Float())
	} else{
		ih = int(js.Global().Get("document").Get("documentElement").Get("clientHeight").Float())
		iw = int(js.Global().Get("document").Get("documentElement").Get("clientWidth").Float())
	}
	return (top >= 0) && (left >= 0) && (bottom <= ih) && (right <= iw)	
}

func partiallyVisible(n js.Value) bool{
	bounding:= n.Call("getBoundingClientRect")
	top:= int(bounding.Get("top").Float())
	//bottom:= int(bounding.Get("bottom").Float())
	left:= int(bounding.Get("left").Float())
	//right:= int(bounding.Get("right").Float())

	w:= JSValue(GetWindow().AsElement())
	var ih int
	var iw int
	innerHeight := w.Get("innerHeight")
	if innerHeight.Truthy(){
		ih = int(innerHeight.Float())
		iw = int(w.Get("innerWidth").Float())
	} else{
		ih = int(js.Global().Get("document").Get("documentElement").Get("clientHeight").Float())
		iw = int(js.Global().Get("document").Get("documentElement").Get("clientWidth").Float())
	}
	return (top >= 0) && (left >= 0) && (top <= ih) && (left <= iw)	
}

func TrapFocus(e *ui.Element) *ui.Element{ // TODO what to do if no eleemnt is focusable? (edge-case)
	e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		m:= JSValue(evt.Origin())
		focusableslist:= `button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])`
		focusableElements:= m.Call("querySelectorAll",focusableslist)
		count:= int(focusableElements.Get("length").Float())-1
		firstfocusable:= focusableElements.Index(0)

		lastfocusable:= focusableElements.Index(count)

		h:= ui.NewEventHandler(func(evt ui.Event)bool{
			a:= js.Global().Get("document").Get("activeElement")
			v:=evt.Value().(ui.Object)
			vkey,ok:= v.Get("key")
			if !ok{
				panic("event value is supposed to have a key field.")
			}
			key:= string(vkey.(ui.String))
			if key != "Tab"{
				return false
			}

			if _,ok:= v.Get("shiftKey");ok{
				if a.Equal(firstfocusable){
					focus(lastfocusable)
					evt.PreventDefault()
				}
			} else{
				if a.Equal(lastfocusable){
					focus(firstfocusable)
					evt.PreventDefault()
				}
			}
			return false
		})
		evt.Origin().Root().AddEventListener("keydown",h)
		// Watches unmounted once
		evt.Origin().OnUnmounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			evt.Origin().Root().RemoveEventListener("keydown",h)
			return false
		}).RunOnce())
		
		focus(firstfocusable)

		return false
	}))
	return e
}


// NewDocument returns the root of new js app. It is the top-most element
// in the tree of Elements that consitute the full document.
func NewDocument(id string, options ...string) Document {
	return Document{ui.BasicElement{LoadFromStorage(newDocument(id, options...))}}
}

type BodyElement struct{
	ui.BasicElement
}

var newBody = Elements.NewConstructor("body",func(id string) *ui.Element{
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a body constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "body"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlBody:= js.Global().Get("document").Get("body")
	exist:= htmlBody.Truthy()
	if !exist{
		htmlBody= js.Global().Get("document").Call("createElement","body")
	}else{
		htmlBody = reset(htmlBody)
	}

	n := NewNativeElementWrapper(htmlBody)
	e.Native = n
	if !exist {
		SetAttribute(e, "id", id)
	}

	e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		evt.Origin().Root().Set("ui","body",evt.Origin())
		return false
	}))

	return e
}, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence, AllowScrollRestoration)


func Body(id string, options ...string) BodyElement{
	return BodyElement{ui.BasicElement{LoadFromStorage(newBody(id,options...))}}
}

// reset is used to delete all eventlisteners from an Element
func reset(element js.Value) js.Value {
	clone := element.Call("cloneNode")
	parent := element.Get("parentNode")
	if !parent.IsNull() {
		element.Call("replaceWith", clone)
	}
	return clone
}

// Head refers to the <head> HTML element of a HTML document, which contains metadata and links to 
// resources such as title, scripts, stylesheets.
type Head struct{
	ui.BasicElement
}

var newHead = Elements.NewConstructor("head",func(id string)*ui.Element{
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a head constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "head"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlHead:= js.Global().Get("document").Get("head")
	exist:= htmlHead.Truthy()
	if !exist{
		htmlHead= js.Global().Get("document").Call("createElement","head")
	}else{
		htmlHead = reset(htmlHead)
	}

	n := NewNativeElementWrapper(htmlHead)
	e.Native = n
	if !exist {
		SetAttribute(e, "id", id)
	}

	e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		evt.Origin().Root().Set("ui","head",evt.Origin())
		return false
	}))

	return e
})

func NewHead(id string, options ...string) Head{
	return Head{ui.BasicElement{LoadFromStorage(newHead(id,options...))}}
}

// Meta : for definition and examples, see https://developer.mozilla.org/en-US/docs/Web/HTML/Element/meta
type Meta struct{
	ui.BasicElement
}

func(m Meta) SetAttribute(name,value string) Meta{
	SetAttribute(m.AsElement(),name,value)
	return m
}

var newMeta = Elements.NewConstructor("meta",func(id string)*ui.Element{
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a meta constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "meta"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlMeta:=  js.Global().Get("document").Call("getElementById", id)
	exist := !htmlMeta.IsNull()

	if !exist {
		htmlMeta = js.Global().Get("document").Call("createElement", "meta")
	} else {
		htmlMeta = reset(htmlMeta)
	}

	n := NewNativeElementWrapper(htmlMeta)
	e.Native = n
	if !exist {
		SetAttribute(e, "id", id)
	}

	return e
})

func NewMeta(id string, options ...string) Meta{
	return Meta{ui.BasicElement{LoadFromStorage(newMeta(id,options...))}}
}

// Script is an ELement that refers to the HTML ELement of the same name that embeds executable 
// code or data.
type Script struct{
	ui.BasicElement
}

func(s Script) Src(source string) Script{
	SetAttribute(s.AsElement(),"src",source)
	return s
}

func(s Script) Type(typ string) Script{
	SetAttribute(s.AsElement(),"type",typ)
	return s
}

func(s Script) Async() Script{
	SetAttribute(s.AsElement(),"async","")
	return s
}

func(s Script) Defer() Script{
	SetAttribute(s.AsElement(),"defer","")
	return s
}

func(s Script) SetInnerHTML(content string) Script{
	SetInnerHTML(s.AsElement(),content)
	return s
}

var newScript = Elements.NewConstructor("script",func(id string)*ui.Element{
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a script constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "script"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlScript:=  js.Global().Get("document").Call("getElementById", id)
	exist := !htmlScript.IsNull()

	if !exist {
		htmlScript = js.Global().Get("document").Call("createElement", "script")
	} else {
		htmlScript = reset(htmlScript)
	}

	n := NewNativeElementWrapper(htmlScript)
	e.Native = n
	if !exist {
		SetAttribute(e, "id", id)
	}

	return e
})

func NewScript(id string, options ...string) Script{
	return Script{ui.BasicElement{LoadFromStorage(newScript(id,options...))}}
}

// Base allows to define the baseurl or the basepath for the links within a page.
// In our current use-case, it will mostly be used when generating HTML (SSR or SSG).
// It is then mostly a build-time concern.
type Base struct{
	ui.BasicElement
}

func(b Base) SetHREF(url string) Base{
	b.AsElement().SetUI("href",ui.String(url))
	return b
}

var newBase = Elements.NewConstructor("base",func(id string)*ui.Element{
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a base constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "base"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlBase:=  js.Global().Get("document").Call("getElementById", id)
	exist := !htmlBase.IsNull()

	if !exist {
		htmlBase = js.Global().Get("document").Call("createElement", "base")
	} else {
		htmlBase = reset(htmlBase)
	}

	n := NewNativeElementWrapper(htmlBase)
	e.Native = n
	if !exist {
		SetAttribute(e, "id", id)
	}

	e.Watch("ui","href",e,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		SetAttribute(evt.Origin(),"href",string(evt.NewValue().(ui.String)))
		return false
	}))

	return e
})

func NewBase(id string, options ...string) Base{
	return Base{ui.BasicElement{LoadFromStorage(newBase(id,options...))}}
}


// NoScript refers to an element that defines a section of HTMNL to be inserted in a page if a script
// type is unsupported on the page of scripting is turned off.
// As such, this is mostly useful during SSR or SSG, for examplt to display a message if javascript
// is disabled.
// Indeed, if scripts are disbaled, wasm will not be able to insert this dynamically into the page.
type NoScript struct{
	ui.BasicElement
}

func(s NoScript) SetInnerHTML(content string) NoScript{
	SetInnerHTML(s.AsElement(),content)
	return s
}

var newNoScript = Elements.NewConstructor("noscript",func(id string)*ui.Element{
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a noscript constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "noscript"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlNoScript:=  js.Global().Get("document").Call("getElementById", id)
	exist := !htmlNoScript.IsNull()

	if !exist {
		htmlNoScript = js.Global().Get("document").Call("createElement", "noscript")
	} else {
		htmlNoScript = reset(htmlNoScript)
	}

	n := NewNativeElementWrapper(htmlNoScript)
	e.Native = n
	if !exist {
		SetAttribute(e, "id", id)
	}

	return e
})

func NewNoScript(id string, options ...string) Script{
	return Script{ui.BasicElement{LoadFromStorage(newNoScript(id,options...))}}
}

// Link refers to the <link> HTML Element which allow to specify the location of external resources
// such as stylesheets or a favicon.
type Link struct{
	ui.BasicElement
}

func(l Link) SetAttribute(name,value string) Link{
	SetAttribute(l.AsElement(),name,value)
	return l
}

var newLink = Elements.NewConstructor("link",func(id string)*ui.Element{
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a link constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "link"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlLink:=  js.Global().Get("document").Call("getElementById", id)
	exist := !htmlLink.IsNull()

	if !exist {
		htmlLink = js.Global().Get("document").Call("createElement", "link")
	} else {
		htmlLink = reset(htmlLink)
	}

	n := NewNativeElementWrapper(htmlLink)
	e.Native = n
	if !exist {
		SetAttribute(e, "id", id)
	}

	return e
})

func NewLink(id string, options ...string) Link{
	return Link{ui.BasicElement{LoadFromStorage(newLink(id,options...))}}
}


// Content Sectioning and other HTML Elements

// DivElement is a concrete type that holds the common interface to Div *ui.Element objects.
// i.e. ui.Element whose constructor name is "div" and represents html div elements.
type DivElement struct {
	ui.BasicElement
}

func (d DivElement) Contenteditable(b bool) DivElement {
	d.AsElement().SetDataSetUI("contenteditable", ui.Bool(b))
	return d
}

func (d DivElement) SetText(str string) DivElement {
	d.AsElement().SetDataSetUI("text", ui.String(str))
	return d
}

var newDiv = Elements.NewConstructor("div", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a div constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "div"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
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

	e.Watch("ui", "text", e, textContentHandler)

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence, AllowScrollRestoration)

// Div is a constructor for html div elements.
// The name constructor argument is used by the framework for automatic route
// and automatic link generation.
func Div(id string, options ...string) DivElement {
	return DivElement{ui.BasicElement{LoadFromStorage(newDiv(id, options...))}}
}

// LoadFromStorage will load an element properties.
func LoadFromStorage(e *ui.Element) *ui.Element {
	lb,ok:=e.Get("event","storesynced")
	if ok{
		if isSynced:=lb.(ui.Bool); isSynced{
			return e
		}
		
	}
	pmode := ui.PersistenceMode(e)
	storage, ok := e.ElementStore.PersistentStorer[pmode]
	if ok {
		err := storage.Load(e)
		if err != nil {
			log.Print(err)
			return e
		}
		e.Set("event","storesynced",ui.Bool(true))
	}
	return e
}

// PutInStorage stores an element properties in storage (localstorage or sessionstorage).
func PutInStorage(e *ui.Element) *ui.Element{
	pmode := ui.PersistenceMode(e)
	storage,ok:= e.ElementStore.PersistentStorer[pmode]
	if !ok{
		return e
	}
	for cat,props:= range e.Properties.Categories{
		if cat != "event"{
			for prop,val:= range props.Local{
				storage.Store(e,cat,prop,val)
			}
			for prop,val:=range props.Inheritable{
				storage.Store(e,cat,prop,val,true)
			}
		}		
	}
	e.Set("event","storesynced",ui.Bool(true))
	return e
}

// ClearFromStorage will clear an element properties from storage.
func ClearFromStorage(e *ui.Element) *ui.Element{
	pmode:=ui.PersistenceMode(e)
	storage,ok:= e.ElementStore.PersistentStorer[pmode]
	if ok{
		storage.Clear(e)
		// reset the categories index/list for the element
		idx,ok:= e.Get("index","categories")
		if ok{
			index:=idx.(ui.List)[:0]
			e.Set("index","categories",index)
		}
	}
	return e
}

// isPersisted checks whether an element exist in storage alrready
func isPersisted(e *ui.Element) bool{
	pmode:=ui.PersistenceMode(e)

	var s string
	switch pmode{
	case"sessionstorage":
		s = "sessionStorage"
	case "localstorage":
		s = "localStorage"
	default:
		return false
	}

	store := jsStore{js.Global().Get(s)}
	_, ok := store.Get(e.ID)
	return ok
}

// TODO implement spellcheck and autocomplete methods
type TextAreaElement struct {
	ui.BasicElement
}

func (t TextAreaElement) Text() string {
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

func (t TextAreaElement) SetText(text string) TextAreaElement {
	t.AsElement().SetDataSetUI("text", ui.String(text))
	return t
}

func (t TextAreaElement) SetCols(i int) TextAreaElement {
	t.AsElement().SetDataSetUI("cols", ui.Number(i))
	return t
}

func (t TextAreaElement) SetRows(i int) TextAreaElement {
	t.AsElement().SetDataSetUI("rows", ui.Number(i))
	return t
}

func (t TextAreaElement) SetMinLength(m int) TextAreaElement{
	SetAttribute(t.AsElement(),"minlength",strconv.Itoa(m))
	return t
}

func (t TextAreaElement) SetMaxLength(m int) TextAreaElement{
	SetAttribute(t.AsElement(),"maxlength",strconv.Itoa(m))
	return t
}

func (t TextAreaElement) SetForm(formid string) TextAreaElement{
	SetAttribute(t.AsElement(),"form",formid)
	return t
}

func (t TextAreaElement) SetName(name string) TextAreaElement{
	SetAttribute(t.AsElement(),"name",name)
	return t
}

func (t TextAreaElement) SetPlaceholder(text string) TextAreaElement{
	SetAttribute(t.AsElement(),"placeholder",text)
	return t
}

// SetWrap allows to define how text should wrap. "soft" by default, it can be "hard" or "off".
func(t TextAreaElement) SetWrap(mod string) TextAreaElement{
	v:= "sofft"
	if mod == "hard" || mod == "off"{
		v = mod
	}
	t.AsElement().SetUI("wrap",ui.String(v))
	return t
}

func(t TextAreaElement) Required() TextAreaElement{
	SetAttribute(t.AsElement(),"required","")
	return t
}

func(t TextAreaElement) ReadOnly() TextAreaElement{
	SetAttribute(t.AsElement(),"readonly","")
	return t
}

func(t TextAreaElement) Disabled(b bool) TextAreaElement{
	t.AsElement().SetDataSetUI("disabled",ui.Bool(b))
	return t
}

var newTextArea = Elements.NewConstructor("textarea", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a textarea constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "textarea"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
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
			old := JSValue(evt.Origin()).Get("value").String()
			if string(s) != old {
				SetAttribute(evt.Origin(), "value", string(s))
			}
		}
		return false
	}))

	e.Watch("ui", "rows", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		SetAttribute(evt.Origin(), "rows", strconv.Itoa(int( evt.NewValue().(ui.Number))))
		return false
	}))

	e.Watch("ui", "wrap", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		SetAttribute(evt.Origin(), "wrap", string(evt.NewValue().(ui.String)))
		return false
	}))

	e.Watch("ui", "cols", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		if n, ok := evt.NewValue().(ui.Number); ok {
			SetAttribute(evt.Origin(), "rows", strconv.Itoa(int(n)))
			return false
		}
		return true
	}))

	e.Watch("ui","disabled",e,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		SetAttribute(evt.Origin(),"disabled",strconv.FormatBool(bool(evt.NewValue().(ui.Bool))))
		return false
	}))

	n := NewNativeElementWrapper(htmlTextArea)
	e.Native = n

	SetAttribute(e, "id", id)
	return e
}, allowTextAreaDataBindingOnBlur, allowTextAreaDataBindingOnInput, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)


// TextArea is a constructor for a textarea html element.
func TextArea(id string, options ...string) TextAreaElement {
	e:= newTextArea(id, options...)
	return TextAreaElement{ui.BasicElement{LoadFromStorage(e)}}
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
			e.AddEventListener("blur", callback)
			return e
		}
		mode := datacapturemode[0]
		if mode == onInput {
			e.AddEventListener("input", callback)
			return e
		}

		// capture textarea value on blur by default
		e.AddEventListener("blur", callback)
		return e
	}
}

type HeaderElement struct {
	ui.BasicElement
}

var newHeader= Elements.NewConstructor("header", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a header constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "header"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
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

// Header is a constructor for a html header element.
func Header(id string, options ...string) HeaderElement {
	return HeaderElement{ui.BasicElement{LoadFromStorage(newHeader(id, options...))}}
}

type FooterElement struct {
	ui.BasicElement
}

var newFooter= Elements.NewConstructor("footer", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a footer constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "footer"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
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

// Footer is a constructor for an html footer element.
func Footer(id string, options ...string) FooterElement {
	return FooterElement{ui.BasicElement{LoadFromStorage(newFooter(id, options...))}}
}

type SectionElement struct {
	ui.BasicElement
}

var newSection= Elements.NewConstructor("section", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a section constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "section"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
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

// Section is a constructor for html section elements.
func Section(id string, options ...string) SectionElement {
	return SectionElement{ui.BasicElement{LoadFromStorage(newSection(id, options...))}}
}

type H1Element struct {
	ui.BasicElement
}

func (h H1Element) SetText(s string) H1Element {
	h.AsElement().SetDataSetUI("text", ui.String(s))
	return h
}

var newH1= Elements.NewConstructor("h1", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a h1 constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "h1"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
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

	e.Watch("ui", "text", e, textContentHandler)
	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

// H1 is a constructor for html heading H1 elements.
func H1(id string, options ...string) H1Element {
	return H1Element{ui.BasicElement{LoadFromStorage(newH1(id, options...))}}
}

type H2Element struct {
	ui.BasicElement
}

func (h H2Element) SetText(s string) H2Element {
	h.AsElement().SetDataSetUI("text", ui.String(s))
	return h
}

var newH2= Elements.NewConstructor("h2", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a h2 constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "h2"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
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

	e.Watch("ui", "text", e,textContentHandler)
	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

// H2 is a constructor for html heading H2 elements.
func H2(id string, options ...string) H2Element {
	return H2Element{ui.BasicElement{LoadFromStorage(newH2(id, options...))}}
}

type H3Element struct {
	ui.BasicElement
}

func (h H3Element) SetText(s string) H3Element {
	h.AsElement().SetDataSetUI("text", ui.String(s))
	return h
}

var newH3= Elements.NewConstructor("h3", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a h3 constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "h3"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
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

	e.Watch("ui", "text", e,textContentHandler)
	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

// H3 is a constructor for html heading H3 elements.
func H3(id string, options ...string) H3Element {
	return H3Element{ui.BasicElement{LoadFromStorage(newH3(id, options...))}}
}

type H4Element struct {
	ui.BasicElement
}

func (h H4Element) SetText(s string) H4Element {
	h.AsElement().SetDataSetUI("text", ui.String(s))
	return h
}

var newH4= Elements.NewConstructor("h4", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a h4 constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "h4"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
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

	e.Watch("ui", "text", e, textContentHandler)
	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

// H4 is a constructor for html heading H4 elements.
func H4(id string, options ...string) H4Element {
	return H4Element{ui.BasicElement{LoadFromStorage(newH4(id, options...))}}
}

type H5Element struct {
	ui.BasicElement
}

func (h H5Element) SetText(s string) H5Element {
	h.AsElement().SetDataSetUI("text", ui.String(s))
	return h
}

var newH5= Elements.NewConstructor("h5", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a h5 constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "h5"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
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

	e.Watch("ui", "text", e, textContentHandler)
	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

// H5 is a constructor for html heading H5 elements.
func H5(id string, options ...string) H5Element {
	return H5Element{ui.BasicElement{LoadFromStorage(newH5(id, options...))}}
}

type H6Element struct {
	ui.BasicElement
}

func (h H6Element) SetText(s string) H6Element {
	h.AsElement().SetDataSetUI("text", ui.String(s))
	return h
}

var newH6= Elements.NewConstructor("h6", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a h6 constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "h6"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
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

	e.Watch("ui", "text", e,textContentHandler)
	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)


// H6 is a constructor for html heading H6 elements.
func H6(id string, options ...string) H6Element {
	return H6Element{ui.BasicElement{LoadFromStorage(newH6(id, options...))}}
}

type SpanElement struct {
	ui.BasicElement
}

func (s SpanElement) SetText(str string) SpanElement {
	s.AsElement().SetDataSetUI("text", ui.String(str))
	return s
}

var newSpan= Elements.NewConstructor("span", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a span constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "span"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
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

	e.Watch("ui", "text", e, textContentHandler)

	if !exist {
		SetAttribute(e, "id", id)
	}
	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

// Span is a constructor for html span elements.
func Span(id string, options ...string) SpanElement {
	return SpanElement{ui.BasicElement{LoadFromStorage(newSpan(id, options...))}}
}

type ArticleElement struct {
	ui.BasicElement
}


var newArticle= Elements.NewConstructor("article", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a <article> constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "article"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlArticle := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlArticle.IsNull()
	if !exist {
		htmlArticle = js.Global().Get("document").Call("createElement", "article")
	} else {
		htmlArticle = reset(htmlArticle)
	}

	n := NewNativeElementWrapper(htmlArticle)
	e.Native = n

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions


	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Article(id string, options ...string) ArticleElement {
	return ArticleElement{ui.BasicElement{LoadFromStorage(newArticle(id, options...))}}
}


type AsideElement struct {
	ui.BasicElement
}

var newAside= Elements.NewConstructor("aside", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a <aside> constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "aside"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlAside := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlAside.IsNull()
	if !exist {
		htmlAside = js.Global().Get("document").Call("createElement", "aside")
	} else {
		htmlAside = reset(htmlAside)
	}

	n := NewNativeElementWrapper(htmlAside)
	e.Native = n

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions


	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Aside(id string, options ...string) AsideElement {
	return AsideElement{ui.BasicElement{LoadFromStorage(newAside(id, options...))}}
}

type MainElement struct {
	ui.BasicElement
}

var newMain= Elements.NewConstructor("main", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a <main> constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "main"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlMain := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlMain.IsNull()
	if !exist {
		htmlMain = js.Global().Get("document").Call("createElement", "main")
	} else {
		htmlMain = reset(htmlMain)
	}

	n := NewNativeElementWrapper(htmlMain)
	e.Native = n

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions


	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Main(id string, options ...string) MainElement {
	return MainElement{ui.BasicElement{LoadFromStorage(newMain(id, options...))}}
}


type ParagraphElement struct {
	ui.BasicElement
}

func (p ParagraphElement) SetText(s string) ParagraphElement {
	p.AsElement().SetDataSetUI("text", ui.String(s))
	return p
}

var newParagraph= Elements.NewConstructor("p", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a p constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "p"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
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
		JSValue(evt.Origin()).Set("innerText", string(evt.NewValue().(ui.String)))
		return false
	}))
	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

// Paragraph is a constructor for html paragraph elements.
func Paragraph(id string, options ...string) ParagraphElement {
	return ParagraphElement{ui.BasicElement{LoadFromStorage(newParagraph(id, options...))}}
}

type NavElement struct {
	ui.BasicElement
}

var newNav= Elements.NewConstructor("nav", func(id string) *ui.Element {
		e:= Elements.GetByID(id)
		if e!= nil{
			// Let's check that this element's constructory is a body constructor
			c,ok:= e.Get("internals","constructor")
			if !ok{
				panic("a UI element without the constructor property, should not be happening")
			}
			if s:= string(c.(ui.String)); s == "nav"{
				return e
			}	
		}
		e = ui.NewElement(id, Elements.DocType)
		e = enableClasses(e)

		htmlNav := js.Global().Get("document").Call("getElementById", id)
		exist := !htmlNav.IsNull()
		if !exist {
			htmlNav = js.Global().Get("document").Call("createElement", "nav")
		}

		n := NewNativeElementWrapper(htmlNav)
		e.Native = n

		if !exist {
			SetAttribute(e, "id", id)
		}

		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

// Nav is a constructor for a html nav element.
func Nav(id string, options ...string) NavElement {
	return NavElement{ui.BasicElement{LoadFromStorage(newNav(id, options...))}}
}


type AnchorElement struct {
	ui.BasicElement
}

func (a AnchorElement) SetHREF(target string) AnchorElement {
	a.AsElement().SetDataSetUI("href", ui.String(target))
	return a
}

func (a AnchorElement) FromLink(link ui.Link,  targetid ...string) AnchorElement {
	var hash string
	var id string
	if len(targetid) ==1{
		if targetid[0] != ""{
			id = targetid[0]
			hash = "#"+targetid[0]
		}
	}
	a.AsElement().Watch("event", "verified", link, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		a.SetHREF(link.URI()+hash)
		return false
	}).RunASAP())

	a.AsElement().Watch("data", "active", link, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		a.AsElement().SetDataSetUI("active", evt.NewValue())
		return false
	}).RunASAP())


	a.AsElement().AddEventListener("click", ui.NewEventHandler(func(evt ui.Event) bool {
		v:=evt.Value().(ui.Object)
		rb,ok:= v.Get("ctrlKey")
		if ok{
			if b:=rb.(ui.Bool);b{
				return false
			}
		}
		evt.PreventDefault()
		link.Activate(id)
		return false
	}))

	a.AsElement().SetData("link",link.AsElement())

	pm,ok:= a.AsElement().Get("internals","prefetchmode")
	if ok && !prefetchDisabled(){
		switch t:= string(pm.(ui.String));t{
		case "intent":
			a.AsElement().AddEventListener("mouseover",ui.NewEventHandler(func(evt ui.Event)bool{
				link.Prefetch()
				return false
			}))
		case "render":
			a.AsElement().OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
				link.Prefetch()
				return false
			}))
		}
	} else if !prefetchDisabled(){ // make prefetchable on intent by default
		a.AsElement().AddEventListener("mouseover",ui.NewEventHandler(func(evt ui.Event)bool{
			link.Prefetch()
			return false
		}))
	}

	return a
}

func (a AnchorElement) OnActive(h *ui.MutationHandler) AnchorElement {
	a.AsElement().Watch("ui", "active", a, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		b := evt.NewValue().(ui.Bool)
		if !b {
			return false
		}
		return h.Handle(evt)
	}).RunASAP())
	return a
}

func (a AnchorElement) OnInactive(h *ui.MutationHandler) AnchorElement {
	a.AsElement().Watch("ui", "active", a, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		b := evt.NewValue().(ui.Bool)
		if b {
			return false
		}
		return h.Handle(evt)
	}).RunASAP())
	return a
}

func (a AnchorElement) SetText(text string) AnchorElement {
	a.AsElement().SetDataSetUI("text", ui.String(text))
	return a
}

var newAnchor= Elements.NewConstructor("a", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a anchor constructor 'a'
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "a"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
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

	withSimpleAttributeWatcher(e,"href")

	e.Watch("ui", "text", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		JSValue(evt.Origin()).Set("text", string(evt.NewValue().(ui.String)))
		return false
	}))

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence, AllowPrefetchOnIntent, AllowPrefetchOnRender)

// Anchor creates an html anchor element.
func Anchor(id string, options ...string) AnchorElement {
	return AnchorElement{ui.BasicElement{LoadFromStorage(newAnchor(id, options...))}}
}

var AllowPrefetchOnIntent = ui.NewConstructorOption("prefetchonintent", func(e *ui.Element)*ui.Element{
	if !prefetchDisabled(){
		e.Set("internals","prefetchmode",ui.String("intent"))
	}
	return e
})

var AllowPrefetchOnRender = ui.NewConstructorOption("prefetchonrender", func(e *ui.Element)*ui.Element{
	if !prefetchDisabled(){
		e.Set("internals","prefetchmode",ui.String("render"))
	}
	return e
})

func EnablePrefetchOnIntent() string{
	return "prefetchonintent"
}

func EnablePrefetchOnRender() string{
	return "prefetchonrender"
}

func SetPrefetchMaxAge(t time.Duration){
	ui.PrefetchMaxAge = t
}

func DisablePrefetching(){
	ui.PrefetchMaxAge = -1
}

func prefetchDisabled() bool{
	return ui.PrefetchMaxAge < 0
}

type ButtonElement struct {
	ui.BasicElement
}

// Autofocus is mainly useful for SSR/SSG. Since SPA pages are rarely reloaded, the Autofocus modifer
// should be used instead.
func (b ButtonElement) Autofocus(t bool) ButtonElement {
	b.AsElement().SetDataSetUI("autofocus", ui.Bool(t))
	return b
}

func (b ButtonElement) Disabled(t bool) ButtonElement {
	b.AsElement().SetDataSetUI("disabled", ui.Bool(t))
	return b
}

func(b ButtonElement) SetForm(formid string) ButtonElement{
	SetAttribute(b.AsElement(),"form",formid)
	return b
}

func (b ButtonElement) SetText(str string) ButtonElement {
	b.AsElement().SetDataSetUI("text", ui.String(str))
	return b
}

var newButton= Elements.NewConstructor("button", func(id  string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a button constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "button"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
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


	withBoolAttributeWatcher(e,"disabled")
	withBoolAttributeWatcher(e,"autofocus")

	withSimpleAttributeWatcher(e,"form")
	withSimpleAttributeWatcher(e,"type")
	withSimpleAttributeWatcher(e,"name")

	e.Watch("ui", "text", e, textContentHandler)

	SetAttribute(e, "id", id)
	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

// Button returns a button ui.BasicElement.
// TODO (add attribute watchers for form button element)
func Button(typ string, id string, options ...string) ButtonElement {
	e:= newButton(id, options...)
	SetAttribute(e, "type", typ)
	return ButtonElement{ui.BasicElement{LoadFromStorage(e)}}
}

type LabelElement struct {
	ui.BasicElement
}

func (l LabelElement) SetText(s string) LabelElement {
	l.AsElement().SetUI("text", ui.String(s))
	return l
}

func (l LabelElement) For(elementid string) LabelElement {
	l.AsElement().SetUI("for", ui.String(elementid))
	return l
}

var newLabel= Elements.NewConstructor("label", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a label constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "label"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
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
	withSimpleAttributeWatcher(e,"for")
	e.Watch("ui", "text", e, textContentHandler)
	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Label(id string, options ...string) LabelElement {
	return LabelElement{ui.BasicElement{LoadFromStorage(newLabel(id, options...))}}
}

type InputElement struct {
	ui.BasicElement
}

func (i InputElement) Value() ui.String {
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

func (i InputElement) Blur() {
	native, ok := i.AsElement().Native.(NativeElement)
	if !ok {
		panic("native element should be of doc.NativeELement type")
	}
	native.Value.Call("blur")
}

func (i InputElement) Focus() {
	native, ok := i.AsElement().Native.(NativeElement)
	if !ok {
		panic("native element should be of doc.NativeELement type")
	}
	native.Value.Call("focus")
}

func (i InputElement) Clear() {
	native, ok := i.AsElement().Native.(NativeElement)
	if !ok {
		panic("native element should be of doc.NativeELement type")
	}
	native.Value.Set("value", "")
}

// Attr allows to set attributes on the HTML element.
// The value is not conserved in state so it is not dynamically changeable by setting a value on the 
// ui.Element.
func(i InputElement) Attr(name, value string) InputElement{
	SetAttribute(i.AsElement(),name,value)
	return i
}

var newInputElement= Elements.NewConstructor("input", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is an input constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "input"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
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
		JSValue(evt.Origin()).Set("value", string(evt.NewValue().(ui.String)))
		return false
	}))

	withSimpleAttributeWatcher(e,"accept")

	withBoolAttributeWatcher(e,"autocomplete")

	withSimpleAttributeWatcher(e,"capture")

	e.Watch("ui", "checked", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		if evt.NewValue().(ui.Bool) {
			JSValue(evt.Origin()).Set("checked", true)
			return false
		}
		JSValue(evt.Origin()).Set("checked", false)
		return false
	}))

	withBoolAttributeWatcher(e,"disabled")

	withSimpleAttributeWatcher(e,"inputmode")

	e.Watch("ui", "maxlength", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		if i:= evt.NewValue().(ui.Number); int(i) > 0 {
			SetAttribute(evt.Origin(), "maxlength", strconv.Itoa(int(i)))
			return false
		}
		RemoveAttribute(evt.Origin(), "maxlength")
		return false
	}))

	e.Watch("ui", "minlength", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		if i:= evt.NewValue().(ui.Number); int(i) > 0 {
			SetAttribute(evt.Origin(), "minlength", strconv.Itoa(int(i)))
			return false
		}
		RemoveAttribute(evt.Origin(), "minlength")
		return false
	}))

	e.Watch("ui", "step", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		if i:= evt.NewValue().(ui.Number); int(i) > 0 {
			SetAttribute(evt.Origin(), "step", strconv.Itoa(int(i)))
			return false
		}
		RemoveAttribute(evt.Origin(), "step")
		return false
	}))

	e.Watch("ui", "min", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		SetAttribute(evt.Origin(), "min", strconv.Itoa(int(evt.NewValue().(ui.Number))))
		return false
	}))

	e.Watch("ui", "max", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		SetAttribute(evt.Origin(), "max", strconv.Itoa(int(evt.NewValue().(ui.Number))))
		return false
	}))

	e.Watch("ui", "multiple", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		if evt.NewValue().(ui.Bool){
			SetAttribute(evt.Origin(), "multiple", "")
			return false
		}
		RemoveAttribute(evt.Origin(), "multiple")
		return false
	}))

	withSimpleAttributeWatcher(e,"type")

	SetAttribute(e, "id", id)
	return e
}, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence, inputOption("radio"), 
	inputOption("button"), inputOption("checkbox"), inputOption("color"), inputOption("date"), 
	inputOption("datetime-local"), inputOption("email"), inputOption("file"), inputOption("hidden"), 
	inputOption("image"), inputOption("month"), inputOption("number"), inputOption("password"),
	inputOption("range"), inputOption("reset"), inputOption("search"), inputOption("submit"), 
	inputOption("tel"), inputOption("text"), inputOption("time"), inputOption("url"), inputOption("week"))

func inputOption(name string) ui.ConstructorOption{
	return ui.NewConstructorOption(name,func(e *ui.Element)*ui.Element{
		e.SetUI("type",ui.String(name))
		return e
	})
}


func Input(typ string,id string, options ...string) InputElement { // TODO use constructor option for type
	options = append(options, typ)
	e:= newInputElement(id, options...)
	return InputElement{ui.BasicElement{LoadFromStorage(e)}}
}

type ImgElement struct {
	ui.BasicElement
}

func (i ImgElement) Src(s string) ImgElement {
	i.AsElement().SetDataSetUI("src", ui.String(s))
	return i
}

func (i ImgElement) Alt(s string) ImgElement {
	i.AsElement().SetDataSetUI("alt", ui.String(s))
	return i
}

var newImage= Elements.NewConstructor("img", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is an img constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "img"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
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
	SetAttribute(e, "id", id)

	withSimpleAttributeWatcher(e,"src")
	withSimpleAttributeWatcher(e,"alt")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Img(id string, options ...string) ImgElement {
	return ImgElement{ui.BasicElement{LoadFromStorage(newImage(id, options...))}}
}

var newAudio = Elements.NewConstructor("audio", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is an <audio> constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "audio"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)
	if e!= nil{
		// Let's check that this element's constructory is an audio constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "audio"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
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

	SetAttribute(e, "id", id)
	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var newVideo = Elements.NewConstructor("video", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a <video> constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "video"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)
	if e!= nil{
		// Let's check that this element's constructory is a video constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "video"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlVideo := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlVideo.IsNull()
	if !exist {
		htmlVideo = js.Global().Get("document").Call("createElement", "video")
	} else {
		htmlVideo = reset(htmlVideo)
	}

	SetAttribute(e, "id", id)

	n := NewNativeElementWrapper(htmlVideo)
	e.Native = n
	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)


type SourceElement struct{
	ui.BasicElement
}

func(s SourceElement) SetSrc(src string) SourceElement{
	s.AsElement().SetUI("src",ui.String(src))
	return s
}


func(s SourceElement) SetType(typ string) SourceElement{
	s.AsElement().SetUI("type",ui.String(typ))
	return s
}


var newSource = Elements.NewConstructor("source", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a source constructor.
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "source"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlSource := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlSource.IsNull()
	if !exist {
		htmlSource = js.Global().Get("document").Call("createElement", "source")
	} else {
		htmlSource = reset(htmlSource)
	}

	n := NewNativeElementWrapper(htmlSource)
	e.Native = n

	SetAttribute(e, "id", id)

	withSimpleAttributeWatcher(e,"src")
	withSimpleAttributeWatcher(e,"type")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Source(id string, options ...string) SourceElement{
	return SourceElement{ui.BasicElement{LoadFromStorage(newSource(id,options...))}}
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

var newUl= Elements.NewConstructor("ul", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a ul constructor.
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "ul"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
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
	
	SetAttribute(e, "id", id)

	h := ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		list, ok := evt.NewValue().(ui.List)
		if !ok {
			return true
		}

		for i, v := range list {
			item := Elements.GetByID(id + "-item-" + strconv.Itoa(i))
			if item != nil {
				ListItem{ui.BasicElement{item}}.SetValue(v)
			} else {
				item = NewListItem(id+"-item-"+strconv.Itoa(i)).SetValue(v).AsBasicElement().AsElement()
			}

			evt.Origin().AppendChild(ui.BasicElement{item})
		}
		return false
	})
	e.Watch("ui", "list", e, h)

	return e
}, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func NewUl(id string, options ...string) List {
	return List{ui.BasicElement{LoadFromStorage(newUl(id, options...))}}
}

type OrderedList struct {
	ui.BasicElement
}

func (l OrderedList) SetValue(lobjs ui.List) OrderedList {
	l.AsElement().Set("data", "value", lobjs)
	return l
}

var newOl= Elements.NewConstructor("ol", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a ol constructor.
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "ol"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
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

	SetAttribute(e, "id", id)
	
	return e
}, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func NewOl(id string, typ string, numberingstart int, options ...string) OrderedList {
	e:= newOl(id, options...)
	SetAttribute(e, "type", typ)
	SetAttribute(e, "start", strconv.Itoa(numberingstart))
	return OrderedList{ui.BasicElement{LoadFromStorage(e)}}
}

type ListItem struct {
	ui.BasicElement
}

func (li ListItem) SetValue(v ui.Value) ListItem {
	li.AsElement().SetDataSetUI("value", v)
	return li
}

var newListItem= Elements.NewConstructor("li", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a li constructor.
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "li"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
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

func NewListItem(id string, options ...string) ListItem {
	return ListItem{ui.BasicElement{LoadFromStorage(newListItem(id, options...))}}
}

// Table Elements

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

type col struct {
	ui.BasicElement
}

type Tfoot struct {
	ui.BasicElement
}

var newThead= Elements.NewConstructor("thead", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a thead constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "thead"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
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

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func NewThead(id string, options ...string) Thead {
	return Thead{ui.BasicElement{LoadFromStorage(newThead(id, options...))}}
}


var newTr= Elements.NewConstructor("tr", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a tr constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "tr"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
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

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func NewTr(id string, options ...string) Tr {
	return Tr{ui.BasicElement{LoadFromStorage(newTr(id, options...))}}
}

var newTd= Elements.NewConstructor("td", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a td constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "td"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
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

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func NewTd(id string, options ...string) Td {
	return Td{ui.BasicElement{LoadFromStorage(newTd(id, options...))}}
}

var newTh= Elements.NewConstructor("th", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a th constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "th"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
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

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func NewTh(id string, options ...string) Th {
	return Th{ui.BasicElement{LoadFromStorage(newTh(id, options...))}}
}

var newTable= Elements.NewConstructor("table", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a table constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "table"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
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

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func NewTable(id string, options ...string) Table {
	return Table{ui.BasicElement{LoadFromStorage(newTable(id, options...))}}
}



// Code is typically used to indicate that the text it contains is computer code and may therefore be 
// formatted differently.
// To represent multiple lines of code, wrap the <code> element within a <pre> element. 
// The <code> element by itself only represents a single phrase of code or line of code.
type Code struct {
	ui.BasicElement
}

func (c Code) SetText(str string) Code {
	c.AsElement().SetDataSetUI("text", ui.String(str))
	return c
}

var newCode= Elements.NewConstructor("code", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		// Let's check that this element's constructory is a <code> constructor
		c,ok:= e.Get("internals","constructor")
		if !ok{
			panic("a UI element without the constructor property, should not be happening")
		}
		if s:= string(c.(ui.String)); s == "code"{
			return e
		}	
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlCode := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlCode.IsNull()
	if !exist {
		htmlCode = js.Global().Get("document").Call("createElement", "code")
	} else {
		htmlCode = reset(htmlCode)
	}

	n := NewNativeElementWrapper(htmlCode)
	e.Native = n

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions

	e.Watch("ui", "text", e, textContentHandler)

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func NewCode(id string, options ...string) Code {
	return Code{ui.BasicElement{LoadFromStorage(newCode(id, options...))}}
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
	c = strings.ReplaceAll(c, classname, " ")

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


// Buttonifyier returns en element modifier that can turn an element into a clickable non-anchor 
// naviagtion element.
func Buttonifyier(link ui.Link) func(*ui.Element) *ui.Element {
	callback := ui.NewEventHandler(func(evt ui.Event) bool {
		link.Activate()
		return false
	})
	return func(e *ui.Element)*ui.Element{
		e.AddEventListener("click",callback)
		return e
	}
}


var textContentHandler = ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
	str, ok := evt.NewValue().(ui.String)
	if !ok {
		return true
	}
	JSValue(evt.Origin()).Set("textContent", string(str))

	return false
})

func withSimpleAttributeWatcher(e *ui.Element,attr string){
	e.Watch("ui",attr,e,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		SetAttribute(evt.Origin(),attr,string(evt.NewValue().(ui.String)))
		return false
	}))
}

func withBoolAttributeWatcher(e *ui.Element, attr string){
	e.Watch("ui", attr, e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		if evt.NewValue().(ui.Bool) {
			SetAttribute(evt.Origin(), attr, "")
		}
		RemoveAttribute(evt.Origin(), attr)
		return false
	}))
}


// Attr is a modifier that allows to set the value of an attribute if supported.
// Idf the element is not watching the ui property named after the attribute name, it does nothing.
func Attr(name,value string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI(name,ui.String(value))
		return e
	}
}