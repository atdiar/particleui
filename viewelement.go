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

	// necessary if the element we make a viewElement of was already mounted. It doesn't get reattached unless modification
	l, ok := e.Global.Get("internals", "views")
	if !ok {
		list := NewList(e)
		e.Global.Set("internals", "views", list)
	} else {
		list, ok := l.(List)
		if !ok {
			list = NewList(e)
			e.Global.Set("internals", "views", list)
		} else {
			list = append(list, e)
			e.Global.Set("internals", "views", list)
		}
	}
	return v
}

func (v ViewElement) SetDefaultView(name string) ViewElement { // TODO DEBUG OnUnmount vs OnUnmounted
	if strings.HasPrefix(name, ":") {
		panic("FAILURE: cannot choose a route parameter as a default route. A value is required.")
	}
	ve := v.AsElement()
	ve.SetDataSetUI("defaultview", String(name))
	ve.OnUnmount(NewMutationHandler(func(evt MutationEvent) bool {
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

// ActivateView sets the active view of  a ViewElement.
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
	v.AsElement().Watch("ui", "activeview", v.AsElement(), NewMutationHandler(func(evt MutationEvent) bool {
		view := evt.NewValue().(String)
		if string(view) != viewname {
			return false
		}
		return h.Handle(evt)
	}))
}

func (e *Element) addView(v View) *Element {
	if e.InactiveViews == nil {
		e.InactiveViews = make(map[string]View) // Important to put that on top... it creates
		// effectively a ViewElement out of an Elmeent. attach below depends on that
	}

	if v.Elements() != nil {
		for _, child := range v.Elements().List {
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
		panic("this is likely to be a programmer error. VIew name inputs can not lead with a colon.")
	}
	if e.ActiveView == name {
		return nil
	}

	newview, ok := e.InactiveViews[name]
	if !ok {
		if isParameter(e.ActiveView) {
			// let's check the name recorded in the state
			n, ok := e.Get("ui", "activeview")
			if !ok {
				panic("FAILURE: parameterized view is activated but no activeview name exists in state")
			}
			if nm := string(n.(String)); nm == name {
				return nil
			}

			e.SetDataSetUI("viewparameter", String(name))
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
			cccl := make([]*Element, len(e.Children.List))
			copy(cccl, e.Children.List)
			for _, child := range e.Children.List {
				e.removeChild(BasicElement{child})
				attach(e, child, false)
			}
			e.InactiveViews[string(oldviewname)] = NewView(string(oldviewname), cccl...)
		}
		e.ActiveView = ":" + p
		for _, newchild := range view.Elements().List {
			e.appendChild(BasicElement{newchild})
		}
		e.SetDataSetUI("viewparameter", String(name))
		e.SetUI("activeview", String(name))
		return nil
	}

	// 1. replace the current view into e.InactiveViews
	cccl := make([]*Element, len(e.Children.List))
	copy(cccl, e.Children.List)
	for _, child := range e.Children.List {
		e.removeChild(BasicElement{child})
		attach(e, child, false)
	}
	e.InactiveViews[e.ActiveView] = NewView(string(e.ActiveView), cccl...)

	// 2. mount the target view
	e.ActiveView = name
	for _, child := range newview.Elements().List {
		e.appendChild(BasicElement{child})
	}
	delete(e.InactiveViews, name)
	e.SetUI("activeview", String(name))
	return nil
}
