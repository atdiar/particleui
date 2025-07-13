// Package doc defines the default set of Element constructors, native interfaces,
// events and event handlers, and animation properties used to build js-based UIs.

package doc

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	ui "github.com/atdiar/particleui"
	js "github.com/atdiar/particleui/drivers/js/compat"
)

// TODO implement IndexedDB storage backaend

// jsStore provides a synchronous-like wrapper for JavaScript storage APIs (localStorage, sessionStorage, IndexedDB).
type jsStore struct {
	// The underlying JavaScript object (e.g., window.localStorage, window.sessionStorage, or window.indexedDBSyncInstance)
	store js.Value
	// A flag to indicate if this store is an IndexedDB store, which requires awaiting promises.
	isIndexedDB bool
}

// Storage retrieves a data storage instance based on the storage type.
// The type can be "sessionstorage", "localstorage", or "indexeddb".
// Usually it is not needed since storage is mostly meant to happen automatically at the component level.
// But it can be useful to access the storage directly and manually for some custom use cases.
// An example of that is chunked data loading and processing for data stored in indexedDB and which
// might not fit in memory.
// The API of the storage object is simple, similar to that of the JavaScript storage APIs:
// - Get(key string) (js.Value, bool): Retrieves a value by key. Returns js.Undefined() and false if the key does not exist.
// - Set(key string, value js.Value): Stores a key-value pair.
// - Delete(key string): Removes a key-value pair by key.
func Storage(typ string) *jsStore {
	window := js.Global()
	if !window.Truthy() {
		DEBUG("window is not available, cannot access storage")
		return nil
	}

	var storeInstance js.Value
	var isIndexedDB bool

	switch typ {
	case "sessionstorage":
		storeInstance = window.Get("sessionStorage")
		isIndexedDB = false
	case "localstorage":
		storeInstance = window.Get("localStorage")
		isIndexedDB = false
	case "indexeddb":
		storeInstance = window.Get("indexedDBSyncInstance")
		isIndexedDB = true
		if !storeInstance.Truthy() {
			DEBUG("window.indexedDBSyncInstance is not available, cannot access IndexedDB")
			return nil
		}
	default:
		DEBUG("Unsupported storage type:", typ)
		return nil
	}

	return &jsStore{store: storeInstance, isIndexedDB: isIndexedDB}
}

// Get retrieves a value from the JavaScript storage.
// This function IS SYNCHRONOUS from its Go caller's perspective.
// It blocks the Go routine until the JS operation completes (via awaitPromise).
func (s jsStore) Get(key string) (js.Value, bool) {
	var v js.Value

	if s.isIndexedDB {
		// For IndexedDB, call the async getItem and await its result
		resultPromise := s.store.Call("getItem", key)
		res := <-awaitPromise(resultPromise)
		if res.error != nil {
			DEBUG("IndexedDB Get error for key '%s':", key, res.error)
			// Decide how to handle an error from awaitPromise for 'Get'.
			// For simplicity here, we'll treat it as not found and log the error.
			return js.Undefined(), false
		}
		v = res.Value // This will be js.Undefined() if the key was not found
	} else {
		// For localStorage/sessionStorage, call getItem directly (synchronously)
		v = s.store.Call("getItem", key)
	}

	// For both IndexedDB (undefined for not found) and localStorage/sessionStorage (null for not found)
	// js.Value.Truthy() correctly identifies if a value was found.
	return v, v.Truthy()
}

// Set stores a key-value pair in the JavaScript storage.
// This function IS SYNCHRONOUS from its Go caller's perspective.
// It blocks the Go routine until the JS operation completes (via awaitPromise).
func (s jsStore) Set(key string, value js.Value) error { // Now returns an error
	if s.isIndexedDB {
		// For IndexedDB, call the async setItem and await its completion
		resultPromise := s.store.Call("setItem", key, value)
		res := <-awaitPromise(resultPromise) // We only care about errors here
		if res.error != nil {
			DEBUG("IndexedDB Set error for key '%s':", key, res.error)
			return res.error
		}
	} else {
		// For localStorage/sessionStorage, call setItem directly
		// Stringify value for localStorage/sessionStorage as they only store strings
		JSON := js.Global().Get("JSON")
		res := JSON.Call("stringify", value)
		s.store.Call("setItem", key, res)
	}
	return nil // Success
}

// Delete removes an item from the JavaScript storage.
// This function IS SYNCHRONOUS from its Go caller's perspective.
// It blocks the Go routine until the JS operation completes (via awaitPromise).
func (s jsStore) Delete(key string) error { // Now returns an error

	if s.isIndexedDB {
		// For IndexedDB, call the async removeItem and await its completion
		resultPromise := s.store.Call("removeItem", key)
		res := <-awaitPromise(resultPromise) // We only care about errors here
		if res.error != nil {
			DEBUG("IndexedDB Delete error for key '%s':", key, res.error)
			return res.error
		}
	} else {
		// For localStorage/sessionStorage, call removeItem directly
		s.store.Call("removeItem", key)
	}
	return nil // Success
}

// Clear all items from the storage.
// This function IS SYNCHRONOUS from its Go caller's perspective.
func (s jsStore) Clear() error {
	if s.isIndexedDB {
		resultPromise := s.store.Call("clear")
		res := <-awaitPromise(resultPromise)
		if res.error != nil {
			DEBUG("IndexedDB Clear error:", res.error)
			return res.error
		}
	} else {
		s.store.Call("clear")
	}
	return nil
}

// awaitPromise is a helper to await a JavaScript Promise from Go.
// It blocks the Go routine until the Promise resolves or rejects.
func awaitPromise(p js.Value) chan struct {
	js.Value
	error
} {
	resultChan := make(chan struct {
		js.Value
		error
	}, 1) // Buffered channel so the goroutine doesn't block sending if nobody is listening yet

	go func() { // The actual blocking logic now runs in a new goroutine
		var result js.Value
		var err error

		// Create JS functions for resolve and reject callbacks
		// These functions will be called by JavaScript on the same goroutine as the 'then' call originates from
		// (which is the goroutine this `go func()` is running on).
		var resolve js.Func
		var reject js.Func

		resolve = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			//DEBUG("Promise resolved with:", args[0])
			result = args[0]
			resultChan <- struct {
				js.Value
				error
			}{result, nil} // Send result to channel
			// Release the resolve function to avoid memory leaks
			resolve.Release() // Release the JS function reference
			reject.Release()  // Release the reject function reference
			return nil
		})

		reject = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			// Check if args[0] is an Error object or a simple string
			if len(args) > 0 && args[0].Type() == js.TypeObject && args[0].Get("message").Truthy() {
				err = errors.New(args[0].Get("message").String())
			} else if len(args) > 0 {
				DEBUG("Promise rejected with:", args[0])
				dEBUGJS(args[0])
				err = errors.New(args[0].String())
			} else {
				err = errors.New("unknown promise rejection error")
			}
			resultChan <- struct {
				js.Value
				error
			}{js.Undefined(), err} // Send error to channel
			resolve.Release() // Release the JS function reference
			reject.Release()  // Release the reject function reference
			return nil
		})

		// Call the then method on the promise with our resolve and reject functions
		p.Call("then", resolve, reject)

		// This goroutine will now simply exit. The result will be sent via the channel
		// when the JS promise eventually resolves/rejects and calls resolve/reject JS FuncOf callbacks.
		// There's no need for a 'select {}' or blocking here because the result is sent asynchronously.
	}()

	return resultChan // Return the channel immediately
}

// CompressStringZlib compresses a given string using zlib with BestCompression.
// It returns the compressed data as a byte slice and an error if any occurs.
func CompressStringZlib(input string) ([]byte, error) {
	var b bytes.Buffer
	z, err := zlib.NewWriterLevel(&b, zlib.BestCompression)
	if err != nil {
		return nil, fmt.Errorf("error creating zlib writer: %w", err)
	}

	_, err = z.Write([]byte(input))
	if err != nil {
		return nil, fmt.Errorf("error writing to zlib writer: %w", err)
	}

	// Close the writer to flush all compressed data to the buffer
	err = z.Close()
	if err != nil {
		return nil, fmt.Errorf("error closing zlib writer: %w", err)
	}

	return b.Bytes(), nil
}

// DecompressStringZlib decompresses a zlib-compressed byte slice back into a string.
// It returns the decompressed string and an error if any occurs.
func DecompressStringZlib(compressedData []byte) (string, error) {
	b := bytes.NewReader(compressedData)
	r, err := zlib.NewReader(b)
	if err != nil {
		return "", fmt.Errorf("error creating zlib reader: %w", err)
	}
	defer r.Close() // Ensure the reader is closed

	decompressed := new(bytes.Buffer)
	_, err = decompressed.ReadFrom(r) // ReadFrom is efficient for this
	if err != nil {
		return "", fmt.Errorf("error reading decompressed data: %w", err)
	}

	return decompressed.String(), nil
}

// Let's add sessionstorage and localstorage for Element properties.
// For example, an Element which would have been created with the sessionstorage option
// would have every set properties stored in sessionstorage, available for
// later recovery. It enables to have data that persists runs and loads of a
// web app.

func storer(s string) func(element *ui.Element, category string, propname string, value ui.Value, flags ...bool) {
	return func(element *ui.Element, category string, propname string, value ui.Value, flags ...bool) {
		if category != Namespace.Data && category != Namespace.UI {
			return
		}
		window := js.Global()
		if !window.Truthy() {
			DEBUG("window is not available, cannot store in storage")
			return
		}
		var storeInstance js.Value
		var isIndexedDB bool

		switch s {
		case "sessionstorage":
			storeInstance = window.Get("sessionStorage")
			isIndexedDB = false
		case "localstorage":
			storeInstance = window.Get("localStorage")
			isIndexedDB = false
		case "indexeddb":
			storeInstance = window.Get("indexedDBSyncInstance")
			isIndexedDB = true
			if !storeInstance.Truthy() {
				DEBUG("window.indexedDBSyncInstance is not available, cannot store in IndexedDB")
				return
			}
		default:
			DEBUG("Unsupported storage type:", s)
			return
		}

		store := jsStore{store: storeInstance, isIndexedDB: isIndexedDB}
		_, ok := store.Get("zui-connected")
		if !ok {
			return
		}

		props := make([]any, 0, 64)

		c, ok := element.Properties.Categories[category]
		if !ok {
			props = append(props, propname)
			// log.Print("all props stored...", props) // DEBUG
			v := js.ValueOf(props)
			store.Set(element.ID, v)
		} else {
			for k := range c.Local {
				props = append(props, k)
			}
			v := js.ValueOf(props)
			store.Set(strings.Join([]string{element.ID, category}, "/"), v)
		}

		item := value.RawValue()
		v := stringify(item)
		store.Set(strings.Join([]string{element.ID, category, propname}, "/"), js.ValueOf(v))
		return
	}
}

var sessionstorefn = storer("sessionstorage")
var localstoragefn = storer("localstorage")
var indexeddbstorefn = storer("indexeddb")

func loader(s string) func(e *ui.Element) error {
	return func(e *ui.Element) error {
		window := js.Global()
		if !window.Truthy() {
			if DebugMode {
				// DEBUG is a no-op in production builds
				DEBUG("window is not available, cannot load from storage")
			}
			return nil
		}
		var storeInstance js.Value
		var isIndexedDB bool

		switch s {
		case "sessionstorage":
			storeInstance = window.Get("sessionStorage")
			isIndexedDB = false
		case "localstorage":
			storeInstance = window.Get("localStorage")
			isIndexedDB = false
		case "indexeddb":
			storeInstance = window.Get("indexedDBSyncInstance")
			isIndexedDB = true
			if !storeInstance.Truthy() {
				DEBUG("window.indexedDBSyncInstance is not available, cannot load from IndexedDB")
				return errors.New("IndexedDB instance not available")
			}
		default:
			return errors.New("Unsupported storage type: " + s)
		}

		store := jsStore{store: storeInstance, isIndexedDB: isIndexedDB}
		_, ok := store.Get("zui-connected")
		if !ok {
			return errors.New("storage is disconnected")
		}
		id := e.ID

		// Let's retrieve the category index for this element, if it exists in the sessionstore

		categories := []string{Namespace.Data, Namespace.UI}
		uiloaders := make([]func(), 0, 64)

		for _, category := range categories {
			jsonprops, ok := store.Get(strings.Join([]string{id, category}, "/"))
			if !ok {
				continue
			}

			properties := make([]string, 0, 64)
			err := json.Unmarshal([]byte(jsonprops.String()), &properties)
			if err != nil {
				return err
			}

			for _, property := range properties {
				// let's retrieve the propname (it is suffixed by the proptype)
				// then we can retrieve the value
				// log.Print("debug...", category, property) // DEBUG

				propname := property
				jsonvalue, ok := store.Get(strings.Join([]string{e.ID, category, propname}, "/"))
				if ok {
					var rawvaluemapstring string
					err = json.Unmarshal([]byte(jsonvalue.String()), &rawvaluemapstring)
					if err != nil {
						return err
					}

					rawvalue := make(map[string]interface{})
					err = json.Unmarshal([]byte(rawvaluemapstring), &rawvalue)
					if err != nil {
						return err
					}
					val := ui.ValueFrom(rawvalue)

					if category == Namespace.Data {
						ui.LoadProperty(e, category, propname, val)
					}

					if category == Namespace.UI {
						uiloaders = append(uiloaders, func() { // TODO remove since unused
							e.SetUI(propname, val)
						})
					}
					//log.Print("LOADED PROPMAP: ", e.Properties, category, propname, rawvalue.Value()) // DEBUG
				}
			}

			//log.Print(categories, properties) //DEBUG
			//lch := ui.NewLifecycleHandlers(e.Root)
			e.OnMounted(ui.OnMutation(func(evt ui.MutationEvent) bool {
				for _, load := range uiloaders {
					load()
				}
				ui.Rerender(evt.Origin())
				return false
			}).RunOnce().RunASAP())

		}

		return nil
	}
}

var loadfromsession = loader("sessionstorage")
var loadfromlocalstorage = loader("localstorage")
var loadfromindexeddb = loader("indexeddb")

func clearer(s string) func(element *ui.Element) {
	return func(element *ui.Element) {
		window := js.Global()
		if !window.Truthy() {
			DEBUG("window is not available, cannot clear storage")
			return
		}
		var storeInstance js.Value
		var isIndexedDB bool

		switch s {
		case "sessionstorage":
			storeInstance = window.Get("sessionStorage")
			isIndexedDB = false
		case "localstorage":
			storeInstance = window.Get("localStorage")
			isIndexedDB = false
		case "indexeddb":
			storeInstance = window.Get("indexedDBSyncInstance")
			isIndexedDB = true
			if !storeInstance.Truthy() {
				DEBUG("window.indexedDBSyncInstance is not available, cannot clear IndexedDB")
				return
			}
		default:
			DEBUG("Unsupported storage type:", s)
			return
		}

		store := jsStore{store: storeInstance, isIndexedDB: isIndexedDB}

		_, ok := store.Get("zui-connected")
		if !ok {
			return
		}
		id := element.ID
		categories := []string{Namespace.Data, Namespace.UI}

		for _, category := range categories {
			// Let's retrieve the category index for this element, if it exists in the sessionstore
			jsonproperties, ok := store.Get(strings.Join([]string{id, category}, "/"))
			if !ok {
				return
			}

			properties := make([]string, 0, 64)

			err := json.Unmarshal([]byte(jsonproperties.String()), &properties)
			if err != nil {
				store.Delete(id)
				panic("An error occured when removing an element from storage. It's advised to reinitialize " + s)
			}

			for _, property := range properties {
				// let's retrieve the propname (it is suffixed by the proptype)
				// then we can retrieve the value
				// log.Print("debug...", category, property) // DEBUG

				store.Delete(strings.Join([]string{id, category, property}, "/"))
			}

			store.Delete(strings.Join([]string{id, category}, "/"))
		}
	}
}

var clearfromsession = clearer("sessionstorage")
var clearfromlocalstorage = clearer("localstorage")
var clearfromindexeddb = clearer("indexeddb")

var cleanStorageOnDelete = ui.NewConstructorOption("cleanstorageondelete", func(e *ui.Element) *ui.Element {
	e.OnDeleted(ui.OnMutation(func(evt ui.MutationEvent) bool {
		ClearFromStorage(evt.Origin())
		if e.Native != nil {
			j, ok := JSValue(e)
			if !ok {
				return false
			}
			if j.Truthy() {
				j.Call("remove")
			}
		}

		return false
	}))
	return e
})

// isPersisted checks whether an element exist in storage already
func isPersisted(e *ui.Element) bool {
	pmode := ui.PersistenceMode(e)

	var isIndexedDB bool

	var s string
	switch pmode {
	case "sessionstorage":
		s = "sessionStorage"
	case "localstorage":
		s = "localStorage"
	case "indexeddb":
		s = "indexedDBSyncInstance"
		isIndexedDB = true
	default:
		return false
	}

	store := jsStore{store: js.Global().Get(s), isIndexedDB: isIndexedDB}
	_, ok := store.Get(e.ID)
	return ok
}

// LoadFromStorage will load an element properties.
// If the corresponding native DOM Element is marked for hydration, by the presence of a data-hydrate
// atribute, the props are loaded from this attribute instead.
func LoadFromStorage(a ui.AnyElement) *ui.Element {
	e := a.AsElement()
	if e == nil {
		panic("loading a nil element")
	}

	pmode := ui.PersistenceMode(e)

	storage, ok := e.Configuration.PersistentStorer[pmode]

	if ok {
		err := storage.Load(e)
		if err != nil {
			panic(err)
		}
	}
	return e
}

// PutInStorage stores an element data in storage (localstorage or sessionstorage).
func PutInStorage(a ui.AnyElement) *ui.Element {
	e := a.AsElement()
	if LRMode != "false" {
		if e.ID != "mutation-recorder" {
			return e
		}
	}

	pmode := ui.PersistenceMode(e)
	storage, ok := e.Configuration.PersistentStorer[pmode]
	if !ok {
		return e
	}

	for cat, props := range e.Properties.Categories {
		if cat != Namespace.Data && cat != Namespace.UI {
			continue
		}
		for prop, val := range props.Local {
			storage.Store(e, cat, prop, val)
		}
	}
	return e
}

// ClearFromStorage will clear an element properties from storage.
func ClearFromStorage(a ui.AnyElement) *ui.Element {
	e := a.AsElement()
	pmode := ui.PersistenceMode(e)

	storage, ok := e.Configuration.PersistentStorer[pmode]
	if ok {
		storage.Clear(e)
	}
	return e
}

// AllowSessionStoragePersistence is a constructor option.
// A constructor option allows us to add custom optional behaviors to Element constructors.
// If made available to a constructor function, the coder may decide to enable
//
//	session storage of the properties of an Element  created with said constructor.
var AllowSessionStoragePersistence = ui.NewConstructorOption("sessionstorage", func(e *ui.Element) *ui.Element {
	e.Set("internals", "persistence", ui.String("sessionstorage"))
	return e
})

var AllowAppLocalStoragePersistence = ui.NewConstructorOption("localstorage", func(e *ui.Element) *ui.Element {
	e.Set("internals", "persistence", ui.String("localstorage"))
	return e
})

// AllowIndexedDBPersistence is a constructor option for IndexedDB storage.
var AllowIndexedDBPersistence = ui.NewConstructorOption("indexeddb", func(e *ui.Element) *ui.Element {
	e.Set("internals", "persistence", ui.String("indexeddb"))
	return e
})

func EnableSessionPersistence() string {
	return "sessionstorage"
}

func EnableLocalPersistence() string {
	return "localstorage"
}

func EnableIndexedDBPersistence() string {
	return "indexeddb"
}

func SyncOnDataMutation(e *ui.Element, propname string) *ui.Element {
	e.Watch(Namespace.Data, propname, e, ui.OnMutation(func(evt ui.MutationEvent) bool {
		PutInStorage(e)
		return false
	}))
	return e
}
