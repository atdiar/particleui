// Package ui is a library of functions for simple, generic gui development.
package ui

import (
	"errors"
	"log"
)

var (
	ErrNoTemplate = errors.New("Element template missing")
)

type ElementStore struct {
	DocType   string
	Templates map[string]Element
	ByID      map[string]*Element
}

func (e ElementStore) ElementFromTemplate(name string) *Element {
	t, ok := e.Templates[name]
	if ok {
		return &t
	}
	return nil
}

func (e ElementStore) NewTemplate(t *Element) {
	e.Templates[t.Name] = *t
}

func Constructor(es ElementStore) func(string, string) (*Element, error) {
	New := func(name string, id string) (*Element, error) {
		if e := es.ElementFromTemplate(name); e != nil {
			e.Name = name
			e.ID = id
			e.DocType = es.DocType
			// TODO copy any map field
			return e, nil
		}
		return nil, ErrNoTemplate
	}
	return New
}

// Element is the building block of the User Interface. Everything is described
// as an Element having some mutable properties (graphic properties or data properties)
// From the window to the buttons on a page.
type Element struct {
	root        *Element
	subtreeRoot *Element // detached if subtree root has no parent unless subtreeroot == root
	path        *Elements

	Parent *Element

	Name    string
	ID      string
	DocType string

	UIProperties PropertyStore
	Data         DataStore

	OnMutation map[string]MutationHandlers // list of mutation handlers stored at elementID/propertyName (Elements react to change in other elements they are monitoring)
	OnEvent    EventListeners              // EventHandlers are to be called when the named event has fired.

	// Proper event handling requires to assert the interface to have access to the underlying object so that target id may be retrieved
	// amongst other event properties. the handling should be reflected in the actual dom via modification of the underlying js object.

	Children *Elements

	Native interface{}

	inherit bool
}

type PropertyStore struct {
	GlobalShared map[string]interface{}

	Default map[string]interface{}

	Inherited map[string]interface{} //Inherited property cannot be mutated by the inheritor

	Local map[string]interface{}

	Inheritable map[string]interface{} // the value of a property overrides ithe value stored in any of its predecessor value store
	// map key is the address of the element's  property
	// being watched and elements is the list of elements watching this property
	// Inheritable encompasses overidden values and inherited values that are being passed down.
	Watchers map[string]*Elements
}

func (p PropertyStore) NewWatcher(propName string, watcher *Element) {
	list, ok := p.Watchers[propName]
	if !ok {
		p.Watchers[propName] = NewElements(watcher)
		return
	}
	list.Insert(watcher, len(list.List))
}
func (p PropertyStore) RemoveWatcher(propName string, watcher *Element) {}

func (p PropertyStore) Get(propName string) (interface{}, bool) {}
func (p PropertyStore) Set(propName string, value interface{})  {} // don't forget to propagate mutation event to watchers

type DataStore struct {
	Store     map[string]interface{}
	Immutable map[string]interface{}

	// map key is the address of the data being watched (e.g. id/dataname)
	// being watched and elements is the list of elements watching this property
	Watchers map[string]*Elements
}

func (d DataStore) NewWatcher(label string, watcher *Element) {
	list, ok := d.Watchers[label]
	if !ok {
		d.Watchers[label] = NewElements(watcher)
		return
	}
	list.Insert(watcher, len(list.List))
}
func (d DataStore) RemoveWatcher(label string, watcher *Element) {}

func (d DataStore) Get(label string) (interface{}, bool) {}
func (d DataStore) Set(label string, value interface{})  {} // do not forget to notify watcher Elements of change

func NewDataStore() DataStore {
	return DataStore{make(map[string]interface{}), make(map[string]interface{}), make(map[string]*Elements)}
}

type Elements struct {
	List []*Element
}

func NewElements(elements ...*Element) *Elements {
	return &Elements{elements}
}

func (e *Elements) Insert(el *Element, index int) *Elements {
	nel := make([]*Element, 0)
	nel = append(nel, e.List[:index]...)
	nel = append(nel, el)
	nel = append(nel, e.List[index:]...)
	e.List = nel
	return e
}

func (e *Element) Handle(evt Event) bool {
	evt.SetCurrentTarget(e)
	return e.OnEvent.Handle(evt)
}

//
func (e *Element) DispatchEvent(evt Event) *Element {

	if e.Detached() {
		log.Print("Error: Element detached. should not happen.")
		return e // should not really happen
	}
	if e.path == nil {
		log.Print("Error: Element path does not exist.") // should not happen if the libaray is correctly implemented
		return e
	}

	// First we apply the capturing event handlers PHASE 1
	evt.SetPhase(1)
	var done bool
	for _, ancestor := range e.path.List {
		if evt.Stopped() {
			return e
		}

		done = ancestor.Handle(evt) // Handling deemed finished in user side logic
		if done || evt.Stopped() {
			return e
		}
	}

	// Second phase: we handle the events at target
	evt.SetPhase(2)
	done = e.Handle(evt)
	if done {
		return e
	}

	// Third phase : bubbling
	if !evt.Bubbles() {
		return e
	}
	evt.SetPhase(3)
	for k := len(e.path.List) - 1; k >= 0; k-- {
		ancestor := e.path.List[k]
		if evt.Stopped() {
			return e
		}
		done = ancestor.Handle(evt)
		if done {
			return e
		}
	}
	return e
}

func (e *Element) Parse(payload string) *Element      { return e }
func (e *Element) Unparse(outputformat string) string {}

func (e *Element) AddInnerElementselements(...*Element) *Element { return e }

func (e *Element) Watch(datalabel string, target *Element, handler func(MutationEvent)) *Element {
	return e
}
func (e *Element) Unwatch(datalabel string, target *Element) *Element { return e }

func (e *Element) AddEventListener(event string, handler *EventHandler) *Element    { return e }
func (e *Element) RemoveEventListener(event string, handler *EventHandler) *Element { return e }

func (e *Element) Detached() bool {
	if e.subtreeRoot.Parent == nil && e.subtreeRoot != e.root {
		return true
	}
	return false
}
