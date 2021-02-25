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
	DocType      string
	Constructors map[string]func(name, id string, options ...OptionalArguments) *Element
	ConstructorsOptions map[string]map[string]func(args ...interface{}) func(*Element)*Element
	ByID         map[string]*Element
	Matrix *Element // the matrix Element stores the global state shared by all *Elements
}

// ConstructionOption defines a type of function that is used so that certain options
// are enabled when using a constructor function that creates new Elements.
// These construction options must have been prealably registered by name.
type OptionalArguments func() (string, []interface{})

// WithOption returns an OptionalArguments object which is a function that,
// when called, returns arguments to be passed to a Constructor Function.
// It allows the Constructor function to  use pre-registered  Element constructing
// functions that may add additional Element configuration properties for example.
func WithOption(optionName string, args ...interface{}) OptionalArguments {
	return func() (string,[]interface{}){
		return optionName, args
	}
}

type ConstructorOption struct{
	Name string
	Configurator func(args ...interface{}) func(*Element)*Element
}

func NewConstructorOption(name string, configuratorFn func(args ...interface{}) func(*Element)*Element) ConstructorOption {
	return ConstructorOption{name, configuratorFn}
}

func NewElementStore(storeid string,doctype string) *ElementStore {
	matrix:= NewElement("matrix",storeid,doctype)
	es:= &ElementStore{doctype, make(map[string]func(name string, id string,options ...OptionalArguments) *Element, 0),make(map[string]map[string]func(args ...interface{}) func(*Element)*Element,0), make(map[string]*Element),matrix}
	matrix.WatchGroup("",matrix,NewMutationHandler(func(evt MutationEvent)bool{
		for _,element := range es.ByID{
			element.Set("global",evt.ObservedKey(), evt.NewValue(), false)
		}
		return false
		}))
	return es
}

// NewConstructor registers and returns a new Element construcor function.
func (e *ElementStore) NewConstructor(elementname string, constructor func(name string, id string) *Element, options ...ConstructorOption) func(elname string, elid string, args ...OptionalArguments) *Element {

	// First we register the options that are passed with the Constructor definition
	if options != nil{
		for _,option:= range options{
			n:= option.Name
			f:= option.Configurator
			optlist,ok:= e.ConstructorsOptions[elementname]
			if !ok{
				optlist = make(map[string]func(args ...interface{})func(*Element)*Element)
			}
			optlist[n]=f
		}
	}

	// Then we create the element constructor to return
	c := func(name string, id string, optionsArgs ...OptionalArguments) *Element {
		element := constructor(name, id)
		element.matrix = e.Matrix
		element.WatchGroup("",element, NewMutationHandler(func(evt MutationEvent)bool{
			element.Set("global", evt.ObservedKey(),evt.NewValue(),false)
			return false
			}))
			// TODO optionalArgs  apply the corresponding options
			for _,opt:=range optionsArgs{
				name, args := opt()
				r,ok:= e.ConstructorsOptions[elementname]
				if ok{
					config,ok:=r[name]
					if ok{
						element= config(args...)(element)
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

// EnableGlobalPropertyAccess will, when passed as a configuration option to an Element
// contructor, grant an Element the right to update a global property. Global properties
// are those that belong to the "global" namespace (a.k.a. category) of every Element properties.
func EnableGlobalPropertyAccess(propertyname string) func(*Element)*Element{
	return func(e *Element)*Element{
		e.canMutateGlobalScope = true
		e.matrix.Watch("",propertyname, e, NewMutationHandler(func(evt MutationEvent)bool{
			e.matrix.setGlobal(propertyname,evt.NewValue(),false)
			return false
			}))
		return e
	}
}

// Element is the building block of the User Interface. Everything is described
// as an Element having some mutable properties (graphic properties or data properties)
// From the window to the buttons on a page.
type Element struct {
	matrix *Element //where it all begins :)
	root        *Element
	subtreeRoot *Element // detached if subtree root has no parent unless subtreeroot == root
	path        *Elements

	Parent *Element

	Name    string
	ID      string
	DocType string

	Properties PropertyStore
	canMutateGlobalScope bool
	PropMutationHandlers   *MutationCallbacks     // list of mutation handlers stored at elementID/propertyName (Elements react to change in other elements they are monitoring)
	EventHandlers          EventListeners         // EventHandlers are to be called when the named event has fired.
	NativeEventUnlisteners NativeEventUnlisteners // Allows to remove event listeners on the native element, registered when bridging event listeners from the native UI platform.

	Children   *Elements
	ActiveView string // holds the name of the view currently displayed. If parameterizable, holds the name of the parameter

	ViewAccessPath *viewNodes // List of views that lay on the path to the Element

	AlternateViews map[string]ViewElements // this is a  store for  named views: alternative to the Children field, used for instance to implement routes/ conditional rendering.

	Native NativeElement
}

// NewElement returns a new Element with no properties, no event or mutation handlers.
// Essentlially an empty shell to be customized.
func NewElement(name string, id string, doctype string) *Element {
	return &Element{
		nil,
		nil,
		nil,
		nil,
		nil,
		name,
		id,
		doctype,
		NewPropertyStore(),
		false,
		NewMutationCallbacks(),
		NewEventListenerStore(),
		NewNativeEventUnlisteners(),
		nil,
		"",
		nil,
		nil,
		nil,
	}
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

func(e *Elements) AtIndex(index int) *Element{
	elements := e.List
	if index <0 || index >= len(elements){
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

	if e.Mounted() {
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
				oldview, ok := e.Get("internals", "activeview")
				oldviewname, ok2 := oldview.(string)
				viewIsParameterized := (oldviewname != e.ActiveView)
				if ok && ok2 && oldviewname != "" && e.Children != nil {
					for _, child := range e.Children.List {
						detach(child)
						if !viewIsParameterized {
							attach(e, child, false)
						}
					}
					if !viewIsParameterized {
						// the view is not parameterized
						e.AlternateViews[oldviewname] = NewViewElements(oldviewname, e.Children.List...)
					}
				}

				// Let's append the new view Elements
				for _, newchild := range view.Elements().List {
					e.AppendChild(newchild)
				}
				e.Set("internals", "activeview", name, false)
				e.ActiveView = parameterName
				return nil
			}
		}
		return errors.New("View does not exist.")
	}

	// first we detach the current active View and reattach it as an alternative View if non-parameterized
	oldview, ok := e.Get("internals", "activeview")
	oldviewname, ok2 := oldview.(string)
	viewIsParameterized := (oldviewname != e.ActiveView)
	if ok && ok2 && oldviewname != "" && e.Children != nil {
		for _, child := range e.Children.List {
			detach(child)
			if !viewIsParameterized {
				attach(e, child, false)
			}
		}
		if !viewIsParameterized {
			// the view is not parameterized
			e.AlternateViews[oldviewname] = NewViewElements(oldviewname, e.Children.List...)
		}
	}
	// we attach and activate the desired view
	for _, child := range newview.Elements().List {
		e.AppendChild(child)
	}
	delete(e.AlternateViews, name)
	e.Set("internals", "activeview", name, false)
	e.ActiveView = name

	return nil
}

// attach will link a child Element to the subtree its target parent belongs to.
// It does not however position it in any view specifically. At this stage,
// the Element can not be rendered as part of the view.
func attach(parent *Element, child *Element, activeview bool) {
	defer func() {
		child.Set("event", "attached", true, false)
		if child.Mounted(){
			child.Set("event","mounted", true,false)
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

	e.Set("event", "attached", false, false)
	e.Set("event","mounted",false,false)

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

func (e *Element) PrependChild(child *Element) *Element {
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

// ReplaceChild will replace the target child Element with another.
// Be wary that mutation Watchers and event listeners remain unchanged by default.
// The addition or removal of change observing obk=jects is left at the discretion
// of the user.
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
	if e.subtreeRoot.Parent == nil && e.subtreeRoot != e.root {
		return true
	}
	return false
}

// Get retrieves the value stored for the named property located under the given
// category. The "" category returns the content of the "global" property category.
// The "global" namespace is a local copy of the data that resides in the global
// shared scope common to all Element objects of an ElementStore.
func (e *Element) Get(category, propname string) (interface{}, bool) {
	if category == ""{
		category = "global"
	}
	return e.Properties.Get(category, propname)
}

// Set inserts a key/value pair under a given category in the element property store.
// The "global" category is read-only.
// The global scope may be mutated by changing a property of the "" category (empty string).
// The changes will only take effect if the  Element was granted Global scope access rights for the
// property in question via the EnableGlobalPropertyAccess functional option.
//
func (e *Element) Set(category string, propname string, value interface{}, inheritable bool) {
	if category=="global"{
		log.Print("this namespace is read-only (global). It may not be mutated directly. [see docs])")
		return
	}
	if category == "" && !e.canMutateGlobalScope{
		log.Print("Element does not have sufficient rights to try and mutate global scope")
		return
	}
	e.Properties.Set(category, propname, value, inheritable)
	evt := e.NewMutationEvent(category, propname, value)
	e.PropMutationHandlers.DispatchEvent(evt)
}

func(e *Element) setGlobal(propname string, value interface{}, inheritable bool){
	e.Properties.Set("", propname, value, inheritable)
	evt := e.NewMutationEvent("", propname, value)
	e.PropMutationHandlers.DispatchEvent(evt)
}

// Delete removes the property stored for the given category if it exists.
// Inherited properties cannot be deleted.
// Default properties cannot be deleted either for now.
func (e *Element) Delete(category string, propname string) {
	e.Properties.Delete(category, propname)
	evt := e.NewMutationEvent(category, propname, nil)
	e.PropMutationHandlers.DispatchEvent(evt)
}

func SetDefault(e *Element, category string, propname string, value interface{}) {
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

func Get(e *Element, key string) (interface{}, bool) {
	return e.Get("data", key)
}

func Set(e *Element, key string, value interface{}) {
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
func MakeToggable(conditionName string, e *Element, firstView ViewElements, secondView ViewElements, initialconditionvalue interface{}) *Element {
	e.AddView(firstView).AddView(secondView)

	toggle := NewMutationHandler(func(evt MutationEvent) bool {
		value, ok := evt.NewValue().(bool)
		if !ok {
			value = false
		}
		if value {
			e.ActivateView(firstView.Name())
		}
		e.ActivateView(secondView.Name())
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

// Get retrieves the value of a property stored within a given category.
// A category acts as a namespace for property keys.
func (p PropertyStore) Get(category string, propname string) (interface{}, bool) {
	ps, ok := p.Categories[category]
	if !ok {
		return nil, false
	}
	return ps.Get(propname)
}

func (p PropertyStore) Set(category string, propname string, value interface{}, inheritable ...bool) {
	ps, ok := p.Categories[category]
	if !ok {
		ps = newProperties()
		p.Categories[category] = ps
	}
	var isInheritable bool
	if inheritable != nil && len(inheritable) == 1 && inheritable[0] == true {
		isInheritable = true
	}
	ps.Set(propname, value, isInheritable)
}

func (p PropertyStore) Delete(category string, propname string) {
	ps, ok := p.Categories[category]
	if !ok {
		return
	}
	ps.Delete(propname)
}

func (p PropertyStore) SetDefault(category string, propname string, value interface{}) {
	ps, ok := p.Categories[category]
	if !ok {
		ps = newProperties()
		p.Categories[category] = ps
	}
	ps.SetDefault(propname, value)
}

type Properties struct {
	Default map[string]interface{}

	Inherited map[string]interface{} //Inherited property cannot be mutated by the inheritor

	Local map[string]interface{}

	Inheritable map[string]interface{} // the value of a property overrides the value stored in any of its predecessor value store
	// map key is the address of the element's  property
	// being watched and elements is the list of elements watching this property
	// Inheritable encompasses overidden values and inherited values that are being passed down.
	Watchers map[string]*Elements
}

func newProperties() Properties {
	return Properties{make(map[string]interface{}), make(map[string]interface{}), make(map[string]interface{}), make(map[string]interface{}), make(map[string]*Elements)}
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

func (p Properties) Get(propName string) (interface{}, bool) {
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

func (p Properties) Set(propName string, value interface{}, inheritable bool) {
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

func (p Properties) SetDefault(propName string, value interface{}) {
	p.Default[propName] = value
}
