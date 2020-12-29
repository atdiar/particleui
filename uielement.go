// Package uitree is a library of functions for simple, generic gui development.
package uitree

import "sync"

type ElementStore struct {
	ByID map[string]*Element
}

func (e ElementStore) Put(el *Element) {
	_, ok := e.ByID[el.ID]
	if !ok {
		e.ByID[el.ID] = el
	}
}

func (e ElementStore) GetByID(id string) *Element {
	v, ok := e.ByID[id]
	if !ok {
		return nil
	}
	return v
}

func (e ElementStore) GetByType(typ string) Elements {
	n := NewElements()
	for _, v := range e.ByID {
		if v.Type == typ {
			n.Insert(v, len(n.List))
		}
	}
	return *n
}

func (e ElementStore) GetByName(name string) Elements {
	n := NewElements()
	for _, v := range e.ByID {
		if v.Name == name {
			n.Insert(v, len(n.List))
		}
	}
	return *n
}

func Constructor() (func(string, string, string) *Element, map[string]Elements, map[string]*Element) {
	ElementStoredByType := make(map[string]Element)
	ElementStoredByID := make(map[string]*Element)
	New := func(namespace string, eltype string, id string) *Element {
		e, ok := ElementStoredByType[eltype]
		if ok {
			new := Element{}
			new.Name = namespace
			new.mu = &sync.Mutex{}
			new.Type = eltype
			new.ID = id
			new.UIProperties.Shared = e.UIProperties.Shared
			new.UIProperties.Local = make(map[string]string)
			for k, v := range e.UIProperties.Local {
				new.UIProperties.Local[k] = v
			}
			new.UIProperties.Watchers = make(map[string]*Elements) // mapping string properties with a list of observing Element types
			new.mutationHandlers = e.mutationHandlers
			new.watchedUIMutattions = make(map[string]string)
			new.watchedDataMutations = make(map[string]interface{})

		}
	}
	return New, ElementStoredByType, ElementStoredByID
}

// Element is the building block of the User Interface. Everything is described
// as an Element having some mutable properties (graphic properties or data properties)
// From the window to the buttons on a page.
type Element struct {
	Parent *Element

	Name string
	Type string
	ID   string

	UIProperties map[string]Properties // properties are namespaced
	Data         Data

	// watched mutations are recipient for the mutated values
	// Each time the method to add to these are called, we should try and  find
	// a mutation Handler if any has been registered.
	// TODO Use heap so that the history of mutations is conserved map[string][]string
	watchedUIMutattions  map[string]string      // the key is the property name and the value is the new value
	watchedDataMutations map[string]interface{} // the key is the data name and the value its new value
	mutationHandlers     map[string]MutationHandlers

	Children *Elements

	mu *sync.Mutex
}

type Properties struct {
	Shared map[string]string
	Local  map[string]string // A property in local will override a  global property

	// map key is the address of the element's  property
	// being watched and elements is the list of elements watching this property
	Watchers map[string]*Elements
}

type Data struct {
	Store map[string]interface{}

	// map key is the address of the data being watched
	// being watched and elements is the list of elements watching this property
	Watchers map[string]Elements
}

func NewDataStore() Data {
	return Data{make(map[string]interface{})}
}

type Elements struct {
	List []*Element
}

func NewElements(elements ...*Element) *Elements {
	return &Elements{elements}
}
func (e *Elements) Insert(el *Element, index int) *Elements {
	nel := make([]*Elements, 0)
	nel = append(nel, e.List[:index]...)
	nel = append(nel, el)
	nel = append(nel, e.List[index:]...)
	e.List = nel
	return e
}

type MutationHandlers struct {
	Handlers []func(*Element) interface{}
}

func (e *Element) Parse(format string, payload string) *Element {}
func (e *Element) Inner(elements ...*Element) *Element          {}
