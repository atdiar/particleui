// Package ui is a library of functions for simple, generic gui development.
package ui

import (
	//"encoding/base32"
	"errors"
	"log"
	"math/rand"
	"strings"
)

var (
	ErrNoTemplate = errors.New("Element template missing")
	DEBUG         = log.Print // DEBUG
)

// newIDgenerator returns a function used to create new IDs. It uses
// a Pseudo-Random Number Generator (PRNG) as it is desirable to generate deterministic sequences.
// Evidently, as users navigate the app differently and may create new Elements
func newIDgenerator(charlen int, seed int64) func() string {
	source := rand.NewSource(seed)
	r := rand.New(source)
	return func() string {
		var letter = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
		b := make([]rune, charlen)
		for i := range b {
			b[i] = letter[r.Intn(len(letter))]
		}
		return string(b)
	}
}



// ElementStore defines a namespace for a list of Element constructors. // TODO Make immutable
type ElementStore struct {
	ID  string
	DocType                  string
	Constructors             map[string]func(id string, optionNames ...string) *Element
	GlobalConstructorOptions map[string]func(*Element) *Element
	ConstructorsOptions      map[string]map[string]func(*Element) *Element

	PersistentStorer map[string]storageFunctions
	RuntimePropTypes map[string]bool

	MutationCapture bool
	MutationReplay bool

	Seed int64
	IDCharLength int
	genID func() string

	// Registry is a map of all the document roots created from an ElementStore.
	//Registry map[string]*Element
	// TODO  use mutex for concurrent modifications of the registry
}

type storageFunctions struct {
	Load  func(*Element) error
	Store func(e *Element, category string, propname string, value Value, flags ...bool)
	Clear func(*Element)
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
	es := &ElementStore{storeid,doctype, make(map[string]func(id string, optionNames ...string) *Element, 0), make(map[string]func(*Element) *Element), make(map[string]map[string]func(*Element) *Element, 0), make(map[string]storageFunctions, 5), make(map[string]bool,8),false,false,8,21, newIDgenerator(8,21)}
	es.RuntimePropTypes["event"]=true
	es.RuntimePropTypes["navigation"]=true
	es.RuntimePropTypes["runtime"]=true
	es.ApplyGlobalOption(AllowDataFetching)

	es.NewConstructor("observable",func(id string)*Element{ // TODO check if this shouldn't be done at the coument level rather
		o:= newObservable(id)
		o.AsElement().TriggerEvent("mountable")
		o.AsElement().TriggerEvent("mounted")
		return o.AsElement()
	})
	
	return es
}

// AddRuntimePropType allows for the definition of a specific category of *Element properties that can
// not be stored in memory as they are purely a runtime/transient concept (such as "event").
func(e *ElementStore) AddRuntimePropType(name string) *ElementStore{
	e.RuntimePropTypes[name]=true
	return e
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
func (e *ElementStore) AddPersistenceMode(name string, loadFromStore func(*Element) error, store func(*Element, string, string, Value, ...bool), clear func(*Element)) *ElementStore {
	e.PersistentStorer[name] = storageFunctions{loadFromStore, store,clear}
	return e
}


 // EnableMutationCapture enables mutation capture of the UI tree. This is used for debugging, 
 // implementing hot reloading, SSR (server-side-rendering) etc.
 // It basically captures a trace of the program execution that can be replayed later.
func (e *ElementStore) EnableMutationCapture() *ElementStore {	e.MutationCapture = true; e.MutationReplay = false;	return e}


// EnableMutationReplay enables mutation replay of the UI tree. This is used to recover the state corresponding
// to a UI tree that has already been rendered.
func(e *ElementStore) EnableMutationReplay() *ElementStore{	e.MutationReplay = true; e.MutationCapture = false;	return e}

// NewAppRoot returns the starting point of an app. It is a viewElement whose main
// view name is the root id string.
func (e *ElementStore) NewAppRoot(id string) *Element {
	el := NewElement(id, e.DocType)
	el.registry = make(map[string]*Element,4096)
	el.root = el
	el.Parent = el // initially nil DEBUG
	el.subtreeRoot = el
	el.ElementStore = e
	el.Global = NewElement(id+"-globalstate", e.DocType)
	// DEBUG el.path isn't set
	registerElement(el,el)

	el.Set("internals", "root", Bool(true))
	el.TriggerEvent( "mounted", Bool(true))
	el.TriggerEvent( "mountable", Bool(true))

	
	return el
}

func RegisterElement(approot *Element, e *Element){
	e.root = approot.AsElement()
	registerElement(approot,e)


	e.OnDeleted(NewMutationHandler(func(evt MutationEvent)bool{
		unregisterElement(evt.Origin().Root(),evt.Origin())
		return false
	}).RunOnce())
}

func registerElement(root,e *Element) {
	t:= GetById(root,e.ID)
	if t != nil{
		*e = *t
		return 
	}
	root.registry[e.ID]=e
	e.BindValue("internals","documentstate",root)
	e.TriggerEvent("registered")
}

func unregisterElement(root,e *Element) {
	if root.registry == nil{
		DEBUG("internal err: root element should have an element registry.")
		return
	}

	delete(root.registry,e.ID)
}

// Registered indicates whether an Element is currently registered for any (unspecified) User Interface tree.
func(e *Element) Registered() bool{
	_,ok:= e.Get("event","registered")
	return ok
}


// GetByID finds any element that has been part of the UI tree at least once by its ID.
func GetById(root *Element, id string) *Element {
	if root.registry == nil{
		panic("internal err: root element should have an element registry.")
	}

	return root.registry[id]
}

func(e *ElementStore) AddConstructorOptions(elementtype string, options ...ConstructorOption) *ElementStore{
	optlist, ok := e.ConstructorsOptions[elementtype]
	if !ok {
		optlist = make(map[string]func(*Element) *Element)
		e.ConstructorsOptions[elementtype] = optlist
	}

	for _, option := range options {		
		optlist[option.Name] = option.Configurator
	}

	return e
}

// SeedIDgenerator sets a new seed for the NewID method which generates IDs using a PRNG.
func(e *ElementStore) SeedIDgenerator(seed int64) *ElementStore{
	e.Seed = seed
	e.genID = newIDgenerator(e.IDCharLength,seed)
	return e
}

func(e *ElementStore) IDLength(l int) *ElementStore{
	e.IDCharLength = l
	e.genID = newIDgenerator(l,e.Seed)
	return e
}

// NewID returns a  new PRNG generated ID used to provide unique IDs to Elements.
// If tge generation needs to be deterministic over the duration of the APP, don't use this method
// in conditional statements, goroutines, etc.
// It means that dynamically created elements should specify their own IDs instead of relying on 
// anything that uses this. (Element contructors may indirectly expose the usage of this method for instance)
func(e *ElementStore) NewID() string{
	id:= e.genID()
	/*v:= e.GetByID(id)
	if v != nil{
		return e.NewID()
	}*/ 

	// TODO check for id conflicts ?
	return id
}

// NewConstructor registers and returns a new Element construcor function.
func (e *ElementStore) NewConstructor(elementtype string, constructor func(id string) *Element, options ...ConstructorOption) func(id string, optionNames ...string) *Element {

	optlist, ok := e.ConstructorsOptions[elementtype]
	if !ok {
		optlist = make(map[string]func(*Element) *Element)
		e.ConstructorsOptions[elementtype] = optlist
	}
	// First we register the options that are passed with the Constructor definition
	for _, option := range options {		
		optlist[option.Name] = option.Configurator
	}

	// Then we create the element constructor to return
	c := func(id string, optionNames ...string) *Element {
		element := constructor(id)
		element.Set("internals", "constructor", String(elementtype))
		element.ElementStore = e

		// Let's apply the global constructor options
		for _, fn := range e.GlobalConstructorOptions {
			element = fn(element)
		}
		// Let's apply the remaining options
		for _, opt := range optionNames {
			r, ok := e.ConstructorsOptions[elementtype]
			if ok {
				config, ok := r[opt]
				if ok {
					element = config(element)
				} else{
					DEBUG(opt," is not an available option for ",elementtype)
				}
			}
		}

		return element
	}
	e.Constructors[elementtype] = c
	return c
}


func(e *ElementStore) NewObservable(id string, options ...string) Observable{
	c:= e.Constructors["observable"]
	o:= c(id, options...)
	o.ElementStore = e
	return Observable{o}
}

func(e *ElementStore) NewElement(id string) *Element{
	r:= NewElement(id, e.DocType)
	r.ElementStore = e
	return r
}


// Element is the building block of the User Interface. Everything is described
// as an Element having some mutable properties (graphic properties or data properties)
// From the window to the buttons on a page.
// Elements may have a unique parent: hence, Views cannot share any Element.
type Element struct {
	ElementStore *ElementStore
	registry map[string]*Element
	Global       *Element // holds ownership of the global state
	root         *Element
	subtreeRoot  *Element // detached if subtree root has no parent unless subtreeroot == root
	path         *Elements

	Parent *Element

	Alias    string
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

	// these fields are always nil, except for root elements
	// The top level Element is the root node that represents a document: it should control navigation i.e. 
	// document state.
	router *Router
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
func NewElement(id string, doctype string) *Element {
	if strings.Contains(id, "/") {
		panic("An id may not use a slash: " + id + " is not valid.")
	}
	e := &Element{
		nil,
		nil,
		nil,
		nil,
		nil,
		NewElements(),
		nil,
		"",
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
		nil,
	}

	e.OnMountable(NewMutationHandler(func(evt MutationEvent)bool{
		RegisterElement(evt.Origin().Root(),evt.Origin())
		e.TriggerEvent("registered")
		return false
	}).RunOnce())
	
		// fetch support
	e.enablefetching()

	e.OnMounted(NewMutationHandler(func(evt MutationEvent)bool{
		evt.Origin().Fetch()
		return false
	}))

	return e
}

var AllowDataFetching = NewConstructorOption("allowdatafetching", func(e *Element) *Element {
	if e.DocType!=e.ElementStore.DocType{
		return e
	}
	e.enablefetching()
	return e
})



// Root returns the top-most element in the *Element tree.
// All navigation properties are registered on it.
func (e *Element) Root() *Element {
	return e.root
}

// AnyElement is an interface type implemented by *Element :
// Notably BasicElement and ViewElement.
type AnyElement interface {
	AsElement() *Element
}

func (e *Element) IsRoot() bool {
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
	res := &Elements{make([]*Element, 0, 512)}
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
		copy(e.List,elements)
		return e
	}
	nl := make([]*Element,len(elements)+len(e.List),len(elements)+len(e.List)+512 )
	nl = append(nl,elements...)
	nl = append(nl, e.List...)
	e.List = nl
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
	var index int
	nl:= e.List[:0]
	//for _, element := range e.List {
	for i:=0;i<len(e.List);i++{
		element:=e.List[i]
		if element.ID == el.ID {
			continue
		}
		nl=append(nl, element)
		index++
	}
	for i:= index;i<len(e.List);i++{
		e.List[i]=nil
	}
	
	e.List = nl[:index]
	return e
}

func (e *Elements) RemoveAll() *Elements {
	// for k:= range e.List{
	for k:=0;k<len(e.List);k++{
		e.List[k]= nil
	}
	e.List = e.List[:0]
	return e
}

func (e *Elements) Replace(old *Element, new *Element) *Elements {
	for i:=0;i<len(e.List);i++{
		element:=e.List[i]
		if element == old {
			e.List[i] = new
			return e
		}
	}
	return e
}

func (e *Elements) Includes(el *Element) bool {
	for i:=0;i<len(e.List);i++{
		element:=e.List[i]
		if element == nil{
			continue
		}
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

// DispatchEvent triggers an Event on a given target Element.
// If the NativeDispatch variable is nil, the event propagation occurs on the Go side .
// Otherwise, a native platform event is triggered.
//
// Events are propagated following the model set by web browser DOM events:
// 3 phases being the capture phase, at-target and then bubbling up if allowed.
func (e *Element) DispatchEvent(evt Event) bool {
	native:= NativeDispatch
	if !e.Mounted() {
		panic("FAILURE: element notmounted? " + e.ID)
	}

	if native != nil {
		if _,ok:= evt.(DispatchNative);ok{
			native(evt)
			return true
		}
	}

	if e.path == nil {
		log.Print("Error: Element path does not exist (yet).")
		return true
	}

	// First we apply the capturing event handlers PHASE 1
	evt.SetPhase(1)
	var done bool
	for _, ancestor := range e.path.List {
		if evt.Stopped() {
			return true
		}
		evt.SetCurrentTarget(ancestor)
		done = ancestor.Handle(evt) // Handling deemed finished in user side logic
		if done || evt.Stopped() {
			return true
		}
	}

	// Second phase: we handle the events at target
	evt.SetPhase(2)
	evt.SetCurrentTarget(e)
	done = e.Handle(evt)
	if done {
		return true
	}

	// Third phase : bubbling
	if !evt.Bubbles() {
		return true
	}
	evt.SetPhase(3)
	for k := len(e.path.List) - 1; k >= 0; k-- {
		ancestor := e.path.List[k]
		if evt.Stopped() {
			return true
		}
		evt.SetCurrentTarget(ancestor)
		done = ancestor.Handle(evt)
		if done {
			return true
		}
	}
	return done
}



// attach will link a child Element to the subtree its target parent belongs to.
// It does not however position it in any view specifically. At this stage,
// the Element can not be rendered as part of the view.
func attach(parent *Element, child *Element, activeview bool) {
	if activeview {
		child.Parent = parent
		child.path.InsertFirst(parent).InsertFirst(parent.path.List...)
	}
	child.root = parent.root
	child.Global = parent.Global
	child.subtreeRoot = parent.subtreeRoot

	child.link(parent)
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

	//child.TriggerEvent( "attach", Bool(true))
	if child.Mountable() {
		child.TriggerEvent( "mountable", Bool(true))
		// we can set mountable without checking if it was already set because we
		// know that attach is only called for detached subtrees, i.e. they are also
		// not mountable nor mounted.

		if child.Mounted() {
			child.TriggerEvent("mount", Bool(true))
		}
	}
}

func finalize(child *Element, attaching bool, wasmounted bool) {
	if attaching {
		if child.Mounted() {
			child.TriggerEvent("mounted", Bool(true))
		}

	} else { // detaching
		if wasmounted{
			child.TriggerEvent("unmounted",Bool(true))
		}
	}

	for _, descendant := range child.Children.List {
		finalize(descendant, attaching, wasmounted)
	}
	child.computeRoute() // called for its side-effect i.e. computing the ViewAccessPath
}

// detach will unlink an Element from its parent. If the element was in a view,
// the element is still being rendered until it is removed. However, it should
// not be able to react to events or mutations. TODO review the latter part.
func detach(e *Element) {
	if e.Parent == nil {
		return
	}

	if e.IsRoot() {
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

	//e.TriggerEvent( "attach", Bool(false))
	//e.TriggerEvent( "mountable", Bool(false))
	e.TriggerEvent( "unmount", Bool(false)) // i.e. unmount

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
		log.Panicf("Doctypes do not match. Parent has %s while child Element has %s", e.DocType, child.DocType)
		return e
	}
	
	if child.Parent != nil {
		child.Parent.removeChild(child)
	}

	attach(e, child, true)
	e.Children.InsertLast(child)

	if e.Native != nil {
		e.Native.AppendChild(child)
	}
	//child.TriggerEvent( "attached", Bool(true))

	finalize(child, true, e.Mounted())

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
		child.Parent.removeChild(child)
	}

	attach(e, child, true)
	e.Children.InsertFirst(child)

	if e.Native != nil {
		e.Native.PrependChild(child)
	}

	//child.TriggerEvent( "attached", Bool(true))

	finalize(child, true, e.Mounted())

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
		child.Parent.removeChild(child)
	}

	attach(e, child, true)
	e.Children.Insert(child, index)

	if e.Native != nil {
		e.Native.InsertChild(child, index)
	}

	//child.TriggerEvent( "attached", Bool(true))

	finalize(child, true, e.Mounted())

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
		log.Panicf("Doctypes do not match. Parent has %s while child Element has %s", e.DocType, new.DocType)
		return e
	}
	oldwasmounted:= old.Mounted()

	_, ok := e.hasChild(old)
	if !ok {
		return e
	}

	if new.Parent != nil {
		new.Parent.removeChild(new)
	}

	detach(old.AsElement())
	attach(e, new.AsElement(), true)
	e.Children.Replace(old.AsElement(), new.AsElement())

	if e.Native != nil {
		e.Native.ReplaceChild(old.AsElement(), new.AsElement())
	}

	//old.TriggerEvent( "attached", Bool(false))
	//new.TriggerEvent( "attached", Bool(true))

	finalize(old, false, oldwasmounted)
	finalize(new, true, e.Mounted())

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
	wasmounted:= child.Mounted()
	detach(child)
	e.Children.Remove(child)

	if e.Native != nil {
		e.Native.RemoveChild(child)
	}

	//child.TriggerEvent( "attached", Bool(false))
	finalize(child, false,wasmounted)

	return e
}

func (e *Element) RemoveChildren() *Element {
	return e.removeChildren()
}

func (e *Element) removeChildren() *Element {
	/*l := make([]*Element, len(e.Children.List))
	copy(l, e.Children.List)
	for _, child := range l {
		e.removeChild(child)
	}*/
	m:= e.Mounted()
	for _,child:= range e.Children.List{
		detach(child)
		if e.Native != nil{
			e.Native.RemoveChild(child)
		}
		defer finalize(child, false,m)
	}
	e.Children.RemoveAll()

	return e
}

func (e *Element) DeleteChild(childEl AnyElement) *Element {
	child := childEl.AsElement()
	child.TriggerEvent( "deleting", Bool(true))
	child.DeleteChildren()
	e.RemoveChild(childEl)

	if child.isViewElement() {
		for _, view := range child.InactiveViews {
			for _, el := range view.Elements().List {
				el.Set("internals", "deleted", Bool(true))
			}
		}
	}

	child.Set("internals", "deleted", Bool(true))
	
	return e
}

func (e *Element) DeleteChildren() *Element {
	m:= e.Mounted()
	if e.Children != nil{
		for _, child := range e.Children.List {
			child.TriggerEvent( "deleting", Bool(true))
			child.DeleteChildren()
			if child.isViewElement() {
				for _, view := range child.InactiveViews {
					for _, el := range view.Elements().List {
						el.Set("internals", "deleted", Bool(true))
					}
				}
			}

			detach(child)
			if e.Native != nil{
				e.Native.RemoveChild(child)
			}
			defer finalize(child, false,m)
			defer child.Set("internals", "deleted", Bool(true))
		}
		e.Children.RemoveAll()
	}
	
	return e
}

func(e *Element) BindDeletion(source *Element) *Element{
	return e.Watch("event","deleted",source, NewMutationHandler(func(evt MutationEvent)bool{
		Delete(evt.Origin())
		return false
	}).RunASAP().RunOnce())
}

// Delete allows for the deletion of an element regardless of whether it has a parent.
func Delete(e *Element){
	if e.Parent!=nil{
		e.Parent.DeleteChild(e)
		return
	}
	e.TriggerEvent("deleting",Bool(true))
	e.DeleteChildren()

	if e.isViewElement() {
		for _, view := range e.InactiveViews {
			for _, el := range view.Elements().List {
				el.Set("internals", "deleted", Bool(true))				
			}
			view.Elements().RemoveAll()
		}
	}

	e.Set("internals", "deleted", Bool(true))
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
				ele.Parent.removeChild(ele)
			}
			attach(e, ele, true)
			e.Children.InsertLast(ele)
			children = append(children, ele)
		}
		n.SetChildren(children...)
		for _, child := range children {
			//child.TriggerEvent( "attached", Bool(true))
			finalize(child, true,false)
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
	m:= e.Mounted()
	
	e.RemoveChildren()
	if n, ok := e.Native.(interface{ SetChildren(...*Element) }); ok {
		for _, el := range any {
			if e.DocType != el.DocType {
				log.Printf("Doctypes do not match. Parent has %s while child Element has %s", e.DocType, el.DocType)
				panic("SetChildren failed: wrong doctype")
			}
			if el.Parent != nil {
				el.Parent.removeChild(el)
			}

			attach(e, el, true)
			e.Children.InsertLast(el)
		}
		
		for _, child := range any {
			//child.TriggerEvent( "attached", Bool(true))
			finalize(child, true,m)
		}
		
		n.SetChildren(any...)
		return e
	} else{
		for _, el := range any {
			e.AppendChild(el)
			// el.ActiveView = e.ActiveView // TODO verify this is correct
		}
	}
	
	return e
}

// OnMutation allows for the registration of a MutationHandler to be called when a property is mutated.
func(e *Element) OnMutation(category string, propname string, h *MutationHandler) *Element{
	e.Watch(category,propname,e,h)
	return e
}

// BindValue allows for the binding of a property to another element's property.
// This only allows for one way binding which is sufficient for all purposes and has the benefit
// of not creating a circular dependency and allowing Elements to be independently designed.
func(e *Element) BindValue(category string, propname string, source *Element) *Element{
	if source == nil{
		panic("unable to bind to a nil *Element")
	}
	if source.ID == e.ID{
		return e
	}

	if e.bound(category,propname,source){
		return e
	}

	e.Watch(category,propname,source,NewMutationHandler(func(evt MutationEvent) bool{
		e.Set(category,propname,evt.NewValue())
		return false
	}).RunASAP().binder())
	return e
}

func(e *Element) bound(category string, propname string, source *Element) bool{
	p, ok := source.Properties.Categories[category]
	if !ok{
		return false
	}

	if !p.IsWatching(propname,e){
		return false
	}

	if e.PropMutationHandlers.list == nil{
		return false
	}

	mh,ok:= e.PropMutationHandlers.list[source.ID+"/"+category+"/"+propname]
	if !ok{
		return false
	}

	for i:=0;i<len(mh.list);i++{
		h:=mh.list[i]
		if h.binding{
			return true
		}
	}

	return false
}

func(e *Element) fetching(propname string) bool{
	p, ok := e.Properties.Categories["data"]
	if !ok{
		return false
	}

	if !p.IsWatching(propname,e){
		return false
	}

	if e.PropMutationHandlers.list == nil{
		return false
	}

	mh,ok:= e.PropMutationHandlers.list[e.ID+"/"+"data"+"/"+propname]
	if !ok{
		return false
	}

	for i:=0;i<len(mh.list);i++{
		h:=mh.list[i]
		if h.fetching{
			return true
		}
	}

	return false
}

// Watch allows to observe a property of an element. Properties are classified into categories
// (aka namespaces). As soon as the property changes, the mutation handler is executed.
// a *MutationHandler is sinmply a wrapper around a function that handles the MutationEvent triggered
// when setting(mutationg) a property.
func (e *Element) Watch(category string, propname string, owner Watchable, h *MutationHandler) *Element {
	if owner.AsElement() == nil{
		panic("unable to watch element properties as it is nil")
	}
	if h.Once{
		return e.watchOnce(category,propname,owner,h)
	}

	p, ok := owner.AsElement().Properties.Categories[category]
	if !ok {
		p = newProperties()
		owner.AsElement().Properties.Categories[category] = p
	}
	if p.Watchers == nil{
		DEBUG("unexpected nil watchers")
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
		if e.ID != owner.AsElement().ID{
			e.Unwatch(category, propname, owner)
		}
		return false
	}))

	if h.ASAP{
		val, ok := owner.AsElement().Get(category, propname)
		if ok {
			h.Handle(owner.AsElement().NewMutationEvent(category, propname, val, nil))
		}
	}

	return e
}


// watchOnce allows to have a mutation handler that runs only once for the occurence of a mutation.
// Important note; it does not necessarily run for the first mutation. The property change tracking
// might have been added late, after a few mutations had already occured.

func(e *Element) watchOnce(category string, propname string, owner Watchable, h *MutationHandler) *Element{
	var g *MutationHandler
	if h.ASAP{
		g= NewMutationHandler(func(evt MutationEvent)bool{
			b:= h.Handle(evt)
			evt.Origin().PropMutationHandlers.Remove(owner.AsElement().ID+"/"+category+"/"+propname, g)
			return b
		}).RunASAP()
	} else{
		g= NewMutationHandler(func(evt MutationEvent)bool{
			b:= h.Handle(evt)
			evt.Origin().PropMutationHandlers.Remove(owner.AsElement().ID+"/"+category+"/"+propname, g)
			return b
		})
	}
	
	return e.Watch(category,propname,owner,g)
}

// removeHandler allows for the removal of a Mutation Handler.
// Can be used to clean up, for instance in the case of 
func (e *Element) RemoveMutationHandler(category string, propname string, owner Watchable, h *MutationHandler) *Element {
	_, ok := owner.AsElement().Properties.Categories[category]
	if !ok {
		return e
	}
	e.PropMutationHandlers.Remove(owner.AsElement().ID+"/"+category+"/"+propname, h)
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


func (e *Element) RemoveEventListener(event string, handler *EventHandler) *Element {
	e.EventHandlers.RemoveEventHandler(event, handler)
	if NativeEventBridge != nil{
		if e.NativeEventUnlisteners.List != nil {
			e.NativeEventUnlisteners.Apply(event)
		}
	}
	return e
}

// AddEventListener registers a function to be run each time a given event occurs on an element.
// Once the Go-defined event handler runs, event propagation stops on the native side. It is picked up 
// on the Go side however(the event propagates in the UI tree)
//
// As such, event delegation, which relies on event propagation by capture or bubbling, does not 
// require to listen to a native side event.(NativeEventBridge can be nil in that case). At target, 
// the native event will have been transformed into a pure Go Event.
func (e *Element) AddEventListener(event string, handler *EventHandler) *Element {
	nativebinding:= NativeEventBridge
	h := NewMutationHandler(func(evt MutationEvent) bool {
		e.EventHandlers.AddEventHandler(event, handler)
		if nativebinding != nil {
			nativebinding(event, e, handler.Capture)
		}
		return false
	})
	e.OnMounted(h.RunASAP().RunOnce())


	e.OnDeleted(NewMutationHandler(func(evt MutationEvent) bool {
		e.RemoveEventListener(event, handler)
		return false
	}))

	return e
}

// Mountable returns whether the element is attached to the main app tree.
// This includes Mounted Elements, and Elements that are part of an inactive view.
func (e *Element) Mountable() bool {
	if e.IsRoot() {
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
	if e.IsRoot() {
		return true
	}

	if !e.Mountable() {
		return false
	}
	l := e.path.List
	if len(l) == 0 {
		return false
	}
	return l[0].IsRoot()
}

func (e *Element) OnMount(h *MutationHandler) {
	e.WatchEvent("mount", e, h)
}

func (e *Element) OnMounted(h *MutationHandler) {
	e.WatchEvent("mounted", e, h)
}

func (e *Element) OnMountable(h *MutationHandler) {
	e.Watch("event","mountable", e, h)
}

func(e *Element) OnRegistered(h *MutationHandler){
	e.WatchEvent("registered", e, h.RunASAP().RunOnce())
}


// OnUnmount can be used to make a change right before an element starts unmounting.
// One potential use case is to deal with animations as an elemnt disappear from the page.
func (e *Element) OnUnmount(h *MutationHandler) {
	e.WatchEvent("unmount", e, h)
}

func (e *Element) OnUnmounted(h *MutationHandler) {
	e.WatchEvent("unmounted", e, h)
}


// TODO make behaviour similar to running AsAP and Once.
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

	val, ok := e.Get("internals", "deleted")
	if ok {
		h.Handle(e.NewMutationEvent("internals", "deleted", val, nil))
		return
	}
	var g *MutationHandler
	g= NewMutationHandler(func(evt MutationEvent)bool{
		e.CancelAllTransitions()
		b:= h.Handle(evt)
		evt.Origin().PropMutationHandlers.Remove(evt.Origin().ID+"/"+"internals"+"/"+"deleted", g)
		return b
	})


	e.PropMutationHandlers.Add(e.ID+"/"+"internals"+"/"+"deleted",g )
}

func isRuntimeCategory(e *ElementStore, category string) bool{
	_,ok:= e.RuntimePropTypes[category]
	return ok
}

func newEventValue(v Value) Object{
	o:= NewObject()
	o.Set("value",v)
	o.Set("id",String(newEventID()))
	// todo add uuid value sincve events are unique

	return o
}

func(e *Element) TriggerEvent(name string, value ...Value){
	n:= len(value)
	switch {
	case n == 0:
		e.Set("event", name,newEventValue(Bool(true)))
	case n==1:
		e.Set("event", name,newEventValue(value[0]))
	default:
		e.Set("event", name,newEventValue(NewList(value...)))
	}
}

// WatchEventt enables an elements to watch for an event occuring on any Element including itself.
func(e *Element) WatchEvent(name string, target Watchable, h *MutationHandler){
	e.Watch("event",name,target,h)
}

/*
// Canonicalize returns the base32 encoding of a string if it contains the delimmiter "/"
// It can be used 
func Canonicalize(s string) string{
	if strings.Contains(s,"/"){
		return base32.StdEncoding.EncodeToString([]byte(s))
	}
	return s
}
*/

// Get retrieves the value stored for the named property located under the given
// category. The "" category returns the content of the "global" property category.
// The "global" namespace is a local copy of the data that resides in the global
// shared scope common to all Element objects of an ElementStore.
func (e *Element) Get(category, propname string) (Value, bool) {
	return e.Properties.Get(category, propname)
}

// Set inserts a key/value pair under a given category in the element property store.
// NOTE:some categories that store runtime data are never persisted: e.g. "event" as it corresponds
// to transient, runtime-only props.
func (e *Element) Set(category string, propname string, value Value) {
	if strings.Contains(category, "/") || strings.Contains(propname, "/") {
		panic("category string and/or propname seems to contain a slash. This is not accepted, try a base32 encoding. (" + category + "," + propname + ")")
	}


	// Persist property if persistence mode has been set at Element creation
	pmode := PersistenceMode(e)

	oldvalue, ok := e.Get(category, propname)

	if ok {
		if Equal(value, oldvalue) { // idempotence		TODO nake sure that deepequality is optimized	
			return
		}
	}

	if e.ElementStore != nil {
		storage, ok := e.ElementStore.PersistentStorer[pmode]
		if ok && !isRuntimeCategory(e.ElementStore,category) {
			storage.Store(e, category, propname, value)
		}
	}

	e.Properties.Set(category, propname, value)

	// Mutation event propagation
	evt := e.NewMutationEvent(category, propname, value, oldvalue)

	props, ok := e.Properties.Categories[category]
	if !ok {
		panic("category should exist since property should have been stored")
	}
	watchers, ok := props.Watchers[propname]
	if ok && watchers != nil{
		var needcleanup bool
		var index int
		wl:= watchers.List[:0]

		for i:=0; i<len(watchers.List);i++{
			w:= watchers.List[i]
			if w == nil{
				if !needcleanup{
					wl = watchers.List[:i]
					index = i+1
					needcleanup = true
				}
				continue
			}
			w.PropMutationHandlers.DispatchEvent(evt)
			if needcleanup{
				wl = append(wl,w)
				index++
			}
		}
		if needcleanup{
			for i:= index; i<len(watchers.List);i++{
				watchers.List[i] = nil
			}
			watchers.List = wl[:index]
		}
	}

	if e.ElementStore!= nil && e.ElementStore.MutationCapture{
		if e.Registered(){
			m:= NewObject()
			m.Set("id",String(e.ID))
			m.Set("cat",String(category))
			m.Set("prop",String(propname))
			m.Set("val",Copy(value))
			l,ok:= e.Get("internals","mutationtrace")
			if !ok{
				l=NewList(m)
				e.Root().Set("internals","mutationtrace",l)
			} else{
				list:= l.(List)
				list = append(list,m)
				e.Root().Set("internals","mutationtrace",list)
			}
		}
	}

}

func mutationReplay(root *Element){
	l,ok:= root.Get("internals","mutationtrace")
	if ok{
		list:= l.(List)
		for _,m:= range list{
			obj:= m.(Object)

			id,ok:= obj.Get("id")
			if !ok{
				panic("mutation record should have an id")
			}

			cat,ok:= obj.Get("cat")
			if !ok{
				panic("mutation record should have a category")
			}

			prop,ok := obj.Get("prop")
			if !ok{
				panic("mutation record should have a property")
			}

			val,ok:= obj.Get("val")
			if !ok{
				panic("mutation record should have a value")
			}

			if category:=cat.(String).String(); category == "ui"{
				_,ok= obj.Get("sync")
				if ok{
					e:= GetById(root,id.(String).String())
					if e==nil{
						panic("FWERR: element " + id.(String).String() + " does not exist. Unable to recover Pre-rendered state")	
					}
					LoadProperty(e,"ui",prop.(String).String(),val)
				}

				e:= GetById(root,id.(String).String())
				if e==nil{
					panic("FWERR: element " + id.(String).String() + " does not exist. Unable to recover Pre-rendered state")	
				}
				e.Set("ui",prop.(String).String(),val)
			} else{
				e:= GetById(root,id.(String).String())
					if e==nil{
						panic("FWERR: element " + id.(String).String() + " does not exist. Unable to recover Pre-rendered state")	
					}
					LoadProperty(e,category,prop.(String).String(),val)
			}	
		}
		root.Set("internals","mutationtrace",nil)
		root.TriggerEvent("documentstaterecovered")
	}
}


func (e *Element) GetData(propname string) (Value, bool) {
	return e.Get("data", propname)
}

// SetData inserts a key/value pair under the "data" category in the element property store.
// It does not automatically update any potential property representation stored
// for rendering use in the "ui" category/namespace.
func (e *Element) SetData(propname string, value Value) {
	e.Set("data", propname, value)
}

// SetDataSetUI will set a "data" property and update the same-name property value
// located in the "ui namespace/category and used to update the the User Interface, for instance, for rendering..
// Typically NOT used when the data is being updated from the UI.
// This is used where Synchronization between the data and its representation are
// needed. (in opposition to automatically updating the UI via data mutation observervation)
func (e *Element) SetDataSetUI(propname string, value Value) {
	oldvalue, _ := e.GetData(propname)
	// Persist property if persistence mode has been set at Element creation
	pmode := PersistenceMode(e)

	if e.ElementStore != nil {
		storage, ok := e.ElementStore.PersistentStorer[pmode]
		if ok {
			storage.Store(e, "data", propname, value)
		}
	}
	e.Properties.Set("data", propname, value)

	// e.Set("ui", propname, value, flags...) // Initial position but let's put that after data change propagation DEBUG

	evt := e.NewMutationEvent("data", propname, value, oldvalue)

	props, ok := e.Properties.Categories["data"]
	if !ok {
		panic("category should exist since property should have been stored")
	}
	watchers, ok := props.Watchers[propname]
	if ok && watchers != nil {
		for i:= 0; i<len(watchers.List);i++{
			w := watchers.List[i]
			w.PropMutationHandlers.DispatchEvent(evt)
		}
	}

	e.Set("ui", propname, value)
}

// SyncUISetData is used in event handlers when a user changed a value accessible
// via the User Interface, typically.
// It does not trigger mutationahdnler of the "ui" namespace
// (to avoid rerendering an already up-to-date User Interface)
//
//
func (e *Element) SyncUISetData(propname string, value Value) {

	// Persist property if persistence mode has been set at Element creation
	pmode := PersistenceMode(e)


	e.Properties.Set("ui", propname, value)

	if e.ElementStore != nil {
		storage, ok := e.ElementStore.PersistentStorer[pmode]
		if ok {
			storage.Store(e, "ui", propname, value)
		}
	}
	if e.ElementStore.MutationCapture{
		if e.Registered(){
			m:= NewObject()
			m.Set("id",String(e.ID))
			m.Set("cat",String("ui"))
			m.Set("prop",String(propname))
			m.Set("val",Copy(value))
			m.Set("sync",Bool(true))
			l,ok:= e.Get("internals","mutationtrace")
			if !ok{
				l=NewList(m)
				e.Root().Set("internals","mutationtrace",l)
			} else{
				list:= l.(List)
				list = append(list,m)
				e.Root().Set("internals","mutationtrace",list)
			}
		}
	}

	e.Set("data", propname, value)
}

// LoadProperty is a function typically used to restore a UI Element properties.
// It does not trigger mutation events. 
// For properties of the 'ui' namespace, i.e. properties that are used for rendering,
// we create and dispatch a mutation event since loading a property modifies the UI.
func LoadProperty(e *Element, category string, propname string, value Value) {
	e.Properties.Load(category, propname, value)
}

// Rerender basically refires mutation events of the "ui" property namespace, effectively triggering
// a re-render of the User Interface.
// It works because the UI drawn on screen is built from idempotent functions: the "ui" mutation 
// handlers.
// ATTENTION: in fact the UI should be a "pure" function of the "ui" properties. Which means that 
// rendering of an element should not have side-effects, notably should not mutate another element's UI.
func Rerender(e *Element) *Element{
	category:= "ui"
	p,ok:= e.Properties.Categories["ui"]
	if !ok{
		return e
	}
	propset := make(map[string]struct{},256)

	for prop,value:= range p.Local{
		if _,exist:= propset[prop];!exist{
			propset[prop]=struct{}{}
			evt := e.NewMutationEvent(category, prop, value, nil)
			e.PropMutationHandlers.DispatchEvent(evt)
		}	
	}
	
	return e
}


// computeRoute returns the path to an Element.
//
// This path may be parameterized if the element is ocntained by an unmounted parametered view.
//
// Important notice: views that are nested within a fixed element use that Element ID for routing.
// In effect, the id acts as a namespace.
// In order for links using the routes to these views to not be breaking between refresh/reruns of an app (hard requirement for online link-sharing), the ID of the parent element
// should be generated so as to not change. Using a PRNG-based ID generator is very unlikely to be a good-fit here.
//
// For instance, if we were to create a dynamic view composed of retrieved tweets, we would not use an ID generator but probably reuse the tweet ID gotten via http call for each Element.
// Building a shareable link toward any of these elements still require that every ID generated in the path is stable across app refresh/re-runs.
func (e *Element) computeRoute() string {
	var uri string
	e.ViewAccessPath = computePath(newViewNodes(), e.ViewAccessNode)
	if e.ViewAccessPath == nil || len(e.ViewAccessPath.Nodes) == 0 {
		return uri
	}

	for k, n := range e.ViewAccessPath.Nodes {
		view := n.Name
		if n.Element.Mounted(){
			v,ok:= n.Element.Get("ui","activeview")
			if!ok{
				panic("couldn't find current view name")
			}
			view = string(v.(String))
		}
		path := "/" + n.Element.ID + "/" + view
		if k == 0 {
			if e.Mountable() {
				path = "/" + view
			}
		}
		uri = uri + path
	}
	return uri
}

// Route returns the string that reporesents the URL path that allows for the elemnt to be displayed.
// This string may be parameteriwed if the element is contained in an unmounted parametered view.
// if the element is not mountable, an empty string is returned.
func(e *Element) Route() string{
	var uri string
	if e.ViewAccessPath == nil || len(e.ViewAccessPath.Nodes) == 0 {
		return uri
	}

	for k, n := range e.ViewAccessPath.Nodes {
		view := n.Name
		if n.Element.Mounted(){
			v,ok:= n.Element.Get("ui","activeview")
			if!ok{
				panic("couldn't find current view name while generating route string")
			}
			view = string(v.(String))
		}
		path := "/" + n.Element.ID + "/" + view
		if k == 0 {
			if e.Mountable() {
				path = "/" + view
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
	return PropertyStore{make(map[string]Properties,16)}
}

func (p PropertyStore) Load(category string, propname string, value Value) {
	ps, ok := p.Categories[category]
	if !ok {
		ps = newProperties()
		p.Categories[category] = ps
	}
	ps.Local[propname] = value
}

func(p PropertyStore) NewWatcher(category string, propname string, watcher *Element){
	ps, ok := p.Categories[category]
	if !ok {
		ps = newProperties()
		p.Categories[category] = ps
	}
	ps.NewWatcher(propname,watcher)
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

func (p PropertyStore) Set(category string, propname string, value Value) {
	ps, ok := p.Categories[category]
	if !ok {
		ps = newProperties()
		p.Categories[category] = ps
	}
	ps.Set(propname, value)
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



type Properties struct {
	Local map[string]Value
	Watchers map[string]*Elements
}

func newProperties() Properties {
	return Properties{make(map[string]Value), make(map[string]*Elements)}
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
	
	for i:=0;i<len(list.List);i++{
		w:=list.List[i]
		if w == nil{
			continue
		}
		if  watcher.ID == w.ID{
			list.List[i]= nil
		}
	}
}

func (p Properties) IsWatching(propname string, e *Element) bool {
	list, ok := p.Watchers[propname]
	if !ok {
		return false
	}
	return list.Includes(e)
}

// Get returns a copy of the value stored for a given property.
func (p Properties) Get(propName string) (Value, bool) {
	v, ok := p.Local[propName]
	if ok {
		return Copy(v), ok
	}
	return nil, false
}

func (p Properties) Set(propName string, value Value) {
	p.Local[propName] = Copy(value)
}

func (p Properties) Delete(propname string) {
	delete(p.Local, propname)
}
