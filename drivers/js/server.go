//go:build server


package doc

import (
	"io"
	"strings"
	"net/http"
	"net/url"
	"net/http/cookiejar"
	"os"
	"time"
	"sync"
	"path/filepath"
	"log"

	"github.com/atdiar/particleui"

	"golang.org/x/net/html"
)



var (
	// DOCTYPE holds the document doctype.
	DOCTYPE = "html"
	// Elements stores wasm-generated HTML ui.Element constructors.
	Elements = ui.NewElementStore("default", DOCTYPE)
	mu *sync.Mutex

	DefaultPattern = "/"
	StaticPath = "assets"


	ServeMux *http.ServeMux
	Server *http.Server = &http.Server{Addr:":8080",Handler:ServeMux}
	
	HTMLhandlerModifier func(http.Handler)http.Handler
	renderHTMLhandler http.Handler


	httpHandler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		mu.Lock()
		defer mu.Unlock()
		// Cookie handling
		cj, err := cookiejar.New(nil)
		if err != nil{
			http.Error(w,"Error creating cookiejar",http.StatusInternalServerError)
			return
		}
		ui.CookieJar = cj
		ui.HttpClient.Jar = ui.CookieJar
		cj.SetCookies(r.URL, r.Cookies())



		h:= renderHTMLhandler
		if HTMLhandlerModifier != nil{
			h=HTMLhandlerModifier(h)
		}

		w = cookiejarWriter{w,r.URL}
		h.ServeHTTP(w,r)

	})
)

type cookiejarWriter struct{
	http.ResponseWriter
	URL *url.URL
}

func(c cookiejarWriter) Write(b []byte) (int,error){
	cookies:= ui.CookieJar.Cookies(c.URL)
	for _,cookie:= range cookies{
		http.SetCookie(c.ResponseWriter,cookie)
	}
	return c.ResponseWriter.Write(b)
}
/*
 Server-side HTML rendering TODO place behind compile directive

*/


func ChangeServeMux(s *http.ServeMux) {
	if s == nil{
		s = http.NewServeMux()
	}
	s.Handle(DefaultPattern,httpHandler)
}



// ChangeServer changes the http.Server
// It also registers doc.ServeMux as a http request handler.
func ChangeServer(s *http.Server){
	s.Handler = ServeMux
	Server = s
}


// NewBuilder registers a new document building function. This function should not be called
// manually afterwards.
func NewBuilder(f func()Document){
	fileServer := http.FileServer(http.Dir(StaticPath))

	renderHTMLhandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		path, err := filepath.Abs(r.URL.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		path = filepath.Join(StaticPath, r.URL.Path)

		fi, err := os.Stat(path)
		if err == nil && !fi.IsDir(){
			fileServer.ServeHTTP(w,r)
			return
		}


		if d:=GetDocument(); d != nil{
			d.Delete()
		}
		document:= f()
		router:= ui.GetRouter()
		route:= r.URL.Path
		_,routeexist:= router.Match(route)
		if routeexist != nil{
			w.WriteHeader(http.StatusNotFound)
		}

		router.GoTo(route)
		err= document.Render(w)
		if err != nil{ 
			switch err{
			case ui.ErrNotFound:
				w.WriteHeader(http.StatusNotFound)
			case ui.ErrFrameworkFailure:
				w.WriteHeader(http.StatusInternalServerError)
			case ui.ErrUnauthorized:
				w.WriteHeader(http.StatusUnauthorized)
			}
		}		
	})


	// TODO reset global state i.e. ElementStore and Document's BuildOption ?
	ListenAndServe = func(){
		log.Print("Listening on: "+Server.Addr)
		if Server.TLSConfig == nil{
			Server.ListenAndServe()
		} else{
			Server.ListenAndServeTLS("","")
		}		
	}
	
}


var windowTitleHandler = ui.NewMutationHandler(func(evt ui.MutationEvent) bool { // abstractjs
	// TODO need to set the document title somehow (set the relevant attribute)

	return false
})

var historyMutationHandler = ui.NewMutationHandler(func(evt ui.MutationEvent)bool{ // abstractjs
	return false
})

var navreadyHandler =  ui.NewMutationHandler(func(evt ui.MutationEvent) bool {// abstractjs
	e:= evt.Origin()
	e.ElementStore.MutationCapture = true
	e.Watch("internals","lastmutation",e.ElementStore.Global,ui.NewMutationHandler(func(event ui.MutationEvent)bool{
		// TODO if mutation is not fetch type event, append it to internals,globalstatehistory
		var history ui.List
		hl,ok:= e.Get("internals","globalstatehistory")
		if !ok{
			history = ui.NewList()
		}
		history = hl.(ui.List)
		history = append(history,event.NewValue())
		e.Set("internals","globalstatehistory",history)
		return false
	}))

	return false
})



// SetInnerHTML sets the innerHTML property of HTML elements.
// Please note that it is unsafe to sets client submittd HTML inputs.
func SetInnerHTML(e *ui.Element, html string) *ui.Element {
	// TODO
	return e
} // abstractjs

// LoadFromStorage will load an element properties.
// If the corresponding native DOM Element is marked for hydration, by the presence of a data-hydrate
// atribute, the props are loaded from this attribute instead.
// abstractjs
func LoadFromStorage(e *ui.Element) *ui.Element {
	return e
}

// PutInStorage stores an element properties in storage (localstorage or sessionstorage).
func PutInStorage(e *ui.Element) *ui.Element{
	return e
}

// ClearFromStorage will clear an element properties from storage.
func ClearFromStorage(e *ui.Element) *ui.Element{
	return e
}

func isPersisted(e *ui.Element) bool{
	return false
}

func NewNativeElementIfAbsent(id string, tag string) (ui.NativeElement,bool){
	if tag == "window"{
		return  NewNativeElementWrapper(nil), true
	}

	if tag == "html"{
		return NewNativeElementWrapper(nil), true
	}

	if tag == "body"{
		n := &html.Node{}
		n.Type = html.RawNode
		n.Data = tag

		return NewNativeElementWrapper(n), true
	}

	if tag == "head"{
		n := &html.Node{}
		n.Type = html.RawNode
		n.Data = tag

		return NewNativeElementWrapper(n), true
	}

	n := &html.Node{}
	n.Type = html.RawNode
	n.Data = tag

	return NewNativeElementWrapper(n), true
}

// NativeElement defines a wrapper around a *html.Node that implements the
// ui.NativeElementWrapper interface.
type NativeElement struct {
	Value *html.Node
}

func NewNativeElementWrapper(n *html.Node) NativeElement {
	return NativeElement{n}
}

func (n NativeElement) AppendChild(child *ui.Element) {
	c:= child.Native.(NativeElement).Value
	n.Value.AppendChild(c)
}

func (n NativeElement) PrependChild(child *ui.Element) {
	c:= child.Native.(NativeElement).Value
	n.Value.InsertBefore(c, n.Value.FirstChild)
	
}

func (n NativeElement) InsertChild(child *ui.Element, index int) {
	if index < 0{
		panic("index must be a positive integer")
	}
	if n.Value.FirstChild == nil{
		n.AppendChild(child)
		return
	}

	var currentAtIndex = n.Value.FirstChild
	var idx int
	
	
	for i:= 0; i<= index;i++{
		if currentAtIndex.NextSibling == nil{
			if i < index{
				currentAtIndex = n.Value.LastChild
				idx = -1
			}
			break
		}
		currentAtIndex = currentAtIndex.NextSibling
		idx++
	}

	if idx == -1{
		n.AppendChild(child)
		return 
	}

	n.Value.InsertBefore(child.Native.(NativeElement).Value, currentAtIndex)

}

func (n NativeElement) ReplaceChild(old *ui.Element, new *ui.Element) {
	oldc:= old.Native.(NativeElement).Value
	newc:= new.Native.(NativeElement).Value
	if oldc.Parent == n.Value {
		n.Value.InsertBefore(newc,oldc)
		n.Value.RemoveChild(oldc)
	}
}

func (n NativeElement) RemoveChild(child *ui.Element) {
	c:= child.Native.(NativeElement).Value
	if c.Parent == n.Value{
		n.Value.RemoveChild(c)
	}
}

func (n NativeElement) SetChildren(children ...*ui.Element) {
	// first we need to delete children if there are any
	var stop bool
	var current = n.Value.FirstChild

	if current != nil{
		for !stop{
			next := current.NextSibling
			if next == nil{
				stop = true
			}
			n.Value.RemoveChild(current)
			current = next
		}
	}

	for _,c:= range children{
		n.AppendChild(c)
	}
}

var AllowScrollRestoration = ui.NewConstructorOption("scrollrestoration", func(e *ui.Element) *ui.Element {
	return e
})

func Focus(e ui.AnyElement, scrollintoview bool){}

func IsInViewPort(e *ui.Element) bool{
	return true
}

func TrapFocus(e *ui.Element) *ui.Element{ return e}

func enableDataBinding(datacapturemode ...mutationCaptureMode) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		return e
	}
}


func (i InputElement) Blur() {}

func (i InputElement) Focus() {}

func (i InputElement) Clear() {}



func newTimeRanges() jsTimeRanges{
	var j = ui.NewObject()

	var length int
	
	j.Set("length",ui.Number(length))

	starts:= ui.NewList()
	ends := ui.NewList()

	j.Set("start",starts)
	j.Set("end",ends)
	return jsTimeRanges(j)
}


func(a AudioElement) Buffered() jsTimeRanges{
	// TODO get from attr ?
	return newTimeRanges()
}

func(a AudioElement)CurrentTime() time.Duration{
	// TODO get from attr ?
	return 0
}

func(a AudioElement)Duration() time.Duration{
	// TODO get from attr ?
	return  0
}

func(a AudioElement)PlayBackRate() float64{
	// TODO get from attr ?
	return 0
}

func(a AudioElement)Ended() bool{
	// TODO get from attr ?
	return false
}

func(a AudioElement)ReadyState() float64{
	// TODO get from attr ?
	return 0
}

func(a AudioElement)Seekable()  jsTimeRanges{
	// TODO get from attr ?
	return newTimeRanges()
}

func(a AudioElement) Volume() float64{
	// TODO get from attr ?
	return  0
}


func(a AudioElement) Muted() bool{
	// TODO get from attr ?
	return false
}

func(a AudioElement) Paused() bool{
	// TODO get from attr ?
	return false
}

func(a AudioElement) Loop() bool{
	// TODO get from attr ?
	return false
}



func(v VideoElement) Buffered() jsTimeRanges{
	// TODO get from attr ?
	return newTimeRanges()
}

func(v VideoElement)CurrentTime() time.Duration{
	// TODO get from attr ?
	return 0
}

func(v VideoElement)Duration() time.Duration{
	// TODO get from attr ?
	return  0
}

func(v VideoElement)PlayBackRate() float64{
	// TODO get from attr ?
	return 0
}

func(v VideoElement)Ended() bool{
	// TODO get from attr ?
	return false
}

func(v VideoElement)ReadyState() float64{
	// TODO get from attr ?
	return 0
}

func(v VideoElement)Seekable()  jsTimeRanges{
	return newTimeRanges()
}

func(v VideoElement) Volume() float64{
	// TODO get from attr ?
	return 0
}


func(v VideoElement) Muted() bool{
	// TODO get from attr ?
	return false
}

func(v VideoElement) Paused() bool{
	// TODO get from attr ?
	return false
}

func(v VideoElement) Loop() bool{
	// TODO get from attr ?
	return false
}



func AddClass(target *ui.Element, classname string) {
	// TODO
}

func RemoveClass(target *ui.Element, classname string) {
	// TODO
}

func Classes(target *ui.Element) []string {
	// TODO
	return nil
}

func enableClasses(e *ui.Element) *ui.Element {
	// TODO
	return e
}

func GetAttribute(target *ui.Element, name string) string {
	// TODO
	return ""
}

// abstractjs
func SetAttribute(target *ui.Element, name string, value string) {
	// TODO
}

// abstractjs
func RemoveAttribute(target *ui.Element, name string) {

}

var textContentHandler = ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
	/*str, ok := evt.NewValue().(ui.String)
	if !ok {
		return true
	}
	JSValue(evt.Origin()).Set("textContent", string(str)) */

	// TODO

	return false
})

var paragraphTextHandler = ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
	//JSValue(evt.Origin()).Set("innerText", string(evt.NewValue().(ui.String)))

	// TODO
	return false
})

func numericPropertyWatcher(propname string) *ui.MutationHandler{
	return ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		// JSValue(evt.Origin()).Set(propname,float64(evt.NewValue().(ui.Number)))
		return false
	})
}

func boolPropertyWatcher(propname string) *ui.MutationHandler{
	return ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		// JSValue(evt.Origin()).Set(propname,bool(evt.NewValue().(ui.Bool)))
		return false
	})
}

func stringPropertyWatcher(propname string) *ui.MutationHandler{
	return ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		// JSValue(evt.Origin()).Set(propname,string(evt.NewValue().(ui.String)))
		return false
	})
}


func clampedValueWatcher(propname string, min int,max int) *ui.MutationHandler{
	return ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		/*v:= float64(evt.NewValue().(ui.Number))
		if v < float64(min){
			v = float64(min)
		}

		if v > float64(max){
			v = float64(max)
		}
		JSValue(evt.Origin()).Set(propname,v)
		*/
		return false
	})
}


// +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

func (d Document) Render(w io.Writer) error {
	return html.Render(w, RenderHTMLTree(d))
}

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
	
	n := &html.Node{}
	n.Type = nodetype
	n.Data = data

	attrs, ok := e.GetData("attrs")
	if !ok {
		return n
	}
	tattrs, ok := attrs.(ui.Object)
	if !ok {
		panic("attributes is supposed to be a ui.Object type")
	}
	for k, v := range tattrs {
		a := html.Attribute{"", k, string(v.(ui.String))}
		n.Attr = append(n.Attr, a)
	}

	
	// Element state should be stored serialized in script Element and hydration attribute should be set
	// on the Document Node
	if e.ID == GetDocument().AsElement().ID{
		n.Attr = append(n.Attr,html.Attribute{"",HydrationAttrName,"true"})
	}
	

	return n
}

func RenderHTMLTree(document Document) *html.Node {
	doc := document.AsBasicElement()
	h:= &html.Node{}
	n:= renderHTMLTree(doc.AsElement(),&h)
	statenode:= generateStateHistoryRecordElement()
	if statenode != nil{
		h.AppendChild(statenode)
	}

	return n
}

func renderHTMLTree(e *ui.Element, pHead **html.Node) *html.Node {
	d := e.Native.(NativeElement)
	v:= d.Value

	if e.ID == GetDocument().Head().AsElement().ID{
		pHead = &v
	}
	d.SetChildren(nil) // removes any child if present

	if e.Children != nil && e.Children.List != nil {
		for _, child := range e.Children.List {
			c:= renderHTMLTree(child, pHead)
			v.AppendChild(c)
		}
	}

	return v
}

func generateStateHistoryRecordElement() *html.Node{
	state:=  SerializeStateHistory()
	script:= `<script id='` + GetDocument().AsElement().ID+SSRStateSuffix+`' type="application/json">
	` + state + `
	<script>`
	scriptNode, err:= html.Parse(strings.NewReader(script))
	if err!= nil{
		panic(err)
	}
	return scriptNode
}

func recoverStateHistory(){}
var recoverStateHistoryHandler = ui.NoopMutationHandler
