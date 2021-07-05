// Package ui is a library of functions for simple, generic gui development.
package ui

import (
	"errors"
	"log"
	"net/url"
	"strings"
)

var (
	ErrNotFound         = errors.New("Not Found")
	ErrUnauthorized     = errors.New("Unauthorized")
	ErrFrameworkFailure = errors.New("Framework Failure")
)

// Router stores shortcuts to given states of the application.
// These shortcuts take the form of URIs.
// The router is also in charge of modifying the application state to reach any
// state registered as a shortcut upon request.
type Router struct {
	BaseURL string
	root    ViewElement

	Links map[string]Link

	Routes *rnode

	History *NavHistory

	LeaveTrailingSlash bool
}

// NewRouter takes an Element object which should be the entry point of the router
// as well as the document root which should be the entry point of the document/application tree.
func NewRouter(baseurl string, rootview ViewElement) *Router {
	u, err := url.Parse(baseurl)
	if err != nil {
		panic(err)
	}
	base := strings.TrimSuffix(u.Path, "/")
	rootview.Element().Global.Set("internals", "baseurl", String(baseurl), true)
	r := &Router{base, rootview, make(map[string]Link, 300), newrootrnode(rootview), NewNavigationHistory().Push("/"), false}

	// Routes registration and treemutation based registration
	v, ok := r.root.Element().Root().Get("internals", "views")
	if ok {
		l, ok := v.(List)
		if ok {
			for _, val := range l {
				viewEl, ok := val.(*Element)
				if !ok || !viewEl.isViewElement() {
					panic("internals/views does not hold a proper Element")
				}
				r.insert(ViewElement{viewEl})
			}
		}
	}
	r.root.Element().Root().Watch("event", "docupdate", r.root.Element(), NewMutationHandler(func(evt MutationEvent) bool {
		v, ok := evt.NewValue().(*Element)
		if !ok {
			panic("Framework error: only an *Element is an acceptable value for docupdate")
		}
		if !v.isViewElement() {
			panic("Only a view is acceptable for a docupdate")
		}
		r.insert(ViewElement{v})
		return false
	}))

	return r
}

// GoTo changes the application state by updating the current route
func (r *Router) GoTo(route string) {
	r.root.Element().Root().Set("navigation", "routechangerequest", String(route))
	r.History.Push(route)
}

func (r *Router) GoBack() {
	if r.History.BackAllowed() {
		r.root.Element().Root().Set("navigation", "routechangerequest", String(r.History.Back()))
	}
}

func (r *Router) GoForward() {
	if r.History.ForwardAllowed() {
		r.root.Element().Root().Set("navigation", "routechangerequest", String(r.History.Forward()))
	}
}

// OnNotfound enables the addition of a special view to the root ViewElement.
// The router should navigate toward it when no match has been found for a given input route.
func (r *Router) OnNotfound(dest View) *Router {
	r.root.AddView(dest)

	r.root.Element().Root().Watch("navigation", "notfound", r.root.Element().Root(), NewMutationHandler(func(evt MutationEvent) bool {
		r.GoTo(r.BaseURL + "/" + dest.Name())
		return false
	}))
	return r
}

// OnUnauthorized enables the addition of a special view to the root ViewElement.
// The router should navigate toward it when access to an input route is not granted
// due to insufficient rights.
func (r *Router) OnUnauthorized(dest View) *Router {
	r.root.AddView(dest)

	r.root.Element().Root().Watch("navigation", "unauthorized", r.root.Element().Root(), NewMutationHandler(func(evt MutationEvent) bool {
		r.GoTo(r.BaseURL + "/" + dest.Name())
		return false
	}))
	return r
}

// OnAppfailure enables the addition of a special view to the root ViewElement.
// The router should navigate toward it when a malfunction occured.
func (r *Router) OnAppfailure(dest View) *Router {
	r.root.AddView(dest)

	r.root.Element().Root().Watch("navigation", "appfailure", r.root.Element().Root(), NewMutationHandler(func(evt MutationEvent) bool {
		r.GoTo(r.BaseURL + "/" + dest.Name())
		return false
	}))
	return r
}

func (r *Router) insert(v ViewElement) {
	// check that v has r.root as ancestor
	vap := v.Element().ViewAccessPath
	if vap == nil {
		return
	}
	ancestry := vap.Nodes[0].Element
	if ancestry.ID != r.root.Element().ID {
		return
	}
	nrn := newchildrnode(v, r.Routes)
	r.Routes.insert(nrn)
}

// Match returns whether a route is valid or not. It can be used in tests to
// Make sure that an app links are not breaking.
func (r *Router) Match(route string) error {
	_, err := r.Routes.match(route)
	return err

}

// handler returns a mutation handler which deals with route change.
func (r *Router) handler() *MutationHandler {
	mh := NewMutationHandler(func(evt MutationEvent) bool {
		nroute, ok := evt.NewValue().(String)
		if !ok {
			log.Print("route mutation has wrong type... something must be wrong", evt.NewValue())
			r.root.Element().Root().Set("navigation", "appfailure", Bool(true))
			return true
		}
		newroute := string(nroute)
		if !r.LeaveTrailingSlash {
			if newroute[len(newroute)-1:] == "/" {
				newroute = newroute[:len(newroute)-1]
			}
		}
		newroute = strings.TrimPrefix(newroute, r.BaseURL)

		// 1. Let's see if the URI matches any of the registered routes. (TODO)
		a, err := r.Routes.match(newroute)
		if err != nil {
			r.root.Element().Root().Set("navigation", "unauthorized", Bool(true))
			return true
		}
		err = a()
		if err != nil {
			r.root.Element().Root().Set("navigation", "unauthorized", Bool(true))
			return true
		}

		r.root.Element().Root().SyncUISetData("currentroute", evt.NewValue())
		return false
	})
	return mh
}

// OnRoutechangeRequest allows to trigger a mutation handler before a route change
// is effective. It needs to be called before ListenAndServe. Returning true should
// cancel the current routechangerequest. (enables hijacking of the route change process)
func (r *Router) OnRoutechangeRequest(m *MutationHandler) {
	r.root.Element().Root().Watch("navigation", "routechangerequest", r.root.Element().Root(), m)
}

// ListenAndServe registers a listener for route change.
// It should only be called after the app structure has been fully built.
// It listens on the element that receives routechangeevent (first argument)
//
//
// Example of JS bridging : the nativeEventBridge should add a popstate event listener to window
// It should also dispatch a RouteChangeEvent to bridge browser url mutation into the Go side
// after receiving notice of popstate event firing.
func (r *Router) ListenAndServe(eventname string, target *Element, nativebinding NativeEventBridge) {
	r.verifyLinkActivation()
	root := r.root
	routeChangeHandler := NewEventHandler(func(evt Event) bool {
		log.Print("this is a router test: " + evt.Value()) // DEBUG
		if evt.Type() != eventname {
			log.Print("Event of wrong type. Expected: " + eventname)
			root.Element().Root().Set("navigation", "appfailure", String("500: RouteChangeEvent of wrong type."))
			return true // means that event handling has to stop
		}
		// the target element route should be changed to the event NewRoute value.
		root.Element().Root().Set("navigation", "routechangerequest", String(evt.Value()), false)
		return false
	})

	target.AddEventListener(eventname, routeChangeHandler, nativebinding)
	root.Element().Root().Watch("navigation", "routechangerequest", root.Element().Root(), r.handler())
}

func (r *Router) verifyLinkActivation() {
	for _, l := range r.Links {
		_, ok := l.Raw.Get("event", "activated")
		if !ok {
			panic("Link activation failure: " + l.URI())
		}
	}
}

/*

   router nodes


*/
// A rnode is a router node. It holds information about the viewElement,
// the value field holding the viewid for the corresponding ViewElement,
// and a map of the potential children ViewElements classified by views (via viewnames)
type rnode struct {
	root  *rnode
	value string // just a copy of the ViewElement.Element().ID
	ViewElement
	next map[string]map[string]*rnode // Each rnode has a list of views and each view may link to multiple same level ViewElement map[viewname]map[viewid]rnode
}

func newchildrnode(v ViewElement, root *rnode) *rnode {
	m := make(map[string]map[string]*rnode)
	for k := range v.Element().InactiveViews {
		m[k] = nil
	}
	if a := v.Element().ActiveView; a != "" {
		m[a] = nil
	}
	return &rnode{root, v.Element().ID, v, nil}
}

func newrootrnode(v ViewElement) *rnode {
	r := newchildrnode(v, nil)
	r.root = r
	return r
}

// insert  adds an arbitrary rnode to the rnode trie if  possible (the root
// ViewElement of the rnode ViewAccessPath should be that of the root rnode )
func (rn *rnode) insert(nrn *rnode) {
	v := nrn.ViewElement

	viewpath := v.Element().ViewAccessPath
	if viewpath == nil {
		return
	}
	viewpathnodes := viewpath.Nodes
	if viewpathnodes[0].Element.ID != rn.root.ViewElement.Element().ID {
		return
	}
	l := len(viewpathnodes)
	// attach iteratively the rnodes
	refnode := rn
	viewname := ""
	for i, node := range viewpathnodes {
		if i+1 < l {
			// each view should be a rootnode and should be attached in succession. The end node is our argument.
			view := ViewElement{viewpathnodes[i+1].Element}
			nr := newchildrnode(view, rn)
			refnode.attach(node.View.Name(), nr)
			refnode = nr
			viewname = node.View.Name()
		}
	}
	refnode.attach(viewname, nrn)
}

// attach links to rnodes that corresponds to viewElements that succeeds each other
func (r *rnode) attach(targetviewname string, nr *rnode) {
	m, ok := r.next[targetviewname]
	if !ok {
		m = make(map[string]*rnode)
		r.next[targetviewname] = m
	}
	r, ok = m[nr.ViewElement.Element().ID]
	if !ok {
		m[nr.ViewElement.Element().ID] = nr
	} // else it has already been attached
}

// match verifies that a route passed as arguments corresponds to a given view state.
func (r *rnode) match(route string) (activationFn func() error, err error) {
	activations := make([]func() error, 0)
	route = strings.TrimPrefix(route, "/")
	segments := strings.SplitAfter(route, "/")
	ls := len(segments)

	if ls == 0 {
		return nil, ErrNotFound
	}

	var param string
	m, ok := r.next[segments[0]] // 0 is the index of the viewname at the root ViewElement m is of type map[string]*rnode
	if !ok {
		// Let's see if the ViewElement has a parameterizable view
		param, ok = r.ViewElement.hasParameterizedView()

		if ok {
			if !r.ViewElement.isViewAuthorized(param) {
				return nil, ErrUnauthorized
			}
			r.ViewElement.Element().Set("navigation", param, String(segments[0]))
			if ls != 1 { // we get the next rnodes mapped by viewname
				m, ok = r.next[param]
				if !ok {
					return nil, ErrFrameworkFailure
				}
			}
		}

	}

	// Does other children views need activation? Let's check for it.
	if ls == 1 {
		// check authorization
		if param != "" {
			if r.ViewElement.isViewAuthorized(param) {
				a := func() error {
					return r.ViewElement.ActivateView(segments[0])
				}
				return a, nil
			}
			return nil, ErrUnauthorized
		}
		if r.ViewElement.isViewAuthorized(segments[0]) {
			a := func() error {
				return r.ViewElement.ActivateView(segments[0])
			}
			return a, nil
		}
		return nil, ErrUnauthorized
	}

	if ls%2 != 1 {
		return nil, ErrNotFound
	}

	viewcount := (ls - ls%2) / 2

	// Let's get the next rnode and check that the view mentionned in the route exists (segment[2i+2])

	for i := 1; i <= viewcount; i++ {
		routesegment := segments[2*i]       //ids
		nextroutesegment := segments[2*i+1] //viewnames
		r, ok = m[routesegment]
		if !ok {
			return nil, ErrNotFound
		}

		if r.value != routesegment {
			return nil, ErrNotFound
		}

		// Now that we have the rnode, we can try to see if the nextroutesegment holding the viewname
		// is in the r.next. If not, we check whether the viewElement can be parameterized
		// and the new map pf next rnode is then retrieved if possible.
		m, ok = r.next[nextroutesegment]
		if !ok {
			// Let's see if the ViewElement has a parameterizable view
			param, ok = r.ViewElement.hasParameterizedView()

			if ok {
				if !r.ViewElement.isViewAuthorized(param) {
					return nil, ErrUnauthorized
				}
				r.ViewElement.Element().Set("navigation", param, String(segments[2*i])) // TODO check

				m, ok = r.next[param] // we get the next rnodes mapped by viewnames
				if !ok {
					return nil, ErrFrameworkFailure
				}

			} else {
				return nil, ErrNotFound
			}
		}
		if !r.ViewElement.isViewAuthorized(nextroutesegment) {
			return nil, ErrUnauthorized
		}
		a := func() error {
			return r.ViewElement.ActivateView(nextroutesegment)
		}
		activations = append(activations, a)
	}
	activationFn = func() error {
		for _, a := range activations {
			err := a()
			if err != nil {
				return err
			}
		}
		return nil
	}
	return activationFn, nil
}

/*

	Navigation link creation

*/

// Link holds the representation (under the form of an URI) of the application state
// required for the target View to be available for display on screen.
type Link struct {
	Raw *Element

	Target   ViewElement
	ViewName string

	Router *Router
}

func (l Link) URI() string {
	return l.Target.Element().Route() + "/" + l.Target.Element().ID + "/" + l.ViewName
}

func (l Link) Activate() {
	l.Router.GoTo(l.URI())
}

func (r *Router) NewLink(target ViewElement, viewname string) Link {
	// If previously created, it has been memoized. let's retrieve it then. otherwise,
	// let's create it.
	l, ok := r.Links[target.Element().ID+"/"+viewname]
	if ok {
		return l
	}

	e := NewElement(viewname, target.Element().ID+"/"+viewname, r.root.Element().DocType)
	nh := NewMutationHandler(func(evt MutationEvent) bool {
		e.Set("event", "activated", Bool(true))
		return false
	})
	e.Watch("event", "mounted", target.Element(), nh)
	l = Link{e, target, viewname, r}
	r.Links[target.Element().ID+"/"+viewname] = l

	return l
}

/*

   Navigation History

*/

// NavHistory holds the Navigation History. (aka NavStack)
type NavHistory struct {
	Stack  []string
	Cursor int
}

func NewNavigationHistory() *NavHistory {
	return &NavHistory{make([]string, 0, 300), 0}
}

func (n *NavHistory) Push(URI string) *NavHistory {
	n.Stack = append(n.Stack[:n.Cursor], URI)
	n.Cursor++
	return n
}

func (n *NavHistory) Back() string {
	if n.BackAllowed() {
		n.Cursor--
	}
	return n.Stack[n.Cursor]
}

func (n *NavHistory) Forward() string {
	if n.ForwardAllowed() {
		n.Cursor++
	}
	return n.Stack[n.Cursor]
}

func (n *NavHistory) BackAllowed() bool {
	return n.Cursor > 0
}

func (n *NavHistory) ForwardAllowed() bool {
	return n.Cursor < len(n.Stack)-1
}
