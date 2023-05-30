package ui

import (
	"errors"
	"strings"
)

// ViewElement defines a type of Element which can display different named versions.
// A version is defined as a View.
type ViewElement struct {
	Raw *Element
}

// Element returns the underlying *Element corresponding to the view.
// A ViewElement constitutes merely an interface for specific *Element objects.
func (v ViewElement) AsElement() *Element {
	return v.Raw
}

func (v ViewElement) watchable() {}
func (v ViewElement) uiElement() {}

// hasParameterizedView return the parameter name stripped from the initial colon ( ":")
// if it exists.
func (v ViewElement) hasParameterizedView() (string, bool) {
	e := v.AsElement()
	if strings.HasPrefix(e.ActiveView, ":") {
		return strings.TrimPrefix(e.ActiveView, ":"), true
	}
	for k := range e.InactiveViews {
		if strings.HasPrefix(k, ":") {
			return strings.TrimPrefix(k, ":"), true
		}
	}
	return "", false
}

func NewViewElement(e *Element, views ...View) ViewElement {
	if e.InactiveViews == nil {
		e.InactiveViews = make(map[string]View) // Important to put that on top... it creates
		// effectively a ViewElement out of an Elmeent. attach below depends on that
	} else{
		panic("FAILURE: cannot create a ViewElement out of an Element which already has views")
	}

	v := ViewElement{e}
	for _, view := range views {
		v.AddView(view)
	}
	v.SetAuthorization("",true)

	e.OnMounted(NewMutationHandler(func(evt MutationEvent) bool {
		l, ok := evt.Origin().Root.Get("internals", "views")
		if !ok {
			list := NewList(String((evt.Origin().ID)))
			evt.Origin().Root.Set("internals", "views", list)
		} else {
			list, ok := l.(List)
			if !ok {
				list = NewList(String(evt.Origin().ID))
				evt.Origin().Root.Set("internals", "views", list)
			} else {
				list = list.Append(String(evt.Origin().ID))
				evt.Origin().Root.Set("internals", "views", list)
			}
		}
		return false

	}).RunASAP().RunOnce())

	// a viewElement should have a default view that should activated when mounting, unless
	e.OnMounted(defaultViewMounter) // TODO remove
	

	e.OnDeleted(NewMutationHandler(func(evt MutationEvent)bool{
		l, ok := evt.Origin().Root.Get("internals", "views")
		if ok{
			list, ok := l.(List)
			if ok{
				list = list.Filter(func(v Value)bool{
					return !Equal(v,String(evt.Origin().ID))
				})
				evt.Origin().Root.Set("internals", "views", list)
			}
		}
		return false
	}))

	e.Watch("ui","activeview",e,NewMutationHandler(func(evt MutationEvent) bool {
		// TODO sync e.ActiveView with evt.NewValue()
		evt.Origin().TriggerEvent("viewactivated",evt.NewValue())
		return false
	}))

	// onstart MutationHandler
	onstart:= NewMutationHandler(func(evt MutationEvent) bool {
		vname := evt.NewValue().(String).String()
		auth:= ViewElement{evt.Origin()}.IsViewAuthorized(vname)

		if !auth {
			DEBUG("unauthorized view: ", vname)
			v.AsElement().ErrorTransition("activateview", String("Unauthorized"))
			return false
		}
		evt.Origin().activateView(vname)
		
		return false
	})

	// onerror MutationHandler
	onerror:= NewMutationHandler(func(evt MutationEvent) bool {
		evt.Origin().Set("internals","viewactivation",evt.NewValue())
		return false
	})

	// oncancel MutationHandler
	oncancel:= NewMutationHandler(func(evt MutationEvent) bool {
		evt.Origin().Set("internals","viewactivation",evt.NewValue())
		return false
	})

	// onend MutationHandler
	onend := NewMutationHandler(func(evt MutationEvent) bool {
		// If no transition Error, then the transition was successful
		if !TransitionError(evt.Origin(), "activateview") && !TransitionCancelled(evt.Origin(), "activateview") {
			evt.Origin().SetUI("activeview", evt.NewValue())
		}
		
		return false
	})

	e.DefineTransition("activateview",onstart,onerror,oncancel,onend)

	return v
}

var defaultViewMounter = NewMutationHandler(func(evt MutationEvent) bool {
	e:=evt.Origin()
	var ok bool
	
	_,ok= e.Get("internals","mountdefaultview")
	if ok{
		v:= e.retrieveView("")
		if v == nil{
			if e.ActiveView == "" {
				return false // defaultview is already mounted
			}
			panic("FAILURE: default view is not defined")
		}
		oldview := NewView(e.ActiveView, e.Children.List...)
		e.RemoveChildren()
		e.addView(oldview)

		e.ActiveView = ""

		if v != nil{
			e.SetChildrenElements(v.elements.List...)
			delete(e.InactiveViews, "")
		}
		
	}
	
	return false
})

// ChangeDefaultView sets the default view of a ViewElement. It is the view that will be displayed when
// a ViewElement mounts.
func (v ViewElement) ChangeDefaultView(elements ...*Element) ViewElement {
	e:= v.AsElement()
	e.Set("internals","mountdefaultview",Bool(true))
	n:= NewView("", elements...)
	v.AddView(n)
	return v
}

// AddView adds a view to a ViewElement.
func (v ViewElement) AddView(view View) ViewElement {
	v.SetAuthorization(view.Name(), true)
	v.AsElement().addView(view)
	
	return v
}

// RetrieveView returns a pointer to a View if it exists. The View should not
// be active. If the view is active or does not exist, a nil View pointer is returned.
func (v ViewElement) RetrieveView(name string) *View {
	return v.AsElement().retrieveView(name)
}

// SetAuthorization is a shortcut for the ("authorized",viewname) prop that allows
// to determine whether a view is accessible or not.
func (v ViewElement) SetAuthorization(viewname string, isAuthorized bool) {
	v.AsElement().Set("authorized", viewname, Bool(isAuthorized))
}

// IsViewAuthorized is a predicate function returning the authorization status
// of a view.
func (v ViewElement) IsViewAuthorized(name string) bool {
	val, ok := v.AsElement().Get("authorized", name)
	if !ok {
		return false
	}
	b := val.(Bool)
	return bool(b)
}

// HasStaticView returns true if a ViewElement has a non-parametered view corresponding to a given name
func (v ViewElement) HasStaticView(name string) bool { // name should not start with a colon
	if v.AsElement().ActiveView == name {
		return true
	}
	inactiveviews := v.AsElement().InactiveViews
	for k, _ := range inactiveviews {
		if k == name {
			return true
		}
	}
	return false
}

// ActivateView sets the active view of a ViewElement.
// If no View exists for the name argument or is not authorized, an error is returned.
func (v ViewElement) ActivateView(name string) error {
	e:= v.AsElement()
	e.StartTransition("activateview", String(name))

	if TransitionError(e, "activateview") {
		v,err := TransitionEndValue(e, "activateview")
		if err != nil {
			panic(err)
		}
		l:= v.(List)
		return errors.New(l.Get(1).(String).String())
	}
	return nil
}

// OnParamChange registers a MutationHandler that will be triggered when a view parameter changes.
// The view paraemeter holds the current name of the active, parametered, view.
func (v ViewElement) OnParamChange(h *MutationHandler) {
	v.AsElement().Watch("ui", "viewparameter", v, h)
}

// OnActivated registers a MutationHandler that will be triggered each time a view has been activated.
func (v ViewElement) OnActivated(viewname string, h *MutationHandler) {
	nh := NewMutationHandler(func(evt MutationEvent) bool {
		view := evt.NewValue().(String)
		if string(view) != viewname {
			return false
		}
		return h.Handle(evt)
	})
	if h.Once {
		nh = nh.RunOnce()
	}

	if h.ASAP {
		nh = nh.RunASAP()
	}
	v.AsElement().WatchEvent("viewactivated", v.AsElement(), nh)
}

func (v ViewElement) IsParameterizedView(viewname string) bool {
	if _, ok := v.hasParameterizedView(); !ok {
		return false
	}
	return !v.HasStaticView(viewname)
}

// prefetchView triggers data prefetching for a ViewElement.
// It requires the name of the view that will be activated as argument so that it can start
// prefetching the elements that are part of the target view (if unactivated).
// and then triggers prefetching on the view itself.
func (v ViewElement) prefetchView(name string) {
	ve := v.AsElement()
	if v.HasStaticView(name) && v.IsViewAuthorized(name) && ve.ActiveView != name {
		for _, c := range ve.Children.List {
			c.Prefetch()
		}
	}
}

func (e *Element) addView(v View) *Element {
	if e.InactiveViews == nil {
		e.InactiveViews = make(map[string]View)
	}

	defer func(){
		e.InactiveViews[v.Name()] = v
	}()

	if v.Elements() != nil {
		for _, child := range v.Elements().List {
			var detachFn func()
			if child.Parent != nil{
				detachFn = detach(child)
				
			}
			child.ViewAccessNode = newViewAccessNode(child, v.Name())
			attachFn := attach(e, child)
			
			if detachFn != nil{
				defer detachFn() // Execute the detach function
			}
			
			defer attachFn() // Execute the attach function
		}
	}
	
	return e
}

func (e *Element) retrieveView(name string) *View {
	v, ok := e.InactiveViews[name]
	if !ok {
		return nil
	}
	return &v
}

func isParameter(name string) bool {
	if strings.HasPrefix(name, ":") && len(name) > 1 {
		return true
	}
	return false
}

func (e *Element) activateView(name string) {
	if isParameter(name) {
		panic("this is likely to be a programmer error. View name inputs can not lead with a colon.")
	}

	if name == ""{
		panic("frmwork error: view name can't be the empty string. This is reserved for default view and never 'activated'.")
	}
	if e.ActiveView == name {
		e.EndTransition("activateview", String(name)) // already active
		return
	}

	// TODO should actiation cancellation be considered an error state?

	newview, ok := e.InactiveViews[name]
	if !ok {
		if isParameter(e.ActiveView) {
			// let's check the name recorded in the state
			n, ok := e.Get("ui", "activeview")
			if !ok {
				panic("FAILURE: parameterized view is activated but no activeview name exists in state")
			}
			if nm := string(n.(String)); nm == name {
				e.EndTransition("activateview", String(name)) // already active
				return
			}

			e.Set("ui", "viewparameter", String(name)) // necessary because not every change of (ui,activeview) is a viewparameter change.
			e.EndTransition("activateview", String(name))
			return
		}
		// Support for parameterized views

		p, ok := ViewElement{e}.hasParameterizedView()
		if !ok {
			e.ErrorTransition("activateview", String(name), String("this view does not exist"))
			return
		}
		view := e.InactiveViews[":"+p]
		oldviewname := e.ActiveView
		if oldviewname != "" {
			oldview := NewView(oldviewname, e.Children.List...)
			e.RemoveChildren()
			e.AsElement().addView(oldview)
			/*for _, child := range e.Children.List {
				finalize := detach(child)

				if e.Native != nil {
					e.Native.RemoveChild(child)
				}

				//attach(e, child, false)
				finalize()
			}
			e.Children.RemoveAll()
			*/
		}
		e.ActiveView = ":" + p

		e.SetChildrenElements(view.elements.List...)

		e.Set("ui", "viewparameter", String(name))
		e.EndTransition("activateview", String(name))
		return
	}

	// 1. replace the current view into e.InactiveViews
	var oldview View

	if e.ActiveView == ""{
		_,ok:= e.Get("internals", "defaultview")
		if ok{
			oldview = NewView(e.ActiveView, e.Children.List...)		
			e.RemoveChildren()
			e.addView(oldview)
		}
	}else{
		oldview = NewView(e.ActiveView, e.Children.List...)
		e.RemoveChildren()
		e.addView(oldview)
	}


	// 2. mount the target view
	e.ActiveView = name
	e.SetChildrenElements(newview.elements.List...)

	delete(e.InactiveViews, name)
	
	e.EndTransition("activateview", String(name))	
}

// AddView is an *Element modifier that is used to add an activable named view to an element.
func AddView(name string, elements ...AnyElement) func(*Element) *Element {
	return func(e *Element) *Element {
		v := NewView(name, convertAny(elements...)...)
		if e.isViewElement() {
			ViewElement{e}.AddView(v)
			return e
		}
		NewViewElement(e, v)
		return e
	}
}

// DefaultView is an *Element modifier that defines a default View for an *Element.
func DefaultView(elements ...*Element) func(*Element) *Element {
	return func(e *Element) *Element {
		ViewElement{e}.ChangeDefaultView(elements...)
		return e
	}
}

func convertAny(elements ...AnyElement) []*Element {
	res := make([]*Element, 0, len(elements))
	for _, e := range elements {
		res = append(res, e.AsElement())
	}
	return res
}
