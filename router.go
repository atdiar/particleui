// Package ui is a library of functions for simple, generic gui development.
package ui

// Router allows for a genre of condiitional rendering where Elements belonging
// to the App tree register with a router a function that will mutate their state.
// (in other words, the action of mutating an Element is delegated to an outside object: the router)
// This router may apply such mutating functions depending on some external events such
// as a native browser url change event.
type Router struct {
	GlobalHandlers []func() // used to implement guards such as when user has not the access rights to reach some app state. (authentication?)
	RouteHandlers  map[string]func()

	*Element // used to watch over the *root

	// may not need the below
	CurrentRoute string // this is the router state descibed by a string value (also describing partially the )

	Parent     *Router
	Subrouters map[string]*Router
}

func NewRouter(root *Element, routeprefix string) *Router {
	return &Router{make([]func(), 0), make(map[string]func()), root, routeprefix, nil, make(map[string]*Router)}
}

func (r *Router) Register(route string, handler func()) {
	if r.RouteHandlers == nil {
		r.RouteHandlers = make(map[string]func())
		r.RouteHandlers[route] = handler
		return
	}
	rh, ok := r.RouteHandlers[route]
	if !ok {
		r.RouteHandlers[route] = handler
		return
	}
	r.RouteHandlers[route] = func() { rh(); handler() }
}

func (r *Router) RegisterGlobal(handler func()) {
	if r.GlobalHandlers == nil {
		r.GlobalHandlers = make([]func(), 0)
	}
	r.GlobalHandlers = append(r.GlobalHandlers, handler)
}

func (r *Router) NewSubRouter(subroute string, parentroute string, handler func()) *Router {

}
