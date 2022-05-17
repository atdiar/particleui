// Package ui is a library of functions for simple, generic gui development.
package ui

import (
	"errors"
	"log"
	"math/rand"
	"strconv"
	"strings"
)

var (
	ErrNoTemplate = errors.New("Element template missing")
	DEBUG         = log.Print // DEBUG
)

// NewIDgenerator returns a function used to create new IDs for Elements. It uses
// a Pseudo-Random Number Generator (PRNG) as it is disirable to have as deterministic
// IDs as possible. Notably for the mostly tstaic elements.
// Evidently, as users navigate the app differently and mya create new Elements
// in a different order (hence calling the ID generator is path-dependent), we
// do not expect to have the same id structure for different runs of a same program.
func NewIDgenerator(seed int64) func() string {
	return func() string {
		rand.Seed(seed)
		str := strconv.Itoa(rand.Int())
		return str
	}
}

// Stores holds a list of ElementStore. Every newly created ElementStore should be
// listed here by defauklt.
var Stores = newElementStores()

type elementStores struct {
	stores map[string]*ElementStore
}

func newElementStores() elementStores {
	v := make(map[string]*ElementStore)
	return elementStores{v}
}

func (e elementStores) Get(storeid string) (*ElementStore, bool) {
	res, ok := e.stores[storeid]
	return res, ok
}

func (e elementStores) Set(store *ElementStore) {
	_, ok := e.stores[store.Global.ID]
	if ok {
		log.Print("ElementStore already exists")
		return
	}
	e.stores[store.Global.ID] = store
}

// ElementStore defines a namespace for a list of Element constructors.
type ElementStore struct {
	DocType                  string
	Constructors             map[string]func(name, id string, optionNames ...string) *Element
	GlobalConstructorOptions map[string]func(*Element) *Element
	ConstructorsOptions      map[string]map[string]func(*Element) *Element
	ByID                     map[string]*Element

	PersistentStorer map[string]storageFunctions

	Global *Element // the global Element stores the global state shared by all *Elements
}

type storageFunctions struct {
	Load  func(*Element) error
	Store func(e *Element, category string, propname string, value Value, flags ...bool)
}

// ConstructorOption defines a type for optional function that can be called on
// Element construction. It allows to specify optional Element construction behaviours.
// Useful if we want to be able to return different types of buttons from a button
// Element constructor for example.
type ConstructorOption struct {
	Name         string
	Configurator func(*Element) *Element
}

func NewConstructorOption(name string, configuratorFn func(*Element) *Element) ConstructorOption {
	fn := func(e *Element) *Element {
		a, ok := e.Get("internals", "constructoroptions")
		if !ok {
			a = NewList(String(name))
			e.Set("internals", "constructoroptions", a)
		}
		l, ok := a.(List)
		if !ok {
			log.Print("Unexpected error. constructoroptions should be stored as a ui.List")
			a := NewList(String(name))
			e.Set("internals", "constructoroptions", a)
		}
		for _, copt := range l {
			if copt == String(name) {
				return configuratorFn(e)
			}
		}
		e.Set("internals", "constructoroptions", append(l, String(name)))

		return configuratorFn(e)
	}
	return ConstructorOption{name, fn}
}

// NewElementStore creates a new namespace for a list of Element constructors.
func NewElementStore(storeid string, doctype string) *ElementStore {
	global := NewElement("global", storeid, doctype)
	es := &ElementStore{doctype, make(map[string]func(name string, id string, optionNames ...string) *Element, 0), make(map[string]func(*Element) *Element), make(map[string]map[string]func(*Element) *Element, 0), make(map[string]*Element), make(map[string]storageFunctions, 5), global}
	Stores.Set(es)
	return es
}

// ApplyGlobalOption registers a Constructor option that will be called for every
// element constructed.
// Rationale: implementing dark-mode aware ui elements easily.
func (e *ElementStore) ApplyGlobalOption(c ConstructorOption) *ElementStore {
	e.GlobalConstructorOptions[c.Name] = c.Configurator
	return e
}

// AddPersistenceMode allows to define alternate ways to persist Element properties
// from the default in-memory.
// For instance, in a web setting, we may want to be able to persist data in
// webstorage so that on refresh, the app state can be recovered.
func (e *ElementStore) AddPersistenceMode(name string, loadFromStore func(*Element) error, store func(*Element, string, string, Value, ...bool)) *ElementStore {
	e.PersistentStorer[name] = storageFunctions{loadFromStore, store}
	return e
}

// NewAppRoot returns the starting point of an app. It is a viewElement whose main
// view neame is the root id.
func (e *ElementStore) NewAppRoot(id string) BasicElement {
	el := NewElement("root", id, e.DocType)
	el.root = el
	el.Parent = el // initially nil DEBUG
	el.subtreeRoot = el
	el.ElementStore = e
	el.Global = e.Global
	// DEBUG el.path isn't set

	el.Set("internals", "root", Bool(true))
	el.Set("event", "attached", Bool(true))
	el.Set("event", "mounted", Bool(true))
	el.Set("event", "mountable", Bool(true))
	el.Set("event", "firstmount", Bool(true))
	el.Set("event", "firsttimemounted", Bool(true))
	return BasicElement{el}
}

// NewConstructor registers and returns a new Element construcor function.
func (e *ElementStore) NewConstructor(elementname string, constructor func(name string, id string) *Element, options ...ConstructorOption) func(elname string, elid string, optionNames ...string) *Element {
	options = append(options, allowPropertyInheritanceOnMount)
	// First we register the options that are passed with the Constructor definition
	if options != nil {
		for _, option := range options {
			n := option.Name
			f := option.Configurator
			optlist, ok := e.ConstructorsOptions[elementname]
			if !ok {
				optlist = make(map[string]func(*Element) *Element)
				e.ConstructorsOptions[elementname] = optlist
			}
			optlist[n] = f
		}
	}

	// Then we create the element constructor to return
	c := func(name string, id string, optionNames ...string) *Element {
		element := constructor(name, id)
		element.Set("internals", "constructor", String(elementname))
		element.Global = e.Global
		element.ElementStore = e

		// Let's apply the global constructor options
		for _, fn := range e.GlobalConstructorOptions {
			element = fn(element)
		}
		// TODO optionalArgs  apply the corresponding options
		for _, opt := range optionNames {
			r, ok := e.ConstructorsOptions[elementname]
			if ok {
				config, ok := r[opt]
				if ok {
					element = config(element)
				}
			}
		}

		e.ByID[id] = element
		return element
	}
	e.Constructors[elementname] = c
	return c
}

func (e *ElementStore) GetByID(id string) *Element {
	v, ok := e.ByID[id]
	if !ok {
		return nil
	}
	return v
}

func (e *ElementStore) Delete(id string) {
	delete(e.ByID, id)
}

// Element is the building block of the User Interface. Everything is described
// as an Element having some mutable properties (graphic properties or data properties)
// From the window to the buttons on a page.
// Elements may have a unique parent: hence, Views cannot share any Element.
type Element struct {
	ElementStore *ElementStore
	Global       *Element // holds ownership of the global state
	root         *Element
	subtreeRoot  *Element // detached if subtree root has no parent unless subtreeroot == root
	path         *Elements

	Parent *Element

	Name    string
	ID      string
	DocType string

	Properties             PropertyStore
	PropMutationHandlers   *MutationCallbacks     // list of mutation handlers stored at elementID/propertyName (Elements react to change in other elements they are monitoring)
	EventHandlers          EventListeners         // EventHandlers are to be called when the named event has fired.
	NativeEventUnlisteners NativeEventUnlisteners // Allows to remove event listeners on the native element, registered when bridging event listeners from the native UI platform.

	Children   *Elements
	ActiveView string // holds the name of the view currently displayed. If parameterizable, holds the name of the parameter

	ViewAccessPath *viewNodes // List of views that lay on the path to the Element
	ViewAccessNode *viewAccessNode

	InactiveViews map[string]View

	Native NativeElement
}

func (e *Element) AsElement() *Element { return e }
func (e *Element) AsViewElement() (ViewElement, bool) {
	if e.isViewElement() {
		return ViewElement{e}, true
	}
	return ViewElement{nil}, false
}
func (e *Element) isViewElement() bool { return e.InactiveViews != nil }
func (e *Element) watchable()          {}

// NewElement returns a new Element with no properties, no event or mutation handlers.
// Essentially an empty shell to be customized.
func NewElement(name string, id string, doctype string) *Element {
	if strings.Contains(id, "/") {
		panic("An id may not use a slash: " + id + " is not valid.")
	}
	e := &Element{
		nil,
		nil,
		nil,
		nil,
		NewElements(),
		nil,
		name,
		id,
		doctype,
		NewPropertyStore(),
		NewMutationCallbacks(),
		NewEventListenerStore(),
		NewNativeEventUnlisteners(),
		NewElements(),
		"",
		newViewNodes(),
		newViewAccessNode(nil, ""),
		nil,
		nil,
	}
	return e
}

// Root returns the top-most eklement in the *Element tree.
// All navigation properties are registred on it.
func (e *Element) Root() *Element {
	return e.root
}

// AnyElement is an interface type implemented by *Element :
// Notably BasicElement and ViewElement.
type AnyElement interface {
	AsElement() *Element
}

func (e *Element) isRoot() bool {
	if e == nil {
		return false
	}
	return e == e.Parent
}

func PersistenceMode(e *Element) string {
	mode := ""
	v, ok := e.Get("internals", "persistence")
	if ok {
		s, ok := v.(String)
		if ok {
			mode = string(s)
		}
	}
	return mode
}

type Elements struct {
	List []*Element
}

func NewElements(elements ...*Element) *Elements {
	res := &Elements{make([]*Element, 0, 50)}
	res.List = append(res.List, elements...)
	return res
}

func (e *Elements) InsertLast(elements ...*Element) *Elements {
	e.List = append(e.List, elements...)
	return e
}

func (e *Elements) InsertFirst(elements ...*Element) *Elements {
	c := cap(e.List)
	l := len(e.List)
	le := len(elements)
	if c > (l + le) {
		e.List = e.List[:l+le]
		copy(e.List[le:], e.List)
		for i, element := range elements {
			e.List[i] = element
		}
		return e
	}
	DEBUG("cap is ", c, " and len is ", l, " for ", le, " new elements.")
	DEBUG(e)
	e.List = append(elements, e.List...)
	return e
}

func (e *Elements) Insert(el *Element, index int) *Elements {
	if index == len(e.List) {
		e.List = append(e.List, el)
		return e
	}
	e.List = append(e.List[:index+1], e.List[index:]...)
	e.List[index] = el
	return e
}

func (e *Elements) AtIndex(index int) *Element {
	elements := e.List
	if index < 0 || index >= len(elements) {
		return nil
	}
	return elements[index]
}

func (e *Elements) Remove(el *Element) *Elements {
	index := -1
	for k, element := range e.List {
		if element.ID == el.ID {
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
	cap := cap(e.List)
	e.List = make([]*Element, 0, cap)
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

func (e *Elements) Includes(el *Element) bool {
	for _, element := range e.List {
		if element.ID == el.ID {
			return true
		}
	}
	return false
}

// Handle calls up the event handlers in charge of processing the event for which
// the Element is listening.
func (e *Element) Handle(evt Event) bool {
	evt.SetCurrentTarget(e)
	return e.EventHandlers.Handle(evt)
}

// DispatchEvent is used typically to propagate UI events throughout the ui tree.
// If a nativebinding (type NativeEventBridge) is provided, the event will be dispatched
// on the native host only using the nativebinding function.
//
// It may require an event object to be created from the native event object implementation.
// Events are propagated following the model set by web browser DOM events:
// 3 phases being the capture phase, at-target and then bubbling up if allowed.
func (e *Element) DispatchEvent(evt Event, nativebinding NativeDispatch) *Element {
	if !e.Mounted() {
		panic("FAILURE: element notmounted? " + e.ID)
		return e
	}
	if nativebinding != nil {
		nativebinding(evt)
		return e
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


// attach will link a child Element to the subtree its target parent belongs to.
// It does not however position it in any view specifically. At this stage,
// the Element can not be rendered as part of the view.
func attach(parent *Element, child *Element, activeview bool) {
	DEBUG("attach")
	if activeview {
		DEBUG("active view attached")
		child.Parent = parent
		child.path.InsertFirst(parent).InsertFirst(parent.path.List...)
	}
	child.root = parent.root
	child.subtreeRoot = parent.subtreeRoot

	child.link(BasicElement{parent})
	//child.ViewAccessPath = computePath(child.ViewAccessPath,child.ViewAccessNode)

	for _, descendant := range child.Children.List {
		detach(descendant)
		attach(child, descendant, true)
	}

	for _, descendants := range child.InactiveViews {
		for _, descendant := range descendants.Elements().List {
			detach(descendant)
			attach(child, descendant, false)
		}
	}

	child.Set("event", "attach", Bool(true))
	if child.Mountable() {
		child.Set("event", "mountable", Bool(true))
		// we can set mountable without checking if it was already set because we
		// know that attach is only called for detached subtrees, i.e. they are also
		// not mountable nor mounted.
		if child.Mounted() {
			_, ok := child.Get("event", "firstmount")
			if !ok {
				child.Set("event", "firstmount", Bool(true))
			}
			child.Set("event", "mount", Bool(true))
		}
	}
}

func finalize(child *Element, attached bool) {
	if attached {
		if child.Mounted() {
			_, ok := child.Get("event", "firsttimemounted")
			if !ok {
				child.Set("event", "firsttimemounted", Bool(true))
			}
			child.Set("event", "mounted", Bool(true))
		}

	} else {
		m, ok := child.Get("event", "mounted")
		if ok {
			if vm := m.(Bool); vm {
				child.Set("event", "mounted", Bool(false))
			}
		}
	}

	for _, descendant := range child.Children.List {
		finalize(descendant, attached)
	}
}

// detach will unlink an Element from its parent. If the element was in a view,
// the element is still being rendered until it is removed. However, it should
// not be able to react to events or mutations. TODO review the latter part.
func detach(e *Element) {
	if e.Parent == nil {
		return
	}

	if e.isRoot() {
		panic("FAILURE: attempt to detach the root element.")
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
	e.ViewAccessNode.previous = nil
	//e.ViewAccessPath = computePath(newViewNodes(), e.ViewAccessNode)

	e.Set("event", "attach", Bool(false))
	e.Set("event", "mountable", Bool(false))
	e.Set("event", "mount", Bool(false)) // i.e. unmount

	// got to update the subtree with the new subtree root and path
	for _, descendant := range e.Children.List {
		detach(descendant)
		attach(e, descendant, true)
	}

	for _, descendants := range e.InactiveViews {
		for _, descendant := range descendants.Elements().List {
			detach(descendant)
			attach(e, descendant, false)
		}
	}
}

func (e *Element) hasParent(any AnyElement) bool {
	anye := any.AsElement()
	if e.path == nil {
		return false
	}
	parents := e.path.List
	for _, parent := range parents {
		if parent.ID == anye.ID {
			return true
		}
	}
	return false
}

// AppendChild appends a new element to the Element's children.
// If the element being appended is mounted on the main tree that starts from a
// root Element, the root Element will see its ("event","docupdate") property
// set with the value of the appendee.
func (e *Element) AppendChild(childEl AnyElement) *Element {
	return e.appendChild(childEl)
}

func (e *Element) appendChild(childEl AnyElement) *Element {
	child := childEl.AsElement()
	if e.DocType != child.DocType {
		log.Printf("Doctypes do not match. Parent has %s while child Element has %s", e.DocType, child.DocType)
		return e
	}
	if child.Parent != nil {
		child.Parent.removeChild(BasicElement{child})
	}

	attach(e, child, true)
	e.Children.InsertLast(child)

	if e.Native != nil {
		e.Native.AppendChild(child)
	}

	child.Set("event", "attached", Bool(true))

	finalize(child, true)

	return e
}

func (e *Element) PrependChild(childEl AnyElement) *Element {
	return e.prependChild(childEl)
}

func (e *Element) prependChild(childEl AnyElement) *Element {
	child := childEl.AsElement()
	if e.DocType != child.DocType {
		log.Printf("Doctypes do not match. Parent has %s while child Element has %s", e.DocType, child.DocType)
		return e
	}

	if child.Parent != nil {
		child.Parent.removeChild(BasicElement{child})
	}

	attach(e, child, true)
	e.Children.InsertFirst(child)

	if e.Native != nil {
		e.Native.PrependChild(child)
	}

	child.Set("event", "attached", Bool(true))

	finalize(child, true)

	return e
}

func (e *Element) InsertChild(childEl AnyElement, index int) *Element {
	return e.insertChild(childEl, index)
}

func (e *Element) insertChild(childEl AnyElement, index int) *Element {
	child := childEl.AsElement()
	if e.DocType != child.DocType {
		log.Printf("Doctypes do not match. Parent has %s while child Element has %s", e.DocType, child.DocType)
		return e
	}
	if child.Parent != nil {
		child.Parent.removeChild(BasicElement{child})
	}

	attach(e, child, true)
	e.Children.Insert(child, index)

	if e.Native != nil {
		e.Native.InsertChild(child, index)
	}

	child.Set("event", "attached", Bool(true))

	finalize(child, true)

	return e
}

func (e *Element) ReplaceChild(old AnyElement, new AnyElement) *Element {
	return e.replaceChild(old, new)
}

// replaceChild will replace the target child Element with another.
// Be wary that mutation Watchers and event listeners remain unchanged by default.
// The addition or removal of change observing objects is left at the discretion
// of the user.
func (e *Element) replaceChild(oldEl AnyElement, newEl AnyElement) *Element {
	old := oldEl.AsElement()
	new := newEl.AsElement()
	if e.DocType != new.DocType {
		log.Printf("Doctypes do not match. Parent has %s while child Element has %s", e.DocType, new.DocType)
		return e
	}

	_, ok := e.hasChild(old)
	if !ok {
		return e
	}

	if new.Parent != nil {
		new.Parent.removeChild(BasicElement{new})
	}

	detach(old.AsElement())
	attach(e, new.AsElement(), true)
	e.Children.Replace(old.AsElement(), new.AsElement())

	if e.Native != nil {
		e.Native.ReplaceChild(old.AsElement(), new.AsElement())
	}

	old.Set("event", "attached", Bool(false))
	new.Set("event", "attached", Bool(true))

	finalize(old, false)
	finalize(new, true)

	return e
}

func (e *Element) RemoveChild(childEl AnyElement) *Element {
	return e.removeChild(childEl)
}

func (e *Element) removeChild(childEl AnyElement) *Element {
	child := childEl.AsElement()
	_, ok := e.hasChild(child)
	if !ok {
		return e
	}

	detach(child)
	e.Children.Remove(child)

	if e.Native != nil {
		e.Native.RemoveChild(child)
	}

	child.Set("event", "attached", Bool(false))
	finalize(child, false)

	return e
}

func (e *Element) RemoveChildren() *Element {
	return e.removeChildren()
}

func (e *Element) removeChildren() *Element {
	l := make([]*Element, len(e.Children.List))
	copy(l, e.Children.List)
	for _, child := range l {
		e.removeChild(BasicElement{child})
	}
	return e
}

func (e *Element) DeleteChild(childEl AnyElement) *Element {
	child := childEl.AsElement()
	child.Set("event", "deleting", Bool(true))
	child.DeleteChildren()
	e.RemoveChild(childEl)

	if child.isViewElement() {
		for _, view := range child.InactiveViews {
			for _, el := range view.Elements().List {
				el.Set("internals", "deleted", Bool(true))
				e.ElementStore.Delete(el.ID)
			}
		}
	}

	child.Set("internals", "deleted", Bool(true))
	e.ElementStore.Delete(child.ID)

	return e
}

func (e *Element) DeleteChildren() *Element {
	l := make([]*Element, len(e.Children.List))
	copy(l, e.Children.List)
	for _, child := range l {
		e.DeleteChild(BasicElement{child})
	}
	return e
}

func (e *Element) ShareLifetimeOf(any AnyElement) *Element {
	e.Watch("internals", "deleted", any.AsElement(), NewMutationHandler(func(evt MutationEvent) bool {
		if e.Parent != nil {
			e.Parent.DeleteChild(e)
			return false
		}
		e.DeleteChildren()
		_, ok := e.Get("internals", "deleted")
		if !ok {
			e.Set("internals", "deleted", Bool(true))
			e.ElementStore.Delete(e.ID)
		}
		return false
	}))
	return e
}

func (e *Element) hasChild(any *Element) (int, bool) {
	if e.Children == nil {
		return -1, false
	}

	for k, child := range e.Children.List {
		if child.ID == any.ID {
			return k, true
		}
	}
	return -1, false
}

func (e *Element) SetChildren(any ...AnyElement) *Element {
	e.RemoveChildren()
	if n, ok := e.Native.(interface{ SetChildren(...*Element) }); ok {
		children := make([]*Element, 0, len(any))
		for _, el := range any {
			ele := el.AsElement()
			if e.DocType != ele.DocType {
				log.Printf("Doctypes do not match. Parent has %s while child Element has %s", e.DocType, ele.DocType)
				panic("SetChildren failed: wrong doctype")
			}
			if ele.Parent != nil {
				ele.Parent.removeChild(BasicElement{ele})
			}
			attach(e, ele, true)
			e.Children.InsertLast(ele)
			children = append(children, ele)
		}
		n.SetChildren(children...)
		for _, child := range children {
			child.Set("event", "attached", Bool(true))
			finalize(child, true)
		}
		return e
	}
	for _, el := range any {
		e.AppendChild(el)
		// el.ActiveView = e.ActiveView // TODO verify this is correct
	}
	return e
}

func (e *Element) SetChildrenElements(any ...*Element) *Element {
	e.RemoveChildren()
	if n, ok := e.Native.(interface{ SetChildren(...*Element) }); ok {
		for _, el := range any {
			if e.DocType != el.DocType {
				log.Printf("Doctypes do not match. Parent has %s while child Element has %s", e.DocType, el.DocType)
				panic("SetChildren failed: wrong doctype")
			}
			if el.Parent != nil {
				el.Parent.removeChild(BasicElement{el})
			}
			attach(e, el, true)
			e.Children.InsertLast(el)
		}
		n.SetChildren(any...)
		for _, child := range any {
			child.Set("event", "attached", Bool(true))
			finalize(child, true)
		}
		return e
	}
	for _, el := range any {
		e.AppendChild(el)
		// el.ActiveView = e.ActiveView // TODO verify this is correct
	}
	return e
}

// SetMergeChildren will set an Element children list without removing
// Elements that are already present.
func (e *Element) SetMergeChildren(any ...AnyElement) *Element {
	nc := make([]*Element, 0, len(any))
	for _, el := range any {
		nc = append(nc, el.AsElement())
	}
	transform(e, nc, false)
	return e
}

// SetMergeChildren will set an Element children list without removing
// Elements that are already present.
// It deletes the elements that are absent from the children list.
func (e *Element) SetMergeChildrenClean(any ...AnyElement) *Element {
	nc := make([]*Element, 0, len(any))
	for _, el := range any {
		nc = append(nc, el.AsElement())
	}
	transform(e, nc, true)
	return e
}

type childrenSet map[string]int

func newChildrenSet(list ...*Element) childrenSet {
	m := make(map[string]int, len(list))
	for index, element := range list {
		m[element.ID] = index
	}
	return m
}

func (s childrenSet) Contains(id string) bool {
	_, ok := s[id]
	return ok
}
func intersect(s childrenSet, c childrenSet) childrenSet {
	m := make(map[string]int, len(s))
	for id := range c {
		if s.Contains(id) {
			m[id] = 0
		}
	}
	return m
}

func (s childrenSet) Ordered(l []*Element) (childrenSet, []*Element) {
	res := make([]*Element, 0, len(s))
	m := make(map[string]int, len(s))
	n := 0
	for _, e := range l {
		if s.Contains(e.ID) {
			res = append(res, e)
			m[e.ID] = n
			n++
		}
	}
	return m, res
}

func transform(parent *Element, destination []*Element, delete bool) {
	original := make([]*Element, len(parent.Children.List))
	copy(original, parent.Children.List)

	oriset := newChildrenSet(original...)
	destset := newChildrenSet(destination...)
	inter := intersect(oriset, destset)

	oset, olist := inter.Ordered(original)
	_, dlist := inter.Ordered(destination)

	for i, e := range dlist {
		oldindex := oset[e.ID]
		oldpos := oriset[e.ID]

		if oldindex != i {
			oldelement := olist[i]
			oldelementoldpos := oriset[oldelement.ID]

			parent.InsertChild(e, oldelementoldpos)
			parent.InsertChild(oldelement, oldpos)
			oset[e.ID] = i
			oset[oldelement.ID] = oldindex

			oriset[e.ID] = oldelementoldpos
			oriset[oldelement.ID] = oldpos
			original[oldpos] = oldelement
			original[oldelementoldpos] = e
		}
	}
	// Now that the best LCS has been formed, we modify the original slice by
	// deletion/insertion.
	orig := original[:0]
	for _, e := range original {
		if !destset.Contains(e.ID) {
			if delete {
				parent.DeleteChild(e)
				continue
			}
			parent.RemoveChild(e)
			continue
		}
		orig = append(orig, e)
	}
	for i := len(orig); i < len(original); i++ { // cleanup for garbage collection
		original[i] = nil
	}

	insertAt := 0
	for _, e := range destination {
		if !oriset.Contains(e.ID) {
			parent.InsertChild(e, insertAt)
			insertAt++
			continue
		}
		insertAt++
	}
}

// Watch observes the owner of a property registered under a given category for change.
// When a change has occured, a callback registered as a *MutationHandler is applied.
func (e *Element) Watch(category string, propname string, owner Watchable, h *MutationHandler) *Element {
	p, ok := owner.AsElement().Properties.Categories[category]
	if !ok {
		p = newProperties()
		owner.AsElement().Properties.Categories[category] = p
	}
	alreadywatching := p.IsWatching(propname, e)

	if !alreadywatching {
		p.NewWatcher(propname, e)
	}

	e.PropMutationHandlers.Add(owner.AsElement().ID+"/"+category+"/"+propname, h)

	eventcat, ok := owner.AsElement().Properties.Categories["internals"]
	if !ok {
		eventcat = newProperties()
		owner.AsElement().Properties.Categories["internals"] = eventcat
	}
	alreadywatching = eventcat.IsWatching("deleted", e)

	if !alreadywatching {
		eventcat.NewWatcher("deleted", e)
	}

	e.PropMutationHandlers.Add(owner.AsElement().ID+"/"+"internals"+"/"+"deleted", NewMutationHandler(func(evt MutationEvent) bool {
		e.Unwatch(category, propname, owner)
		return false
	}))

	return e
}

// WatchASAP is similar to watch but triggers the callback as soon as possible,
// if a value has been detected for the property, while Watch awaits for a
// mutation event to occur.
// Mostly useful when a callback should be triggered as soon as we detect that a
// event property ("event", propname) has occured.
// (Reminder: event properies are properties that are only mutated at runtime
// and cannot be stored in any long-term storage. They define transient state).
func (e *Element) WatchASAP(category string, propname string, owner Watchable, h *MutationHandler) *Element {
	p, ok := owner.AsElement().Properties.Categories[category]
	if !ok {
		p = newProperties()
		owner.AsElement().Properties.Categories[category] = p
	}
	alreadywatching := p.IsWatching(propname, e)

	if !alreadywatching {
		p.NewWatcher(propname, e)
	}

	e.PropMutationHandlers.Add(owner.AsElement().ID+"/"+category+"/"+propname, h)

	eventcat, ok := owner.AsElement().Properties.Categories["internals"]
	if !ok {
		eventcat = newProperties()
		owner.AsElement().Properties.Categories["internals"] = eventcat
	}
	alreadywatching = eventcat.IsWatching("deleted", e)

	if !alreadywatching {
		eventcat.NewWatcher("deleted", e)
	}

	e.PropMutationHandlers.Add(owner.AsElement().ID+"/"+"internals"+"/"+"deleted", NewMutationHandler(func(evt MutationEvent) bool {
		e.Unwatch(category, propname, owner)
		return false
	}))

	val, ok := owner.AsElement().Get(category, propname)
	if ok {
		h.Handle(owner.AsElement().NewMutationEvent(category, propname, val, nil))
	}

	return e
}

// Unwatch cancels mutation observing for the property of the owner Element
// registered under the given category.
func (e *Element) Unwatch(category string, propname string, owner Watchable) *Element {
	p, ok := owner.AsElement().Properties.Categories[category]
	if !ok {
		return e
	}
	p.RemoveWatcher(propname, e)
	e.PropMutationHandlers.RemoveAll(owner.AsElement().ID + "/" + category + "/" + propname)
	return e
}

// WatchGroup enables the triggering of a callback for any change to a given catgeory.
func (e *Element) WatchGroup(category string, target Watchable, h *MutationHandler) *Element {
	p, ok := target.AsElement().Properties.Categories[category]
	if !ok {
		p = newProperties()
		target.AsElement().Properties.Categories[category] = p
	}
	p.NewWatcher("existifallpropertieswatched", e)
	e.PropMutationHandlers.Add(target.AsElement().ID+"/"+category+"/"+"existifallpropertieswatched", h)
	return e
}

// UnwatchGroup cancels the monitoring of changes for a given category.
func (e *Element) UnwatchGroup(category string, owner *Element) *Element {
	p, ok := owner.Properties.Categories[category]
	if !ok {
		return e
	}
	p.RemoveWatcher("existifallpropertieswatched", e)
	return e
}

/*
func (e *Element) AddEventListener(event string, handler *EventHandler, nativebinding NativeEventBridge) *Element {
	e.EventHandlers.AddEventHandler(event, handler)
	if nativebinding != nil {
		nativebinding(event, e)
	}
	return e

}
*/
func (e *Element) RemoveEventListener(event string, handler *EventHandler, native bool) *Element {
	e.EventHandlers.RemoveEventHandler(event, handler)
	if native {
		if e.NativeEventUnlisteners.List != nil {
			e.NativeEventUnlisteners.Apply(event)
		}
	}
	return e
}

func (e *Element) AddEventListener(event string, handler *EventHandler, nativebinding NativeEventBridge) *Element {
	h := NewMutationHandler(func(evt MutationEvent) bool {
		e.EventHandlers.AddEventHandler(event, handler)
		if nativebinding != nil {
			nativebinding(event, e)
		}
		return false
	})
	e.OnFirstTimeMounted(h)

	/*g := NewMutationHandler(func(evt MutationEvent) bool {
		var native bool
		if nativebinding != nil {
			native = true
		}
		e.RemoveEventListener(event, handler, native)

		return false
	})

	e.OnDisMount(g)

	*/

	e.OnDeleted(NewMutationHandler(func(evt MutationEvent) bool {
		var native bool
		if nativebinding != nil {
			native = true
		}

		e.RemoveEventListener(event, handler, native)

		return false
	}))

	return e
}

// Mountable returns whether the element is attached to the main app tree.
// This includes Mounted Elements, and Elements that are part of an inactive view.
func (e *Element) Mountable() bool {
	if e.isRoot() {
		return true
	}

	if e.Root() == nil {
		return false
	}
	_, isroot := e.Root().Get("internals", "root")
	return isroot
}

// Mounted returns true if an Element is directly reachable from the root of an app tree.
// This does not include elements existing on inactivated view paths.
func (e *Element) Mounted() bool {
	if e.isRoot() {
		return true
	}

	if !e.Mountable() {
		return false
	}
	l := e.path.List
	if len(l) == 0 {
		return false
	}
	return l[0].isRoot()
}

func (e *Element) OnMount(h *MutationHandler) {
	nh := NewMutationHandler(func(evt MutationEvent) bool {
		b, ok := evt.NewValue().(Bool)
		if !ok || !bool(b) {
			return false
		}
		return h.Handle(evt)
	})
	e.Watch("event", "mount", e, nh)
}

func (e *Element) OnMounted(h *MutationHandler) {
	nh := NewMutationHandler(func(evt MutationEvent) bool {
		b, ok := evt.NewValue().(Bool)
		if !ok || !bool(b) {
			return false
		}
		return h.Handle(evt)
	})
	e.Watch("event", "mounted", e, nh)
}

func (e *Element) OnFirstTimeMounted(h *MutationHandler) {
	nh := NewMutationHandler(func(evt MutationEvent) bool {
		b, ok := evt.NewValue().(Bool)
		if !ok || !bool(b) {
			return false
		}
		return h.Handle(evt)
	})
	e.WatchASAP("event", "firsttimemounted", e, nh)
}

func (e *Element) OnUnmount(h *MutationHandler) {
	nh := NewMutationHandler(func(evt MutationEvent) bool {
		b, ok := evt.NewValue().(Bool)
		if !ok {
			return true
		}
		if bool(b) {
			return false
		}
		return h.Handle(evt)
	})
	e.Watch("event", "mount", e, nh)
}

func (e *Element) OnUnmounted(h *MutationHandler) {
	nh := NewMutationHandler(func(evt MutationEvent) bool {
		b, ok := evt.NewValue().(Bool)
		if !ok {
			return true
		}
		if bool(b) {
			return false
		}
		return h.Handle(evt)
	})
	e.Watch("event", "mounted", e, nh)
}

func (e *Element) OnDeleted(h *MutationHandler) {
	eventcat, ok := e.Properties.Categories["internals"]
	if !ok {
		eventcat = newProperties()
		e.Properties.Categories["internals"] = eventcat
	}
	alreadywatching := eventcat.IsWatching("deleted", e)

	if !alreadywatching {
		eventcat.NewWatcher("deleted", e)
	}

	e.PropMutationHandlers.Add(e.ID+"/"+"internals"+"/"+"deleted", h)
}

// Get retrieves the value stored for the named property located under the given
// category. The "" category returns the content of the "global" property category.
// The "global" namespace is a local copy of the data that resides in the global
// shared scope common to all Element objects of an ElementStore.
func (e *Element) Get(category, propname string) (Value, bool) {
	return e.Properties.Get(category, propname)
}

// Set inserts a key/value pair under a given category in the element property store.
// First flag in the variadic argument, if true, denotes whether the property should be inheritable.
// The "ui" category is unformally reserved for properties that are a UI representation
// of data.
// NOTE: One category is never persisted: "event" as it corresonds
// to transient, runtime-only props.
func (e *Element) Set(category string, propname string, value Value, flags ...bool) {
	if strings.Contains(category, "/") || strings.Contains(propname, "/") {
		panic("category string and/or propname seems to contain a slash. This is not accepted. (" + category + "," + propname + ")")
	}
	var inheritable bool
	if len(flags) > 0 {
		inheritable = flags[0]
	}
	// Persist property if persistence mode has been set at Element creation
	pmode := PersistenceMode(e)

	oldvalue, ok := e.Get(category, propname)

	if ok {
		if category == "ui" {
			_, ok := e.Get("internals", "memoize")
			if ok {
				if Equal(value, oldvalue) { // idempotence
					return
				}
			}
		}
	}

	if e.ElementStore != nil {
		storage, ok := e.ElementStore.PersistentStorer[pmode]
		if ok && category != "event" {
			storage.Store(e, category, propname, value, flags...)
		}
	}

	e.Properties.Set(category, propname, value, inheritable)

	// Mutation event propagation
	evt := e.NewMutationEvent(category, propname, value, oldvalue)

	props, ok := e.Properties.Categories[category]
	if !ok {
		panic("category should exist since property should have been stored")
	}
	watchers, ok := props.Watchers[propname]
	if !ok {
		return
	}
	if watchers == nil {
		return
	}
	for _, w := range watchers.List {
		w.PropMutationHandlers.DispatchEvent(evt)
	}
	//e.PropMutationHandlers.DispatchEvent(evt)
}

func (e *Element) Memoize() *Element {
	e.Set("internals", "memoize", Bool(true))
	return e
}

func (e *Element) GetData(propname string) (Value, bool) {
	return e.Get("data", propname)
}

// SetData inserts a key/value pair under the "data" category in the element property store.
// First flag in the variadic argument, if true, denotes whether the property should be inheritable.
// It does not automatically update any potential property representation stored
// for rendering use in the "ui" category/namespace.
func (e *Element) SetData(propname string, value Value, flags ...bool) {
	e.Set("data", propname, value, flags...)
}

// SetUI stores data used for Graphical rendering in the "ui" namespace (stands for
// user interface). This namespace should remain private to an Element and used
// to trigger User Interface updates.
// Hence, the UI State is constituted of the values used for the representation of
// data to the end-user.
// Synchronization between the Data and the UI is therfore manual.
// UI props shall not be interdependent.
// Other Element may want to "watch" the corresponding data namespace instead if
// there exist interconnections.
func (e *Element) SetUI(propname string, value Value, flags ...bool) {
	e.Set("ui", propname, value, flags...)
}

// SetDataSetUI will set a "data" property and update the same-name property value
// located in the "ui namespace/category and used to update the the User Interface, for instance, for rendering..
// Typically NOT used when the data is being updated from the UI.
// This is used where Synchronization between the data and its representation are
// needed. (in opposition to automatically updating the UI via data mutation observervation)
func (e *Element) SetDataSetUI(propname string, value Value, flags ...bool) {
	var inheritable bool
	if len(flags) > 0 {
		inheritable = flags[0]
	}
	oldvalue, _ := e.GetData(propname)
	// Persist property if persistence mode has been set at Element creation
	pmode := PersistenceMode(e)

	if e.ElementStore != nil {
		storage, ok := e.ElementStore.PersistentStorer[pmode]
		if ok {
			storage.Store(e, "data", propname, value, flags...)
		}
	}
	e.Properties.Set("data", propname, value, inheritable)

	e.Set("ui", propname, value, flags...)

	evt := e.NewMutationEvent("data", propname, value, oldvalue)

	props, ok := e.Properties.Categories["data"]
	if !ok {
		panic("category should exist since property should have been stored")
	}
	watchers, ok := props.Watchers[propname]
	if !ok {
		return
	}
	if watchers == nil {
		return
	}
	for _, w := range watchers.List {
		w.PropMutationHandlers.DispatchEvent(evt)
	}
}

// SyncUISetData is used in event handlers when a user changed a value accessible
// via the User Interface, typically.
// It does not trigger mutationahdnler of the "ui" namespace
// (to avoid rerendering an already up-to-date User Interface)
//
//
// First flag in the variadic argument, if true, denotes whether the property should be inheritable.
func (e *Element) SyncUISetData(propname string, value Value, flags ...bool) {
	var inheritable bool
	if len(flags) > 0 {
		inheritable = flags[0]
	}

	// Persist property if persistence mode has been set at Element creation
	pmode := PersistenceMode(e)

	if e.ElementStore != nil {
		storage, ok := e.ElementStore.PersistentStorer[pmode]
		if ok {
			storage.Store(e, "ui", propname, value, flags...)
		}
	}

	e.Properties.Set("ui", propname, value, inheritable)

	e.Set("data", propname, value, flags...)
}

// SyncUI can be used to synchronize the UI state. It does not trigger
// mutation handlers which are typically used to propagate UI state changes to
// the User Interface.
// This method can be used when data state and ui state are decoupled. (rare)
// An example of such decoupling is browser history and the framework handling
// of the navigation history.
// In fact, this decoupling is somewhat artificial: navigation history just cannot be
// recovered entirely from ("ui","history") because it also requires ("ui","currentroute").
func (e *Element) SyncUI(propname string, value Value, flags ...bool) {
	var inheritable bool
	if len(flags) > 0 {
		inheritable = flags[0]
	}

	// Persist property if persistence mode has been set at Element creation
	pmode := PersistenceMode(e)

	if e.ElementStore != nil {
		storage, ok := e.ElementStore.PersistentStorer[pmode]
		if ok {
			storage.Store(e, "ui", propname, value, flags...)
		}
	}

	e.Properties.Set("ui", propname, value, inheritable)
}

// LoadProperty is a function typically used to return a UI Element to a
// given state.
// The proptype is a string that describes the property (default,inherited, local, or inheritable).
// For properties of the 'ui' namespace, i.e. properties that are used for rendering,
// we create and dispatch a mutation event since loading a property is change inducing at the
// UI level.
func LoadProperty(e *Element, category string, propname string, proptype string, value Value) {
	e.Properties.Load(category, propname, proptype, value)
	if category == "ui" {
		evt := e.NewMutationEvent(category, propname, value, nil)
		e.PropMutationHandlers.DispatchEvent(evt)
	}
}

// Delete removes the property stored for the given category if it exists.
// Inherited properties cannot be deleted.
// Default properties cannot be deleted either for now.
func (e *Element) Delete(category string, propname string) {
	oldvalue, _ := e.Get(category, propname)
	e.Properties.Delete(category, propname)
	evt := e.NewMutationEvent(category, propname, nil, oldvalue)
	e.PropMutationHandlers.DispatchEvent(evt)
}

func SetDefault(e *Element, category string, propname string, value Value) {
	e.Properties.SetDefault(category, propname, value)
}

func InheritProperties(target *Element, src *Element, categories ...string) {
	for cat, ps := range src.Properties.Categories {
		if categories != nil {
			for _, c := range categories {
				if c == cat {
					pst, ok := target.Properties.Categories[cat]
					if !ok {
						pst = newProperties()
						target.Properties.Categories[cat] = pst
					}
					pst.Inherit(ps)
					break
				}
			}
			continue
		}
		pst, ok := target.Properties.Categories[cat]
		if !ok {
			pst = newProperties()
			target.Properties.Categories[cat] = pst
		}
		pst.Inherit(ps)
	}
}

var allowPropertyInheritanceOnMount = NewConstructorOption("propertyinheritance", func(e *Element) *Element {
	h := NewMutationHandler(func(evt MutationEvent) bool {
		element := evt.Origin()
		InheritProperties(element, element.Parent)
		return false
	})
	e.OnMount(h)
	return e
})

// EnablePropertyAutoInheritance is an option that when passed to an Element
// constructor, allows an Element to inherit the properties of its parent
// when it is mounted in the DOM tree.
func EnablePropertyAutoInheritance() string {
	return "propertyinheritance"
}

// Route returns the path to an Element.
// If the path to an Element includes a parameterized view, the returned route is
// parameterized as well.
//
// Important notice: views that are nested within a fixed element use that Element ID for routing.
// In effect, the id acts as a namespace.
// In order for links using the routes to these views to not be breaking between refresh/reruns of an app (hard requirement for online link-sharing), the ID of the parent element
// should be generated so as to not change. Using a PRNG-based ID generator is very likely to not be a good-fit here.
//
// For instance, if we were to create a dynamic view composed of retrieved tweets, we would not use an ID generator but probably reuse the tweet ID gotten via http call for each Element.
// Building a shareable link toward any of these elements still require that every ID generated in the path is stable across app refresh/re-runs.
func (e *Element) Route() string {
	var uri string
	e.ViewAccessPath = computePath(newViewNodes(), e.ViewAccessNode)
	if e.ViewAccessPath == nil || len(e.ViewAccessPath.Nodes) == 0 {
		return uri
	}

	for k, n := range e.ViewAccessPath.Nodes {
		path := "/" + n.Element.ID + "/" + n.Name
		if k == 0 {
			if e.Mountable() {
				path = "/" + n.Name
			}
		}
		uri = uri + path
	}
	return uri
}

// View defines a type for a named list of children Element
// A View can depend on a parameter.
type View struct {
	name         string
	elements     *Elements
	Parameterize func(parameter string, v View) (*View, error)
}

func (v View) Name() string        { return v.name }
func (v View) Elements() *Elements { return v.elements }
func (v View) ApplyParameter(paramvalue string) (*View, error) {
	return v.Parameterize(paramvalue, v)
}

// NewView can be used to create a list of children Elements to append to an element, for display.
// In effect, allowing to create a named view. (note the lower case letter)
// The true definition of a view is: an *Element and a named list of child Elements (View) constitute a view.
// An example of use would be an empty window that would be filled with different child elements
// upon navigation.
// A parameterized view can be created by using a naming scheme such as ":parameter" (string with a leading colon)
// In the case, the parameter can be retrieve by the router.
func NewView(name string, elements ...*Element) View {
	for _, el := range elements {
		el.ActiveView = name
	}
	return View{name, NewElements(elements...), nil}
}

/*
// NewParameterizedView defines a parameterized, named, list of *Element composing a view.
// The Elements can be parameterized by applying a function submitted as argument.
// This function can and probably should implement validation.
// It may for instance be used to verify that the parameter value belongs to a finite
// set of accepted values.
func NewParameterizedView(parametername string, paramFn func(string, View) (*View, error), elements ...*Element) View {
	if !strings.HasPrefix(parametername, ":") {
		parametername = ":" + parametername
	}
	n := NewView(parametername, elements...)
	n.Parameterize = paramFn
	return n
}
*/

type viewAccessNode struct {
	previous *viewAccessNode
	*Element
	viewname string
}

func newViewAccessNode(v *Element, viewname string) *viewAccessNode {
	return &viewAccessNode{nil, v, viewname}
}

func (v *viewAccessNode) Link(any AnyElement) {
	e := any.AsElement()
	if !e.isViewElement() {
		if e.ViewAccessNode != nil {
			v.previous = e.ViewAccessNode.previous
			//v.viewname = e.ViewAccessNode.viewname
			v.Element = e.ViewAccessNode.Element
			return
		}
		v.previous = nil
		v.Element = nil
		v.viewname = ""
		return
	}
	v.previous = e.ViewAccessNode
	v.Element = e
}

func (child *Element) link(any AnyElement) {
	parent := any.AsElement()
	if !parent.isViewElement() {
		child.ViewAccessNode = parent.ViewAccessNode
		return
	}
	child.ViewAccessNode.Element = parent
	child.ViewAccessNode.previous = parent.ViewAccessNode
}

func computePath(p *viewNodes, v *viewAccessNode) *viewNodes {
	if v == nil || v.Element == nil {
		return p
	}
	node := newViewNode(v.Element, v.viewname)
	p.Prepend(node)
	if v.previous != nil {
		return computePath(p, v.previous)
	}
	return p
}

/*
unc(v *viewAccessNode) LinkView(any ViewElement,viewname string){
	e:= any.AsElement()

	insert:= newViewAccessNode(e,viewname)
	insert.previous = e.ViewAccessNode.previous
	v.previous = insert
}

func(v *viewAccessNode) Link(e *Element){
	if !e.isViewElement(){
		v.previous = e.ViewAccessNode.previous
		return
	}
}

*/

type viewNodes struct {
	Nodes []viewNode
}

func newViewNodes() *viewNodes {
	return &viewNodes{make([]viewNode, 0, 30)}
}

func (v *viewNodes) Copy() *viewNodes {
	if v == nil {
		return nil
	}
	c := make([]viewNode, len(v.Nodes), cap(v.Nodes))
	copy(c, v.Nodes)
	return &viewNodes{c}
}

func (v *viewNodes) Append(Nodes ...viewNode) *viewNodes {
	v.Nodes = append(v.Nodes, Nodes...)
	return v
}

func (v *viewNodes) Prepend(Nodes ...viewNode) *viewNodes {
	v.Nodes = append(Nodes, v.Nodes...)
	return v
}

type viewNode struct {
	*Element
	Name string
}

func newViewNode(e *Element, viewname string) viewNode {
	return viewNode{e, viewname}
}

// PropertyStore allows for the storage of Key Value pairs grouped by namespaces
// called categories. Key being the property name and Value its value.
type PropertyStore struct {
	Categories map[string]Properties
}

func NewPropertyStore() PropertyStore {
	return PropertyStore{make(map[string]Properties)}
}

func (p PropertyStore) Load(category string, propname string, proptype string, value Value) {
	ps, ok := p.Categories[category]
	if !ok {
		ps = newProperties()
		p.Categories[category] = ps
	}
	proptype = strings.ToLower(proptype)
	switch proptype {
	case "default":
		ps.Default[propname] = value
	case "inherited":
		ps.Inherited[propname] = value
	case "local":
		ps.Local[propname] = value
	case "inheritable":
		ps.Inheritable[propname] = value
	default:
		return
	}
}

// Get retrieves the value of a property stored within a given category.
// A category acts as a namespace for property keys.
func (p PropertyStore) Get(category string, propname string) (Value, bool) {
	ps, ok := p.Categories[category]
	if !ok {
		return nil, false
	}
	return ps.Get(propname)
}

func (p PropertyStore) Set(category string, propname string, value Value, flags ...bool) {
	ps, ok := p.Categories[category]
	if !ok {
		ps = newProperties()
		p.Categories[category] = ps
	}
	var Inheritable bool
	if len(flags) > 0 {
		Inheritable = flags[0]
	}
	ps.Set(propname, value, Inheritable)
}

func (p PropertyStore) Delete(category string, propname string) {
	ps, ok := p.Categories[category]
	if !ok {
		return
	}
	ps.Delete(propname)
}

func (p PropertyStore) HasCategory(category string) bool {
	_, ok := p.Categories[category]
	return ok
}

func (p PropertyStore) HasProperty(category string, propname string) bool {
	ps, ok := p.Categories[category]
	if !ok {
		return false
	}

	_, ok = ps.Get(propname)
	return ok
}

func (p PropertyStore) SetDefault(category string, propname string, value Value) {
	ps, ok := p.Categories[category]
	if !ok {
		ps = newProperties()
		p.Categories[category] = ps
	}
	ps.SetDefault(propname, value)
}

type Properties struct {
	Default map[string]Value

	Inherited map[string]Value //Inherited property cannot be mutated by the inheritor

	Local map[string]Value

	Inheritable map[string]Value // the value of a property overrides the value stored in any of its predecessor value store
	// map key is the address of the element's  property
	// being watched and elements is the list of elements watching this property
	// Inheritable encompasses overidden values and inherited values that are being passed down.
	Watchers map[string]*Elements
}

func newProperties() Properties {
	return Properties{make(map[string]Value), make(map[string]Value), make(map[string]Value), make(map[string]Value), make(map[string]*Elements)}
}

func (p Properties) NewWatcher(propName string, watcher *Element) {
	list, ok := p.Watchers[propName]
	if !ok {
		p.Watchers[propName] = NewElements(watcher)
		return
	}
	list.Insert(watcher, len(list.List))
}
func (p Properties) RemoveWatcher(propName string, watcher *Element) {
	list, ok := p.Watchers[propName]
	if !ok {
		return
	}
	list.Remove(watcher)
}

func (p Properties) IsWatching(propname string, e *Element) bool {
	list, ok := p.Watchers[propname]
	if !ok {
		return false
	}
	return list.Includes(e)
}

func (p Properties) Get(propName string) (Value, bool) {
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
	return nil, false
}

func (p Properties) Set(propName string, value Value, inheritable bool) {
	if inheritable {
		p.Inheritable[propName] = value
		return
	}
	p.Local[propName] = value
}

func (p Properties) Delete(propname string) {
	delete(p.Local, propname)
}

func (p Properties) Inherit(source Properties) {
	if source.Inheritable != nil {
		for k, v := range source.Inheritable {
			p.Inherited[k] = v
		}
	}
}

// PropertyGroup returns a string denoting whether a property is a default one,
// an inherited one, a local one, or inheritable.
func (p Properties) PropertyGroup(propname string) string {
	_, ok := p.Default[propname]
	if ok {
		return "Default"
	}
	_, ok = p.Inherited[propname]
	if ok {
		return "Inherited"
	}
	_, ok = p.Local[propname]
	if ok {
		return "Local"
	}
	_, ok = p.Inheritable[propname]
	if ok {
		return "Inheritable"
	}
	return ""
}

func (p Properties) SetDefault(propName string, value Value) {
	p.Default[propName] = value
}
