//go:build !server 

// Package doc defines the default set of Element constructors, native interfaces,
// events and event handlers, and animation properties used to build js-based UIs.


package doc

import (
	"context"
	//"encoding/json"
	"github.com/goccy/go-json"
	//"errors"
	"log"
	"strings"
	"syscall/js"
	"time"
	"github.com/atdiar/particleui"
)

var (
	// DOCTYPE holds the document doctype.
	DOCTYPE = "html/js"
	// Elements stores wasm-generated HTML ui.Element constructors.
	Elements                      = ui.NewElementStore("default", DOCTYPE).
		AddPersistenceMode("sessionstorage", loadfromsession, sessionstorefn, clearfromsession).
		AddPersistenceMode("localstorage", loadfromlocalstorage, localstoragefn, clearfromlocalstorage).
		ApplyGlobalOption(cleanStorageOnDelete).
		AddConstructorOptions("observable",AllowSessionStoragePersistence,AllowAppLocalStoragePersistence)
)


// NewBuilder registers a new document building function.
func NewBuilder(f func()Document)(ListenAndServe func(context.Context)){
	return  func(ctx context.Context){
		f().ListenAndServe(ctx)
	}
}

var dEBUGJS = func(v js.Value, isJsonString ...bool){
	if isJsonString!=nil{
		o:= js.Global().Get("JSON").Call("parse",v)
		js.Global().Get("console").Call("log",o)
		return
	}
	js.Global().Get("console").Call("log",v)
}

// abstractjs 
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
// abstractjs
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

		if !propertyExists || !indexed {
			props := make([]interface{}, 0, 4)
			c, ok := element.Properties.Categories[category]
			if !ok {
				props = append(props, propname)
				v := js.ValueOf(props)
				store.Set(strings.Join([]string{element.ID, category}, "/"), v)
			} else {
				for k := range c.Local {
					props = append(props, k)
				}

				props = append(props, propname)
				// log.Print("all props stored...", props) // DEBUG
				v := js.ValueOf(props)
				store.Set(strings.Join([]string{element.ID, category}, "/"), v) 
			}
		}
		item := value.RawValue()
		v := stringify(item)
		store.Set(strings.Join([]string{element.ID, category, propname}, "/"),js.ValueOf(v))
		element.TriggerEvent("storesynced",ui.Bool(false))
		return
	}
}



var sessionstorefn = storer("sessionStorage")
var localstoragefn = storer("localStorage")

func loader(s string) func(e *ui.Element) error { // abstractjs
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
				propname := property
				jsonvalue, ok := store.Get(strings.Join([]string{e.ID, category, propname}, "/"))
				if ok {					
					var rawvaluemapstring string
					err = json.Unmarshal([]byte(jsonvalue.String()), &rawvaluemapstring)
					if err != nil {
						return err
					}
					
					rawvalue := make(map[string]interface{})
					err = json.Unmarshal([]byte(rawvaluemapstring), &rawvalue)
					if err != nil {
						return err
					}
					
					ui.LoadProperty(e, category, propname, ui.Object(rawvalue).MarkedRaw().Value())
					//log.Print("LOADED PROPMAP: ", e.Properties, category, propname, rawvalue.Value()) // DEBUG
				}
			}
		}
		e.OnRegistered(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			ui.Rerender(e)
			return false
		}).RunOnce())
		
		return nil
	}
}

var loadfromsession = loader("sessionStorage")
var loadfromlocalstorage = loader("localStorage")

func clearer(s string) func(element *ui.Element){ // abstractjs
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

				store.Delete(strings.Join([]string{id, category, property}, "/")) 
			}
			store.Delete(strings.Join([]string{id, category}, "/")) 
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
			j,ok:= JSValue(e)
			if !ok{
				return false
			}
			if j.Truthy(){
				j.Call("remove") // abstractjs
			}
		}
		
		return false
	}))
	return e
})

/*var NoFetchWhileHydrating = ui.NewConstructorOption("nofetchwhilehydrating", func(e *ui.Element)*ui.Element{
	e.On
	return e
})
*/

// isPersisted checks whether an element exist in storage already
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

var titleElementChangeHandler = ui.NewMutationHandler(func(evt ui.MutationEvent) bool { 
	SetTextContent(evt.Origin(),evt.NewValue().(ui.String).String())
	return false
})

func SetTextContent(e *ui.Element, text string) {
	if e.Native != nil {
		nat, ok := e.Native.(NativeElement)
		if !ok {
			panic("trying to set text content on a non-DOM element")
		}
		nat.Value.Set("textContent", string(text))
	}
}


var windowTitleHandler = ui.NewMutationHandler(func(evt ui.MutationEvent) bool { // abstractjs
	target := evt.Origin()
	newtitle, ok := evt.NewValue().(ui.String)
	if !ok {
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


// TODO
// it also recovers the mutationtrace
func nativeDocumentAlreadyRendered(root *ui.Element, document js.Value, SSRStateNodeID string) bool{
	if !document.Truthy(){
		return false
	}
	//  get native document status by looking for the ssr hint encoded in the page (data attribute)
	// the data attribute should be removed once the document state is replayed.
	statenode:= document.Call("getElementById",SSRStateNodeID )
	if !statenode.Truthy(){
		// TODO: check if the document is already rendered, at least partially, still.
		return false
	}
	state:= statenode.Get("textContent").String()
	v,err:= DeserializeStateHistory(state)
	if err != nil{
		panic(err)
	}
	root.Set("internals","mutationtrace",v)	
	root.WatchEvent("mutationreplayed",root,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		statenode.Call("remove")
		return false
	}))

	return true
}

func ConnectNative(e *ui.Element, tag string) (ui.NativeElement,bool){
	id:= e.ID
	if e.IsRoot(){
		//
		if  nativeDocumentAlreadyRendered(e,js.Global().Get("document"), id + SSRStateSuffix){
			e.ElementStore.EnableMutationReplay()
			e.WatchEvent("mutationreplayed",e,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{		
				evt.Origin().ElementStore.MutationReplay = false
				return false
			}))
		}
	}
	
	if e.ElementStore.MutationReplay{
		e.WatchEvent("mutationreplayed",e,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{

			if tag == "window"{
				wd := js.Global().Get("document").Get("defaultView")
				if !wd.Truthy() {
					panic("unable to access windows")
				}
				evt.Origin().Native = NewNativeElementWrapper(wd)
			}
		
			if tag == "html"{
				root:= js.Global().Get("document").Call("getElementById",id)
				if !root.Truthy(){
					root = js.Global().Get("document").Get("documentElement")
					if !root.Truthy() {
						panic("failed to instantiate root element for the document")
					}
				}
				evt.Origin().Native = NewNativeElementWrapper(root)
			}
		
			if tag == "body"{
				element:= js.Global().Get("document").Call("getElementById",id)
				if !element.Truthy(){
					element= js.Global().Get("document").Get(tag)
					if !element.Truthy(){
						element= js.Global().Get("document").Call("createElement",tag)
					}
				}
				evt.Origin().Native = NewNativeElementWrapper(element)
			}
		
			if tag == "head"{
				element:= js.Global().Get("document").Call("getElementById",id)
				if !element.Truthy(){
					element= js.Global().Get("document").Get(tag)
					if !element.Truthy(){
						element= js.Global().Get("document").Call("createElement",tag)
					}
				}
				evt.Origin().Native = NewNativeElementWrapper(element)
			}
		
			element:= js.Global().Get("document").Call("getElementById",id)
			if !element.Truthy(){
				element= js.Global().Get("document").Call("createElement",tag)
			}
			evt.Origin().Native = NewNativeElementWrapper(element) 
				
			return false

		}).RunOnce())
		return nil,false
	}
	if tag == "window"{
		wd := js.Global().Get("document").Get("defaultView")
		if !wd.Truthy() {
			panic("unable to access windows")
		}
		return  NewNativeElementWrapper(wd), true
	}

	if tag == "html"{
		root:= js.Global().Get("document").Call("getElementById",id)
		if !root.Truthy(){
			root = js.Global().Get("document").Get("documentElement")
			if !root.Truthy() {
				panic("failed to instantiate root element for the document")
			}
			return NewNativeElementWrapper(root), false
		}
		return NewNativeElementWrapper(root), true
	}

	if tag == "body"{
		element:= js.Global().Get("document").Call("getElementById",id)
		if !element.Truthy(){
			element= js.Global().Get("document").Get(tag)
			if !element.Truthy(){
				element= js.Global().Get("document").Call("createElement",tag)
			}
			return NewNativeElementWrapper(element), false
		}
		return NewNativeElementWrapper(element), true
	}

	if tag == "head"{
		element:= js.Global().Get("document").Call("getElementById",id)
		if !element.Truthy(){
			element= js.Global().Get("document").Get(tag)
			if !element.Truthy(){
				element= js.Global().Get("document").Call("createElement",tag)
			}
			return NewNativeElementWrapper(element), false
		}
		return NewNativeElementWrapper(element), true
	}

	element:= js.Global().Get("document").Call("getElementById",id)
	if !element.Truthy(){
		element= js.Global().Get("document").Call("createElement",tag)
		return NewNativeElementWrapper(element), false
	}
	return NewNativeElementWrapper(element), true
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
func JSValue(el ui.AnyElement) (js.Value,bool) { // TODO  unexport
	e:= el.AsElement()
	n, ok := e.Native.(NativeElement)
	if !ok {
		return js.Value{},ok
	}
	return n.Value, true
}

// SetInnerHTML sets the innerHTML property of HTML elements.
// Please note that it is unsafe to sets client submittd HTML inputs.
func SetInnerHTML(e *ui.Element, html string) *ui.Element {
	jsv,ok := JSValue(e)
	if !ok{
		return e
	}
	jsv.Set("innerHTML", html)
	return e
} // abstractjs

// LoadFromStorage will load an element properties.
// If the corresponding native DOM Element is marked for hydration, by the presence of a data-hydrate
// atribute, the props are loaded from this attribute instead.
// abstractjs
func LoadFromStorage(e *ui.Element) *ui.Element {
	//n:= JSValue(e) TODO delete this line

	if e == nil {
		panic("loading a nil element")
	}
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
// abstractjs
var AllowScrollRestoration = ui.NewConstructorOption("scrollrestoration", func(e *ui.Element) *ui.Element {
	if e.ID == GetDocument(e).AsElement().ID{
		if js.Global().Get("history").Get("scrollRestoration").Truthy() {
			js.Global().Get("history").Set("scrollRestoration", "manual")
		}
		return rootScrollRestorationSupport(e)
	}

	e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		e.WatchEvent("document-loaded", e.Root(), ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			router := ui.GetRouter(evt.Origin())

			ejs,ok := JSValue(e)
			if !ok{
				return false
			}
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
							e.TriggerEvent("shouldscroll", ui.Bool(false)) //always scroll root
						}
					}
					return false
				})
				e.WatchEvent("navigation-end", e.Root(), h)
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
			e.TriggerEvent("shouldscroll", ui.Bool(true))
		}
		return false
	}))
	return e
})




var historyMutationHandler = ui.NewMutationHandler(func(evt ui.MutationEvent)bool{ // abstractjs
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
})

var navinitHandler =  ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
	e:= evt.Origin()

	// Retrieve history and deserialize URL into corresponding App state.
	hstate := js.Global().Get("history").Get("state")
	
	if hstate.Truthy() {
		hstateobj := ui.NewObject().MarkedRaw()
		err := json.Unmarshal([]byte(hstate.String()), &hstateobj)
		if err == nil {
			evt.Origin().SyncUISetData("history", hstateobj.Value())
		}
	}

	route := js.Global().Get("location").Get("pathname").String()
	e.TriggerEvent("navigation-routechangerequest", ui.String(route))
	return false
})

var rootScrollRestorationSupport = func(root *ui.Element)*ui.Element { // abstractjs
	e:= root
	n:= e.Native.(NativeElement).Value
	router := ui.GetRouter(root)

	ejs := js.Global().Get("document").Get("scrollingElement")

	e.SetDataSetUI("scrollrestore", ui.Bool(true))

	GetDocument(e).Window().AsElement().AddEventListener("scroll", ui.NewEventHandler(func(evt ui.Event) bool {
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
			elid:=v.(ui.String).String()
			el:= GetDocument(evt.Origin()).GetElementById(elid)

			if el != nil && el.Mounted(){
				Focus(el,false)
				if newpageaccess{
					if !partiallyVisible(el){
						DEBUG("focused element not in view...scrolling")
						n.Call("scrollIntoView")
					}
				}
					
			}
		} else{
			elid:=v.(ui.String).String()
			el:= GetDocument(evt.Origin()).GetElementById(elid)

			if el != nil && el.Mounted(){
				Focus(el,false)
				if newpageaccess{
					if !partiallyVisible(el){
						DEBUG("focused element not in view...scrolling")
						n.Call("scrollIntoView")
					}
				}
					
			}
		}
		
		return false
	})
	e.WatchEvent("navigation-end", e, h)

	return e
}

// Focus triggers the focus event asynchronously on the JS side.
func Focus(e ui.AnyElement, scrollintoview bool){ // abstractjs
	if !e.AsElement().Mounted(){
		return
	}

	ui.DoAsync(e.AsElement().Root(), func() {
		n,ok:= JSValue(e.AsElement())
		if !ok{
			return
		}
		focus(n)
		if scrollintoview{
			if !partiallyVisible(e.AsElement()){
				n.Call("scrollIntoView")
			}
		}
	})
}

func focus(e js.Value){ // abstractjs
	e.Call("focus",map[string]interface{}{"preventScroll": true})
}


// abstractjs
func IsInViewPort(e *ui.Element) bool{
	n,ok:= JSValue(e)
	if !ok{
		return false
	}
	bounding:= n.Call("getBoundingClientRect")
	top:= int(bounding.Get("top").Float())
	bottom:= int(bounding.Get("bottom").Float())
	left:= int(bounding.Get("left").Float())
	right:= int(bounding.Get("right").Float())

	w,ok:= JSValue(GetDocument(e).Window().AsElement())
	if !ok{
		panic("seems that the window is not connected to its native DOM element")
	}
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

// abstractjs
func partiallyVisible(e *ui.Element) bool{
	n,ok:= JSValue(e)
	if !ok{
		return false
	}
	bounding:= n.Call("getBoundingClientRect")
	top:= int(bounding.Get("top").Float())
	//bottom:= int(bounding.Get("bottom").Float())
	left:= int(bounding.Get("left").Float())
	//right:= int(bounding.Get("right").Float())

	w,ok:= JSValue(GetDocument(e).Window().AsElement())
	if !ok{
		panic("seems that the window is not connected to its native DOM element")
	}
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

// abstractjs
func TrapFocus(e *ui.Element) *ui.Element{ // TODO what to do if no eleemnt is focusable? (edge-case)
	e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		m,ok:= JSValue(evt.Origin())
		if !ok{
			return false
		}
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



var paragraphTextHandler = ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
	j,ok:= JSValue(evt.Origin())
	if !ok{
		return false
	}
	j.Set("innerText", string(evt.NewValue().(ui.String)))
	return false
})


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
			e.SyncUISetData("text", ui.String(s))
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




func (i InputElement) Blur() {
	native, ok := i.AsElement().Native.(NativeElement)
	if !ok {
		panic("native element should be of doc.NativeELement type")
	}
	ui.DoAsync(i.Root(), func(){
		native.Value.Call("blur")
	})
}

func (i InputElement) Focus() {
	native, ok := i.AsElement().Native.(NativeElement)
	if !ok {
		panic("native element should be of doc.NativeELement type")
	}
	ui.DoAsync(i.Root(), func(){
		native.Value.Call("focus")
	})
}

func (i InputElement) Clear() {
	native, ok := i.AsElement().Native.(NativeElement)
	if !ok {
		panic("native element should be of doc.NativeELement type")
	}
	ui.DoAsync(i.Root(), func(){
		native.Value.Set("value", "")
	})
}

func newTimeRanges(v js.Value) jsTimeRanges{
	var j = ui.NewObject()

	var length int
	l:= v.Get("length")
	
	if l.Truthy(){
		length = int(l.Float())
	}
	j.Set("length",ui.Number(length))

	starts:= ui.NewList()
	ends := ui.NewList()
	for i:= 0; i<length;i++{
		st:= ui.Number(v.Call("start",i).Float())
		en:= ui.Number(v.Call("end",i).Float())
		starts[i]=st
		ends[i]=en
	}
	j.Set("start",starts)
	j.Set("end",ends)
	return jsTimeRanges(j)
}


func(a AudioElement) Buffered() jsTimeRanges{
	j,ok:= JSValue(a.AsElement())
	if !ok{
		panic("element is not connected to Native dom node.")
	}
	
	b:= j.Get("buiffered")
	return newTimeRanges(b)
}

func(a AudioElement)CurrentTime() time.Duration{
	j,ok:= JSValue(a.AsElement())
	if !ok{
		panic("element is not connected to Native dom node.")
	}
	return time.Duration(j.Get("currentTime").Float())* time.Second
}

func(a AudioElement)Duration() time.Duration{
	j,ok:= JSValue(a.AsElement())
	if !ok{
		panic("element is not connected to Native dom node.")
	}
	return  time.Duration(j.Get("duration").Float())*time.Second
}

func(a AudioElement)PlayBackRate() float64{
	j,ok:= JSValue(a.AsElement())
	if !ok{
		panic("element is not connected to Native dom node.")
	}
	return j.Get("playbackRate").Float()
}

func(a AudioElement)Ended() bool{
	j,ok:= JSValue(a.AsElement())
	if !ok{
		panic("element is not connected to Native dom node.")
	}
	return j.Get("ended").Bool()
}

func(a AudioElement)ReadyState() float64{
	j,ok:= JSValue(a.AsElement())
	if !ok{
		panic("element is not connected to Native dom node.")
	}
	return j.Get("readyState").Float()
}

func(a AudioElement)Seekable()  jsTimeRanges{
	j,ok:= JSValue(a.AsElement())
	if !ok{
		panic("element is not connected to Native dom node.")
	}
	b:= j.Get("seekable")
	return newTimeRanges(b)
}

func(a AudioElement) Volume() float64{
	j,ok:= JSValue(a.AsElement())
	if !ok{
		panic("element is not connected to Native dom node.")
	}
	return j.Get("volume").Float()
}


func(a AudioElement) Muted() bool{
	j,ok:= JSValue(a.AsElement())
	if !ok{
		panic("element is not connected to Native dom node.")
	}
	return j.Get("muted").Bool()
}

func(a AudioElement) Paused() bool{
	j,ok:= JSValue(a.AsElement())
	if !ok{
		panic("element is not connected to Native dom node.")
	}
	return j.Get("paused").Bool()
}

func(a AudioElement) Loop() bool{
	j,ok:= JSValue(a.AsElement())
	if !ok{
		panic("element is not connected to Native dom node.")
	}
	return j.Get("loop").Bool()
}



func(v VideoElement) Buffered() jsTimeRanges{
	j,ok:= JSValue(v.AsElement())
	if !ok{
		panic("element is not connected to Native dom node.")
	}
	b:= j.Get("buiffered")
	return newTimeRanges(b)
}

func(v VideoElement)CurrentTime() time.Duration{
	j,ok:= JSValue(v.AsElement())
	if !ok{
		panic("element is not connected to Native dom node.")
	}
	return time.Duration(j.Get("currentTime").Float())* time.Second
}

func(v VideoElement)Duration() time.Duration{
	j,ok:= JSValue(v.AsElement())
	if !ok{
		panic("element is not connected to Native dom node.")
	}
	return  time.Duration(j.Get("duration").Float())*time.Second
}

func(v VideoElement)PlayBackRate() float64{
	j,ok:= JSValue(v.AsElement())
	if !ok{
		panic("element is not connected to Native dom node.")
	}
	return j.Get("playbackRate").Float()
}

func(v VideoElement)Ended() bool{
	j,ok:= JSValue(v.AsElement())
	if !ok{
		panic("element is not connected to Native dom node.")
	}
	return j.Get("ended").Bool()
}

func(v VideoElement)ReadyState() float64{
	j,ok:= JSValue(v.AsElement())
	if !ok{
		panic("element is not connected to Native dom node.")
	}
	return j.Get("readyState").Float()
}

func(v VideoElement)Seekable()  jsTimeRanges{
	j,ok:= JSValue(v.AsElement())
	if !ok{
		panic("element is not connected to Native dom node.")
	}
	b:= j.Get("seekable")
	return newTimeRanges(b)
}

func(v VideoElement) Volume() float64{
	j,ok:= JSValue(v.AsElement())
	if !ok{
		panic("element is not connected to Native dom node.")
	}
	return  j.Get("volume").Float()
}


func(v VideoElement) Muted() bool{
	j,ok:= JSValue(v.AsElement())
	if !ok{
		panic("element is not connected to Native dom node.")
	}
	return j.Get("muted").Bool()
}

func(v VideoElement) Paused() bool{
	j,ok:= JSValue(v.AsElement())
	if !ok{
		panic("element is not connected to Native dom node.")
	}
	return j.Get("paused").Bool()
}

func(v VideoElement) Loop() bool{
	j,ok:= JSValue(v.AsElement())
	if !ok{
		panic("element is not connected to Native dom node.")
	}
	return j.Get("loop").Bool()
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



func GetAttribute(target *ui.Element, name string) string {
	native, ok := target.Native.(NativeElement)
	if !ok {
		log.Print("Cannot retrieve Attribute on non-expected wrapper type")
		return ""
	}
	res:= native.Value.Call("getAttribute", name)
	if  res.IsNull(){
		return "null"
	}
	return res.String()
}

// abstractjs
func SetAttribute(target *ui.Element, name string, value string) {
	var attrmap ui.Object
	m, ok := target.Get("data", "attrs")
	if !ok {
		attrmap = ui.NewObject()
	} else {
		attrmap, ok = m.(ui.Object)
		if !ok {
			panic("data/attrs should be stored as a ui.Object")
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

// abstractjs
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


// abstractjs
var textContentHandler = ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
	j,ok:= JSValue(evt.Origin())
	if !ok{
		return false
	}

	str, ok := evt.NewValue().(ui.String)
	if !ok {
		return true
	}
	j.Set("textContent", string(str))

	return false
})


func clampedValueWatcher(propname string, min int,max int) *ui.MutationHandler{
	return ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		j,ok:= JSValue(evt.Origin())
		if !ok{
			return false
		}
		v:= float64(evt.NewValue().(ui.Number))
		if v < float64(min){
			v = float64(min)
		}

		if v > float64(max){
			v = float64(max)
		}
		j.Set(propname,v)
		return false
	})
}

func numericPropertyWatcher(propname string) *ui.MutationHandler{
	return ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		j,ok:= JSValue(evt.Origin())
		if !ok{
			panic("element doesn't seem to have been connected to thecorresponding Native DOM Element")
		}
		j.Set(propname,float64(evt.NewValue().(ui.Number)))
		return false
	})
}

func boolPropertyWatcher(propname string) *ui.MutationHandler{
	return ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		j,ok:= JSValue(evt.Origin())
		if !ok{
			panic("element doesn't seem to have been connected to thecorresponding Native DOM Element")
		}
		j.Set(propname,bool(evt.NewValue().(ui.Bool)))
		return false
	})
}

func stringPropertyWatcher(propname string) *ui.MutationHandler{
	return ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		j,ok:= JSValue(evt.Origin())
		if !ok{
			panic("element doesn't seem to have been connected to thecorresponding Native DOM Element")
		}
		j.Set(propname,string(evt.NewValue().(ui.String)))
		return false
	})
}

