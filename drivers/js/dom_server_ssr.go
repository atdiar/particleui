//go:build server && ssr

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
	"path/filepath"
	"strings"

	"github.com/atdiar/particleui"

	"github.com/atdiar/particleui/drivers/js/compat"
	"golang.org/x/net/html"
	//"golang.org/x/net/html/atom"
)

var (
	uipkg = "github.com/atdiar/particleui"

	absStaticPath  = filepath.Join(".", "dev", "build", "app")
	absIndexPath   = filepath.Join(absStaticPath, "index.html")
	absCurrentPath = filepath.Join(".", "dev", "build", "server", "csr")

	StaticPath, _ = filepath.Rel(absCurrentPath, absStaticPath)
	IndexPath, _  = filepath.Rel(absCurrentPath, absIndexPath)

	host string
	port string

	release bool
	nohmr   bool

	ServeMux *http.ServeMux
	Server   *http.Server = newDefaultServer()

	RenderHTMLhandler http.Handler
)

func init() {
	Elements.EnableMutationCapture()

	flag.StringVar(&host, "host", "localhost", "Host name for the server")
	flag.StringVar(&port, "port", "8888", "Port number for the server")

	flag.BoolVar(&release, "release", false, "Build the app in release mode")
	flag.BoolVar(&nohmr, "nohmr", false, "Disable hot module reloading")

	flag.Parse()

	if !release {
		DevMode = "true"
	}

	if !nohmr {
		HMRMode = "true"
	}

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

// modifyClient returns a round-tripper modiffied client that can forego the network and
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

// NewBuilder registers a new document building function.
// In Server Rendering mode (ssr or csr), it starts a server.
// It accepts functions that can be used to modify the global state (environment) in which a document is built.
func NewBuilder(f func() Document, buildEnvModifiers ...func()) (ListenAndServe func(ctx context.Context)) {
	fileServer := http.FileServer(http.Dir(StaticPath))

	RenderHTMLhandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, err := filepath.Abs(r.URL.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		path = filepath.Join(StaticPath, r.URL.Path)

		fi, err := os.Stat(path)
		if err == nil && !fi.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}

		// This is the offline shell of the app. It could be used
		// to serve a kind of static husk via CDNs if needed (TODO?)
		// At this point, the fetches have not been sent. Navigation
		// triggers the first fetches and can take advantage of the
		// cookies having been put into the cookiejar.
		document := f()
		document.Element.HttpClient.Jar.SetCookies(r.URL, r.Cookies())

		withNativejshelpers(&document)

		err = document.mutationRecorder().Replay()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		document.mutationRecorder().Capture()

		go func() {
			document.ListenAndServe(r.Context()) // launches a new UI thread
		}()

		ui.DoSync(func() {
			router := document.Router()
			route := r.URL.Path
			_, routeexist := router.Match(route)
			if routeexist != nil {
				w.WriteHeader(http.StatusNotFound)
			}
			router.GoTo(route)
		})

		err = document.Render(w)
		if err != nil {
			switch err {
			case ui.ErrNotFound:
				w.WriteHeader(http.StatusNotFound)
			case ui.ErrFrameworkFailure:
				w.WriteHeader(http.StatusInternalServerError)
			case ui.ErrUnauthorized:
				w.WriteHeader(http.StatusUnauthorized)
			}
		}

	})

	for _, m := range buildEnvModifiers {
		m()
	}

	return func(ctx context.Context) {
		if ctx == nil {
			ctx = context.Background()
		}
		ctx, shutdown := context.WithCancel(ctx)

		if DevMode != "false" {
			ServeMux.Handle("/stop", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Trigger server shutdown logic
				shutdown()
				fmt.Fprintln(w, "Server is shutting down...")
			}))
		}

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
	return html.Render(w, newHTMLDocument(d).Node())
}

func newHTMLDocument(document Document) js.Value {
	doc := document.AsElement()
	h := js.ValueOf(&html.Node{Type: html.DoctypeNode})
	n := doc.Native.(NativeElement).Value
	h.Call("appendChild", n)
	statenode := generateStateHistoryRecordElement(doc) // TODO review all this logic
	if statenode != nil {
		document.Head().AsElement().Native.(NativeElement).Value.Call("appendChild", statenode)
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
