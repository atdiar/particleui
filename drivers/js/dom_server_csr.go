//go:build server && csr

package doc

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	ui "github.com/atdiar/particleui"
	"github.com/fsnotify/fsnotify"

	"golang.org/x/net/html"
	//"golang.org/x/net/html/atom"
)

var (
	uipkg = "github.com/atdiar/particleui"

	SourcePath = filepath.Join(".", "dev")
	StaticPath = filepath.Join(".", "dev", "build", "app")
	IndexPath  = filepath.Join(StaticPath, "index.html")

	host string
	port string

	release  bool
	nohmr    bool
	basepath string

	ServeMux *http.ServeMux
	Server   *http.Server

	RenderHTMLhandler http.Handler
)

// NOTE: the default entry path is stored in the BasePath variable stored in dom.go

func init() {
	flag.StringVar(&host, "host", "localhost", "Host name for the server")
	flag.StringVar(&port, "port", "8888", "Port number for the server")

	flag.BoolVar(&release, "release", false, "Build the app in release mode")
	flag.BoolVar(&nohmr, "nohmr", false, "Disable hot module reloading")

	flag.StringVar(&basepath, "basepath", BasePath, "Base path for the server")

	flag.Parse()

	if !release {
		DevMode = "true"
	} else {
		SourcePath = filepath.Join(".", "release")
		StaticPath = filepath.Join(".", "release", "build", "app")
		IndexPath = filepath.Join(StaticPath, "index.html")
	}

	if !nohmr {
		HMRMode = "true"
	}
	Server = newDefaultServer()

}

func newDefaultServer() *http.Server {
	return &http.Server{
		Addr:    host + ":" + port,
		Handler: ServeMux,
	}
}

type customRoundTripper struct {
	mux       *http.ServeMux
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
func modifyClient(c *http.Client) *http.Client {
	if c == nil {
		c = &http.Client{}
	}
	if c.Transport == nil {
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

// NewBuilder accepts a function that builds a document as a UI tree and returns a function
// that enables event listening for this document.
// On the client, these are client side javascrip events.
// On the server, these events are http requests to a given UI endpoint, translated then in a navigation event
// for the document.
var NewBuilder = func(f func() *Document, buildEnvModifiers ...func()) (ListenAndServe func(ctx context.Context)) {

	fileServer := http.FileServer(http.Dir(StaticPath))

	RenderHTMLhandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if HMRMode != "false" || !nohmr {
			w.Header().Set("Cache-Control", "no-cache")
		}

		// Clean the URL path to prevent directory traversal
		cleanedPath := filepath.Clean(r.URL.Path)

		// Join the cleaned path with the static directory
		path := filepath.Join(StaticPath, cleanedPath)

		// Check if the requested file exists
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			// If the file does not exist, serve index.html
			http.ServeFile(w, r, filepath.Join(StaticPath, BasePath, "index.html"))
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// If the file exists, serve it
		fileServer.ServeHTTP(w, r)
	})

	for _, m := range buildEnvModifiers {
		m()
	}

	ServeMux = http.NewServeMux()
	Server.Handler = ServeMux

	return func(ctx context.Context) {
		if ctx == nil {
			ctx = context.Background()
		}
		ctx, shutdown := context.WithCancel(ctx)
		var activehmr bool

		var SSEChannel *SSEController
		var mu = &sync.Mutex{}

		ServeMux.Handle(BasePath, RenderHTMLhandler)

		if DevMode != "false" {
			ServeMux.Handle("/stop", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Trigger server shutdown logic
				shutdown()
				fmt.Fprintln(w, "Server is shutting down...")
			}))
		}

		if HMRMode == "true" {
			// TODO: Implement Server-Sent Event logic for browser reload
			// Implement filesystem watching and trigger compile on change
			// (in another goroutine) if it's a go file. If any file change, send SSE message to frontend
			//
			// 1. Watch ./dev/*.go files. If any is modified, try to recompile. IF not successful nothing happens of course.
			// 2. Watch ./dev/build/app folder. If anything changed, send SSE message to frontend to reload the page.

			// path to the directory containing the source files

			watcher, err := WatchDir(SourcePath, func(event fsnotify.Event) {
				// Only rebuild if the event is for a .go file
				if filepath.Ext(event.Name) == ".go" {
					// file name: main.go
					sourceFile := "main.go"

					// Ensure the output directory is already existing
					if _, err := os.Stat(StaticPath); os.IsNotExist(err) {
						panic("Output directory should already exist")
					}

					targetPath, err := filepath.Rel(SourcePath, StaticPath)
					if err != nil {
						panic(err)
					}
					targetPath = filepath.Join(targetPath, "main.wasm")

					// add the relevant build and linker flags
					args := []string{"build"}
					ldflags := ldflags()
					if ldflags != "" {
						args = append(args, "-ldflags", ldflags)
					}

					args = append(args, "-o", targetPath, sourceFile)

					cmd := exec.Command("go", args...)
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					cmd.Dir = SourcePath // current directory where the build command is run
					cmd.Env = append(cmd.Environ(), "GOOS=js", "GOARCH=wasm")

					err = cmd.Run()
					if err == nil {
						fmt.Println("main.wasm was rebuilt.")
					}
				}
			})

			if err != nil {
				log.Println(err)
				log.Println("Unable to watch for changes in ./dev folder.")
			} else {
				defer watcher.Close()

				// watching for changes made to the output files which should be in the ./dev/build/app directory
				// ie. ../../dev/build/app

				wc, err := WatchDir(StaticPath, func(event fsnotify.Event) {
					// Send event to trigger a page reload
					log.Println("Something changed: ", event.String()) // DEBUG
					mu.Lock()
					SSEChannel.SendEvent("reload", event.String(), "", "")
					log.Println("reload Event sent to frontend") // DEBUG
					mu.Unlock()
				})
				if err != nil {
					log.Println(err)
					panic("Unable to watch for changes in ./dev/build/app folder.")
				}
				defer wc.Close()
				activehmr = true
				ServeMux.Handle("/sse", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					log.Println("SSE connection established")
					s := NewSSEController()
					mu.Lock()
					SSEChannel = s
					mu.Unlock()
					s.ServeHTTP(w, r)
				}))
			}
		}

		// return server info including whether hmr is active
		ServeMux.Handle("/info", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "Server is running on port "+port)
			fmt.Fprintln(w, "HMR status active: ", activehmr)
		}))

		go func() { // allows for graceful shutdown signaling
			if Server.TLSConfig == nil {
				if err := Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Fatal(err)
				}
			} else {
				if err := Server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
					log.Fatal(err)
				}
			}
		}()

		log.Print("Listening on: " + Server.Addr)

		for {
			select {
			case <-ctx.Done():
				err := Server.Shutdown(ctx)
				if err != nil {
					panic(err)
				}
				log.Printf("Server shutdown")
				os.Exit(0)
			}
		}
	}
}

// +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

func (d Document) Render(w io.Writer) error {
	return html.Render(w, newHTMLDocument(d))
}

func newHTMLDocument(document Document) *html.Node {
	doc := document.AsElement()
	h := &html.Node{Type: html.DoctypeNode}
	n := doc.Native.(NativeElement).Value.Node()
	h.AppendChild(n)
	statenode := generateStateHistoryRecordElement(doc) // TODO review all this logic
	if statenode != nil {
		document.Head().AsElement().Native.(NativeElement).Value.Call("appenChild", statenode)
	}

	return h
}

func generateStateHistoryRecordElement(root *ui.Element) *html.Node {
	state := SerializeStateHistory(root)
	script := `<script id='` + SSRStateElementID + `' type="application/json">
	` + state + `
	<script>`
	scriptNode, err := html.Parse(strings.NewReader(script))
	if err != nil {
		panic(err)
	}
	return scriptNode
}

func recoverStateHistory() {}

var recoverStateHistoryHandler = ui.NoopMutationHandler

func ldflags() string {
	flags := make(map[string]string)

	flags[uipkg+"/drivers/js.DevMode"] = DevMode
	flags[uipkg+"/drivers/js.SSGMode"] = SSGMode
	flags[uipkg+"/drivers/js.SSRMode"] = SSRMode
	flags[uipkg+"/drivers/js.HMRMode"] = HMRMode

	var ldflags []string
	for key, value := range flags {
		ldflags = append(ldflags, fmt.Sprintf("-X %s=%s", key, value))
	}
	return strings.Join(ldflags, " ")
}
