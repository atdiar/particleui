// Package ui is a library of functions for simple, generic gui development.
package ui

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
)

var (
	ErrNotFound         = errors.New("Not Found")
	ErrUnauthorized     = errors.New("Unauthorized")
	ErrFrameworkFailure = errors.New("Framework Failure")
)

func newCancelableNavContext()(context.Context, context.CancelFunc){
	return  context.WithCancel(context.Background())
}



// GetRouter returns the navigation router if it has been created for a given UI tree, referenced
// by its root *Element node.
// A navigation router is a URL scheme based way to transform the UI tree, i.e. changing its state.
// Each transformation is referenced by a given URL then.
// The URL can be seen As The Engine Of *Static Application State (UATEOSAS),
//
// *Static because dynamic parts are fetched
// If it has not yet, it panics.
func GetRouter(root AnyElement) *Router {
	e:= root.AsElement()
	if e.router == nil {
		DEBUG("FAILURE: trying to retrieve router before it has been created.")
	}
	return e.router
}

func(e *Element) OnRouterMounted(fn func(*Router)) {
	useRouter(e,fn)
}


// useRouter is a convenience function that allows for an Element to call a
// router-using function when mounted.
func useRouter(user AnyElement, fn func(*Router)) {
	h := NewMutationHandler(func(evt MutationEvent) bool {
		evt.Origin().WatchEvent("router-mounted",evt.Origin().AsElement().Root,NewMutationHandler(func(event MutationEvent)bool{
			fn(GetRouter(event.Origin()))
			return false
		}).RunASAP().RunOnce())
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

	// Navigation context
	NavContext context.Context

	// Navigation context cancellation function
	CancelNavigation context.CancelFunc

	Links map[string]Link

	Routes *rnode

	History *NavHistory

	LeaveTrailingSlash bool
}

// NewRouter takes an Element object which should be the entry point of the router.
// By default, the router basepath is initialized to "/".
func NewRouter(rootview ViewElement, options ...func(*Router)*Router) *Router {
	if rootview.AsElement().Root.router != nil {
		panic("A router has already been created")
	}
	if !rootview.AsElement().Mountable() {
		panic("router can only use a view attached to the main tree as a navigation Outlet.")
	}

	r := &Router{ rootview,nil,nil, make(map[string]Link, 300), newrootrnode(rootview), NewNavigationHistory(rootview.AsElement().Root), false}

	r.Outlet.AsElement().Root.WatchEvent("docupdate", r.Outlet.AsElement().Root, NewMutationHandler(func(evt MutationEvent) bool {
		_, navready := evt.Origin().Get("navigation", "ready")
		if !navready {
			v, ok := evt.Origin().Get("internals", "views")
			if ok {
				l, ok := v.(List)
				if ok {
					for _, val := range l.UnsafelyUnwrap() {
						viewRef, ok := val.(String)
						if !ok {
							panic("expected an Element ID string stored for this ViewElementt")
						}
						viewEl:= GetById(r.Outlet.AsElement().Root,viewRef.String())
						r.insert(ViewElement{viewEl})
					}
				}
			}
		}
		return false
	}))

	r.Outlet.AsElement().Root.WatchEvent("navigation-start",r.Outlet.AsElement().Root,NewMutationHandler(func(evt MutationEvent)bool{
		if r.CancelNavigation != nil{
			r.CancelNavigation()
		}
		
		NavContext,CancelNav := newCancelableNavContext()
		r.NavContext = NavContext
		r.CancelNavigation = CancelNav

		return false
	}))

	r.Outlet.AsElement().ElementStore.NewConstructor("zui_link", func(id string)*Element{
		e:= NewElement(id,"ROUTER")
		RegisterElement(r.Outlet.AsElement().Root,e)
		return e
	})

	for _,option:= range options{
		r = option(r)
	}

	rootview.AsElement().Root.router = r
	r.Outlet.AsElement().Root.TriggerEvent("router-mounted")
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
	r.Outlet.AsElement().Root.Set("navigation","targetviewid", String(v.AsElement().ID))
	if err != nil {
		log.Print(err) // DEBUG
		if err == ErrNotFound {
			log.Print("this is strange", err) // DEBUG
			r.Outlet.AsElement().Root.TriggerEvent("navigation-notfound", String(newroute))
			return false
		}
		if err == ErrUnauthorized {
			log.Print(err) // DEBUG
			r.Outlet.AsElement().Root.TriggerEvent("navigation-unauthorized", String(newroute))
			return false
		}
		if err == ErrFrameworkFailure {
			log.Print(err) //DEBUG
			r.Outlet.AsElement().Root.TriggerEvent("navigation-appfailure", String(newroute))
			return false
		}
	}
	err = a()
	if err != nil {
		log.Print("activation failure ", err) // DEBUG
		r.Outlet.AsElement().Root.TriggerEvent("navigation-unauthorized", String(newroute))
		return false
	}
	if found{
		r.Outlet.AsElement().Root.Set("navigation","hash",String(hash))
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
	r.Outlet.AsElement().Root.SetUI("currentroute", String(route))
	r.Outlet.AsElement().Root.SetUI("history", r.History.Value())

	
	r.Outlet.AsElement().Root.TriggerEvent("navigation-start", String(route))

	ok := r.tryNavigate(route)
	if !ok {
		DEBUG("NAVIGATION FAILED FOR SOME REASON.") // DEBUG
	}

	r.Outlet.AsElement().Root.TriggerEvent("navigation-end", String(route))
	
	
}

func (r *Router) GoBack() {
	if r.History.BackAllowed() {
		r.Outlet.AsElement().Root.TriggerEvent("navigation-routechangerequest", String(r.History.Back()))
	}
}

func (r *Router) GoForward() {
	if r.History.ForwardAllowed() {
		r.Outlet.AsElement().Root.TriggerEvent("navigation-routechangerequest", String(r.History.Forward()))
	}
}

// RedirectTo can be used to trigger route redirection.
func (r *Router) RedirectTo(route string) {
	r.Outlet.AsElement().Root.TriggerEvent("navigation-routeredirectrequest", String(route))
}

// Hijack short-circuits navigation to create a redirection rule for a specific route to an alternate 
// destination.
func (r *Router) Hijack(route string, destination string) {
	r.OnRoutechangeRequest(NewMutationHandler(func(evt MutationEvent) bool {
		navroute := evt.NewValue().(String)
		if string(navroute) == route {
			//r.History.Push(route)
			r.Outlet.AsElement().Root.TriggerEvent("navigation-routechangerequest", String(destination))
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
	r.Outlet.AsElement().Root.WatchEvent("navigation-notfound", r.Outlet.AsElement().Root, h)
	return r
}

// OnUnauthorized reacts to the navigation state being set to unauthorized.
// It may occur when there are insufficient rights to displaya given view for instance.
// It is not advised to navigate here. It is better to represent the app error state directly.
func (r *Router) OnUnauthorized(h *MutationHandler) *Router {
	r.Outlet.AsElement().Root.WatchEvent("navigation-unauthorized", r.Outlet.AsElement().Root, h)
	return r
}

// OnAppfailure reacts to the navigation state being set to "appfailure".
// It may occur when a malfunction occured.
// The MutationHandler informs of the behavior to addopt in this case.
func (r *Router) OnAppfailure(h *MutationHandler) *Router {
	r.Outlet.AsElement().Root.WatchEvent("navigation-appfailure", r.Outlet.AsElement().Root, h)
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

func (r *Router) RouteList() []string {
    var routes []string
    r.traverseRoutes(r.Routes, "", &routes)
    return routes
}

func (r *Router) traverseRoutes(node *rnode, currentPath string, routes *[]string) {
    if node == nil {
        return
    }

    // Iterate through each view in the node
    for viewName, viewMap := range node.next {
        for id, nextNode := range viewMap {
            newPath := strings.Trim(fmt.Sprintf("%s/%s/%s", currentPath, viewName, id), "/")
            if len(nextNode.next) == 0 {
                // If this is a leaf node, add the path to the routes
                *routes = append(*routes, newPath)
            } else {
                // Otherwise, continue traversing
                r.traverseRoutes(nextNode, newPath, routes)
            }
        }
    }
}

// handler returns a mutation handler which deals with route change.
func (r *Router) handler() *MutationHandler {
	mh := NewMutationHandler(func(evt MutationEvent) bool {
		nroute, ok := evt.NewValue().(String)
		if !ok {
			log.Print("route mutation has wrong type... something must be wrong", evt.NewValue())
			r.Outlet.AsElement().Root.TriggerEvent("navigation-appfailure", Bool(true))
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
		h, ok := r.Outlet.AsElement().Root.Get("data", "history")
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
				for i := 0; i < cursor-n; i++ {
					r.History.Back()
				}
			} else if r.History.Cursor < n {
				// we are going forward
				r.History.ImportState(h)
				for i := 0; i < n-cursor; i++ {
					r.History.Forward()
				}
				
			} else{
				r.History.ImportState(h)
			}
			
			
		}
		r.Outlet.AsElement().Root.SetUI("currentroute", String(newroute))
		r.Outlet.AsElement().Root.SetUI("history", r.History.Value())

		// Let's see if the URI matches any of the registered routes. (TODO)
		v,_,a, err := r.Routes.match(newroute)
		r.Outlet.AsElement().Root.Set("navigation","targetviewid", String(v.AsElement().ID))
		if err != nil {
			log.Print("NOTFOUND", err, newroute) // DEBUG
			if err == ErrNotFound {
				r.Outlet.AsElement().Root.TriggerEvent("navigation-notfound", String(newroute))
				//return false
			}
			if err == ErrUnauthorized {
				log.Print("unauthorized for: " + newroute) //DEBUG
				r.Outlet.AsElement().Root.TriggerEvent("navigation-unauthorized", String(newroute))
				//return false
			}
			if err == ErrFrameworkFailure {
				log.Print("APPFAILURE: ", err) // DEBUG
				r.Outlet.AsElement().Root.TriggerEvent("navigation-appfailure", String(newroute))
				//return false
			}
		} else{
			r.Outlet.AsElement().Root.TriggerEvent("navigation-start", String(newroute))
			err = a()
			if err != nil {
				r.Outlet.AsElement().Root.TriggerEvent("navigation-unauthorized", String(newroute))
				DEBUG("activation failure",err)
			}

			if found{
				r.Outlet.AsElement().Root.Set("navigation","hash",String(hash))
			}
		}
		

		

		r.Outlet.AsElement().Root.TriggerEvent("navigation-end", String(newroute))
		

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
			r.Outlet.AsElement().Root.TriggerEvent("navigation-appfailure", Bool(true))
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
		r.Outlet.AsElement().Root.SetUI("currentroute", String(newroute))
		r.Outlet.AsElement().Root.SetUI("history", r.History.Value())

		// 1. Let's see if the URI matches any of the registered routes.
		v,_, a, err := r.Routes.match(newroute)
		r.Outlet.AsElement().Root.Set("navigation","targetviewid", String(v.AsElement().ID))
		if err != nil {
			log.Print(err, newroute) // DEBUG
			if err == ErrNotFound {
				r.Outlet.AsElement().Root.TriggerEvent("navigation-notfound", String(newroute))
				//return false
			}
			if err == ErrUnauthorized {
				log.Print("unauthorized for: " + newroute) //DEBUG
				r.Outlet.AsElement().Root.TriggerEvent("navigation-unauthorized", String(newroute))
				//return false
			}
			if err == ErrFrameworkFailure {
				log.Print(err) //DEBUG
				r.Outlet.AsElement().Root.TriggerEvent("navigation-appfailure", String(newroute))
				//return false
			}
		} else{
			r.Outlet.AsElement().Root.TriggerEvent("navigation-start", String(newroute))
			err = a()
			if err != nil {
				log.Print(err) // DEBUG
				log.Print("unauthorized for: " + newroute)
				r.Outlet.AsElement().Root.TriggerEvent("navigation-unauthorized", String(newroute))
			}

			if found{
				r.Outlet.AsElement().Root.Set("navigation","hash",String(hash))
			}
		}

		
		
		r.Outlet.AsElement().Root.TriggerEvent("navigation-end", String(newroute))


		return false
	})
	return mh
}

// OnRoutechangeRequest allows to trigger a mutation handler before a route change
// is effective. It needs to be called before ListenAndServe. Returning true should
// cancel the current routechangerequest. (enables hijacking of the route change process)
func (r *Router) OnRoutechangeRequest(m *MutationHandler) {
	r.Outlet.AsElement().Root.WatchEvent("navigation-routechangerequest", r.Outlet.AsElement().Root, m)
}

// ListenAndServe registers a listener for route change after having verified links.
// It should only be called after the app structure has been fully built.
// It listens on the element that receives routechangeevent (first argument)
// It is also the point at which the app runs, in case the document can be dynamically altered.
// It needs to be called and if there is no external navigation event to listen to, 
// an empty string can be passed to the events parameter.
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
func (r *Router) ListenAndServe(ctx context.Context, events string, target AnyElement) {
	if ctx == nil{
		ctx = context.Background()
	}
	r.verifyLinkActivation()
	root := r.Outlet

	// Let's make sure that all the mounted views have been registered.
	v, ok := r.Outlet.AsElement().Root.Get("internals", "views")
	if ok {
		l, ok := v.(List)
		if ok {
			for _, val := range l.UnsafelyUnwrap() {
				viewRef, ok := val.(String)
				if !ok {
					panic("internals/views does not hold a proper Reference")
				}
				viewEl:= GetById(r.Outlet.AsElement().Root,viewRef.String())
				if viewEl == nil{
					panic("framework error: view not found")
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
		root.AsElement().Root.TriggerEvent("navigation-routechangerequest", u.(String))
		return false
	})

	root.AsElement().Root.WatchEvent("navigation-routechangerequest", root.AsElement().Root, r.handler())
	root.AsElement().Root.WatchEvent("navigation-routeredirectrequest", root.AsElement().Root, r.redirecthandler())
	r.Outlet.AsElement().Root.TriggerEvent("document-loaded")
	
	eventnames:= strings.Split(events," ")
	for _,event:= range eventnames{
		target.AsElement().AddEventListener(event, routeChangeHandler)
	}
	
	for {
		select{
		case <-ctx.Done():
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
			if !r.ViewElement.IsViewAuthorized(param) {
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
			if r.ViewElement.IsViewAuthorized(param) {
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
			if r.ViewElement.IsViewAuthorized(segments[0]) {
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
					if !r.ViewElement.IsViewAuthorized(param) {
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
			if !r.ViewElement.IsViewAuthorized(nextroutesegment) {
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
	u,_:= l.Raw.GetUI("uri")
	uri:= string(u.(String))
	return uri
}

func (l Link) Activate(targetid ...string) {
	if len(targetid) == 1{
		if targetid[0] != ""{
			l.Raw.TriggerEvent("activate", String(targetid[0]))
			return 
		}
	}
	l.Raw.TriggerEvent("activate", Bool(true))
}

func (l Link) IsActive() bool {
	status, ok := l.Raw.GetUI("active")
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
	l.Raw.TriggerEvent("prefetchlink",Bool(true))
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
		c,ok:= r.Outlet.AsElement().ElementStore.Constructors["zui_link"]
		if !ok{
			panic("zui_ERROR: somehow the link constructor has not been registered.")
		}
		e := c(r.Outlet.AsElement().ID+"-"+viewname)
		e.SetUI("viewelements",NewList(String(r.Outlet.AsElement().ID)).Commit())
		e.SetUI("viewnames", NewList(String(viewname)).Commit())
		e.SetUI("uri", String("/"+viewname))
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
	RegisterElement(r.Outlet.AsElement().Root,e)

	// Let's retrieve the target viewElement and corresponding view name
	v,ok:= e.GetUI("viewelements")
	if !ok{
		panic("Link creation seems to be incomplete. The list of viewElements for the path it denotes should be present.")
	}
	/*n,ok:= e.GetData("viewnames")
	if !ok{
		panic("Link creation seems to be incomplete. The list of viewnames for the path it denotes should be present.")
	}*/
	vl:= v.(List)
	//nl:= n.(List)
	view:= ViewElement{GetById(r.Outlet.AsElement().Root,vl.Get(len(vl.UnsafelyUnwrap())-1).(String).String())}
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
	e.WatchEvent("mountable", view.AsElement(), NewMutationHandler(func(evt MutationEvent) bool {
		b := evt.NewValue().(Bool)
		if !b {
			return false
		}
		return nh.Handle(evt)
	}).RunASAP())

	// TODO rework this so that it RUnASAP and also only if the target is mounted/mountable DEBUG trying data instead of ui layer
	e.Watch("ui", "currentroute", r.Outlet.AsElement().Root, NewMutationHandler(func(evt MutationEvent) bool {
		route := evt.NewValue().(String).String()
		lnk,_:= e.GetUI("uri")
		link:= lnk.(String).String()

		if string(route) == link {
			e.SetUI("active", Bool(true))
		} else {
			e.SetUI("active", Bool(false))
		}

		return false
	}))

	r.Links[l.URI()] = l

	// TODO do we need to be able to disable navigation if link points to current route?
	r.Outlet.AsElement().WatchEvent("activate",e,NewMutationHandler(func(evt MutationEvent)bool{
		var hash string
		if s,ok:= evt.NewValue().(String);ok{
			hash = "#"+string(s)
		}
		r.GoTo(l.URI()+hash)
		return false
	}))

	r.Outlet.AsElement().WatchEvent("prefetchlink",e,NewMutationHandler(func(evt MutationEvent)bool{
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
		vl = vl.MakeCopy().Append(String(ve.AsElement().ID)).Commit()
		nl = nl.MakeCopy().Append(String(viewname)).Commit()
		ne.SetUI("viewelements",vl)
		ne.SetUI("viewnames",nl)
		uri:="/" + string(nl.Get(0).(String))
		for i,velem:= range vl.UnsafelyUnwrap(){
			if i==0{
				continue
			}
			id:= string(velem.(String))
			vname:= string(nl.Get(i).(String))
			uri = "/" + id + "/" + vname
		}
		ne.SetUI("uri",String(uri))
		return Link{ne}
	}
}


func(r *Router) RetrieveLink(URI string) (Link,bool){
	l, ok := r.Links[URI]
	return l,ok	
}


func isValidLink(l Link) bool{
	e:= l.AsElement()
	v,ok:=e.GetUI("viewelements")
	if !ok{
		return false
	}
	n,ok:= e.GetUI("viewnames")
	if !ok{
		return false
	}
	vl:= v.(List)
	nl:= n.(List)

	targetview:= GetById(l.AsElement().Root,string(vl.Get(len(vl.UnsafelyUnwrap())-1).(String)))
	viewname := string(nl.Get(len(nl.UnsafelyUnwrap())-1).(String))

	vap:= targetview.ViewAccessPath.Nodes
	if len(vap) != len(vl.UnsafelyUnwrap())-1{
		DEBUG("viewaccespath and link depth do not match. Some view might have been skipped")
		return false
	}
	for i,n:= range vap{
		vnode:= GetById(l.AsElement().Root,string(vl.Get(i).(String)))
		if vnode.ID != n.Element.ID{
			return false
		}
		vname:= string(nl.Get(i).(String))
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
	return  v.HasStaticView(vname)
}

/*

   Navigation History

*/

// NavHistory holds the Navigation History. (aka NavStack)
type NavHistory struct {
	AppRoot *Element
	Stack  []string
	State  []Observable
	Cursor int
	NewState func(id string) Observable
	RecoverState func(Observable) Observable
	Length int
}

// Get is used to retrieve a Value from the history state.
func (n *NavHistory) Get(propname string) (Value, bool) {
	return n.State[n.Cursor].Get("data", propname)
}

// Set is used to insert a value in the history state.
func (n *NavHistory) Set(propname string, val Value) {
	n.State[n.Cursor].Set("data", propname, val)
	n.AppRoot.TriggerEvent("history-change",String(propname)) // TODO more finegrained change persistence using propname event value
}

func NewNavigationHistory(approot *Element) *NavHistory {
	n:= &NavHistory{}
	n.AppRoot = approot
	n.Stack = make([]string, 0, 1024)
	n.State = make([]Observable, 0, 1024)
	n.Cursor = -1
	n.NewState = func(id string) Observable{
		e:= GetById(approot,id)
		if e != nil{
			return Observable{e}
		}
		o:= approot.ElementStore.NewObservable(id)
		RegisterElement(approot,o.AsElement())
		// Initially was done in the constructore but might be more appropriate here
		o.AsElement().TriggerEvent("mountable")
		o.AsElement().TriggerEvent("mounted")
		return o
	}
	n.RecoverState = func(o Observable)Observable{
		o.Set("data","new", Bool(false))
		return o
	}
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
	o.Set("stack",NewList(stack...).Commit())

	// Prepare State for serialization
	state:=make([]Value,len(n.State))
	for i,entry:= range n.State{
		state[i]= String(entry.AsElement().ID) // TODO store state objects in navhistory registry and implement recovery
	}
	o.Set("state",NewList(state...).Commit())

	return o.Commit()
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
	stk,ok:= h.Get("stack")
	if !ok{
		DEBUG("No stack found in history state")
		return nil
	}
	stack:= stk.(List)
	hlen:= len(stack.UnsafelyUnwrap())

	stt,ok:= h.Get("state")
	if !ok{
		DEBUG("No state found in history state")
		return nil
	}
	state:= stt.(List)

	if hlen>len(n.Stack){
		for i:=n.Cursor+1; i<hlen;i++{
			entry:= stack.Get(i)
			nexturl:= entry.(String)
			n.Stack=append(n.Stack,string(nexturl))

			stentry:= state.Get(i)
			stateObjid:= stentry.(String).String()
			
			stobj:= GetById(n.AppRoot,stateObjid)
			if stobj==nil{
				stobj=n.NewState("hstate"+strconv.Itoa(i)).AsElement()
			}
			recstate:= Observable{stobj}
			
			_,ok:= recstate.Get("data","new")
			if !ok{
				recstate=n.RecoverState(recstate)
			}
			
			n.State = append(n.State,recstate)
		}
	}

		/*
	for _,s:= range n.State{
		s.Set("data","new",Bool(false))
	}
	*/
	
	
	return n
}

func (n *NavHistory) Push(URI string) *NavHistory {
	if len(n.Stack) >= n.Length {
		panic("navstack capacity overflow")
	}
	if n.Cursor>=0{
		n.State[n.Cursor].Set("data","new", Bool(false)) // used to discover whether the current navigation entry is accessed for the first time or not
	}
	n.Cursor++
	n.Stack = append(n.Stack[:n.Cursor], URI)
	n.State = append(n.State[:n.Cursor], n.NewState("hstate"+strconv.Itoa(n.Cursor)))
	n.State[n.Cursor].Set("data","new", Bool(true))
	
	return n
}

func(n *NavHistory) CurrentEntryIsNew() bool{
	v,ok:= n.State[n.Cursor].Get("data","new")
	if !ok{
		panic("Unable to find (data, new) cursor is : "+ fmt.Sprint(n.Cursor))
	}
	return bool(v.(Bool))
}

func (n *NavHistory) Replace(URI string) *NavHistory {
	n.Stack[n.Cursor] = URI
	n.State[n.Cursor] = n.NewState("hstate"+strconv.Itoa(n.Cursor))
	n.State[n.Cursor].Set("data","new",Bool(false))
	
	// TODO what to do here? perhaps nothing, perhaps the state should be labeled new or the reverse? 
	return n
}

func (n *NavHistory) Back() string {
	if n.BackAllowed() {
		if n.Cursor == len(n.Stack)-1{
			n.State[n.Cursor].Set("data","new", Bool(false)) // TODO should we check the value or use memoization to avoid unnecessary ops on forward()?
		}
		n.Cursor--
	}
	return n.Stack[n.Cursor]
}

func (n *NavHistory) Forward() string {
	if n.ForwardAllowed() {
		if n.Cursor>=0{
			n.State[n.Cursor].Set("data","new", Bool(false))
		}
		n.Cursor++
		n.State[n.Cursor].Set("data","new", Bool(false))
	}
	
	/*v,ok := n.State[n.Cursor].Get("data","new")
	if !ok{
		//n.State[n.Cursor].Set("data","new", Bool(true))
		panic("unable to find new page marker")
	} else{
		n.State[n.Cursor].Set("data","new", v)
	}
	*/
	return n.Stack[n.Cursor]
}

func (n *NavHistory) BackAllowed(step ...int) bool {
	return n.Cursor > 0
}

func (n *NavHistory) ForwardAllowed(step ...int) bool {
	return n.Cursor < len(n.Stack)-1
}
