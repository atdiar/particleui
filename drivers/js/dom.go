// Package doc defines the default set of Element constructors, native interfaces,
// events and event handlers, and animation properties used to build js-based UIs.
package doc

import (
	"encoding/json"
	//"errors"
	"log"
	"strconv"
	"strings"

	//"syscall/js"
	"net/url"
	"runtime"
	"time"

	"github.com/atdiar/particleui"
)


func init(){
	ui.NativeEventBridge = NativeEventBridge
	ui.NativeDispatch = NativeDispatch
}

var (
	
	mainDocument *Document

	// ListenAndServe triggers the construction of the UI tree and startup the App.
	ListenAndServe func()

	// DocumentInitializer is a Document specific modifier that is called on creation of a 
	// new document. By assigning a new value to this global function, we can hook new behaviors
	// into a NewDocument call.
	// That can be useful to pass specific properties to a new document object that will specialize 
	// construction of the document.
	DocumentInitializer func(Document) Document = func(d Document) Document{return d}
)



//var NewID = ui.NewIDgenerator(time.Now().UnixNano())

func InBrowser() bool{
	if runtime.GOARCH== "wasm" && runtime.GOOS == "js"{
		return true
	}
	return false
}

// mutationCaptureMode describes how a Go App may capture textarea value changes
// that happen in native javascript. For instance, when a blur event is dispatched
// or when any mutation is observed via the MutationObserver API.
type mutationCaptureMode int

const (
	onBlur mutationCaptureMode = iota
	onInput
)



func SerializeStateHistory() string{
	d:= GetDocument().AsElement()
	sth,ok:= d.Get("internals","globalstatehistory")
	if !ok{
		return ""
	}
	state:= sth.(ui.List)

	return stringify(state.RawValue())
}

func DeserializeStateHistory(rawstate string) (ui.Value,error){
	state:= ui.NewObject()
	err:= json.Unmarshal([]byte(rawstate),&state)
	if err!= nil{
		return nil,err
	}
	
	return state.Value(), nil
}

func stringify(v interface{}) string {
	res, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(res)
}


// Window is a type that represents a browser window
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

	e.ElementStore = Elements
	e.Parent = e
	e.Native,_ = NewNativeElementIfAbsent("defaultView","window")

	e.Watch("ui", "title", e, windowTitleHandler)

	return e
})



func newWindow(title string, options ...string) Window {
	e:= newWindowConstructor("window", options...)
	e.Set("ui", "title", ui.String(title))
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

func(d Document) OnReady(h *ui.MutationHandler){
	d.AsElement().Watch("navigation","ready",d,h)
}

func(d Document) Delete(){ // TODO check for dangling references
	ui.DoSync(func(){
		e:= d.AsElement()
		ui.CancelNav()
		e.DeleteChildren()
		mainDocument = nil
		Elements.Delete(e.ID)
	})
}

func(d Document) SetTitle(title string){
	d.AsElement().SetDataSetUI("title",ui.String(title))
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

func GetDocument() *Document{
	return mainDocument
}

var newDocument = Elements.NewConstructor("html", func(id string) *ui.Element {

	e := Elements.NewAppRoot(id).AsElement()

	e.Native,_ = NewNativeElementIfAbsent("documentElement", "html")
	SetAttribute(e, "id", id)

	e.Watch("ui","lang",e,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		SetAttribute(evt.Origin(),"lang",string(evt.NewValue().(ui.String)))
		return false
	}).RunASAP())

	

    e.Watch("ui","history",e,historyMutationHandler)

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
	

	e.AppendChild(Head("head"))
	e.AppendChild(Body("body"))

	recoverStateHistory()

	e.Watch("navigation", "ready", e,navreadyHandler)
	e.Watch("runtime","recoverablestatehistory",e,recoverStateHistoryHandler)
	e.Watch("ui", "title", e, documentTitleHandler)
	return e
}, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence, AllowScrollRestoration)

var documentTitleHandler= ui.NewMutationHandler(func(evt ui.MutationEvent) bool { 
	d:= Document{ui.BasicElement{evt.Origin()}}
	t:= Title.WithID("documenttitle")()
	t.Set(string(evt.NewValue().(ui.String)))
	d.Head().AppendChild(t)

	return false
})

func replayStateHistory(e *ui.Element) {
	rh,ok:= e.Get("internals","globalstatehistory")
	if !ok{
		panic("somehow recovering state failed. Unexpected error")
	}
	history, ok:= rh.(ui.List)
	if !ok{
		panic("state history should have been a ui.List. Wrong type. Unexpected error")
	}
	for _,rawop:= range history{
		op:= rawop.(ui.Object)
		elementid:= string(op.MustGetString("id"))
		category:= string(op.MustGetString("cat"))
		propname:= string(op.MustGetString("prop"))
		value,_:= op.Get("val")
		el:= Elements.GetByID(elementid)
		if el == nil{
			panic("Unable to recover state for this element id. Element  doesn't exist")
		}
		el.Set(category,propname,value)
	}
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



// NewDocument returns the root of new js app. It is the top-most element
// in the tree of Elements that consitute the full document.
// Options such as the location of persisted data can be passed to the constructor of an instance.
func NewDocument(id string, options ...string) Document {
	d:= Document{ui.BasicElement{LoadFromStorage(newDocument(id, options...))}}
	d = DocumentInitializer(d)
	mainDocument = &d
	return d
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

	tag:= "body"
	var exist bool
	e.Native, exist= NewNativeElementIfAbsent(id, tag)
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



// Head refers to the <head> HTML element of a HTML document, which contains metadata and links to 
// resources such as title, scripts, stylesheets.
type HeadElement struct{
	ui.BasicElement
}

var newHead = Elements.NewConstructor("head",func(id string)*ui.Element{
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	tag:= "head"
	var exist bool
	e.Native, exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		evt.Origin().Root().Set("ui","head",evt.Origin())
		return false
	}))

	return e
})


var Head = headConstructor(func (options ...string) HeadElement {
	return HeadElement{ui.BasicElement{LoadFromStorage(newHead(Elements.NewID(), options...))}}
})

type headConstructor func(...string) HeadElement
func(c headConstructor) WithID(id string) func(options ...string)HeadElement{
	return func(options ...string) HeadElement {
		return HeadElement{ui.BasicElement{LoadFromStorage(newHead(id, options...))}}
	}
}

// Meta : for definition and examples, see https://developer.mozilla.org/en-US/docs/Web/HTML/Element/meta
type MetaElement struct{
	ui.BasicElement
}

func(m MetaElement) SetAttribute(name,value string) MetaElement{
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

	tag:= "meta"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	return e
})


var Meta = metaConstructor(func (options ...string) MetaElement {
	return MetaElement{ui.BasicElement{LoadFromStorage(newMeta(Elements.NewID(), options...))}}
})

type metaConstructor func(...string) MetaElement
func(c metaConstructor) WithID(id string) func(options ...string)MetaElement{
	return func(options ...string) MetaElement {
		return MetaElement{ui.BasicElement{LoadFromStorage(newMeta(id, options...))}}
	}
}


type TitleElement struct{
	ui.BasicElement
}

func(m TitleElement) Set(title string) TitleElement{
	m.AsElement().SetDataSetUI("title",ui.String(title))
	return m
}

var newTitle = Elements.NewConstructor("title",func(id string)*ui.Element{
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	tag:= "title"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}
	e.Watch("ui","title",e,titleElementChangeHandler)

	return e
})


var Title = titleConstructor(func (options ...string) TitleElement {
	return TitleElement{ui.BasicElement{LoadFromStorage(newTitle(Elements.NewID(), options...))}}
})

type titleConstructor func(...string) TitleElement
func(c titleConstructor) WithID(id string) func(options ...string)TitleElement{
	return func(options ...string) TitleElement {
		return TitleElement{ui.BasicElement{LoadFromStorage(newTitle(id, options...))}}
	}
}

// ScriptElement is an ELement that refers to the HTML ELement of the same name that embeds executable 
// code or data.
type ScriptElement struct{
	ui.BasicElement
}

func(s ScriptElement) Src(source string) ScriptElement{
	SetAttribute(s.AsElement(),"src",source)
	return s
}

func(s ScriptElement) Type(typ string) ScriptElement{
	SetAttribute(s.AsElement(),"type",typ)
	return s
}

func(s ScriptElement) Async() ScriptElement{
	SetAttribute(s.AsElement(),"async","")
	return s
}

func(s ScriptElement) Defer() ScriptElement{
	SetAttribute(s.AsElement(),"defer","")
	return s
}

func(s ScriptElement) SetInnerHTML(content string) ScriptElement{
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

	tag:= "script"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	return e
})


var Script = scriptConstructor(func (options ...string) ScriptElement {
	return ScriptElement{ui.BasicElement{LoadFromStorage(newScript(Elements.NewID(), options...))}}
})

type scriptConstructor func(...string) ScriptElement
func(c scriptConstructor) WithID(id string) func(options ...string)ScriptElement{
	return func(options ...string) ScriptElement {
		return ScriptElement{ui.BasicElement{LoadFromStorage(newScript(id, options...))}}
	}
}


// BaseElement allows to define the baseurl or the basepath for the links within a page.
// In our current use-case, it will mostly be used when generating HTML (SSR or SSG).
// It is then mostly a build-time concern.
type BaseElement struct{
	ui.BasicElement
}

func(b BaseElement) SetHREF(url string) BaseElement{
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

	tag:= "base"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	e.Watch("ui","href",e,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		SetAttribute(evt.Origin(),"href",string(evt.NewValue().(ui.String)))
		return false
	}))

	return e
})

var Base = baseConstructor(func (options ...string) BaseElement {
	return BaseElement{ui.BasicElement{LoadFromStorage(newBase(Elements.NewID(), options...))}}
})

type baseConstructor func(...string) BaseElement
func(c baseConstructor) WithID(id string) func(options ...string)BaseElement{
	return func(options ...string) BaseElement {
		return BaseElement{ui.BasicElement{LoadFromStorage(newBase(id, options...))}}
	}
}


// NoScriptElement refers to an element that defines a section of HTMNL to be inserted in a page if a script
// type is unsupported on the page of scripting is turned off.
// As such, this is mostly useful during SSR or SSG, for examplt to display a message if javascript
// is disabled.
// Indeed, if scripts are disbaled, wasm will not be able to insert this dynamically into the page.
type NoScriptElement struct{
	ui.BasicElement
}

func(s NoScriptElement) SetInnerHTML(content string) NoScriptElement{
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

	tag:= "noscript"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	return e
})

var NoScript = noscriptConstructor(func (options ...string) NoScriptElement {
	return NoScriptElement{ui.BasicElement{LoadFromStorage(newNoScript(Elements.NewID(), options...))}}
})

type noscriptConstructor func(...string) NoScriptElement
func(c noscriptConstructor) WithID(id string) func(options ...string)NoScriptElement{
	return func(options ...string) NoScriptElement {
		return NoScriptElement{ui.BasicElement{LoadFromStorage(newNoScript(id, options...))}}
	}
}

// Link refers to the <link> HTML Element which allow to specify the location of external resources
// such as stylesheets or a favicon.
type LinkElement struct{
	ui.BasicElement
}

func(l LinkElement) SetAttribute(name,value string) LinkElement{
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

	tag:= "link"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	return e
})

var Link = linkConstructor(func (options ...string) LinkElement {
	return LinkElement{ui.BasicElement{LoadFromStorage(newLink(Elements.NewID(), options...))}}
})

type linkConstructor func(...string) LinkElement
func(c linkConstructor) WithID(id string) func(options ...string)LinkElement{
	return func(options ...string) LinkElement {
		return LinkElement{ui.BasicElement{LoadFromStorage(newLink(id, options...))}}
	}
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

	
	tag:= "div"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
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
var Div = divConstructor(func (options ...string) DivElement {
	return DivElement{ui.BasicElement{LoadFromStorage(newDiv(Elements.NewID(), options...))}}
})

type divConstructor func(...string) DivElement
func(d divConstructor) WithID(id string) func(options ...string)DivElement{
	return func(options ...string) DivElement {
		return DivElement{ui.BasicElement{LoadFromStorage(newDiv(id, options...))}}
	}
}
// var test = Div.WithID("test")(EnableLocalPersistence())

const SSRStateSuffix = "-ssr-state"
const HydrationAttrName = "data-needh2o"



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

func(t textAreaModifer) Value(text string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("value", ui.String(text))
		return e
	}
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

	tag:= "textarea"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	withStringAttributeWatcher(e,"value")

	withNumberAttributeWatcher(e,"rows")
	withNumberAttributeWatcher(e,"cols")

	withStringAttributeWatcher(e,"wrap")

	withBoolAttributeWatcher(e,"disabled")
	withBoolAttributeWatcher(e,"required")
	withStringAttributeWatcher(e,"name")
	withBoolAttributeWatcher(e,"readonly")
	withStringAttributeWatcher(e,"autocomplete")
	withStringAttributeWatcher(e,"spellcheck")


	return e
}, allowTextAreaDataBindingOnBlur, allowTextAreaDataBindingOnInput, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)


// TextArea is a constructor for a textarea html element.
var TextArea = textareaConstructor(func (options ...string) TextAreaElement {
	return TextAreaElement{ui.BasicElement{LoadFromStorage(newTextArea(Elements.NewID(), options...))}}
})

type textareaConstructor func(...string) TextAreaElement
func(c textareaConstructor) WithID(id string) func(options ...string)TextAreaElement{
	return func(options ...string) TextAreaElement {
		return TextAreaElement{ui.BasicElement{LoadFromStorage(newTextArea(id, options...))}}
	}
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

	tag:= "header"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

// Header is a constructor for a html header element.
var Header = headerConstructor(func (options ...string) HeaderElement {
	return HeaderElement{ui.BasicElement{LoadFromStorage(newHeader(Elements.NewID(), options...))}}
})

type headerConstructor func(...string) HeaderElement
func(c headerConstructor) WithID(id string) func(options ...string)HeaderElement{
	return func(options ...string) HeaderElement {
		return HeaderElement{ui.BasicElement{LoadFromStorage(newHeader(id, options...))}}
	}
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

	tag:= "footer"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

// Footer is a constructor for an html footer element.
var Footer = footerConstructor(func (options ...string) FooterElement {
	return FooterElement{ui.BasicElement{LoadFromStorage(newFooter(Elements.NewID(), options...))}}
})

type footerConstructor func(...string) FooterElement
func(c footerConstructor) WithID(id string) func(options ...string)FooterElement{
	return func(options ...string) FooterElement {
		return FooterElement{ui.BasicElement{LoadFromStorage(newFooter(id, options...))}}
	}
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

	tag:= "section"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

// Section is a constructor for html section elements.
var Section = sectionConstructor(func (options ...string) SectionElement {
	return SectionElement{ui.BasicElement{LoadFromStorage(newSection(Elements.NewID(), options...))}}
})

type sectionConstructor func(...string) SectionElement
func(c sectionConstructor) WithID(id string) func(options ...string)SectionElement{
	return func(options ...string) SectionElement {
		return SectionElement{ui.BasicElement{LoadFromStorage(newSection(id, options...))}}
	}
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

	tag:= "h1"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	e.Watch("ui", "text", e, textContentHandler)
	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

// H1 is a constructor for html heading H1 elements.
var H1 = h1Constructor(func (options ...string) H1Element {
	return H1Element{ui.BasicElement{LoadFromStorage(newH1(Elements.NewID(), options...))}}
})

type h1Constructor func(...string) H1Element
func(c h1Constructor) WithID(id string) func(options ...string)H1Element{
	return func(options ...string) H1Element {
		return H1Element{ui.BasicElement{LoadFromStorage(newH1(id, options...))}}
	}
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

	tag:= "h2"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	e.Watch("ui", "text", e,textContentHandler)
	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

// H2 is a constructor for html heading H2 elements.
var H2 = h2Constructor(func (options ...string) H2Element {
	return H2Element{ui.BasicElement{LoadFromStorage(newH2(Elements.NewID(), options...))}}
})

type h2Constructor func(...string) H2Element
func(c h2Constructor) WithID(id string) func(options ...string)H2Element{
	return func(options ...string) H2Element {
		return H2Element{ui.BasicElement{LoadFromStorage(newH2(id, options...))}}
	}
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

	tag:= "h3"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	e.Watch("ui", "text", e,textContentHandler)
	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

// H3 is a constructor for html heading H3 elements.
var H3 = h3Constructor(func (options ...string) H3Element {
	return H3Element{ui.BasicElement{LoadFromStorage(newH3(Elements.NewID(), options...))}}
})

type h3Constructor func(...string) H3Element
func(c h3Constructor) WithID(id string) func(options ...string)H3Element{
	return func(options ...string) H3Element {
		return H3Element{ui.BasicElement{LoadFromStorage(newH3(id, options...))}}
	}
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

	tag:= "h4"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	e.Watch("ui", "text", e, textContentHandler)
	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

// H4 is a constructor for html heading H4 elements.
var H4 = h4Constructor(func (options ...string) H4Element {
	return H4Element{ui.BasicElement{LoadFromStorage(newH4(Elements.NewID(), options...))}}
})

type h4Constructor func(...string) H4Element
func(c h4Constructor) WithID(id string) func(options ...string)H4Element{
	return func(options ...string) H4Element {
		return H4Element{ui.BasicElement{LoadFromStorage(newH4(id, options...))}}
	}
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

	tag:= "h5"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	e.Watch("ui", "text", e, textContentHandler)
	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

// H5 is a constructor for html heading H5 elements.
var H5 = h5Constructor(func (options ...string) H5Element {
	return H5Element{ui.BasicElement{LoadFromStorage(newH5(Elements.NewID(), options...))}}
})

type h5Constructor func(...string) H5Element
func(c h5Constructor) WithID(id string) func(options ...string)H5Element{
	return func(options ...string) H5Element {
		return H5Element{ui.BasicElement{LoadFromStorage(newH5(id, options...))}}
	}
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

	tag:= "h6"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	e.Watch("ui", "text", e,textContentHandler)
	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)


// H6 is a constructor for html heading H6 elements.
var H6 = h6Constructor(func (options ...string) H6Element {
	return H6Element{ui.BasicElement{LoadFromStorage(newH6(Elements.NewID(), options...))}}
})

type h6Constructor func(...string) H6Element
func(c h6Constructor) WithID(id string) func(options ...string)H6Element{
	return func(options ...string) H6Element {
		return H6Element{ui.BasicElement{LoadFromStorage(newH6(id, options...))}}
	}
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

	tag:= "span"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	e.Watch("ui", "text", e, textContentHandler)

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

// Span is a constructor for html span elements.
var Span = spanConstructor(func (options ...string) SpanElement {
	return SpanElement{ui.BasicElement{LoadFromStorage(newSpan(Elements.NewID(), options...))}}
})

type spanConstructor func(...string) SpanElement
func(c spanConstructor) WithID(id string) func(options ...string)SpanElement{
	return func(options ...string) SpanElement {
		return SpanElement{ui.BasicElement{LoadFromStorage(newSpan(id, options...))}}
	}
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

	tag:= "article"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}


	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var Article = articleConstructor(func (options ...string) ArticleElement {
	return ArticleElement{ui.BasicElement{LoadFromStorage(newArticle(Elements.NewID(), options...))}}
})

type articleConstructor func(...string) ArticleElement
func(c articleConstructor) WithID(id string) func(options ...string)ArticleElement{
	return func(options ...string) ArticleElement {
		return ArticleElement{ui.BasicElement{LoadFromStorage(newArticle(id, options...))}}
	}
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

	tag:= "aside"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var Aside = asideConstructor(func (options ...string) AsideElement {
	return AsideElement{ui.BasicElement{LoadFromStorage(newAside(Elements.NewID(), options...))}}
})

type asideConstructor func(...string) AsideElement
func(c asideConstructor) WithID(id string) func(options ...string)AsideElement{
	return func(options ...string) AsideElement {
		return AsideElement{ui.BasicElement{LoadFromStorage(newAside(id, options...))}}
	}
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

	tag:= "main"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var Main = mainConstructor(func (options ...string) MainElement {
	return MainElement{ui.BasicElement{LoadFromStorage(newMain(Elements.NewID(), options...))}}
})

type mainConstructor func(...string) MainElement
func(c mainConstructor) WithID(id string) func(options ...string)MainElement{
	return func(options ...string) MainElement {
		return MainElement{ui.BasicElement{LoadFromStorage(newMain(id, options...))}}
	}
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

	tag:= "p"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	e.Watch("ui", "text", e, paragraphTextHandler)
	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

// Paragraph is a constructor for html paragraph elements.
var Paragraph = paragraphConstructor(func (options ...string) ParagraphElement {
	return ParagraphElement{ui.BasicElement{LoadFromStorage(newParagraph(Elements.NewID(), options...))}}
})

type paragraphConstructor func(...string) ParagraphElement
func(c paragraphConstructor) WithID(id string) func(options ...string)ParagraphElement{
	return func(options ...string) ParagraphElement {
		return ParagraphElement{ui.BasicElement{LoadFromStorage(newParagraph(id, options...))}}
	}
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

		tag:= "nav"
		var exist bool
		e.Native,exist = NewNativeElementIfAbsent(id, tag)
		if !exist {
			SetAttribute(e, "id", id)
		}

		return e
	}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

// Nav is a constructor for a html nav element.
var Nav = navConstructor(func (options ...string) NavElement {
	return NavElement{ui.BasicElement{LoadFromStorage(newNav(Elements.NewID(), options...))}}
})

type navConstructor func(...string) NavElement
func(c navConstructor) WithID(id string) func(options ...string)NavElement{
	return func(options ...string) NavElement {
		return NavElement{ui.BasicElement{LoadFromStorage(newNav(id, options...))}}
	}
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

	tag:= "a"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	withStringAttributeWatcher(e,"href")

	withStringPropertyWatcher(e,"text")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence, AllowPrefetchOnIntent, AllowPrefetchOnRender)

// Anchor creates an html anchor element.
var Anchor = anchorConstructor(func (options ...string) AnchorElement {
	return AnchorElement{ui.BasicElement{LoadFromStorage(newAnchor(Elements.NewID(), options...))}}
})

type anchorConstructor func(...string) AnchorElement
func(c anchorConstructor) WithID(id string) func(options ...string)AnchorElement{
	return func(options ...string) AnchorElement {
		return AnchorElement{ui.BasicElement{LoadFromStorage(newAnchor(id, options...))}}
	}
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

func(m buttonModifer) Text(str string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("text", ui.String(str))
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

	tag:= "button"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	withBoolAttributeWatcher(e,"disabled")
	withBoolAttributeWatcher(e,"autofocus")

	withStringAttributeWatcher(e,"form")
	withStringAttributeWatcher(e,"type")
	withStringAttributeWatcher(e,"name")

	e.Watch("ui", "text", e, textContentHandler)

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

// Button returns a button ui.BasicElement.
// TODO (add attribute watchers for form button element)
var Button = buttonConstructor(func (options ...string) ButtonElement {
	return ButtonElement{ui.BasicElement{LoadFromStorage(newButton(Elements.NewID(), options...))}}
})

type buttonConstructor func(...string) ButtonElement
func(c buttonConstructor) WithID(id string) func(options ...string)ButtonElement{
	return func(options ...string) ButtonElement {
		return ButtonElement{ui.BasicElement{LoadFromStorage(newButton(id, options...))}}
	}
}

type LabelElement struct {
	ui.BasicElement
}

type labelModifier struct{}
var LabelModifier labelModifier

func(m labelModifier) Text(str string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("text", ui.String(str))
		return e
	}
}

func(m labelModifier) For(e *ui.Element) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			d:= GetDocument()
			
			evt.Origin().Watch("event","navigationend",d,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
				if e.Mounted(){
					e.SetDataSetUI("for", ui.String(e.ID))
				} else{
					DEBUG("label for attributes couldb't be set") // panic instead?
				}
				return false
			}).RunOnce())
			return false
		}).RunOnce())
		return e
	}
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

	tag:= "label"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	withStringAttributeWatcher(e,"for")
	e.Watch("ui", "text", e, textContentHandler)
	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var Label = labelConstructor(func (options ...string) LabelElement {
	return LabelElement{ui.BasicElement{LoadFromStorage(newLabel(Elements.NewID(), options...))}}
})

type labelConstructor func(...string) LabelElement
func(c labelConstructor) WithID(id string) func(options ...string)LabelElement{
	return func(options ...string) LabelElement {
		return LabelElement{ui.BasicElement{LoadFromStorage(newLabel(id, options...))}}
	}
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


var newInput= Elements.NewConstructor("input", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	tag:= "input"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	withStringPropertyWatcher(e,"value")

	
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


var Input = inputConstructor(func (typ string,options ...string) InputElement {
	if typ != ""{
		options = append(options, typ)
	}
	return InputElement{ui.BasicElement{LoadFromStorage(newInput(Elements.NewID(), options...))}}
})

type inputConstructor func(typ string, options ...string) InputElement
func(c inputConstructor) WithID(id string) func(typ string, options ...string)InputElement{
	return func(typ string, options ...string) InputElement {
		if typ != ""{
			options = append(options, typ)
		}
		return InputElement{ui.BasicElement{LoadFromStorage(newInput(id, options...))}}
	}
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

	tag:= "output"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions
	withStringAttributeWatcher(e,"form")
	withStringAttributeWatcher(e,"name")
	withBoolAttributeWatcher(e,"disabled")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var Output = outputConstructor(func (options ...string) OutputElement {
	return OutputElement{ui.BasicElement{LoadFromStorage(newOutput(Elements.NewID(), options...))}}
})

type outputConstructor func(...string) OutputElement
func(c outputConstructor) WithID(id string) func(options ...string)OutputElement{
	return func(options ...string) OutputElement {
		return OutputElement{ui.BasicElement{LoadFromStorage(newOutput(id, options...))}}
	}
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

var newImg= Elements.NewConstructor("img", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	tag:= "img"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	withStringAttributeWatcher(e,"src")
	withStringAttributeWatcher(e,"alt")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var Img = imgConstructor(func (options ...string) ImgElement {
	return ImgElement{ui.BasicElement{LoadFromStorage(newImg(Elements.NewID(), options...))}}
})

type imgConstructor func(...string) ImgElement
func(c imgConstructor) WithID(id string) func(options ...string)ImgElement{
	return func(options ...string) ImgElement {
		return ImgElement{ui.BasicElement{LoadFromStorage(newImg(id, options...))}}
	}
}



type jsTimeRanges ui.Object

func(j jsTimeRanges) Start(index int) time.Duration{
	ti,ok:= ui.Object(j).Get("start")
	if !ok{
		panic("Bad timeRange encoding. No start found")
	}
	ranges := ti.(ui.List)
	if index >= len(ranges){
		panic("no time ramge at index, index out of bounds")
	}
	return time.Duration(ranges[index].(ui.Number)) *time.Second
}

func(j jsTimeRanges) End(index int) time.Duration{
	ti,ok:= ui.Object(j).Get("end")
	if !ok{
		panic("Bad timeRange encoding. No start found")
	}
	ranges := ti.(ui.List)
	if index >= len(ranges){
		panic("no time ramge at index, index out of bounds")
	}
	return time.Duration(ranges[index].(ui.Number)) *time.Second
}

func(j jsTimeRanges) Length() int{
	l,ok:= ui.Object(j).Get("length")
	if !ok{
		panic("bad timerange encoding")
	}
	return int(l.(ui.Number))
}


// AudioElement
type AudioElement struct{
	ui.BasicElement
}

type audioModifier struct{}
var AudioModifier audioModifier

func(m audioModifier) Autoplay(b bool)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("autoplay", ui.Bool(b))
		return e
	}
}

func(m audioModifier) Controls(b bool)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("controls", ui.Bool(b))
		return e
	}
}

func(m audioModifier) CrossOrigin(option string)func(*ui.Element)*ui.Element{
	mod:= ui.String("anonymous")
	if option == "use-credentials"{
		mod = ui.String(option)
	}
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("crossorigin", mod)
		return e
	}
}

func(m audioModifier) Loop(b bool)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("loop", ui.Bool(b))
		return e
	}
}

func(m audioModifier) Muted(b bool)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("muted", ui.Bool(b))
		return e
	}
}

func(m audioModifier) Preload(option string)func(*ui.Element)*ui.Element{
	mod:= ui.String("metadata")
	switch option{
	case "none":
		mod = ui.String(option)
	case "auto":
		mod = ui.String(option)
	case "":
		mod = ui.String("auto")
	}
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("preload", mod)
		return e
	}
}

func(m audioModifier) Src(src string)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("src", ui.String(src))
		return e
	}
}

func(m audioModifier) CurrentTime(t float64)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("currentTime", ui.Number(t))
		return e
	}
}


func(m audioModifier) PlayBackRate(r float64)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("playbackRate", ui.Number(r))
		return e
	}
}

func(m audioModifier) Volume(v float64)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("volume", ui.Number(v))
		return e
	}
}

func(m audioModifier) PreservesPitch(b bool)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("preservesPitch", ui.Bool(b))
		return e
	}
}

func(m audioModifier) DisableRemotePlayback(b bool)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("disableRemotePlayback", ui.Bool(b))
		return e
	}
}

var newAudio = Elements.NewConstructor("audio", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	tag:= "audio"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	withStringAttributeWatcher(e,"src")
	withStringAttributeWatcher(e,"preload")
	withBoolAttributeWatcher(e,"muted")
	withBoolAttributeWatcher(e,"loop")
	withStringAttributeWatcher(e,"crossorigin")
	withBoolAttributeWatcher(e,"controls")
	withBoolAttributeWatcher(e,"autoplay")

	withMediaElementPropertyWatchers(e)

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var Audio = audioConstructor(func (options ...string) AudioElement {
	return AudioElement{ui.BasicElement{LoadFromStorage(newAudio(Elements.NewID(), options...))}}
})

type audioConstructor func(...string) AudioElement
func(c audioConstructor) WithID(id string) func(options ...string)AudioElement{
	return func(options ...string) AudioElement {
		return AudioElement{ui.BasicElement{LoadFromStorage(newAudio(id, options...))}}
	}
}

// VideoElement
type VideoElement struct{
	ui.BasicElement
}

type videoModifier struct{}
var VideoModifier videoModifier

func(m videoModifier) Height(h float64) func(*ui.Element)*ui.Element{
	return func(e *ui.Element) *ui.Element{
		e.SetDataSetUI("height", ui.Number(h))
		return e
	}
}

func(m videoModifier) Width(w float64) func(*ui.Element)*ui.Element{
	return func(e *ui.Element) *ui.Element{
		e.SetDataSetUI("width", ui.Number(w))
		return e
	}
}

func(m videoModifier) Poster(url string)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("poster", ui.String(url))
		return e
	}
}


func(m videoModifier) PlaysInline(b bool)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("playsinline", ui.Bool(b))
		return e
	}
}

func(m videoModifier) Controls(b bool)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("controls", ui.Bool(b))
		return e
	}
}

func(m videoModifier) CrossOrigin(option string)func(*ui.Element)*ui.Element{
	mod:= ui.String("anonymous")
	if option == "use-credentials"{
		mod = ui.String(option)
	}
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("crossorigin", mod)
		return e
	}
}

func(m videoModifier) Loop(b bool)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("loop", ui.Bool(b))
		return e
	}
}

func(m videoModifier) Muted(b bool)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("muted", ui.Bool(b))
		return e
	}
}

func(m videoModifier) Preload(option string)func(*ui.Element)*ui.Element{
	mod:= ui.String("metadata")
	switch option{
	case "none":
		mod = ui.String(option)
	case "auto":
		mod = ui.String(option)
	case "":
		mod = ui.String("auto")
	}
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("preload", mod)
		return e
	}
}

func(m videoModifier) Src(src string)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("src", ui.String(src))
		return e
	}
}

func(m videoModifier) CurrentTime(t float64)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("currentTime", ui.Number(t))
		return e
	}
}

func(m videoModifier) DefaultPlayBackRate(r float64)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("defaultPlaybackRate", ui.Number(r))
		return e
	}
}

func(m videoModifier) PlayBackRate(r float64)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("playbackRate", ui.Number(r))
		return e
	}
}

func(m videoModifier) Volume(v float64)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("volume", ui.Number(v))
		return e
	}
}

func(m videoModifier) PreservesPitch(b bool)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetDataSetUI("preservesPitch", ui.Bool(b))
		return e
	}
}

var newVideo = Elements.NewConstructor("video", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	tag:= "video"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	withNumberAttributeWatcher(e,"width")
	withNumberAttributeWatcher(e,"height")
	withStringAttributeWatcher(e,"src")
	withStringAttributeWatcher(e,"preload")
	withStringAttributeWatcher(e,"poster")
	withBoolAttributeWatcher(e,"playsinline")
	withBoolAttributeWatcher(e,"muted")
	withBoolAttributeWatcher(e,"loop")
	withStringAttributeWatcher(e,"crossorigin")
	withBoolAttributeWatcher(e,"controls")

	withMediaElementPropertyWatchers(e)

	SetAttribute(e, "id", id)

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var Video = videoConstructor(func (options ...string) VideoElement {
	return VideoElement{ui.BasicElement{LoadFromStorage(newVideo(Elements.NewID(), options...))}}
})

type videoConstructor func(...string) VideoElement
func(c videoConstructor) WithID(id string) func(options ...string)VideoElement{
	return func(options ...string) VideoElement {
		return VideoElement{ui.BasicElement{LoadFromStorage(newVideo(id, options...))}}
	}
}

// SourceElement
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

	tag:= "source"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	withStringAttributeWatcher(e,"src")
	withStringAttributeWatcher(e,"type")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var Source = sourceConstructor(func (options ...string) SourceElement {
	return SourceElement{ui.BasicElement{LoadFromStorage(newSource(Elements.NewID(), options...))}}
})

type sourceConstructor func(...string) SourceElement
func(c sourceConstructor) WithID(id string) func(options ...string)SourceElement{
	return func(options ...string) SourceElement {
		return SourceElement{ui.BasicElement{LoadFromStorage(newSource(id, options...))}}
	}
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

	tag:= "ul"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

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

var Ul = ulConstructor(func (options ...string) UlElement {
	return UlElement{ui.BasicElement{LoadFromStorage(newUl(Elements.NewID(), options...))}}
})

type ulConstructor func(...string) UlElement
func(c ulConstructor) WithID(id string) func(options ...string)UlElement{
	return func(options ...string) UlElement {
		return UlElement{ui.BasicElement{LoadFromStorage(newUl(id, options...))}}
	}
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

	tag:= "ol"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}
	
	return e
}, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)


var Ol = olConstructor(func (typ string, offset int, options ...string) OlElement {
	e:= newOl(Elements.NewID(), options...)
	SetAttribute(e, "type", typ)
	SetAttribute(e, "start", strconv.Itoa(offset))
	return OlElement{ui.BasicElement{LoadFromStorage(e)}}
})

type olConstructor func(typ string, offset int, options ...string) OlElement
func(c olConstructor) WithID(id string) func(typ string, offset int, options ...string)OlElement{
	return func(typ string, offset int, options ...string) OlElement {
		e:= newOl(id, options...)
		SetAttribute(e, "type", typ)
		SetAttribute(e, "start", strconv.Itoa(offset))
		return OlElement{ui.BasicElement{LoadFromStorage(e)}}
	}
}

type LiElement struct {
	ui.BasicElement
}

func (li LiElement) SetValue(v ui.Value) LiElement {
	li.AsElement().SetDataSetUI("value", v)
	return li
}

var newLi= Elements.NewConstructor("li", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	tag:= "li"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	onuimutation := ui.NewMutationHandler(func(evt ui.MutationEvent) bool {

		// we apply the modifications to the UI
		v := evt.NewValue()
		var item *ui.Element
		switch t := v.(type) {
		case ui.String:
			item = NewTextNode().SetValue(t).Element()
		case ui.Bool:
			item = NewTextNode().Element()
			item.SetDataSetUI("text", t)
		case ui.Number:
			item = NewTextNode().Element()
			item.SetDataSetUI("text", t)
		case ui.Object:
			item = NewTextNode().Element()
			item.SetDataSetUI("text", t)
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

var Li = liConstructor(func (options ...string) LiElement {
	return LiElement{ui.BasicElement{LoadFromStorage(newLi(Elements.NewID(), options...))}}
})

type liConstructor func(...string) LiElement
func(c liConstructor) WithID(id string) func(options ...string)LiElement{
	return func(options ...string) LiElement {
		return LiElement{ui.BasicElement{LoadFromStorage(newLi(id, options...))}}
	}
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

	tag:= "thead"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var Thead = theadConstructor(func (options ...string) TheadElement {
	return TheadElement{ui.BasicElement{LoadFromStorage(newThead(Elements.NewID(), options...))}}
})

type theadConstructor func(...string) TheadElement
func(c theadConstructor) WithID(id string) func(options ...string)TheadElement{
	return func(options ...string) TheadElement {
		return TheadElement{ui.BasicElement{LoadFromStorage(newThead(id, options...))}}
	}
}


var newTr= Elements.NewConstructor("tr", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	tag:= "tr"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var Tr = trConstructor(func (options ...string) TrElement {
	return TrElement{ui.BasicElement{LoadFromStorage(newTr(Elements.NewID(), options...))}}
})

type trConstructor func(...string) TrElement
func(c trConstructor) WithID(id string) func(options ...string)TrElement{
	return func(options ...string) TrElement {
		return TrElement{ui.BasicElement{LoadFromStorage(newTr(id, options...))}}
	}
}

var newTd= Elements.NewConstructor("td", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	tag:= "td"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var Td = tdConstructor(func (options ...string) TdElement {
	return TdElement{ui.BasicElement{LoadFromStorage(newTd(Elements.NewID(), options...))}}
})

type tdConstructor func(...string) TdElement
func(c tdConstructor) WithID(id string) func(options ...string)TdElement{
	return func(options ...string) TdElement {
		return TdElement{ui.BasicElement{LoadFromStorage(newTd(id, options...))}}
	}
}

var newTh= Elements.NewConstructor("th", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	tag:= "th"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var Th = thConstructor(func (options ...string) ThElement {
	return ThElement{ui.BasicElement{LoadFromStorage(newTh(Elements.NewID(), options...))}}
})

type thConstructor func(...string) ThElement
func(c thConstructor) WithID(id string) func(options ...string)ThElement{
	return func(options ...string) ThElement {
		return ThElement{ui.BasicElement{LoadFromStorage(newTh(id, options...))}}
	}
}

var newTbody= Elements.NewConstructor("tbody", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	tag:= "tbody"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var Tbody = tbodyConstructor(func (options ...string) TbodyElement {
	return TbodyElement{ui.BasicElement{LoadFromStorage(newTbody(Elements.NewID(), options...))}}
})

type tbodyConstructor func(...string) TbodyElement
func(c tbodyConstructor) WithID(id string) func(options ...string)TbodyElement{
	return func(options ...string) TbodyElement {
		return TbodyElement{ui.BasicElement{LoadFromStorage(newTbody(id, options...))}}
	}
}

var newTfoot= Elements.NewConstructor("tfoot", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	tag:= "tfoot"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var Tfoot = tfootConstructor(func (options ...string) TfootElement {
	return TfootElement{ui.BasicElement{LoadFromStorage(newTfoot(Elements.NewID(), options...))}}
})

type tfootConstructor func(...string) TfootElement
func(c tfootConstructor) WithID(id string) func(options ...string)TfootElement{
	return func(options ...string) TfootElement {
		return TfootElement{ui.BasicElement{LoadFromStorage(newTfoot(id, options...))}}
	}
}

var newCol= Elements.NewConstructor("col", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	tag:= "col"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	withNumberAttributeWatcher(e,"span")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var Col = colConstructor(func (options ...string) ColElement {
	return ColElement{ui.BasicElement{LoadFromStorage(newCol(Elements.NewID(), options...))}}
})

type colConstructor func(...string) ColElement
func(c colConstructor) WithID(id string) func(options ...string)ColElement{
	return func(options ...string) ColElement {
		return ColElement{ui.BasicElement{LoadFromStorage(newCol(id, options...))}}
	}
}

var newColGroup= Elements.NewConstructor("colgroup", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	tag:= "colgroup"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	withNumberAttributeWatcher(e,"span")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var ColGroup = colgroupConstructor(func (options ...string) ColGroupElement {
	return ColGroupElement{ui.BasicElement{LoadFromStorage(newColGroup(Elements.NewID(), options...))}}
})

type colgroupConstructor func(...string) ColGroupElement
func(c colgroupConstructor) WithID(id string) func(options ...string)ColGroupElement{
	return func(options ...string) ColGroupElement {
		return ColGroupElement{ui.BasicElement{LoadFromStorage(newColGroup(id, options...))}}
	}
}

var newTable= Elements.NewConstructor("table", func(id string) *ui.Element {
	e:= Elements.GetByID(id)
	if e!= nil{
		panic(id + " : this id is already in use")
	}
	e = ui.NewElement(id, Elements.DocType)
	e = enableClasses(e)

	tag:= "table"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var Table = tableConstructor(func (options ...string) TableElement {
	return TableElement{ui.BasicElement{LoadFromStorage(newTable(Elements.NewID(), options...))}}
})

type tableConstructor func(...string) TableElement
func(c tableConstructor) WithID(id string) func(options ...string)TableElement{
	return func(options ...string) TableElement {
		return TableElement{ui.BasicElement{LoadFromStorage(newTable(id, options...))}}
	}
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

	tag:= "canvas"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	withNumberAttributeWatcher(e,"height")
	withNumberAttributeWatcher(e,"width")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var Canvas = canvasConstructor(func (options ...string) CanvasElement {
	return CanvasElement{ui.BasicElement{LoadFromStorage(newCanvas(Elements.NewID(), options...))}}
})

type canvasConstructor func(...string) CanvasElement
func(c canvasConstructor) WithID(id string) func(options ...string)CanvasElement{
	return func(options ...string) CanvasElement {
		return CanvasElement{ui.BasicElement{LoadFromStorage(newCanvas(id, options...))}}
	}
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

	tag:= "svg"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	withNumberAttributeWatcher(e,"height")
	withNumberAttributeWatcher(e,"width")
	withStringAttributeWatcher(e,"viewbox")
	withStringAttributeWatcher(e,"x")
	withStringAttributeWatcher(e,"y")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var Svg = svgConstructor(func (options ...string) SvgElement {
	return SvgElement{ui.BasicElement{LoadFromStorage(newSvg(Elements.NewID(), options...))}}
})

type svgConstructor func(...string) SvgElement
func(c svgConstructor) WithID(id string) func(options ...string)SvgElement{
	return func(options ...string) SvgElement {
		return SvgElement{ui.BasicElement{LoadFromStorage(newSvg(id, options...))}}
	}
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

	tag:= "summary"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	e.Watch("ui", "text", e, textContentHandler)

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var Summary = summaryConstructor(func (options ...string) SummaryElement {
	return SummaryElement{ui.BasicElement{LoadFromStorage(newSummary(Elements.NewID(), options...))}}
})

type summaryConstructor func(...string) SummaryElement
func(c summaryConstructor) WithID(id string) func(options ...string)SummaryElement{
	return func(options ...string) SummaryElement {
		return SummaryElement{ui.BasicElement{LoadFromStorage(newSummary(id, options...))}}
	}
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

	tag:= "details"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	e.Watch("ui", "text", e, textContentHandler)
	withBoolAttributeWatcher(e,"open")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var Details = detailsConstructor(func (options ...string) DetailsElement {
	return DetailsElement{ui.BasicElement{LoadFromStorage(newDetails(Elements.NewID(), options...))}}
})

type detailsConstructor func(...string) DetailsElement
func(c detailsConstructor) WithID(id string) func(options ...string)DetailsElement{
	return func(options ...string) DetailsElement {
		return DetailsElement{ui.BasicElement{LoadFromStorage(newDetails(id, options...))}}
	}
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

	tag:= "dialog"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	withBoolAttributeWatcher(e,"open")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var Dialog = dialogConstructor(func (options ...string) DialogElement {
	return DialogElement{ui.BasicElement{LoadFromStorage(newDialog(Elements.NewID(), options...))}}
})

type dialogConstructor func(...string) DialogElement
func(c dialogConstructor) WithID(id string) func(options ...string)DialogElement{
	return func(options ...string) DialogElement {
		return DialogElement{ui.BasicElement{LoadFromStorage(newDialog(id, options...))}}
	}
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

	tag:= "code"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	e.Watch("ui", "text", e, textContentHandler)

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var Code = codeConstructor(func (options ...string) CodeElement {
	return CodeElement{ui.BasicElement{LoadFromStorage(newCode(Elements.NewID(), options...))}}
})

type codeConstructor func(...string) CodeElement
func(c codeConstructor) WithID(id string) func(options ...string)CodeElement{
	return func(options ...string) CodeElement {
		return CodeElement{ui.BasicElement{LoadFromStorage(newCode(id, options...))}}
	}
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

	tag:= "embed"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	withNumberAttributeWatcher(e,"height")
	withNumberAttributeWatcher(e,"width")
	withStringAttributeWatcher(e,"type")
	withStringAttributeWatcher(e,"src")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var Embed = embedConstructor(func (options ...string) EmbedElement {
	return EmbedElement{ui.BasicElement{LoadFromStorage(newEmbed(Elements.NewID(), options...))}}
})

type embedConstructor func(...string) EmbedElement
func(c embedConstructor) WithID(id string) func(options ...string)EmbedElement{
	return func(options ...string) EmbedElement {
		return EmbedElement{ui.BasicElement{LoadFromStorage(newEmbed(id, options...))}}
	}
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

	tag:= "object"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	withNumberAttributeWatcher(e,"height")
	withNumberAttributeWatcher(e,"width")
	withStringAttributeWatcher(e,"type")
	withStringAttributeWatcher(e,"data")
	withStringAttributeWatcher(e,"form")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var Object = objectConstructor(func (options ...string) ObjectElement {
	return ObjectElement{ui.BasicElement{LoadFromStorage(newObject(Elements.NewID(), options...))}}
})

type objectConstructor func(...string) ObjectElement
func(c objectConstructor) WithID(id string) func(options ...string)ObjectElement{
	return func(options ...string) ObjectElement {
		return ObjectElement{ui.BasicElement{LoadFromStorage(newObject(id, options...))}}
	}
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

	tag:= "datalist"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var Datalist = datalistConstructor(func (options ...string) DatalistElement {
	return DatalistElement{ui.BasicElement{LoadFromStorage(newDatalist(Elements.NewID(), options...))}}
})

type datalistConstructor func(...string) DatalistElement
func(c datalistConstructor) WithID(id string) func(options ...string)DatalistElement{
	return func(options ...string) DatalistElement {
		return DatalistElement{ui.BasicElement{LoadFromStorage(newDatalist(id, options...))}}
	}
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

	tag:= "option"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	withStringAttributeWatcher(e,"value")
	withStringAttributeWatcher(e,"label")
	withBoolAttributeWatcher(e,"disabled")
	withBoolAttributeWatcher(e,"selected")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var Option = optionConstructor(func (options ...string) OptionElement {
	return OptionElement{ui.BasicElement{LoadFromStorage(newOption(Elements.NewID(), options...))}}
})

type optionConstructor func(...string) OptionElement
func(c optionConstructor) WithID(id string) func(options ...string)OptionElement{
	return func(options ...string) OptionElement {
		return OptionElement{ui.BasicElement{LoadFromStorage(newOption(id, options...))}}
	}
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

	tag:= "optgroup"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	withStringAttributeWatcher(e,"label")
	withBoolAttributeWatcher(e,"disabled")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var Optgroup = optgroupConstructor(func (options ...string) OptgroupElement {
	return OptgroupElement{ui.BasicElement{LoadFromStorage(newOptgroup(Elements.NewID(), options...))}}
})

type optgroupConstructor func(...string) OptgroupElement
func(c optgroupConstructor) WithID(id string) func(options ...string)OptgroupElement{
	return func(options ...string) OptgroupElement {
		return OptgroupElement{ui.BasicElement{LoadFromStorage(newOptgroup(id, options...))}}
	}
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

	tag:= "fieldset"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	withStringAttributeWatcher(e,"form")
	withStringAttributeWatcher(e,"name")
	withBoolAttributeWatcher(e,"disabled")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var Fieldset = fieldsetConstructor(func (options ...string) FieldsetElement {
	return FieldsetElement{ui.BasicElement{LoadFromStorage(newFieldset(Elements.NewID(), options...))}}
})

type fieldsetConstructor func(...string) FieldsetElement
func(c fieldsetConstructor) WithID(id string) func(options ...string)FieldsetElement{
	return func(options ...string) FieldsetElement {
		return FieldsetElement{ui.BasicElement{LoadFromStorage(newFieldset(id, options...))}}
	}
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

	tag:= "legend"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	e.Watch("ui", "text", e, textContentHandler)

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var Legend = legendConstructor(func (options ...string) LegendElement {
	return LegendElement{ui.BasicElement{LoadFromStorage(newLegend(Elements.NewID(), options...))}}
})

type legendConstructor func(...string) LegendElement
func(c legendConstructor) WithID(id string) func(options ...string)LegendElement{
	return func(options ...string) LegendElement {
		return LegendElement{ui.BasicElement{LoadFromStorage(newLegend(id, options...))}}
	}
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

	tag:= "progress"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	withNumberAttributeWatcher(e,"max")
	withNumberAttributeWatcher(e,"value")

	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var Progress = progressConstructor(func (options ...string) ProgressElement {
	return ProgressElement{ui.BasicElement{LoadFromStorage(newProgress(Elements.NewID(), options...))}}
})

type progressConstructor func(...string) ProgressElement
func(c progressConstructor) WithID(id string) func(options ...string)ProgressElement{
	return func(options ...string) ProgressElement {
		return ProgressElement{ui.BasicElement{LoadFromStorage(newProgress(id, options...))}}
	}
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

	tag:= "select"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	withStringAttributeWatcher(e,"form")
	withStringAttributeWatcher(e,"name")
	withBoolAttributeWatcher(e,"disabled")
	withBoolAttributeWatcher(e,"required")
	withBoolAttributeWatcher(e,"multiple")
	withNumberAttributeWatcher(e,"size")
	withStringAttributeWatcher(e,"autocomplete")


	return e
}, AllowTooltip, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

var Select = selectConstructor(func (options ...string) SelectElement {
	return SelectElement{ui.BasicElement{LoadFromStorage(newSelect(Elements.NewID(), options...))}}
})

type selectConstructor func(...string) SelectElement
func(c selectConstructor) WithID(id string) func(options ...string)SelectElement{
	return func(options ...string) SelectElement {
		return SelectElement{ui.BasicElement{LoadFromStorage(newSelect(id, options...))}}
	}
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

	tag:= "form"
	var exist bool
	e.Native,exist = NewNativeElementIfAbsent(id, tag)
	if !exist {
		SetAttribute(e, "id", id)
	}

	e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		_,ok:= e.Get("ui","action")
		if !ok{
			evt.Origin().SetDataSetUI("action",ui.String(evt.Origin().Route()))
		}
		return false
	}))


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

var Form = formConstructor(func (options ...string) FormElement {
	return FormElement{ui.BasicElement{LoadFromStorage(newForm(Elements.NewID(), options...))}}
})

type formConstructor func(...string) FormElement
func(c formConstructor) WithID(id string) func(options ...string)FormElement{
	return func(options ...string) FormElement {
		return FormElement{ui.BasicElement{LoadFromStorage(newForm(id, options...))}}
	}
}




func AddClass(target *ui.Element, classname string) {
	category := "css"
	classes, ok := target.Get(category, "class")
	if ok {
		c, ok := classes.(ui.String)
		if !ok {
			target.Set(category, "class", ui.String(classname))
			return
		}
		sc := string(c)
		if !strings.Contains(sc, classname) {
			sc = strings.TrimSpace(sc + " " + classname)
			target.Set(category, "class", ui.String(sc))
		}
		return
	}
	target.Set(category, "class", ui.String(classname))
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

	target.Set(category, "class", ui.String(c))
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
			return false
		}
		RemoveAttribute(evt.Origin(), attr)
		return false
	}))
}

func withMediaElementPropertyWatchers(e *ui.Element) *ui.Element{
	withNumberPropertyWatcher(e,"currentTime")
	withNumberPropertyWatcher(e,"defaultPlaybackRate")
	withBoolPropertyWatcher(e,"disableRemotePlayback")
	withNumberPropertyWatcher(e,"playbackRate")
	withClampedNumberPropertyWatcher(e,"volume",0,1)
	withBoolPropertyWatcher(e,"preservesPitch")
	return e
}


func withStringPropertyWatcher(e *ui.Element,propname string){
	e.Watch("ui",propname,e,stringPropertyWatcher(propname))
}

func withBoolPropertyWatcher(e *ui.Element,propname string){
	e.Watch("ui",propname,e,boolPropertyWatcher(propname))
}

func withNumberPropertyWatcher(e *ui.Element,propname string){
	e.Watch("ui",propname,e,numericPropertyWatcher(propname))
}

func withClampedNumberPropertyWatcher(e *ui.Element, propname string, min int, max int){
	e.Watch("ui",propname,e,clampedValueWatcher(propname, min,max))
}





// Attr is a modifier that allows to set the value of an attribute if supported.
// If the element is not watching the ui property named after the attribute name, it does nothing.
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