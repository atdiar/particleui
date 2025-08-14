package ui

import (
	"errors"
	"net/url"
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
	} else {
		panic("FAILURE: cannot create a ViewElement out of an Element which already has views")
	}

	v := ViewElement{e}
	for _, view := range views {
		v.AddView(view)
	}
	v.SetAuthorization("", true)

	e.OnMounted(OnMutation(func(evt MutationEvent) bool {
		l, ok := evt.Origin().Root.Get(Namespace.Internals, "views")
		if !ok {
			list := NewList(String((evt.Origin().ID)))
			evt.Origin().Root.Set(Namespace.Internals, "views", list.Commit())
		} else {
			list, ok := l.(List)
			if !ok {
				list = NewList(String(evt.Origin().ID)).Commit()
				evt.Origin().Root.Set(Namespace.Internals, "views", list)
			} else {
				list = list.MakeCopy().Append(String(evt.Origin().ID)).Commit()
				evt.Origin().Root.Set(Namespace.Internals, "views", list)
			}
		}
		return false

	}).RunASAP().RunOnce())

	// a viewElement should have a default view that should activated when mounting, unless
	e.OnMounted(defaultViewMounter) // TODO remove

	e.OnUnmount(OnMutation(func(evt MutationEvent) bool {
		v.ActivateView("")
		return false
	}))

	v.OnChange(OnMutation(func(evt MutationEvent) bool {
		if _, ok := v.AsElement().Get(Namespace.Navigation, "query"); ok {
			v.AsElement().Set(Namespace.Navigation, "query", NewObject().Commit())
		}
		return false
	}))

	e.OnDeleted(OnMutation(func(evt MutationEvent) bool {
		l, ok := evt.Origin().Root.Get(Namespace.Internals, "views")
		if ok {
			list, ok := l.(List)
			if ok {
				list = list.Filter(func(v Value) bool {
					return !Equal(v, String(evt.Origin().ID))
				})
				evt.Origin().Root.Set(Namespace.Internals, "views", list)
			}
		}
		return false
	}))

	e.Watch(Namespace.UI, "activeview", e, OnMutation(func(evt MutationEvent) bool {
		vname := evt.NewValue().(String).String()
		if v.HasStaticView(vname) {
			e.ActiveView = vname
		} else if pv, ok := v.hasParameterizedView(); ok {
			e.ActiveView = ":" + pv
		} else {
			e.ActiveView = ""
		}
		evt.Origin().TriggerEvent("viewactivated", evt.NewValue())
		return false
	}))

	// onstart MutationHandler
	onstart := OnMutation(func(evt MutationEvent) bool {
		vname := evt.NewValue().(String).String()
		auth := ViewElement{evt.Origin()}.IsViewAuthorized(vname)

		if !auth {
			DEBUG("unauthorized view: ", vname)
			v.AsElement().ErrorTransition(prop.ActivateView, String(vname), String("Unauthorized"))
			return false
		}
		evt.Origin().activateView(vname)

		return false
	})

	// onerror MutationHandler
	onerror := OnMutation(func(evt MutationEvent) bool {
		evt.Origin().Set(Namespace.Internals, prop.ViewActivation, evt.NewValue())
		return false
	})

	// oncancel MutationHandler
	oncancel := OnMutation(func(evt MutationEvent) bool {
		evt.Origin().Set(Namespace.Internals, prop.ViewActivation, evt.NewValue())
		return false
	})

	// onend MutationHandler
	onend := OnMutation(func(evt MutationEvent) bool {
		evt.Origin().SetUI("activeview", evt.NewValue())
		return false
	})

	e.DefineTransition(prop.ActivateView, onstart, onerror, oncancel, onend)

	return v
}

// ActiveViewName retrieves the name of the current active view if it exists. // TODO DEBUG remove bool return and instates a default?
func (v ViewElement) ActiveViewName() (string, bool) {
	a, ok := v.AsElement().GetUI("activeview")
	if ok {
		return a.(String).String(), ok
	}
	return "", ok
}

func (v ViewElement) ViewExists(name string) bool {
	_, ok := v.AsElement().InactiveViews[name]
	if ok {
		return true
	}
	return v.AsElement().ActiveView == name
}

var defaultViewMounter = OnMutation(func(evt MutationEvent) bool {
	e := evt.Origin()
	var ok bool

	_, ok = e.Get(Namespace.Internals, "mountdefaultview")
	if ok {
		v := e.retrieveView("")
		if v == nil {
			if e.ActiveView == "" {
				return false // defaultview is already mounted
			}
			panic("FAILURE: default view is not defined")
		}
		oldview := NewView(e.ActiveView, e.Children.List...)
		e.RemoveChildren()
		e.addView(oldview)

		e.ActiveView = ""

		if v != nil {
			e.SetChildren(v.elements.List...)
			delete(e.InactiveViews, "")
		}

	}

	return false
})

// ChangeDefaultView sets the default view of a ViewElement. It is the view that will be displayed when
// a ViewElement mounts.
func (v ViewElement) ChangeDefaultView(elements ...*Element) ViewElement {
	e := v.AsElement()
	e.Set(Namespace.Internals, "mountdefaultview", Bool(true))
	n := NewView("", elements...)
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
	e := v.AsElement()
	e.StartTransition(prop.ActivateView, String(name))

	if TransitionErrored(e, prop.ActivateView) {
		v, err := TransitionEndValue(e, prop.ActivateView)
		if err != nil {
			panic(err)
		}
		l := v.(List)
		return errors.New(l.Get(1).(String).String())
	}
	return nil
}

func (v ViewElement) SetQueryValidator(viewname string, validator *ValidationSchema) {
	// SetQueryValidator sets a QueryValidator for a given view.
	// The validator will be used to validate query parameters when the view is activated.
	var validatorStr string
	if validator != nil {
		validatorBStr, err := validator.MarshalJSON()
		if err != nil {
			DEBUG("error marshalling validator: ", err)
			// panic ?
		}
		validatorStr = string(validatorBStr)
	} else {
		return
	}
	o, ok := v.AsElement().Get(Namespace.Navigation, "queryvalidator")
	if !ok {
		o := NewObject()
		o.Set(viewname, String(validatorStr))
		v.AsElement().Set(Namespace.Navigation, "queryvalidator", o.Commit())
		// If the query is invalid, it gets ignored.
		v.OnQuery(viewname, OnMutation(func(evt MutationEvent) bool {
			if !v.QueryIsValidFor(viewname) {
				return true
			}
			return false
		}))
		return
	}
	o = o.(Object).MakeCopy().Set(viewname, String(validatorStr)).Commit()
	v.AsElement().Set(Namespace.Navigation, "queryvalidator", o)
}

func (v ViewElement) QueryIsValidFor(viewname string) bool {
	// ValidateQuery checks if the query parameters for a given view are valid
	// according to the QueryValidator set for that view.
	o, ok := v.AsElement().Get(Namespace.Navigation, "queryvalidator")
	if !ok {
		return true // no validator set
	}
	queryValidator, ok := o.(Object).Get(viewname)
	if !ok {
		return true // no validator for this view
	}
	validator, ok := queryValidator.(String)
	if !ok {
		return true // not a valid validator
	}
	// Unmarshal the validator
	var schema ValidationSchema
	err := schema.UnmarshalJSON([]byte(validator))
	if err != nil {
		DEBUG("error unmarshalling query validator: ", err)
		return false
	}
	// Get the current query parameters
	query, ok := v.AsElement().Get(Namespace.Navigation, "query")
	if !ok {
		err := ValidateQueryParams(schema, nil)
		if err != nil {
			DEBUG("error validating query parameters: ", err)
			return false // no query parameters set
		} else {
			return true // no query parameters set, but no error either
		}
	}
	qryObh := query.(Object)
	var params = url.Values{}
	err = Deserialize(qryObh, &params)
	if err != nil {
		DEBUG("error deserializing query parameters: ", err)
		return false // error deserializing query parameters
	}
	err = ValidateQueryParams(schema, params)
	if err != nil {
		return false
	}
	return true // query parameters are valid
}

// OnParamChange registers a MutationHandler that will be triggered when a view parameter changes.
// The view parameter holds the current name of the active, parametered, view.
func (v ViewElement) OnParamChange(h *MutationHandler) {
	v.AsElement().Watch(Namespace.UI, "viewparameter", v, h)
}

// OnActivated registers a MutationHandler that will be triggered each time a view has been activated.
// MutationHanlders can be registered to handle the presence of potential
// query parameters when a view is being activated on navigation.
func (v ViewElement) OnActivated(viewname string, h *MutationHandler) {

	nh := OnMutation(func(evt MutationEvent) bool {
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

func (v ViewElement) OnChange(h *MutationHandler) {
	// OnChange registers a MutationHandler that will be triggered when the active view changes.
	// This is useful to handle changes in the active view, such as when a new view is activated.
	v.AsElement().Watch(Namespace.UI, "activeview", v, h)
}

func (v ViewElement) OnQuery(activeViewName string, h *MutationHandler) {
	// OnQuery registers a MutationHandler that will be triggered
	// when a new query is set for a given view.
	var g *MutationHandler
	g = OnMutation(func(evt MutationEvent) bool {
		name, _ := v.ActiveViewName()
		if name == activeViewName {
			if h != nil {
				return h.Handle(evt)
			}
			return false
		}
		return false
	})

	if h.Once {
		g = g.RunOnce()
	}
	if h.ASAP {
		g = g.RunASAP()
	}

	v.AsElement().Watch(Namespace.Navigation, "query", v, g)
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

	defer func() {
		e.InactiveViews[v.Name()] = v
	}()

	if v.Elements() != nil {
		for _, child := range v.Elements().List {
			if child.Parent != nil {
				child.Parent.RemoveChild(child)

			}
			child.ViewAccessNode = newViewAccessNode(child, v.Name())
			attachFn := attach(e, child)
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

	if e.ActiveView == name {
		// TODO handle the case where name == "" and e.ActiveView == ""
		// In that case, depending whether there is a default view,
		// might want to activate it.
		if name != "" {
			e.EndTransition(prop.ActivateView, String(name)) // already active
			return
		}
		newview, ok := e.InactiveViews[name]
		if !ok {
			// if there is a default view, then it is active, otherwise
			// there must be a mistake.
			_, ok := e.Get(Namespace.Internals, "mountdefaultview")
			if ok {
				e.EndTransition(prop.ActivateView, String(name)) // already active
				return
			}
			// TODO panic or silently return? Or transition Error
			e.ErrorTransition(prop.ActivateView, String(name), String("no default view defined!"))
			return
		}
		e.SetChildren(newview.elements.List...)
		delete(e.InactiveViews, name)
		e.EndTransition(prop.ActivateView, String(name))
		return
	}

	// TODO should activation cancellation be considered an error state?

	newview, ok := e.InactiveViews[name]
	if !ok {
		if name == "" {
			// TODO handle the case where there is no default
			// view defined.
			_, ok := e.Get(Namespace.Internals, "mountdefaultview")
			if ok {
				e.EndTransition(prop.ActivateView, String(name)) // already active
				return
			}
			e.ErrorTransition(prop.ActivateView, String(name), String("no default view defined."))
			return
		}

		if isParameter(e.ActiveView) {
			// let's check the name recorded in the state
			n, ok := e.Get(Namespace.UI, "activeview")
			if !ok {
				panic("FAILURE: parameterized view is activated but no activeview name exists in state")
			}
			if nm := string(n.(String)); nm == name {
				e.EndTransition(prop.ActivateView, String(name)) // already active
				return
			}

			e.Set(Namespace.UI, "viewparameter", String(name)) // necessary because not every change of (ui,activeview) is a viewparameter change.
			e.EndTransition(prop.ActivateView, String(name))
			return
		}
		// Support for parameterized views

		p, ok := ViewElement{e}.hasParameterizedView()
		if !ok {
			e.ErrorTransition(prop.ActivateView, String(name), String("this view does not exist"))
			return
		}
		view := e.InactiveViews[":"+p]
		//oldviewname := e.ActiveView
		e.ActiveView = ":" + p

		e.SetChildren(view.elements.List...)

		e.Set(Namespace.UI, "viewparameter", String(name))
		e.EndTransition(prop.ActivateView, String(name))
		return
	}

	// 1. replace the current view into e.InactiveViews
	oldview := NewView(e.ActiveView, e.Children.List...)
	e.RemoveChildren()
	e.addView(oldview)

	// 2. mount the target view
	e.ActiveView = name
	e.SetChildren(newview.elements.List...)

	delete(e.InactiveViews, name)

	e.EndTransition(prop.ActivateView, String(name))
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
