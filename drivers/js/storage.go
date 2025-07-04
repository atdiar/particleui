// Package doc defines the default set of Element constructors, native interfaces,
// events and event handlers, and animation properties used to build js-based UIs.

package doc

import (
	"encoding/json"
	"errors"
	"strings"

	ui "github.com/atdiar/particleui"
	js "github.com/atdiar/particleui/drivers/js/compat"
)

// TODO implement IndexedDB storage backaend

type jsStore struct {
	store js.Value
}

func (s jsStore) Get(key string) (js.Value, bool) {
	v := s.store.Call("getItem", key)
	if !v.Truthy() {
		return v, false
	}
	return v, true
}

func (s jsStore) Set(key string, value js.Value) {
	JSON := js.Global().Get("JSON")
	res := JSON.Call("stringify", value)
	s.store.Call("setItem", key, res)
}

func (s jsStore) Delete(key string) {
	s.store.Call("removeItem", key)
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
		store := jsStore{window.Get(s)}
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

var sessionstorefn = storer("sessionStorage")
var localstoragefn = storer("localStorage")

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
		store := jsStore{window.Get(s)}
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

var loadfromsession = loader("sessionStorage")
var loadfromlocalstorage = loader("localStorage")

func clearer(s string) func(element *ui.Element) {
	return func(element *ui.Element) {
		window := js.Global()
		if !window.Truthy() {
			DEBUG("window is not available, cannot clear storage")
			return
		}
		store := jsStore{window.Get(s)}
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

var clearfromsession = clearer("sessionStorage")
var clearfromlocalstorage = clearer("localStorage")

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

	var s string
	switch pmode {
	case "sessionstorage":
		s = "sessionStorage"
	case "localstorage":
		s = "localStorage"
	default:
		return false
	}

	store := jsStore{js.Global().Get(s)}
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

func EnableSessionPersistence() string {
	return "sessionstorage"
}

func EnableLocalPersistence() string {
	return "localstorage"
}

func SyncOnDataMutation(e *ui.Element, propname string) *ui.Element {
	e.Watch(Namespace.Data, propname, e, ui.OnMutation(func(evt ui.MutationEvent) bool {
		PutInStorage(e)
		return false
	}))
	return e
}
