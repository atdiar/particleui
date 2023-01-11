// Package ui is a library of functions for simple, generic gui development.
package ui

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"
)

var (
	ErrNotFound         = errors.New("Not Found")
	ErrUnauthorized     = errors.New("Unauthorized")
	ErrFrameworkFailure = errors.New("Framework Failure")
)
// NavContext holds the navigation context. It is cancelled before being reinitialized on 
// Each new navigation start.
var NavContext, CancelNav = newCancelableNavContext()

func newCancelableNavContext()(context.Context, context.CancelFunc){
	return  context.WithCancel(context.Background())
}

// router is a singleton as only one router can be created at a time.
// It can be retrieved by a call to GetRouter
var router *Router

// GetRouter returns the application Router object if it has been created.
// If it has not yet, it panics.
// Henceforth, it is only safe to call this function 
func GetRouter() *Router {
	if router == nil {
		panic("FAILURE: trying to retrieve router before it has been created.")
	}
	return router
}

// UseRouter is a convenience function that allows for an Element to call a
// router-using function when mounted.
func UseRouter(user AnyElement, fn func(*Router)) {
	h := NewMutationHandler(func(evt MutationEvent) bool {
		evt.Origin().Watch("event","initrouter",evt.Origin().AsElement().Root(),NewMutationHandler(func(evt MutationEvent)bool{
			fn(GetRouter())
			return false
		}).RunASAP())
		return false
	}).RunASAP().RunOnce()
	user.AsElement().OnMounted(h)
}

// Router stores shortcuts to given states of the application.
// These shortcuts take the form of URIs.
// The router is also in charge of modifying the application state to reach any
// state registered as a shortcut upon request.
type Router struct {
	Outlet   ViewElement

	Links map[string]Link

	Routes *rnode

	History *NavHistory

	LeaveTrailingSlash bool
}

// NewRouter takes an Element object which should be the entry point of the router.
// By default, the router basepath is initialized to "/".
func NewRouter(rootview ViewElement, options ...func(*Router)*Router) *Router {
	if router != nil {
		panic("A router has already been created")
	}
	if !rootview.AsElement().Mountable() {
		panic("router can only use a view attached to the main tree as a navigation Outlet.")
	}

	r := &Router{ rootview, make(map[string]Link, 300), newrootrnode(rootview), NewNavigationHistory(), false}

	r.Outlet.AsElement().Root().Watch("event", "docupdate", r.Outlet.AsElement().Root(), NewMutationHandler(func(evt MutationEvent) bool {
		_, navready := r.Outlet.AsElement().Root().Get("navigation", "ready")
		if !navready {
			v, ok := r.Outlet.AsElement().Global.Get("internals", "views")
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

	r.Outlet.AsElement().Root().Watch("event","navigationstart",r.Outlet.AsElement().Root(),NewMutationHandler(func(evt MutationEvent)bool{
		CancelNav()
		NavContext,CancelNav = newCancelableNavContext()
		// TODO if state is being replayed, cancelnav
		return false
	}))

	r.Outlet.AsElement().ElementStore.NewConstructor("pui_link", func(id string)*Element{
		e:= NewElement(id,r.Outlet.AsElement().ElementStore.DocType)
		return e
	})

	for _,option:= range options{
		r = option(r)
	}

	router = r
	r.Outlet.AsElement().Root().Set("event","initrouter",Bool(true))
	return r
}

func (r *Router) tryNavigate(newroute string) bool {
	// 0. Retrieve hash if it exists
	route,hash,found:= strings.Cut(newroute,"#")
	if found{
		newroute = route
	}

	// 1. Let's see if the URI matches any of the registered routes.
	v,_,a, err := r.Routes.match(newroute)
	r.Outlet.AsElement().Root().Set("navigation", "targetview", v.AsElement())
	if err != nil {
		log.Print(err) // DEBUG
		if err == ErrNotFound {
			log.Print("this is strange", err) // DEBUG
			r.Outlet.AsElement().Root().Set("navigation", "notfound", String(newroute))
			return false
		}
		if err == ErrUnauthorized {
			log.Print(err) // DEBUG
			r.Outlet.AsElement().Root().Set("navigation", "unauthorized", String(newroute))
			return false
		}
		if err == ErrFrameworkFailure {
			log.Print(err) //DEBUG
			r.Outlet.AsElement().Root().Set("navigation", "appfailure", String(newroute))
			return false
		}
	}
	err = a()
	if err != nil {
		log.Print("activation failure", err) // DEBUG
		r.Outlet.AsElement().Root().Set("navigation", "unauthorized", String(newroute))
		return false
	}
	if found{
		r.Outlet.AsElement().Root().Set("navigation","hash",String(hash))
	}
	return true
}

// GoTo changes the application state by updating the current route
// To make sure that the route provided as argument exists, use the match method.
func (r *Router) GoTo(route string) {
	if !r.LeaveTrailingSlash {
		route = strings.TrimSuffix(route, "/")
	}

	r.History.Push(route)
	r.Outlet.AsElement().Root().SetDataSetUI("currentroute", String(route))
	r.Outlet.AsElement().Root().SetDataSetUI("history", r.History.Value())

	
	r.Outlet.AsElement().Root().Set("event", "navigationstart", String(route))

	ok := r.tryNavigate(route)
	if !ok {
		DEBUG("NAVIGATION FAILED FOR SOME REASON.") // DEBUG
	}

	r.Outlet.AsElement().Root().Set("event", "navigationend", String(route))
	
	
}

func (r *Router) GoBack() {
	if r.History.BackAllowed() {
		r.Outlet.AsElement().Root().Set("navigation", "routechangerequest", String(r.History.Back()))
	}
}

func (r *Router) GoForward() {
	if r.History.ForwardAllowed() {
		r.Outlet.AsElement().Root().Set("navigation", "routechangerequest", String(r.History.Forward()))
	}
}

// RedirectTo can be used to trigger route redirection.
func (r *Router) RedirectTo(route string) {
	r.Outlet.AsElement().Root().Set("navigation", "routeredirectrequest", String(route))
}

// Hijack short-circuits navigation to create a redirection rule for a specific route to an alternate 
// destination.
func (r *Router) Hijack(route string, destination string) {
	r.OnRoutechangeRequest(NewMutationHandler(func(evt MutationEvent) bool {
		navroute := evt.NewValue().(String)
		if string(navroute) == route {
			//r.History.Push(route)
			r.Outlet.AsElement().Root().Set("navigation", "routechangerequest", String(destination))
			return true
		}
		return false
	}))
}

// Hijack is a router modifier function that allows the router to redirect navigation to a specific route
// to a different  destination.
func Hijack(route, destination string) func(*Router)*Router{
	return func(r *Router) *Router{
		r.Hijack(route,destination)
		return r
	}
}

// OnNotfound reacts to the navigation 'notfound' property being set. It can enable the display of 
// a "page not found" view.
// It is not advised to navigate here. It is better to represent the app error state directly.
func (r *Router) OnNotfound(h *MutationHandler) *Router {
	r.Outlet.AsElement().Root().Watch("navigation", "notfound", r.Outlet.AsElement().Root(), h)
	return r
}

// OnUnauthorized reacts to the navigation state being set to unauthorized.
// It may occur when there are insufficient rights to displaya given view for instance.
// It is not advised to navigate here. It is better to represent the app error state directly.
func (r *Router) OnUnauthorized(h *MutationHandler) *Router {
	r.Outlet.AsElement().Root().Watch("navigation", "unauthorized", r.Outlet.AsElement().Root(), h)
	return r
}

// OnAppfailure reacts to the navigation state being set to "appfailure".
// It may occur when a malfunction occured.
// The MutationHandler informs of the behavior to addopt in this case.
func (r *Router) OnAppfailure(h *MutationHandler) *Router {
	r.Outlet.AsElement().Root().Watch("navigation", "appfailure", r.Outlet.AsElement().Root(), h)
	return r
}

func (r *Router) insert(v ViewElement) {
	nrn := newchildrnode(v, r.Routes)
	r.Routes.insert(nrn)
}

// Match returns whether a route is valid or not. It can be used in tests to
// Make sure that app links are not breaking.
func (r *Router) Match(route string) (prefetch func(),err error) {
	_,p,_, err := r.Routes.match(route)
	return p, err

}

// handler returns a mutation handler which deals with route change.
func (r *Router) handler() *MutationHandler {
	mh := NewMutationHandler(func(evt MutationEvent) bool {
		nroute, ok := evt.NewValue().(String)
		if !ok {
			log.Print("route mutation has wrong type... something must be wrong", evt.NewValue())
			r.Outlet.AsElement().Root().Set("navigation", "appfailure", Bool(true))
			return true
		}
		newroute := string(nroute)
		if !r.LeaveTrailingSlash {
			newroute = strings.TrimSuffix(newroute, "/")
		}

		// Retrieve hash if it exists
		route,hash,found:= strings.Cut(newroute,"#")
		if found{
			newroute = route
		}

		// Determination of navigation history action 
		h, ok := r.Outlet.AsElement().Root().Get("data", "history")
		if !ok {
			r.History.Push(newroute)
		} else {
			ho, ok := h.(Object)
			if !ok {
				panic("history object of wrong type")
			}
			v, ok := ho.Get("cursor")
			if !ok {
				panic("unable to retrieve history object cursor value")
			}
			n := int(v.(Number))
			cursor:= r.History.Cursor


			if r.History.Cursor > n {
				
				// we are going back
				//DEBUG("back from: ",r.History.Cursor, " to ",n)
				for i := 0; i < cursor-n; i++ {
					r.History.Back()
				}
			} else if r.History.Cursor < n {
				r.History.ImportState(h)
				for i := 0; i < n-cursor; i++ {
					r.History.Forward()
				}

			} else{
				//DEBUG("from: ",cursor, " to ",n, "by importing state")
				r.History.ImportState(h)
			}

			
		}
		r.Outlet.AsElement().Root().SetDataSetUI("currentroute", String(newroute))
		r.Outlet.AsElement().Root().SetDataSetUI("history", r.History.Value())

		// Let's see if the URI matches any of the registered routes. (TODO)
		v,_,a, err := r.Routes.match(newroute)
		r.Outlet.AsElement().Root().Set("navigation", "targetview", v.AsElement())
		if err != nil {
			log.Print("NOTFOUND", err, newroute) // DEBUG
			if err == ErrNotFound {
				r.Outlet.AsElement().Root().Set("navigation", "notfound", String(newroute))
			}
			if err == ErrUnauthorized {
				log.Print("unauthorized for: " + newroute) //DEBUG
				r.Outlet.AsElement().Root().Set("navigation", "unauthorized", String(newroute))
			}
			if err == ErrFrameworkFailure {
				log.Print("APPFAILURE: ", err) // DEBUG
				r.Outlet.AsElement().Root().Set("navigation", "appfailure", String(newroute))
			}
		} else{
			r.Outlet.AsElement().Root().Set("event", "navigationstart", String(newroute))
			err = a()
			if err != nil {
				r.Outlet.AsElement().Root().Set("navigation", "unauthorized", String(newroute))
				DEBUG("activation failure",err)
			}

			if found{
				r.Outlet.AsElement().Root().Set("navigation","hash",String(hash))
			}
		}
		

		

		r.Outlet.AsElement().Root().Set("event", "navigationend", String(newroute))
		

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
			r.Outlet.AsElement().Root().Set("navigation", "appfailure", Bool(true))
			return true
		}
		newroute := string(nroute)
		if !r.LeaveTrailingSlash {
			newroute = strings.TrimSuffix(newroute, "/")
		}

		// Retrieve hash if it exists
		route,hash,found:= strings.Cut(newroute,"#")
		if found{
			newroute = route
		}

		r.History.Replace(newroute)
		r.Outlet.AsElement().Root().SetDataSetUI("currentroute", String(newroute))
		r.Outlet.AsElement().Root().SetDataSetUI("history", r.History.Value())

		// 1. Let's see if the URI matches any of the registered routes.
		v,_, a, err := r.Routes.match(newroute)
		r.Outlet.AsElement().Root().Set("navigation", "targetview", v.AsElement())
		if err != nil {
			log.Print(err, newroute) // DEBUG
			if err == ErrNotFound {
				r.Outlet.AsElement().Root().Set("navigation", "notfound", String(newroute))
			}
			if err == ErrUnauthorized {
				log.Print("unauthorized for: " + newroute) //DEBUG
				r.Outlet.AsElement().Root().Set("navigation", "unauthorized", String(newroute))
			}
			if err == ErrFrameworkFailure {
				log.Print(err) //DEBUG
				r.Outlet.AsElement().Root().Set("navigation", "appfailure", String(newroute))
			}
		} else{
			r.Outlet.AsElement().Root().Set("event", "navigationstart", String(newroute))
			err = a()
			if err != nil {
				log.Print(err) // DEBUG
				log.Print("unauthorized for: " + newroute)
				r.Outlet.AsElement().Root().Set("navigation", "unauthorized", String(newroute))
			}

			if found{
				r.Outlet.AsElement().Root().Set("navigation","hash",String(hash))
			}
		}

		
		
		r.Outlet.AsElement().Root().Set("event", "navigationend", String(newroute))

		
		

		//DEBUG("redirect ",*r.History)

		return false
	})
	return mh
}

// OnRoutechangeRequest allows to trigger a mutation handler before a route change
// is effective. It needs to be called before ListenAndServe. Returning true should
// cancel the current routechangerequest. (enables hijacking of the route change process)
func (r *Router) OnRoutechangeRequest(m *MutationHandler) {
	r.Outlet.AsElement().Root().Watch("navigation", "routechangerequest", r.Outlet.AsElement().Root(), m)
}

// ListenAndServe registers a listener for route change.
// It should only be called after the app structure has been fully built.
// It listens on the element that receives routechangeevent (first argument)
// 
// For Implementers
//
// The event value should be of type Object. The event value should be registered on this object
// for the "value" key. This value should be of type String.
//
//
// Example of JS bridging : the nativeEventBridge should add a popstate event listener to window
// It should also dispatch a RouteChangeEvent to bridge browser url mutation into the Go side
// after receiving notice of popstate event firing.
func (r *Router) ListenAndServe(events string, target *Element) {
	r.verifyLinkActivation()
	root := r.Outlet

	// Let's make sure that all the mounted views have been registered.
	v, ok := r.Outlet.AsElement().Global.Get("internals", "views")
	if ok {
		l, ok := v.(List)
		if ok {
			for _, val := range l {
				viewEl, ok := val.(*Element)
				if !ok || !viewEl.isViewElement() {
					panic("internals/views does not hold a proper Element")
				}
				if viewEl.Mountable() {
					r.insert(ViewElement{viewEl})
				}
			}
		}
	}

	routeChangeHandler := NewEventHandler(func(evt Event) bool {
		u,ok:= evt.Value().(Object).Get("value")
		if !ok{
			panic("framework error: event value format unexpected. Should have a value field")
		}
		root.AsElement().Root().Set("navigation", "routechangerequest", u.(String))
		return false
	})

	root.AsElement().Root().Watch("navigation", "routechangerequest", root.AsElement().Root(), r.handler())
	root.AsElement().Root().Watch("navigation", "routeredirectrequest", root.AsElement().Root(), r.redirecthandler())
	r.Outlet.AsElement().Root().Set("navigation", "ready", Bool(true))
	
	eventnames:= strings.Split(events," ")
	for _,event:= range eventnames{
		target.AddEventListener(event, routeChangeHandler)
	}
	
	for {
		select{
		case f:= <-WorkQueue:
			f()
		}
	}
}

func (r *Router) verifyLinkActivation() {
	for _, l := range r.Links {
		_, ok := l.Raw.Get("internals", "verified")
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
	return r.ViewElement.AsElement().ID
}

func (r *rnode) update() *rnode {
	for k := range r.ViewElement.AsElement().InactiveViews {
		m, ok := r.next[k]
		if !ok {
			m = make(map[string]*rnode)
			r.next[k] = m
		}
	}
	a := r.ViewElement.AsElement().ActiveView
	m, ok := r.next[a]
	if !ok && a != "" {
		m = make(map[string]*rnode)
		r.next[a] = m
	}
	return r
}

func newchildrnode(v ViewElement, root *rnode) *rnode {
	m := make(map[string]map[string]*rnode)
	for k := range v.AsElement().InactiveViews {
		m[k] = make(map[string]*rnode)
	}
	if a := v.AsElement().ActiveView; a != "" {
		m[a] = make(map[string]*rnode)
	}
	r := &rnode{root, v.AsElement().ID, v, m}
	r.update()
	return r
}

func newrootrnode(v ViewElement) *rnode {
	r := newchildrnode(v, nil)
	r.root = r
	return r
}

// insert  adds an arbitrary rnode to the rnode trie if  possible (the root
// ViewElement of the rnode ViewAccessPath should be that of the root rnode )
func (rn *rnode) insert(nrn *rnode) {
	if rn != rn.root {
		panic("only root rnode can call insert")
	}
	v := nrn.ViewElement

	if nrn.ID() == rn.root.ID() {
		rn.root.update()
		return
	}
	viewpath := computePath(newViewNodes(), v.AsElement().ViewAccessNode)
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
	if ancestor.ID != rn.root.ViewElement.AsElement().ID {
		log.Print("Houston, we have a problem. Everything shall start from rnode toot ViewElement")
		return
	}
	l := len(viewpathnodes)
	// attach iteratively the rnodes
	refnode := rn
	viewname := viewpathnodes[0].Name
	log.Println(viewpathnodes, l, rn)
	for i, node := range viewpathnodes {
		if i+1 < l {
			// each ViewElement should be turned into a *rnode and should be attached in succession. The end node is our argument.
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
	r.update()
	m, ok := r.next[targetviewname]
	if !ok {
		m = make(map[string]*rnode)
		r.next[targetviewname] = m
	}
	_, ok = m[nr.ViewElement.AsElement().ID]
	if !ok {
		m[nr.ViewElement.AsElement().ID] = nr
	} // else it has already been attached
	nr.update()
}

// match verifies that a route passed as arguments corresponds to a given view state.
func (r *rnode) match(route string) (targetview ViewElement, prefetchFn func(), activationFn func() error, err error) {
	activations := make([]func() error, 0, 10)
	prefetchers := make([]func(),0,10)
	route = strings.TrimPrefix(route, "/")
	segments := strings.Split(route, "/")
	ls := len(segments)
	targetview = r.ViewElement // DEBUG TODO is it the true targetview? 
	if ls == 0 {
		return targetview,nil, nil,nil
	}

	var param string

	m, ok := r.next[segments[0]] // 0 is the index of the viewname at the root ViewElement m is of type map[string]*rnode
	if !ok {
		// Let's see if the ViewElement has a parameterizable view
		param, ok = r.ViewElement.hasParameterizedView()
		if ok {
			if !r.ViewElement.isViewAuthorized(param) {
				return targetview,nil, nil, ErrUnauthorized
			}
			if ls != 1 { // we get the next rnodes mapped by viewname
				m, ok = r.next[param]
				if !ok {
					return targetview,nil, nil, ErrFrameworkFailure
				}
			}
		} else {
			return targetview,nil,nil, ErrNotFound
		}
	}

	// Do other children views need activation? Let's check for it.
	if ls >= 1 && ls%2 == 1 {
		// check authorization
		if param != "" {
			if r.ViewElement.isViewAuthorized(param) {
				a := func() error {
					return r.ViewElement.ActivateView(segments[0])
				}
				activations = append(activations, a)

				p:= func(){
					r.ViewElement.AsElement().Prefetch()
				}
				prefetchers = append(prefetchers,p)

			} else {
				return targetview,nil,nil, ErrUnauthorized
			}
		} else {
			if r.ViewElement.isViewAuthorized(segments[0]) {
				a := func() error {
					return r.ViewElement.ActivateView(segments[0])
				}
				activations = append(activations, a)

				p:= func(){
					r.ViewElement.AsElement().Prefetch()
					r.ViewElement.prefetchView(segments[0])
				}
				prefetchers = append(prefetchers,p)
			} else {
				return targetview,nil, nil, ErrUnauthorized
			}
		}
	}

	if ls%2 != 1 {
		DEBUG("Incorrect URI scheme")
		return targetview,nil,nil, ErrNotFound
	}
	if ls > 1 {
		viewcount := (ls - ls%2) / 2

		// Let's get the next rnode and check that the view mentionned in the route exists (segment[2i+2])

		for i := 1; i <= viewcount; i++ {
			routesegment := segments[2*i-1]   //ids
			nextroutesegment := segments[2*i] //viewnames
			r, ok := m[routesegment]
			if !ok {
				return targetview,nil, nil, ErrNotFound
			}

			targetview = r.ViewElement
			if r.value != routesegment {
				return targetview,nil,nil, ErrNotFound
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
						return targetview,nil, nil, ErrUnauthorized
					}

					m, ok = r.next[param] // we get the next rnodes mapped by viewnames
					if !ok {
						return targetview,nil,nil, ErrFrameworkFailure
					}

				} else {
					return targetview,nil,nil, ErrNotFound
				}
			}
			if !r.ViewElement.isViewAuthorized(nextroutesegment) {
				return targetview,nil, nil, ErrUnauthorized
			}
			a := func() error {
				return r.ViewElement.ActivateView(nextroutesegment)
			}
			activations = append(activations, a)

			if !ok{
				p:= func(){
					r.ViewElement.AsElement().Prefetch()
				}
				prefetchers = append(prefetchers,p)
			} else{
				p:= func(){
					r.ViewElement.AsElement().Prefetch()
					r.ViewElement.prefetchView(nextroutesegment)
				}
				prefetchers = append(prefetchers,p)
			}
		}
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

	prefetchFn = func(){
		for _, p:= range prefetchers{
			p()
		}
	}
	return targetview,prefetchFn, activationFn, nil
}

/*

	Navigation link creation

*/

// Link holds the representation (under the form of an URI) of the application state
// required for the target View to be available for display on screen.
// A link can be watched.
type Link struct {
	Raw *Element
}

func (l Link) URI() string {
	u,_:= l.Raw.GetData("uri")
	uri:= string(u.(String))
	return uri
}

func (l Link) Activate(targetid ...string) {
	if len(targetid) == 1{
		if targetid[0] != ""{
			l.Raw.Set("event","activate", String(targetid[0]))
			return 
		}
	}
	l.Raw.Set("event","activate", Bool(true))
}

func (l Link) IsActive() bool {
	status, ok := l.Raw.GetData("active")
	if !ok {
		return false
	}
	st, ok := status.(Bool)
	if !ok {
		panic("wrong type for link validation IsActive predicate value")
	}
	return bool(st)
}

func(l Link) Prefetch(){
	l.Raw.Set("event","prefetchlink",Bool(true))
}

func (l Link) AsElement() *Element {
	return l.Raw
}

func (l Link) watchable() {}

// NewLink returns a Link object.
//
// The first argument is the name of the view to be activated for the main ViewElement. (starting
// point of the router, aka router outlet)
// It generates an URI similar to /viewname.
//
// It then accepts Link modifying functions that allow for example to further specify a path to 
// into  a nested ViewElement. (See Path function)
// The URI generated will see path fragments concatenated to it such as:
// /viewname/nestdeviewElementA/nestedviewname2...
//
// Note that such a Link object does not offer any guarantees on its validity.
// However, link creation is verified at app startup and invalid links should trigger a panic.
// (for links created dynamically during runtime, there is no check)
func (r *Router) NewLink(viewname string, modifiers ...func(Link)Link) Link {
	// If previously created, it has been memoized. let's retrieve it then. otherwise,
	// let's create it.

	if isParameter(viewname){
		panic(viewname + " is not a valid view name.")
	}
	
	l,ok:= r.Links["/"+viewname]
	if !ok{
		// Let's retrieve the link constructor
		c,ok:= r.Outlet.AsElement().ElementStore.Constructors["pui_link"]
		if !ok{
			panic("pui_ERROR: somehow the link constructor has not been regristered.")
		}
		e := c(r.Outlet.AsElement().ID+"-"+viewname)
		e.SetData("viewelements",NewList(r.Outlet.AsElement()))
		e.SetData("viewnames", NewList(String(viewname)))
		e.SetData("uri", String("/"+viewname))
		l= Link{e}
	}
	
	for _,m:= range modifiers{
		l = m(l)
	}
	

	ll, ok:= r.Links[l.URI()]
	if ok {
		return ll
	}
	e:= l.AsElement()

	// Let's retrieve the target viewElement and corresponding view name
	v,ok:= e.GetData("viewelements")
	if !ok{
		panic("Link creation seems to be incomplete. The list of viewElements for the path it denotes should be present.")
	}
	/*n,ok:= e.GetData("viewnames")
	if !ok{
		panic("Link creation seems to be incomplete. The list of viewnames for the path it denotes should be present.")
	}*/
	vl:= v.(List)
	//nl:= n.(List)
	view:= ViewElement{vl[len(vl)-1].(*Element)}
	//viewname = string(nl[len(nl)-1].(String))



	nh := NewMutationHandler(func(evt MutationEvent) bool {
		if isValidLink(Link{e}){
			_, ok := e.Get("internals", "verified")
			if !ok {
				e.Set("internals", "verified", Bool(true))
			}
		}

		return false
	})
	e.Watch("event", "mountable", view.AsElement(), NewMutationHandler(func(evt MutationEvent) bool {
		b := evt.NewValue().(Bool)
		if !b {
			return false
		}
		return nh.Handle(evt)
	}).RunASAP())
	e.Watch("data", "currentroute", r.Outlet.AsElement().Root(), NewMutationHandler(func(evt MutationEvent) bool {
		route := evt.NewValue().(String)
		lnk,_:= e.GetData("uri")
		link:= strings.TrimPrefix(string(lnk.(String)),"/")
		
		if string(route) == link {
			e.SyncUISetData("active", Bool(true))
		} else {
			e.SyncUISetData("active", Bool(false))
		}

		return false
	}))

	r.Links[l.URI()] = l

	r.Outlet.AsElement().Watch("event","activate",e,NewMutationHandler(func(evt MutationEvent)bool{
		var hash string
		if s,ok:= evt.NewValue().(String);ok{
			hash = "#"+string(s)
		}
		r.GoTo(l.URI()+hash)
		return false
	}))

	r.Outlet.AsElement().Watch("event","prefetchlink",e,NewMutationHandler(func(evt MutationEvent)bool{
		p,err:= r.Match(l.URI())
		if err!= nil{
			return true
		}
		p()
		return false
	}))


	return l
}

// Path is a link modifying function that allows to link to a more deeply nested app state,
// specified by the nested ViewElement and the corresponding name for the view that the latter 
// should display. This creates a path fragment.
//
// Note that if the link being modified does not target a direct parent of the Path fragment ViewELement,
// the link will be invalid.
// Hence, it is not possible to skip an intermediary path fragment or add them out-of-order.
func Path(ve ViewElement, viewname string) func(Link)Link{
	if isParameter(viewname){
		panic(viewname + " is not a valid view name.")
	}
	return func(l Link)Link{
		e:= l.AsElement()
		ne:= NewElement(ve.AsElement().ID+"-"+viewname, e.DocType)

		v,ok:=e.GetData("viewelements")
		if !ok{
			panic("Link creation seems to be incomplete. The list of viewElements for the path it denotes should be present.")
		}
		n,ok:= e.GetData("viewnames")
		if !ok{
			panic("Link creation seems to be incomplete. The list of viewnames for the path it denotes should be present.")
		}
		vl:= v.(List)
		nl:= n.(List)
		vl = append(vl,ve.AsElement())
		nl = append(nl,String(viewname))
		ne.SetData("viewelements",vl)
		ne.SetData("viewnames",nl)
		uri:="/" + string(nl[0].(String))
		for i,velem:= range vl{
			if i==0{
				continue
			}
			id:= velem.(*Element).ID
			vname:= string(nl[i].(String))
			uri = "/" + id + "/" + vname
		}
		ne.SetData("uri",String(uri))
		return Link{ne}
	}
}


func(r *Router) RetrieveLink(URI string) (Link,bool){
	l, ok := r.Links[URI]
	return l,ok	
}


func isValidLink(l Link) bool{
	e:= l.AsElement()
	v,ok:=e.GetData("viewelements")
	if !ok{
		return false
	}
	n,ok:= e.GetData("viewnames")
	if !ok{
		return false
	}
	vl:= v.(List)
	nl:= n.(List)

	targetview:= vl[len(vl)-1].(*Element)
	viewname := string(nl[len(nl)-1].(String))

	vap:= targetview.ViewAccessPath.Nodes
	if len(vap) != len(vl)-1{
		DEBUG("viewaccespath and link depth do not match. Some view might have been skipped")
		return false
	}
	for i,n:= range vap{
		vnode:= vl[i].(*Element)
		if vnode.ID != n.Element.ID{
			return false
		}
		vname:= string(nl[i].(String))
		if !hasView(ViewElement{vnode},vname){
			return false
		}
	}
	return hasView(ViewElement{targetview},viewname)
}

func hasView(v ViewElement, vname string)bool{
	if _,ok:=v.hasParameterizedView();ok{
		return true
	}
	return  v.hasStaticView(vname)
}

/*

   Navigation History

*/

// NavHistory holds the Navigation History. (aka NavStack)
type NavHistory struct {
	Stack  []string
	State  []Observable
	Cursor int
	NewState func(id string) Observable
	RecoverState func(Observable) Observable
	Length int
}

// Get is used to retrieve a Value from the history state.
func (n *NavHistory) Get(category, propname string) (Value, bool) {
	return n.State[n.Cursor].Get(category, propname)
}

// Set is used to insert a value in the history state.
func (n *NavHistory) Set(category string, propname string, val Value) {
	n.State[n.Cursor].Set(category, propname, val)
}

func NewNavigationHistory() *NavHistory {
	n:= &NavHistory{}
	n.Stack = make([]string, 0, 1024)
	n.State = make([]Observable, 0, 1024)
	n.Cursor = -1
	n.NewState = func(id string) Observable{
		return newObservable(id)
	}
	n.RecoverState = func(o Observable)Observable{return o}
	n.Length = 1024
	return n
}
func (n *NavHistory) Value() Value {
	o := NewObject()
	o.Set("cursor", Number(n.Cursor))

	// Prepare Stack for serialization
	stack:=make([]Value,len(n.Stack))
	for i,entry:= range n.Stack{
		stack[i]= String(entry)
	}
	o.Set("stack",List(stack))

	// Prepare State for serialization
	state:=make([]Value,len(n.State))
	for i,entry:= range n.State{
		state[i]= entry.AsElement()
	}
	o.Set("state",List(state))

	return o
}

// Note that router state may be synchronized/persisted only on page change at times.
// In which case, on app reload, the latest nav history state may be lost if no persistence
// was forced beforehand.
// Hence, one should be careful before storing app state in the framework history object, 
// depending on the implementation. (some state object could be persisted for each mutation)
func(n *NavHistory) ImportState(v Value) *NavHistory{
	h,ok:= v.(Object)
	if !ok{
		return n
	
	}
	stk:= h["stack"]
	stack:= stk.(List)
	hlen:= len(stack)

	stt:= h["state"]
	state:= stt.(List)

	if hlen>len(n.Stack){
		for i:=n.Cursor+1; i<hlen;i++{
			entry:= stack[i]
			nexturl:= entry.(String)
			n.Stack=append(n.Stack,string(nexturl))

			stentry:= state[i]
			stateObj:= stentry.(*Element)
			n.State = append(n.State, n.RecoverState(Observable{stateObj}))
		}
	}
	
	return n
}

func (n *NavHistory) Push(URI string) *NavHistory {
	if len(n.Stack) >= n.Length {
		panic("navstack capacity overflow")
	}
	if n.Cursor>=0{
		n.State[n.Cursor].Set("internals","new", Bool(false)) // used to discover whether the current navigation entry is accessed for the first time or not
	}
	n.Cursor++
	n.Stack = append(n.Stack[:n.Cursor], URI)
	n.State = append(n.State[:n.Cursor], n.NewState("hstate"+strconv.Itoa(n.Cursor)))
	n.State[n.Cursor].Set("internals","new", Bool(true))
	
	return n
}

func(n *NavHistory) CurrentEntryIsNew() bool{
	v,ok:= n.State[n.Cursor].Get("internals","new")
	if !ok{
		DEBUG("Unable to find (internals, new) cursor is : ",n.Cursor)
		DEBUG(n.Stack,n.State)
		return true
	}
	return bool(v.(Bool))
}

func (n *NavHistory) Replace(URI string) *NavHistory {
	n.Stack[n.Cursor] = URI
	v,ok:= n.State[n.Cursor].Get("internals","new")
	n.State[n.Cursor] = n.NewState("hstate"+strconv.Itoa(n.Cursor))
	if !ok{
		n.State[n.Cursor].Set("internals","new", Bool(true))
	} else{
		n.State[n.Cursor].Set("internals","new",v)
	}
	
	// TODO what to do here? perhaps nothing, perhpas the state should be labeled new or the reverse? 
	return n
}

func (n *NavHistory) Back() string {
	if n.BackAllowed() {
		if n.Cursor == len(n.Stack)-1{
			n.State[n.Cursor].Set("internals","new", Bool(false)) // TODO should we check the value or use memoization to avoid unnecessary ops on forward()?
		}
		n.Cursor--
	}
	return n.Stack[n.Cursor]
}

func (n *NavHistory) Forward() string {
	if n.ForwardAllowed() {
		if n.Cursor>=0{
			n.State[n.Cursor].Set("internals","new", Bool(false))
		}
		n.Cursor++
	}
	v,ok := n.State[n.Cursor].Get("internals","new")
	if !ok{
		n.State[n.Cursor].Set("internals","new", Bool(true))
	} else{
		n.State[n.Cursor].Set("internals","new", v)
	}
	return n.Stack[n.Cursor]
}

func (n *NavHistory) BackAllowed(step ...int) bool {
	return n.Cursor > 0
}

func (n *NavHistory) ForwardAllowed(step ...int) bool {
	return n.Cursor < len(n.Stack)-1
}
