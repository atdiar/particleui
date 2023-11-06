//go:build !server 

// Package doc defines the default set of Element constructors, native interfaces,
// events and event handlers, and animation properties used to build js-based UIs.


package doc

import (
	"context"
	"encoding/json"
	//"github.com/segmentio/encoding/json"
	"errors"
	"log"
	"strings"
	"syscall/js"
	"runtime"
	"runtime/debug"
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
		AddConstructorOptionsTo("observable",AllowSessionStoragePersistence,AllowAppLocalStoragePersistence).
		ApplyGlobalOption(allowdatapersistence)
)

// TODO on init, Apply EnableMutationCapture to Elements if ldlflags -X tag is set for the buildtype variable to "dev" 
// Also, the mutationtrace should be stored in the sessionstorage or localstorage
// And the mutationtrace should replay once the document is ready.


// NewBuilder registers a new document building function.
func NewBuilder(f func()Document)(ListenAndServe func(context.Context)){
	return  func(ctx context.Context){
		// GC is triggered only when the browser is idle.
		debug.SetGCPercent(-1)
		js.Global().Set("triggerGC", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			runtime.GC()
			return nil
		}))

		d:=f()
		withNativejshelpers(&d)

		scrIdleGC := d.Script().SetInnerHTML(`
			let lastGC = Date.now();

			function runGCDuringIdlePeriods(deadline) {

				if (deadline.didTimeout || !deadline.timeRemaining()) {
					setTimeout(() => window.requestIdleCallback(runGCDuringIdlePeriods), 120000); // Schedule next idle callback in 2 minutes
					return;
				}
				
				let now = Date.now();
				if (now - lastGC >= 120000) { // Check if at least 2 minutes passed since last GC
					window.triggerGC(); // Trigger GC
					lastGC = now;
				}

				// Schedule a new callback for the next idle time, but not sooner than 2 minutes from now
				setTimeout(() => window.requestIdleCallback(runGCDuringIdlePeriods), 120000); // Schedule next idle callback in 2 minutes
			}

			// Start the loop
			window.requestIdleCallback(runGCDuringIdlePeriods);
	
		`)
		d.Head().AppendChild(scrIdleGC)

		/*d.WatchEvent("document-loaded",d,ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			evt.Origin().WatchEvent("document-loaded", evt.Origin(),ui.NewMutationHandler(func(event ui.MutationEvent)bool{
				js.Global().Call("onWasmDone")
				return false
			}))
			
			return false
		}))*/
		

		d.AfterEvent("document-loaded",d,ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			js.Global().Call("onWasmDone")
			return false
		}))

		err := d.mutationRecorder().Replay()
		if err != nil{
			d.mutationRecorder().Clear()
			d=f()
			withNativejshelpers(&d)

			scrIdleGC := d.Script().SetInnerHTML(`
				let lastGC = Date.now();

				function runGCDuringIdlePeriods(deadline) {

					if (deadline.didTimeout || !deadline.timeRemaining()) {
						setTimeout(() => window.requestIdleCallback(runGCDuringIdlePeriods), 120000); // Schedule next idle callback in 2 minutes
						return;
					}
					
					let now = Date.now();
					if (now - lastGC >= 120000) { // Check if at least 2 minutes passed since last GC
						window.triggerGC(); // Trigger GC
						lastGC = now;
					}

					// Schedule a new callback for the next idle time, but not sooner than 2 minutes from now
					setTimeout(() => window.requestIdleCallback(runGCDuringIdlePeriods), 120000); // Schedule next idle callback in 2 minutes
				}

				// Start the loop
				window.requestIdleCallback(runGCDuringIdlePeriods);
		
			`)
			d.Head().AppendChild(scrIdleGC)

			d.AfterEvent("document-loaded",d,ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
				js.Global().Call("onWasmDone")
				return false
			}))

			DEBUG(err)
		}
		d.mutationRecorder().Capture()

		d.ListenAndServe(ctx)
	}
}

/*

SSR is implemented as mutation capture without replay on the server and
mutation replay without capture on the client.

Hot Reloading is implemented as mutation capture with replay on the client.

Pure CSR (client side rendering) does not capture, nor replay mutations.
Pure SSG (server side rendering) does not capture, nor replay mutations.

*/

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
		if category != "data"{
			return 
		}
		store := jsStore{js.Global().Get(s)}
		_,ok:= store.Get("zui-connected")
		if !ok{
			return
		}

		props := make([]interface{}, 0, 64)

		c,ok:= element.Properties.Categories[category]
		if !ok{
			props = append(props, propname)
			// log.Print("all props stored...", props) // DEBUG
			v := js.ValueOf(props)
			store.Set(element.ID, v) 
		} else{
			for k:= range c.Local{
				props = append(props, k)
			}
			v := js.ValueOf(props)
			store.Set(element.ID, v)
		}
	
		item := value.RawValue()
		v := stringify(item)
		store.Set(strings.Join([]string{element.ID, category, propname}, "/"),js.ValueOf(v))
		return
	}
}



var sessionstorefn = storer("sessionStorage")
var localstoragefn = storer("localStorage")

func loader(s string) func(e *ui.Element) error { // abstractjs
	return func(e *ui.Element) error {
		
		store := jsStore{js.Global().Get(s)}
		_,ok:= store.Get("zui-connected")
		if !ok{
			return errors.New("storage is disconnected")
		}
		id := e.ID

		// Let's retrieve the category index for this element, if it exists in the sessionstore
		jsonprops, ok := store.Get(id)
		if !ok {
			return nil // Not necessarily an error in the general case. element just does not exist in store
		}

		properties := make([]string, 0, 64)
		err := json.Unmarshal([]byte(jsonprops.String()), &properties)
		if err != nil {
			return err
		}

		category:= "data"
		uiloaders:= make([]func(),0,64)

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
				val:= ui.ValueFrom(rawvalue)

				ui.LoadProperty(e, category, propname, val)
				if category == "data"{
					uiloaders = append(uiloaders, func(){
						if e.IsRenderData(propname){
							e.SetUI(propname, val)
						}
					})
				}
				//log.Print("LOADED PROPMAP: ", e.Properties, category, propname, rawvalue.Value()) // DEBUG
			}
		}

		
		//log.Print(categories, properties) //DEBUG
		
		e.OnRegistered(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			for _,load:= range uiloaders{
				load()
			}
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
		_,ok:= store.Get("zui-connected")
		if !ok{
			return 
		}
		id := element.ID
		category:= "data"

		// Let's retrieve the category index for this element, if it exists in the sessionstore
		jsonproperties, ok := store.Get(id)
		if !ok {
			return
		}

		properties := make([]string, 0, 50)
		
		err := json.Unmarshal([]byte(jsonproperties.String()), &properties)
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

		store.Delete(id)
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


func nativeDocumentAlreadyRendered() bool{
	//  get native document status by looking for the ssr hint encoded in the page (data attribute)
	// the data attribute should be removed once the document state is replayed.
	statenode:= js.Global().Get("document)").Call("getElementById",SSRStateElementID )
	if !statenode.Truthy(){
		// TODO: check if the document is already rendered, at least partially, still.
		return false
	}

	/*
	state:= statenode.Get("textContent").String()
	v,err:= DeserializeStateHistory(state)
	if err != nil{
		panic(err)
	}
	
	root.Set("internals","mutationtrace",v)	
	root.WatchEvent("mutation-replayed",root,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		statenode.Call("remove")
		return false
	}))
	*/

	return true
}

func ConnectNative(e *ui.Element, tag string){
	id:= e.ID
	if e.IsRoot(){
		if  nativeDocumentAlreadyRendered() && e.ElementStore.MutationReplay{
			e.ElementStore.Disconnected = true

			statenode:= js.Global().Get("document)").Call("getElementById",SSRStateElementID )
			state:= statenode.Get("textContent").String()
			v,err:= DeserializeStateHistory(state)
			if err != nil{
				panic(err)
			}
			
			e.Set("internals","mutationtrace",v)	

			e.WatchEvent("mutation-replayed",e,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{		
				// TODO check value to see if replay  error or not?
				e.ElementStore.MutationReplay = false
				statenode.Call("remove")
				evt.Origin().TriggerEvent("connect-native")
				evt.Origin().ElementStore.Disconnected = false
				return false
			}))
		}
	}
	
	if e.ElementStore.Disconnected{
		e.WatchEvent("connect-native",e,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{

			if tag == "window"{
				wd := js.Global().Get("document").Get("defaultView")
				if !wd.Truthy() {
					panic("unable to access windows")
				}
				evt.Origin().Native = NewNativeElementWrapper(wd, "Window")
				return false
			}
		
			if tag == "html"{
				// connect localStorage and sessionStorage
				ls :=  jsStore{js.Global().Get("localStorage")}
				ss :=  jsStore{js.Global().Get("sessionStorage")}
				ls.Set("zui-connected",js.ValueOf(true))
				ss.Set("zui-connected",js.ValueOf(true))

				root:= js.Global().Get("document").Call("getElementById",id)
				if !root.Truthy(){
					root = js.Global().Get("document").Get("documentElement")
					if !root.Truthy() {
						panic("failed to instantiate root element for the document")
					}
				}
				evt.Origin().Native = NewNativeHTMLElement(root)
				SetAttribute(e, "id", evt.Origin().ID)

				
				return false
			}
		
			if tag == "body"{
				element:= js.Global().Get("document").Call("getElementById",id)
				if !element.Truthy(){
					element= js.Global().Get("document").Get(tag)
					if !element.Truthy(){
						element= js.Global().Get("document").Call("createElement",tag)
					}
				}
				evt.Origin().Native = NewNativeHTMLElement(element)
				SetAttribute(e, "id", evt.Origin().ID)
				return false
			}

			if tag == "script"{
				cancreeatewithID := js.Global().Get("createElementWithID").Truthy()
				element:= js.Global().Get("document").Call("getElementById",id)
				if !element.Truthy(){
					if !cancreeatewithID{
						element= js.Global().Get("document").Call("createElement",tag)
					} else{
						element= js.Global().Call("createElementWithID",tag,id)
					}
				}
				evt.Origin().Native = NewNativeHTMLElement(element)
				SetAttribute(e, "id", evt.Origin().ID)
				return false
		 	}
		
			if tag == "head"{
				element:= js.Global().Get("document").Call("getElementById",id)
				defer func(){
					// We should also add the scrip that enables batch execution:
					batchscript := js.Global().Get("document").Call("createElement","script")
					batchscript.Set("textContent",`
					window.elements = {};
		
					window.createElementWithID = function(tagName, id) {
					  let element = document.createElement(tagName);
					  element.id = id;
					  window.elements[id] = element;
					  return element;
					};
					
					window.deleteElementWithID = function(id) {
					  let element = window.elements[id];
					  if (element) {
						if (element.parentNode) {
						  element.parentNode.removeChild(element);
						}
						delete window.elements[id];
					  }
					};
		
					
					window.applyBatchOperations = function(parentElementID, encodedOperations) {
						const operationsBinary = atob(encodedOperations);
						const operationsData = new DataView(new Uint8Array(operationsBinary.split('').map(ch => ch.charCodeAt(0))).buffer);

					
						for (let i = 0; i < operationsBinary.length; i++) {
							operationsData.setUint8(i, operationsBinary.charCodeAt(i));
						}
					
						let offset = 0;
						const fragment = document.createDocumentFragment();
						const parentElement = window.getElement(parentElementID); // get parent element using its ID
					
						while (offset < operationsData.byteLength) {
							const operationLen = operationsData.getUint8(offset++);
							const operation = operationsBinary.slice(offset, offset + operationLen);
							offset += operationLen;
					
							const idLen = operationsData.getUint8(offset++);
							const elementID = operationsBinary.slice(offset, offset + idLen);
							offset += idLen;
					
							const index = operationsData.getUint32(offset);
							offset += 4;
					
							const element = window.getElement(elementID);
							if (!element) continue;
					
							switch (operation) {
								case "Insert":
									if (fragment.children.length > index) {
										fragment.insertBefore(element, fragment.children[index]);
									} else {
										fragment.appendChild(element);
									}
									break;
								case "Remove":
									if (element.parentNode) {
										element.parentNode.removeChild(element);
									}
									break;
							}
						}
						
						parentElement.appendChild(fragment);
					};
					
					`)
					element.Call("append", batchscript)
				}()
				if !element.Truthy(){
					element= js.Global().Get("document").Get(tag)
					if !element.Truthy(){
						element= js.Global().Call("createElement",tag)
					}
				}
		
				evt.Origin().Native = NewNativeHTMLElement(element)
				SetAttribute(e, "id", e.ID)
				return false
			}
		
			element:= js.Global().Call("getElement",id)
			if !element.Truthy(){
				element= js.Global().Call("createElementWithID",tag,id)
			}
			evt.Origin().Native = NewNativeHTMLElement(element) 
				
			return false

		}).RunOnce())

		return
	}
	if tag == "window"{
		wd := js.Global().Get("document").Get("defaultView")
		if !wd.Truthy() {
			panic("unable to access windows")
		}
		e.Native =  NewNativeElementWrapper(wd, "Window")
		return 
	}

	if tag == "html"{
		// connect localStorage and sessionSTtorage
		ls :=  jsStore{js.Global().Get("localStorage")}
		ss :=  jsStore{js.Global().Get("sessionStorage")}
		ls.Set("zui-connected",js.ValueOf(true))
		ss.Set("zui-connected",js.ValueOf(true))

		root:= js.Global().Get("document").Call("getElementById",id)
		if !root.Truthy(){
			root = js.Global().Get("document").Get("documentElement")
			if !root.Truthy() {
				panic("failed to instantiate root element for the document")
			}
			e.Native =  NewNativeHTMLElement(root)
			return 
		}
		e.Native =  NewNativeHTMLElement(root)
		SetAttribute(e, "id", e.ID)
		
		return
	}

	if tag == "body"{
		element:= js.Global().Get("document").Call("getElementById",id)
		if !element.Truthy(){
			element= js.Global().Get("document").Get(tag)
			if !element.Truthy(){
				element= js.Global().Get("document").Call("createElement",tag)
			}
			e.Native = NewNativeHTMLElement(element)
			SetAttribute(e, "id", e.ID)
			return
		}
		e.Native =  NewNativeHTMLElement(element)
		SetAttribute(e, "id", e.ID)
		return
	}

	if tag == "script"{
		cancreeatewithID := js.Global().Get("createElementWithID").Truthy()
		element:= js.Global().Get("document").Call("getElementById",id)
		if !element.Truthy(){
			if !cancreeatewithID{
				element= js.Global().Get("document").Call("createElement",tag)
			} else{
				element= js.Global().Call("createElementWithID",tag,id)
			}
		}
		e.Native = NewNativeHTMLElement(element)
		SetAttribute(e, "id", e.ID)
		return
	}

	if tag == "head"{
		element:= js.Global().Get("document").Call("getElementById",id)
		defer func(){
			// We should also add the scrip that enables batch execution:
			batchscript := js.Global().Get("document").Call("createElement","script")
			batchscript.Set("textContent",`
			window.elements = {};

			window.getElement = function(id) {
				let element = document.getElementById(id);
				if (!element) {
				  element = window.elements[id];
				}
				return element;
			};

			window.createElementWithID = function(tagName, id) {
			  let element = document.createElement(tagName);
			  element.id = id;
			  window.elements[id] = element;
			  return element;
			};
			
			window.deleteElementWithID = function(id) {
			  let element = window.elements[id];
			  if (element) {
				if (element.parentNode) {
				  element.parentNode.removeChild(element);
				}
				delete window.elements[id];
			  }
			};

			
			window.applyBatchOperations = function(parentElementID, encodedOperations) {
				const operationsBinary = atob(encodedOperations);
				const operationsData = new DataView(new Uint8Array(operationsBinary.split('').map(ch => ch.charCodeAt(0))).buffer);

			
				for (let i = 0; i < operationsBinary.length; i++) {
					operationsData.setUint8(i, operationsBinary.charCodeAt(i));
				}
			
				let offset = 0;
				const fragment = document.createDocumentFragment();
				const parentElement = window.getElement(parentElementID); // get parent element using its ID
			
				while (offset < operationsData.byteLength) {
					const operationLen = operationsData.getUint8(offset++);
					const operation = operationsBinary.slice(offset, offset + operationLen);
					offset += operationLen;
			
					const idLen = operationsData.getUint8(offset++);
					const elementID = operationsBinary.slice(offset, offset + idLen);
					offset += idLen;
			
					const index = operationsData.getUint32(offset);
					offset += 4;
			
					const element = window.getElement(elementID);
					if (!element) continue;
			
					switch (operation) {
						case "Insert":
							if (fragment.children.length > index) {
								fragment.insertBefore(element, fragment.children[index]);
							} else {
								fragment.appendChild(element);
							}
							break;
						case "Remove":
							if (element.parentNode) {
								element.parentNode.removeChild(element);
							}
							break;
					}
				}
				
				parentElement.appendChild(fragment);
			};
			
			(function() {
				const originalDocumentElement = document.documentElement;
	
				const handler = {
					get: function(target, propKey) {
						const origMethod = target[propKey];
						if (typeof origMethod === 'function') {
							return function(...args) {
								if (propKey === 'scrollTo') {
									console.log('scrollTo', args);
								}
								return origMethod.apply(this, args);
							};
						} else if (propKey === 'scrollTop' || propKey === 'scrollLeft') {
							return target[propKey];
						}
					},
					set: function(target, propKey, value) {
						if (propKey === 'scrollTop' || propKey === 'scrollLeft') {
							console.log(propKey, value);
						}
						target[propKey] = value;
						return true;
					}
				};
	
				const proxy = new Proxy(originalDocumentElement, handler);
				document.documentElement = proxy;
			})();
			
			`)
			element.Call("append", batchscript)
		}()
		if !element.Truthy(){
			element= js.Global().Get("document").Get(tag)
			if !element.Truthy(){
				element= js.Global().Call("createElement",tag)
			}
			e.Native =  NewNativeHTMLElement(element)
			SetAttribute(e, "id", e.ID)
			return
		}
		
		e.Native =  NewNativeHTMLElement(element)
		SetAttribute(e, "id", e.ID)
		return
	}

	element:= js.Global().Call("getElement",id)
	if !element.Truthy(){
		element= js.Global().Call("createElementWithID",tag,id)
		e.Native = NewNativeHTMLElement(element)
		return
	}
	e.Native =  NewNativeHTMLElement(element)
	return
}


// NativeElement defines a wrapper around a js.Value that implements the
// ui.NativeElementWrapper interface.
type NativeElement struct {
	js.Value
	typ string
}

// NewNativeElementWrapper creates a new NativeElement from a js.Value.
func NewNativeElementWrapper(v js.Value, typ string) NativeElement {
	return NativeElement{v,typ}
}

// NewNativeHTMLElement creates a new Native HTML Element from a js.Value of a HTMLELement.
func NewNativeHTMLElement(v js.Value) NativeElement {
	return NativeElement{v,"HTMLElement"}
}

func (n NativeElement) AppendChild(child *ui.Element) {
	v, ok := child.Native.(NativeElement)
	if !ok {
		log.Print("wrong format for native element underlying objects.Cannot append " + child.ID)
		return
	}
	if n.typ == "HTMLElement"{
		n.Value.Call("append", v.Value)
	}
	
}

func (n NativeElement) PrependChild(child *ui.Element) {
	v, ok := child.Native.(NativeElement)
	if !ok {
		log.Print("wrong format for native element underlying objects.Cannot prepend " + child.ID)
		return
	}
	if n.typ == "HTMLElement"{
		n.Value.Call("prepend", v.Value)
	}
}

func (n NativeElement) InsertChild(child *ui.Element, index int) {
	v, ok := child.Native.(NativeElement)
	if !ok {
		log.Print("wrong format for native element underlying objects.Cannot insert " + child.ID)
		return
	}
	if n.typ == "HTMLElement"{
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
}

func (n NativeElement) ReplaceChild(old *ui.Element, new *ui.Element) {
	nold, ok := old.Native.(NativeElement)
	if !ok {
		log.Print("wrong format for native element underlying objects.Cannot replace " + old.ID)
		return
	}
	if n.typ == "HTMLElement"{
		nnew, ok := new.Native.(NativeElement)
		if !ok {
			log.Print("wrong format for native element underlying objects.Cannot replace with " + new.ID)
			return
		}
		//nold.Call("replaceWith", nnew) also works
		n.Value.Call("replaceChild", nnew.Value, nold.Value)
	}
}

func (n NativeElement) RemoveChild(child *ui.Element) {
	v, ok := child.Native.(NativeElement)
	if !ok {
		log.Print("wrong format for native element underlying objects.Cannot remove ", child.Native)
		return
	}
	if n.typ == "HTMLElement"{
		v.Value.Call("remove")
	}

}

func (n NativeElement) Delete(child *ui.Element){
	if n.typ == "HTMLElement"{
		js.Global().Call("deleteElementWithID",child.ID)
	}
}


func (n NativeElement) SetChildren(children ...*ui.Element) {
	if n.typ == "HTMLElement"{
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
}


func (n NativeElement) BatchExecute(parentid string, opslist string) {
	if n.typ == "HTMLElement"{
		js.Global().Call("applyBatchOperations",parentid, opslist)
	}
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

	if e == nil {
		panic("loading a nil element")
	}

	pmode := ui.PersistenceMode(e)

	storage, ok := e.ElementStore.PersistentStorer[pmode]
	
	if ok {
		err := storage.Load(e)
		if err != nil {
			panic(err)
		}
	}

	return e
}

// PutInStorage stores an element data in storage (localstorage or sessionstorage).
func PutInStorage(e *ui.Element) *ui.Element{
	pmode := ui.PersistenceMode(e)
	storage,ok:= e.ElementStore.PersistentStorer[pmode]
	if !ok{
		return e
	}


	for cat,props:= range e.Properties.Categories{
		if cat != "data"{
			continue
		}
		for prop,val:= range props.Local{
			storage.Store(e,cat,prop,val)
		}
	}
	return e
}

// ClearFromStorage will clear an element properties from storage.
func ClearFromStorage(e *ui.Element) *ui.Element{
	pmode:=ui.PersistenceMode(e)

	storage,ok:= e.ElementStore.PersistentStorer[pmode]
	if ok{
		storage.Clear(e)
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
// 
var AllowScrollRestoration = ui.NewConstructorOption("scrollrestoration", func(el *ui.Element) *ui.Element {
	el.WatchEvent("registered", el.Root, ui.NewMutationHandler(func(event ui.MutationEvent) bool {
		e:=event.Origin()
		if e.IsRoot(){
			if js.Global().Get("history").Get("scrollRestoration").Truthy() {
				js.Global().Get("history").Set("scrollRestoration", "manual")
			}
			e.WatchEvent("document-ready",e,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
				rootScrollRestorationSupport(evt.Origin())
				return false
			}).RunOnce()) // TODO Check that we really want to do this on the main document on navigation-end.
			
			return false
		}
	
		e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			e.WatchEvent("document-ready", e.Root, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
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
					e.SetUI("scrollrestore", ui.Bool(true)) // DEBUG SetUI instead of SetDataSetUI, as this is no business logic but UI logic
					e.AddEventListener("scroll", ui.NewEventHandler(func(evt ui.Event) bool {
						scrolltop := ui.Number(ejs.Get("scrollTop").Float())
						scrollleft := ui.Number(ejs.Get("scrollLeft").Float())
						router.History.Set(e.ID+"-"+"scrollTop", scrolltop)
						router.History.Set(e.ID+"-"+"scrollLeft", scrollleft)
						return false
					}))
	
					h := ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
						b, ok := e.GetEventValue("shouldscroll")
						if !ok {
							return false
						}
						if scroll := b.(ui.Bool); scroll {
							t, ok := router.History.Get(e.ID+"-"+"scrollTop")
							if !ok {
								ejs.Set("scrollTop", 0)
								ejs.Set("scrollLeft", 0)
								return false
							}
							l, ok := router.History.Get(e.ID+"-"+"scrollLeft")
							if !ok {
								ejs.Set("scrollTop", 0)
								ejs.Set("scrollLeft", 0)
								return false
							}
							top := t.(ui.Number)
							left := l.(ui.Number)
							ejs.Set("scrollTop", float64(top))
							ejs.Set("scrollLeft", float64(left))
							if e.ID != e.Root.ID {
								e.TriggerEvent("shouldscroll", ui.Bool(false)) //always scroll root
							}
						}
						return false
					}).RunASAP()

					e.WatchEvent("document-ready",e.Root,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
						evt.Origin().WatchEvent("navigation-end", evt.Origin().Root, h)
						return false
					}).RunASAP().RunOnce())

				} else {
					e.SetUI("scrollrestore", ui.Bool(false)) // DEBUG SetUI instead of SetDataSetUI as this is not business logic
				}
				return false
			}))
			return false
		}).RunASAP().RunOnce())
	
		e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent) bool { // TODO DEBUG Mounted is not the appopriate event
	
			sc, ok := e.GetUI("scrollrestore")
			if !ok {
				return false
			}
			if scrollrestore := sc.(ui.Bool); scrollrestore {
				e.TriggerEvent("shouldscroll", ui.Bool(true))
			}
			return false
		}))

		return false
	}).RunASAP())
	return el
	
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
		bcursor,ok:= browserhistory.Get("cursor")
		if ok{
			bhc:= bcursor.(ui.Number)
			hcursor,ok:= history.Get("cursor")
			if !ok{
				panic("history cursor is missing")
			}
			hc:= hcursor.(ui.Number)
			if bhc==hc {
				s := stringify(history.RawValue())
				js.Global().Get("history").Call("replaceState", js.ValueOf(s), "", route)
			} else{
				s := stringify(history.RawValue())
				js.Global().Get("history").Call("pushState", js.ValueOf(s), "", route)
			}
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
		hstateobj := make(map[string]interface{})
		err := json.Unmarshal([]byte(hstate.String()), &hstateobj)
		if err == nil {
			hso:= ui.ValueFrom(hstateobj).(ui.Object)
			// Check that the sate is valid. It is valid if it contains a cursor.
			_, ok := hso.Get("cursor")
			if ok{
				evt.Origin().SyncUISyncData("history", hso)
			} else{
				evt.Origin().SyncUI("history", hso.Value())
			}
		}
	}
	
	route := js.Global().Get("location").Get("pathname").String()
	e.TriggerEvent("navigation-routechangerequest", ui.String(route))
	return false
})


var rootScrollRestorationSupport = func(root *ui.Element)*ui.Element { // abstractjs
	e:= root
	n:= e.Native.(NativeElement).Value
	r := ui.GetRouter(root)

	ejs := js.Global().Get("document").Get("scrollingElement")

	e.SetUI("scrollrestore", ui.Bool(true)) // DEBUG SetUI instead of SetDataSetUI, as this is no business logic but UI logic

	d:= getDocumentRef(e)
	d.Window().AsElement().AddEventListener("scroll", ui.NewEventHandler(func(evt ui.Event) bool {
		scrolltop := ui.Number(ejs.Get("scrollTop").Float())
		scrollleft := ui.Number(ejs.Get("scrollLeft").Float())
		r.History.Set(e.ID+"-"+"scrollTop", scrolltop)
		r.History.Set(e.ID+"-"+"scrollLeft", scrollleft)
		return false
	}))
	

	h := ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		router := ui.GetRouter(evt.Origin().Root)
		newpageaccess:= router.History.CurrentEntryIsNew()
		
		t, oktop := router.History.Get(e.ID+"-"+"scrollTop")
		l, okleft := router.History.Get(e.ID+"-"+"scrollLeft")

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
		v,ok:= router.History.Get("focusedElementId")
		if !ok{
			v,ok= e.Get("ui","focus")
			if !ok{
				return false
			}
			elid:=v.(ui.String).String()
			el:= getDocumentRef(e).GetElementById(elid)

			if el != nil && el.Mounted(){
				Focus(el,false)
				if newpageaccess{
					if !partiallyVisible(el){
						n.Call("scrollIntoView")
					}
				}
					
			}
		} else{
			elid:=v.(ui.String).String()
			el:= getDocumentRef(e).GetElementById(elid)

			if el != nil && el.Mounted(){

				Focus(el,false)
				if newpageaccess{
					if !partiallyVisible(el){
						n.Call("scrollIntoView")
					}
				}
					
			}
		}
		
		return false
	}).RunASAP()

	e.WatchEvent("document-ready",e,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		evt.Origin().WatchEvent("navigation-end", evt.Origin(), h)
		return false
	}).RunASAP().RunOnce())

	return e
}

func activityStateSupport(e *ui.Element)*ui.Element{
	GetDocument(e).Window().AsElement().AddEventListener("pagehide", ui.NewEventHandler(func(evt ui.Event) bool {
		e.TriggerEvent("before-unactive")
		return false
	}))

		// visibilitychange
	e.AddEventListener("visibilitychange", ui.NewEventHandler(func(evt ui.Event) bool {
		visibilityState := js.Global().Get("document").Get("visibilityState").String()
		if visibilityState == "hidden"{
			e.TriggerEvent("before-unactive")
		} 
		return false
	}))

	return e
}

// Focus triggers the focus event asynchronously on the JS side.
func Focus(e ui.AnyElement, scrollintoview bool){ // abstractjs
	if !e.AsElement().Mounted(){
		return
	}

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
}

func focus(e js.Value){ // abstractjs
	js.Global().Call("queueFocus", e)
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

	w,ok:= JSValue(getDocumentRef(e).Window().AsElement())
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
		evt.Origin().Root.AddEventListener("keydown",h)
		// Watches unmounted once
		evt.Origin().OnUnmounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			evt.Origin().Root.RemoveEventListener("keydown",h)
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
			e.SyncUISyncData("text", ui.String(s))
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
	js.Global().Call("queueBlur", native.Value)
}

func (i InputElement) Focus() {
	native, ok := i.AsElement().Native.(NativeElement)
	if !ok {
		panic("native element should be of doc.NativeELement type")
	}
	js.Global().Call("queueFocus", native.Value)
	
}

func (i InputElement) Clear() {
	native, ok := i.AsElement().Native.(NativeElement)
	if !ok {
		panic("native element should be of doc.NativeELement type")
	}
	js.Global().Call("queueClear", native.Value)
	
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
		starts.Set(i,st)
		ends.Set(i,en)
	}
	j.Set("start",starts.Commit())
	j.Set("end",ends.Commit())
	return jsTimeRanges(j.Commit())
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

// todo abstractjs
func makeStyleSheet(observable *ui.Element, id string) *ui.Element {
	
	new := ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		rss:= js.Global().New("CSSStyleSheet",struct{
			baseURL string 
			media []any 
			disabled bool
			}{"", nil, false},
		)
		evt.Origin().Native = NativeElement{Value: rss, typ: "CSSStyleSheet"}

		d,ok:= JSValue(GetDocument(evt.Origin()))
		if !ok{
			panic("stylesheet is not registered on document or document is not connected to its native dom element")
		}

		d.Get("adoptedStyleSheets").Call("concat",rss)
		

		return false
	})

	enable:= ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		s,ok:= JSValue(evt.Origin())
		if !ok{
			return false
		}
		s.Set("disabled",false)
		evt.Origin().SetUI("active", ui.Bool(true))
		return false
	})

	disable:= ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		s,ok:= JSValue(evt.Origin())
		if !ok{
			return false
		}
		s.Set("disabled",true)
		evt.Origin().SetUI("active", ui.Bool(false))
		return false
	})

	update := ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		s,ok:= JSValue(evt.Origin())
		if !ok{
			return false
		}
		s.Call("replaceSync",StyleSheet{evt.Origin()}.String())
		return false
	})
	observable.WatchEvent("new", observable, new)
	observable.WatchEvent("enable", observable, enable)
	observable.WatchEvent("disable", observable, disable)
	observable.Watch("ui","stylesheet", observable, update)
	observable.OnDeleted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		// TODO remove from adopted stylesheets
		d,ok:= JSValue(GetDocument(evt.Origin()))
		if !ok{
			panic("stylesheet is not registered on document or document is not connected to its native dom element")
		}

		sheet,ok:= JSValue(evt.Origin())
		if !ok{
			panic("stylesheet is not connected to its native dom element")
		}

		as := d.Get("adoptedStyleSheets")
		fas := js.Global().Call("filterByValue", as,sheet)
		d.Set("adoptedStyleSheets",fas)

		return false
	}))
	return observable
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
	var am =  ui.NewObject()
	m, ok := target.Get("data", "attrs")
	if ok {
		attrmap, ok = m.(ui.Object)
		if !ok {
			panic("data/attrs should be stored as a ui.Object")
		}
		am = attrmap.MakeCopy()
	}

	attrmap = am.Set(name, ui.String(value)).Commit()
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
	var am = ui.NewObject()
	attrmap, ok := m.(ui.Object)
	if !ok {
		panic("data/attrs should be stored as a ui.Object")
	}
	am = attrmap.MakeCopy()
	am.Delete(name)
	target.SetData("attrs", am.Commit())
	
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
}).RunASAP() // TODO DEBUG just added RunASAP


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
	}).RunASAP()
}

func numericPropertyWatcher(propname string) *ui.MutationHandler{
	return ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		j,ok:= JSValue(evt.Origin())
		if !ok{
			panic("element doesn't seem to have been connected to thecorresponding Native DOM Element")
		}
		j.Set(propname,float64(evt.NewValue().(ui.Number)))
		return false
	}).RunASAP()
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
	}).RunASAP()
}

