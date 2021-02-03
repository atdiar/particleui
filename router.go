// Package ui is a library of functions for simple, generic gui development.
package ui

import (
	"errors"
	"log"
	"net/url"
	"strings"
)

// Router allows for a genre of condiitional rendering where Elements belonging
// to the App tree register with a router a function that will mutate their state.
// (in other words, the action of mutating an Element is delegated to an outside object: the router)
// This router may apply such mutating functions depending on some external events such
// as a native browser url change event.
type Router struct {
	GlobalHandlers []RouteChangeHandler // used to implement guards such as when user has not the access rights to reach some app state. (authentication?)
	BaseURL        string
	root           *Element

	*Element // used to watch over the *root

	Routes *routeNode

	// may not need the below
	CurrentRoute string // this is the router state descibed by a string value (also describing partially the )
	CurrentView  *Element
	Registrees   []*registree // Holds the views that have not been added to the

	LeaveTrailingSlash bool
}

func NewRouter(rootview *Element, documentroot *Element) (*Router, error) {
	if view.viewAdjacence() {
		return nil, errors.New("The router cannot be created on a view that has siblings. It should be a unique entry point ot the app. ")
	}
	documentroot.Set("internals","baseurl",rootview.Route())
	return &Router{make([]RouteChangeHandler, 0), rootview.Route(), documentroot, nil, newRouteNode("/"), "/", nil, nil, false}, nil
}

func (r *Router) SetBaseURL(base string) *Router { // TODO may delete this
	u, err := url.Parse(base)
	if err != nil {
		return r
	}
	r.BaseURL = strings.TrimSuffix(u.Path, "/")
	r.root.Set("internals","baseurl",r.BaseURL)
	return r
}

func (r *Router) AddGlobalHandlers(handlers ...RouteChangeHandler) {
	if r.GlobalHandlers == nil {
		r.GlobalHandlers = make([]RouteChangeHandler, 0)
	}
	r.GlobalHandlers = append(r.GlobalHandlers, handlers...)
}

// Rogister stores the route leading to the display of an *Element. It should
// typically used for view-type Elements (i.e. Elements with named internal versions)
//
func (r *Router) Register(view *Element, middleware ...RouteChangeHandler) { // TODO RegisterWithName (for shorter alias so we can do Redirect("notfound"))
	if r.Registrees == nil {
		r.Registrees = make([]*registree, 0)
	}
	r.Registrees = append(r.Registrees, newRegistree(view, middleware...))
}

// NotFound
func (r *Router) NotFoundView(viewElements ...*Element) {
	r.Element.AddView(NewViewElements("NotFound", viewElements...))
}

func (r *Router) UnauthorizedView(viewElements ...*Element) {
	r.Element.AddView(NewViewElements("Unauthorized", viewElements...))
}

func (r *Router) AppErrorView(viewElements ...*Element) {
	r.Element.AddView(NewViewElements("AppError", viewElements...))
}

func (r *Router) ErrorNotFound() {
	v := r.Element.RetrieveView("NotFound")
	if v == nil {
		log.Print("Page not found but unable to redirect toward the error page")
		return
	}
	els := v.Elements().List
	if len(els) <= 0 {
		log.Print("View element missing to the Page not found")
		return
	}
	var e *Element
	for _, el := range els {
		if el != nil {
			e = el
			break
		}
	}
	r.Redirect(e.Route(), nil)
}

func (r *Router) ErrorUnauthorized() {
	v := r.Element.RetrieveView("Unauthorized")
	if v == nil {
		log.Print("Unauthorized but unable to redirect toward the Unauthorized page")
		return
	}
	els := v.Elements().List
	if len(els) <= 0 {
		log.Print("View element missing to the Unauthorized notice page")
		return
	}
	var e *Element
	for _, el := range els {
		if el != nil {
			e = el
			break
		}
	}
	r.Redirect(e.Route(), nil)
}

func (r *Router) ErrorAppLogic() {
	v := r.Element.RetrieveView("AppError")
	if v == nil {
		log.Print("AppError but unable to redirect toward the AppError page")
		return
	}
	els := v.Elements().List
	if len(els) <= 0 {
		log.Print("View element missing to the AppError notice page")
		return
	}
	var e *Element
	for _, el := range els {
		if el != nil {
			e = el
			break
		}
	}
	r.Redirect(e.Route(), nil)
}

// Redirect creates and dispatches an event from the router to the root
// of the app, requesting for a route change. This request is
// typically dispatched to the native host via the nativebinding function instead
// when provided.
func (r *Router) Redirect(route string, nativebinding NativeDispatch) {
	r.root.DispatchEvent(NewRouteChangeEvent(route, r.root), nativebinding)
}

// mount stores the route of an Element as well as the route of the Element it is a view of.
// A route represents a serie of states for the User-Interface that needs to be
// activated for an element to show up on screen. (or be added to the DOM tree, said otherwise in JS terms)
func (r *Router) mount(element *Element, middleware ...RouteChangeHandler) {
	if element.ViewAccessPath == nil || len(element.ViewAccessPath.nodes) == 0 {
		return
	}
	route := element.Route()
	r.Routes.Insert(route, newRegistree(element, middleware...))
}

// Handler returns a mutation handler which deals with route change.
func (r *Router) Handler() *MutationHandler {
	mh := NewMutationHandler(func(evt MutationEvent) bool {
		newroute, ok := evt.NewValue().(string)
		if !ok {
			log.Print("route mutation has wrong type... something must be wrong", evt.NewValue())
			r.ErrorAppLogic()
			return true
		}
		if !r.LeaveTrailingSlash {
			if newroute[len(newroute)-1:] == "/" {
				newroute = newroute[:len(newroute)-1]
			}
		}
		newroute = strings.TrimPrefix(newroute, r.BaseURL)

		Target, ApplyRouteFn := r.Routes.Match(newroute, nil)
		if ApplyRouteFn == nil {
			// No route match
			r.ErrorNotFound()
			return true
		}
		for _, glmiddleware := range r.GlobalHandlers {
			stop := glmiddleware.Handle(Target)
			if stop {
				r.ErrorUnauthorized()
				return stop
			}
		}
		err := ApplyRouteFn()
		if err != nil {
			r.ErrorUnauthorized()
			return true
		}
		return false // todo  i'm wondering if this is the right value to return.
	})
	return mh
}

func (r *Router) serve() {
	if r.Registrees == nil {
		log.Print("No route has been registered it seems.")
	}
	for _, registree := range r.Registrees {
		r.mount(registree.Target, registree.Handlers...)
	}
}

// ListenAndServe registers a listener for route change.
// It should only be called after the app structure has been fully built.
//
// Example of JS bridging : the nativeEventBridge should add a popstate event listener to window
// It should also dispatch a RouteChangeEvent to bridge browser url mutation into the Go side
// after receiving notice of popstate event firing.
func (r *Router) ListenAndServe(nativebinding NativeEventBridge) {
	root := r.root
	routeChangeHandler := NewEventHandler(func(evt Event) bool {
		event, ok := evt.(RouteChangeEvent)
		if !ok {
			log.Print("Event of wrong type. Expected a RouteChangeEvent firing")
			return true // means that event handling has to stop
		}
		// the target element route should be changed to the event NewRoute value.
		root.Set("events", event.Type(), event.NewRoute(), false)
		return false
	})
	// TODO create a litst of events handled by Go
	// Native drivers will be in charge of mapping the events to their native names
	// for binding.
	root.AddEventListener("routechange", routeChangeHandler, nativebinding)
	r.Watch("events", "routechange", root, r.Handler())
	r.serve()
}

type RouteChangeEvent interface {
	NewRoute() string
	Event
}

// NewRouteChangeEvent creates a new Event that is specifically structured to
// inform about a change in the current route. In other terms, aprt from the
// basic Event interface, it implements a NewRoute method which returns the newly
// created current route.
// It takes as second argument the Element which holds the route variable.
// In javascript browser, that would be the Element representing the window
// element, window.location being the route as a URL.
func NewRouteChangeEvent(newroute string, routeChangeTarget *Element) RouteChangeEvent {
	return newroutechangeEvent(newroute, routeChangeTarget)
}

type routeChangeEvent struct {
	Event
	route string
}

func (r routeChangeEvent) NewRoute() string {
	return r.route
}

func newroutechangeEvent(newroute string, root *Element) routeChangeEvent {
	e := NewEvent("routechange", false, false, root, nil)
	return routeChangeEvent{e, newroute}
}

type RouteChangeHandler interface {
	Handle(target *Element) bool
}

type RouteChangeHandleFunc func(target *Element) bool

func (r RouteChangeHandleFunc) Handle(e *Element) bool {
	return r(e)
}

// Each Element is uniquely qualified by a route which represents the serie of
// partial states (values for named Views) that the app should be in.
//
// We build a datastructure shaped like a trie which will hold the registered
// routes decomposed in segments (potentially shared between routes).
type routeNode struct {
	value    string
	Children map[string]*routeNode

	Element *registree // Holds Element we want to have visual access to. Just need to  run the middlewaer and then activate the views on its ViewAccessPath field
}

func newRouteNode(value string) *routeNode {
	return &routeNode{value, nil, nil}
}

func (r *routeNode) Insert(route string, element *registree) {
	segments := strings.SplitAfter(route, "/")
	if segments[0] != r.value {
		panic("Inserting route into router's trie failed. This is a library error.")
	}
	if len(segments) <= 1 {
		r.Element = element
		return
	}
	next := segments[1:]
	if r.Children == nil {
		r.Children = make(map[string]*routeNode)
	}
	c, ok := r.Children[next[0]]
	if !ok {
		c = newRouteNode(next[0])
		r.Children[next[0]] = c
	}
	c.Insert(strings.TrimPrefix(route, segments[0]), element)
}

// Match will check whether the input route usually retrieved from the RouteChangeEvent
// corresponds to one of the routes that was registered.
// If a match is found, it returns the target Element wanted for display and a
// function that mutates the app state, activating the serie of wiews necessary
// to display the target Element.
// If the route contains a query string, it is stored in the target Element datatstore
// under the "querystring" Key.
func (r *routeNode) Match(route string, NavigateFn func() error) (*Element, func() error) {
	segments := strings.SplitAfter(route, "/")
	pathsegment := segments[0]
	var querystring string
	if len(segments) == 1 {
		u, err := url.Parse(pathsegment)
		if err != nil {
			log.Print("route format is invalid")
			return nil, nil
		}
		pathsegment = u.Path
		querystring = u.RawQuery
	}
	if pathsegment != r.value {
		if r.value[:1] != ":" {
			return nil, nil
		}
	}
	NewNavigateFn := func() error {
		if NavigateFn != nil {
			err := NavigateFn()
			if err != nil {
				return err
			}
		}
		for _, h := range r.Element.Handlers {
			stop := h.Handle(r.Element.Target)
			if stop {
				return errors.New("Unauthorized")
			}
		}
		if r.Element.Target.AlternateViews != nil {
			err := r.Element.Target.ActivateView(pathsegment)
			if err != nil {
				return err
			}
			if querystring != "" {
				r.Element.Target.Set("router", "query", querystring, false)
			}
		}
		return nil
	}

	if len(segments) <= 1 {
		return r.Element.Target, NewNavigateFn
	}
	next := segments[1:]
	if r.Children == nil {
		return nil, nil // No route seems to be registered
	}
	c, ok := r.Children[next[0]]
	if !ok {
		return nil, nil // here again, no route seems to have been registered
	}
	return c.Match(strings.TrimPrefix(route, segments[0]), NewNavigateFn)
}

type routeParameters struct {
	List []struct {
		Name  string
		Value string
	}
}

func newRouteParameters() *routeParameters {
	return &routeParameters{make([]struct {
		Name  string
		Value string
	}, 0)}
}

func (r *routeParameters) Set(name string, value string) {
	if r.List != nil {
		r.List = make([]struct {
			Name  string
			Value string
		}, 0)
	}
	r.List = append(r.List, struct {
		Name  string
		Value string
	}{name, value})
}

type registree struct {
	Target   *Element
	Handlers []RouteChangeHandler
}

func newRegistree(e *Element, handlers ...RouteChangeHandler) *registree {
	return &registree{e, handlers}
}
