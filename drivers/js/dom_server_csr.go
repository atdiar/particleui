//go:build server && csr


package doc

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"
	"path/filepath"
	"log"
	
	"github.com/fsnotify/fsnotify"
	"github.com/atdiar/particleui"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var (
	// DOCTYPE holds the document doctype.
	DOCTYPE = "html"
	// Elements stores wasm-generated HTML ui.Element constructors.
	Elements = ui.NewElementStore("default", DOCTYPE).ApplyGlobalOption(allowDataFetching).EnableMutationCapture()

	uipkg = "github.com/atdiar/particleui"

	DefaultPattern = "/"

	absStaticPath = filepath.Join(".","dev","build","app")
	absIndexPath = filepath.Join(absStaticPath,"index.html")
	absCurrentPath = filepath.Join(".","dev","build","server","csr")

	StaticPath,_ = filepath.Rel(absCurrentPath,absStaticPath)
	IndexPath,_ = filepath.Rel(absCurrentPath,absIndexPath)

	host string
	port string

	release bool
	nohmr bool

	ServeMux *http.ServeMux
	Server *http.Server = newDefaultServer()
	
	
	RenderHTMLhandler http.Handler

)

func init(){
	flag.StringVar(&host,"host", "localhost", "Host name for the server")
	flag.StringVar(&port, "port", "8888", "Port number for the server")

	flag.BoolVar(&release, "release", false, "Build the app in release mode")
	flag.BoolVar(&nohmr, "nohmr", false, "Disable hot module reloading")
	
	flag.Parse()

	if !release{
		DevMode = "true"
	}

	if !nohmr{
		HMRMode = "true"
	}

}


func newDefaultServer() *http.Server{
	return &http.Server{
		Addr:    host + ":" + port,
		Handler: ServeMux,
	}
}



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

// modifyClient returns a round-tripper modified client that can forego the network and 
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


// NewBuilder registers a new document building function. 
// In Server Rendering mode (ssr or csr), it starts a server.
// It accepts functions that can be used to modify the global state (environment) in which a document is built.
func NewBuilder(f func()Document, buildEnvModifiers ...func())(ListenAndServe func(ctx context.Context)){
	fileServer := http.FileServer(http.Dir(StaticPath))

	RenderHTMLhandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		path, err := filepath.Abs(r.URL.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		path = filepath.Join(StaticPath, r.URL.Path)

		_, err = os.Stat(path)
		if os.IsNotExist(err) {
			// file does not exist, serve index.html
			http.ServeFile(w, r, filepath.Join(StaticPath, IndexPath))
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fileServer.ServeHTTP(w, r)
	})

	for _,m:= range buildEnvModifiers{
		m()
	}

	ServeMux = http.NewServeMux()
	Server.Handler = ServeMux


	return func(ctx context.Context){
		if ctx == nil{
			ctx = context.Background()
		}
		ctx, shutdown := context.WithCancel(ctx)
		var activehmr bool

		ServeMux.Handle(DefaultPattern,RenderHTMLhandler)
		
		if DevMode != "false"{
			ServeMux.Handle("/stop",http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Trigger server shutdown logic
				shutdown()
				fmt.Fprintln(w, "Server is shutting down...")
			}))
		}

		if HMRMode == "true"{
			// TODO: Implement Server-Sent Event logic for browser reload
			// Implement filesystem watching and trigger compile on change
			// (in another goroutine) if it's a go file. If any file change, send SSE message to frontend
			//
			// 1. Watch ./dev/*.go files. If any is modified, try to recompile. IF not successful nothing happens of course.
			// 2. Watch ./dev/build/app folder. If anything changed, send SSE message to frontend to reload the page.

			// path to the directory containing the source files
			outputPath, err := filepath.Rel(absCurrentPath,filepath.Join(".","dev","build","app"))
			if err != nil{
				log.Println(err)
				panic("Can't find path to output directory ./dev/build/app")
			}

			srcDirPath,err := filepath.Rel(absCurrentPath,filepath.Join(".","dev"))
			if err != nil{
				log.Println(err)
				log.Println("Unable to watch for changes in ./dev folder, couldn't find path.")
			} else{

				// watching for changes made to the source files which should be in the ./dev directory
				// ie. ../../../dev
				watcher, err := WatchDir(srcDirPath, func(event fsnotify.Event) {
					// Only rebuild if the event is for a .go file
					if filepath.Ext(event.Name) == ".go" {
						// path to main.go
						sourceFile := filepath.Join(srcDirPath, "main.go")
						// let's build main.go TODO: shouldn't rebuild the server.. might need to 
						// review impl of zui (or not, here it should be agnostic so might as well 
						// reimplement the logic with the few specific requirements)
						// Ensure the output directory is already existing
						outputDir := filepath.Dir(outputPath)
						if _, err := os.Stat(outputDir); os.IsNotExist(err) {
							panic("Output directory should already exist")
						}
						// add the relevant build and linker flags
						args := []string{"build"}
						ldflags:= ldflags()
						if ldflags != "" {
							args = append(args, "-ldflags", ldflags)	
						}

						args = append(args, "-o", outputPath)

						args = append(args, sourceFile)
						cmd := exec.Command("go", args...)
						cmd.Stdout = os.Stdout
						cmd.Stderr = os.Stderr
						cmd.Dir = filepath.Join(".","dev")
						cmd.Env = append(cmd.Environ(), "GOOS=js", "GOARCH=wasm")

						err := cmd.Run()
						if err == nil {
							fmt.Println("main.wasm was rebuilt.")
						}						
					}
				})

				if err != nil{
					log.Println(err)
					panic("Unable to watch for changes in ./dev folder.")
				}
				defer watcher.Close()

				// watching for changes made to the output files which should be in the ./dev/build/app directory
				// ie. ../../dev/build/app
				wc, err := WatchDir(outputPath, func(event fsnotify.Event) {
					// Send event to trigger a page reload
					SSEChannel.SendEvent("reload",event.String(),"","")	
				})
				if err != nil{
					log.Println(err)
					panic("Unable to watch for changes in ./dev/build/app folder.")
				} 
				defer wc.Close()
				activehmr = true				
				ServeMux.Handle("/sse", SSEChannel)
			}
		}

		// return server info including whether hmr is active
		ServeMux.Handle("/info",http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "Server is running on port "+port)
			fmt.Fprintln(w, "HMR status active: ", activehmr)
		}))

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

		log.Print("Listening on: "+Server.Addr)

		for{
			select{
			case <-ctx.Done():
				err:= Server.Shutdown(ctx)
				if err!= nil{
					panic(err)
				}
				log.Printf("Server shutdown")
				os.Exit(0)
			}
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

	j.Set("start",ui.NewList().Commit())
	j.Set("end",ui.NewList().Commit())
	return jsTimeRanges(j.Commit())
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


func ldflags() string {
	flags := make(map[string]string)

	flags[uipkg + "/drivers/js.DevMode"] = DevMode
	flags[uipkg + "/drivers/js.SSGMode"] = SSGMode
	flags[uipkg + "/drivers/js.SSRMode"] = SSRMode
	flags[uipkg + "/drivers/js.HMRMode"]= HMRMode

	var ldflags []string
	for key, value := range flags {
		ldflags = append(ldflags, fmt.Sprintf("-X %s=%s", key, value))
	}
	return strings.Join(ldflags, " ")
}
