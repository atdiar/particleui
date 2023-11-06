//go:build server


package doc

import (
	"bytes"
	"context"
	"io"
	"strings"
	"net"
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
	"golang.org/x/net/html/atom"
)

func init(){
	ui.HttpClient = modifyClient(ui.HttpClient)
}

var (
	// DOCTYPE holds the document doctype.
	DOCTYPE = "html"
	// Elements stores wasm-generated HTML ui.Element constructors.
	Elements = ui.NewElementStore("default", DOCTYPE).EnableMutationCapture()
	mu *sync.Mutex

	DefaultPattern = "/"
	StaticPath = "assets"


	ServeMux *http.ServeMux
	Server *http.Server = newDefaultServer()
	
	// HTMLHandlerModifier, when not nil, enables to change the behaviour of the request handling.
	// It corresponds loosely to the ability of adding middleware/endware request handler, by using
	// request handler composition.
	// One use-case could be to process http cookies to retrieve info that could be used to customize
	// Document creation, which can be done by changing the DocumentInitializer.
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

func newDefaultServer() *http.Server{
	host := os.Getenv("HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8888"
	}

	server := &http.Server{
		Addr:    host + ":" + port,
		Handler: ServeMux,
	}
	return server
}

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


type customRoundTripper struct {
	mux      *http.ServeMux
	transport http.RoundTripper
}

func (rt *customRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	host, _, err := net.SplitHostPort(r.URL.Host)
	if err != nil {
		// handle error
		panic(err)
	}

	serverHost, _, err := net.SplitHostPort(Server.Addr)
	if err != nil {
		panic(err)
	}

	if host == serverHost {
		// Dispatch the request to the local ServeMux
		w := &responseRecorder{}
		Server.Handler.ServeHTTP(w, r)
		return w.Result(), nil
	} else {
		// Forward the request to the remote server using the custom transport
		return rt.transport.RoundTrip(r)
	}
}

// modifyClient returns a round-tripper modiffied client that can forego the network and 
// generate the response as per the servemux when the host is the server it runs onto. 
func modifyClient(c *http.Client) *http.Client{
	if c == nil{
		c = &http.Client{}
	}
	if c.Transport == nil{
		c.Transport = &http.Transport{}
	}

	c.Transport = &customRoundTripper{ServeMux, c.Transport}

	return c
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       []byte
}

func (w *responseRecorder) Write(b []byte) (int, error) {
	w.body = append(w.body, b...)
	return w.ResponseWriter.Write(b)
}

func (w *responseRecorder) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseRecorder) Result() *http.Response {
	return &http.Response{
		Status:     http.StatusText(w.statusCode),
		StatusCode: w.statusCode,
		Body:       io.NopCloser(bytes.NewReader(w.body)),
		Header:     w.Header(),
	}
}


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

// NewDocumentInitializerHandler returns a http.Handler that sets the DocumentInitializer
// If needed, it should be called as a middleware by wrapping HTMLHandlerModifier.
func NewDocumentInitializerHandler(modifier func(r *http.Request, d Document)) http.Handler{
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		DocumentInitializer = func(d Document)Document{
			modifier(r,d)
			if DocumentInitializer != nil{
				return DocumentInitializer(d)
			}
			return d
		}
	})
}

// NewBuilder registers a new document building function. This function should not be called
// manually afterwards.
func NewBuilder(f func()Document)(ListenAndServe func(ctx context.Context)){
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

		document:= f()
		withNativejshelpers(&document)
		err := d.mutationRecorder().Replay()
		if err != nil{
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		d.mutationRecorder().Capture()
		
		go func(){
			document.ListenAndServe(r.Context())	// launches a new UI thread
		}()
		

		ui.DoSync(func() {
			router:= document.Router()
			route:= r.URL.Path
			_,routeexist:= router.Match(route)
			if routeexist != nil{
				w.WriteHeader(http.StatusNotFound)
			}
			router.GoTo(route)
		})
		
		
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

	return func(ctx context.Context){
		log.Print("Listening on: "+Server.Addr)
		if ctx == nil{
			ctx = context.Background()
		}
		go func(){ // allows for graceful shutdown signaling
			if Server.TLSConfig == nil{
				if err:=Server.ListenAndServe(); err!= nil && err != http.ErrServerClosed{
					log.Fatal(err)
				}
			} else{
				if err:= Server.ListenAndServeTLS("",""); err!= nil && err != http.ErrServerClosed{
					log.Fatal(err)
				}
			}		
		}()

		for{
			select{
			case <-ctx.Done():
				err:= Server.Shutdown(ctx)
				if err!= nil{
					panic(err)
				}
			}
			log.Printf("Server shutdown")
		}
		
	}
	
}


func makeStyleSheet(observable *ui.Element, id string) *ui.Element {
	return observable
}

var titleElementChangeHandler = ui.NewMutationHandler(func(evt ui.MutationEvent) bool { // abstractjs
	setTextContent(evt.Origin(),string(evt.NewValue().(ui.String)))
	return false
})

var historyMutationHandler = ui.NewMutationHandler(func(evt ui.MutationEvent)bool{ // abstractjs
	return false
})

var navinitHandler =  ui.NewMutationHandler(func(evt ui.MutationEvent) bool {// abstractjs
	return false
})

var checkDOMready = NoopMutationHandler

// SetInnerHTML sets the innerHTML property of HTML elements.
// Please note that it is unsafe to sets client submitted HTML inputs.
func SetInnerHTML(e *ui.Element, innerhtml string) *ui.Element {
	p,err:= html.Parse(strings.NewReader(innerhtml))
	if err!=nil{
		panic(err)

	}
	e.Native.(NativeElement).SetChildren(nil)
	e.Native.(NativeElement).Value.AppendChild(p)
	return e
}

// setTextContent sets the textContent of HTML elements.
func setTextContent(e *ui.Element, text string) *ui.Element {

	n:= e.Native.(NativeElement).Value
	f:= n.FirstChild
	c:= f
	for c != nil{
		if c.Type == html.TextNode{
			c.Data = text
			return e
		}
		c = f.NextSibling
	}
	n.AppendChild(textNode(text))
	return e
}

func textNode(s string) *html.Node{
	return &html.Node{Type: html.TextNode,Data: s}
}

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

func ConnectNative(e *ui.Element, tag string) (ui.NativeElement,bool){
	if tag == "window"{
		return  NewNativeElementWrapper(nil), true
	}

	n := &html.Node{}
	n.Type = html.ElementNode
	n.Data = tag
	n.DataAtom = atom.Lookup([]byte(tag))

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

func activityStateSupport (e *ui.Element) *ui.Element{return e}

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
	return newTimeRanges()
}

func(a AudioElement)CurrentTime() time.Duration{
	return 0
}

func(a AudioElement)Duration() time.Duration{
	return  0
}

func(a AudioElement)PlayBackRate() float64{
	return 0
}

func(a AudioElement)Ended() bool{
	return false
}

func(a AudioElement)ReadyState() float64{
	return 0
}

func(a AudioElement)Seekable()  jsTimeRanges{
	return newTimeRanges()
}

func(a AudioElement) Volume() float64{
	return  0
}


func(a AudioElement) Muted() bool{
	return false
}

func(a AudioElement) Paused() bool{

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



func enableClasses(e *ui.Element) *ui.Element {
	h := ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		target := evt.Origin()
		_, ok := target.Native.(NativeElement)
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
			SetAttribute(evt.Origin(),"class",string(classes))
			return false
		}
		RemoveAttribute(evt.Origin(),"class")
		return false
	})
	e.Watch("css", "class", e, h)
	return e
}

func GetAttribute(target *ui.Element, name string) string {
	for _,a:= range target.Native.(NativeElement).Value.Attr{
		if a.Key == name{
			return a.Val
		}
		continue
	}
	return ""
}

func SetAttribute(target *ui.Element, name string, value string) {
	Attrs:= target.Native.(NativeElement).Value.Attr

	for _,a:= range Attrs{
		if a.Key == name{
			a.Val = value
			return
		}
		continue
	}
	Attrs = append(Attrs,html.Attribute{"",name,value})

}

// abstractjs
func RemoveAttribute(target *ui.Element, name string) {
	Attrs:= target.Native.(NativeElement).Value.Attr
	var index = -1

	for i,a:= range Attrs{
		if a.Key == name{
			index = i
			break
		}
		continue
	}
	if index > -1{
		copy(Attrs[:index],Attrs[index+1:])
		Attrs[len(Attrs)-1]=html.Attribute{}
		Attrs = Attrs[:len(Attrs)-1]
	}

}


var textContentHandler = ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
	str := string(evt.NewValue().(ui.String))
	setTextContent(evt.Origin(),str)
	return false
})

var paragraphTextHandler = ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
	setTextContent(evt.Origin(),string(evt.NewValue().(ui.String)))
	return false
})

func numericPropertyWatcher(propname string) *ui.MutationHandler{
	return ui.NoopMutationHandler
}

func boolPropertyWatcher(propname string) *ui.MutationHandler{
	return ui.NoopMutationHandler
}

func stringPropertyWatcher(propname string) *ui.MutationHandler{
	return ui.NoopMutationHandler
}


func clampedValueWatcher(propname string, min int,max int) *ui.MutationHandler{
	return ui.NoopMutationHandler
}


// +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

func (d Document) Render(w io.Writer) error {
	return html.Render(w, newHTMLDocument(d))
}

func newHTMLDocument(document Document) *html.Node {
	doc := document.AsElement()
	h:= &html.Node{Type: html.DoctypeNode}
	n:= doc.Native.(NativeElement).Value
	h.AppendChild(n)
	statenode:= generateStateHistoryRecordElement(doc) // TODO review all this logic
	if statenode != nil{
		document.Head().AsElement().Native.(NativeElement).Value.AppendChild(statenode)
	}

	return h
}

func generateStateHistoryRecordElement(root *ui.Element) *html.Node{
	state:=  SerializeStateHistory(root)
	script:= `<script id='` + SSRStateElementID+`' type="application/json">
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
