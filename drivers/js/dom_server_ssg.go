//go:build server && ssg

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

	ui "github.com/atdiar/particleui"
	js "github.com/atdiar/particleui/drivers/js/compat"
	"golang.org/x/net/html"
	//"golang.org/x/net/html/atom"
)

var (
	uipkg = "github.com/atdiar/particleui"

	StaticPath string
	IndexPath  string

	host string
	port string

	release  bool
	nohmr    bool
	noserver bool

	renderedRoute string
	outputDir     string

	ServeMux *http.ServeMux
	Server   *http.Server = newDefaultServer()

	RenderHTMLhandler http.Handler

	verbose bool
)

// NOTE: the default entry path is stored in the BasePath variable stored in dom.go

func init() {
	flag.StringVar(&host, "host", "localhost", "Host name for the server")
	flag.StringVar(&port, "port", "8888", "Port number for the server")

	flag.BoolVar(&noserver, "noserver", false, "Generate the pages without starting a server")
	flag.BoolVar(&release, "release", false, "Build the app in release mode")
	flag.BoolVar(&nohmr, "nohmr", false, "Disable hot module reloading")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.BoolVar(&verbose, "v", false, "Enable verbose logging")
	flag.StringVar(&basepath, "basepath", BasePath, "Base path for the server")

	flags.StringVar(&outputDir, "output", "../../../../client", "Output directory for the static files")
	flags.StringVar(&renderedRoute, "route", ".", "Route to render")

	flag.Parse()

	if !release {
		DevMode = "true"
	}

	if !nohmr {
		HMRMode = "true"
	}

	exe, _ := os.Executable()
	exeDir := filepath.Dir(exe)
	relPath, _ := filepath.Rel(exeDir, outputDir)

	StaticPath = relPath
	IndexPath = filepath.Join(StaticPath, "index.html")
}

func (n NativeElement) SetChildren(children ...*ui.Element) {
	// should delete all children and add the new ones
	n.Value.Get("children").Call("remove")

	for _, child := range children {
		v, ok := child.Native.(NativeElement)
		if !ok {
			return
		}
		n.Value.Call("append", v.Value)
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

// NewBuilder accepts a function that builds a document as a UI tree and returns a function
// that enables event listening for this document.
// On the client, these are client side javascrip events.
// On the server, these events are http requests to a given UI endpoint, translated then in a navigation event
// for the document.
func NewBuilder(f func() Document, buildEnvModifiers ...func()) (ListenAndServe func(ctx context.Context)) {
	fileServer := http.FileServer(http.Dir(StaticPath))

	// First we need to create the document and render the pages
	// ssg is basically about atomically serving a prenavigated app.
	document := f()
	withNativejshelpers(&document)

	err := document.mutationRecorder().Replay()
	if err != nil {
		panic(err)
	}
	document.mutationRecorder().Capture()

	// Creating the sitemap.xml file and putting it under the static directory
	// that should have been created in the output directory.
	err = CreateSitemap(document, filepath.Join(StaticPath, "sitemap.xml"))
	if err != nil {
		panic(err)
	}

	go func() {
		document.ListenAndServe(nil) // launches a new UI thread
	}()

	// Should generate the file system based structure of the website.
	ui.DoSync(func() {
		// Traverse the document routes and generate the corresponding files
		// in the output directory.
		numPages, err := CreatePages(document)
		if err != nil {
			fmt.Printf("Error creating pages: %v\n", err)
		} else {
			if verbose {
				fmt.Printf("Created %d pages\n", numPages)
			}
		}
	})

	RenderHTMLhandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Serve the index.html file for all requests
		// Clean the URL path to prevent directory traversal
		cleanedPath := filepath.Clean(r.URL.Path)

		// Join the cleaned path with the static directory
		path := filepath.Join(StaticPath, cleanedPath)

		// Check if the requested file exists
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			// If the file does not exist, serve index.html
			http.ServeFile(w, r, IndexPath) // Use IndexPath directly since it's already joined with StaticPath
			return
		} else if err != nil {
			// If there's an error checking the file, return an internal server error
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// If the file exists, serve it
		fileServer.ServeHTTP(w, r)
	})

	for _, m := range buildEnvModifiers {
		m()
	}

	if noserver {
		return func(ctx context.Context) {
			if renderedRoute != "" {
				if renderedRoute == "." {
					// we want to render every possible route. This should generate the whole website
					// in the output directory under the form of sattic files in nested directories.
					n, err := CreatePages(document)
					if err != nil {
						fmt.Printf("Error creating pages: %v\n", err)
						os.Exit(1)
					} else {
						if verbose {
							fmt.Printf("Created %d pages\n", n)
						}
					}

				} else {
					// We render the route that was specified in the command line.
					// This should generate the corresponding file in the output directory.
					document.Router().GoTo(renderedRoute)
					err := document.CreatePage(filepath.Join(StaticPath, renderedRoute, "index.html"))
					if err != nil {
						fmt.Printf("Error creating page for route '%s': %v\n", renderedRoute, err)
						os.Exit(1)
					} else {
						if verbose {
							fmt.Printf("Created page for route '%s'\n", renderedRoute)
						}
					}
				}

			}
		}
	}

	// TODO modify HMR mode to account for ssg structural changes. (no wasm etc, different output directory etc.)
	// ******************************
	return func(ctx context.Context) {
		ctx, shutdown := context.WithCancel(ctx)

		ServeMux.Handle(BasePath, RenderHTMLhandler)

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

func CreatePages(doc Document) (int, error) {
	// Use StaticPath instead of hardcoded path
	router := doc.Router()
	if router == nil {
		err := doc.CreatePage(filepath.Join(StaticPath, "index.html"))
		return 1, err
	}

	var count int
	for _, route := range routes {
		fullPath := filepath.Join(StaticPath, route, "index.html")
		if verbose {
			fmt.Printf("Creating page for route '%s' at '%s'\n", route, fullPath)
		}
		doc.Router().GoTo(route)
		if err := doc.CreatePage(fullPath); err != nil {
			return count, fmt.Errorf("error creating page for route '%s': %w", route, err)
		}
		count++
	}
	return count, nil
}

// CreatePage creates a single page for the document at the specified filePath.
func (d Document) CreatePage(filePath string) error {
	// Create the directory if it doesn't exist
	dirPath := filepath.Dir(filePath)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return err
	}

	// Determine the path for the CSS file
	cssFilePath := filepath.Join(dirPath, "style.css")

	// Generate the stylesheet for this page
	if err := d.CreateStylesheet(cssFilePath); err != nil {
		return fmt.Errorf("error creating stylesheet: %w", err)
	}
	if verbose {
		fmt.Printf("Created stylesheet at '%s'\n", cssFilePath)
	}

	// Append stylesheet link to the document head
	cssRelPath := "./style.css"
	link := d.Link().SetAttribute("href", cssRelPath).SetAttribute("rel", "stylesheet")
	d.Head().AppendChild(link)

	// Create and open the file
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Render the document
	if err := d.Render(file); err != nil {
		return err
	}

	return nil
}

func (d Document) CreateStylesheet(cssFilePath string) error {
	rl, ok := d.Get("internals", "activestylesheets")
	if !ok {
		return fmt.Errorf("no active stylesheets found")
	}
	l := rl.(ui.List) // list of stylesheetIDs in the order they should be applied

	var cssContent strings.Builder

	for _, sheetID := range l.UnsafelyUnwrap() {
		sheet, ok := d.GetStyleSheet(sheetID.(ui.String).String())
		if !ok {
			panic("stylesheet not found")
		}
		cssContent.WriteString(sheet.String())

	}

	return os.WriteFile(cssFilePath, []byte(cssContent.String()), 0644)
}

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
