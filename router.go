// Package ui is a library of functions for simple, generic gui development.
package ui

import (
	"log"
	"strings"
)

// Router allows for a genre of condiitional rendering where Elements belonging
// to the App tree register with a router a function that will mutate their state.
// (in other words, the action of mutating an Element is delegated to an outside object: the router)
// This router may apply such mutating functions depending on some external events such
// as a native browser url change event.
type Router struct {
	GlobalHandlers []RouteChangeHandler // used to implement guards such as when user has not the access rights to reach some app state. (authentication?)
	routePrefix    string
	root           *Element

	*Element // used to watch over the *root

	Routes *routeNode

	// may not need the below
	CurrentRoute string // this is the router state descibed by a string value (also describing partially the )
	CurrentView  *Element
	Registrees   []*registree // Holds the views that have not been added to the
}

func NewRouter(root *Element, prefix string) *Router {
	return &Router{make([]RouteChangeHandler, 0), prefix, root, nil, newRouteNode("/"), "/", nil, nil}
}

func (r *Router) CatchAll(handlers ...RouteChangeHandler) {
	if r.GlobalHandlers == nil {
		r.GlobalHandlers = make([]RouteChangeHandler, 0)
	}
	r.GlobalHandlers = append(r.GlobalHandlers, handlers...)
}

func (r *Router) Register(element *Element, middleware ...RouteChangeHandler) {
	if r.Registrees == nil {
		r.Registrees = make([]*registree, 0)
	}
	r.Registrees = append(r.Registrees, newRegistree(element, middleware...))
}

// load stores the route of an Element as well as the route of the Element it is a view of.
// A route represents a serie of states for the User-Interface that needs to be
// active for an element to show up on screen. (or be added to the DOM tree in JS terms)
func (r *Router) load(element *Element, middleware ...RouteChangeHandler) {
	if element.ViewAccessPath == nil || len(element.ViewAccessPath.nodes) == 0 {
		return
	}

	route := element.Route()
	view := element.ViewAccessPath.nodes[len(element.ViewAccessPath.nodes)-1].Element
	if route == "" {
		return
	}
	r.Routes.Insert(route, newRegistree(element, middleware...))
	r.load(view, middleware...)
	// recursive loading in the trie of the view states that need to be accessible for the Element to show on the render tree.
	// if the view is loaded with a different set of middleware, the new ones supersedes the old set of middleware .
}

func (r *Router) Handler() *MutationHandler {
	var h MutationHandler // TODO define route mutation Handler
  mh:= NewMutationHandler(func(evt MutationEvent) bool {
    newroute,ok:= evt.NewValue().(string)
    if !ok{
      // TODO redirect toward the PAGE NOT FOUND view
    }
    
  })
	return &h
}

func (r *Router) serve() {
	if r.Registrees == nil {
		log.Print("No route has been registered it seems.")
	}
	for _, registree := range r.Registrees {
		r.load(registree.Target, registree.Handlers...)
	}
}

// ListenAndServe registers a listener for route change.
func (r *Router) ListenAndServe(root *Element, eb NativeEventBridge) {
	onroutechange := NewEventHandler(func(evt Event) bool {
		event, ok := evt.(RouteChangeEvent)
		if !ok {
			log.Print("Event of wrong type. Expected a RouteChangeEvent firing")
			return true // means that event handling has to stop
		}
		// the taget element route should be changed to the event NewRoute value.
		root.Set("routechange", event.NewRoute())
		return false
	})

	root.AddEventListener("routechange", onroutechange, eb)
	r.Watch("routechange", root, r.Handler())
  r.serve()
}

type RouteChangeEvent interface {
	NewRoute() string
	Event
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

	Element*registree // Holds Element we want to have visual access to. Just need to  run the middlewaer and then activate the views on its ViewAccessPath field
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

func (r *routeNode) TryRetrieve(route string, params *routeParameters) (*registree, *routeParameters) {
	segments := strings.SplitAfter(route, "/")
	if segments[0] != r.value {
		if r.value[:1] != ":" {
			return nil, nil
		}
		params.Set(r.value, segments[0])
	}

	if len(segments) <= 1 {
		return r.Element, params
	}
	next := segments[1:]
	if r.Children == nil {
		return nil, nil // No route seems to be registered
	}
	c, ok := r.Children[next[0]]
	if !ok {
		return nil, nil // here again, no route seems to have been registered
	}
	return c.TryRetrieve(strings.TrimPrefix(route, segments[0]), params)
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
