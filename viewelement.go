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
	}
	v := ViewElement{e}
	for _, view := range views {
		v.AddView(view)
	}

	e.OnMounted(NewMutationHandler(func(evt MutationEvent) bool {
		l, ok := evt.Origin().Root().Get("internals", "views")
		if !ok {
			list := NewList(String((evt.Origin().ID)))
			evt.Origin().Root().Set("internals", "views", list)
		} else {
			list, ok := l.(List)
			if !ok {
				list = NewList(String(evt.Origin().ID))
				evt.Origin().Root().Set("internals", "views", list)
			} else {
				list = append(list, String(evt.Origin().ID))
				evt.Origin().Root().Set("internals", "views", list)
			}
		}
		return false

	}).RunASAP().RunOnce())
	

	e.OnDeleted(NewMutationHandler(func(evt MutationEvent)bool{
		l, ok := evt.Origin().Global.Get("internals", "views")
		if ok{
			list, ok := l.(List)
			if ok{
				list = list.Filter(func(v Value)bool{
					return !Equal(v,String(evt.Origin().ID))
				})
				evt.Origin().Global.Set("internals", "views", list)
			}
		}
		return false
	}))

	return v
}

func (v ViewElement) SetDefaultView(name string) ViewElement { // TODO DEBUG OnUnmount vs OnUnmounted
	if strings.HasPrefix(name, ":") {
		panic("FAILURE: cannot choose a route parameter as a default route. A value is required.")
	}
	ve := v.AsElement()
	ve.SetDataSetUI("defaultview", String(name))
	ve.OnMounted(NewMutationHandler(func(evt MutationEvent) bool {
		n, ok := ve.Get("ui", "defaultview")
		if !ok {
			return false
		}
		nm := string(n.(String))
		v.ActivateView(nm)
		return false
	}))
	return v
}

// AddView adds a view to a ViewElement.
func (v ViewElement) AddView(view View) ViewElement {
	v.AsElement().addView(view)
	v.AsElement().Set("authorized", view.Name(), Bool(true))
	return v
}

// RetrieveView returns a pointer to a View if it exists. The View should not
// be active.
func (v ViewElement) RetrieveView(name string) *View {
	return v.AsElement().retrieveView(name)
}

// SetAuthorization is a shortcut for the ("authorized",viewname) prop that allows
// to determine whether a view is accessible or not.
func (v ViewElement) SetAuthorization(viewname string, isAuthorized bool) {
	v.AsElement().Set("authorized", viewname, Bool(isAuthorized))
}

// isViewAuthorized is a predicate function returning the authorization status
// of a view.
func (v ViewElement) IsViewAuthorized(name string) bool {
	return v.isViewAuthorized(name)
}

func (v ViewElement) isViewAuthorized(name string) bool {
	val, ok := v.AsElement().Get("authorized", name)
	if !ok {
		return false
	}
	b := val.(Bool)
	return bool(b)
}

func (v ViewElement) hasStaticView(name string) bool { // name should not start with a colon
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

func (v ViewElement) HasStaticView(name string) bool {
	return v.hasStaticView(name)
}

// ActivateView sets the active view of a ViewElement.
// If no View exists for the name argument or is not authorized, an error is returned.
func (v ViewElement) ActivateView(name string) error {
	val, ok := v.AsElement().Get("authorized", name)
	if !ok {
		panic(errors.New("authorization error " + name + " " + v.AsElement().ID)) // it's ok to panic here. the client can send the stacktrace. Should not happen.
	}
	auth := val.(Bool)

	if auth != Bool(true) {
		return errors.New("Unauthorized")
	}

	return v.AsElement().activateView(name)
}

func (v ViewElement) OnParamChange(h *MutationHandler) {
	v.AsElement().Watch("ui", "viewparameter", v, h)
}

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
	return !v.hasStaticView(viewname)
}

// prefetchView triggers data prefetching for a ViewElement.
// It requires the name of the view that will be activated as argument so that it can start
// prefetching the elements that are part of the target view (if unactivated).
// and then triggers prefetching on the view itself.
func (v ViewElement) prefetchView(name string) {
	ve := v.AsElement()
	if v.hasStaticView(name) && v.isViewAuthorized(name) && ve.ActiveView != name {
		for _, c := range ve.Children.List {
			c.Prefetch()
		}
	}
}

func (e *Element) addView(v View) *Element {
	if e.InactiveViews == nil {
		e.InactiveViews = make(map[string]View) // Important to put that on top... it creates
		// effectively a ViewElement out of an Elmeent. attach below depends on that
	}

	if v.Elements() != nil {
		for _, child := range v.Elements().List {
			if child.Parent != nil{
				child.Parent.RemoveChild(child)
			}
			child.ViewAccessNode = newViewAccessNode(child, v.Name())
			attach(e, child, false)
		}
	}
	e.InactiveViews[v.Name()] = v
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

func (e *Element) activateView(name string) error {
	if isParameter(name) {
		panic("this is likely to be a programmer error. View name inputs can not lead with a colon.")
	}
	if e.ActiveView == name {
		n,ok:=e.Get("ui", "activeview")
		if !ok || n.(String).String() != name{
			panic("active view is not set correctly")
		}
		e.TriggerEvent("viewactivated", String(name)) // DEBUG ??? not needed nmormally
		return nil
	}

	if e.ActiveView == ""{
		v,ok:= e.Get("ui", "activeview")
		if ok{
			if v.(String).String() == name{
				e.ActiveView = name
				delete(e.InactiveViews, name)
				return nil
			}
		}
	}

	wasmounted:= e.Mounted()

	newview, ok := e.InactiveViews[name]
	DEBUG(e.ActiveView," to be replaced by inactive view ", newview.elements.List)
	if !ok {
		if isParameter(e.ActiveView) {
			// let's check the name recorded in the state
			n, ok := e.Get("ui", "activeview")
			if !ok {
				panic("FAILURE: parameterized view is activated but no activeview name exists in state")
			}
			if nm := string(n.(String)); nm == name {
				DEBUG("parameterized view is already active")
				return nil
			}

			e.Set("ui", "viewparameter", String(name)) // necessary because not every change of (ui,activeview) is a viewparameter change.
			e.Set("ui", "activeview", String(name))
			e.TriggerEvent("viewactivated", String(name))
			DEBUG(name)
			return nil
		}
		// Support for parameterized views

		p, ok := ViewElement{e}.hasParameterizedView()
		if !ok {
			return errors.New("View does not exist for " + name)
		}
		view := e.InactiveViews[":"+p]
		oldviewname := e.ActiveView
		if oldviewname != "" {
			e.InactiveViews[oldviewname] = NewView(oldviewname, e.Children.List...)
			for _, child := range e.Children.List {
				detach(child)

				if e.Native != nil {
					e.Native.RemoveChild(child)
				}

				attach(e, child, false)
				finalize(child,true,wasmounted)
			}
			e.Children.RemoveAll()
		}
		e.ActiveView = ":" + p
		/*for _, newchild := range view.Elements().List {
			e.appendChild(BasicElement{newchild})
		}
		*/ // todo Review this as it should work. elements don't seem removed
		e.SetChildrenElements(view.elements.List...)

		e.Set("ui", "viewparameter", String(name))
		e.Set("ui", "activeview", String(name))
		e.TriggerEvent("viewactivated", String(name))
		return nil
	}

	// 1. replace the current view into e.InactiveViews
	cccl := make([]*Element, len(e.Children.List))
	copy(cccl, e.Children.List)
	for _, child := range e.Children.List {
		e.removeChild(child)
		attach(e, child, false)
	}
	e.InactiveViews[e.ActiveView] = NewView(string(e.ActiveView), cccl...)

	// 2. mount the target view
	e.ActiveView = name
	for _, child := range newview.Elements().List {
		e.appendChild(child)
	}

	delete(e.InactiveViews, name)
	e.Set("ui", "activeview", String(name))
	e.TriggerEvent("viewactivated", String(name))
	return nil
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

// AddDefaultView is an *Element modifier that defines a View for an *Element.
// It gets activated each time the *Element gets mounted.
func AddDefaultView(name string, elements ...AnyElement) func(*Element) *Element {
	return func(e *Element) *Element {
		e = AddView(name, elements...)(e)
		ViewElement{e}.SetDefaultView(name)
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
