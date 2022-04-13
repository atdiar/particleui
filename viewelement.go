// Package ui is a library of functions for simple, generic gui development.
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

func (v ViewElement) OnActivation(viewname string, h *MutationHandler) {
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
	newview, ok := e.InactiveViews[name]
	if !ok {
		// Check if view is active
		if e.ActiveView == name {
			return nil
		}
		if isParameter(e.ActiveView) {
			// let's check the name recorded in the state
			n, ok := e.Get("ui", "activeview")
			if !ok {
				return errors.New("View is unknown.")
			}
			vname, ok := n.(String)
			if !ok {
				panic("wrong type for view name. This is likely a library error")
			}
			if string(vname) != name {
				return errors.New("View is unknown. Expected " + string(vname) + " instead of " + name)
			}
			return nil
		}
		// Support for parameterized views
		if len(e.InactiveViews) != 0 {
			var view View
			var parameterName string
			for k, v := range e.InactiveViews {
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
				cccl := make([]*Element, len(e.Children.List))
				copy(cccl, e.Children.List)
				if ok && ok2 && oldviewname != "" && e.Children != nil {
					for _, child := range e.Children.List {
						if !viewIsParameterized {
							e.removeChild(BasicElement{child})
							attach(e, child, false)
						}
					}
					if !viewIsParameterized {
						// the view is not parameterized
						e.InactiveViews[string(oldviewname)] = NewView(string(oldviewname), cccl...)
					}
				}
				e.ActiveView = parameterName
				// Let's append the new view Elements
				for _, newchild := range view.Elements().List {
					e.appendChild(BasicElement{newchild})
				}
				e.SetUI("activeview", String(name), false)

				return nil
			}
		}
		return errors.New("View does not exist.")
	}

	// first we detach the current active View and reattach it as an alternative View if non-parameterized
	oldview, ok := e.Get("ui", "activeview")
	oldviewname, ok2 := oldview.(String)
	viewIsParameterized := (string(oldviewname) != e.ActiveView)
	cccl := make([]*Element, len(e.Children.List))
	copy(cccl, e.Children.List)
	if ok && ok2 && e.Children != nil {
		for _, child := range e.Children.List {
			if !viewIsParameterized {
				e.removeChild(BasicElement{child})
				attach(e, child, false)
			}
		}
		if !viewIsParameterized {
			// the view is not parameterized, we put it back in the set of activable views
			e.InactiveViews[string(oldviewname)] = NewView(string(oldviewname), cccl...)
		}
	}
	e.ActiveView = name
	// we attach and activate the desired view
	for _, child := range newview.Elements().List {
		e.appendChild(BasicElement{child})
	}
	delete(e.InactiveViews, name)
	e.SetUI("activeview", String(name), false)

	return nil
}
