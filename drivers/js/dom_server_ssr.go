//go:build server && ssr && !csr

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
	"github.com/yosssi/gohtml"

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

func init() {
	Elements.EnableMutationCapture()

	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.BoolVar(&verbose, "v", false, "Enable verbose logging")

	flag.StringVar(&host, "host", "localhost", "Host name for the server")
	flag.StringVar(&port, "port", "8888", "Port number for the server")

	flag.BoolVar(&release, "release", false, "Build the app in release mode")
	flag.BoolVar(&nolr, "nolr", false, "Disable live reloading")

	flag.StringVar(&basepath, "basepath", BasePath, "Base path for the server")
	flag.StringVar(&render, "render", "", "specify the page(s) that will be rendered to html")

	flag.Parse()

	if !release {
		DevMode = "true"
	}

	StaticDir = filepath.Join("..", "..", "..", "client", basepath)
	IndexPath = filepath.Join(StaticDir, "index.html")

	if !nolr {
		LRMode = "true"
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
func NewBuilder(f func() *Document, buildEnvModifiers ...func()) (ListenAndServe func(ctx context.Context)) {
	fileServer := http.FileServer(DisableDirectoryListing(http.Dir(StaticDir)))

	RenderHTMLhandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if LRMode != "false" || !nolr {
			w.Header().Set("Cache-Control", "no-cache")
		}
		DEBUG("RenderHTMLhandler called for: ", r.URL.Path)
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
			// Otherwise, we let the handler generate the appropriate page
		}

		document := f()
		document.Element.HttpClient = modifyClient(document.Element.HttpClient)
		document.Element.HttpClient.Jar.SetCookies(r.URL, r.Cookies())

		withNativejshelpers(document)

		err = document.mutationRecorder().Replay()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		document.mutationRecorder().Capture()

		start := make(chan struct{})
		go func() {
			document.ListenAndServe(r.Context(), start)
		}()

		ui.DoSync(r.Context(), document.AsElement(), func() {
			router := document.Router()
			route := r.URL.Path
			_, routeexist := router.Match(route)
			if routeexist != nil {
				DEBUG("route not found: ", route)
				w.WriteHeader(http.StatusNotFound)
			}
			router.GoTo(route)
		})

		err = document.Render(w)
		if err != nil {
			if verbose {
				fmt.Printf("Error rendering document: %v\n", err)
			}
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

	ServeMux = http.NewServeMux()
	Server.Handler = ServeMux

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

// +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

func (d *Document) Render(w io.Writer) error {
	var buf bytes.Buffer
	if err := html.Render(&buf, newHTMLDocument(d).Parent); err != nil {
		return err
	}

	formatted := gohtml.Format(buf.String())
	_, err := w.Write([]byte(formatted))
	return err
}

func newHTMLDocument(document *Document) *html.Node {
	h, ok := JSValue(document.AsElement())
	if !ok {
		panic("Error converting Document to HTML Node")
	}

	statenode := generateStateHistoryRecordElement(document.AsElement()) // TODO review all this logic
	if statenode != nil {
		h.Call("appendChild", statenode)
	}

	return h.Node()
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
	flags[uipkg+"/drivers/js.LRMode"] = LRMode

	var ldflags []string
	for key, value := range flags {
		ldflags = append(ldflags, fmt.Sprintf("-X %s=%s", key, value))
	}
	return strings.Join(ldflags, " ")
}

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
	// DEBUG
	fmt.Printf("Creating page at '%s'\n", dirPath)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return err
	}

	// Append stylesheet link to the document head
	// Note here how we have to supply an ID so that there is no conflict when
	// doing a replay of a server side render. The reason being that the ID generator is
	// deterministic on purpose for perfect replay but does not allow for difference in rendering
	// where a new element is introduced on only one platform and might take the ID.

	ss, ok := d.GetCurrentStyleSheet()
	if ok {
		cssRelPath := strings.Join([]string{"./", ss, ".css"}, "")
		link := d.Link.WithID(strings.Join([]string{d.AsElement().ID, ss, "stylesheet"}, "-")).SetAttribute("href", cssRelPath).SetAttribute("rel", "stylesheet")
		d.Head().AppendChild(link)

		// Determine the path for the CSS file
		cssFilePath := filepath.Join(dirPath, cssRelPath)

		// Generate the stylesheet for this page
		if err := d.CreateStylesheet(cssFilePath); err != nil {
			return fmt.Errorf("error creating stylesheet: %w", err)
		}
		if verbose {
			fmt.Printf("Created stylesheet at '%s'\n", cssFilePath)
		}
	}

	// Create and open the file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creating file '%s': %w", filePath, err)
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
		// return fmt.Errorf("no active stylesheets found") // DEBUG
		return nil
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
