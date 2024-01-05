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

func Repl(d *Document) *ReplElement {
	e := d.Div()
	// TODO append the gcversion script with the value attriobute set to the go compiler version in use


	return &ReplElement{e.AsElement()}
}


var vfs = `

class VirtualFileSystem {
    constructor() {
        this.dbName = 'virtualFileSystem';
        this.storeName = 'files';
        this.db = null;
        this.openFiles = new Map();
        this.nextFd = 1000;
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

    writeFile(path, data, callback) {
        const store = this._getFileStore('readwrite');
        const request = store.put(data, path);
        request.onsuccess = () => callback(null);
        request.onerror = (event) => callback(event.target.error);
    }

    readFile(path, callback) {
        const store = this._getFileStore('readonly');
        const request = store.get(path);
        request.onsuccess = (event) => callback(null, event.target.result);
        request.onerror = (event) => callback(event.target.error);
    }

    open(path, flags, mode, callback) {
        const fd = this.nextFd++;
        this.openFiles.set(fd, { path, flags, mode, position: 0 });
        callback(null, fd);
    }

    close(fd, callback) {
        if (!this.openFiles.has(fd)) {
            callback(new Error('Invalid file descriptor'));
            return;
        }
        this.openFiles.delete(fd);
        callback(null);
    }

    read(fd, buffer, offset, length, position, callback) {
        const file = this.openFiles.get(fd);
        if (!file) {
            callback(new Error('Invalid file descriptor'));
            return;
        }

        this.readFile(file.path, (err, data) => {
            if (err) {
                callback(err);
                return;
            }

            const fileData = new Uint8Array(data);
            const end = position !== null ? position + length : file.position + length;
            const readData = fileData.subarray(file.position, end);

            buffer.set(readData, offset);
            file.position = end;
            callback(null, readData.length, buffer);
        });
    }

    write(fd, buffer, offset, length, position, callback) {
        const file = this.openFiles.get(fd);
        if (!file) {
            callback(new Error('Invalid file descriptor'));
            return;
        }

        this.readFile(file.path, (err, data) => {
            if (err && err.message !== 'File not found') {
                callback(err);
                return;
            }

            const fileData = data ? new Uint8Array(data) : new Uint8Array(0);
            const end = position !== null ? position + length : file.position + length;
            const newFileData = new Uint8Array(Math.max(fileData.length, end));
            newFileData.set(fileData);
            newFileData.set(buffer.subarray(offset, offset + length), file.position);

            this.writeFile(file.path, newFileData, (writeErr) => {
                if (writeErr) {
                    callback(writeErr);
                    return;
                }

                file.position = end;
                callback(null);
            });
        });
    }

    fetchAndCacheFile(path) {
        return new Promise((resolve, reject) => {
            this.readFile(path, (err, data) => {
                if (!err && data) {
                    resolve(data); // File is in cache
                } else {
                    // File not in cache, fetch from server
                    fetch(path).then(response => {
                        if (!response.ok) {
                            throw new Error("Network response was not ok for ${path}");
                        }
                        return response.arrayBuffer();
                    }).then(buffer => {
                        // Cache the fetched file
                        this.writeFile(path, new Uint8Array(buffer), (writeErr) => {
                            if (writeErr) {
                                reject(writeErr);
                            } else {
                                resolve(buffer);
                            }
                        });
                    }).catch(fetchErr => {
                        reject(fetchErr);
                    });
                }
            });
        });
    }

    checkAndUpdateVersion(newVersion, callback) {
        const versionKey = 'vfs-version';
        this.readFile(versionKey, (err, data) => {
            const currentVersion = data ? parseInt(new TextDecoder().decode(data), 10) : 0;
            if (newVersion > currentVersion) {
                // Invalidate cache and update version
                this.invalidateCache(() => {
                    this.writeFile(versionKey, new TextEncoder().encode(newVersion.toString()), callback);
                });
            } else {
                callback(null); // No update needed
            }
        });
    }

    invalidateCache(callback) {
        const store = this._getFileStore('readwrite');
        const request = store.clear(); // Clear all data in the store
        request.onsuccess = () => callback(null);
        request.onerror = (event) => callback(event.target.error);
    }

    // Additional methods for handling file descriptors and other operations could be added here
	if needed.
}
`


var loaderscript = `

	let cmds = {};
	const manifestUrl = '${PkgDirURL}/manifest.txt'; // Assuming the manifest file is at this URL

	const exec = (wasm, args) => new Promise((resolve, reject) => {
		const go = new Go();
		go.exit = resolve;
		go.argv = go.argv.concat(args || []);
		WebAssembly.instantiate(wasm, go.importObject).then((result) => go.run(result.instance)).catch(reject);
	});

	const vfs = new VirtualFileSystem();

	var gcversionScript = document.getElementById('gcversion');
	newCompilerVersion = gcversionScript ? (gcversionScript.getAttribute('value') || 0) : 0;

	function getPackageName(path) {
		const parts = path.split(/[\/.]/);
		return parts.slice(1, parts.length - 1).join('/');
	}

	vfs.checkAndUpdateVersion(newCompilerVersion, (err) => {
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

				// Fetch and cache files based on the manifest
				Promise.all(
					packagePaths.map((path) => vfs.fetchAndCacheFile(path))
					.concat(
						['compile', 'link', 'gofmt'].map(cmd => 
							vfs.fetchAndCacheFile('cmd/' + cmd + '.wasm').then(buf => {
								cmds[cmd] = new Uint8Array(buf);
							})
						)
					)
				).then(() => {
					const encoder = new TextEncoder('utf-8');

					// Dynamically create the contents of /importcfg
					const importcfgContent = packagePaths.map(p => `+"`"+`packagefile ${getPackageName(p)}=${p}`+"`"+`).join("\\n");
					writeFile('/importcfg', encoder.encode(importcfgContent));

					// Dynamically create the contents of /importcfg.link
					const importcfgLinkContent = "packagefile command-line-arguments=main.a\\n" + importcfgContent;
					writeFile('/importcfg.link', encoder.encode(importcfgLinkContent));
				});
			});
	});

`


// TODO compilation and code execution need to remain within a given time bound (implement timeouts)