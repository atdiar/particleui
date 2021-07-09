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
	outlet  ViewElement

	Links map[string]Link

	Routes *rnode

	History *NavHistory

	LeaveTrailingSlash bool
}

// NewRouter takes an Element object which should be the entry point of the router
// as well as the document root which should be the entry point of the document/application tree.
func NewRouter(baseurl string, rootview ViewElement) *Router {
	if !rootview.Element().Mounted() {
		panic("router can only use a view attached to the main tree as a navigation outlet.")
	}
	u, err := url.Parse(baseurl)
	if err != nil {
		panic(err)
	}

	r := &Router{u.Path, rootview, make(map[string]Link, 300), newrootrnode(rootview), NewNavigationHistory(), false}

	r.outlet.Element().Root().Watch("event", "docupdate", r.outlet.Element().Root(), NewMutationHandler(func(evt MutationEvent) bool {
		_, navready := r.outlet.Element().Root().Get("navigation", "ready")
		if !navready {
			v, ok := r.outlet.Element().Global.Get("internals", "views")
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
		}
		return false
	}))

	return r
}

func (r *Router) tryNavigate(newroute string) bool {
	if !r.LeaveTrailingSlash {
		newroute = strings.TrimSuffix(newroute, "/")
	}
	newroute = strings.TrimPrefix(newroute, r.BaseURL)

	// 1. Let's see if the URI matches any of the registered routes. (TODO)
	a, err := r.Routes.match(newroute)
	if err != nil {
		if err == ErrNotFound {
			r.outlet.Element().Root().Set("navigation", "notfound", Bool(true))
			return false
		}
		if err == ErrUnauthorized {
			r.outlet.Element().Root().Set("navigation", "unauthorized", Bool(true))
			return false
		}
		if err == ErrFrameworkFailure {
			r.outlet.Element().Root().Set("navigation", "appfailure", Bool(true))
			return false
		}
	}
	err = a()
	if err != nil {
		r.outlet.Element().Root().Set("navigation", "unauthorized", Bool(true))
		return false
	}
	return true
}

// GoTo changes the application state by updating the current route
func (r *Router) GoTo(route string) {
	route = strings.TrimPrefix(route, r.BaseURL)
	route = strings.TrimPrefix(route, "/")
	if !r.LeaveTrailingSlash {
		route = strings.TrimSuffix(route, "/")
	}
	ok := r.tryNavigate(route)
	if !ok {
		return
	}

	r.outlet.Element().Root().SetDataSyncUI("currentroute", String(route))
	r.History.Push(route)
	//r.outlet.Element().Set("navigation","index",Number(r.History.Cursor))
	log.Println(*r.History) //DEBUG
}

func (r *Router) GoBack() {
	if r.History.BackAllowed() {
		r.outlet.Element().Root().Set("navigation", "routechangerequest", String(r.History.Back()))
	}
}

func (r *Router) GoForward() {
	if r.History.ForwardAllowed() {
		r.outlet.Element().Root().Set("navigation", "routechangerequest", String(r.History.Forward()))
	}
}

func (r *Router) RedirectTo(route string) {
	route = strings.TrimPrefix(route, r.BaseURL)
	route = strings.TrimPrefix(route, "/")
	if !r.LeaveTrailingSlash {
		route = strings.TrimSuffix(route, "/")
	}
	r.outlet.Element().Root().Set("navigation", "routeredirectrequest", String(route))
	r.History.Replace(route)
}

// OnNotfound enables the addition of a special view to the outlet ViewElement.
// The router should navigate toward it when no match has been found for a given input route.
func (r *Router) OnNotfound(dest View) *Router {
	r.outlet.AddView(dest)
	//r.insert(r.outlet)
	r.outlet.Element().Root().Watch("navigation", "notfound", r.outlet.Element().Root(), NewMutationHandler(func(evt MutationEvent) bool {
		r.RedirectTo(dest.Name())
		return false
	}))
	return r
}

// OnUnauthorized enables the addition of a special view to the outlet ViewElement.
// The router should navigate toward it when access to an input route is not granted
// due to insufficient rights.
func (r *Router) OnUnauthorized(dest View) *Router {
	r.outlet.AddView(dest)

	r.outlet.Element().Root().Watch("navigation", "unauthorized", r.outlet.Element().Root(), NewMutationHandler(func(evt MutationEvent) bool {
		r.RedirectTo(dest.Name())
		return false
	}))
	return r
}

// OnAppfailure enables the addition of a special view to the outlet ViewElement.
// The router should navigate toward it when a malfunction occured.
func (r *Router) OnAppfailure(dest View) *Router {
	r.outlet.AddView(dest)

	r.outlet.Element().Root().Watch("navigation", "appfailure", r.outlet.Element().Root(), NewMutationHandler(func(evt MutationEvent) bool {
		r.RedirectTo(dest.Name())
		return false
	}))
	return r
}

func (r *Router) insert(v ViewElement) {
	log.Print("Call to insert: ", v.Element().ID) // DEBUG
	// check that v has r.outlet as ancestor
	vap := v.Element().ViewAccessPath
	if vap == nil || len(vap.Nodes) == 0 {
		if r.Routes.ID() == v.Element().ID {
			nrn := newchildrnode(v, r.Routes)
			r.Routes.insert(nrn)
		}
		return
	}
	var ancestry *Element
	if len(vap.Nodes) == 0 {
		return // this is the outlet view
	} else {
		ancestry = vap.Nodes[0].Element
	}

	if ancestry.ID != r.outlet.Element().ID {
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
			r.outlet.Element().Root().Set("navigation", "appfailure", Bool(true))
			return true
		}
		newroute := string(nroute)
		if !r.LeaveTrailingSlash {
			newroute = strings.TrimSuffix(newroute, "/")
		}
		newroute = strings.TrimPrefix(newroute, r.BaseURL)
		newroute = strings.TrimPrefix(newroute, "/")

		// 1. Let's see if the URI matches any of the registered routes. (TODO)
		a, err := r.Routes.match(newroute)
		if err != nil {
			log.Print("NOTFOUND", err, newroute) // DEBUG
			if err == ErrNotFound {
				r.outlet.Element().Root().Set("navigation", "notfound", Bool(true))
				return false
			}
			if err == ErrUnauthorized {
				log.Print("unauthorized for: " + newroute) //DEBUG
				r.outlet.Element().Root().Set("navigation", "unauthorized", Bool(true))
				return false
			}
			if err == ErrFrameworkFailure {
				log.Print("APPFAILURE: ", err) // DEBUG
				r.outlet.Element().Root().Set("navigation", "appfailure", Bool(true))
				return false
			}
		}
		err = a()
		if err != nil {
			log.Print("UNACTIVABLE: ", err) // DEBUG
			log.Print("unauthorized for: " + newroute)
			r.outlet.Element().Root().Set("navigation", "unauthorized", Bool(true))
			return false
		}

		r.outlet.Element().Root().SyncUISetData("currentroute", evt.NewValue())
		r.History.Push(newroute)
		log.Println(*r.History) //DEBUG
		return false
	})
	return mh
}

// redirecthandler returns a mutation handler which deals with route redirections.
func (r *Router) redirecthandler() *MutationHandler {
	mh := NewMutationHandler(func(evt MutationEvent) bool {
		nroute, ok := evt.NewValue().(String)
		if !ok {
			log.Print("route mutation has wrong type... something must be wrong", evt.NewValue())
			r.outlet.Element().Root().Set("navigation", "appfailure", Bool(true))
			return true
		}
		newroute := string(nroute)
		if !r.LeaveTrailingSlash {
			newroute = strings.TrimSuffix(newroute, "/")
		}
		newroute = strings.TrimPrefix(newroute, r.BaseURL)

		// 1. Let's see if the URI matches any of the registered routes. (TODO)
		a, err := r.Routes.match(newroute)
		if err != nil {
			log.Print(err, newroute) // DEBUG
			if err == ErrNotFound {
				r.outlet.Element().Root().Set("navigation", "notfound", Bool(true))
				return false
			}
			if err == ErrUnauthorized {
				log.Print("unauthorized for: " + newroute) //DEBUG
				r.outlet.Element().Root().Set("navigation", "unauthorized", Bool(true))
				return false
			}
			if err == ErrFrameworkFailure {
				log.Print(err) //DEBUG
				r.outlet.Element().Root().Set("navigation", "appfailure", Bool(true))
				return false
			}
		}
		err = a()
		if err != nil {
			log.Print(err) // DEBUG
			log.Print("unauthorized for: " + newroute)
			r.outlet.Element().Root().Set("navigation", "unauthorized", Bool(true))
			return false
		}

		r.outlet.Element().Root().SyncUISetData("redirectroute", evt.NewValue())
		r.History.Push(newroute)
		log.Println(*r.History) //DEBUG
		return false
	})
	return mh
}

// OnRoutechangeRequest allows to trigger a mutation handler before a route change
// is effective. It needs to be called before ListenAndServe. Returning true should
// cancel the current routechangerequest. (enables hijacking of the route change process)
func (r *Router) OnRoutechangeRequest(m *MutationHandler) {
	r.outlet.Element().Root().Watch("navigation", "routechangerequest", r.outlet.Element().Root(), m)
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
	root := r.outlet

	// Let's make sure that all the mounted views have been registered.
	v, ok := r.outlet.Element().Global.Get("internals", "views")
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

	routeChangeHandler := NewEventHandler(func(evt Event) bool {
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
	root.Element().Root().Watch("navigation", "routeredirectrequest", root.Element().Root(), r.redirecthandler())
	r.outlet.Element().Root().Set("navigation", "ready", Bool(true))
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

func (r *rnode) ID() string {
	return r.ViewElement.Element().ID
}

func newchildrnode(v ViewElement, root *rnode) *rnode {
	m := make(map[string]map[string]*rnode)
	for k := range v.Element().InactiveViews {
		m[k] = nil
	}
	if a := v.Element().ActiveView; a != "" {
		m[a] = nil
	}
	return &rnode{root, v.Element().ID, v, m}
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

	if nrn.ID() == rn.root.ID() {
		for k := range v.Element().InactiveViews {
			rn.next[k] = nil
		}
		if a := v.Element().ActiveView; a != "" {
			rn.next[a] = nil
		}
		return
	}

	viewpath := v.Element().ViewAccessPath
	if viewpath == nil {
		return
	}
	viewpathnodes := viewpath.Nodes
	var ancestor *Element
	if len(viewpathnodes) == 0 {
		return
	} else {
		ancestor = viewpathnodes[0].Element
	}
	if ancestor.ID != rn.root.ViewElement.Element().ID {
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
			refnode.attach(node.Name, nr)
			refnode = nr
			viewname = node.Name
		}
	}
	refnode.attach(viewname, nrn)
}

// attach links to rnodes that corresponds to viewElements that succeeds each other
func (r *rnode) attach(targetviewname string, nr *rnode) {
	log.Println("ATTACH: ", targetviewname, nr) // DEBUG
	m, ok := r.next[targetviewname]
	if !ok {
		m = make(map[string]*rnode)
		r.next[targetviewname] = m
	}
	if m == nil {
		m = make(map[string]*rnode)
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
	segments := strings.Split(route, "/")
	log.Print(segments) //DEBUG
	ls := len(segments)
	if ls == 0 {
		return nil, nil
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
		} else {
			return nil, ErrNotFound
		}
	}

	// Do other children views need activation? Let's check for it.
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
		log.Print("Not the right number of segments")
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

	e := NewElement(viewname, target.Element().ID+"/"+viewname, r.outlet.Element().DocType)
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
	return &NavHistory{make([]string, 0, 300), -1}
}

func (n *NavHistory) Push(URI string) *NavHistory {
	n.Cursor++
	n.Stack = append(n.Stack[:n.Cursor], URI)

	return n
}

func (n *NavHistory) Replace(URI string) *NavHistory {
	n.Stack[n.Cursor] = URI
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
