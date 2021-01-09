// Package ui is a library of functions for simple, generic gui development.
package ui

import (
	"errors"
	"log"
	"math/rand"
	"strings"
)

var (
	ErrNoTemplate = errors.New("Element template missing")
)

// NewIDgenerator returns a function used to create new IDs for Elements. It uses
// a Pseudo-Random Number Generator (PRNG) as it is disirable to have as deterministic
// IDs as possible. Notably for the mostly tstaic elements.
// Evidently, as users navigate the app differently and mya create new Elements
// in a different order (hence calling the ID generator is path-dependent), we
// do not expect to have the same id structure for different runs of a same program.
func NewIDgenerator(seed int64) func() string {
	return func() string {
		bstr := make([]byte, 32)
		rand.Seed(seed)
		_, _ = rand.Read(bstr)
		return string(bstr)
	}
}

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

	OnMutation             MutationCallbacks      // list of mutation handlers stored at elementID/propertyName (Elements react to change in other elements they are monitoring)
	OnEvent                EventListeners         // EventHandlers are to be called when the named event has fired.
	NativeEventUnlisteners NativeEventUnlisteners // Allows to remove event listeners on the native element, registered when bridging event listeners from the native UI platform.

	Children   *Elements
	ActiveView string // Holds the name of the current set of children Element
	//ViewAccessPath records the Elements which have defined alternative views and the value for which the view makes the current element reachable for rendering.
	ViewAccessPath *viewNodes // it's different from the path because some subtree may all belong to the same view, if there is no element with multiple available views inbetween

	AlternateViews map[string]*Elements // this is a  store for  named views: alternative to the Children field, used for instance to implement routes/ conditional rendering.

	Native NativeElementWrapper
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
func (p PropertyStore) RemoveWatcher(propName string, watcher *Element) {
	list, ok := p.Watchers[propName]
	if !ok {
		return
	}
	list.Remove(watcher)
}

func (p PropertyStore) Get(propName string) (interface{}, bool) {
	v, ok := p.Inheritable[propName]
	if ok {
		return v, ok
	}
	v, ok = p.Local[propName]
	if ok {
		return v, ok
	}
	v, ok = p.Inherited[propName]
	if ok {
		return v, ok
	}
	v, ok = p.Default[propName]
	if ok {
		return v, ok
	}
	v, ok = p.GlobalShared[propName]
	if ok {
		return v, ok
	}
	return nil, false
}
func (p PropertyStore) Set(propName string, value interface{}, inheritable bool) {
	if inheritable {
		p.Inheritable[propName] = value
		return
	}
	p.Local[propName] = value
} // don't forget to propagate mutation event to watchers

func (p PropertyStore) Inherit(source PropertyStore) {
	if source.Inheritable != nil {
		for k, v := range source.Inheritable {
			p.Inherited[k] = v
		}
	}
}

func (p PropertyStore) SetDefault(propName string, value interface{}) {
	p.Default[propName] = value
}

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
func (d DataStore) RemoveWatcher(label string, watcher *Element) {
	v, ok := d.Watchers[label]
	if !ok {
		return
	}
	v.Remove(watcher)
}

func (d DataStore) Get(label string) (interface{}, bool) {
	if v, ok := d.Immutable[label]; ok {
		return v, ok
	}
	v, ok := d.Store[label]
	return v, ok
}
func (d DataStore) Set(label string, value interface{}) {
	if _, ok := d.Immutable[label]; ok {
		return
	}
	d.Store[label] = value
}

func NewDataStore() DataStore {
	return DataStore{make(map[string]interface{}), make(map[string]interface{}), make(map[string]*Elements)}
}

type Elements struct {
	List []*Element
}

func NewElements(elements ...*Element) *Elements {
	return &Elements{elements}
}

func (e *Elements) InsertLast(elements ...*Element) *Elements {
	e.List = append(e.List, elements...)
	return e
}

func (e *Elements) InsertFirst(elements ...*Element) *Elements {
	e.List = append(elements, e.List...)
	return e
}

func (e *Elements) Insert(el *Element, index int) *Elements {
	nel := make([]*Element, 0)
	nel = append(nel, e.List[:index]...)
	nel = append(nel, el)
	nel = append(nel, e.List[index:]...)
	e.List = nel
	return e
}

func (e *Elements) Remove(el *Element) *Elements {
	index := -1
	for k, element := range e.List {
		if element == el {
			index = k
			break
		}
	}
	if index >= 0 {
		e.List = append(e.List[:index], e.List[index+1:]...)
	}
	return e
}

func (e *Elements) RemoveAll() *Elements {
	e.List = nil
	return e
}

func (e *Elements) Replace(old *Element, new *Element) *Elements {
	for k, element := range e.List {
		if element == old {
			e.List[k] = new
			return e
		}
	}
	return e
}

// Handle calls up the event handlers in charge of processing the event for which
// the Element is listening.
func (e *Element) Handle(evt Event) bool {
	evt.SetCurrentTarget(e)
	return e.OnEvent.Handle(evt)
}

// DispatchEvent is used typically to propagate UI events throughout the ui tree.
// It may require an event object to be created from the native event object implementation.
// Events are propagated following the model set by web browser DOM events:
// 3 phases being the capture phase, at-target and then bubbling up if allowed.
func (e *Element) DispatchEvent(evt Event) *Element {

	if e.Detached() {
		log.Print("Error: Element detached. should not happen.")
		// TODO review which type of event could walk up a detached subtree
		// for instance, how to update darkmode on detached elements especially
		// on attachment. (life cycles? +  globally propagated values from root + mutations propagated in spite of detachment status)
		return e // can happen if we are building a document fragment and try to dispatch a custom event
	}
	if e.path == nil {
		log.Print("Error: Element path does not exist (yet).")
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

// func (e *Element) Parse(payload string) *Element      { return e }
// func (e *Element) Unparse(outputformat string) string {}

// AddView enables conditional rendering for an Element, specifying multiple
// named version for the list of direct children Elements.
func (e *Element) AddView(v View) *Element {
	for _, child := range v.Elements.List {
		child.ViewAccessPath = child.ViewAccessPath.Prepend(newViewNode(e, v.Name)).Prepend(e.ViewAccessPath.nodes...)
		attach(e, child, false)
	}

	e.AlternateViews[v.Name] = v.Elements
	return e
}

// ActivateVIew is used to render the desired named view for a given Element.
func (e *Element) ActivateView(name string) *Element {
	newview, ok := e.AlternateViews[name]
	if !ok {
		// Support for parameterized views TODO
		if len(e.AlternateViews) !=0 {
			var viewElements *Elements
			var parameterName string
			for k, v := range e.AlternateViews {
				if strings.HasPrefix(k, ":") {
					parameterName = k
					viewElements = v
					break
				}
			}
			if parameterName != "" {
				if len(parameterName) == 1{
					log.Print("Bad view parameter. Needs to be longer than 0 character.")
					return e
				}
				parameter := parameterName[1:]
				// let's set the parameter value (name) on the children elements in case their
				// display depends on it which is probably the case. // TODO think about how the user should make sure that the child Elements API includes this parameter
				for _, v := range viewElements.List {
					v.Set(parameter, name)
				}
				e.ActiveView = name
				// Let's detach the former view items and attach the ViewElements
				for _, child := range e.Children.List {
					detach(child)
				}

				for _, newchild := range viewElements.List {
					e.AppendChild(newchild)
				}
				return e
			}
		}
		log.Print("View does not exist.")
		return e
	}

	// first we detach the current active View and reattach it as an alternative View
	oldviewname := e.ActiveView
	if oldviewname != "" || e.Children != nil {
		for _, child := range e.Children.List {
			detach(child)
			attach(e, child, false)
		}
		e.AlternateViews[oldviewname] = e.Children
	}
	// we attach the desired view
	for _, child := range newview.List {
		e.AppendChild(child)
	}
	delete(e.AlternateViews, name)
	e.ActiveView = name

	return e
}

// attach will link a child Element to the subtree its target parent belongs to.
// It does not however position it in any view specifically. At this staged,
// the Element can not be rendered as part of the view.
func attach(parent, child *Element, activeview bool) {
	if activeview {
		child.Parent = parent
		child.path.InsertFirst(parent).InsertFirst(parent.path.List...)
	}
	child.root = parent.root // attached once means attached for ever unless attached to a new app *root (imagining several apps can be ran concurrently and can share ui elements)
	child.subtreeRoot = parent.subtreeRoot

	// if the child is not a navigable view(meaning that its alternateViews is nil, then it's viewadress is its parent's)
	// otherwise, it's its own to which is prepended its parent's viewAddress.
	if child.AlternateViews == nil {
		child.ViewAccessPath = parent.ViewAccessPath
	} else {
		child.ViewAccessPath = child.ViewAccessPath.Prepend(parent.ViewAccessPath.nodes...)
	}

	for _, descendant := range child.Children.List {
		attach(child, descendant, true)
	}

	for _, descendants := range child.AlternateViews {
		for _, descendant := range descendants.List {
			attach(child, descendant, false)
		}
	}
}

// detach will unlink an Element from its parent. If the element was in a view,
// the element is still being rendered until it is removed. However, it should
// not be anle to react to events or mutations. TODO review the latter part.
func detach(e *Element) {
	if e.Parent == nil {
		return
	}
	e.subtreeRoot = e

	// reset e.path to start with the top-most element i.e. "e" in the current case
	index := -1
	for k, ancestor := range e.path.List {
		if ancestor == e.Parent {
			index = k
			break
		}
	}
	if index >= 0 {
		e.path.List = e.path.List[index+1:]
	}

	e.Parent = nil

	// ViewAccessPath handling:
	if e.AlternateViews == nil {
		e.ViewAccessPath = nil
	} else {
		e.ViewAccessPath.nodes = e.ViewAccessPath.nodes[len(e.ViewAccessPath.nodes)-1:]
	}

	// got to update the subtree with the new subtree root and path
	for _, descendant := range e.Children.List {
		attach(e, descendant, true)
	}

	for _, descendants := range e.AlternateViews {
		for _, descendant := range descendants.List {
			attach(e, descendant, false)
		}
	}
}

// AppendChild appends a new element to the element's children list for the active
// view being rendered.
func (e *Element) AppendChild(child *Element) *Element {
	if e.DocType != child.DocType {
		log.Printf("Doctypes do not macth. Parent has %s while child Element has %s", e.DocType, child.DocType)
		return e
	}
	attach(e, child, true)

	e.Children.InsertLast(child)
	if e.Native != nil {
		e.Native.AppendChild(child)
	}
	return e
}

func (e *Element) Prepend(child *Element) *Element {
	if e.DocType != child.DocType {
		log.Printf("Doctypes do not macth. Parent has %s while child Element has %s", e.DocType, child.DocType)
		return e
	}
	attach(e, child, true)

	e.Children.InsertFirst(child)
	if e.Native != nil {
		e.Native.PrependChild(child)
	}
	return e
}
func (e *Element) InsertChild(child *Element, index int) *Element {
	if e.DocType != child.DocType {
		log.Printf("Doctypes do not macth. Parent has %s while child Element has %s", e.DocType, child.DocType)
		return e
	}
	attach(e, child, true)

	e.Children.Insert(child, index)
	if e.Native != nil {
		e.Native.InsertChild(child, index)
	}
	return e
}
func (e *Element) ReplaceChild(old *Element, new *Element) *Element {
	if e.DocType != new.DocType {
		log.Printf("Doctypes do not macth. Parent has %s while child Element has %s", e.DocType, new.DocType)
		return e
	}
	attach(e, new, true)

	detach(old)

	e.Children.Replace(old, new)
	if e.Native != nil {
		e.Native.ReplaceChild(old, new)
	}
	return e
}
func (e *Element) RemoveChild(child *Element) *Element {
	detach(child)

	if e.Native != nil {
		e.Native.RemoveChild(child)
	}
	return e
}

func (e *Element) RemoveChildren() *Element {
	for _, child := range e.Children.List {
		e.RemoveChild(child)
	}
	return e
}

func (e *Element) Watch(datalabel string, mutationSource *Element, h *MutationHandler) *Element {
	mutationSource.Data.NewWatcher(datalabel, e)
	e.OnMutation.Add(mutationSource.ID+"/"+datalabel, h)
	return e
}
func (e *Element) Unwatch(datalabel string, mutationSource *Element) *Element {
	mutationSource.Data.RemoveWatcher(datalabel, e)
	return e
}

func (e *Element) AddEventListener(event string, handler *EventHandler, nativebinding NativeEventBridge) *Element {
	e.OnEvent.AddEventHandler(event, handler)
	if nativebinding != nil {
		nativebinding(event, e)
	}
	return e
}
func (e *Element) RemoveEventListener(event string, handler *EventHandler, native bool) *Element {
	e.OnEvent.RemoveEventHandler(event, handler)
	if native {
		if e.NativeEventUnlisteners.List != nil {
			e.NativeEventUnlisteners.Apply(event)
		}
	}
	return e
}

// Detached returns whether the subtree the current Element belongs to is attached
// to the main tree or not.
func (e *Element) Detached() bool {
	if e.subtreeRoot.Parent == nil && e.subtreeRoot != e.root {
		return true
	}
	return false
}

func (e *Element) Get(label string) (interface{}, bool) {
	return e.Data.Get(label)
}
func (e *Element) Set(label string, value interface{}) {
	e.Data.Set(label, value)
	evt := e.NewMutationEvent(label, value).Data()
	e.OnMutation.DispatchEvent(evt)
}

func (e *Element) GetUI(propName string) (interface{}, bool) {
	return e.UIProperties.Get(propName)
}

func (e *Element) SetUI(propName string, value interface{}, inheritable bool) {
	e.UIProperties.Set(propName, value, inheritable)
	evt := e.NewMutationEvent(propName, value).UI()
	e.OnMutation.DispatchEvent(evt)
}

// MakeToggable is a simple example of conditional rendering.
// It allows an Element to have a single child Element that can be switched with another one.
//
// For example, if we have a toggable button, one for login and one for logout,
// We can implement the switch between login and logout by switching the inner Elements.
//
// This is merely an example as we could implement toggling between more than two
// Elements quite easily.
// Routing will probably be implemented this way, toggling between states
// when a mutationevent such as browser history occurs.
func MakeToggable(conditionName string, e *Element, firstView View, secondView View, initialconditionvalue interface{}) *Element {
	e.AddView(firstView).AddView(secondView)

	toggle := NewMutationHandler(func(evt MutationEvent) {
		value, ok := evt.NewValue().(bool)
		if !ok {
			value = false
		}
		if value {
			e.ActivateView(firstView.Name)
		}
		e.ActivateView(secondView.Name)
	})

	e.Watch(conditionName, e, toggle)

	e.Set(conditionName, initialconditionvalue)
	return e
}

type View struct {
	Name     string
	Elements *Elements
}

// NewView can be used to create a list of children Elements to append to an element, for display.
// In effect, a named view.
// An example of use would be an empty window that would be filled with different child elements
// upon navigation.
// A parameterized view can be created by using a naming scheme such as ":parameter" (string with a leading colon)
// In that case,
func NewView(name string, elements ...*Element) View {
	for _, el := range elements {
		el.ActiveView = name
	}
	return View{name, NewElements(elements...)}
}

type viewNodes struct {
	nodes []viewNode
}

func newViewNodes() *viewNodes {
	return &viewNodes{make([]viewNode, 0)}
}

func (v *viewNodes) Append(nodes ...viewNode) *viewNodes {
	v.nodes = append(v.nodes, nodes...)
	return v
}

func (v *viewNodes) Prepend(nodes ...viewNode) *viewNodes {
	v.nodes = append(nodes, v.nodes...)
	return v
}

type viewNode struct {
	*Element
	ViewName string
}

func newViewNode(e *Element, view string) viewNode {
	return viewNode{e, view}
}
