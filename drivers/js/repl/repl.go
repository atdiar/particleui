package repl

import (
	"archive/zip"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/atdiar/particleui"
	. "github.com/atdiar/particleui/drivers/js"
	doc "github.com/atdiar/particleui/drivers/js"
	"github.com/atdiar/particleui/drivers/js/compat"
)

var (
	CompileWasmURL string
	LinkWasmURL    string
	GofmtWasmURL   string
	PkgDirURL      string
)

// decompressArchive takes a Uint8Array containing a zip archive, decompresses it,
// and returns a map of file paths to Uint8Array of their contents.
func decompressArchive(this js.Value, inputs []js.Value) interface{} {
	if len(inputs) < 1 {
		return map[string]interface{}{"error": "No input provided"}
	}

	inputJS := inputs[0] // The first argument is the Uint8Array from JavaScript

	// Prepare a Go byte slice with the same length as the input Uint8Array
	inputData := make([]byte, inputJS.Length())
	js.CopyBytesToGo(inputData, inputJS) // Copy the data into Go's memory space

	// Use the zip package to read the archive from the inputData byte slice
	zipReader, err := zip.NewReader(bytes.NewReader(inputData), int64(len(inputData)))
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	files := make(map[string]interface{})
	for _, zipFile := range zipReader.File {
		fileReader, err := zipFile.Open()
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}

		fileContent, err := io.ReadAll(fileReader)
		fileReader.Close() // Make sure to close the reader to avoid resource leaks
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}

		// Here we directly create a new Uint8Array in JavaScript context for each file's content.
		jsFileContent := js.Global().Get("Uint8Array").New(len(fileContent))
		js.CopyBytesToJS(jsFileContent, fileContent)
		files[zipFile.Name] = jsFileContent
	}

	return files
}

type ReplElement struct {
	*ui.Element
}

// TextArea returns the textarea element that holds the code input
func (r ReplElement) TextArea() TextAreaElement {
	return TextAreaElement{GetDocument(r.AsElement()).GetElementById(r.AsElement().ID + "-textarea")}
}

// Output returns the div element that holds the output status of the code execution
func (r ReplElement) Output() DivElement {
	return DivElement{GetDocument(r.AsElement()).GetElementById(r.AsElement().ID + "-output")}
}

// Iframe returns the iframe element that holds the rendered output from loading the wasm file.
func (r ReplElement) Iframe() IframeElement {
	return IframeElement{GetDocument(r.AsElement()).GetElementById(r.AsElement().ID + "-iframe")}
}

// BtnRun returns the button element that triggers the code execution
func (r ReplElement) BtnRun() ButtonElement {
	return ButtonElement{GetDocument(r.AsElement()).GetElementById(r.AsElement().ID + "-run")}
}

// BtnFmt returns the button element that triggers the code formatting
func (r ReplElement) BtnFmt() ButtonElement {
	return ButtonElement{GetDocument(r.AsElement()).GetElementById(r.AsElement().ID + "-fmt")}
}

// CodeInputContainer returns the div element that holds the contentarea and the buttons
func (r ReplElement) CodeInputContainer() DivElement {
	return DivElement{GetDocument(r.AsElement()).GetElementById(r.AsElement().ID + "-codeinput")}
}

// ButtonsContainer returns the div element that holds the buttons
func (r ReplElement) ButtonsContainer() DivElement {
	return DivElement{GetDocument(r.AsElement()).GetElementById(r.AsElement().ID + "-buttons")}
}

// Repl returns a ReplElement which is able to compile and render UI code in the browser locally.
func Repl(d *Document, id string, pathToCompilerAssetDir string, options ...string) ReplElement {
	repl := d.Div.WithID(id, options...)
    doc.SyncOnDataMutation(repl.AsElement(), "text")
	// TODO append the gcversion script with the value attribute set to the go compiler version in use
	textarea := d.TextArea.WithID(id + "-textarea")
	outputfield := d.Div.WithID(id + "-output")
	iframeresult := d.Iframe.WithID(id+"-iframe", "about:blank")

	btnRun := d.Button.WithID(id+"-run", "input")
	btnFmt := d.Button.WithID(id+"-fmt", "input")

	// double binding between repl and contentarea subcomponent.
	// on sync mutation event, the repl top div gets synced as well. (caused by input events for instance)
	// if the repl's code is set from somewhere however, the contentarea will get set to the same value too.
	repl.Watch("data", "value", textarea, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		repl.SyncData("value", evt.NewValue())
		return false
	}).OnSync())

	repl.Watch("data", "value", repl, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		TextAreaModifier.Value(evt.NewValue().(ui.String).String())(textarea.AsElement())
		return false
	}).RunASAP())

	E(repl,
		Children(
			E(iframeresult),
			E(d.Div.WithID(id+"-codeinput"),
				Children(
					E(d.Div.WithID(id+"-buttons"),
						Children(
							E(btnRun),
							E(btnFmt),
						),
					),
					E(textarea,
						Listen("keydown", ui.NewEventHandler(func(evt ui.Event) bool {
							if evt.(KeyboardEvent).Key() == "Tab" {
								evt.PreventDefault()
								evt.StopPropagation()
								TextAreaModifier.Value(textarea.Text() + "\t")(textarea.AsElement())
								return false
							}
							return false
						})),
						Listen("input", ui.NewEventHandler(func(evt ui.Event) bool {
							// DEBUG TODO check that the value corresponds to the value of the element,
							// even in case where it has been modified somehow sucha as is the case when tabbing.
							evt.CurrentTarget().SyncUISyncData("value", WithStrConv(evt.Value()))
							return false
						})),
					),
					E(outputfield),
				),
			),
		),
	)

	// Let's add the event listeners for the buttons and the contentarea
	// The contentarea should be hijacked to prevent the default behavior of the tab key
	// which is to move the focus to the next element in the document.

	// let's add the scripts for the virtual filesystem and the compiler suite loader
	// if they haven't already been added to the document Head.
	// A script with id "gcversion" that holds the compiler version string should also be appended.

	if _, ok := d.GetEventValue("replInitialized"); !ok {
		h := d.Head()

		/*
					// gcversion script
					s := d.Script.WithID("gcversion")
			        // the go version should be present in the assetURI. e.g. /wasmgc_1.16.3/
			        // the version is the last part of the path.
					SetAttribute(s.AsElement(), "value", compilerversion)
					h.AppendChild(s)
		*/

		/*
		   <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/codemirror/5.65.5/codemirror.min.css">
		   <script src="https://cdnjs.cloudflare.com/ajax/libs/codemirror/5.65.5/codemirror.min.js"></script>
		   <!-- Include the Go mode for CodeMirror -->
		   <script src="https://cdnjs.cloudflare.com/ajax/libs/codemirror/5.65.5/mode/go/go.min.js"></script>


		*/

		// add code mirror script with go mode
		cm := d.Script.WithID("codemirror")
		SetAttribute(cm.AsElement(), "src", "https://cdnjs.cloudflare.com/ajax/libs/codemirror/5.65.5/codemirror.min.js")
		h.AppendChild(cm)

		// add go mode for code mirror
		gomode := d.Script.WithID("gomode")
		SetAttribute(gomode.AsElement(), "src", "https://cdnjs.cloudflare.com/ajax/libs/codemirror/5.65.5/mode/go/go.min.js")
		h.AppendChild(gomode)

		// add code mirror css
		cmcss := d.Link.WithID("codemirrorcss")
		SetAttribute(cmcss.AsElement(), "rel", "stylesheet")
		SetAttribute(cmcss.AsElement(), "href", "https://cdnjs.cloudflare.com/ajax/libs/codemirror/5.65.5/codemirror.min.css")
		h.AppendChild(cmcss)

		// css class for compile error line highlighting
		lineerrorcss := d.Style().SetInnerHTML(`
            .cm-error-line {
                background-color: #fdd;
            }
        `)
		h.AppendChild(lineerrorcss)

		// virtual file system script
		v := d.Script().SetInnerHTML(vfs)
		h.AppendChild(v)

		// compiler suite loader script
		c := d.Script().SetInnerHTML(loaderscript)
		h.AppendChild(c)

		importcfglinkGen := js.FuncOf(func(this js.Value, args []js.Value) any {
			mainFilePath := args[0].String()
			importCfgPath := args[1].String()
			outputFilePath := args[2].String()
			return generateImportCfgLink(mainFilePath, importCfgPath, outputFilePath)
		})

		js.Global().Set("decompressArchive", js.FuncOf(decompressArchive))

		js.Global().Set("generateImportCfgLink", importcfglinkGen)

		// prefetch the minimum compiler suite in the background

		d.TriggerEvent("replInitialized")
	}

	return ReplElement{repl.AsElement()}.activatecontentarea()
}

// Let's add the event listeners for the text content of the editable div
func (r ReplElement) activatecontentarea() ReplElement {
	// The contentarea should be hijacked to prevent the default behavior of the tab key
	r.TextArea().AddEventListener("keydown", ui.NewEventHandler(func(evt ui.Event) bool {
		if evt.(KeyboardEvent).Key() == "Tab" {
			evt.PreventDefault()
			r.TextArea().SyncUISyncData("value", WithStrConv(ui.String(r.TextArea().Text()+"\t")))
			return false
		}
		return false
	}))

	// the contentarea should retrieve the text and store it in the text property of the div element
	r.TextArea().AddEventListener("input", ui.NewEventHandler(func(evt ui.Event) bool {
		r.TextArea().SyncUISyncData("value", WithStrConv(evt.Value()))
		return false
	}))

	return r
}

// generateImportCfgLink generates the importcfg.lnk file.
// This is necessary to be able to link the main package with its precompiled dependencies.
func generateImportCfgLink(mainFilePath, importCfgPath, outputFilePath string) error {
	imports, err := extractImports(mainFilePath)
	if err != nil {
		return err
	}
	importCfg, err := os.ReadFile(importCfgPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(importCfg), "\n")
	importMap := make(map[string]string)
	for _, line := range lines {
		parts := strings.Split(line, "=")
		if len(parts) == 2 {
			importMap[parts[0][12:]] = parts[1] // Remove "packagefile " prefix
		}
	}

	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	// Write the special first line for the main package
	_, err = outputFile.WriteString("packagefile command-line-arguments=main.a\n")
	if err != nil {
		return err
	}

	// Write the dependencies
	for _, imp := range imports {
		if path, exists := importMap[imp]; exists {
			_, err := outputFile.WriteString(fmt.Sprintf("packagefile %s=%s\n", imp, path))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// extractImports parses the main.go file and extracts import paths.
func extractImports(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var imports []string
	scanner := bufio.NewScanner(file)
	inImportBlock := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "import (") {
			inImportBlock = true
			continue
		} else if inImportBlock && strings.Contains(line, ")") {
			break
		}

		if inImportBlock || strings.HasPrefix(strings.TrimSpace(line), "import ") {
			trimmed := strings.Trim(line, "\t \"")
			if trimmed != "" && trimmed != "import" {
				imports = append(imports, trimmed)
			}
		}
	}

	return imports, scanner.Err()
}

type replModifier struct{}

var ReplModifier replModifier

func (r replModifier) TextArea(modifiers ...func(*ui.Element) *ui.Element) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		t := ReplElement{e}.TextArea().AsElement()
		for _, m := range modifiers {
			m(t)
		}
		return e
	}
}

func (r replModifier) Output(modifiers ...func(*ui.Element) *ui.Element) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		t := ReplElement{e}.Output().AsElement()
		for _, m := range modifiers {
			m(t)
		}
		return e
	}
}

func (r replModifier) Iframe(modifiers ...func(*ui.Element) *ui.Element) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		t := ReplElement{e}.Iframe().AsElement()
		for _, m := range modifiers {
			m(t)
		}
		return e
	}
}

func (r replModifier) BtnRun(modifiers ...func(*ui.Element) *ui.Element) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		t := ReplElement{e}.BtnRun().AsElement()
		for _, m := range modifiers {
			m(t)
		}
		return e
	}
}

func (r replModifier) BtnFmt(modifiers ...func(*ui.Element) *ui.Element) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		t := ReplElement{e}.BtnFmt().AsElement()
		for _, m := range modifiers {
			m(t)
		}
		return e
	}
}

func (r replModifier) CodeInputContainer(modifiers ...func(*ui.Element) *ui.Element) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		t := ReplElement{e}.CodeInputContainer().AsElement()
		for _, m := range modifiers {
			m(t)
		}
		return e
	}
}

func (r replModifier) ButtonsContainer(modifiers ...func(*ui.Element) *ui.Element) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		t := ReplElement{e}.ButtonsContainer().AsElement()
		for _, m := range modifiers {
			m(t)
		}
		return e
	}
}

var vfs = `

    class IndexedDBFileCache {
            this.dbName = dbName;
            this.storeName = storeName;
            this.db = null;
            this._initDB();
        }

        _initDB() {
            const request = indexedDB.open(this.dbName, 1);
            request.onupgradeneeded = (event) => {
                const db = event.target.result;
                if (!db.objectStoreNames.contains(this.storeName)) {
                    db.createObjectStore(this.storeName);
                }
            };
            request.onsuccess = (event) => {
                this.db = event.target.result;
            };
            request.onerror = (event) => {
                console.error('IndexedDB error:', event.target.errorCode);
            };
        }

        _getFileStore(mode) {
            const tx = this.db.transaction(this.storeName, mode);
            return tx.objectStore(this.storeName);
        }

        cacheFile(path, data) {
            return new Promise((resolve, reject) => {
                const store = this._getFileStore('readwrite');
                const request = store.put(data, path);
                request.onsuccess = () => resolve();
                request.onerror = (event) => reject(event.target.error);
            });
        }

        retrieveFile(path) {
            return new Promise((resolve, reject) => {
                const store = this._getFileStore('readonly');
                const request = store.get(path);
                request.onsuccess = (event) => resolve(event.target.result);
                request.onerror = (event) => reject(event.target.error);
            });
        }

        removeFile(path) {
            return new Promise((resolve, reject) => {
                const store = this._getFileStore('readwrite');
                const request = store.delete(path);
                request.onsuccess = () => resolve();
                request.onerror = (event) => reject(event.target.error);
            });
        }

        clearCache() {
            return new Promise((resolve, reject) => {
                const store = this._getFileStore('readwrite');
                const request = store.clear();
                request.onsuccess = () => resolve();
                request.onerror = (event) => reject(event.target.error);
            });
        }

        async fetchAndCacheFile(path) {
            try {
                const data = await this.retrieveFile(path);
                if (!data) {
                    // File not in cache, fetch from server
                    const response = await fetch(path);
                    if (!response.ok) {
                        throw new Error('Failed to fetch file at ' + path + ': ' + response.statusText);
                    }
                    const buffer = await response.arrayBuffer();
                    await this.cacheFile(path, new Uint8Array(buffer));
                }
            } catch (error) {
                // If there's an error in fetching or caching, rethrow to make it visible to the caller
                throw new Error('Error in fetching and caching file at ' + path + ': ' + error);
            }
        } 
    }

    async function prefetchAndCacheGoPackagesWithVersion() {
        const manifestUrl = '/wasmgc/manifest.json';
    
        try {
            const manifestResponse = await fetch(manifestUrl);
            if (!manifestResponse.ok) {
                throw new Error('Failed to fetch ' + manifestUrl + ': ' + manifestResponse.statusText);
            }
            const manifest = await manifestResponse.json();
            const goVersion = manifest.goversion;
            const cacheDbName = 'goPackagesCacheDB_' + goVersion.replace(/\./g, '_');
            const cache = new IndexedDBFileCache(cacheDbName, 'goFiles');
    
            const minLibraryUrl = '/' + manifest.min_library + '.zip';
            const zipResponse = await fetch(minLibraryUrl);
            if (!zipResponse.ok) {
                throw new Error('Failed to download min_library.zip');
            }
            const zipArrayBuffer = await zipResponse.arrayBuffer();
            const decompressedFiles = await window.decompressArchive(new Uint8Array(zipArrayBuffer));
    
            const cacheTasks = Object.keys(decompressedFiles).map(async (filename) => {
                const fileData = decompressedFiles[filename];
                await cache.cacheFile('/' + manifest.library + '/' + filename, fileData);
            });
    
            const wasmFilenames = ['compile', 'gofmt', 'link'];
            const wasmTasks = wasmFilenames.map(async (key) => {
                const wasmUrl = '/' + manifest[key];
                const response = await fetch(wasmUrl);
                if (!response.ok) {
                    throw new Error('Failed to download ' + wasmUrl);
                }
                const arrayBuffer = await response.arrayBuffer();
                await cache.cacheFile(wasmUrl, new Uint8Array(arrayBuffer));
            });

            // Fetch and cache the importcfg file
            const importcfgUrl = '/' + manifest.importcfg;
            const importcfgResponse = await fetch(importcfgUrl);
            if (!importcfgResponse.ok) {
                throw new Error('Failed to download ' + importcfgUrl);
            }
            const importcfgArrayBuffer = await importcfgResponse.arrayBuffer();
            const importcfgTask = cache.cacheFile(importcfgUrl, new Uint8Array(importcfgArrayBuffer));
        
            await Promise.all([...cacheTasks, ...wasmTasks, importcfgTask]);

            globalThis.goVersion = goVersion;

        } catch (error) {
            console.error('Error prefetching Go packages:', error);
        }
    }       
}
`

var loaderscript = `
    "use strict";
    
    (function() {
        globalThis.compiledWasmModules = globalThis.compiledWasmModules || {};
        const enc = new TextEncoder("utf-8");
        const dec = new TextDecoder("utf-8");

        globalThis.goStdout = '';
        globalThis.goStderr = '';

        // working directory should be base? (DEBUG TODO)
        let wd = '/';

        const process = globalThis.process;
        process.cwd = () => wd;
        process.chdir = (dir) => {
            wd = dir;
        };


        let cmds = {};
        const manifestUrl = '${PkgDirURL}/'; // Assuming the manifest file is at this URL
    

        const absPath = (path) => {
            // if path is absolute, this is noop
            // if path is relative, prepend the working directory
            return path.startsWith('/') ? path : wd + path;
        };

        const isDirectory = (path) => {
            return path.endsWith('/');
        };

        const exec = (wasm, args) => new Promise((resolve, reject) => {
            const go = new Go();
            go.exit = resolve;
            go.argv = go.argv.concat(args || ["main.go"]);
            WebAssembly.instantiate(wasm, go.importObject).then((result) => go.run(result.instance)).catch(reject);
        });

        const ENOENT = () => {
            const err = new Error('no such file or directory');
            err.code = 'ENOENT';
            return err;
        }

        const ENOSYS = () => {
            const err = new Error('function not implemented');
            err.code = 'ENOSYS';
            return err;
        }

        const EEXIST = () => {
            const err = new Error('file already exists');
            err.code = 'EEXIST';
            return err;
        }

        const cache = new IndexedDBFileCache();
        const vfs = globalThis.fs;

        let nextFd = 42; // Start from 42 because 0, 1, and 2 are usually reserved for stdin, stdout, and stderr
        const openFiles = new Map(); // Map of file descriptors

        const updateParentDirectory = (filePath) => {
            const parentDir = getParentDirectory(filePath);
            if (!parentDir) return; // If there's no parent directory, nothing to update
        
            if (!filesystem[parentDir]) {
                console.warn("Parent directory does not exist for", filePath);
                return;
            }
        
            const fileName = filePath.split('/').pop(); // Extract the file or directory name
            if (!filesystem[parentDir].contents.includes(fileName)) {
                filesystem[parentDir].contents.push(fileName);
            }
        };
        
        const getParentDirectory = (filePath) => {
            if (!filePath.includes('/')) return ''; // No parent directory
        
            const pathParts = filePath.split('/');
            pathParts.pop(); // Remove the last part (file or directory name)
            return pathParts.join('/') + (pathParts.length > 1 ? '/' : ''); // Rejoin the path
        };
        

        // Constants for open flags
        const O_CREAT = 1; // Binary 001
        const O_RDONLY = 2; // Binary 010
        const O_WRONLY = 4; // Binary 100
        const O_RDWR = 6;   // Binary 110
        const O_TRUNC = 8;  // Binary 1000
        const O_APPEND = 16; // Binary 10000

        fs.open = (path, flags, mode, callback) => {
            path = absPath(path);
        
            // Handle file creation if O_CREAT is set
            if (flags & O_CREAT && !filesystem[path]) {
                filesystem[path] = { content: '' }; // Create a new file with empty content
                updateParentDirectory(path);
            }
        
            // If file doesn't exist and O_CREAT is not set, return an error
            if (!filesystem[path]) {
                callback(ENOENT()); // No such file or directory
                return;
            }
        
            // Handle file truncation if O_TRUNC is set
            if (flags & O_TRUNC) {
                filesystem[path].content = ''; // Truncate the file content
            }
        
            // Create and store the file descriptor
            const fd = nextFd++;
            const fileDetails = {
                path: path,
                flags: flags,
                position: (flags & O_APPEND) ? filesystem[path].content.length : 0, // Set position for append mode
            };
            openFiles.set(fd, fileDetails);
        
            callback(null, fd);
        };

        fs.close = (fd, callback) => {
            if (!openFiles.has(fd)) {
                callback(new Error('Invalid file descriptor'));
                return;
            }
            openFiles.delete(fd);
            callback(null);
        };

        fs.read = (fd, buffer, offset, length, position, callback) => {
            if (!openFiles.has(fd)) {
                callback(new Error('Invalid file descriptor'));
                return;
            }

            const file = openFiles.get(fd);
            if (!(file.flags & O_RDONLY)) {
                callback(new Error('File not opened with read access'));
                return;
            }

            const fileContent = filesystem[file.path].content;
            if (position !== null) {
                file.position = position;
            }
            const bytesRead = Math.min(Buffer.from(fileContent).copy(buffer, offset, file.position, file.position + length), length);
            file.position += bytesRead;
            callback(null, bytesRead, buffer);
        };

        fs.write = (fd, buffer, offset, length, position, callback) => {
            if (!openFiles.has(fd)) {
                callback(new Error('Invalid file descriptor'));
                return;
            }
        
            const file = openFiles.get(fd);

            if (fd === 1 || fd === 2) { // Handle stdout and stderr
                const output = Buffer.from(buffer, offset, length).toString('utf8');
                if (fd === 1) {
                    globalThis.goStdout += output;  // Append to global stdout
                } else {
                    globalThis.goStderr += output;  // Append to global stderr
                }
                callback(null, length, buffer);
                return;
            }
        

            if (!(file.flags & O_WRONLY)) {
                callback(new Error('File not opened with write access'));
                return;
            }

            const data = Buffer.from(buffer).toString('utf8', offset, offset + length);
            if (position !== null) {
                file.position = position;
            }
            if (file.flags & O_APPEND) {
                file.position = filesystem[file.path].content.length; // Append to the end
            }
            const bytesWritten = data.length;
            filesystem[file.path].content = filesystem[file.path].content.slice(0, file.position) + data + filesystem[file.path].content.slice(file.position + bytesWritten);
            file.position += bytesWritten;
            callback(null, bytesWritten, buffer);
        };

        fs.writeSync = (fd, buffer) => {
            if (!openFiles.has(fd)) {
                throw new Error('Invalid file descriptor');
            }

            const file = openFiles.get(fd);

            // Handle stdout and stderr
            if (fd === 1 || fd === 2) {
                const output = Buffer.from(buffer).toString('utf8');
                if (fd === 1) { // stdout
                    globalThis.goStdout += output;  // Append to global stdout
                } else { // stderr
                    globalThis.goStderr += output;  // Append to global stderr
                }
                return buffer.length;
            }

            if (!(file.flags & O_WRONLY)) {
                throw new Error('File not opened with write access');
            }

            const data = Buffer.from(buffer).toString('utf8');
            if (file.flags & O_APPEND) {
                file.position = filesystem[file.path].content.length; // Append to the end
            }
            const bytesWritten = data.length;
            filesystem[file.path].content = filesystem[file.path].content.slice(0, file.position) + data + filesystem[file.path].content.slice(file.position + bytesWritten);
            file.position += bytesWritten;
            return bytesWritten;
        };    
        

        fs.mkdir = (path, perm, callback, createIntermediateDirs = false) => {
            path = absPath(path);
            if (!path.endsWith('/')) {
                path += '/'; // Ensure path ends with a slash
            }

            const pathParts = path.split('/').filter(p => p);
            let currentPath = '/';

            for (const part of pathParts) {
                currentPath += part + '/';
                if (!filesystem[currentPath]) {
                    if (createIntermediateDirs) {
                        // Create intermediate directory
                        filesystem[currentPath] = { contents: [] };
                        updateParentDirectory(currentPath);
                    } else {
                        callback(ENOENT()); // No such file or directory
                        return;
                    }
                }
            }
            callback(null);
        };

        
        

        fs.rmdir = (path, callback) => {
            path = absPath(path);
            if (!isDirectory(path) || !filesystem[path]) {
                callback(new Error('ENOTDIR')); // Not a directory or does not exist
            } else {
                delete filesystem[path]; // Remove the directory
                callback(null);
            }
        };
        

        fs.unlink = (path, callback) => {
            path = absPath(path);
            if (isDirectory(path) || !filesystem[path]) {
                callback(new Error('EISDIR')); // Is a directory or does not exist
            } else {
                delete filesystem[path]; // Remove the file
                callback(null);
            }
        };
        

        fs.rename = (oldPath, newPath, callback) => {
            oldPath = absPath(oldPath);
            newPath = absPath(newPath);
            if (!filesystem[oldPath]) {
                callback(ENOENT())); // Source does not exist
            } else if (filesystem[newPath]) {
                callback(EEXIST()); // Destination already exists
            } else {
                filesystem[newPath] = filesystem[oldPath]; // Rename/move
                delete filesystem[oldPath];
                callback(null);
            }
        };

        fs.utimes = (path, atime, mtime, callback) => {
            path = absPath(path);
            if (!filesystem[path]) {
                callback(ENOENT()); // No such file or directory
                return;
            }
        
            if (!filesystem[path].metadata) {
                filesystem[path].metadata = {};
            }
        
            filesystem[path].metadata.atime = atime;
            filesystem[path].metadata.mtime = mtime;
        
            callback(null);
        };

        fs.stat = fs.lstat = (path, callback) => {
            path = absPath(path);if (e.keyCode == 13) { // enter
                if (e.shiftKey) { // +shift
                  run();
                  e.preventDefault();
                  return false;
                } if (e.ctrlKey) { // +control
                  fmt();
                  e.preventDefault();
                } else {
                  autoindent(e.target);
                }
              }
            const file = filesystem[path];
        
            if (file === undefined) {
                callback(ENOENT());
                return;
            }l
        
            // Determine if the path is a directory
            const isDirectory = path.endsWith('/');
        
            // Set mode bits: 0o40000 for directories, 0o10000 for files
            let mode = isDirectory ? 0o40000 : 0o10000;
        
            callback(null, {
                mode: mode,
                dev: 0,
                ino: 0,
                nlink: 0,
                uid: 0,
                gid: 0,
                rdev: 0,
                size: isDirectory ? 0 : file.content.length,
                blksize: 0,
                blocks: 0,
                atimeMs: file.metadata?.atime?.getTime() || Date.now(),
                mtimeMs: file.metadata?.mtime?.getTime() || Date.now(),
                ctimeMs: file.metadata?.ctime?.getTime() || Date.now(),
                isDirectory: () => isDirectory,
            });
        };
        

        fs.fstat = (fd, callback) => {
            if (!openFiles.has(fd)) {
                callback(new Error('Invalid file descriptor'));
                return;
            }
        
            const file = openFiles.get(fd);
            const filePath = file.path;
            const fileData = filesystem[filePath];
        
            if (!fileData) {
                callback(new Error('File does not exist'));
                return;
            }
        
            const isDirectory = filePath.endsWith('/');
            const mode = isDirectory ? 0o40000 : 0o10000; // Directory or regular file
        
            callback(null, {
                mode: mode,
                size: isDirectory ? 0 : fileData.content.length,
                atimeMs: fileData.metadata?.atime?.getTime() || Date.now(),
                mtimeMs: fileData.metadata?.mtime?.getTime() || Date.now(),
                ctimeMs: fileData.metadata?.ctime?.getTime() || Date.now(),
                isDirectory: () => isDirectory,
                // Other properties as in stat
            });
        };
        
        
        // check status of cache and update if necessary via  prefetchAndCacheGoPackagesWithVersion()
        // Modify readFromGoFS and writeToGoFS to use the cache.
        // Unlike traditional cache, this is reversed:
        //     - check if the file is in the vfs first
        //     - if it is, return it
        //     - if it is not, check the cache
        //     - if it is, put it in the vfs and return it
        //     - if it is not and this is a compiler resource, fetch it, put it in the cache and in the vfs and return it
        //     - if it is not and this is not a compiler asset resource, return an error.

        const readFromGoFS = async (path) => {
            path = absPath(path);
            // Check if the file is in the vfs first
            if (fs.hasOwnProperty(path)) {
                return fs[path];
            } else {
                // If it is not in the vfs, check the cache
                try {
                    const data = await cache.retrieveFile(path);
                    if (data) {
                        // If it is in the cache, put it in the vfs and return it
                        fs[path] = data;
                        return data;
                    } else {
                        // If it is not in the cache and this is a compiler resource
                        if (isCompilerAsset(path)) {
                            // Fetch it, put it in the cache and in the vfs, and return it
                            const fetchedData = await fetchAndCacheFile(path); // Assumes fetchAndCacheFile is implemented to fetch and cache
                            fs[path] = fetchedData;
                            return fetchedData;
                        } else {
                            // If it is not and this is not a compiler asset resource, throw an error
                            throw new Error('File not found and is not a compiler asset resource: ' + path);
                        }
                    }
                } catch (error) {
                    console.error('Error reading from GoFS:', error);
                    throw error; // Rethrow or handle as needed
                }
            }
        };

        const writeToGoFS = (path, data) => {
            path = absPath(path);
            if (typeof data === 'string') {
                data = enc.encode(data); // Assuming 'enc' is previously defined for encoding strings
            }
            // Simply write to the vfs
            fs[path] = data;
        };

        // Helper function to determine if a path belongs to compiler assets
        function isCompilerAsset(path) {
            // Implement logic to determine if the path belongs to compiler assets
            // For simplicity, let's assume anything under '/wasmgc/' is a compiler asset
            return path.startsWith('/wasmgc/');
        }

        function getPackageName(path) {
            const parts = path.split(/[\/.]/);
            return parts.slice(1, parts.length - 1).join('/');
        }

        

        // TODO Need to generate importcfg.link: the function should already be registered from wasm go

        function formatGoCode(replElementID, timeout = 30000) {
            const textareaID = replElementID + "-textarea";
            const outputID = replElementID + "-output";
            let timeoutReached = false;
            let formatPromise;
            globalThis.goStdout = '';
            globalThis.goStderr = '';
        
            // Timeout handler
            const timeoutHandler = setTimeout(() => {
                timeoutReached = true;
                document.getElementById(outputID).textContent = 'Formatting exceeded the time limit.';
                if (formatPromise && formatPromise.cancel) {
                    formatPromise.cancel();
                }
            }, timeout);
        
            formatPromise = new Promise((resolve, reject) => {
                const goCode = document.getElementById(textareaID).value;
                writeToGoFS('/main.go', enc.encode(goCode)); // Assuming writeToGoFS is defined
                exec(cmds['gofmt'], ['-w', '/main.go'])
                    .then(() => {
                        if (!timeoutReached) {
                            clearTimeout(timeoutHandler);
        
                            if (globalThis.goStderr) {
                                document.getElementById(outputID).textContent = globalThis.goStderr;
                                reject(new Error('Formatting failed'));
                                return;
                            }
        
                            let formattedCode = readFromGoFS('/main.go'); 
                            document.getElementById(textareaID).value = dec.decode(formattedCode); 
                            resolve();
                        }
                    })
                    .catch(error => {
                        if (!timeoutReached) {
                            clearTimeout(timeoutHandler);
                            // document.getElementById(outputID).textContent = 'Formatting failed: ' + error;
                            reject(error);
                        }
                    });
            });
        
            return formatPromise;
        }
        

        function runGoCode(replElementID, timeout = 30000) {
            const textareaID = replElementID + "-textarea";
            const outputID = replElementID + "-output";
            const iframeID = replElementID + "-iframe";
            let timeoutReached = false;
            let compilationPromise;
            globalThis.goStdout = '';
            globalThis.goStderr = '';
        
            // Clear the output area
            clearHighlights(CodeMirrorInstances[textareaID]);
            document.getElementById(outputID).textContent = '';
        
            // Timeout handler
            const timeoutHandler = setTimeout(() => {
                timeoutReached = true;
                document.getElementById(outputID).textContent = 'Compilation exceeded the time limit.';
                if (compilationPromise && compilationPromise.cancel) {
                    compilationPromise.cancel();
                }
            }, timeout);
        
            compilationPromise = new Promise((resolve, reject) => {
                const goCode = document.getElementById(textareaID).value;
                writeToGoFS('/main.go', enc.encode(goCode));
                exec(cmds['compile'], ['-p', 'main', '-complete', '-dwarf=false', '-pack', '-importcfg', '/wasmgc/importcfg', 'main.go'])
                .then(() => {
                    if (globalThis.goStderr) {
                        document.getElementById(outputID).textContent = globalThis.goStderr;
                        reject(new Error('Compilation failed'));
                        processAndHighlightErrors(globalThis.goStderr, replElementID);
                        return;
                    }

                    // Reset stdout and stderr before linking
                    globalThis.goStdout = '';
                    globalThis.goStderr = '';

                    globalThis.generateImportCfgLink('/main.go', '/wasmgc/importcfg', '/wasmgc/importcfg.link');

                    return exec(cmds['link'], ['-importcfg', '/wasmgc/importcfg.link', '-buildmode=exe', 'main.a']);
                })
                .then(output => {
                    if (globalThis.goStderr) {
                        document.getElementById(outputID).textContent = globalThis.goStderr;
                        reject(new Error('Linking failed'));
                        processAndHighlightErrors(globalThis.goStderr, replElementID);
                        return;
                    }

                    if (!timeoutReached) {
                        clearTimeout(timeoutHandler);
                        if (output) {
                            // Display compilation errors
                            document.getElementById(outputID).textContent = new TextDecoder().decode(output);
                            reject();
                        } else {
                            // Handle successful compilation
                            loadWasmInIframe(iframeID, output);
                            resolve();
                        }
                    }
                })
                .catch(error => {
                    if (!timeoutReached) {
                        clearTimeout(timeoutHandler);
                        document.getElementById(outputID).textContent = 'Compilation failed: ' + error;
                        reject();
                    }
                });
            });
            return formatGoCode(replElementID).then(() => compilationPromise);
        }
        
        function loadWasmInIframe(iframeID, output) {
            // Create an HTML string that includes the necessary logic to load and run the main.wasm
            globalThis.compiledWasmModules[iframeID] = output;
            const html = ` + "`" + iframepage + "`;" + `
            
            // Let's set the src doc of the iframe to the html string
            document.getElementById(iframeID).srcdoc = html;
        }
    
    
    })();

    function createGoEditor(replElementID) {
        textareaID = replElementID + "-textarea";
        var editor = CodeMirror.fromTextArea(document.getElementById(textareaID), {
            mode: "text/x-go",
            lineNumbers: true,
            indentUnit: 4,
            matchBrackets: true,
            autoCloseBrackets: true,
            theme: "default"
        });
    
        // Set up keydown event listener
        editor.on("keydown", function(editor, e) {
            if (e.keyCode === 13) { // Enter key
                if (e.shiftKey) {
                    RunGoCode(replElementID);
                    e.preventDefault(); // Prevent the default action (new line)
                    return false;
                } else if (e.ctrlKey) {
                    // Format the code
                    formatGoCode(replElementID);
                    e.preventDefault(); // Prevent the default action (new line)
                }
            }
        });

        // let's store the editor instance in a map on the globalThis object
        globalThis.cmgoEditors = globalThis.goEditors || {};
        globalThis.cmgoEditors[replElementID] = editor;
            
        return editor;
    }
    
    
    // Highlight error lines
    function lineHighlight(error, editorInstance) {
        var regex = /.*?:(\d+):\d+: .*/g;
        var match;
        while ((match = regex.exec(error)) !== null) {
            editorInstance.addLineClass(parseInt(match[1])-1, 'background', 'cm-error-line');
        }
    }
    
    // Clear error highlights
    function clearHighlights(editorInstance) {
        editorInstance.eachLine((line) => {
            editorInstance.removeLineClass(line, 'background', 'cm-error-line');
        });
    }

    // Function to highlight errors and clear previous highlights
    function processAndHighlightErrors(gcStdErr, replElementID) {
        // Assuming you can access the CodeMirror instance directly
        // If you store it globally or in a way that's accessible here
        const editorInstance = globalThis.cmgoEditors[replElementID]; // Example access method

        // Clear previous error highlights
        clearHighlights(editorInstance);

        // Highlight new errors
        lineHighlight(gcStdErr, editorInstance);
    }
    
`
var iframepage = `
                    <!DOCTYPE html>
                    <html>
                        <head>
                            <meta charset="utf-8">
                            <title>GoWasm repl</title>
                            <script id = "goruntime>
                                let wasmLoadedResolver, loadEventResolver;
                                window.wasmLoaded = new Promise(resolve => wasmLoadedResolver = resolve);
                                window.loadEventFired = new Promise(resolve => loadEventResolver = resolve);
                            
                                window.onWasmDone = function() {
                                    wasmLoadedResolver();
                                }
                            
                                window.addEventListener('load', () => {
                                    loadEventResolver();
                                });

                                const wasm = window.compiledWasmModules['${iframeID}'];
                                const go = new Go();
 
                                WebAssembly.instantiate(wasm, go.importObject).then((result) => go.run(result.instance));

                                Promise.all([window.wasmLoaded, window.loadEventFired]).then(() => {
                                    setTimeout(() => {
                                        console.log("about to dispatch PageReady event...");
                                        window.dispatchEvent(new Event('PageReady'));
                                    }, 50);
                                });
                            </script>
                        </head>
                        <body>
                        </body>
                    </html>
                `
