// Package ui is a library of functions for simple, generic gui development.
package ui

import (
	"errors"
	"log"
	"net/url"
	"strconv"
	"strings"
)

var (
	ErrNotFound         = errors.New("Not Found")
	ErrUnauthorized     = errors.New("Unauthorized")
	ErrFrameworkFailure = errors.New("Framework Failure")
)

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
		evt.Origin().WatchASAP("event","initrouter",evt.Origin().AsElement().Root(),NewMutationHandler(func(evt MutationEvent)bool{
			fn(GetRouter())
			return false
		}))
		return false
	})
	user.AsElement().OnFirstTimeMounted(h)
}

// Router stores shortcuts to given states of the application.
// These shortcuts take the form of URIs.
// The router is also in charge of modifying the application state to reach any
// state registered as a shortcut upon request.
type Router struct {
	BasePath string
	outlet   ViewElement

	Links map[string]Link

	Routes *rnode

	History *NavHistory

	LeaveTrailingSlash bool
}

// NewRouter takes an Element object which should be the entry point of the router.
func NewRouter(basepath string, rootview ViewElement, options ...func(*Router)*Router) *Router {
	if router != nil {
		panic("A router has already been created")
	}
	if !rootview.AsElement().Mountable() {
		panic("router can only use a view attached to the main tree as a navigation outlet.")
	}
	u, err := url.Parse(basepath)
	if err != nil {
		panic(err)
	}

	r := &Router{u.Path, rootview, make(map[string]Link, 300), newrootrnode(rootview), NewNavigationHistory(), false}

	r.outlet.AsElement().Root().Watch("event", "docupdate", r.outlet.AsElement().Root(), NewMutationHandler(func(evt MutationEvent) bool {
		_, navready := r.outlet.AsElement().Root().Get("navigation", "ready")
		if !navready {
			v, ok := r.outlet.AsElement().Global.Get("internals", "views")
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

	for _,option:= range options{
		r = option(r)
	}

	router = r
	r.outlet.AsElement().Root().Set("event","initrouter",Bool(true))
	return r
}

func (r *Router) tryNavigate(newroute string) bool {
	// 0. Retrieve hash if it exists
	route,hash,found:= strings.Cut(newroute,"#")
	if found{
		newroute = route
	}

	// 1. Let's see if the URI matches any of the registered routes.
	a, err := r.Routes.match(newroute)
	if err != nil {
		log.Print(err) // DEBUG
		if err == ErrNotFound {
			log.Print("this is strange", err) // DEBUG
			r.outlet.AsElement().Root().Set("navigation", "notfound", String(newroute))
			return false
		}
		if err == ErrUnauthorized {
			log.Print(err) // DEBUG
			r.outlet.AsElement().Root().Set("navigation", "unauthorized", String(newroute))
			return false
		}
		if err == ErrFrameworkFailure {
			log.Print(err) //DEBUG
			r.outlet.AsElement().Root().Set("navigation", "appfailure", String(newroute))
			return false
		}
	}
	err = a()
	if err != nil {
		log.Print("activation failure", err) // DEBUG
		r.outlet.AsElement().Root().Set("navigation", "unauthorized", String(newroute))
		return false
	}
	if found{
		r.outlet.AsElement().Root().Set("navigation","hash",String(hash))
	}
	return true
}

// GoTo changes the application state by updating the current route
func (r *Router) GoTo(route string) {
	if !r.LeaveTrailingSlash {
		route = strings.TrimSuffix(route, "/")
	}
	route = strings.TrimPrefix(route, r.BasePath)

	r.outlet.AsElement().Root().Set("event", "navigationstart", String(route))

	r.History.Push(route)
	ok := r.tryNavigate(route)
	if !ok {
		log.Print("NAVIGATION FAILED FOR SOME REASON.") // DEBUG
		return
	}

	r.outlet.AsElement().Root().SetData("history", r.History.Value())
	r.outlet.AsElement().Root().Set("event", "navigationend", String(route))
	r.outlet.AsElement().Root().SetDataSetUI("currentroute", String(route))
	DEBUG("goto: ",route, r.History.Cursor)
}

func (r *Router) GoBack() {
	if r.History.BackAllowed() {
		r.outlet.AsElement().Root().Set("navigation", "routechangerequest", String(r.History.Back()))
	}
}

func (r *Router) GoForward() {
	if r.History.ForwardAllowed() {
		r.outlet.AsElement().Root().Set("navigation", "routechangerequest", String(r.History.Forward()))
	}
}

// RedirectTo can be used during a routechangerequest event to reroute to an alternate location.
// Typically useful when handling the three navigation failure modes (notfound, unauthorized, appfailure).
// Use Hijack instead if you need to register a given rerouting behavior.
func (r *Router) RedirectTo(route string) {
	r.outlet.AsElement().Root().Set("navigation", "routeredirectrequest", String(route))
}

// Hijack short-circuits navigation to allow for the redirection of a specific route to an alternate 
// destination.
func (r *Router) Hijack(route string, destination string) {
	r.OnRoutechangeRequest(NewMutationHandler(func(evt MutationEvent) bool {
		navroute := evt.NewValue().(String)
		if string(navroute) == route {
			r.History.Push(route)
			r.RedirectTo(destination)
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

// OnNotfound reacts to the navigation 'notfound' property being set. It can allow
// to go to an alternate route for instance, which could display a "page not found"
// error message for example. This behavior would be defined in the MutationHandler.
func (r *Router) OnNotfound(h *MutationHandler) *Router {
	r.outlet.AsElement().Root().Watch("navigation", "notfound", r.outlet.AsElement().Root(), h)
	return r
}

// OnUnauthorized reacts to the navigation state being set to unauthorized.
// It may occur when there are insufficient rights to displaya given view for instance.
func (r *Router) OnUnauthorized(h *MutationHandler) *Router {
	r.outlet.AsElement().Root().Watch("navigation", "unauthorized", r.outlet.AsElement().Root(), h)
	return r
}

// OnAppfailure reacts to the navigation state being set to "appfailure".
// It may occur when a malfunction occured.
// The MutationHandler informs of the behavior to addopt in this case.
func (r *Router) OnAppfailure(h *MutationHandler) *Router {
	r.outlet.AsElement().Root().Watch("navigation", "appfailure", r.outlet.AsElement().Root(), h)
	return r
}

func (r *Router) insert(v ViewElement) {
	nrn := newchildrnode(v, r.Routes)
	r.Routes.insert(nrn)
}

// Match returns whether a route is valid or not. It can be used in tests to
// Make sure that an app links are not breaking.
func (r *Router) Match(route string) error {
	_, err := r.Routes.match(route)
	return err

}

/*
// canonicalBase is mainly used to get rid of the trailing slash if present.
func canonicalBase(s string) string {
	t := strings.SplitAfter(s, "/")
	var res string
	for i := 0; i < len(t)-1; i++ {
		res = res + t[i]
	}
	return res
}

*/

// handler returns a mutation handler which deals with route change.
func (r *Router) handler() *MutationHandler {
	mh := NewMutationHandler(func(evt MutationEvent) bool {
		nroute, ok := evt.NewValue().(String)
		if !ok {
			log.Print("route mutation has wrong type... something must be wrong", evt.NewValue())
			r.outlet.AsElement().Root().Set("navigation", "appfailure", Bool(true))
			return true
		}
		newroute := string(nroute)
		if !r.LeaveTrailingSlash {
			newroute = strings.TrimSuffix(newroute, "/")
		}
		newroute = strings.TrimPrefix(newroute, r.BasePath)

		// Retrieve hash if it exists
		route,hash,found:= strings.Cut(newroute,"#")
		if found{
			newroute = route
		}

		// Let's see if the URI matches any of the registered routes. (TODO)
		a, err := r.Routes.match(newroute)
		if err != nil {
			log.Print("NOTFOUND", err, newroute) // DEBUG
			if err == ErrNotFound {
				r.outlet.AsElement().Root().Set("navigation", "notfound", String(newroute))
				return true
			}
			if err == ErrUnauthorized {
				log.Print("unauthorized for: " + newroute) //DEBUG
				r.outlet.AsElement().Root().Set("navigation", "unauthorized", String(newroute))
				return true
			}
			if err == ErrFrameworkFailure {
				log.Print("APPFAILURE: ", err) // DEBUG
				r.outlet.AsElement().Root().Set("navigation", "appfailure", String(newroute))
				return true
			}
		}
		r.outlet.AsElement().Root().Set("event", "navigationstart", String(newroute))
		err = a()
		if err != nil {
			r.outlet.AsElement().Root().Set("navigation", "unauthorized", String(newroute))
			DEBUG("activation failure",err)
			return true
		}

		if found{
			r.outlet.AsElement().Root().Set("navigation","hash",String(hash))
		}

		// Determination of navigation history action 
		h, ok := r.outlet.AsElement().Root().Get("ui", "history")
		if !ok {
			//DEBUG("no ui history")
			r.History.Push(newroute)
			r.outlet.AsElement().Root().SetData("history", r.History.Value())
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
				//DEBUG("from: ",cursor, " to ",n)
				r.History.ImportState(h)
			}

			r.outlet.AsElement().Root().SetData("history", r.History.Value())
		}
		r.outlet.AsElement().Root().Set("event", "navigationend", String(newroute))
		r.outlet.AsElement().Root().SetDataSetUI("currentroute", String(newroute))

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
			r.outlet.AsElement().Root().Set("navigation", "appfailure", Bool(true))
			return true
		}
		newroute := string(nroute)
		if !r.LeaveTrailingSlash {
			newroute = strings.TrimSuffix(newroute, "/")
		}
		newroute = strings.TrimPrefix(newroute, r.BasePath)

		// Retrieve hash if it exists
		route,hash,found:= strings.Cut(newroute,"#")
		if found{
			newroute = route
		}

		// 1. Let's see if the URI matches any of the registered routes.
		a, err := r.Routes.match(newroute)
		if err != nil {
			log.Print(err, newroute) // DEBUG
			if err == ErrNotFound {
				r.outlet.AsElement().Root().Set("navigation", "notfound", String(newroute))
				return true
			}
			if err == ErrUnauthorized {
				log.Print("unauthorized for: " + newroute) //DEBUG
				r.outlet.AsElement().Root().Set("navigation", "unauthorized", String(newroute))
				return true
			}
			if err == ErrFrameworkFailure {
				log.Print(err) //DEBUG
				r.outlet.AsElement().Root().Set("navigation", "appfailure", String(newroute))
				return true
			}
		}
		r.outlet.AsElement().Root().Set("event", "navigationstart", String(newroute))
		err = a()
		if err != nil {
			log.Print(err) // DEBUG
			log.Print("unauthorized for: " + newroute)
			r.outlet.AsElement().Root().Set("navigation", "unauthorized", String(newroute))
			return true
		}

		if found{
			r.outlet.AsElement().Root().Set("navigation","hash",String(hash))
		}

		r.History.Replace(newroute)
		r.outlet.AsElement().Root().SetData("history", r.History.Value())
		
		r.outlet.AsElement().Root().Set("event", "navigationend", String(newroute))

		r.outlet.AsElement().Root().SetDataSetUI("redirectroute", String(newroute))

		DEBUG("redirect ",*r.History)

		return false
	})
	return mh
}

// OnRoutechangeRequest allows to trigger a mutation handler before a route change
// is effective. It needs to be called before ListenAndServe. Returning true should
// cancel the current routechangerequest. (enables hijacking of the route change process)
func (r *Router) OnRoutechangeRequest(m *MutationHandler) {
	r.outlet.AsElement().Root().Watch("navigation", "routechangerequest", r.outlet.AsElement().Root(), m)
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
	v, ok := r.outlet.AsElement().Global.Get("internals", "views")
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
		root.AsElement().Root().Set("navigation", "routechangerequest", evt.Value())
		return false
	})

	root.AsElement().Root().Watch("navigation", "routechangerequest", root.AsElement().Root(), r.handler())
	root.AsElement().Root().Watch("navigation", "routeredirectrequest", root.AsElement().Root(), r.redirecthandler())
	r.outlet.AsElement().Root().Set("navigation", "ready", Bool(true))
	target.AddEventListener(eventname, routeChangeHandler, nativebinding)

	c := make(chan struct{}, 0)
	<-c
}

func (r *Router) verifyLinkActivation() {
	for _, l := range r.Links {
		_, ok := l.Raw.Get("event", "verified")
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
func (r *rnode) match(route string) (activationFn func() error, err error) {
	activations := make([]func() error, 0, 10)
	route = strings.TrimPrefix(route, "/")
	segments := strings.Split(route, "/")
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
	if ls >= 1 && ls%2 == 1 {
		// check authorization
		if param != "" {
			if r.ViewElement.isViewAuthorized(param) {
				a := func() error {
					return r.ViewElement.ActivateView(segments[0])
				}
				activations = append(activations, a)
			} else {
				return nil, ErrUnauthorized
			}
		} else {
			if r.ViewElement.isViewAuthorized(segments[0]) {
				a := func() error {
					return r.ViewElement.ActivateView(segments[0])
				}
				activations = append(activations, a)
			} else {
				return nil, ErrUnauthorized
			}
		}
	}

	if ls%2 != 1 {
		log.Print("Incorrect URI scheme")
		return nil, ErrNotFound
	}
	if ls > 1 {
		viewcount := (ls - ls%2) / 2

		// Let's get the next rnode and check that the view mentionned in the route exists (segment[2i+2])

		for i := 1; i <= viewcount; i++ {
			routesegment := segments[2*i-1]   //ids
			nextroutesegment := segments[2*i] //viewnames
			r, ok := m[routesegment]
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
// A link can be watched.
type Link struct {
	Raw *Element
}

func (l Link) URI() string {
	u,_:= l.Raw.GetData("uri")
	uri:= string(u.(String))
	return uri
}

func (l Link) Activate() {
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
// However, link creation is verified at app startups and invalid links should trigger a panic.
func (r *Router) NewLink(viewname string, modifiers ...func(Link)Link) Link {
	// If previously created, it has been memoized. let's retrieve it then. otherwise,
	// let's create it.

	if isParameter(viewname){
		panic(viewname + " is not a valid view name.")
	}
	
	l,ok:= r.Links["/"+viewname]
	if !ok{
		e := NewElement(viewname, r.outlet.AsElement().ID+"-"+viewname, r.outlet.AsElement().DocType)
		e.SetData("viewelements",NewList(r.outlet.AsElement()))
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
	n,ok:= e.GetData("viewnames")
	if !ok{
		panic("Link creation seems to be incomplete. The list of viewnames for the path it denotes should be present.")
	}
	vl:= v.(List)
	nl:= n.(List)
	view:= ViewElement{vl[len(vl)-1].(*Element)}
	viewname = string(nl[len(nl)-1].(String))



	nh := NewMutationHandler(func(evt MutationEvent) bool {
		o:= ViewElement{evt.Origin()}
		if o.hasStaticView(viewname) { // viewname corresponds to an existing view
			_, ok := e.Get("event", "verified")
			DEBUG(viewname, ok)
			if !ok {
				e.Set("event", "verified", Bool(true))
				return false
			}
		}

		if _, ok := o.hasParameterizedView(); ok {
			_, ok := e.Get("event", "verified")
			if !ok {
				e.Set("event", "verified", Bool(true))
			}
		}

		return false
	})
	e.WatchASAP("event", "mountable", view.AsElement(), NewMutationHandler(func(evt MutationEvent) bool {
		b := evt.NewValue().(Bool)
		if !b {
			return false
		}
		return nh.Handle(evt)
	}))
	e.Watch("data", "currentroute", r.outlet.AsElement().Root(), NewMutationHandler(func(evt MutationEvent) bool {
		route := evt.NewValue().(String)
		lnk,_:= e.GetData("uri")
		link:= string(lnk.(String))
		
		if string(route) == link {
			e.SyncUISetData("active", Bool(true))
		} else {
			e.SyncUISetData("active", Bool(false))
		}

		return false
	}))

	r.Links[l.URI()] = l

	r.outlet.AsElement().Watch("event","activate",e,NewMutationHandler(func(evt MutationEvent)bool{
		r.GoTo(l.URI())
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
		ne:= NewElement(viewname, ve.AsElement().ID+"-"+viewname, e.DocType)

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
		return NewObservable(id)
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

	n.Cursor++
	n.Stack = append(n.Stack[:n.Cursor], URI)
	n.State = append(n.State[:n.Cursor], n.NewState("hstate"+strconv.Itoa(n.Cursor)))
	
	return n
}

func (n *NavHistory) Replace(URI string) *NavHistory {
	n.Stack[n.Cursor] = URI
	n.State[n.Cursor] = n.NewState("hstate"+strconv.Itoa(n.Cursor))
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

func (n *NavHistory) BackAllowed(step ...int) bool {
	return n.Cursor > 0
}

func (n *NavHistory) ForwardAllowed(step ...int) bool {
	return n.Cursor < len(n.Stack)-1
}
