// Package ui is a library of functions for simple, generic gui development.
package ui

import (
	"errors"
	"log"
	"math/rand"
	"strings"
	"time"
	"encoding/base64"
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
		str:= base64.RawStdEncoding.EncodeToString(bstr)
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

func (e *ElementStore) NewAppRoot(id string) *Element {
	el := NewElement("root", id, e.DocType)
	el.root = el
	el.subtreeRoot = el
	el.ElementStore = e
	el.Global = e.Global
	// DEBUG el.path isn't set
	return el
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

// Element is the building block of the User Interface. Everything is described
// as an Element having some mutable properties (graphic properties or data properties)
// From the window to the buttons on a page.
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

	AlternateViews map[string]ViewElements // this is a  store for  named views: alternative to the Children field, used for instance to implement routes/ conditional rendering.

	Native NativeElement
}

func (e *Element) Element() *Element { return e }

// NewElement returns a new Element with no properties, no event or mutation handlers.
// Essentially an empty shell to be customized.
func NewElement(name string, id string, doctype string) *Element {
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
		nil,
		nil,
		nil,
	}
	e.Watch("ui", "command", e, DefaultCommandHandler)
	return e
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
	if nativebinding != nil {
		nativebinding(evt)
		return e
	}

	if !e.Mounted() {
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

// AddView adds a named list of children Elements to an Element creating as such
// a named version of the internal state of an Element.
// In a sense, it enables conditional rendering for an Element by allowing to
// switch between the different named internal states.
func (e *Element) AddView(v ViewElements) *Element {
	for _, child := range v.Elements().List {
		child.ViewAccessPath = child.ViewAccessPath.Prepend(newViewNode(e, v)).Prepend(e.ViewAccessPath.nodes...)
		attach(e, child, false)
	}

	e.AlternateViews[v.Name()] = v
	return e
}

// DeleteView  deletes any view that exists for the current Element but is not
// displayed.
func (e *Element) DeleteView(name string) *Element {
	v, ok := e.AlternateViews[name]
	if !ok {
		return e
	}
	for _, el := range v.Elements().List {
		detach(el)
	}
	delete(e.AlternateViews, name)
	return e
}

// RetrieveView will return a pointer to a non-displayed view for the current element.
// If the named ViewElements does not exist, nil is returned.
func (e *Element) RetrieveView(name string) *ViewElements {
	v, ok := e.AlternateViews[name]
	if !ok {
		return nil
	}
	return &v
}

// ActivateVIew is used to render the desired named view for a given Element.
func (e *Element) ActivateView(name string) error {
	newview, ok := e.AlternateViews[name]
	if !ok {
		// Support for parameterized views
		if len(e.AlternateViews) != 0 {
			var view ViewElements
			var parameterName string
			for k, v := range e.AlternateViews {
				if strings.HasPrefix(k, ":") {
					parameterName = k
					view = v
					break
				}
			}
			if parameterName != "" {
				if len(parameterName) == 1 {
					return errors.New("Bad view name parameter. Needs to be longer than 0 character.")
				}
				// Now that we have found a matching parameterized view, let's try to retrieve the actual
				// view corresponding to the submitted value "name"
				v, err := view.ApplyParameter(name)
				if err != nil {
					// This parameter does not seem to be accepted.
					return err
				}
				view = *v

				// Let's detach the former view items
				oldview, ok := e.Get("ui", "activeview")
				oldviewname, ok2 := oldview.(String)
				viewIsParameterized := (string(oldviewname) != e.ActiveView)
				if ok && ok2 && oldviewname != "" && e.Children != nil {
					for _, child := range e.Children.List {
						detach(child)
						if !viewIsParameterized {
							attach(e, child, false)
						}
					}
					if !viewIsParameterized {
						// the view is not parameterized
						e.AlternateViews[string(oldviewname)] = NewViewElements(string(oldviewname), e.Children.List...)
					}
				}

				// Let's append the new view Elements
				for _, newchild := range view.Elements().List {
					e.AppendChild(newchild)
				}
				e.Set("ui", "activeview", String(name), false)
				e.ActiveView = parameterName
				return nil
			}
		}
		return errors.New("View does not exist.")
	}

	// first we detach the current active View and reattach it as an alternative View if non-parameterized
	oldview, ok := e.Get("ui", "activeview")
	oldviewname, ok2 := oldview.(String)
	viewIsParameterized := (string(oldviewname) != e.ActiveView)
	if ok && ok2 && oldviewname != "" && e.Children != nil {
		for _, child := range e.Children.List {
			detach(child)
			if !viewIsParameterized {
				attach(e, child, false)
			}
		}
		if !viewIsParameterized {
			// the view is not parameterized
			e.AlternateViews[string(oldviewname)] = NewViewElements(string(oldviewname), e.Children.List...)
		}
	}
	// we attach and activate the desired view
	for _, child := range newview.Elements().List {
		e.AppendChild(child)
	}
	delete(e.AlternateViews, name)
	e.Set("ui", "activeview", String(name), false)
	e.ActiveView = name

	return nil
}

// attach will link a child Element to the subtree its target parent belongs to.
// It does not however position it in any view specifically. At this stage,
// the Element can not be rendered as part of the view.
func attach(parent *Element, child *Element, activeview bool) {
	defer func() {
		child.Set("event", "attached", Bool(true), false)
		if child.Mounted() {
			child.Set("event", "mounted", Bool(true), false)
		}
	}()
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
		for _, descendant := range descendants.Elements().List {
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

	e.Set("event", "attached", Bool(false))
	e.Set("event", "mounted", Bool(false))

	// got to update the subtree with the new subtree root and path
	for _, descendant := range e.Children.List {
		attach(e, descendant, true)
	}

	for _, descendants := range e.AlternateViews {
		for _, descendant := range descendants.Elements().List {
			attach(e, descendant, false)
		}
	}
}

// AppendChild appends a new element to the element's children list for the active
// view being rendered.
func (e *Element) AppendChild(child *Element) *Element {
	log.Print(child) // DEBUG
	if e.DocType != child.DocType {
		log.Printf("Doctypes do not match. Parent has %s while child Element has %s", e.DocType, child.DocType)
		return e
	}
	if child.Parent != nil {
		child.Parent.RemoveChild(child)
	}

	attach(e, child, true)

	e.Children.InsertLast(child)
	if e.Native != nil {
		e.Native.AppendChild(child)
	}
	return e
}

func (e *Element) PrependChild(child *Element) *Element {
	if e.DocType != child.DocType {
		log.Printf("Doctypes do not match. Parent has %s while child Element has %s", e.DocType, child.DocType)
		return e
	}

	if child.Parent != nil {
		child.Parent.RemoveChild(child)
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
		log.Printf("Doctypes do not match. Parent has %s while child Element has %s", e.DocType, child.DocType)
		return e
	}
	if child.Parent != nil {
		child.Parent.RemoveChild(child)
	}

	attach(e, child, true)

	e.Children.Insert(child, index)
	if e.Native != nil {
		e.Native.InsertChild(child, index)
	}
	return e
}

// ReplaceChild will replace the target child Element with another.
// Be wary that mutation Watchers and event listeners remain unchanged by default.
// The addition or removal of change observing obk=jects is left at the discretion
// of the user.
func (e *Element) ReplaceChild(old *Element, new *Element) *Element {
	if e.DocType != new.DocType {
		log.Printf("Doctypes do not match. Parent has %s while child Element has %s", e.DocType, new.DocType)
		return e
	}
	if new.Parent != nil {
		new.Parent.RemoveChild(new)
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
	e.Children.Remove(child)

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

// Command defines a type used to represent a UI mutation request.
//
// These commands can be logged in an append-only manner so that they are replayable
// in the order they were registered to recover UI state.
//
// In order to register a command, one just needs to Set the "command" property
// of the "ui" namespace of an Element.
//  Element.Set("ui","command",Command{...})
// As such, the Command type implements the Value interface.
type Command Object

func (c Command) discriminant() discriminant { return "particleui" }
func (c Command) ValueType() string          { return Object(c).ValueType() }
func (c Command) RawValue() Object           { return Object(c).RawValue() }

func (c Command) Name(s string) Command {
	Object(c).Set("name", String(s))
	return c
}

func (c Command) SourceID(s string) Command {
	log.Print("source: ",s) // DEBUG
	Object(c).Set("sourceid", String(s))
	return c
}

func (c Command) TargetID(s string) Command {
	Object(c).Set("targetid", String(s))
	return c
}

func (c Command) Position(p int) Command {
	Object(c).Set("position", Number(p))
	return c
}

func (c Command) Timestamp(t time.Time) Command {
	Object(c).Set("timestamp", String(t.String()))
	return c
}

func NewUICommand() Command {
	c := Command(NewObject().SetType("Command"))
	return c.Timestamp(time.Now().UTC())
}

func AppendChildCommand(child *Element) Command {
	return NewUICommand().Name("appendchild").SourceID(child.ID)
}

func PrependChildCommand(child *Element) Command {
	return NewUICommand().Name("prependchild").SourceID(child.ID)
}

func InsertChildCommand(child *Element, index int) Command {
	return NewUICommand().Name("insertchild").SourceID(child.ID).Position(index)
}

func ReplaceChildCommand(old *Element, new *Element) Command {
	return NewUICommand().Name("replacechild").SourceID(new.ID).TargetID(old.ID)
}

func RemoveChildCommand(child *Element) Command {
	return NewUICommand().Name("removechild").SourceID(child.ID)
}

func RemoveChildrenCommand() Command {
	return NewUICommand().Name("removechildren")
}

func ActivateViewCommand(viewname string) Command {
	return NewUICommand().Name("activateview").SourceID(viewname)
}

// Mutate allows to send a command that aims to change an element, modifying the
// underlying User Interface.
// The default commands allow to change the ActiveView, AppendChild, PrependChild,
// InsertChild, ReplaceChild, RemoveChild, RemoveChildren.
//
// Why not simply use the Element methods?
//
// For the simple reason that commands can be stored to be replayed later whereas
// using the commands directly would not be a recordable action.
func Mutate(e *Element, command Command) {
	e.SetUI("command", command)
}

func (e *Element) Mutate(command Command) *Element {
	Mutate(e, command)
	return e
}

var DefaultCommandHandler = NewMutationHandler(func(evt MutationEvent) bool {
	command, ok := evt.NewValue().(Command)
	if !ok || (command.ValueType() != "Command") {
		log.Print("Wrong format for command property value ")
		return false // returning false so that handling may continue. E.g. a custom Command object was created and a handler for it is registered further down the chain
	}

	commandname, ok := Object(command).Get("name")
	if !ok {
		log.Print("Command is invalid. Missing command name")
		return true
	}
	cname, ok := commandname.(String)
	if !ok {
		log.Print("Command is invalid. Wrong type for command name value")
		return true
	}

	switch string(cname) {
	case "appendchild":
		sourceid, ok := command["sourceid"]
		if !ok {
			log.Print("Command malformed. Missing source id to append to")
			return true
		}
		e := evt.Origin()
		sid, ok := sourceid.(String)
		if !ok {
			log.Print("Error sourceid is not a string ?!")
			return true
		}
		child := e.ElementStore.GetByID(string(sid))
		log.Print(string(sid))
		if child == nil {
			log.Print("could not find item in element store") // DEBUG
			return true
		}
		e.AppendChild(child)
		return false
	case "prependchild":
		sourceid, ok := command["sourceid"]
		if !ok {
			log.Print("Command malformed. Missing source id to append to")
			return true
		}
		e := evt.Origin()
		sid, ok := sourceid.(String)
		if !ok {
			log.Print("Error sourceid is not a string ?!")
			return true
		}
		child := e.ElementStore.GetByID(string(sid))
		if child == nil {
			return true
		}
		e.PrependChild(child)
		return false
	case "insertChild":
		sourceid, ok := command["sourceid"]
		if !ok {
			log.Print("Command malformed. Missing source id to append to")
			return true
		}
		pos, ok := command["position"]
		if !ok {
			log.Print("Command malformed. Missing insertion positiob.")
			return true
		}
		commandpos, ok := pos.(Number)
		if !ok {
			log.Print("position to insert at is not stored as a valid numeric type")
			return true
		}
		if commandpos < 0 {
			return true
		}
		e := evt.Origin()
		sid, ok := sourceid.(String)
		if !ok {
			log.Print("Error sourceid is not a string ?!")
			return true
		}
		child := e.ElementStore.GetByID(string(sid))
		if child == nil {
			return true
		}
		e.InsertChild(child, int(commandpos))
		return false
	case "replacechild":
		sourceid, ok := command["sourceid"]
		if !ok {
			log.Print("Command malformed. Missing source id to append to")
			return true
		}
		targetid, ok := command["targetid"]
		if !ok {
			log.Print("Command malformed. Missing id of target that should be replaced")
			return true
		}
		e := evt.Origin()
		sid, ok := sourceid.(String)
		if !ok {
			log.Print("Error sourceid is not a string ?!")
			return true
		}
		tid, ok := targetid.(String)
		if !ok {
			log.Print("Error targetid is not a string ?!")
			return true
		}
		newc := e.ElementStore.GetByID(string(sid))
		oldc := e.ElementStore.GetByID(string(tid))
		if newc == nil || oldc == nil {
			return true
		}
		e.ReplaceChild(oldc, newc)
		return false
	case "removechild":
		sourceid, ok := command["sourceid"]
		if !ok {
			log.Print("Command malformed. Missing id of source of mutation")
			return true
		}
		e := evt.Origin()
		sid, ok := sourceid.(String)
		if !ok {
			log.Print("Error sourceid is not a string ?!")
			return true
		}
		child := e.ElementStore.GetByID(string(sid))
		if child == nil {
			return true
		}
		e.RemoveChild(child)
		return false
	case "removechildren":
		evt.Origin().RemoveChildren()
		return false
	case "activateview":
		sourceid, ok := command["sourceid"]
		if !ok {
			log.Print("Command malformed. Missing viewname to activate, stored in sourceid")
			return true
		}
		viewname, ok := sourceid.(String)
		if !ok {
			log.Print("Error viewname/sourceid is not a string ?!")
			return true
		}
		err := evt.Origin().ActivateView(string(viewname))
		if err != nil {
			log.Print(err)
		}
		return false
	default:
		return true
	}
})

func (e *Element) Watch(category string, propname string, owner *Element, h *MutationHandler) *Element {
	p, ok := owner.Properties.Categories[category]
	if !ok {
		p = newProperties()
		owner.Properties.Categories[category] = p
	}
	p.NewWatcher(propname, e)
	e.PropMutationHandlers.Add(owner.ID+"/"+category+"/"+propname, h)
	return e
}

func (e *Element) Unwatch(category string, propname string, owner *Element) *Element {
	p, ok := owner.Properties.Categories[category]
	if !ok {
		return e
	}
	p.RemoveWatcher(propname, e)
	return e
}

func (e *Element) WatchGroup(category string, target *Element, h *MutationHandler) *Element {
	p, ok := target.Properties.Categories[category]
	if !ok {
		p = newProperties()
		target.Properties.Categories[category] = p
	}
	p.NewWatcher("existifallpropertieswatched", e)
	e.PropMutationHandlers.Add(target.ID+"/"+category+"/"+"existifallpropertieswatched", h)
	return e
}

func (e *Element) UnwatchGroup(category string, owner *Element) *Element {
	p, ok := owner.Properties.Categories[category]
	if !ok {
		return e
	}
	p.RemoveWatcher("existifallpropertieswatched", e)
	return e
}

func (e *Element) AddEventListener(event string, handler *EventHandler, nativebinding NativeEventBridge) *Element {
	e.EventHandlers.AddEventHandler(event, handler)
	if nativebinding != nil {
		nativebinding(event, e)
	}
	return e
}
func (e *Element) RemoveEventListener(event string, handler *EventHandler, native bool) *Element {
	e.EventHandlers.RemoveEventHandler(event, handler)
	if native {
		if e.NativeEventUnlisteners.List != nil {
			e.NativeEventUnlisteners.Apply(event)
		}
	}
	return e
}

// Mounted returns whether the subtree the current Element belongs to is attached
// to the main tree or not.
func (e *Element) Mounted() bool {
	if e.subtreeRoot == nil {
		return false // kinda DEBUG left because whole implementation sketchy
	}
	if e.subtreeRoot.Parent == nil && e.subtreeRoot == e.root {
		return true
	}
	return false
}

func (e *Element) OnMount(h *MutationHandler) {
	nh := NewMutationHandler(func(evt MutationEvent) bool {
		b, ok := evt.NewValue().(Bool)
		if !ok || !bool(b) {
			return false
		}
		return h.Handle(evt)
	})
	e.Watch("event", "mounted", e, nh)
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
func (e *Element) Set(category string, propname string, value Value, flags ...bool) {
	var inheritable bool
	if len(flags) > 0 {
		inheritable = flags[0]
	}
	// Persist property if persistence mode has been set at Element creation
	pmode := PersistenceMode(e)

	if e.ElementStore != nil {
		storage, ok := e.ElementStore.PersistentStorer[pmode]
		if ok {
			if category != "ui" {
				storage.Store(e, category, propname, value, flags...)
			}
		}
	}

	if category == "ui" && propname != "mutationrecords" {
		mrs, ok := e.Get("ui", "mutationrecords")
		if !ok {
			mrs = NewList()
		}
		mrslist, ok := mrs.(List)
		if !ok {
			mrslist = NewList()
		}
		mrslist = append(mrslist, NewMutationRecord(category, propname, value))
		e.Set("ui", "mutationrecords", mrslist)
	}

	// Mutationrecords persistence
	if e.ElementStore != nil {
		storage, ok := e.ElementStore.PersistentStorer[pmode]
		if ok && category == "ui" && propname == "mutationrecords" {
			storage.Store(e, category, propname, value, flags...)
		}
	}
	e.Properties.Set(category, propname, value, inheritable)
	evt := e.NewMutationEvent(category, propname, value)
	e.PropMutationHandlers.DispatchEvent(evt)
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

// SetUI inserts a key/value pair under the "data" category in the element property store.
// First flag in the variadic argument, if true, denotes whether the property should be inheritable.
// It does not automatically update any potential property representation stored
// for rendering use in the "ui" category/namespace.
func (e *Element) SetUI(propname string, value Value, flags ...bool) {
	e.Set("ui", propname, value, flags...)
}

// SetDataSyncUI will set a "data" property and update the same-name property value
// located in the "ui namespace/category and used by the User Interface, for instance, for rendering..
// Typically NOT used when the data is being updated from the UI.
func (e *Element) SetDataSyncUI(propname string, value Value, flags ...bool) {
	var inheritable bool
	if len(flags) > 0 {
		inheritable = flags[0]
	}
	e.Properties.Set("data", propname, value, inheritable)

	e.Set("ui", propname, value, flags...)

	evt := e.NewMutationEvent("data", propname, value)
	e.PropMutationHandlers.DispatchEvent(evt)
}

// SyncUISetData is used in event handlers when a user changed a value accessible
// via the User Interface, typically.
//
// For instance, after a User event changes the value via a GUI control, we would set
// this value to the new value chosen by the user and then set the corresponding data
// with a call to SetData (and not SetDataSyncUI since the UI value is alread up-to-date).
//
// First flag in the variadic argument, if true, denotes whether the property should be inheritable.
func (e *Element) SyncUISetData(propname string, value Value, flags ...bool) {
	var inheritable bool
	if len(flags) > 0 {
		inheritable = flags[0]
	}
	e.Properties.Set("ui", propname, value, inheritable)
	if propname != "mutationrecords" {
		mrs, ok := e.Get("ui", "mutationrecords")
		if !ok {
			mrs = NewList()
		}
		mrslist, ok := mrs.(List)
		if !ok {
			mrslist = NewList()
		}
		mrslist = append(mrslist, NewMutationRecord("ui", propname, value))
		e.Set("ui", "mutationrecords", mrslist)
	}

	e.SetData(propname, value, flags...)
}

type MutationRecord Object

func (m MutationRecord) discriminant() discriminant { return "particleui" }
func (m MutationRecord) ValueType() string          { return "MutationRecord" }
func (m MutationRecord) RawValue() Object           { return Object(m).RawValue() }

func NewMutationRecord(category string, propname string, value Value) MutationRecord {
	mr := NewObject().SetType("MutationRecord")
	mr.Set("category", String(category))
	mr.Set("property", String(propname))
	mr.Set("value", value)
	mr.Set("timestamp", String(time.Now().UTC().String()))

	return MutationRecord(mr)
}

// LoadProperty is a function typically used to return a UI Element to a
// given state. As such, it does not trigger a mutation event
// The proptype is a string that describes the property (default,inherited, local, or inheritable).
// For properties of the 'ui' namespace, i.e. properties that are used for rendering,
// we create and dispatch a mutation event since loading a property is change inducing at the
// UI level.
func LoadProperty(e *Element, category string, propname string, proptype string, value Value) {
	e.Properties.Load(category, propname, proptype, value)
	if category == "ui" {
		evt := e.NewMutationEvent(category, propname, value)
		e.PropMutationHandlers.DispatchEvent(evt)
	}
}

// Delete removes the property stored for the given category if it exists.
// Inherited properties cannot be deleted.
// Default properties cannot be deleted either for now.
func (e *Element) Delete(category string, propname string) {
	e.Properties.Delete(category, propname)
	evt := e.NewMutationEvent(category, propname, nil)
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

func Get(e *Element, key string) (Value, bool) {
	return e.Get("data", key)
}

func Set(e *Element, key string, value Value) {
	e.Set("data", key, value, false)
}

func Watcher(e *Element, target *Element, key string, h *MutationHandler) {
	e.Watch("data", key, target, h)
}

// Route returns the path to an Element.
// If the path to an Element includes a parameterized view, the returned route is
// parameterized as well.
//
// Important notice: views that are nested within a fixed element use that Element ID for routing.
// In order for links using the routes to these views to not be breaking between refresh/reruns of an app (hard requirement for online link-sharing), the ID of the parent element
// should be generated so as to not change. Using the default PRNG-based ID generator is very likely to not be a good-fit here.
//
// For instance, if we were to create a dynamic view composed of retrieved tweets, we would not use the default ID generator but probably reuse the tweet ID gotten via http call for each Element.
// Building a shareable link toward any of these elements still require that every ID generated in the path is stable across app refresh/re-runs.
func (e *Element) Route() string {
	// TODO if root is window and not app root, might need to implement additional logic to make link creation process stop at app root.
	var Route string
	if e.Mounted() {
		return ""
	}
	if e.ViewAccessPath == nil {
		return "/"
	}

	for _, n := range e.path.List {
		rpath := pathSegment(n, e.ViewAccessPath)
		Route = Route + "/" + rpath
	}
	return Route
}

// viewAdjacence determines whether an Element has more than one adjacent
// sibling view.
func (e *Element) viewAdjacence() bool {
	var count int
	if e.AlternateViews != nil {
		count++
	}
	if e.path != nil && len(e.path.List) > 1 {
		firstAncestor := e.path.List[len(e.path.List)-1]
		if firstAncestor.AlternateViews != nil {
			vnode := e.ViewAccessPath.nodes[len(e.ViewAccessPath.nodes)-1]
			for _, c := range vnode.ViewElements.Elements().List {
				if c.AlternateViews != nil {
					count++
				}
			}
			if count > 1 {
				return true
			}
			return false
		}

		for _, c := range firstAncestor.Children.List {
			if c.AlternateViews != nil {
				count++
			}
		}
		if count > 1 {
			return true
		}
		return false
	}
	return false
}

// pathSegment returns true if the path belongs to a View besides returning the
// first degree relative path of an Element.
// If the view holds Elements which are adjecent view objects, t
func pathSegment(p *Element, views *viewNodes) string {
	rp := p.ID
	if views != nil {
		for _, v := range views.nodes {
			if v.Element.ID == rp {
				rp = v.ViewElements.Name()
				if p.viewAdjacence() {
					rp = p.ID + "/" + rp
				}
				return rp
			}
		}
	}
	return rp
}

// MakeToggable is a simple example of conditional rendering.
// It allows an Element to have a single child Element that can be switched with another one.
//
// For example, if we have a toggable button, one for login and one for logout,
// We can implement the switch between login and logout by switching the inner Elements.
func MakeToggable(conditionName string, e *Element, firstView ViewElements, secondView ViewElements, initialconditionvalue Bool) *Element {
	e.AddView(firstView).AddView(secondView)

	toggle := NewMutationHandler(func(evt MutationEvent) bool {
		value, ok := evt.NewValue().(Bool)
		if !ok {
			value = false
		}
		if bool(value) {
			Mutate(e, ActivateViewCommand(firstView.Name()))
		}
		Mutate(e, ActivateViewCommand(secondView.Name()))
		return true
	})

	e.Watch("data", conditionName, e, toggle)

	e.Set("data", conditionName, initialconditionvalue, false)
	return e
}

// ViewElements defines a type for a named list of children Element that can be appended
// to an Element, constituting as such a "view".
// ViewElements can be parameterized.
type ViewElements struct {
	name         string
	elements     *Elements
	Parameterize func(parameter string, v ViewElements) (*ViewElements, error)
}

func (v ViewElements) Name() string        { return v.name }
func (v ViewElements) Elements() *Elements { return v.elements }
func (v ViewElements) ApplyParameter(paramvalue string) (*ViewElements, error) {
	return v.Parameterize(paramvalue, v)
}

// NewViewElements can be used to create a list of children Elements to append to an element, for display.
// In effect, allowing to create a named view. (note the lower case letter)
// The true definition of a view is: an *Element and a named list of child Elements (ViewElements) constitute a view.
// An example of use would be an empty window that would be filled with different child elements
// upon navigation.
// A parameterized view can be created by using a naming scheme such as ":parameter" (string with a leading colon)
// In the case, the parameter can be retrieve by the router.
func NewViewElements(name string, elements ...*Element) ViewElements {
	for _, el := range elements {
		el.ActiveView = name
	}
	return ViewElements{name, NewElements(elements...), nil}
}

// NewParameterizedView defines a parameterized, named, list of *Element composing a view.
// The Elements can be parameterized by applying a function submitted as argument.
// This function can and probably should implement validation.
// It may for instance be used to verify that the parameter value belongs to a finite
// set of accepted values.
func NewParameterizedView(parametername string, paramFn func(string, ViewElements) (*ViewElements, error), elements ...*Element) ViewElements {
	if !strings.HasPrefix(parametername, ":") {
		parametername = ":" + parametername
	}
	n := NewViewElements(parametername, elements...)
	n.Parameterize = paramFn
	return n
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
	ViewElements
}

func newViewNode(e *Element, view ViewElements) viewNode {
	return viewNode{e, view}
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

type discriminant string // just here to pin the definition of the Value interface to this package

// Value is the type for Element property values.
type Value interface {
	discriminant() discriminant
	RawValue() Object
	ValueType() string
}

func (e *Element) discriminant() discriminant { return "particleui" }
func (e *Element) ValueType() string          { return "Element" }
func (e *Element) RawValue() Object {
	o := NewObject().SetType("Element")

	o["id"] = String(e.ID)
	o["name"] = String(e.Name)
	constructoroptions, ok := e.Get("internals", "constructoroptions")
	if ok {
		o.Set("constructoroptions", constructoroptions)
	}

	constructorname, ok := e.Get("internals", "constructorname")
	if !ok {
		return nil
	}
	cname, ok := constructorname.(String)
	if !ok {
		return nil
	}
	o["constructorname"] = cname

	o["elementstoreid"] = String(e.ElementStore.Global.ID)
	return o.RawValue()
}

type Bool bool

func (b Bool) discriminant() discriminant { return "particleui" }
func (b Bool) RawValue() Object {
	o := NewObject()
	o["typ"] = "Bool"
	o["value"] = bool(b)
	return o.RawValue()
}
func (b Bool) ValueType() string { return "Bool" }

type String string

func (s String) discriminant() discriminant { return "particleui" }
func (s String) RawValue() Object {
	o := NewObject()
	o["typ"] = "String"
	o["value"] = string(s)
	return o.RawValue()
}
func (s String) ValueType() string { return "String" }

type Number float64

func (n Number) discriminant() discriminant { return "particleui" }
func (n Number) RawValue() Object {
	o := NewObject()
	o["typ"] = "Number"
	o["value"] = float64(n)
	return o.RawValue()
}
func (n Number) ValueType() string { return "Number" }

type Object map[string]interface{}

func (o Object) discriminant() discriminant { return "particleui" }

func (o Object) RawValue() Object {
	p := NewObject()
	for k, val := range o {
		v, ok := val.(Value)
		if ok {
			p[k] = map[string]interface{}(v.RawValue())
			continue
		}
		p[k] = val // typ should still be a plain string, calling RawValue twice in a row should be idempotent
		continue
	}
	return p
}

func (o Object) ValueType() string {
	t, ok := o.Get("typ")
	if !ok {
		return "undefined"
	}
	s, ok := t.(string)
	if !ok {
		return "undefined object"
	}
	return string(s)
}

func (o Object) Get(key string) (interface{}, bool) {
	v, ok := o[key]
	return v, ok
}

func (o Object) Set(key string, value Value) {
	o[key] = value
}
func (o Object) SetType(typ string) Object {
	o["typ"] = typ
	return o
}
func (o Object) Value() Value {
	switch o.ValueType() {
	case "Bool":
		v, ok := o.Get("value")
		if !ok {
			return nil
		}
		res, ok := v.(bool)
		if !ok {
			return nil
		}
		return Bool(res)
	case "String":
		v, ok := o.Get("value")
		if !ok {
			return nil
		}
		res, ok := v.(string)
		if !ok {
			return nil
		}
		return String(res)
	case "Number":
		v, ok := o.Get("value")
		if !ok {
			return nil
		}
		res, ok := v.(float64)
		if !ok {
			return nil
		}
		return Number(res)
	case "List":
		v, ok := o.Get("value")
		if !ok {
			return nil
		}
		l, ok := v.([]interface{})
		if !ok {
			return nil
		}
		m := NewList()
		for _, val := range l {
			r, ok := val.(map[string]interface{})
			if ok {
				v := Object(r).Value()
				m = append(m, v)
				continue
			} else {
				return nil
			}
		}
		return m
	case "Object":
		p := NewObject()
		for k, val := range o {
			v, ok := val.(Value)
			if !ok {
				m, ok := val.(map[string]interface{})
				if ok {
					obj := Object(m)
					p.Set(k, obj.Value())
					continue
				}
				p[k] = val
				continue
			}
			u, ok := v.(Object)
			if !ok {
				p.Set(k, v)
				continue
			}
			m, ok := val.(map[string]interface{})
			if ok {
				obj := Object(m)
				p.Set(k, obj.Value())
			}
			p.Set(k, u.Value())
		}
		return p
	case "Command":
		p := NewObject()
		for k, val := range o {
			v, ok := val.(Value)
			if !ok {
				m, ok := val.(map[string]interface{})
				if ok {
					obj := Object(m)
					p.Set(k, obj.Value())
					continue
				}
				p[k] = val
				continue
			}
			u, ok := v.(Object)
			if ok {
				p.Set(k, u.Value())
				continue
			}
			p.Set(k, v)
		}
		return Command(p)
	case "MutationRecord":
		p := NewObject()
		for k, val := range o {
			v, ok := val.(Value)
			if !ok {
				m, ok := val.(map[string]interface{})
				if ok {
					obj := Object(m)
					p.Set(k, obj.Value())
					continue
				}
				p[k] = val
				continue
			}
			u, ok := v.(Object)
			if ok {
				p.Set(k, u.Value())
				continue
			}
			p.Set(k, v)
		}
		return MutationRecord(p)
	case "Element":
		p := NewObject()
		for k, val := range o {
			v, ok := val.(Value)
			if !ok {
				m, ok := val.(map[string]interface{})
				if ok {
					obj := Object(m)
					p.Set(k, obj.Value())
					continue
				}
				p[k] = val
				continue
			}
			u, ok := v.(Object)
			if ok {
				p.Set(k, u.Value())
				continue
			}
			p.Set(k, v)
		}

		id, ok := p.Get("id")
		if !ok {
			return nil
		}
		name, ok := p.Get("name")
		if !ok {
			return nil
		}
		elementstoreid, ok := p.Get("elementstoreid")
		if !ok {
			return nil
		}
		constructorname, ok := p.Get("constructorname")
		if !ok {
			return nil
		}
		elstoreid, ok := elementstoreid.(String)
		if !ok {
			log.Print("Wrong type for ElementStore ID")
			return nil
		}
		// Let's get the elementstore
		elstore, ok := Stores.Get(string(elstoreid))
		if !ok {
			return nil
		}
		// Let's try to see if the element is in the ElementStore already
		elid, ok := id.(String)
		if !ok {
			log.Print("Wrong type for Element ID stored in ui.Value")
			return nil
		}
		element := elstore.GetByID(string(elid))
		if element != nil {
			return element
		}
		// Otherwise we construct it. (TODO: make sure that element constructors try to get the data in store)
		cname, ok := constructorname.(String)
		if !ok {
			log.Print("Wrong type for constructor name.")
			return nil
		}
		constructor, ok := elstore.Constructors[string(cname)]
		if !ok {
			log.Print("constructor not found at thhe recorded name from Element store. Cannot create Element " + elid + "from Value")
		}
		ename, ok := name.(String)
		if !ok {
			log.Print("Element name in Value of wring type.")
			return nil
		}

		coptions := make([]string, 0)
		constructoroptions, ok := p.Get("constructoroptions")
		if ok {
			objoptlist, ok := constructoroptions.(Object)
			if ok {
				voptlist := objoptlist.Value()
				optlist, ok := voptlist.(List)
				if ok {
					for _, opt := range optlist {
						sopt, ok := opt.(String)
						if !ok {
							return nil
						}
						coptions = append(coptions, string(sopt))
					}
				}
			}
		}
		return constructor(string(ename), string(elid), coptions...)

	default:
		return o
	}
}

func NewObject() Object {
	o := Object(make(map[string]interface{}))
	o["typ"] = "Object"
	return o
}

type List []Value

func (l List) discriminant() discriminant { return "particleui" }
func (l List) RawValue() Object {
	o := NewObject().SetType("List")

	raw := make([]interface{}, 0)
	for _, v := range l {
		raw = append(raw, v.RawValue())
	}
	o["value"] = raw
	return o.RawValue()
}
func (l List) ValueType() string { return "List" }

func NewList(val ...Value) List {
	if val != nil {
		return List(val)
	}
	l := make([]Value, 0)
	return List(l)
}
