package repl

import (
	"github.com/atdiar/particleui"
	. "github.com/atdiar/particleui/drivers/js"
)

var (
	CompileWasmURL string
	LinkWasmURL    string
	GofmtWasmURL   string
	PkgDirURL      string
)

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

// CodeInputContainer returns the div element that holds the textarea and the buttons
func (r ReplElement) CodeInputContainer() DivElement {
	return DivElement{GetDocument(r.AsElement()).GetElementById(r.AsElement().ID + "-codeinput")}
}

// ButtonsContainer returns the div element that holds the buttons
func (r ReplElement) ButtonsContainer() DivElement {
	return DivElement{GetDocument(r.AsElement()).GetElementById(r.AsElement().ID + "-buttons")}
}

// Repl returns a ReplElement which is able to compile and render UI code in the browser locally.
func Repl(d *Document, id string, compilerversion string, options ...string) ReplElement {
	repl := d.Div.WithID(id, options...)
	// TODO append the gcversion script with the value attribute set to the go compiler version in use
	textarea := d.TextArea.WithID(id + "-textarea")
	outputfield := d.Div.WithID(id + "-output")
	iframeresult := d.Iframe.WithID(id+"-iframe", "about:blank")

	btnRun := d.Button.WithID(id+"-run", "input")
	btnFmt := d.Button.WithID(id+"-fmt", "input")

	// double binding between repl and textarea subcomponent.
	// on sync mutation event, the repl top div gets synced as well. (caused by input events for instance)
	// if the repl's code is set from somewhere however, the textarea will get set to the same value too.
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

	// Let's add the event listeners for the buttons and the textarea
	// The textarea should be hijacked to prevent the default behavior of the tab key
	// which is to move the focus to the next element in the document.

	// let's add the scripts for the virtual filesystem and the compiler suite loader
	// if they haven't already been added to the document Head.
	// A script with id "gcversion" that holds the compiler version string should also be appended.

	if _, ok := d.GetEventValue("replInitialized"); !ok {
		h := d.Head()

		// gcversion script
		s := d.Script.WithID("gcversion")
		SetAttribute(s.AsElement(), "value", compilerversion)
		h.AppendChild(s)

		// virtual file system script
		v := d.Script().SetInnerHTML(vfs)
		h.AppendChild(v)

		// compiler suite loader script
		c := d.Script().SetInnerHTML(loaderscript)
		h.AppendChild(c)

		d.TriggerEvent("replInitialized")
	}

	return ReplElement{repl.AsElement()}
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
        constructor(dbName = 'fileCacheDB', storeName = 'files') {
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

        fetchAndCacheFile(path) {
            return this.retrieveFile(path).then(data => {
                if (data) {
                    return data; // File is already in cache
                } else {
                    // File not in cache, fetch from server
                    return fetch(path).then(response => {
                        if (!response.ok) {
                            // Invalidate the cache if a critical file fetch fails
                            this.invalidateCache().then(() => {
                                // Optionally set the version to an invalid value
                                const invalidVersion = -1;
                                return this.cacheFile('cache-version', new TextEncoder().encode(invalidVersion.toString()));
                            }).then(() => {
                                throw new Error(` + "`" + `Network response was not ok for ${path}` + "`" + `);
                            });
                        }
                        return response.arrayBuffer();
                    }).then(buffer => {
                        return this.cacheFile(path, new Uint8Array(buffer)).then(() => buffer);
                    });
                }
            });
        }
        
        
    
        checkAndUpdateVersion(newVersion) {
            const versionKey = 'cache-version';
            return this.retrieveFile(versionKey).then(data => {
                const currentVersion = data ? parseInt(new TextDecoder().decode(data), 10) : 0;
                if (newVersion !== currentVersion) {
                    // Invalidate cache and update version
                    return this.clearCache().then(() => {
                        return this.cacheFile(versionKey, new TextEncoder().encode(newVersion.toString()));
                    });
                }
            });
        }
    }

    
    
}
`

var loaderscript = `
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
	const manifestUrl = '${PkgDirURL}/manifest.txt'; // Assuming the manifest file is at this URL
 

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
            callback(new Error("ENOENT")); // No such file or directory
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
                    callback(new Error("ENOENT")); // No such file or directory
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
            callback(new Error('ENOENT')); // Source does not exist
        } else if (filesystem[newPath]) {
            callback(new Error('EEXIST')); // Destination already exists
        } else {
            filesystem[newPath] = filesystem[oldPath]; // Rename/move
            delete filesystem[oldPath];
            callback(null);
        }
    };

    fs.utimes = (path, atime, mtime, callback) => {
        path = absPath(path);
        if (!filesystem[path]) {
            callback(new Error('ENOENT')); // No such file or directory
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
        path = absPath(path);
        const file = filesystem[path];
    
        if (file === undefined) {
            const err = new Error('no such file');
            err.code = 'ENOENT';
            callback(err);
            return;
        }
    
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
    
    
    
    

    const readFromGoFS = (path) => {
        path = absPath(path);
        return fs[path];
    };

    const writeToGoFS = (path, data) => {
        path = absPath(path);
        if typeof data === 'string' {
            data = enc.encode(data);
        }
        fs[path] = data;
    };


	var gcversionScript = document.getElementById('gcversion');
	newCompilerVersion = gcversionScript ? (gcversionScript.getAttribute('value') || 0) : 0;

	function getPackageName(path) {
		const parts = path.split(/[\/.]/);
		return parts.slice(1, parts.length - 1).join('/');
	}

	cache.checkAndUpdateVersion(newCompilerVersion, (err) => {
		if (err) {
			console.error('Error updating version:', err);
			return;
		}

		// Fetch and parse the manifest file
		fetch(manifestUrl)
			.then(response => response.text())
			.then(text => {
				const manifest = {};
				text.split("\\n").forEach(line => {
					const [src, dst] = line.split(" -> ");
					if (src && dst) {
						manifest[src] = dst;
					}
				});
				return manifest;
			})
			.then(manifest => {
				const packagePaths = Object.values(manifest);
                return Promise.all(packagePaths.map(path => cache.fetchAndCacheFile(path)));

				/*
                Promise.all(
					packagePaths.map((path) => cache.fetchAndCacheFile(path))
					.concat(
						['compile', 'link', 'gofmt'].map(cmd => 
							cache.fetchAndCacheFile('cmd/' + cmd + '.wasm').then(buf => {
								cmds[cmd] = new Uint8Array(buf);
							})
						)
					)
                */

				).then(() => {
                    // ... populate the in-memory filesystem

					// Dynamically create the contents of /importcfg
					const importcfgContent = packagePaths.map(p => ` + "`" + `packagefile ${getPackageName(p)}=${p}` + "`" + `).join("\\n");
					writeToGoFS('/importcfg', enc.encode(importcfgContent));

					// Dynamically create the contents of /importcfg.link
					const importcfgLinkContent = "packagefile command-line-arguments=main.a\\n" + importcfgContent;
					writeToGoFS('/importcfg.link', enc.encode(importcfgLinkContent));
				})
                .catch(error => console.error('Error processing manifest:', error));
			});
	});

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
                        document.getElementById(outputID).textContent = 'Formatting failed: ' + error;
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
            exec(cmds['compile'], ['-o', '/main.wasm', '/main.go'])
            .then(() => {
                if (globalThis.goStderr) {
                    document.getElementById(outputID).textContent = globalThis.goStderr;
                    reject(new Error('Compilation failed'));
                    return;
                }

                // Reset stdout and stderr before linking
                globalThis.goStdout = '';
                globalThis.goStderr = '';

                return exec(cmds['link'], ['-o', '/main.wasm', '/main.o']);
            })
            .then(output => {
                if (globalThis.goStderr) {
                    document.getElementById(outputID).textContent = globalThis.goStderr;
                    reject(new Error('Linking failed'));
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


