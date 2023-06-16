// Package doc defines the default set of Element constructors, native interfaces,
// events and event handlers, and animation properties used to build js-based UIs.
package doc

import (
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"encoding/base64"
	"encoding/json"
	//"github.com/goccy/go-json"
	"strconv"
	"strings"
	//"math/rand"
	"net/url"
	"time"
	"runtime"
	"golang.org/x/exp/rand"

	"github.com/atdiar/particleui"
)


func init(){
	ui.NativeEventBridge = NativeEventBridge
	ui.NativeDispatch = NativeDispatch
}


var document *Document

// mutationCaptureMode describes how a Go App may capture textarea value changes
// that happen in native javascript. For instance, when a blur event is dispatched
// or when any mutation is observed via the MutationObserver API.
type mutationCaptureMode int

const (
	onBlur mutationCaptureMode = iota
	onInput
)

// inBrowser indicates whether the document is created in a browser environement or not.
// This
func inBrowser() bool{
	if runtime.GOOS == "js" && (runtime.GOARCH == "wasm"|| runtime.GOARCH == "ecmascript"){
		return true
	}
	return false
}




// newIDgenerator returns a function used to create new IDs. It uses
// a Pseudo-Random Number Generator (PRNG) as it is desirable to generate deterministic sequences.
// Evidently, as users navigate the app differently and may create new Elements
func newIDgenerator(charlen int, seed uint64) func() string {
	source := rand.NewSource(seed)
	r := rand.New(source)
	return func() string {
		var charset = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
		l:= len(charset)
		b := make([]rune, charlen)
		for i := range b {
			b[i] = charset[r.Intn(l)]
		}
		return string(b)
	}
}

var newID = newIDgenerator(16, uint64(time.Now().UnixNano()))


func SerializeStateHistory(e *ui.Element) string{ // TODO review mutationcapture state handling
	d:= GetDocument(e).AsElement()
	sth,ok:= d.Get("internals","mutationtrace")
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
	Raw *ui.Element
}

func (w Window) AsElement() *ui.Element {
	return w.Raw
}

func (w Window) SetTitle(title string) {
	w.AsElement().SetUI("title", ui.String(title))
}

// TODO see if can get height width of window view port, etc.

var newWindowConstructor= Elements.NewConstructor("window", func(id string) *ui.Element {
	e := ui.NewElement("window", "BROWSER")
	

	e.ElementStore = Elements
	e.Parent = e
	ConnectNative(e,"window")

	return e
})



func newWindow(title string, options ...string) Window {
	e:= newWindowConstructor("window", options...)
	e.SetUI("title", ui.String(title))
	return Window{e}
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

var allowdatapersistence = ui.NewConstructorOption("datapersistence", func(e *ui.Element) *ui.Element {
	d:= getDocumentRef(e)

	d.OnLoaded(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		LoadFromStorage(e)
		return false
	}))

	d.OnBeforeUnactive(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		PutInStorage(e)
		return false
	}))
	return e
})

func EnableScrollRestoration() string {
	return "scrollrestoration"
}

var routerConfig = func(r *ui.Router){

	ns:= func(id string) ui.Observable{
		d:= GetDocument(r.Outlet.AsElement())
		o:= d.NewObservable(id,EnableSessionPersistence())
		//PutInStorage(ClearFromStorage(o.AsElement())) // Note: oldsttate is cleared.
		PutInStorage(o.AsElement())
		return o
	}

	rs:= func(o ui.Observable) ui.Observable{
		LoadFromStorage(o.AsElement())
		return o
	}

	r.History.NewState = ns
	r.History.RecoverState = rs

	r.History.AppRoot.WatchEvent("history-change",r.History.AppRoot,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		PutInStorage(r.History.State[r.History.Cursor].AsElement())
		return false
	}).RunASAP())
	
	doc:= GetDocument(r.Outlet.AsElement())
	// Add default navigation error handlers
	// notfound:
	pnf:= doc.Div.WithID(r.Outlet.AsElement().Root.ID+"-notfound").SetText("Page Not Found.")
	SetAttribute(pnf.AsElement(),"role","alert")
	SetInlineCSS(pnf.AsElement(),`all: initial;`)
	

	r.OnNotfound(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		v,ok:= r.Outlet.AsElement().Root.Get("navigation", "targetviewid")
		if !ok{
			panic("targetview should have been set")
		}
		document:=  GetDocument(r.Outlet.AsElement())
		document.Window().SetTitle("Page Not Found")

		tv:= ui.ViewElement{document.GetElementById(v.(ui.String).String())}
		if tv.HasStaticView("notfound"){
			tv.ActivateView("notfound")
			return false
		}
		if r.Outlet.HasStaticView("notfound"){
			r.Outlet.ActivateView("notfound")
			return false
		}
		

		body:= document.Body().AsElement()
		body.SetChildren(pnf.AsElement())
		

		return false
	}))

	// unauthorized
	ui.AddView("unauthorized",doc.Div.WithID(r.Outlet.AsElement().ID+"-unauthorized").SetText("Unauthorized"))(r.Outlet.AsElement())
	r.OnUnauthorized(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		v,ok:= r.Outlet.AsElement().Root.Get("navigation", "targetviewid")
		if !ok{
			panic("targetview should have been set")
		}

		document:=  GetDocument(r.Outlet.AsElement())
		document.Window().SetTitle("Unauthorized")

		tv:= ui.ViewElement{GetDocument(r.Outlet.AsElement()).GetElementById(v.(ui.String).String())}
		if tv.HasStaticView("unauthorized"){
			tv.ActivateView("unauthorized")
			return false // DEBUG TODO return true?
		}
		r.Outlet.ActivateView("unauthorized")
		return false
	}))

	// appfailure
	afd:= doc.Div.WithID("ParticleUI-appfailure").SetText("App Failure")
	r.OnAppfailure(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		document:=  GetDocument(r.Outlet.AsElement())
		document.Window().SetTitle("App Failure")
		r.Outlet.AsElement().Root.SetChildren(afd.AsElement())
		return false
	}))

}

type  idEnabler [T any] interface{
	WithID(id string, options ...string) T
}

type constiface[T any] interface{
	func() T
	idEnabler[T]
}

type gconstructor[T ui.AnyElement, U constiface[T], V ~func(*ui.Element)*ui.Element] func()T

func(c gconstructor[T,U,V]) WithID(id string, options ...string) T{
	var u U
	e := u.WithID(id, options...)
	var v V
	v(e.AsElement())
	return e
}

// For ButtonElement: it has a dedicated Document linked constructor as it has an optional typ argument
type  idEnablerButton [T any] interface{
	WithID(id string, typ string, options ...string) T
}

type buttonconstiface[T any] interface{
	func(typ ...string) T
	idEnablerButton[T]
}

type buttongconstructor[T ui.AnyElement, U buttonconstiface[T], V ~func(*ui.Element)*ui.Element] func(typ ...string)T

func(c buttongconstructor[T,U,V]) WithID(id string, typ string,  options ...string) T{
	var u U
	e := u.WithID(id, typ, options...)
	var v V
	v(e.AsElement())
	return e
}

// For inputElement: it has a dedicated Document linked constructor as it has an additional typ argument
type  idEnablerinput [T any] interface{
	WithID(id string, typ string, options ...string) T
}

type inputconstiface[T any] interface{
	func(typ string) T
	idEnablerinput[T]
}

type inputgconstructor[T ui.AnyElement, U inputconstiface[T], V ~func(*ui.Element)*ui.Element] func(typ string)T

func(c inputgconstructor[T,U,V]) WithID(id string, typ string,  options ...string) T{
	var u U
	e := u.WithID(id, typ, options...)
	var v V
	v(e.AsElement())
	return e
}

//

type Document struct {
	*ui.Element
	
	// id generator with serializable state
	// used to generate unique ids for elements
	rng   *rand.Rand
	src *rand.PCGSource

	// Document should hold the list of all element constructors such as Meta, Title, Div, San etc.
	body bodyConstructor
	head headConstructor
	Meta metaConstructor
	Title titleConstructor
	Script scriptConstructor
	Base baseConstructor
	NoScript noscriptConstructor
	Link linkConstructor
	Div divConstructor
	TextArea textareaConstructor
	Header headerConstructor
	Footer footerConstructor
	Section sectionConstructor
	H1 h1Constructor
	H2 h2Constructor
	H3 h3Constructor
	H4 h4Constructor
	H5 h5Constructor
	H6 h6Constructor
	Span spanConstructor
	Article articleConstructor
	Aside asideConstructor
	Main mainConstructor
	Paragraph paragraphConstructor
	Nav navConstructor
	Anchor anchorConstructor
	Button buttonConstructor
	Label labelConstructor
	Input inputConstructor
	Output outputConstructor
	Img imgConstructor
	Audio audioConstructor
	Video videoConstructor
	Source sourceConstructor
	Ul ulConstructor
	Ol olConstructor
	Li liConstructor
	Table tableConstructor
	Thead theadConstructor
	Tbody tbodyConstructor
	Tr trConstructor
	Td tdConstructor
	Th thConstructor
	Col colConstructor
	ColGroup colgroupConstructor
	Canvas canvasConstructor
	Svg svgConstructor
	Summary summaryConstructor
	Details detailsConstructor
	Dialog dialogConstructor
	Code codeConstructor
	Embed embedConstructor
	Object objectConstructor
	Datalist datalistConstructor
	Option optionConstructor
	Optgroup optgroupConstructor
	Fieldset fieldsetConstructor
	Legend legendConstructor
	Progress progressConstructor
	Select selectConstructor
	Form formConstructor

}

func(d *Document) initializeIDgenerator() {
	var seed uint64
	err := binary.Read(crand.Reader, binary.LittleEndian, &seed)
	if err != nil {
		panic(err)
	}
	d.src= &rand.PCGSource{}
	d.src.Seed(seed)
	d.rng = rand.New(d.src)
	d.saveIDgeneratorState() // saves the initial state. Useful when replaying the app mutaton trace.
	
	d.Watch("internals","PCGSate",d, ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		// pcgstate should be a ui.String. We should recover the PCGSOurce from it by using UnmarshalBinary.
		pcgstr:= evt.NewValue().(ui.String).String()
		pcg,err:= base64.StdEncoding.DecodeString(pcgstr)
		if err != nil{
			panic(err)
		}
		d.src = &rand.PCGSource{}
		d.src.UnmarshalBinary(pcg)
		d.rng = rand.New(d.src)
		return false
	}))
}

func(d *Document) tryLoadIDgeneratorState(){
	pcgstate,ok:= d.Get("internals","PCGSate")
	if !ok{
		panic("failed to find ID generator state")
	}
	// pcgstate should be a ui.String. We should recover the PCGSOurce from it by using UnmarshalBinary.
	pcgstr:= pcgstate.(ui.String).String()
	pcg,err:= base64.StdEncoding.DecodeString(pcgstr)
	if err != nil{
		panic(err)
	}
	d.src = &rand.PCGSource{}
	d.src.UnmarshalBinary(pcg)
	d.rng = rand.New(d.src)

}

func(d *Document) saveIDgeneratorState(){
	pcg,err:= d.src.MarshalBinary()
	if err!= nil{
		panic(err)
	}
	pcgstr:= base64.StdEncoding.EncodeToString(pcg)
	d.Set("internals","PCGSate",ui.String(pcgstr))
}


func (d Document) Window() Window {
	w:= d.GetElementById("window")
	if w != nil{
		return Window{w}
	}
	wd:= newWindow("zui-window")
	ui.RegisterElement(d.AsElement(),wd.Raw)
	wd.Raw.TriggerEvent("mounted", ui.Bool(true))
	wd.Raw.TriggerEvent("mountable", ui.Bool(true))
	d.AsElement().BindValue("ui","title",wd.AsElement())
	return wd
}

func (d Document)GetElementById(id string) *ui.Element{
	return ui.GetById(d.AsElement(),id)
}

func(d Document) newID() string{
	/*

	var charset = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	l:= len(charset)
	b := make([]rune, 32)
	for i := range b {
		b[i] = charset[d.rng.Intn(l)]
	}
	return string(b)

	*/
	return newID() // DEBUG
}

func(d Document) NewObservable(id string, options ...string) ui.Observable{
	if e:=d.GetElementById(id); e != nil{
		ui.Delete(e)
	}
	o:= d.AsElement().ElementStore.NewObservable(id,options...).AsElement()
	
	ui.RegisterElement(d.AsElement(),o)

	return ui.Observable{o}
}	


func(d Document) Head() *ui.Element{
	e:= d.GetElementById("head")
	if e ==nil{ panic("document HEAD seems to be missing for some odd reason...")}
	return e //d.NewComponent(e)
}

func(d Document) Body() *ui.Element{
	e:= d.GetElementById("body")
	if e ==nil{ panic("document BODY seems tobe missing for some odd reason...")}
	return d.NewComponent(e)
}


// NewComponent ensures the registration of a component. A component is a part of an UI that is composed
// of at least one Element.
// Typicvally, component are returned by functions. These functions should integrate a call to NewComponent 
// to ensure its registration with the document.
func (d Document) NewComponent(root ui.AnyElement) *ui.Element{
	el:= root.AsElement()
	if el.Registered(){
		return el
	}
	
	ui.RegisterElement(d.AsElement(),el)

	return el
}

func(d Document) SetLang(lang string) Document{
	d.AsElement().SetUI("lang", ui.String(lang))
	return d
}

func (d Document) OnNavigationEnd(h *ui.MutationHandler){
	d.AsElement().WatchEvent("navigation-end", d, h)
}

func(d Document) OnLoaded(h *ui.MutationHandler){
	d.AsElement().WatchEvent("document-loaded",d,h)
}

func(d Document) OnRouterMounted(h *ui.MutationHandler){
	d.AsElement().WatchEvent("router-mounted",d,h)
}

func(d Document) OnBeforeUnactive(h *ui.MutationHandler){
	d.AsElement().WatchEvent("before-unactive",d,h)
}

// Router returns the router associated with the document. It is nil if no router has been created.
func(d Document) Router() *ui.Router{
	return ui.GetRouter(d.AsElement())
}

func(d Document) Delete(){ // TODO check for dangling references
	ui.DoSync(func(){
		e:= d.AsElement()
		d.Router().NavCancel()
		ui.Delete(e)
	})
}

func(d Document) SetTitle(title string){
	d.AsElement().SetUI("title",ui.String(title))
}

// ListenAndServe is used to start listening to state changes to the document (aka navigation)
// coming from the browser such as popstate.
// It needs to run at the end, after the UI tree has been built.
func(d Document) ListenAndServe(ctx context.Context){
	if d.Element ==nil{
		panic("document is missing")
	}
	ui.GetRouter(d.AsElement()).ListenAndServe(ctx,"popstate", d.Window())
}


func GetDocument(e *ui.Element) Document{
	if document != nil{
		return *document
	}
	if e.Root == nil{
		panic("This element does not belong to any registered subtree of the Document. Root is nil. If root of a component, it should be declared as such by callling the NewComponent method of the document Element.")
	}
	return withStdConstructors(Document{Element:e.Root}) // TODO initialize document *Element constructors
}


// getDocumentRef is needed for the definition of constructors wich need to refer to the document 
// such as body, head or title. Indeed, since they 
func getDocumentRef(e *ui.Element) *Document{
	if document != nil{
		return document
	}
	return &Document{Element:e.Root}
}

var newDocument = Elements.NewConstructor("html", func(id string) *ui.Element {

	e := Elements.NewAppRoot(id).AsElement()

	ConnectNative(e, "html")

	getDocumentRef(e).initializeIDgenerator()

	e.Watch("ui","lang",e,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		SetAttribute(evt.Origin(),"lang",string(evt.NewValue().(ui.String)))
		return false
	}).RunASAP())

	

    e.Watch("ui","history",e,historyMutationHandler)

	// makes ViewElements focusable (focus management support)
	e.Watch("internals", "views",e.Root,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		l:= evt.NewValue().(ui.List)
		viewstr:= l.Get(len(l.UnsafelyUnwrap())-1).(ui.String)
		view := ui.GetById(e, string(viewstr))
		SetAttribute(view,"tabindex","-1")
		e.Watch("ui","activeview",view,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			e.SetUI("focus",ui.String(view.ID))
			return false
		}))
		return false
	}))

	e.OnRouterMounted(func(r *ui.Router){
		e.AddEventListener("focusin",ui.NewEventHandler(func(evt ui.Event)bool{
			r.History.Set("focusedElementId",ui.String(evt.Target().ID))
			return false
		}))
		
	})	
	
	return e
}, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence, AllowScrollRestoration)

var documentTitleHandler= ui.NewMutationHandler(func(evt ui.MutationEvent) bool { 
	d:= GetDocument(evt.Origin())
	ot:= d.GetElementById("document-title")
	if ot == nil{
		t:= d.Title.WithID("document-title")
		t.AsElement().Root = evt.Origin()
		t.Set(string(evt.NewValue().(ui.String)))
		d.Head().AppendChild(t)
		return false
	}
	TitleElement{ot}.Set(string(evt.NewValue().(ui.String)))

	return false
}).RunASAP()

func mutationreplay(d *Document) {

	e:= d.Element
	if !e.ElementStore.MutationReplay{
		return
	}

	rh,ok:= e.Get("internals","mutationtrace")
	if !ok{
		panic("somehow recovering state failed. Unexpected error")
	}
	mutationtrace, ok:= rh.(ui.List)
	if !ok{
		panic("state history should have been a ui.List. Wrong type. Unexpected error")
	}


	for _,rawop:= range mutationtrace.UnsafelyUnwrap(){
		op:= rawop.(ui.Object)
		elementid:= string(op.MustGetString("id"))
		category:= string(op.MustGetString("cat"))
		propname:= string(op.MustGetString("prop"))
		value,_:= op.Get("val")
		el:= GetDocument(e).GetElementById(elementid)
		if el == nil{
			panic("Unable to recover state for this element id. Element  doesn't exist")
		}
		el.BindValue("event","mutationreplayed",e)
		el.Set(category,propname,value)
	}

	e.TriggerEvent("mutationreplayed")
}

func Autofocus(e *ui.Element) *ui.Element{
	e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		evt.Origin().WatchEvent("navigation-end",evt.Origin().Root,ui.NewMutationHandler(func(event ui.MutationEvent)bool{
			r:= ui.GetRouter(event.Origin())
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

// withNativejshelpers returns a modifier that appends a scriptin which naive js functions to be called
// from Go are defined
func withNativejshelpers(d *Document) *Document{
	s:= d.Script.WithID("nativehelpers").
		SetInnerHTML(
			`
			window.focusElement = function(element) {
				if(element) {
					element.focus({preventScroll: true});
				} else {
					console.error('Element is not defined');
				}
			}
			
			window.queueFocus = function(element) {
				queueMicrotask(() => focusElement(element));
			}

			window.blurElement = function(element) {
				if(element) {
					element.blur();
				} else {
					console.error('Element is not defined');
				}
			}

			window.queueBlur = function(element) {
				queueMicrotask(() => blurElement(element));
			}
			
			window.clearFieldValue = function(element) {
				if(element) {
					element.value = "";
				} else {
					console.error('Element is not defined');
				}
			}

			window.queueClear = function(element) {
				queueMicrotask(() => clearFieldValue(element));
			}
			`,
		)
	h:= d.Head()
	h.AppendChild(s)

	return d
}

// NewDocument returns the root of new js app. It is the top-most element
// in the tree of Elements that consitute the full document.
// Options such as the location of persisted data can be passed to the constructor of an instance.
func NewDocument(id string, options ...string) Document {
	d:= Document{Element:newDocument(id, options...)}
	
	d = withStdConstructors(d)

	e:= d.Element

	e.AppendChild(d.head.WithID("head")) 
	e.AppendChild(d.body.WithID("body"))

	e.OnRouterMounted(routerConfig)
	e.WatchEvent("document-loaded", e,navinitHandler)
	e.Watch("ui", "title", e, documentTitleHandler)


	mutationreplay(&d)
	activityStateSupport(e)

	if inBrowser(){
		document = &d
	}

	return d
}

func withStdConstructors(d Document) Document{
	d.body = bodyConstructor(func() BodyElement {
		e:=  BodyElement{newBody(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})

	d.head = headConstructor(func() HeadElement {
		e:=  HeadElement{newHead(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})

	d.Meta = metaConstructor(func() MetaElement {
		e:=  MetaElement{newMeta(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})

	d.Title = titleConstructor(func() TitleElement {
		e:=  TitleElement{newTitle(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})

	d.Script = scriptConstructor(func() ScriptElement {
		e := ScriptElement{newScript(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})

	d.Base = baseConstructor(func() BaseElement {
		e := BaseElement{newBase(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})

	d.NoScript = noscriptConstructor(func() NoScriptElement {
		e := NoScriptElement{newNoScript(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})

	d.Link = linkConstructor(func() LinkElement {
		e := LinkElement{newLink(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})

	d.Div = divConstructor(func() DivElement {
		e := DivElement{newDiv(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})

	d.TextArea = textareaConstructor(func() TextAreaElement {
		e := TextAreaElement{newTextArea(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})

	d.Header = headerConstructor(func() HeaderElement {
		e := HeaderElement{newHeader(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})

	d.Footer = footerConstructor(func() FooterElement {
		e := FooterElement{newFooter(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})

	d.Section = sectionConstructor(func() SectionElement {
		e := SectionElement{newSection(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})

	d.H1 = h1Constructor(func() H1Element {
		e := H1Element{newH1(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})

	d.H2 = h2Constructor(func() H2Element {
		e := H2Element{newH2(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})

	d.H3 = h3Constructor(func() H3Element {
		e := H3Element{newH3(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})

	d.H4 = h4Constructor(func() H4Element {
		e := H4Element{newH4(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})

	d.H5 = h5Constructor(func() H5Element {
		e := H5Element{newH5(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})

	d.H6 = h6Constructor(func() H6Element {
		e := H6Element{newH6(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})

	d.Span = spanConstructor(func() SpanElement {
		e := SpanElement{newSpan(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	
	d.Article = articleConstructor(func() ArticleElement {
		e := ArticleElement{newArticle(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	
	d.Aside = asideConstructor(func() AsideElement {
		e := AsideElement{newAside(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	
	d.Main = mainConstructor(func() MainElement {
		e := MainElement{newMain(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	
	d.Paragraph = paragraphConstructor(func() ParagraphElement {
		e := ParagraphElement{newParagraph(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	
	d.Nav = navConstructor(func() NavElement {
		e := NavElement{newNav(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	
	d.Anchor = anchorConstructor(func() AnchorElement {
		e := AnchorElement{newAnchor(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	
	d.Button = buttonConstructor(func(typ ...string) ButtonElement {
		e := ButtonElement{newButton(d.newID(), typ...)}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	
	d.Label = labelConstructor(func() LabelElement {
		e := LabelElement{newLabel(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	
	d.Input = inputConstructor(func(typ string) InputElement {
		e := InputElement{newInput(d.newID(), typ)}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	
	d.Output = outputConstructor(func() OutputElement {
		e := OutputElement{newOutput(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	
	d.Img = imgConstructor(func() ImgElement {
		e := ImgElement{newImg(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	
	d.Audio = audioConstructor(func() AudioElement {
		e := AudioElement{newAudio(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	
	d.Video = videoConstructor(func() VideoElement {
		e := VideoElement{newVideo(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	
	d.Source = sourceConstructor(func() SourceElement {
		e := SourceElement{newSource(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	
	d.Ul = ulConstructor(func() UlElement {
		e := UlElement{newUl(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	
	d.Ol = olConstructor(func(typ string, offset int) OlElement {
		e := OlElement{newOl(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	
	d.Li = liConstructor(func() LiElement {
		e := LiElement{newLi(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	
	d.Table = tableConstructor(func() TableElement {
		e := TableElement{newTable(d.newID())}
		e.Root = d.Element
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})

	return d
}


type BodyElement struct{
	*ui.Element
}

var newBody = Elements.NewConstructor("body",func(id string) *ui.Element{
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	ConnectNative(e, "body")

	e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		evt.Origin().Root.Set("ui","body",ui.String(evt.Origin().ID))
		return false
	}).RunOnce().RunASAP())

	return e
}, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence, AllowScrollRestoration)


type bodyConstructor func() BodyElement
func(c bodyConstructor) WithID(id string, options ...string)BodyElement{
	return BodyElement{newBody(id, options...)}
}



// Head refers to the <head> HTML element of a HTML document, which contains metadata and links to 
// resources such as title, scripts, stylesheets.
type HeadElement struct{
	*ui.Element
}

var newHead = Elements.NewConstructor("head",func(id string)*ui.Element{
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	ConnectNative(e, "head")


	e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		evt.Origin().Root.Set("ui","head",ui.String(evt.Origin().ID))
		return false
	}).RunOnce().RunASAP())

	return e
})



type headConstructor func() HeadElement
func(c headConstructor) WithID(id string, options ...string)HeadElement{
	return HeadElement{newHead(id, options...)}
}

// Meta : for definition and examples, see https://developer.mozilla.org/en-US/docs/Web/HTML/Element/meta
type MetaElement struct{
	*ui.Element
}

func(m MetaElement) SetAttribute(name,value string) MetaElement{
	SetAttribute(m.AsElement(),name,value)
	return m
}

var newMeta = Elements.NewConstructor("meta",func(id string)*ui.Element{
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "meta"
	ConnectNative(e, tag)

	return e
})


type metaConstructor func() MetaElement
func(c metaConstructor) WithID(id string, options ...string)MetaElement{
	return MetaElement{newMeta(id, options...)}
}


type TitleElement struct{
	*ui.Element
}

func(m TitleElement) Set(title string) TitleElement{
	m.AsElement().SetUI("title",ui.String(title))
	return m
}

var newTitle = Elements.NewConstructor("title",func(id string)*ui.Element{
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "title"
	ConnectNative(e, tag)
	e.Watch("ui","title",e,titleElementChangeHandler)

	return e
})


type titleConstructor func() TitleElement
func(c titleConstructor) WithID(id string,options ...string)TitleElement{
	return TitleElement{newTitle(id, options...)}
}

// ScriptElement is an Element that refers to the HTML Element of the same name that embeds executable 
// code or data.
type ScriptElement struct{
	*ui.Element
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
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "script"
	ConnectNative(e, tag)

	return e
})


type scriptConstructor func() ScriptElement
func(c scriptConstructor) WithID(id string, options ...string)ScriptElement{
	return ScriptElement{newScript(id, options...)}
}


// BaseElement allows to define the baseurl or the basepath for the links within a page.
// In our current use-case, it will mostly be used when generating HTML (SSR or SSG).
// It is then mostly a build-time concern.
type BaseElement struct{
	*ui.Element
}

func(b BaseElement) SetHREF(url string) BaseElement{
	b.AsElement().SetUI("href",ui.String(url))
	return b
}

var newBase = Elements.NewConstructor("base",func(id string)*ui.Element{
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "base"
	ConnectNative(e, tag)

	e.Watch("ui","href",e,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		SetAttribute(evt.Origin(),"href",string(evt.NewValue().(ui.String)))
		return false
	}))

	return e
})


type baseConstructor func() BaseElement
func(c baseConstructor) WithID(id string, options ...string)BaseElement{
	return BaseElement{newBase(id, options...)}
}


// NoScriptElement refers to an element that defines a section of HTMNL to be inserted in a page if a script
// type is unsupported on the page of scripting is turned off.
// As such, this is mostly useful during SSR or SSG, for examplt to display a message if javascript
// is disabled.
// Indeed, if scripts are disbaled, wasm will not be able to insert this dynamically into the page.
type NoScriptElement struct{
	*ui.Element
}

func(s NoScriptElement) SetInnerHTML(content string) NoScriptElement{
	SetInnerHTML(s.AsElement(),content)
	return s
}

var newNoScript = Elements.NewConstructor("noscript",func(id string)*ui.Element{
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "noscript"
	ConnectNative(e, tag)

	return e
})



type noscriptConstructor func() NoScriptElement
func(c noscriptConstructor) WithID(id string, options ...string)NoScriptElement{
	return NoScriptElement{newNoScript(id, options...)}
}

// Link refers to the <link> HTML Element which allow to specify the location of external resources
// such as stylesheets or a favicon.
type LinkElement struct{
	*ui.Element
}

func(l LinkElement) SetAttribute(name,value string) LinkElement{
	SetAttribute(l.AsElement(),name,value)
	return l
}

var newLink = Elements.NewConstructor("link",func(id string)*ui.Element{
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "link"
	ConnectNative(e, tag)

	return e
})


type linkConstructor func() LinkElement
func(c linkConstructor) WithID(id string, options ...string) LinkElement{
	return LinkElement{newLink(id, options...)}
}


// Content Sectioning and other HTML Elements

// DivElement is a concrete type that holds the common interface to Div *ui.Element objects.
// i.e. ui.Element whose constructor name is "div" and represents html div elements.
type DivElement struct {
	*ui.Element
}

func (d DivElement) Contenteditable(b bool) DivElement {
	d.AsElement().SetUI("contenteditable", ui.Bool(b))
	return d
}

func (d DivElement) SetText(str string) DivElement {
	d.AsElement().SetUI("text", ui.String(str))
	return d
}

var newDiv = Elements.NewConstructor("div", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	
	tag:= "div"
	ConnectNative(e, tag)

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
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence, AllowScrollRestoration)



type divConstructor func() DivElement
func(d divConstructor) WithID(id string, options ...string)DivElement{
	return DivElement{newDiv(id, options...)}
}

const SSRStateSuffix = "-ssr-state"
const HydrationAttrName = "data-needh2o"



// TODO implement spellcheck and autocomplete methods
type TextAreaElement struct {
	*ui.Element
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
		e.SetUI("value", ui.String(text))
		return e
	}
}

func(t textAreaModifer) Cols(i int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("cols",ui.Number(i))
		return e
	}
}

func(t textAreaModifer) Rows(i int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("rows",ui.Number(i))
		return e
	}
}

func(t textAreaModifer) MinLength(i int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("minlength",ui.Number(i))
		return e
	}
}

func(t textAreaModifer) MaxLength(i int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("maxlength",ui.Number(i))
		return e
	}
}
func(t textAreaModifer) Required(b bool) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("required",ui.Bool(b))
		return e
	}
}

func (t textAreaModifer) Form(form *ui.Element) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			d:= evt.Origin().Root
			
			evt.Origin().WatchEvent("navigation-end",d,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
				if form.Mounted(){
					e.AsElement().SetUI("form", ui.String(form.ID))
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
		e.SetUI("name",ui.String(name))
		return e
	}
}

func(t textAreaModifer) Placeholder(p string)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("placeholder",ui.String(p))
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
		e.SetUI("wrap",ui.String(v))
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
		e.SetUI("autocomplete",ui.String(val))
		return e
	}
}

func(t textAreaModifer) Spellcheck(mode string)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		v:= "default"
		if mode == "true" || mode == "false"{
			v = mode
		}
		e.SetUI("spellcheck",ui.String(v))
		return e
	}
}

var newTextArea = Elements.NewConstructor("textarea", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "textarea"
	ConnectNative(e, tag)

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
}, allowTextAreaDataBindingOnBlur, allowTextAreaDataBindingOnInput,AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)


// TextArea is a constructor for a textarea html element.


type textareaConstructor func() TextAreaElement
func(c textareaConstructor) WithID(id string, options ...string)TextAreaElement{
	return TextAreaElement{newTextArea(id, options...)}
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
	*ui.Element
}

var newHeader= Elements.NewConstructor("header", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "header"
	ConnectNative(e, tag)

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

// Header is a constructor for a html header element.


type headerConstructor func() HeaderElement
func(c headerConstructor) WithID(id string, options ...string)HeaderElement{
	return HeaderElement{newHeader(id, options...)}
}

type FooterElement struct {
	*ui.Element
}

var newFooter= Elements.NewConstructor("footer", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "footer"
	ConnectNative(e, tag)

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

// Footer is a constructor for an html footer element.


type footerConstructor func() FooterElement
func(c footerConstructor) WithID(id string, options ...string)FooterElement{
	return FooterElement{newFooter(id, options...)}

}

type SectionElement struct {
	*ui.Element
}

var newSection= Elements.NewConstructor("section", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "section"
	ConnectNative(e, tag)

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

// Section is a constructor for html section elements.


type sectionConstructor func() SectionElement
func(c sectionConstructor) WithID(id string, options ...string)SectionElement{
	return SectionElement{newSection(id, options...)}
}

type H1Element struct {
	*ui.Element
}

func (h H1Element) SetText(s string) H1Element {
	h.AsElement().SetUI("text", ui.String(s))
	return h
}

var newH1= Elements.NewConstructor("h1", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "h1"
	ConnectNative(e, tag)

	e.Watch("ui", "text", e, textContentHandler)
	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type h1Constructor func() H1Element
func(c h1Constructor) WithID(id string, options ...string)H1Element{
	return H1Element{newH1(id, options...)}
}

type H2Element struct {
	*ui.Element
}

func (h H2Element) SetText(s string) H2Element {
	h.AsElement().SetUI("text", ui.String(s))
	return h
}

var newH2= Elements.NewConstructor("h2", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "h2"
	ConnectNative(e, tag)

	e.Watch("ui", "text", e,textContentHandler)
	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)


type h2Constructor func() H2Element
func(c h2Constructor) WithID(id string, options ...string)H2Element{
	return H2Element{newH2(id, options...)}
}

type H3Element struct {
	*ui.Element
}

func (h H3Element) SetText(s string) H3Element {
	h.AsElement().SetUI("text", ui.String(s))
	return h
}

var newH3= Elements.NewConstructor("h3", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "h3"
	ConnectNative(e, tag)

	e.Watch("ui", "text", e,textContentHandler)
	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)


type h3Constructor func() H3Element
func(c h3Constructor) WithID(id string, options ...string)H3Element{
	return H3Element{newH3(id, options...)}
}

type H4Element struct {
	*ui.Element
}

func (h H4Element) SetText(s string) H4Element {
	h.AsElement().SetUI("text", ui.String(s))
	return h
}

var newH4= Elements.NewConstructor("h4", func(id string) *ui.Element {
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "h4"
	ConnectNative(e, tag)

	e.Watch("ui", "text", e, textContentHandler)
	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)


type h4Constructor func() H4Element
func(c h4Constructor) WithID(id string, options ...string)H4Element{
	return H4Element{newH4(id, options...)}
}

type H5Element struct {
	*ui.Element
}

func (h H5Element) SetText(s string) H5Element {
	h.AsElement().SetUI("text", ui.String(s))
	return h
}

var newH5= Elements.NewConstructor("h5", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "h5"
	ConnectNative(e, tag)

	e.Watch("ui", "text", e, textContentHandler)
	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)


type h5Constructor func() H5Element
func(c h5Constructor) WithID(id string, options ...string)H5Element{
	return H5Element{newH5(id, options...)}
}

type H6Element struct {
	*ui.Element
}

func (h H6Element) SetText(s string) H6Element {
	h.AsElement().SetUI("text", ui.String(s))
	return h
}

var newH6= Elements.NewConstructor("h6", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "h6"
	ConnectNative(e, tag)

	e.Watch("ui", "text", e,textContentHandler)
	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)


type h6Constructor func() H6Element
func(c h6Constructor) WithID(id string, options ...string)H6Element{
	return H6Element{newH6(id, options...)}
}

type SpanElement struct {
	*ui.Element
}

func (s SpanElement) SetText(str string) SpanElement {
	s.AsElement().SetUI("text", ui.String(str))
	return s
}

var newSpan= Elements.NewConstructor("span", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "span"
	ConnectNative(e, tag)

	e.Watch("ui", "text", e, textContentHandler)

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

// Span is a constructor for html span elements.


type spanConstructor func() SpanElement
func(c spanConstructor) WithID(id string, options ...string)SpanElement{
	return SpanElement{newSpan(id, options...)}
}

type ArticleElement struct {
	*ui.Element
}


var newArticle= Elements.NewConstructor("article", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "article"
	ConnectNative(e, tag)


	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type articleConstructor func() ArticleElement
func(c articleConstructor) WithID(id string, options ...string)ArticleElement{
	return ArticleElement{newArticle(id, options...)}
}


type AsideElement struct {
	*ui.Element
}

var newAside= Elements.NewConstructor("aside", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "aside"
	ConnectNative(e, tag)

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type asideConstructor func() AsideElement
func(c asideConstructor) WithID(id string, options ...string)AsideElement{
	return AsideElement{newAside(id, options...)}
}

type MainElement struct {
	*ui.Element
}

var newMain= Elements.NewConstructor("main", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "main"
	ConnectNative(e, tag)

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type mainConstructor func() MainElement
func(c mainConstructor) WithID(id string, options ...string)MainElement{
	return MainElement{newMain(id, options...)}
}


type ParagraphElement struct {
	*ui.Element
}

func (p ParagraphElement) SetText(s string) ParagraphElement {
	p.AsElement().SetUI("text", ui.String(s))
	return p
}

var newParagraph= Elements.NewConstructor("p", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "p"
	ConnectNative(e, tag)

	e.Watch("ui", "text", e, paragraphTextHandler)
	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

// Paragraph is a constructor for html paragraph elements.


type paragraphConstructor func() ParagraphElement
func(c paragraphConstructor) WithID(id string, options ...string)ParagraphElement{
	return ParagraphElement{newParagraph(id, options...)}
}

type NavElement struct {
	*ui.Element
}

var newNav= Elements.NewConstructor("nav", func(id string) *ui.Element {
		e := Elements.NewElement(id)
		e = enableClasses(e)

		tag:= "nav"
		ConnectNative(e, tag) 

		return e
	},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

// Nav is a constructor for a html nav element.


type navConstructor func() NavElement
func(c navConstructor) WithID(id string, options ...string)NavElement{
	return NavElement{newNav(id, options...)}
}

type AnchorElement struct {
	*ui.Element
}

func (a AnchorElement) SetHREF(target string) AnchorElement {
	a.AsElement().SetUI("href", ui.String(target))
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
	a.AsElement().WatchEvent("verified", link, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		a.SetHREF(link.URI()+hash)
		return false
	}).RunASAP())

	a.AsElement().Watch("ui", "active", link, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		a.SetUI("active", evt.NewValue())
		return false
	}).RunASAP())


	a.AddEventListener("click", ui.NewEventHandler(func(evt ui.Event) bool {
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

	a.SetUI("link", ui.String(link.AsElement().ID))

	

	pm,ok:= a.AsElement().Get("internals","prefetchmode")
	if ok && !prefetchDisabled(){
		switch t:= string(pm.(ui.String));t{
		case "intent":
			a.AddEventListener("mouseover",ui.NewEventHandler(func(evt ui.Event)bool{
				link.Prefetch()
				return false
			}))
		case "render":
			a.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
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
	a.AsElement().SetUI("text", ui.String(text))
	return a
}

var newAnchor= Elements.NewConstructor("a", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "a"
	ConnectNative(e, tag)

	withStringAttributeWatcher(e,"href")

	withStringPropertyWatcher(e,"text")

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence, AllowPrefetchOnIntent, AllowPrefetchOnRender)

// Anchor creates an html anchor element.


type anchorConstructor func() AnchorElement
func(c anchorConstructor) WithID(id string, options ...string)AnchorElement{
	return AnchorElement{newAnchor(id, options...)}
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
	*ui.Element
}

type buttonModifer struct{}
var ButtonModifier buttonModifer

func(m buttonModifer) Disabled(b bool) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("disabled",ui.Bool(b))
		return e
	}
}

func(m buttonModifer) Text(str string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("text", ui.String(str))
		return e
	}
}

func(b buttonModifer) Form(form *ui.Element) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			d:= evt.Origin().Root
			
			evt.Origin().WatchEvent("navigation-end",d,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
				if form.Mounted(){
					e.SetUI("form", ui.String(form.ID))
				}
				return false
			}).RunOnce())
			return false
		}).RunOnce())
		return e
	}
}


func (b ButtonElement) SetDisabled(t bool) ButtonElement {
	b.AsElement().SetUI("disabled", ui.Bool(t))
	return b
}

func (b ButtonElement) SetText(str string) ButtonElement {
	b.AsElement().SetUI("text", ui.String(str))
	return b
}

var newButton= Elements.NewConstructor("button", func(id  string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "button"
	ConnectNative(e, tag)

	withBoolAttributeWatcher(e,"disabled")
	withBoolAttributeWatcher(e,"autofocus")

	withStringAttributeWatcher(e,"form")
	withStringAttributeWatcher(e,"type")
	withStringAttributeWatcher(e,"name")

	e.Watch("ui", "text", e, textContentHandler)

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence, 
	buttonOption("button"), 
	buttonOption("submit"), 
	buttonOption("reset"),
)


type buttonConstructor func(typ ...string) ButtonElement
func(c buttonConstructor) WithID(id string, typ string, options ...string)ButtonElement{
	options = append(options, typ)
	return ButtonElement{newButton(id, options...)}
}

func buttonOption(name string) ui.ConstructorOption{
	return ui.NewConstructorOption(name,func(e *ui.Element)*ui.Element{

		e.SetUI("type",ui.String(name))		

		return e
	})
}

type LabelElement struct {
	*ui.Element
}

type labelModifier struct{}
var LabelModifier labelModifier

func(m labelModifier) Text(str string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("text", ui.String(str))
		return e
	}
}

func(m labelModifier) For(e *ui.Element) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			d:= evt.Origin().Root
			
			evt.Origin().WatchEvent("navigation-end",d,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
				if e.Mounted(){
					e.SetUI("for", ui.String(e.ID))
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
	l.AsElement().SetUI("text", ui.String(s))
	return l
}

func (l LabelElement) For(e *ui.Element) LabelElement {
	l.AsElement().OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		d:= evt.Origin().Root
		
		evt.Origin().WatchEvent("navigation-end",d,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			if e.Mounted(){
				l.AsElement().SetUI("for", ui.String(e.ID))
			}
			return false
		}).RunOnce())
		return false
	}).RunOnce())
	return l
}

var newLabel= Elements.NewConstructor("label", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "label"
	ConnectNative(e, tag)

	withStringAttributeWatcher(e,"for")
	e.Watch("ui", "text", e, textContentHandler)
	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type labelConstructor func() LabelElement
func(c labelConstructor) WithID(id string, options ...string)LabelElement{
	return LabelElement{newLabel(id, options...)}
}

type InputElement struct {
	*ui.Element
}

type inputModifier struct{}
var InputModifier inputModifier

func(i inputModifier) Step(step int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("step",ui.Number(step))
		return e
	}
}

func(i inputModifier) Checked(b bool) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("checked",ui.Bool(b))
		return e
	}
}

func(i inputModifier) Disabled(b bool) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("disabled",ui.Bool(b))
		return e
	}
}

func(i inputModifier) MaxLength(m int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("maxlength",ui.Number(m))
		return e
	}
}

func(i inputModifier) MinLength(m int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("minlength",ui.Number(m))
		return e
	}
}

func(i inputModifier) Autocomplete(b bool) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("autocomplete",ui.Bool(b))
		return e
	}
}

func(i inputModifier) InputMode(mode string)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("inputmode",ui.String(mode))
		return e
	}
}

func(i inputModifier) Size(s int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("size",ui.Number(s))
		return e
	}
}

func(i inputModifier) Placeholder(p string)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("placeholder",ui.String(p))
		return e
	}
}

func(i inputModifier)Pattern(p string)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("pattern",ui.String(p))
		return e
	}
}

func(i inputModifier) Multiple() func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("multiple",ui.Bool(true))
		return e
	}
}

func(i inputModifier) Required(b bool) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("required",ui.Bool(b))
		return e
	}
}

func(i inputModifier) Accept(accept string)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("accept",ui.String(accept))
		return e
	}
}

func(i inputModifier) Src(src string)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("src",ui.String(src))
		return e
	}
}

func(i inputModifier)Alt(alt string)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("alt",ui.String(alt))
		return e
	}
}

func(i inputModifier) Name(name string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("name",ui.String(name))
		return e
	}
}

func(i inputModifier) Height(h int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("height",ui.Number(h))
		return e
	}
}


func(i inputModifier) Width(w int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("width",ui.Number(w))
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
	i.AsElement().SetUI("disabled",ui.Bool(b))
	return i
}


var newInput= Elements.NewConstructor("input", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "input"
	ConnectNative(e, tag)

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

		e.SetUI("type",ui.String(name))		

		return e
	})
}


type inputConstructor func(typ string) InputElement
func(c inputConstructor) WithID(id string, typ string, options ...string) InputElement{
	if typ != ""{
		options = append(options, typ)
	}
	return InputElement{newInput(id, options...)}
}

// OutputElement
type OutputElement struct{
	*ui.Element
}

type outputModifier struct{}
var OutputModifer outputModifier

func(m outputModifier) Form(form *ui.Element) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			d:= evt.Origin().Root
			
			evt.Origin().WatchEvent("navigation-end",d,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
				if form.Mounted(){
					e.SetUI("form", ui.String(form.ID))
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
		e.SetUI("name",ui.String(name))
		return e
	}
}

func(m outputModifier) For(inputs ...*ui.Element) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		var inputlist string
		e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			d:= evt.Origin().Root
			
			evt.Origin().WatchEvent("navigation-end",d,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
				
				for _,input:= range inputs{
					if input.Mounted(){
						inputlist = strings.Join([]string{inputlist, input.ID}, " ")
					} else{
						panic("input missing for output element "+ e.ID)
					}
				}
				e.SetUI("for",ui.String(inputlist))
				return false
			}).RunOnce())
			return false
		}).RunOnce())
		
		return e
	}
}


var newOutput = Elements.NewConstructor("output", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "output"
	ConnectNative(e, tag)

	SetAttribute(e, "id", id) // TODO define attribute setters optional functions
	withStringAttributeWatcher(e,"form")
	withStringAttributeWatcher(e,"name")
	withBoolAttributeWatcher(e,"disabled")

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type outputConstructor func() OutputElement
func(c outputConstructor) WithID(id string, options ...string)OutputElement{
	return OutputElement{newOutput(id, options...)}
}

// ImgElement
type ImgElement struct {
	*ui.Element
}

type imgModifier struct{}
var ImgModifier imgModifier

func(i imgModifier) Src(src string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("src",ui.String(src))
		return e
	}
}


func (i imgModifier) Alt(s string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("alt",ui.String(s))
		return e
	}
}

var newImg= Elements.NewConstructor("img", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "img"
	ConnectNative(e, tag)

	withStringAttributeWatcher(e,"src")
	withStringAttributeWatcher(e,"alt")

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type imgConstructor func() ImgElement
func(c imgConstructor) WithID(id string, options ...string)ImgElement{
	return ImgElement{newImg(id, options...)}
}



type jsTimeRanges ui.Object

func(j jsTimeRanges) Start(index int) time.Duration{
	ti,ok:= ui.Object(j).Get("start")
	if !ok{
		panic("Bad timeRange encoding. No start found")
	}
	ranges := ti.(ui.List)
	if index >= len(ranges.UnsafelyUnwrap()){
		panic("no time ramge at index, index out of bounds")
	}
	return time.Duration(ranges.Get(index).(ui.Number)) *time.Second
}

func(j jsTimeRanges) End(index int) time.Duration{
	ti,ok:= ui.Object(j).Get("end")
	if !ok{
		panic("Bad timeRange encoding. No start found")
	}
	ranges := ti.(ui.List)
	if index >= len(ranges.UnsafelyUnwrap()){
		panic("no time ramge at index, index out of bounds")
	}
	return time.Duration(ranges.Get(index).(ui.Number)) *time.Second
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
	*ui.Element
}

type audioModifier struct{}
var AudioModifier audioModifier

func(m audioModifier) Autoplay(b bool)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("autoplay", ui.Bool(b))
		return e
	}
}

func(m audioModifier) Controls(b bool)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("controls", ui.Bool(b))
		return e
	}
}

func(m audioModifier) CrossOrigin(option string)func(*ui.Element)*ui.Element{
	mod:= ui.String("anonymous")
	if option == "use-credentials"{
		mod = ui.String(option)
	}
	return func(e *ui.Element)*ui.Element{
		e.SetUI("crossorigin", mod)
		return e
	}
}

func(m audioModifier) Loop(b bool)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("loop", ui.Bool(b))
		return e
	}
}

func(m audioModifier) Muted(b bool)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("muted", ui.Bool(b))
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
		e.SetUI("preload", mod)
		return e
	}
}

func(m audioModifier) Src(src string)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("src", ui.String(src))
		return e
	}
}

func(m audioModifier) CurrentTime(t float64)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("currentTime", ui.Number(t))
		return e
	}
}


func(m audioModifier) PlayBackRate(r float64)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("playbackRate", ui.Number(r))
		return e
	}
}

func(m audioModifier) Volume(v float64)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("volume", ui.Number(v))
		return e
	}
}

func(m audioModifier) PreservesPitch(b bool)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("preservesPitch", ui.Bool(b))
		return e
	}
}

func(m audioModifier) DisableRemotePlayback(b bool)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("disableRemotePlayback", ui.Bool(b))
		return e
	}
}

var newAudio = Elements.NewConstructor("audio", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "audio"
	ConnectNative(e, tag)

	withStringAttributeWatcher(e,"src")
	withStringAttributeWatcher(e,"preload")
	withBoolAttributeWatcher(e,"muted")
	withBoolAttributeWatcher(e,"loop")
	withStringAttributeWatcher(e,"crossorigin")
	withBoolAttributeWatcher(e,"controls")
	withBoolAttributeWatcher(e,"autoplay")

	withMediaElementPropertyWatchers(e)

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type audioConstructor func() AudioElement
func(c audioConstructor) WithID(id string, options ...string)AudioElement{
	return AudioElement{newAudio(id, options...)}
}

// VideoElement
type VideoElement struct{
	*ui.Element
}

type videoModifier struct{}
var VideoModifier videoModifier

func(m videoModifier) Height(h float64) func(*ui.Element)*ui.Element{
	return func(e *ui.Element) *ui.Element{
		e.SetUI("height", ui.Number(h))
		return e
	}
}

func(m videoModifier) Width(w float64) func(*ui.Element)*ui.Element{
	return func(e *ui.Element) *ui.Element{
		e.SetUI("width", ui.Number(w))
		return e
	}
}

func(m videoModifier) Poster(url string)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("poster", ui.String(url))
		return e
	}
}


func(m videoModifier) PlaysInline(b bool)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("playsinline", ui.Bool(b))
		return e
	}
}

func(m videoModifier) Controls(b bool)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("controls", ui.Bool(b))
		return e
	}
}

func(m videoModifier) CrossOrigin(option string)func(*ui.Element)*ui.Element{
	mod:= ui.String("anonymous")
	if option == "use-credentials"{
		mod = ui.String(option)
	}
	return func(e *ui.Element)*ui.Element{
		e.SetUI("crossorigin", mod)
		return e
	}
}

func(m videoModifier) Loop(b bool)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("loop", ui.Bool(b))
		return e
	}
}

func(m videoModifier) Muted(b bool)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("muted", ui.Bool(b))
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
		e.SetUI("preload", mod)
		return e
	}
}

func(m videoModifier) Src(src string)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("src", ui.String(src))
		return e
	}
}

func(m videoModifier) CurrentTime(t float64)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("currentTime", ui.Number(t))
		return e
	}
}

func(m videoModifier) DefaultPlayBackRate(r float64)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("defaultPlaybackRate", ui.Number(r))
		return e
	}
}

func(m videoModifier) PlayBackRate(r float64)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("playbackRate", ui.Number(r))
		return e
	}
}

func(m videoModifier) Volume(v float64)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("volume", ui.Number(v))
		return e
	}
}

func(m videoModifier) PreservesPitch(b bool)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("preservesPitch", ui.Bool(b))
		return e
	}
}

var newVideo = Elements.NewConstructor("video", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "video"
	ConnectNative(e, tag)

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
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type videoConstructor func() VideoElement
func(c videoConstructor) WithID(id string, options ...string)VideoElement{
	return VideoElement{newVideo(id, options...)}
}

// SourceElement
type SourceElement struct{
	*ui.Element
}

type sourceModifier struct{}
var SourceModifier sourceModifier

func(s sourceModifier) Src(src string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("src",ui.String(src))
		return e
	}
}


func(s sourceModifier) Type(typ string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("type",ui.String(typ))
		return e
	}
}


var newSource = Elements.NewConstructor("source", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "source"
	ConnectNative(e, tag)

	withStringAttributeWatcher(e,"src")
	withStringAttributeWatcher(e,"type")

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type sourceConstructor func() SourceElement
func(c sourceConstructor) WithID(id string, options ...string)SourceElement{
	return SourceElement{newSource(id, options...)}
}

type UlElement struct {
	*ui.Element
}

func (l UlElement) FromValues(values ...ui.Value) UlElement {
	l.AsElement().SetUI("list", ui.NewList(values...).Commit())
	return l
}

func (l UlElement) Values() ui.List {
	v, ok := l.AsElement().GetData("list")
	if !ok {
		return ui.NewList().Commit()
	}
	list, ok := v.(ui.List)
	if !ok {
		panic("data/list got overwritten with wrong type or something bad has happened")
	}
	return list
}

var newUl= Elements.NewConstructor("ul", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "ul"
	ConnectNative(e, tag)

	return e
}, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type ulConstructor func() UlElement
func(c ulConstructor) WithID(id string, options ...string)UlElement{
	return UlElement{newUl(id, options...)}
}

type OlElement struct {
	*ui.Element
}

func (l OlElement) SetValue(lobjs ui.List) OlElement {
	l.AsElement().Set("data", "value", lobjs)
	return l
}

var newOl= Elements.NewConstructor("ol", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "ol"
	ConnectNative(e, tag)
	
	return e
}, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)


type olConstructor func(typ string, offset int) OlElement
func(c olConstructor) WithID(id string) func(typ string, offset int, options ...string)OlElement{
	return func(typ string, offset int, options ...string) OlElement {
		e:= newOl(id, options...)
		SetAttribute(e, "type", typ)
		SetAttribute(e, "start", strconv.Itoa(offset))
		return OlElement{e}
	}
}

type LiElement struct {
	*ui.Element
}

func(li LiElement) SetElement(e *ui.Element) LiElement{ // TODO Might be unnecessary in which case remove
	li.AsElement().SetChildren(e)
	return li
}

var newLi= Elements.NewConstructor("li", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "li"
	ConnectNative(e, tag)
	
	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type liConstructor func() LiElement
func(c liConstructor) WithID(id string, options ...string)LiElement{
	return LiElement{newLi(id, options...)}
}

// Table Elements

// TableElement
type TableElement struct {
	*ui.Element
}

// TheadElement
type TheadElement struct {
	*ui.Element
}

// TbodyElement
type TbodyElement struct {
	*ui.Element
}

// TrElement
type TrElement struct {
	*ui.Element
}


// TdElement
type TdElement struct {
	*ui.Element
}


// ThElement
type ThElement struct {
	*ui.Element
}

// ColElement
type ColElement struct {
	*ui.Element
}

func(c ColElement) SetSpan(n int) ColElement{
	c.AsElement().SetUI("span",ui.Number(n))
	return c
}

// ColGroupElement
type ColGroupElement struct {
	*ui.Element
}

func(c ColGroupElement) SetSpan(n int) ColGroupElement{
	c.AsElement().SetUI("span",ui.Number(n))
	return c
}

// TfootElement
type TfootElement struct {
	*ui.Element
}

var newThead= Elements.NewConstructor("thead", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "thead"
	ConnectNative(e, tag)

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type theadConstructor func() TheadElement
func(c theadConstructor) WithID(id string, options ...string)TheadElement{
	return TheadElement{newThead(id, options...)}
}


var newTr= Elements.NewConstructor("tr", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "tr"
	ConnectNative(e, tag)

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type trConstructor func() TrElement
func(c trConstructor) WithID(id string, options ...string)TrElement{
	return TrElement{newTr(id, options...)}
}

var newTd= Elements.NewConstructor("td", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "td"
	ConnectNative(e, tag)

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type tdConstructor func() TdElement
func(c tdConstructor) WithID(id string, options ...string)TdElement{
	return TdElement{newTd(id, options...)}
}

var newTh= Elements.NewConstructor("th", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "th"
	ConnectNative(e, tag)

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type thConstructor func() ThElement
func(c thConstructor) WithID(id string, options ...string)ThElement{
	return ThElement{newTh(id, options...)}
}

var newTbody= Elements.NewConstructor("tbody", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "tbody"
	ConnectNative(e, tag)

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type tbodyConstructor func() TbodyElement
func(c tbodyConstructor) WithID(id string, options ...string)TbodyElement{
	return TbodyElement{newTbody(id, options...)}
}

var newTfoot= Elements.NewConstructor("tfoot", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "tfoot"
	ConnectNative(e, tag)

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type tfootConstructor func() TfootElement
func(c tfootConstructor) WithID(id string, options ...string)TfootElement{
	return TfootElement{newTfoot(id, options...)}
}

var newCol= Elements.NewConstructor("col", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "col"
	ConnectNative(e, tag)

	withNumberAttributeWatcher(e,"span")

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type colConstructor func() ColElement
func(c colConstructor) WithID(id string, options ...string)ColElement{
	return ColElement{newCol(id, options...)}
}

var newColGroup= Elements.NewConstructor("colgroup", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "colgroup"
	ConnectNative(e, tag)

	withNumberAttributeWatcher(e,"span")

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type colgroupConstructor func() ColGroupElement
func(c colgroupConstructor) WithID(id string, options ...string)ColGroupElement{
	return ColGroupElement{newColGroup(id, options...)}
}

var newTable= Elements.NewConstructor("table", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "table"
	ConnectNative(e, tag)

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type tableConstructor func() TableElement
func(c tableConstructor) WithID(id string, options ...string)TableElement{
	return TableElement{newTable(id, options...)}
}


type CanvasElement struct{
	*ui.Element
}

type canvasModifier struct{}
var CanvasModifier = canvasModifier{}

func(c canvasModifier) Height(h int)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("height",ui.Number(h))
		return e
	}
}


func(c canvasModifier) Width(w int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("width",ui.Number(w))
		return e
	}
}

var newCanvas = Elements.NewConstructor("canvas",func(id string)*ui.Element{
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "canvas"
	ConnectNative(e, tag)

	withNumberAttributeWatcher(e,"height")
	withNumberAttributeWatcher(e,"width")

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type canvasConstructor func() CanvasElement
func(c canvasConstructor) WithID(id string, options ...string)CanvasElement{
	return CanvasElement{newCanvas(id, options...)}
}

type SvgElement struct{
	*ui.Element
}

type svgModifier struct{}
var SvgModifer svgModifier

func(s svgModifier) Height(h int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("height",ui.Number(h))
		return e
	}
}

func(s svgModifier) Width(w int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("width",ui.Number(w))
		return e
	}
}

func(s svgModifier) Viewbox(attr string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("viewbox",ui.String(attr))
		return e
	}
}


func(s svgModifier) X(x string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("x",ui.String(x))
		return e
	}
}


func(s svgModifier) Y(y string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("y",ui.String(y))
		return e
	}
}

var newSvg = Elements.NewConstructor("svg",func(id string)*ui.Element{
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "svg"
	ConnectNative(e, tag)

	withNumberAttributeWatcher(e,"height")
	withNumberAttributeWatcher(e,"width")
	withStringAttributeWatcher(e,"viewbox")
	withStringAttributeWatcher(e,"x")
	withStringAttributeWatcher(e,"y")

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type svgConstructor func() SvgElement
func(c svgConstructor) WithID(id string, options ...string)SvgElement{
	return SvgElement{newSvg(id, options...)}
}

type SummaryElement struct{
	*ui.Element
}

func (s SummaryElement) SetText(str string) SummaryElement {
	s.AsElement().SetUI("text", ui.String(str))
	return s
}

var newSummary = Elements.NewConstructor("summary", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "summary"
	ConnectNative(e, tag)

	e.Watch("ui", "text", e, textContentHandler)

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type summaryConstructor func() SummaryElement
func(c summaryConstructor) WithID(id string, options ...string)SummaryElement{
	return SummaryElement{newSummary(id, options...)}
}

type DetailsElement struct{
	*ui.Element
}

func (d DetailsElement) SetText(str string) DetailsElement {
	d.AsElement().SetUI("text", ui.String(str))
	return d
}

func(d DetailsElement) Open() DetailsElement{
	d.AsElement().SetUI("open",ui.Bool(true))
	return d
}

func(d DetailsElement) Close() DetailsElement{
	d.AsElement().SetUI("open",nil)
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
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "details"
	ConnectNative(e, tag)

	e.Watch("ui", "text", e, textContentHandler)
	withBoolAttributeWatcher(e,"open")

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type detailsConstructor func() DetailsElement
func(c detailsConstructor) WithID(id string, options ...string)DetailsElement{
	return DetailsElement{newDetails(id, options...)}
}

// Dialog
type DialogElement struct{
	*ui.Element
}


func(d DialogElement) Open() DialogElement{
	d.AsElement().SetUI("open",ui.Bool(true))
	return d
}

func(d DialogElement) Close() DialogElement{
	d.AsElement().SetUI("open",nil)
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
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "dialog"
	ConnectNative(e, tag)

	withBoolAttributeWatcher(e,"open")

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type dialogConstructor func() DialogElement
func(c dialogConstructor) WithID(id string, options ...string)DialogElement{
	return DialogElement{newDialog(id, options...)}
}

// CodeElement is typically used to indicate that the text it contains is computer code and may therefore be 
// formatted differently.
// To represent multiple lines of code, wrap the <code> element within a <pre> element. 
// The <code> element by itself only represents a single phrase of code or line of code.
type CodeElement struct {
	*ui.Element
}

func (c CodeElement) SetText(str string) CodeElement {
	c.AsElement().SetUI("text", ui.String(str))
	return c
}

var newCode= Elements.NewConstructor("code", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "code"
	ConnectNative(e, tag)

	e.Watch("ui", "text", e, textContentHandler)

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type codeConstructor func() CodeElement
func(c codeConstructor) WithID(id string, options ...string)CodeElement{
	return CodeElement{newCode(id, options...)}
}

// Embed
type EmbedElement struct{
	*ui.Element
}

func(e EmbedElement) SetHeight(h int) EmbedElement{
	e.AsElement().SetUI("height",ui.Number(h))
	return e
}

func(e EmbedElement) SetWidth(w int) EmbedElement{
	e.AsElement().SetUI("width",ui.Number(w))
	return e
}

func(e EmbedElement) SetType(typ string) EmbedElement{
	e.AsElement().SetUI("type", ui.String(typ))
	return e
}

func(e EmbedElement) SetSrc(src string) EmbedElement{
	e.AsElement().SetUI("src", ui.String(src))
	return e
}


var newEmbed = Elements.NewConstructor("embed",func(id string)*ui.Element{
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "embed"
	ConnectNative(e, tag)

	withNumberAttributeWatcher(e,"height")
	withNumberAttributeWatcher(e,"width")
	withStringAttributeWatcher(e,"type")
	withStringAttributeWatcher(e,"src")

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type embedConstructor func() EmbedElement
func(c embedConstructor) WithID(id string, options ...string)EmbedElement{
	return EmbedElement{newEmbed(id, options...)}
}

// Object
type ObjectElement struct{
	*ui.Element
}

type objectModifier struct{}
var ObjectModifier = objectModifier{}

func(o objectModifier) Height(h int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("height",ui.Number(h))
		return e
	}
}

func(o objectModifier) Width(w int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("width",ui.Number(w))
		return e
	}
}


func(o objectModifier) Type(typ string)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("type", ui.String(typ))
		return e
	}
}

// Data sets the path to the resource.
func(o objectModifier) Data(u url.URL)func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("data", ui.String(u.String()))
		return e
	}
}
func (o objectModifier) Form(form *ui.Element) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			d:= evt.Origin().Root
			
			evt.Origin().WatchEvent("navigation-end",d,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
				if form.Mounted(){
					e.AsElement().SetUI("form", ui.String(form.ID))
				}
				return false
			}).RunOnce())
			return false
		}).RunOnce())
		return e
	}
}


var newObject = Elements.NewConstructor("object",func(id string)*ui.Element{
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "object"
	ConnectNative(e, tag)

	withNumberAttributeWatcher(e,"height")
	withNumberAttributeWatcher(e,"width")
	withStringAttributeWatcher(e,"type")
	withStringAttributeWatcher(e,"data")
	withStringAttributeWatcher(e,"form")

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type objectConstructor func() ObjectElement
func(c objectConstructor) WithID(id string, options ...string)ObjectElement{
	return ObjectElement{newObject(id, options...)}
}

// Datalist
type DatalistElement struct{
	*ui.Element
}

var newDatalist = Elements.NewConstructor("datalist", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "datalist"
	ConnectNative(e, tag)

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type datalistConstructor func() DatalistElement
func(c datalistConstructor) WithID(id string, options ...string)DatalistElement{
	return DatalistElement{newDatalist(id, options...)}
}

// OptionElement
type OptionElement struct{
	*ui.Element
}

type optionModifier struct{}
var OptionModifer optionModifier

func(o optionModifier) Label(l string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("label",ui.String(l))
		return e
	}
}

func(o optionModifier) Value(value string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("value",ui.String(value))
		return e
	}
}

func(o optionModifier) Disabled(b bool) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("disabled",ui.Bool(b))
		return e
	}
}

func(o optionModifier) Selected() func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("selected",ui.Bool(true))
		return e
	}
}

func(o OptionElement) SetValue(opt string) OptionElement{
	o.AsElement().SetUI("value", ui.String(opt))
	return o
}

var newOption = Elements.NewConstructor("option", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "option"
	ConnectNative(e, tag)

	withStringAttributeWatcher(e,"value")
	withStringAttributeWatcher(e,"label")
	withBoolAttributeWatcher(e,"disabled")
	withBoolAttributeWatcher(e,"selected")

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type optionConstructor func() OptionElement
func(c optionConstructor) WithID(id string, options ...string)OptionElement{
	return OptionElement{newOption(id, options...)}
}

// OptgroupElement
type OptgroupElement struct{
	*ui.Element
}

type optgroupModifier struct{}
var OptgroupModifer optionModifier

func(o optgroupModifier) Label(l string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("label",ui.String(l))
		return e
	}
}


func(o optgroupModifier) Disabled(b bool) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("disabled",ui.Bool(b))
		return e
	}
}


func(o OptgroupElement) SetLabel(opt string) OptgroupElement{
	o.AsElement().SetUI("label", ui.String(opt))
	return o
}

func(o OptgroupElement) SetDisabled(b bool) OptgroupElement{
	o.AsElement().SetUI("disabled",ui.Bool(b))
	return o
}

var newOptgroup = Elements.NewConstructor("optgroup", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "optgroup"
	ConnectNative(e, tag)

	withStringAttributeWatcher(e,"label")
	withBoolAttributeWatcher(e,"disabled")

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type optgroupConstructor func() OptgroupElement
func(c optgroupConstructor) WithID(id string, options ...string)OptgroupElement{
	return OptgroupElement{newOptgroup(id, options...)}
}

// FieldsetElement
type FieldsetElement struct{
	*ui.Element
}

type fieldsetModifier struct{}
var FieldsetModifer fieldsetModifier

func(m fieldsetModifier) Form(form *ui.Element) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			d:= evt.Origin().Root
			
			evt.Origin().WatchEvent("navigation-end",d,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
				if form.Mounted(){
					e.SetUI("form", ui.String(form.ID))
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
		e.SetUI("name",ui.String(name))
		return e
	}
}

func(m fieldsetModifier) Disabled(b bool) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("disabled",ui.Bool(b))
		return e
	}
}


var newFieldset = Elements.NewConstructor("fieldset", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "fieldset"
	ConnectNative(e, tag)

	withStringAttributeWatcher(e,"form")
	withStringAttributeWatcher(e,"name")
	withBoolAttributeWatcher(e,"disabled")

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type fieldsetConstructor func() FieldsetElement
func(c fieldsetConstructor) WithID(id string, options ...string)FieldsetElement{
	return FieldsetElement{newFieldset(id, options...)}
}

// LegendElement
type LegendElement struct{
	*ui.Element
}

func(l LegendElement) SetText(s string) LegendElement{
	l.AsElement().SetUI("text",ui.String(s))
	return l
}

var newLegend = Elements.NewConstructor("legend", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "legend"
	ConnectNative(e, tag)

	e.Watch("ui", "text", e, textContentHandler)

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type legendConstructor func() LegendElement
func(c legendConstructor) WithID(id string, options ...string)LegendElement{
	return LegendElement{newLegend(id, options...)}
}

// ProgressElement
type ProgressElement struct{
	*ui.Element
}

func(p ProgressElement) SetMax(m float64) ProgressElement{
	if m>0{
		p.AsElement().SetUI("max", ui.Number(m))
	}
	
	return p
}

func(p ProgressElement) SetValue(v float64) ProgressElement{
	if v>0{
		p.AsElement().SetUI("value", ui.Number(v))
	}
	
	return p
}

var newProgress = Elements.NewConstructor("progress", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "progress"
	ConnectNative(e, tag)

	withNumberAttributeWatcher(e,"max")
	withNumberAttributeWatcher(e,"value")

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type progressConstructor func() ProgressElement
func(c progressConstructor) WithID(id string, options ...string)ProgressElement{
	return ProgressElement{newProgress(id, options...)}
}

// SelectElement
type SelectElement struct{
	*ui.Element
}

type selectModifier struct{}
var SelectModifier selectModifier

func(m selectModifier) Autocomplete(b bool) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("autocomplete",ui.Bool(b))
		return e
	}
}

func(m selectModifier) Size(s int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("size",ui.Number(s))
		return e
	}
}

func(m selectModifier) Disabled(b bool) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("disabled",ui.Bool(b))
		return e
	}
}

func (m selectModifier) Form(form *ui.Element) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			d:= evt.Origin().Root
			
			evt.Origin().WatchEvent("navigation-end",d,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
				if form.Mounted(){
					e.AsElement().SetUI("form", ui.String(form.ID))
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
		e.SetUI("required",ui.Bool(b))
		return e
	}
}

func(m selectModifier) Name(name string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("name",ui.String(name))
		return e
	}
}


var newSelect = Elements.NewConstructor("select", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "select"
	ConnectNative(e, tag)

	withStringAttributeWatcher(e,"form")
	withStringAttributeWatcher(e,"name")
	withBoolAttributeWatcher(e,"disabled")
	withBoolAttributeWatcher(e,"required")
	withBoolAttributeWatcher(e,"multiple")
	withNumberAttributeWatcher(e,"size")
	withStringAttributeWatcher(e,"autocomplete")


	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)



type selectConstructor func() SelectElement
func(c selectConstructor) WithID(id string, options ...string)SelectElement{
	return SelectElement{newSelect(id, options...)}
}


// FormElement
type FormElement struct{
	*ui.Element
}

type formModifier struct{}
var FormModifer = formModifier{}

func(f formModifier) Name(name string) func(*ui.Element) *ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("name",ui.String(name))
		return e
	}
}

func(f formModifier) Method(methodname string) func(*ui.Element) *ui.Element{
	m:=  "GET"
	if  strings.EqualFold(methodname,"POST"){
		m = "POST"
	}
	return func(e *ui.Element)*ui.Element{
		e.SetUI("method",ui.String(m))
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
		e.SetUI("target",ui.String(m))
		return e
	}
}

func(f formModifier) Action(u url.URL) func(*ui.Element) *ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("action",ui.String(u.String()))
		return e
	}
}

func(f formModifier) Autocomplete() func(*ui.Element) *ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("autocomplete",ui.Bool(true))
		return e
	}
}

func(f formModifier) NoValidate() func(*ui.Element) *ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("novalidate",ui.Bool(true))
		return e
	}
}

func(f formModifier) EncType(enctype string) func(*ui.Element) *ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("enctype",ui.String(enctype))
		return e
	}
}

func(f formModifier) Charset(charset string) func(*ui.Element) *ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.SetUI("accept-charset",ui.String(charset))
		return e
	}
}

var newForm= Elements.NewConstructor("form", func(id string) *ui.Element {
	
	e := Elements.NewElement(id)
	e = enableClasses(e)

	tag:= "form"
	ConnectNative(e, tag)

	e.OnMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		_,ok:= e.Get("ui","action")
		if !ok{
			evt.Origin().SetUI("action",ui.String(evt.Origin().Route()))
		}
		return false
	}).RunOnce().RunASAP())


	withStringAttributeWatcher(e,"accept-charset")
	withBoolAttributeWatcher(e,"autocomplete")
	withStringAttributeWatcher(e,"name")
	withStringAttributeWatcher(e,"action")
	withStringAttributeWatcher(e,"enctype")
	withStringAttributeWatcher(e,"method")
	withBoolAttributeWatcher(e,"novalidate")
	withStringAttributeWatcher(e,"target")

	return e
},AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)


type formConstructor func() FormElement
func(c formConstructor) WithID(id string, options ...string)FormElement{
	return FormElement{newForm(id, options...)}
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
	}).RunOnce())

	withStringPropertyWatcher(e,attr) // IDL attribute support
}


// watches ("ui",attr) for a ui.Number value.
func withNumberAttributeWatcher(e *ui.Element,attr string){
	e.Watch("ui",attr,e,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		SetAttribute(evt.Origin(),attr,strconv.Itoa(int(evt.NewValue().(ui.Number))))
		return false
	}).RunOnce())
	withNumberPropertyWatcher(e,attr) // IDL attribute support
}

// watches ("ui",attr) for a ui.Bool value.
func withBoolAttributeWatcher(e *ui.Element, attr string){
	if !inBrowser(){
		e.Watch("ui", attr, e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			if evt.NewValue().(ui.Bool) {
				SetAttribute(evt.Origin(), attr, "")
				return false
			}
			RemoveAttribute(evt.Origin(), attr)
			return false
		}))
	}
	
	withBoolPropertyWatcher(e,attr) // IDL attribute support
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
		e.SetUI(name,ui.String(value))
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