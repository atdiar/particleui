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

	SourcePath = filepath.Join("..", "..", "..", "..", "..", "src")
	IndexPath  string

	host string
	port string

	release  bool
	nolr     bool
	basepath string

	render    string
	StaticDir string

	ServeMux *http.ServeMux
	Server   *http.Server

	RenderHTMLhandler http.Handler
)

// NOTE: the default entry path is stored in the BasePath variable stored in dom.go

func init() {
	flag.StringVar(&host, "host", "localhost", "Host name for the server")
	flag.StringVar(&port, "port", "8888", "Port number for the server")

	flag.BoolVar(&release, "release", false, "Build the app in release mode")
	flag.BoolVar(&nolr, "nolr", false, "Disable live reloading")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.BoolVar(&verbose, "v", false, "Enable verbose logging")
	flag.StringVar(&basepath, "basepath", BasePath, "Base path for the server")

	flag.StringVar(&render, "render", "", "Route to render")

	flag.Parse()

	if !release {
		DevMode = "true"
	}

	if !nolr {
		LRMode = "true"
	}

	StaticDir = filepath.Join("..", "..", "..", "client", basepath)
	IndexPath = filepath.Join(StaticDir, "index.html")

	Server = newDefaultServer()
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
func NewBuilder(f func() *Document, buildEnvModifiers ...func()) (ListenAndServe func(ctx context.Context)) {
	fileServer := http.FileServer(DisableDirectoryListing(http.Dir(StaticDir)))

	// First we need to create the document and render the pages
	// ssg is basically about atomically serving a prenavigated app.
	document := f()
	withNativejshelpers(document)

	err := document.mutationRecorder().Replay()
	if err != nil {
		panic(err)
	}
	document.mutationRecorder().Capture()

	start := make(chan struct{})
	go func() {
		document.ListenAndServe(nil, start)
	}()

	// Should generate the file system based structure of the website.
	ui.DoSync(ctx, document.AsElement(), func() {
		// Creating the sitemap.xml file and putting it under the static directory
		// that should have been created in the output directory.
		err = CreateSitemap(document, filepath.Join(StaticDir, "sitemap.xml"))
		if err != nil {
			panic(err)
		}

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
		path := filepath.Join(StaticDir, cleanedPath)

		fi, err := os.Stat(path)
		if err == nil && !fi.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}

		// TODO if fi is Dir, check whether it has an index.html file
		if fi != nil && fi.IsDir() {
			// If it's a directory, check for index.html
			indexPath := filepath.Join(path, "index.html")
			if _, err := os.Stat(indexPath); err == nil {
				// Serve the index.html file if it exists
				http.ServeFile(w, r, indexPath)
				return
			}
			// Otherwise, we return a 403 Forbidden error
			http.Error(w, "Forbidden", http.StatusForbidden)
		}
	})

	for _, m := range buildEnvModifiers {
		m()
	}

	return func(ctx context.Context) {
		if ctx == nil {
			ctx = context.Background()
		}

		if render != "" {
			document := f()

			start := make(chan struct{})
			go func() {
				document.ListenAndServe(ctx, start)
			}()
			start <- struct{}{} // wait for the document to be ready
			if render == "." {
				// we want to render every possible route. This should generate the whole website
				// in the output directory under the form of static index.html files in nested directories.
				// DEBUG TODO use ui.DoSync(document.AsElement(), func() {
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
				ui.DoSync(ctx, document.AsElement(), func() {
					router := document.Router()
					if router != nil {
						router.GoTo(render)
						err := document.CreatePage(filepath.Join(StaticDir, render, "index.html"))
						if err != nil {
							fmt.Printf("Error creating page for route '%s': %v\n", render, err)
							os.Exit(1)
						}
						if verbose {
							fmt.Printf("Created page for route '%s'\n", render)
						}
						fmt.Println("Page created at: ", filepath.Join(StaticDir, render, "index.html"))
					} else {
						DEBUG("router was nil")
					}
				})
			}
			return
		}

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

// TODO modify LR mode to account for ssg structural changes. (no wasm etc, different output directory etc.)
// *****************************

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

func CreatePages(doc *Document) (int, error) {
	// Use StaticPath instead of hardcoded path
	router := doc.Router()
	if router == nil {
		err := doc.CreatePage(filepath.Join(StaticDir, "index.html"))
		return 1, err
	}

	var count int
	for route := range router.Links {
		fullPath := filepath.Join(StaticDir, route, "index.html")
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
	flags[uipkg+"/drivers/js.LRMode"] = LRMode

	var ldflags []string
	for key, value := range flags {
		ldflags = append(ldflags, fmt.Sprintf("-X %s=%s", key, value))
	}
	return strings.Join(ldflags, " ")
}

func DisableDirectoryListing(fs http.FileSystem, allowedPaths ...string) http.FileSystem {
	return &noDirectoryFS{
		fs:           fs,
		allowedPaths: allowedPaths,
	}
}

type noDirectoryFS struct {
	fs           http.FileSystem
	allowedPaths []string
}

func (nfs *noDirectoryFS) Open(name string) (http.File, error) {
	f, err := nfs.fs.Open(name)
	if err != nil {
		return nil, err
	}

	s, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	// If it's a directory, check if it's allowed or has index.html
	if s.IsDir() {
		// Check if this path is in the allowed list
		if nfs.isPathAllowed(name) {
			return f, nil // Allow directory listing
		}

		// Not in allowed list, check for index.html
		indexPath := filepath.Join(name, "index.html")
		if _, err := nfs.fs.Open(indexPath); err != nil {
			f.Close()
			return nil, os.ErrPermission // Returns 403 Forbidden
		}
	}

	return f, nil
}

func (nfs *noDirectoryFS) isPathAllowed(requestPath string) bool {
	// Clean the path to handle different formats
	cleanPath := filepath.Clean(requestPath)
	if cleanPath == "." {
		cleanPath = "/"
	}
	if !strings.HasPrefix(cleanPath, "/") {
		cleanPath = "/" + cleanPath
	}

	for _, allowedPath := range nfs.allowedPaths {
		// Clean the allowed path too
		cleanAllowed := filepath.Clean(allowedPath)
		if cleanAllowed == "." {
			cleanAllowed = "/"
		}
		if !strings.HasPrefix(cleanAllowed, "/") {
			cleanAllowed = "/" + cleanAllowed
		}

		// Exact match or subdirectory match
		if cleanPath == cleanAllowed || strings.HasPrefix(cleanPath, cleanAllowed+"/") {
			return true
		}
	}
	return false
}
