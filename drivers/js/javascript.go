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
	Elements                      = ui.NewElementStore("default", DOCTYPE).
		AddPersistenceMode("sessionstorage", loadfromsession, sessionstorefn, clearfromsession).
		AddPersistenceMode("localstorage", loadfromlocalstorage, localstoragefn, clearfromlocalstorage).
		ApplyGlobalOption(cleanStorageOnDelete).
		AddConstructorOptions("observable",AllowSessionStoragePersistence,AllowAppLocalStoragePersistence)
	
	EnablePropertyAutoInheritance = ui.EnablePropertyAutoInheritance
	mainDocument *Document
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
		/*e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			ui.Rerender(e)
			return false
		}).RunOnce())*/
		
		ui.Rerender(e)
		
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
		if e.Native != nil{
			j:= JSValue(e)
			if j.Truthy(){
				j.Call("remove")
			}
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
	//e.Set("event", "attached", ui.Bool(true))
	//e.Set("event", "firstmount", ui.Bool(true))
	//e.Set("event", "firsttimemounted", ui.Bool(true)) TODO DEBUG
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
func JSValue(el ui.AnyElement) js.Value {
	e:= el.AsElement()
	n, ok := e.Native.(NativeElement)
	if !ok {
		DEBUG(e.ID)
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
	e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
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
	}).RunASAP().RunOnce())

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


var newObservable = Elements.Constructors["observable"]

func NewObservable(id string, options ...string) ui.Observable{
	e:= Elements.GetByID(id)
	if e != nil{
		ui.Delete(e)
	}
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
	d.AsElement().SetDataSetUI("lang", ui.String(lang))
	return d
}

func (d Document) OnNavigationEnd(h *ui.MutationHandler){
	d.AsElement().Watch("event","navigationend", d, h)
}

// ListenAndServe is used to start listening to state changes to the document (aka navigation)
// coming from the browser such as popstate.
// It needs to run at the end, after the UI tree has been built.
func(d Document) ListenAndServe(){
	if mainDocument ==nil{
		panic("document is missing")
	}
	ui.GetRouter().ListenAndServe("popstate", GetWindow().AsElement())
}

func GetDocument() Document{
	return *mainDocument
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

	e.Watch("ui","lang",e,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		SetAttribute(evt.Origin(),"lang",string(evt.NewValue().(ui.String)))
		return false
	}).RunASAP())

  

    e.Watch("ui","history",e,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
        var route string
        r,ok:= evt.Origin().Get("ui","currentroute")
        if !ok{
            panic("current route is unknown")
        }
        route = string(r.(ui.String))

        history:= evt.NewValue().(ui.Object)
        browserhistory,ok:= evt.OldValue().(ui.Object)
        if ok{
            bhc:= browserhistory["cursor"].(ui.Number)
            hc:= history["cursor"].(ui.Number)
            if bhc==hc {
                s := stringify(history.RawValue())
                js.Global().Get("history").Call("replaceState", js.ValueOf(s), "", route)
            } else{
				s := stringify(history.RawValue())
        		js.Global().Get("history").Call("pushState", js.ValueOf(s), "", route)
			}
			return false
        }

        s := stringify(history.RawValue())
        js.Global().Get("history").Call("replaceState", js.ValueOf(s), "", route)
        return false
    }))

	// makes ViewElements focusable (focus management support)
	e.Watch("internals", "views",e.Global,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		l:= evt.NewValue().(ui.List)
		view:= l[len(l)-1].(*ui.Element)
		SetAttribute(view,"tabindex","-1")
		e.Watch("ui","activeview",view,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			e.SetDataSetUI("focus",view)
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
				evt.Origin().SyncUISetData("history", hstateobj.Value())
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
	e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		evt.Origin().Watch("event","navigationend",evt.Origin().Root(),ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			r:= ui.GetRouter()
			if !r.History.CurrentEntryIsNew(){
				return false
			}
			Focus(e,true) // only applies if element is mounted
			return false
		}))
		return false
	}).RunASAP().RunOnce())
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
	mainDocument = &Document{ui.BasicElement{LoadFromStorage(newDocument(id, options...))}}
	return GetDocument()
}

type BodyElement struct{
	ui.BasicElement
}

var newBody = Elements.NewConstructor("body",func(id string) *ui.Element{
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlBody:= js.Global().Get("document").Get("body")
	exist:= htmlBody.Truthy()
	if !exist{
		htmlBody= js.Global().Get("document").Call("createElement","body")
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
func resete(element js.Value) js.Value {
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
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlHead:= js.Global().Get("document").Get("head")

	exist:= htmlHead.Truthy()
	if !exist{
		htmlHead= js.Global().Get("document").Call("createElement","head")
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
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlMeta:=  js.Global().Get("document").Call("getElementById", id)
	exist := !htmlMeta.IsNull()

	if !exist {
		htmlMeta = js.Global().Get("document").Call("createElement", "meta")
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
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlScript:=  js.Global().Get("document").Call("getElementById", id)
	exist := !htmlScript.IsNull()

	if !exist {
		htmlScript = js.Global().Get("document").Call("createElement", "script")
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
	b.AsElement().SetDataSetUI("href",ui.String(url))
	return b
}

var newBase = Elements.NewConstructor("base",func(id string)*ui.Element{
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlBase:=  js.Global().Get("document").Call("getElementById", id)
	exist := !htmlBase.IsNull()

	if !exist {
		htmlBase = js.Global().Get("document").Call("createElement", "base")
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
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlNoScript:=  js.Global().Get("document").Call("getElementById", id)
	exist := !htmlNoScript.IsNull()

	if !exist {
		htmlNoScript = js.Global().Get("document").Call("createElement", "noscript")
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
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlLink:=  js.Global().Get("document").Call("getElementById", id)
	exist := !htmlLink.IsNull()

	if !exist {
		htmlLink = js.Global().Get("document").Call("createElement", "link")
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
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
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

type textAreaModifer struct{}
var TextAreaModifer textAreaModifer

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

func(t textAreaModifer) Cols(i int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("cols",ui.Number(i))
		return e
	}
}

func(t textAreaModifer) Rows(i int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("rows",ui.Number(i))
		return e
	}
}

func(t textAreaModifer) MinLength(i int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("minlength",ui.Number(i))
		return e
	}
}

func(t textAreaModifer) MaxLength(i int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("maxlength",ui.Number(i))
		return e
	}
}
func(t textAreaModifer) Required(b bool) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("required",ui.Bool(b))
		return e
	}
}

func (t textAreaModifer) Form(form *ui.Element) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			d:= GetDocument()
			
			evt.Origin().Watch("event","navigationend",d,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
				if form.Mounted(){
					e.AsElement().SetDataSetUI("form", ui.String(form.ID))
				}
				return false
			}).RunOnce())
			return false
		}).RunOnce())
		return e
	}
}

func(t textAreaModifer) Name(name string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("name",ui.String(name))
		return e
	}
}

func(t textAreaModifer) Placeholder(p string)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("placeholder",ui.String(p))
		return e
	}
}


// Wrap allows to define how text should wrap. "soft" by default, it can be "hard" or "off".
func(t textAreaModifer) Wrap(mode string)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		v:= "soft"
		if mode == "hard" || mode == "off"{
			v = mode
		}
		e.SetDataSetUI("wrap",ui.String(v))
		return e
	}
}

func(t textAreaModifer) Autocomplete(on bool)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		var val string
		if on{
			val = "on"
		}else{
			val = "off"
		}
		e.SetDataSetUI("autocomplete",ui.String(val))
		return e
	}
}

func(t textAreaModifer) Spellcheck(mode string)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		v:= "default"
		if mode == "true" || mode == "false"{
			v = mode
		}
		e.SetDataSetUI("spellcheck",ui.String(v))
		return e
	}
}

var newTextArea = Elements.NewConstructor("textarea", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlTextArea := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlTextArea.IsNull()

	if !exist {
		htmlTextArea = js.Global().Get("document").Call("createElement", "textarea")
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

	withNumberAttributeWatcher(e,"rows")
	withNumberAttributeWatcher(e,"cols")

	withStringAttributeWatcher(e,"wrap")

	withBoolAttributeWatcher(e,"disabled")
	withBoolAttributeWatcher(e,"required")
	withStringAttributeWatcher(e,"name")
	withBoolAttributeWatcher(e,"readonly")
	withStringAttributeWatcher(e,"autocomplete")
	withStringAttributeWatcher(e,"spellcheck")

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
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
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
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
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
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlSection := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlSection.IsNull()
	if !exist {
		htmlSection = js.Global().Get("document").Call("createElement", "section")
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
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlH1 := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlH1.IsNull()
	if !exist {
		htmlH1 = js.Global().Get("document").Call("createElement", "h1")
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
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlH2 := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlH2.IsNull()
	if !exist {
		htmlH2 = js.Global().Get("document").Call("createElement", "h2")
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
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlH3 := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlH3.IsNull()
	if !exist {
		htmlH3 = js.Global().Get("document").Call("createElement", "h3")
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
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlH4 := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlH4.IsNull()
	if !exist {
		htmlH4 = js.Global().Get("document").Call("createElement", "h4")
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
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlH5 := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlH5.IsNull()
	if !exist {
		htmlH5 = js.Global().Get("document").Call("createElement", "h5")
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
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlH6 := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlH6.IsNull()
	if !exist {
		htmlH6 = js.Global().Get("document").Call("createElement", "h6")
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
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlSpan := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlSpan.IsNull()
	if !exist {
		htmlSpan = js.Global().Get("document").Call("createElement", "span")
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
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlArticle := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlArticle.IsNull()
	if !exist {
		htmlArticle = js.Global().Get("document").Call("createElement", "article")
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
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlAside := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlAside.IsNull()
	if !exist {
		htmlAside = js.Global().Get("document").Call("createElement", "aside")
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
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlMain := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlMain.IsNull()
	if !exist {
		htmlMain = js.Global().Get("document").Call("createElement", "main")
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
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
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
			panic(id + " : this id is already in use")
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
		if !link.IsActive(){
			link.Activate(id)
		}
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
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlAnchor := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlAnchor.IsNull()
	if !exist {
		htmlAnchor = js.Global().Get("document").Call("createElement", "a")
	} 

	n := NewNativeElementWrapper(htmlAnchor)
	e.Native = n
	if !exist {
		SetAttribute(e, "id", id)
	}

	withStringAttributeWatcher(e,"href")

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

type buttonModifer struct{}
var ButtonModifier buttonModifer

func(m buttonModifer) Disabled(b bool) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("disabled",ui.Bool(b))
		return e
	}
}

func(b buttonModifer) Form(form *ui.Element) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			d:= GetDocument()
			
			evt.Origin().Watch("event","navigationend",d,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
				if form.Mounted(){
					e.SetDataSetUI("form", ui.String(form.ID))
				}
				return false
			}).RunOnce())
			return false
		}).RunOnce())
		return e
	}
}


func (b ButtonElement) SetDisabled(t bool) ButtonElement {
	b.AsElement().SetDataSetUI("disabled", ui.Bool(t))
	return b
}

func (b ButtonElement) SetText(str string) ButtonElement {
	b.AsElement().SetDataSetUI("text", ui.String(str))
	return b
}

var newButton= Elements.NewConstructor("button", func(id  string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlButton := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlButton.IsNull()
	if !exist {
		htmlButton = js.Global().Get("document").Call("createElement", "button")
	} 

	n := NewNativeElementWrapper(htmlButton)
	e.Native = n


	withBoolAttributeWatcher(e,"disabled")
	withBoolAttributeWatcher(e,"autofocus")

	withStringAttributeWatcher(e,"form")
	withStringAttributeWatcher(e,"type")
	withStringAttributeWatcher(e,"name")

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
	l.AsElement().SetDataSetUI("text", ui.String(s))
	return l
}

func (l LabelElement) For(e *ui.Element) LabelElement {
	l.AsElement().OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		d:= GetDocument()
		
		evt.Origin().Watch("event","navigationend",d,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			if e.Mounted(){
				l.AsElement().SetDataSetUI("for", ui.String(e.ID))
			}
			return false
		}).RunOnce())
		return false
	}).RunOnce())
	return l
}

var newLabel= Elements.NewConstructor("label", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlLabel := js.Global().Get("document").Call("getElementById", id)
	if htmlLabel.IsNull() {
		htmlLabel = js.Global().Get("document").Call("createElement", "label")
	}

	n := NewNativeElementWrapper(htmlLabel)
	e.Native = n

	SetAttribute(e, "id", id)
	withStringAttributeWatcher(e,"for")
	e.Watch("ui", "text", e, textContentHandler)
	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Label(id string, options ...string) LabelElement {
	return LabelElement{ui.BasicElement{LoadFromStorage(newLabel(id, options...))}}
}

type InputElement struct {
	ui.BasicElement
}

type inputModifier struct{}
var InputModifier inputModifier

func(i inputModifier) Step(step int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("step",ui.Number(step))
		return e
	}
}

func(i inputModifier) Disabled(b bool) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("disabled",ui.Bool(b))
		return e
	}
}

func(i inputModifier) MaxLength(m int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("maxlength",ui.Number(m))
		return e
	}
}

func(i inputModifier) MinLength(m int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("minlength",ui.Number(m))
		return e
	}
}

func(i inputModifier) Autocomplete(b bool) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("autocomplete",ui.Bool(b))
		return e
	}
}

func(i inputModifier) InputMode(mode string)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("inputmode",ui.String(mode))
		return e
	}
}

func(i inputModifier) Size(s int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("size",ui.Number(s))
		return e
	}
}

func(i inputModifier) Placeholder(p string)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("placeholder",ui.String(p))
		return e
	}
}

func(i inputModifier)Pattern(p string)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("pattern",ui.String(p))
		return e
	}
}

func(i inputModifier) Multiple() func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("multiple",ui.Bool(true))
		return e
	}
}

func(i inputModifier) Required(b bool) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("required",ui.Bool(b))
		return e
	}
}

func(i inputModifier) Accept(accept string)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("accept",ui.String(accept))
		return e
	}
}

func(i inputModifier) Src(src string)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("src",ui.String(src))
		return e
	}
}

func(i inputModifier)Alt(alt string)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("alt",ui.String(alt))
		return e
	}
}

func(i inputModifier) Name(name string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("name",ui.String(name))
		return e
	}
}

func(i inputModifier) Height(h int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("height",ui.Number(h))
		return e
	}
}

func(i inputModifier) Width(w int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("width",ui.Number(w))
		return e
	}
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

func (i InputElement) SetDisabled(b bool)InputElement{
	i.AsElement().SetDataSetUI("disabled",ui.Bool(b))
	return i
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


var newInputElement= Elements.NewConstructor("input", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlInput := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlInput.IsNull()
	if !exist {
		htmlInput = js.Global().Get("document").Call("createElement", "input")
	} 

	n := NewNativeElementWrapper(htmlInput)
	e.Native = n

	e.Watch("ui", "value", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		JSValue(evt.Origin()).Set("value", string(evt.NewValue().(ui.String)))
		return false
	}))

	
	withStringAttributeWatcher(e,"name")
	withBoolAttributeWatcher(e,"disabled")
	withStringAttributeWatcher(e,"form")
	withStringAttributeWatcher(e,"type")
	withStringAttributeWatcher(e,"inputmode")

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

		if name == "file"{
			withStringAttributeWatcher(e,"accept")
			withStringAttributeWatcher(e,"capture")
		}

		if newset("file","email").Contains(name){
			withBoolAttributeWatcher(e,"multiple")
		}

		if newset("checkbox","radio").Contains(name){
			withBoolAttributeWatcher(e,"checked")
		}
		if name == "search" || name == "text"{
			withStringAttributeWatcher(e,"dirname")
		}

		if newset("text","search","url","tel","email","password").Contains(name) {
			withStringAttributeWatcher(e,"pattern")
			withNumberAttributeWatcher(e,"size")
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

		}

		if newset("text","search","url","tel","email","password","number").Contains(name) {
			withStringAttributeWatcher(e,"placeholder")
		}

		if !newset("hidden","range","color","checkbox","radio","button").Contains(name){
			withBoolAttributeWatcher(e,"readonly")
		}

		if !newset("hidden","range","color","button").Contains(name){
			withBoolAttributeWatcher(e,"required")
		}

		if name == "image"{
			withStringAttributeWatcher(e,"src")
			withStringAttributeWatcher(e,"alt")
			withNumberAttributeWatcher(e,"height")
			withNumberAttributeWatcher(e,"width")
		}

		if newset("date","month","week","time","datetime-local","range").Contains(name){
			e.Watch("ui", "step", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
				if i:= evt.NewValue().(ui.Number); int(i) > 0 {
					SetAttribute(evt.Origin(), "step", strconv.Itoa(int(i)))
					return false
				}
				RemoveAttribute(evt.Origin(), "step")
				return false
			}))
			withNumberAttributeWatcher(e, "min")
			withNumberAttributeWatcher(e,"max")
		}

		if !newset("radio","checkbox","button").Contains(name){
			withBoolAttributeWatcher(e,"autocomplete")
		}

		if !newset("hidden","password","radio","checkbox","button").Contains(name){
			withStringAttributeWatcher(e,"list")
		}

		if newset("image","submit").Contains(name){
			withStringAttributeWatcher(e,"formaction")
			withStringAttributeWatcher(e,"formenctype")
			withStringAttributeWatcher(e,"formmethod")
			withBoolAttributeWatcher(e,"formnovalidate")
			withStringAttributeWatcher(e,"formtarget")
		}

		e.SetDataSetUI("type",ui.String(name))		

		return e
	})
}


func Input(typ string,id string, options ...string) InputElement { // TODO use constructor option for type
	if typ != ""{
		options = append(options, typ)
	}
	e:= newInputElement(id, options...)
	return InputElement{ui.BasicElement{LoadFromStorage(e)}}
}

// OutputElement
type OutputElement struct{
	ui.BasicElement
}

type outputModifier struct{}
var OutputModifer outputModifier

func(m outputModifier) Form(form *ui.Element) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			d:= GetDocument()
			
			evt.Origin().Watch("event","navigationend",d,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
				if form.Mounted(){
					e.SetDataSetUI("form", ui.String(form.ID))
				}
				return false
			}).RunOnce())
			return false
		}).RunOnce())
		return e
	}
}

func(m outputModifier) Name(name string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("name",ui.String(name))
		return e
	}
}

func(m outputModifier) For(inputs ...*ui.Element) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		var inputlist string
		e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			d:= GetDocument()
			
			evt.Origin().Watch("event","navigationend",d,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
				
				for _,input:= range inputs{
					if input.Mounted(){
						inputlist += " "+input.ID
					} else{
						panic("input missing for output element "+ e.ID)
					}
				}
				e.SetDataSetUI("for",ui.String(inputlist))
				return false
			}).RunOnce())
			return false
		}).RunOnce())
		
		return e
	}
}


var newOutput = Elements.NewConstructor("output", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlElement := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlElement.IsNull()
	if !exist {
		htmlElement = js.Global().Get("document").Call("createElement", "output")
	} 

	n := NewNativeElementWrapper(htmlElement)
	e.Native = n

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions
	withStringAttributeWatcher(e,"form")
	withStringAttributeWatcher(e,"name")
	withBoolAttributeWatcher(e,"disabled")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Output(id string, options ...string) OutputElement{
	return OutputElement{ui.BasicElement{LoadFromStorage(newOutput(id, options...))}}
}

// ImgElement
type ImgElement struct {
	ui.BasicElement
}

type imgModifier struct{}
var ImgModifier imgModifier

func(i imgModifier) Src(src string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("src",ui.String(src))
		return e
	}
}


func (i imgModifier) Alt(s string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("alt",ui.String(s))
		return e
	}
}

var newImage= Elements.NewConstructor("img", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlImg := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlImg.IsNull()
	if !exist {
		htmlImg = js.Global().Get("document").Call("createElement", "img")
	}

	n := NewNativeElementWrapper(htmlImg)
	e.Native = n
	SetAttribute(e, "id", id)

	withStringAttributeWatcher(e,"src")
	withStringAttributeWatcher(e,"alt")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Img(id string, options ...string) ImgElement {
	return ImgElement{ui.BasicElement{LoadFromStorage(newImage(id, options...))}}
}

var newAudio = Elements.NewConstructor("audio", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
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
	}

	n := NewNativeElementWrapper(htmlAudio)
	e.Native = n

	SetAttribute(e, "id", id)
	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var newVideo = Elements.NewConstructor("video", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
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
	} 

	SetAttribute(e, "id", id)

	n := NewNativeElementWrapper(htmlVideo)
	e.Native = n
	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)


type SourceElement struct{
	ui.BasicElement
}

type sourceModifier struct{}
var SourceModifier sourceModifier

func(s sourceModifier) Src(src string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("src",ui.String(src))
		return e
	}
}


func(s sourceModifier) Type(typ string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("type",ui.String(typ))
		return e
	}
}


var newSource = Elements.NewConstructor("source", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlSource := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlSource.IsNull()
	if !exist {
		htmlSource = js.Global().Get("document").Call("createElement", "source")
	} 

	n := NewNativeElementWrapper(htmlSource)
	e.Native = n

	SetAttribute(e, "id", id)

	withStringAttributeWatcher(e,"src")
	withStringAttributeWatcher(e,"type")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Source(id string, options ...string) SourceElement{
	return SourceElement{ui.BasicElement{LoadFromStorage(newSource(id,options...))}}
}

type UlElement struct {
	ui.BasicElement
}

func (l UlElement) FromValues(values ...ui.Value) UlElement {
	l.AsElement().SetDataSetUI("list", ui.NewList(values...))
	return l
}

func (l UlElement) Values() ui.List {
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
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlList := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlList.IsNull()
	if !exist {
		htmlList = js.Global().Get("document").Call("createElement", "ul")
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
				LiElement{ui.BasicElement{item}}.SetValue(v)
			} else {
				item = Li(id+"-item-"+strconv.Itoa(i)).SetValue(v).AsBasicElement().AsElement()
			}

			evt.Origin().AppendChild(ui.BasicElement{item})
		}
		return false
	})
	e.Watch("ui", "list", e, h)

	return e
}, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Ul(id string, options ...string) UlElement {
	return UlElement{ui.BasicElement{LoadFromStorage(newUl(id, options...))}}
}

type OlElement struct {
	ui.BasicElement
}

func (l OlElement) SetValue(lobjs ui.List) OlElement {
	l.AsElement().Set("data", "value", lobjs)
	return l
}

var newOl= Elements.NewConstructor("ol", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlList := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlList.IsNull()
	if !exist {
		htmlList = js.Global().Get("document").Call("createElement", "ol")
	} 

	n := NewNativeElementWrapper(htmlList)
	e.Native = n

	SetAttribute(e, "id", id)
	
	return e
}, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Ol(id string, typ string, numberingstart int, options ...string) OlElement {
	e:= newOl(id, options...)
	SetAttribute(e, "type", typ)
	SetAttribute(e, "start", strconv.Itoa(numberingstart))
	return OlElement{ui.BasicElement{LoadFromStorage(e)}}
}

type LiElement struct {
	ui.BasicElement
}

func (li LiElement) SetValue(v ui.Value) LiElement {
	li.AsElement().SetDataSetUI("value", v)
	return li
}

var newListItem= Elements.NewConstructor("li", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlListItem := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlListItem.IsNull()
	if !exist {
		htmlListItem = js.Global().Get("document").Call("createElement", "li")
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

func Li(id string, options ...string) LiElement {
	return LiElement{ui.BasicElement{LoadFromStorage(newListItem(id, options...))}}
}

// Table Elements

// TableElement
type TableElement struct {
	ui.BasicElement
}

// TheadELement
type TheadElement struct {
	ui.BasicElement
}

// TbodyElement
type TbodyElement struct {
	ui.BasicElement
}

// TrElement
type TrElement struct {
	ui.BasicElement
}


// TdELement
type TdElement struct {
	ui.BasicElement
}


// ThElement
type ThElement struct {
	ui.BasicElement
}

// ColElement
type ColElement struct {
	ui.BasicElement
}

func(c ColElement) SetSpan(n int) ColElement{
	c.AsElement().SetDataSetUI("span",ui.Number(n))
	return c
}

// ColGroupElement
type ColGroupElement struct {
	ui.BasicElement
}

func(c ColGroupElement) SetSpan(n int) ColGroupElement{
	c.AsElement().SetDataSetUI("span",ui.Number(n))
	return c
}

// TfootElement
type TfootElement struct {
	ui.BasicElement
}

var newThead= Elements.NewConstructor("thead", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlThead := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlThead.IsNull()
	if !exist {
		htmlThead = js.Global().Get("document").Call("createElement", "thead")
	} 

	n := NewNativeElementWrapper(htmlThead)
	e.Native = n

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Thead(id string, options ...string) TheadElement {
	return TheadElement{ui.BasicElement{LoadFromStorage(newThead(id, options...))}}
}


var newTr= Elements.NewConstructor("tr", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlTr := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlTr.IsNull()
	if !exist {
		htmlTr = js.Global().Get("document").Call("createElement", "tr")
	}

	n := NewNativeElementWrapper(htmlTr)
	e.Native = n

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Tr(id string, options ...string) TrElement {
	return TrElement{ui.BasicElement{LoadFromStorage(newTr(id, options...))}}
}

var newTd= Elements.NewConstructor("td", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlElement := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlElement.IsNull()
	if !exist {
		htmlElement = js.Global().Get("document").Call("createElement", "td")
	} 

	n := NewNativeElementWrapper(htmlElement)
	e.Native = n

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Td(id string, options ...string) TdElement {
	return TdElement{ui.BasicElement{LoadFromStorage(newTd(id, options...))}}
}

var newTh= Elements.NewConstructor("th", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlElement:= js.Global().Get("document").Call("getElementById", id)
	exist := !htmlElement.IsNull()
	if !exist {
		htmlElement = js.Global().Get("document").Call("createElement", "th")
	}

	n := NewNativeElementWrapper(htmlElement)
	e.Native = n

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Th(id string, options ...string) ThElement {
	return ThElement{ui.BasicElement{LoadFromStorage(newTh(id, options...))}}
}

var newTbody= Elements.NewConstructor("tbody", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlElement := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlElement.IsNull()
	if !exist {
		htmlElement = js.Global().Get("document").Call("createElement", "tbody")
	} 

	n := NewNativeElementWrapper(htmlElement)
	e.Native = n

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Tbody(id string, options ...string) TbodyElement {
	return TbodyElement{ui.BasicElement{LoadFromStorage(newTbody(id, options...))}}
}

var newTfoot= Elements.NewConstructor("tfoot", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlElement := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlElement.IsNull()
	if !exist {
		htmlElement = js.Global().Get("document").Call("createElement", "tfoot")
	} 

	n := NewNativeElementWrapper(htmlElement)
	e.Native = n

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Tfoot(id string, options ...string) TfootElement {
	return TfootElement{ui.BasicElement{LoadFromStorage(newTfoot(id, options...))}}
}

var newCol= Elements.NewConstructor("col", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlElement := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlElement.IsNull()
	if !exist {
		htmlElement = js.Global().Get("document").Call("createElement", "col")
	}

	n := NewNativeElementWrapper(htmlElement)
	e.Native = n

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions
	withNumberAttributeWatcher(e,"span")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Col(id string, options ...string) ColElement {
	return ColElement{ui.BasicElement{LoadFromStorage(newCol(id, options...))}}
}

var newColGroup= Elements.NewConstructor("colgroup", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlElement := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlElement.IsNull()
	if !exist {
		htmlElement = js.Global().Get("document").Call("createElement", "colgroup")
	}

	n := NewNativeElementWrapper(htmlElement)
	e.Native = n

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions
	withNumberAttributeWatcher(e,"span")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func ColGroup(id string, options ...string) ColGroupElement {
	return ColGroupElement{ui.BasicElement{LoadFromStorage(newColGroup(id, options...))}}
}

var newTable= Elements.NewConstructor("table", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlTable := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlTable.IsNull()
	if !exist {
		htmlTable = js.Global().Get("document").Call("createElement", "table")
	} 

	n := NewNativeElementWrapper(htmlTable)
	e.Native = n

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Table(id string, options ...string) TableElement {
	return TableElement{ui.BasicElement{LoadFromStorage(newTable(id, options...))}}
}


type CanvasElement struct{
	ui.BasicElement
}

type canvasModifier struct{}
var CanvasModifier = canvasModifier{}

func(c canvasModifier) Height(h int)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("height",ui.Number(h))
		return e
	}
}


func(c canvasModifier) Width(w int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("width",ui.Number(w))
		return e
	}
}

var newCanvas = Elements.NewConstructor("canvas",func(id string)*ui.Element{
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlCanvas := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlCanvas.IsNull()
	if !exist {
		htmlCanvas = js.Global().Get("document").Call("createElement", "canvas")
	} 

	n := NewNativeElementWrapper(htmlCanvas)
	e.Native = n

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions
	withNumberAttributeWatcher(e,"height")
	withNumberAttributeWatcher(e,"width")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Canvas(id string, options ...string) CanvasElement {
	return CanvasElement{ui.BasicElement{LoadFromStorage(newCanvas(id, options...))}}
}

type SvgElement struct{
	ui.BasicElement
}

type svgModifier struct{}
var SvgModifer svgModifier

func(s svgModifier) Height(h int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("height",ui.Number(h))
		return e
	}
}

func(s svgModifier) Width(w int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("width",ui.Number(w))
		return e
	}
}

func(s svgModifier) Viewbox(attr string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("viewbox",ui.String(attr))
		return e
	}
}


func(s svgModifier) X(x string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("x",ui.String(x))
		return e
	}
}


func(s svgModifier) Y(y string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("y",ui.String(y))
		return e
	}
}

var newSvg = Elements.NewConstructor("svg",func(id string)*ui.Element{
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlSvg := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlSvg.IsNull()
	if !exist {
		htmlSvg = js.Global().Get("document").Call("createElement", "svg")
	} 

	n := NewNativeElementWrapper(htmlSvg)
	e.Native = n

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions
	withNumberAttributeWatcher(e,"height")
	withNumberAttributeWatcher(e,"width")
	withStringAttributeWatcher(e,"viewbox")
	withStringAttributeWatcher(e,"x")
	withStringAttributeWatcher(e,"y")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Svg(id string, options ...string) SvgElement {
	return SvgElement{ui.BasicElement{LoadFromStorage(newSvg(id, options...))}}
}

type SummaryElement struct{
	ui.BasicElement
}

func (s SummaryElement) SetText(str string) SummaryElement {
	s.AsElement().SetDataSetUI("text", ui.String(str))
	return s
}

var newSummary = Elements.NewConstructor("summary", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlSummary := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlSummary.IsNull()
	if !exist {
		htmlSummary = js.Global().Get("document").Call("createElement", "summary")
	}

	n := NewNativeElementWrapper(htmlSummary)
	e.Native = n

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions

	e.Watch("ui", "text", e, textContentHandler)

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Summary(id string, options ...string) SummaryElement {
	return SummaryElement{ui.BasicElement{LoadFromStorage(newSummary(id, options...))}}
}

type DetailsElement struct{
	ui.BasicElement
}

func (d DetailsElement) SetText(str string) DetailsElement {
	d.AsElement().SetDataSetUI("text", ui.String(str))
	return d
}

func(d DetailsElement) Open() DetailsElement{
	d.AsElement().SetDataSetUI("open",ui.Bool(true))
	return d
}

func(d DetailsElement) Close() DetailsElement{
	d.AsElement().SetDataSetUI("open",nil)
	return d
}

func(d DetailsElement) IsOpened() bool{
	o,ok:= d.AsElement().GetData("open")
	if !ok{
		return false
	}
	_,ok= o.(ui.Bool)
	if !ok{
		return false
	}
	return true
}

var newDetails = Elements.NewConstructor("details", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlDetails := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlDetails.IsNull()
	if !exist {
		htmlDetails = js.Global().Get("document").Call("createElement", "details")
	}

	n := NewNativeElementWrapper(htmlDetails)
	e.Native = n

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions

	e.Watch("ui", "text", e, textContentHandler)
	withBoolAttributeWatcher(e,"open")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Details(id string, options ...string) DetailsElement {
	return DetailsElement{ui.BasicElement{LoadFromStorage(newDetails(id, options...))}}
}

// Dialog
type DialogElement struct{
	ui.BasicElement
}


func(d DialogElement) Open() DialogElement{
	d.AsElement().SetDataSetUI("open",ui.Bool(true))
	return d
}

func(d DialogElement) Close() DialogElement{
	d.AsElement().SetDataSetUI("open",nil)
	return d
}

func(d DialogElement) IsOpened() bool{
	o,ok:= d.AsElement().GetData("open")
	if !ok{
		return false
	}
	_,ok= o.(ui.Bool)
	if !ok{
		return false
	}
	return true
}

var newDialog = Elements.NewConstructor("dialog", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlDialog := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlDialog.IsNull()
	if !exist {
		htmlDialog = js.Global().Get("document").Call("createElement", "dialog")
	}

	n := NewNativeElementWrapper(htmlDialog)
	e.Native = n

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions

	withBoolAttributeWatcher(e,"open")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Dialog(id string, options ...string) DialogElement {
	return DialogElement{ui.BasicElement{LoadFromStorage(newDialog(id, options...))}}
}

// CodeElement is typically used to indicate that the text it contains is computer code and may therefore be 
// formatted differently.
// To represent multiple lines of code, wrap the <code> element within a <pre> element. 
// The <code> element by itself only represents a single phrase of code or line of code.
type CodeElement struct {
	ui.BasicElement
}

func (c CodeElement) SetText(str string) CodeElement {
	c.AsElement().SetDataSetUI("text", ui.String(str))
	return c
}

var newCode= Elements.NewConstructor("code", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlCode := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlCode.IsNull()
	if !exist {
		htmlCode = js.Global().Get("document").Call("createElement", "code")
	}

	n := NewNativeElementWrapper(htmlCode)
	e.Native = n

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions

	e.Watch("ui", "text", e, textContentHandler)

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Code(id string, options ...string) CodeElement {
	return CodeElement{ui.BasicElement{LoadFromStorage(newCode(id, options...))}}
}

// Embed
type EmbedElement struct{
	ui.BasicElement
}

func(e EmbedElement) SetHeight(h int) EmbedElement{
	e.AsElement().SetDataSetUI("height",ui.Number(h))
	return e
}

func(e EmbedElement) SetWidth(w int) EmbedElement{
	e.AsElement().SetDataSetUI("width",ui.Number(w))
	return e
}

func(e EmbedElement) SetType(typ string) EmbedElement{
	e.AsElement().SetDataSetUI("type", ui.String(typ))
	return e
}

func(e EmbedElement) SetSrc(src string) EmbedElement{
	e.AsElement().SetDataSetUI("src", ui.String(src))
	return e
}


var newEmbed = Elements.NewConstructor("embed",func(id string)*ui.Element{
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlEmbed := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlEmbed.IsNull()
	if !exist {
		htmlEmbed = js.Global().Get("document").Call("createElement", "embed")
	} 

	n := NewNativeElementWrapper(htmlEmbed)
	e.Native = n

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions
	withNumberAttributeWatcher(e,"height")
	withNumberAttributeWatcher(e,"width")
	withStringAttributeWatcher(e,"type")
	withStringAttributeWatcher(e,"src")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Embed(id string, options ...string) EmbedElement {
	return EmbedElement{ui.BasicElement{LoadFromStorage(newEmbed(id, options...))}}
}

// Object
type ObjectElement struct{
	ui.BasicElement
}

type objectModifier struct{}
var ObjectModifier = objectModifier{}

func(o objectModifier) Height(h int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("height",ui.Number(h))
		return e
	}
}

func(o objectModifier) Width(w int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("width",ui.Number(w))
		return e
	}
}


func(o objectModifier) Type(typ string)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("type", ui.String(typ))
		return e
	}
}

// Data sets the path to the resource.
func(o objectModifier) Data(u url.URL)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("data", ui.String(u.String()))
		return e
	}
}
func (o objectModifier) Form(form *ui.Element) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			d:= GetDocument()
			
			evt.Origin().Watch("event","navigationend",d,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
				if form.Mounted(){
					e.AsElement().SetDataSetUI("form", ui.String(form.ID))
				}
				return false
			}).RunOnce())
			return false
		}).RunOnce())
		return e
	}
}


var newObject = Elements.NewConstructor("object",func(id string)*ui.Element{
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlElement := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlElement.IsNull()
	if !exist {
		htmlElement = js.Global().Get("document").Call("createElement", "object")
	} 

	n := NewNativeElementWrapper(htmlElement)
	e.Native = n

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions
	withNumberAttributeWatcher(e,"height")
	withNumberAttributeWatcher(e,"width")
	withStringAttributeWatcher(e,"type")
	withStringAttributeWatcher(e,"data")
	withStringAttributeWatcher(e,"form")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Object(id string, options ...string) ObjectElement {
	return ObjectElement{ui.BasicElement{LoadFromStorage(newObject(id, options...))}}
}

// Datalist
type DatalistElement struct{
	ui.BasicElement
}

var newDatalist = Elements.NewConstructor("datalist", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlElement := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlElement.IsNull()
	if !exist {
		htmlElement = js.Global().Get("document").Call("createElement", "datalist")
	} 

	n := NewNativeElementWrapper(htmlElement)
	e.Native = n

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Datalist(id string, options ...string) DatalistElement{
	return DatalistElement{ui.BasicElement{LoadFromStorage(newDatalist(id, options...))}}
}

// OptionElement
type OptionElement struct{
	ui.BasicElement
}

type optionModifier struct{}
var OptionModifer optionModifier

func(o optionModifier) Label(l string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("label",ui.String(l))
		return e
	}
}

func(o optionModifier) Value(value string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("value",ui.String(value))
		return e
	}
}

func(o optionModifier) Disabled(b bool) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("disabled",ui.Bool(b))
		return e
	}
}

func(o optionModifier) Selected() func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("selected",ui.Bool(true))
		return e
	}
}

func(o OptionElement) SetValue(opt string) OptionElement{
	o.AsElement().SetDataSetUI("value", ui.String(opt))
	return o
}

var newOption = Elements.NewConstructor("option", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlElement := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlElement.IsNull()
	if !exist {
		htmlElement = js.Global().Get("document").Call("createElement", "option")
	} 

	n := NewNativeElementWrapper(htmlElement)
	e.Native = n

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions
	withStringAttributeWatcher(e,"value")
	withStringAttributeWatcher(e,"label")
	withBoolAttributeWatcher(e,"disabled")
	withBoolAttributeWatcher(e,"selected")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Option(id string, options ...string) OptionElement{
	return OptionElement{ui.BasicElement{LoadFromStorage(newOption(id, options...))}}
}

// OptgroupElement
type OptgroupElement struct{
	ui.BasicElement
}

type optgroupModifier struct{}
var OptgroupModifer optionModifier

func(o optgroupModifier) Label(l string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("label",ui.String(l))
		return e
	}
}


func(o optgroupModifier) Disabled(b bool) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("disabled",ui.Bool(b))
		return e
	}
}


func(o OptgroupElement) SetLabel(opt string) OptgroupElement{
	o.AsElement().SetDataSetUI("label", ui.String(opt))
	return o
}

func(o OptgroupElement) SetDisabled(b bool) OptgroupElement{
	o.AsElement().SetDataSetUI("disabled",ui.Bool(b))
	return o
}

var newOptgroup = Elements.NewConstructor("optgroup", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlElement := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlElement.IsNull()
	if !exist {
		htmlElement = js.Global().Get("document").Call("createElement", "optgroup")
	} 

	n := NewNativeElementWrapper(htmlElement)
	e.Native = n

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions
	withStringAttributeWatcher(e,"label")
	withBoolAttributeWatcher(e,"disabled")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Optgroup(id string, options ...string) OptgroupElement{
	return OptgroupElement{ui.BasicElement{LoadFromStorage(newOptgroup(id, options...))}}
}

// FieldsetElement
type FieldsetElement struct{
	ui.BasicElement
}

type fieldsetModifier struct{}
var FieldsetModifer fieldsetModifier

func(m fieldsetModifier) Form(form *ui.Element) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			d:= GetDocument()
			
			evt.Origin().Watch("event","navigationend",d,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
				if form.Mounted(){
					e.SetDataSetUI("form", ui.String(form.ID))
				}
				return false
			}).RunOnce())
			return false
		}).RunOnce())
		return e
	}
}

func(m fieldsetModifier) Name(name string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("name",ui.String(name))
		return e
	}
}

func(m fieldsetModifier) Disabled(b bool) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("disabled",ui.Bool(b))
		return e
	}
}


var newFieldset = Elements.NewConstructor("fieldset", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlElement := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlElement.IsNull()
	if !exist {
		htmlElement = js.Global().Get("document").Call("createElement", "fieldset")
	} 

	n := NewNativeElementWrapper(htmlElement)
	e.Native = n

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions
	withStringAttributeWatcher(e,"form")
	withStringAttributeWatcher(e,"name")
	withBoolAttributeWatcher(e,"disabled")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Fieldset(id string, options ...string) FieldsetElement{
	return FieldsetElement{ui.BasicElement{LoadFromStorage(newFieldset(id, options...))}}
}

// LegendElement
type LegendElement struct{
	ui.BasicElement
}

func(l LegendElement) SetText(s string) LegendElement{
	l.AsElement().SetDataSetUI("text",ui.String(s))
	return l
}

var newLegend = Elements.NewConstructor("legend", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlElement := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlElement.IsNull()
	if !exist {
		htmlElement = js.Global().Get("document").Call("createElement", "legend")
	} 

	n := NewNativeElementWrapper(htmlElement)
	e.Native = n

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions
	e.Watch("ui", "text", e, textContentHandler)

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Legend(id string, options ...string) LegendElement{
	return LegendElement{ui.BasicElement{LoadFromStorage(newLegend(id, options...))}}
}

// ProgressElement
type ProgressElement struct{
	ui.BasicElement
}

func(p ProgressElement) SetMax(m float64) ProgressElement{
	if m>0{
		p.AsElement().SetDataSetUI("max", ui.Number(m))
	}
	
	return p
}

func(p ProgressElement) SetValue(v float64) ProgressElement{
	if v>0{
		p.AsElement().SetDataSetUI("value", ui.Number(v))
	}
	
	return p
}

var newProgress = Elements.NewConstructor("progress", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlElement := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlElement.IsNull()
	if !exist {
		htmlElement = js.Global().Get("document").Call("createElement", "progress")
	} 

	n := NewNativeElementWrapper(htmlElement)
	e.Native = n

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions
	withNumberAttributeWatcher(e,"max")
	withNumberAttributeWatcher(e,"value")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Progress(id string, options ...string) ProgressElement{
	return ProgressElement{ui.BasicElement{LoadFromStorage(newProgress(id, options...))}}
}

// SelectElement
type SelectElement struct{
	ui.BasicElement
}

type selectModifier struct{}
var SelectModifier selectModifier

func(m selectModifier) Autocomplete(b bool) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("autocomplete",ui.Bool(b))
		return e
	}
}

func(m selectModifier) Size(s int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("size",ui.Number(s))
		return e
	}
}

func(m selectModifier) Disabled(b bool) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("disabled",ui.Bool(b))
		return e
	}
}

func (m selectModifier) Form(form *ui.Element) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			d:= GetDocument()
			
			evt.Origin().Watch("event","navigationend",d,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
				if form.Mounted(){
					e.AsElement().SetDataSetUI("form", ui.String(form.ID))
				}
				return false
			}).RunOnce())
			return false
		}).RunOnce())
		return e
	}
}

func(m selectModifier) Required(b bool) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("required",ui.Bool(b))
		return e
	}
}

func(m selectModifier) Name(name string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("name",ui.String(name))
		return e
	}
}


var newSelect = Elements.NewConstructor("select", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlElement := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlElement.IsNull()
	if !exist {
		htmlElement = js.Global().Get("document").Call("createElement", "select")
	} 

	n := NewNativeElementWrapper(htmlElement)
	e.Native = n

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions
	withStringAttributeWatcher(e,"form")
	withStringAttributeWatcher(e,"name")
	withBoolAttributeWatcher(e,"disabled")
	withBoolAttributeWatcher(e,"required")
	withBoolAttributeWatcher(e,"multiple")
	withNumberAttributeWatcher(e,"size")
	withStringAttributeWatcher(e,"autocomplete")


	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Select(id string, options ...string) SelectElement{
	return SelectElement{ui.BasicElement{LoadFromStorage(newSelect(id, options...))}}
}


// FormElement
type FormElement struct{
	ui.BasicElement
}

type formModifier struct{}
var FormModifer = formModifier{}

func(f formModifier) Name(name string) func(*ui.Element) *ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("name",ui.String(name))
		return e
	}
}

func(f formModifier) Method(methodname string) func(*ui.Element) *ui.Element{
	m:=  "GET"
	if  strings.EqualFold(methodname,"POST"){
		m = "POST"
	}
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("method",ui.String(m))
		return e
	}
}

func(f formModifier) Target(target string) func(*ui.Element) *ui.Element{
	m:=  "_self"
	if strings.EqualFold(target,"_blank"){
		m = "_blank"
	}
	if strings.EqualFold(target,"_parent"){
		m = "_parent"
	}
	if strings.EqualFold(target,"_top"){
		m = "_top"
	}

	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("target",ui.String(m))
		return e
	}
}

func(f formModifier) Action(u url.URL) func(*ui.Element) *ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("action",ui.String(u.String()))
		return e
	}
}

func(f formModifier) Autocomplete() func(*ui.Element) *ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("autocomplete",ui.Bool(true))
		return e
	}
}

func(f formModifier) NoValidate() func(*ui.Element) *ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("novalidate",ui.Bool(true))
		return e
	}
}

func(f formModifier) EncType(enctype string) func(*ui.Element) *ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("enctype",ui.String(enctype))
		return e
	}
}

func(f formModifier) Charset(charset string) func(*ui.Element) *ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("accept-charset",ui.String(charset))
		return e
	}
}

var newForm= Elements.NewConstructor("form", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	htmlElement := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlElement.IsNull()
	if !exist {
		htmlElement = js.Global().Get("document").Call("createElement", "form")
	} 

	n := NewNativeElementWrapper(htmlElement)
	e.Native = n

	e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		_,ok:= e.Get("ui","action")
		if !ok{
			evt.Origin().SetDataSetUI("action",ui.String(evt.Origin().Route()))
		}
		return false
	}))

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions
	withStringAttributeWatcher(e,"accept-charset")
	withBoolAttributeWatcher(e,"autocomplete")
	withStringAttributeWatcher(e,"name")
	withStringAttributeWatcher(e,"action")
	withStringAttributeWatcher(e,"enctype")
	withStringAttributeWatcher(e,"method")
	withBoolAttributeWatcher(e,"novalidate")
	withStringAttributeWatcher(e,"target")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func Form(id string, options ...string) FormElement {
	return FormElement{ui.BasicElement{LoadFromStorage(newForm(id, options...))}}
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

// watches ("ui",attr) for a ui.String value.
func withStringAttributeWatcher(e *ui.Element,attr string){
	e.Watch("ui",attr,e,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		SetAttribute(evt.Origin(),attr,string(evt.NewValue().(ui.String)))
		return false
	}))
}


// watches ("ui",attr) for a ui.Number value.
func withNumberAttributeWatcher(e *ui.Element,attr string){
	e.Watch("ui",attr,e,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		SetAttribute(evt.Origin(),attr,strconv.Itoa(int(evt.NewValue().(ui.Number))))
		return false
	}))
}

// watches ("ui",attr) for a ui.Bool value.
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
		e.SetDataSetUI(name,ui.String(value))
		return e
	}
}

type set map[string]struct{}

func newset(val ...string) set{
	if val != nil{
		s:=  set(make(map[string]struct{},len(val)))
		for _,v:= range val{
			s[v]= struct{}{}
		}
		return s
	}
	return set(make(map[string]struct{}, 32))
}

func(s set) Contains(str string) bool{
	_,ok:= s[str]
	return ok
}

func(s set) Add(str string) set{
	s[str]= struct{}{}
	return s
}