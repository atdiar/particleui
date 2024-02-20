// Package ui is a library of functions for simple, generic gui development.
package ui

import (
	//"encoding/base32"
	"errors"
	"log"
	"math/rand"
	"net/http"
	"strings"
)

var (
	ErrReplayFailure = errors.New("mutation replay failed")
	DEBUG         = log.Println // DEBUG
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



// ElementStore defines a global context, constiting of properties. methods for
// Elemnts constructors of User Interfaces.
type ElementStore struct {
	DocType                  string
	Constructors             map[string]func(id string, optionNames ...string) *Element
	GlobalConstructorOptions map[string]func(*Element) *Element
	ConstructorsOptions      map[string]map[string]func(*Element) *Element

	PersistentStorer map[string]storageFunctions
	RuntimePropTypes map[string]bool

	MutationCapture bool
	MutationReplay bool
	Disconnected bool // true if the go element tree is not connected to its native counterpart
}

type storageFunctions struct {
	Load  func(*Element) error
	Store func(e *Element, category string, propname string, value Value, flags ...bool)
	Clear func(*Element)
}

// ConstructorOption defines a type for optional *Element modifiers that can be applied during
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
			a = NewList(String(name)).Commit()
			e.Set("internals", "constructoroptions", a)
		}
		l, ok := a.(List)
		if !ok {
			log.Print("Unexpected error. constructoroptions should be stored as a ui.List")
			a := NewList(String(name))
			e.Set("internals", "constructoroptions", a.Commit())
		}
		for _, copt := range l.UnsafelyUnwrap() {
			if copt == String(name) {
				return configuratorFn(e)
			}
		}
		e.Set("internals", "constructoroptions", l.MakeCopy().Append(String(name)).Commit())

		return configuratorFn(e)
	}
	return ConstructorOption{name, fn}
}

// NewElementStore creates a new namespace for a list of Element constructors.
func NewElementStore(storeid string, doctype string) *ElementStore {
	es := &ElementStore{doctype, make(map[string]func(id string, optionNames ...string) *Element, 0), make(map[string]func(*Element) *Element), make(map[string]map[string]func(*Element) *Element, 0), make(map[string]storageFunctions, 5), make(map[string]bool,8),false,false, false}
	es.RuntimePropTypes["event"]=true
	es.RuntimePropTypes["navigation"]=true
	es.RuntimePropTypes["runtime"]=true
	es.RuntimePropTypes["ui"]=true

	es.NewConstructor("observable",func(id string)*Element{ // TODO check if this shouldn't be done at the coument level rather
		o:= newObservable(id)
		//o.AsElement().TriggerEvent("mountable")
		//o.AsElement().TriggerEvent("mounted")
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
func (e *ElementStore) EnableMutationCapture() *ElementStore {	e.MutationCapture = true;	return e}


// EnableMutationReplay enables mutation replay of the UI tree. This is used to recover the state corresponding
// to a UI tree that has already been rendered.
func(e *ElementStore) EnableMutationReplay() *ElementStore{	e.MutationReplay = true;	return e}

// NewAppRoot returns the starting point of an app. It is a viewElement whose main
// view name is the root id string.
func (e *ElementStore) NewAppRoot(id string) *Element {
	el := NewElement(id, e.DocType)
	el.registry = make(map[string]*Element,4096)
	el.Root = el
	el.Parent = nil // initially nil DEBUG
	el.subtreeRoot = el
	el.ElementStore = e
	RegisterElement(el,el)
	el.path = &Elements{make([]*Element,0)}
	el.pathvalid = true

	el.Set("internals", "root", Bool(true))
	el.TriggerEvent( "mounted", Bool(true))
	el.TriggerEvent( "mountable", Bool(true))
	el.isroot = true
	
	return el
}

var unregisterHandler =  NewMutationHandler(func(evt MutationEvent)bool{
	unregisterElement(evt.Origin())
	return false
}).RunOnce()

func RegisterElement(approot *Element, e *Element){
	e.Root = approot.AsElement()
	registerElementfn(approot,e)

	e.OnDeleted(unregisterHandler)
}

func registerElementfn(root,e *Element) {
	t:= GetById(root,e.ID)
	if t != nil{
		*e = *t
		return 
	}
	root.registry[e.ID]=e
	e.registry = root.registry
	e.BindValue("internals","documentstate",root)
	e.TriggerEvent("registered")
}

func unregisterElement(e *Element) {
	if e.Root.registry == nil{
		DEBUG("internal err: root element should have an element registry.")
		return
	}

	delete(e.Root.registry,e.ID)
	e.registry = nil

}

// Registered indicates whether an Element is currently registered for any (unspecified) User Interface tree.
func(e *Element) Registered() bool{
	return e.registry != nil
}


// GetByID finds any element that has been part of the UI tree at least once by its ID.
func GetById(root *Element, id string) *Element {
	if root.registry == nil{
		panic("internal err: root element should have an element registry.")
	}

	return root.registry[id]
}

func(e *ElementStore) AddConstructorOptionsTo(elementtype string, options ...ConstructorOption) *ElementStore{
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

		element.WatchEvent("registered",element, NewMutationHandler(func(evt MutationEvent) bool {
			e:= evt.Origin().ElementStore
			// Let's apply the global constructor options
			for _, fn := range e.GlobalConstructorOptions {
				fn(evt.Origin())
			}
			// Let's apply the remaining options
			for _, opt := range optionNames {
				r, ok := e.ConstructorsOptions[elementtype]
				if ok {
					config, ok := r[opt]
					if ok {
						config(evt.Origin())
					} else{
						DEBUG(opt," is not an available option for ",elementtype)
					}
				}
			}
			return false
		}).RunOnce().RunASAP())
		

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
	Root         *Element
	subtreeRoot  *Element // detached if subtree root has no parent unless subtreeroot == root
	path         *Elements
	pathvalid bool

	Parent *Element
	isroot bool

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
	HttpClient *http.Client
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
		NewElements(),
		false,
		nil,
		false,
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
		nil,
	}

	e.subtreeRoot = e
		

	e.OnDeleted(NewMutationHandler(func(evt MutationEvent) bool {
		evt.Origin().PropMutationHandlers.Add(strings.Join([]string{"internals", "deleted"}, "/"), NewMutationHandler(func(event MutationEvent) bool {
			d:= event.Origin()
		
			d.path.free()
			d.Children.free()

			return false
		}).RunOnce())
		return false
	}).RunOnce())
	

	return e
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
	return e.isroot
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
	res := &Elements{StackPool.Get()}
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
		copy(e.List[le:], e.List[:l])
		copy(e.List[:le], elements)
	} else {
		nl := make([]*Element, len(elements)+l, len(elements)+l+64)
		copy(nl, elements)
		copy(nl[le:], e.List[:l])
		e.List = nl
	}
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
		copy(e.List[index:], e.List[index+1:])
        e.List = e.List[:len(e.List)-1]
	}
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

func(e *Elements) free(){
	e.RemoveAll()
	StackPool.Put(e.List)
	e.List = nil
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
		panic("Error: Element path does not exist (yet).")
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
	for _,ancestor:= range e.path.List {
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

func computeSubtreeRoot(e *Element) *Element {
	if e.Parent == nil{
		return e
	}

	if e.Root != nil{
		if e.Root.isroot{
			return e.Root
		}
	}

	return computeSubtreeRoot(e.Parent)
}


// attach will link a child Element to the subtree its target parent belongs to.
// It does not however position it in any view specifically. At this stage,
// the Element can not be rendered as part of the view.
func attach(parent *Element, child *Element) func() {
    finalizers := finalizersPool.Get()
    stack := StackPool.Get()
    stack = append(stack, child)

	mounting:= parent.Mounted()
	mountable := parent.Mountable()

	inactive:= make(map[*Element]bool,64) // TODO use sync.Pool ?


    for len(stack) > 0 {
        lastIndex := len(stack) - 1
        curr := stack[lastIndex]
		stack[lastIndex] = nil
        stack = stack[:lastIndex]
		//DEBUG(curr, child, curr == nil, child == nil)
		if curr.ID == child.ID{
			curr.Parent = parent
		}
		
		/* Should be taken care of elsewhere by making sure that element constructors register
		elemtents on thte root.


		if curr.Root == nil{
			curr.Root = parent.Root
		} else if parent.Root == nil{
			RegisterElement(curr.Root,parent)
		}
		
		if curr.registry == nil{
			curr.WatchEvent("registered",parent, NewMutationHandler(func(evt MutationEvent)bool{
				RegisterElement(evt.Origin().Root,curr)
				return false
			}).RunOnce().RunASAP())

		}
		
		if parent.registry == nil{
			parent.WatchEvent("registered",curr, NewMutationHandler(func(evt MutationEvent)bool{
				RegisterElement(evt.Origin().Root,parent)
				return false
			}).RunOnce().RunASAP())
		}		
        */
		
		// the subtreeroot can change as opposed to the root which is nil until attached to a document root
        curr.subtreeRoot = computeSubtreeRoot(parent)
        curr.link(curr.Parent)
		
		if !inactive[curr] {
            curr.computePath()
        }
		
    
        stack = append(stack, curr.Children.List...)


        for _, descendants := range curr.InactiveViews {
            //stack = append(stack, descendants.Elements().List...)
			for _, element := range descendants.Elements().List {
				stack = append(stack, element)
				inactive[element] = true
			}
        }

        // Generate finalizer for this element
        finalizers = append(finalizers, func() {

			curr.computeRoute()

            if mountable {
				curr.TriggerEvent( "mountable")

				if mounting {
					if curr.Mounted(){
						curr.TriggerEvent("mounted")
					}
				}
			}			
        })

		if mounting {
			if !inactive[curr]{
				curr.TriggerEvent("mount")
			}
		}
    }
   StackPool.Put(stack)

    // Return a single function that calls all the finalizers
    return func() {
        for i, finalizer := range finalizers {
            finalizer()
			finalizers[i] = nil
        }
		finalizers= finalizers[:0]
		finalizersPool.Put(finalizers)
    }
}



// detach will unlink an Element from its parent. If the element was in a view,
// the element is still being rendered until it is removed. However, it should
// not be able to react to events or mutations. TODO review the latter part.
func detach(e *Element) func() {
	if e.Parent == nil {
		panic("attempting to detach an element who is not attached")
	}

	if e.IsRoot() {
		panic("FAILURE: attempt to detach the root element.")
	}

	
	stack := StackPool.Get()
	
	stack = append(stack, e)
	finalizers := finalizersPool.Get()

	wasmounted := e.Mounted()
	inactive:= make(map[*Element]bool,64) 
	

	for len(stack) > 0 {
		// Pop the top element from the stack.
		lastIndex := len(stack) - 1
		curr := stack[lastIndex]
		stack[lastIndex] = nil
		stack = stack[:lastIndex]
	
		// reset the subtree root
		curr.subtreeRoot = nil

	
		if curr.ID == e.ID{
			curr.Parent = nil
			curr.ViewAccessNode.previous = nil // DEBUG not sure it is even necessary
		}
		// Invalidate the path
		curr.pathvalid = false

		
		if !inactive[curr]{
			// Create a finalizer for this element
			finalizer := func() {
				if wasmounted{
					curr.TriggerEvent("unmounted")
				}
			}
			finalizers = append(finalizers, finalizer)
		}
		
		
	
		// Append children of curr onto the stack.
		stack = append(stack, curr.Children.List...)

		// Append descendants of curr onto the stack.
		for _, descendants := range curr.InactiveViews {
            //stack = append(stack, descendants.Elements().List...)
			for _, element := range descendants.Elements().List {
				stack = append(stack, element)
				inactive[element] = true
			}
        }

		if wasmounted{
			if !inactive[curr]{
				curr.TriggerEvent("unmount")
			}
		}
	}
	
	StackPool.Put(stack)
	

	return func() {
		for i, fn := range finalizers {
			fn()
			finalizers[i] = nil
		}
		finalizers = finalizers[:0]
		finalizersPool.Put(finalizers)
	}
}



func (e *Element) calculatePath() {
    // Reset the slice while keeping the underlying array
    oldLength := len(e.path.List)
    e.path.List = e.path.List[:0] 

    // Traverse from e up to the root or the first element with a valid path.
    for current := e; current != nil; current = current.Parent {
        e.path.List = append(e.path.List, current)
    }

    // Clear unused pointers
    for i := len(e.path.List); i < oldLength; i++ {
        e.path.List[i] = nil
    }

    // Reverse e.path.List
    for i, j := 0, len(e.path.List)-1; i < j; i, j = i+1, j-1 {
        e.path.List[i], e.path.List[j] = e.path.List[j], e.path.List[i]
    }
}



func (e *Element) computePath() {
    if e.Parent == nil {
        // If the parent is nil, the element is either root or detached
        if e.IsRoot() {
            // If the element is the root, its path is always valid
            e.pathvalid = true
        } else {
            // If the element is detached, its path is invalid
            e.pathvalid = false
        }
    } else {
        // If the parent is not nil, try to compute its path
        e.Parent.computePath()

        if e.Parent.pathvalid {
            // If the parent's path is valid, calculate and validate the child's path.
            e.calculatePath()
            e.pathvalid = true
        } else {
            // If the parent's path is not valid, the child's path is also invalid.
            e.pathvalid = false
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

	
	
	e.Children.InsertLast(child)
	finalize := attach(e, child)

	if e.Native != nil {
		e.Native.AppendChild(child)
	}
	//child.TriggerEvent( "attached", Bool(true))
	finalize()

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

	

	e.Children.InsertFirst(child)
	finalize:= attach(e, child)

	if e.Native != nil {
		e.Native.PrependChild(child)
	}
	finalize()

	//child.TriggerEvent( "attached", Bool(true))

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

	e.Children.Insert(child, index)
	finalize:= attach(e, child)

	if e.Native != nil {
		e.Native.InsertChild(child, index)
	}

	finalize()

	//child.TriggerEvent( "attached", Bool(true))

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

	if old.Parent == nil || old.Parent.ID != e.ID{
		DEBUG("Can't replace child "+old.ID+" because it's not a child of "+e.ID)
		return e
	}

	if new.Parent != nil {
		new.Parent.removeChild(new)
	}

	finalizeold := detach(old.AsElement())

	e.Children.Replace(old.AsElement(), new.AsElement())
	finalizenew := attach(e, new.AsElement())

	if e.Native != nil {
		e.Native.ReplaceChild(old.AsElement(), new.AsElement())
	}

	//old.TriggerEvent( "attached", Bool(false))
	//new.TriggerEvent( "attached", Bool(true))
	finalizeold()
	finalizenew()

	return e
}

func (e *Element) RemoveChild(childEl AnyElement) *Element {
	return e.removeChild(childEl)
}

func (e *Element) removeChild(childEl AnyElement) *Element {
	if e == nil{
		DEBUG("trying to remove "+childEl.AsElement().ID+ " from a nil element...")
		return e
	}
	child := childEl.AsElement()
	if child.Parent == nil || child.Parent.ID != e.ID{
		return e
	}

	finalize := detach(child)
	e.Children.Remove(child)

	if e.Native != nil {
		e.Native.RemoveChild(child)
	}

	finalize()

	//child.TriggerEvent( "attached", Bool(false))

	return e
}

func (e *Element) RemoveChildren() *Element {
	return e.removeChildren()
}

func (e *Element) removeChildren() *Element {

	for i := len(e.Children.List) - 1; i >= 0; i-- {
        e.removeChild(e.Children.List[i])
    } 
    return e
}



func (e *Element) DeleteChild(childEl AnyElement) *Element {
	if e == nil{
		DEBUG(e.ID + " is nil")
		return e
	}
	child := childEl.AsElement()

	if child.Parent == nil || child.Parent.ID != e.ID{
		return e
	}

	child.TriggerEvent( "deleting", Bool(true))
	child.DeleteChildren()
	
	finalize := detach(child)
	e.Children.Remove(child)

	if e.Native != nil {
		if d,ok:= e.Native.(interface{Delete(*Element)}); ok{
			d.Delete(child)
		} else{
			e.Native.RemoveChild(child)
		}
	}

	finalize()

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
	if e == nil{
		DEBUG(e.ID + " is nil")
		return e
	}

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

			finalize := detach(child)
			if e.Native != nil{
				e.Native.RemoveChild(child)
			}
			
			defer finalize()
			defer child.Set("internals", "deleted", Bool(true))
		}
		e.Children.RemoveAll()
	}
	
	return e
}

var binddeleteahndler = NewMutationHandler(func(evt MutationEvent)bool{
	Delete(evt.Origin())
	return false
}).RunOnce()

func(e *Element) BindDeletion(source *Element) *Element{
	e.WatchEvent("deleted",source, binddeleteahndler)
	return e
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
	if e == nil{return -1, false}
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

/*
func (e *Element) SetChildren(any ...AnyElement) *Element {
	e.SetChildrenElements(convertAny(any...)...)
	return e
}
*/

func (e *Element) SetChildren(any ...*Element) *Element {
	if n, ok := e.Native.(interface{ BatchExecute(string,string) }); ok {
		allChildren := elementmapsPool.Get()
		oldchildrenIdList := childrenIdList(e,allChildren)

		newchildrenIdList :=  stringsPool.Get()
		for _, el := range any {
			if e.DocType != el.DocType {
				log.Printf("Doctypes do not match. Parent has %s while child Element has %s", e.DocType, el.DocType)
				panic("SetChildren failed: wrong doctype")
			}
			allChildren[el.ID] = el
			newchildrenIdList = append(newchildrenIdList, el.ID)
		}

		// calculate myers-diff that generates the edit trace
		editscript := MyersDiff(oldchildrenIdList, newchildrenIdList)
		finalize:= applyEdits(e, editscript, allChildren)

		encEdits := EncodeEditOperations(editscript)

		n.BatchExecute(e.ID, encEdits)
		
		finalize()
		for k:= range allChildren{
			delete(allChildren,k)
		}
		elementmapsPool.Put(allChildren)
		newchildrenIdList = newchildrenIdList[:0]
		stringsPool.Put(newchildrenIdList)
		return e
	}	

	e.RemoveChildren()
	for _, el := range any {
		e.AppendChild(el)
	}

	return e
}

func childrenIdList(e *Element, m map[string]*Element) []string{
	if e == nil{
		return nil
	}
	if e.Children == nil{
		return nil
	}
	list := make([]string,0,len(e.Children.List))
	for _, child := range e.Children.List{
		m[child.ID] = child
		list = append(list,child.ID)
	}
	return list

}

// OnMutation is a convenience emthod that allows for an ELement to watch one of its own properties
// for change.
func(e *Element) OnMutation(category string, propname string, h *MutationHandler) *Element{
	e.Watch(category,propname,e,h)
	return e
}

// BindValue allows for the binding of a property to another element's property.
// When two elements both bind each other's properties, this is ttwo way-binding.
// Indeed, property mutations are idempotent.
// Otherwise, this is one-way binding.
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

	mh,ok:= e.PropMutationHandlers.list[strings.Join([]string{source.ID, category, propname},"/")] 
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

	mh,ok:= e.PropMutationHandlers.list[strings.Join([]string{e.ID,"data",propname},"/")] 
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

	if h.sync{
		h =  syncMutationHandler(e,h)
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

	e.PropMutationHandlers.Add(strings.Join([]string{owner.AsElement().ID, category, propname},"/"), h) 

	eventcat, ok := owner.AsElement().Properties.Categories["internals"]
	if !ok {
		eventcat = newProperties()
		owner.AsElement().Properties.Categories["internals"] = eventcat
	}
	alreadywatching = eventcat.IsWatching("deleted", e)

	if !alreadywatching {
		eventcat.NewWatcher("deleted", e)
	}

	e.PropMutationHandlers.Add(strings.Join([]string{owner.AsElement().ID, "internals", "deleted"},"/"), NewMutationHandler(func(evt MutationEvent) bool {
		if e.ID != owner.AsElement().ID{
			e.Unwatch(category, propname, owner)
		}
		return false
	}))

	if h.ASAP{
		val, ok := owner.AsElement().Properties.Get(category, propname)
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
			evt.Origin().PropMutationHandlers.Remove(strings.Join([]string{owner.AsElement().ID, category, propname},"/"), g)
			return b
		}).RunASAP()
	} else{
		g= NewMutationHandler(func(evt MutationEvent)bool{
			b:= h.Handle(evt)
			evt.Origin().PropMutationHandlers.Remove(strings.Join([]string{owner.AsElement().ID, category, propname},"/"), g)
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
	e.PropMutationHandlers.Remove(strings.Join([]string{owner.AsElement().ID, category, propname},"/"), h)
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
	e.PropMutationHandlers.RemoveAll(strings.Join([]string{owner.AsElement().ID, category, propname},"/"))
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
		evt.Origin().EventHandlers.AddEventHandler(event, handler)
		if nativebinding != nil {
			nativebinding(event, evt.Origin(), handler.Capture)
		}
		return false
	})
	e.OnMounted(h.RunASAP().RunOnce())


	e.OnDeleted(NewMutationHandler(func(evt MutationEvent) bool {
		evt.Origin().RemoveEventListener(event, handler)
		return false
	}))

	return e
}

func SwapNative(e *Element, newNative NativeElement) *Element {
	e.Native = newNative

	nativebinding:= NativeEventBridge
	if l:=e.EventHandlers.list; l != nil{
		for event,handlers:= range l{
			if handlers != nil{
				if handlerlist:= handlers.List; handlerlist != nil{
					for _, handler:= range handlerlist{
						h := NewMutationHandler(func(evt MutationEvent) bool {
							if nativebinding != nil {
								nativebinding(event, evt.Origin(), handler.Capture)
							}
							return false
						})
						e.OnMounted(h.RunASAP().RunOnce())

						e.OnDeleted(NewMutationHandler(func(evt MutationEvent) bool {
							if NativeEventBridge != nil{
								if e.NativeEventUnlisteners.List != nil {
									e.NativeEventUnlisteners.Apply(event)
								}
							}
							return false
						}))
					}
				}
				
			}

		}
	}

	return e
}

// Mountable returns whether the element is attached to the main app tree.
// This includes Mounted Elements, and Elements that are part of an inactive view.
func (e *Element) Mountable() bool {
	if e.IsRoot() {
		return true
	}

	if e.Root == nil {
		return false
	}
	_, isroot := e.Root.Get("internals", "root")
	return isroot
}

// Mounted returns true if an Element is directly reachable from the root of an app tree.
// This does not include elements existing on inactivated view paths.
func (e *Element) Mounted() bool {
	if e.subtreeRoot == nil{
		return false
	}
    return e.subtreeRoot.isroot
}
/* 
	// Essentially equivalent to:
	func (e *Element) Mounted() bool {
		if e.IsRoot() {
			return true
		}

		if e.Parent == nil{
			return false
		}
		return e.Parent.Mounted()
	}

*/

func (e *Element) OnMount(h *MutationHandler) {
	e.WatchEvent("mount", e, h)
}

func (e *Element) OnMounted(h *MutationHandler) {
	e.WatchEvent("mounted", e, h)
}

func (e *Element) OnMountable(h *MutationHandler) {
	e.WatchEvent("mountable", e, h)
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
		evt.Origin().PropMutationHandlers.Remove(strings.Join([]string{evt.Origin().ID, "internals", "deleted"},"/"), g)
		return b
	})


	e.PropMutationHandlers.Add(strings.Join([]string{e.ID, "internals", "deleted"},"/"),g )
}


func(e *Element) TriggerEvent(name string, value ...Value){
	n:= len(value)
	var val Value
	switch {
	case n == 0:
		val = Bool(true)
	case n==1:
		val = value[0]
	default:
		l:= NewList()
		for _,v:= range value{
			l = l.Append(v)
		}
		val = l.Commit()
	}
	e.Set("event",name,val)
}

// WatchEventt enables an elements to watch for an event occuring on any Element including itself.
func(e *Element) WatchEvent(name string, target Watchable, h *MutationHandler){
	e.Watch("event",name,target,h)
}

func(e *Element) GetEventValue(name string) (Value, bool){
	return e.Properties.Get("event",name)
}

// AfterEvent registers a mutation handler that gets called each time an mutation event occurs and
// has been handled.
func (e *Element)  AfterEvent(eventname string, target Watchable, h *MutationHandler){
	if h.ASAP{
		_,ok:= e.GetEventValue(eventname)
		if ok{
			e.WatchEvent(eventname,target,h.RunOnce())
			return
		}
		if h.Once{
			return
		}
	}

	m:= NewMutationHandler(func(evt MutationEvent)bool{
		g:= NewMutationHandler(func(ev MutationEvent)bool{
			return h.Handle(evt)
		}).RunOnce()
		e.WatchEvent(eventname, evt.Origin(), g)
		return false
	})

	if h.Once{
		m = m.RunOnce()
	}
	e.WatchEvent(eventname,target,m)
}

// Get retrieves the value stored for the named property located under the given
// category. The "" category returns the content of the "global" property category.
// The "global" namespace is a local copy of the data that resides in the global
// shared scope common to all Element objects of an ElementStore.
func (e *Element) Get(category, propname string) (Value, bool) {
	return  e.Properties.Get(category, propname)
}

// Set inserts a key/value pair under a given category in the element property store.
// NOTE:some categories that store runtime data are never persisted: e.g. "event" as it corresponds
// to transient, runtime-only props.
func (e *Element) Set(category string, propname string, value Value) {
	if strings.Contains(category, "/") || strings.Contains(propname, "/") {
		panic("category string and/or propname seems to contain a slash. This is not accepted, try a base32 encoding. (" + category + "," + propname + ")")
	}

	oldvalue, ok := e.Properties.Get(category, propname)

	if ok && category != "event" && category != "data"{
		if Equal(value, oldvalue) { // idempotence			
			return
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

	if mutationcapturing(e){
		if category != "data" && category != "ui"{
			return
		}
		if category == "data" && propname == "mutationlist"{ // TODO: make it less broad a condition
			return
		}

		/*
		if category  == "event" {
			return
		}

		if category  == "internals" && propname == "mutation-capturing"{
			return
		}

		if category  == "internals" && propname == "mutation-replaying"{
			return
		}
		*/

		m:= NewObject()
		m.Set("id",String(e.ID))
		m.Set("cat",String(category))
		m.Set("prop",String(propname))
		m.Set("val",Copy(value))
		e.Root.TriggerEvent("new-mutation",m.Commit())
	}
}



func mutationreplaying(e *Element) bool{
	if e == nil{
		return false
	}
	if e.ElementStore == nil{
		return false
	}
	if !e.ElementStore.MutationReplay{
		return false
	}

	if !e.Registered(){
		return false
	}

	v,ok:= e.Root.Get("internals", "mutation-replaying")
	if !ok{
		return false
	}

	return v.(Bool).Bool()
}

func mutationcapturing(e *Element)bool{
	if e == nil{
		return false
	}
	if e.ElementStore == nil{
		return false
	}
	if !e.ElementStore.MutationCapture{
		return false
	}

	if !e.Registered(){
		return false
	}

	v,ok:= e.Root.Get("internals", "mutation-capturing")
	if !ok{
		return false
	}

	return v.(Bool).Bool()
}


func (e *Element) GetData(propname string) (Value, bool) {
	return e.Get("data", propname)
}

// SetData inserts a key/value pair under the "data" category in the element property store.
// It does not automatically update any potential property representation stored
// for rendering use in the "ui" category/namespace.
func (e *Element) SetData(propname string, value Value) *Element{
	e.Set("data", propname, value)
	return e
}

// GetUI returns the value of a property stored under the "ui" category in the element property store.
func(e *Element) GetUI(propname string) (Value, bool) {
	return e.Get("ui", propname)
}

// SetUI inserts a key/value pair under the "ui" category in the element property store.
// A category is synonymous with a namespace.
// A UI property is a a runtime kind of property. As such, it is not persisted.
// A modification of such a property mirrors changes in  the representation of raw data that traditionally
// sits in the "data"category (aka namespace).
//
// SetUI strictly handles UI data as opposed to SetDataSetUI which handles representable business/model data.
func(e *Element) SetUI(propname string, value Value) *Element{
	e.Set("ui", propname, value)
	return e
}

// SyncUI is used to synchronize an event driven UI change in the backend (GUI for instance) with 
// the state of the document tree on the Go side.
// It is typically used in reaction to native events (e.g. toggling a button)
// It does not trigger any mutation event as we are not modifying the UI as it hhs already been modified.
//
// It is different from SyncUISyncData in that it can be used when the data is not owned by the 
// element in charge of its representation. (allows fordecoupling of data ownership and data representation)
// For instance, all the data ;ay be stored in a global observable Element.
// The UI elements would then only be responsible for rendering the data but not storing it.
//
// It also handles multi-parametered representations (e.g. a table may have different filters, a filter
// does not filter the data, it filters the representation of the data and as such, is purely a UI props.
// In that case, we would use SyncUI without setting a data property, wwhich would be a mistake especially
// if it gets persisted somehow.
func(e *Element) SyncUI(propname string, value Value) *Element{
	if strings.Contains(propname, "/") {
		panic("category string and/or propname seems to contain a slash. This is not accepted, try a base32 encoding. (" + propname + ")")
	}
	e.Properties.Set("ui", propname, value)

	
	if mutationcapturing(e){
		if e.Registered(){
			m:= NewObject()
			m.Set("id",String(e.ID))
			m.Set("cat",String("ui"))
			m.Set("prop",String(propname))
			m.Set("val",value)
			m.Set("sync",Bool(true))
			e.Root.TriggerEvent("new-mutation",m.Commit())
		}
	}
	
	return e
}

func(e *Element) SyncData(propname string, value Value) *Element{
	if strings.Contains(propname, "/") {
		panic("category string and/or propname seems to contain a slash. This is not accepted, try a base32 encoding. (" + propname + ")")
	}
	
	oldvalue, _ := e.GetData(propname)
	// Persist property if persistence mode has been set at Element creation
	
	e.Properties.Set("data", propname, value)

	evt := e.newSyncMutationEvent("data", propname, value, oldvalue)

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

	return e
}


// SetDataSetUI will set a "data" property and update the same-name property value
// located in the "ui namespace/category and used to update the the User Interface, for instance, for rendering..
// Typically NOT used when the data is being updated from the UI.
// This is used where Synchronization between the data and its representation are
// needed. (in opposition to automatically updating the UI via data mutation observervation)
func (e *Element) SetDataSetUI(propname string, value Value) {
	oldvalue, _ := e.GetData(propname)

	
	e.Properties.Set("data", propname, value)

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

	if mutationcapturing(e){
		if e.Registered(){
			m:= NewObject()
			m.Set("id",String(e.ID))
			m.Set("cat",String("data"))
			m.Set("prop",String(propname))
			m.Set("val",value)
			e.Root.TriggerEvent("new-mutation",m.Commit())
		}
	} 

	// Update UI representation, it shouldn't have any side-effects on the data so it's safe to do it 
	// before the mutation event is triggered.
	e.Set("ui", propname, value)
}

// SyncUISyncData is used in event handlers when a user changed a value accessible
// via the User Interface, typically.
// It does not trigger mutationahdnler of the "ui" namespace
// (to avoid rerendering an already up-to-date User Interface)
//
//
func (e *Element) SyncUISyncData(propname string, value Value) {

	e.SyncUI(propname,value) 
	e.SyncData(propname, value)
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
	p,ok:= e.Properties.Categories[category]
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
// Indeed, n UI beig event friven, concurrecny of event triggers introduce non-determinism.
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
			if ok{
				view = string(v.(String))
			}
		}
		path := strings.Join([]string{"", n.Element.ID, view},"/")
		if k == 0 {
			if e.Mountable() {
				path = strings.Join([]string{"", view},"/")
			}
		}
		uri = uri + path
	}

	return uri
}

// Route returns the string that represents the URL path that allows for the element to be displayed.
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
		path := strings.Join([]string{"", n.Element.ID, view},"/")
		if k == 0 {
			if e.Mountable() {
				path = strings.Join([]string{"", view},"/")
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
	v,ok:= ps.Get(propname)
	
	return v,ok
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
	return Properties{make(map[string]Value,128), make(map[string]*Elements,64)}
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
		return v, ok
	}
	return nil, false
}

func(p Properties) Watched(propname string) bool{
	_, ok := p.Watchers[propname]
	return ok
}

func (p Properties) Set(propName string, value Value) {
	p.Local[propName] = value
}

func (p Properties) Delete(propname string) {
	delete(p.Local, propname)
}
