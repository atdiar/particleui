// Package ui is a library of functions for simple, generic gui development.
package ui

import (
	"errors"
	"log"
	"net/url"
	"strings"
)

var(
	ErrNotFound = errors.New("Not Found")
	ErrUnauthorized = errors.New("Unauthorized")
	ErrFrameworkFailure = errors.New("Framework Failure")
)


// Router stores shortcuts to given states of the application.
// These shortcuts take the form of URIs.
// The router is also in charge of modifying the application state to reach any
// state registered as a shortcut upon request.
type Router struct {
	GlobalHandlers []RouteChangeHandler // used to implement guards such as when user has not the access rights to reach some app state. (authentication?)
	BaseURL        string
	root           ViewElement

	Links map[string]Link

	Routes *routeNode

	History *NavHistory

	LeaveTrailingSlash bool
}

// NewRouter takes an Element object which should be the entry point of the router
// as well as the document root which should be the entry point of the document/application tree.
// TODO eliminate documentroot ... redundant
func NewRouter(baseurl string, documentroot ViewElement) *Router {
	u, err := url.Parse(baseurl)
	if err != nil {
		return panic(err)
	}
	base := strings.TrimSuffix(u.Path, "/")
	documentroot.Element().Global.Set("internals", "baseurl", String(baseurl), true)
	r:= &Router{make([]RouteChangeHandler, 0), base, documentroot, make(map[string]Link,300), newRouteNode("/"), NewNavigationHistory().Push("/"), false}
	return r


// AddCatchAllHandlers allows to add routing request handlers applying to  every route.
// Successive calls to this method will append handlers to the current list.
func (r *Router) AddCatchAllHandlers(handlers ...RouteChangeHandler) {
	if r.GlobalHandlers == nil {
		r.GlobalHandlers = make([]RouteChangeHandler, 0)
	}
	r.GlobalHandlers = append(r.GlobalHandlers, handlers...)
}


// GoTo changes the application state by updating the current route
func (r *Router) GoTo(route string) {
	r.root.Element().SetDataSyncUI("currentroute", String(route))
	r.History.Push(route)
}

func(r *Router) GoBack(){
	if r.History.BackAllowed(){
		r.root.Element().SetDataSyncUI("currentroute", String(r.History.GoBack()))
	}
}

func(r *Router) GoForward(){
	if r.History.ForwardAllowed(){
		r.root.Element().SetDataSyncUI("currentroute", String(r.History.GoForward()))
	}
}

// OnNotFound enables the addition of a special view to the root ViewElement.
// The router should navigate toward it when no match has been found for a given input route.
func(r *Router) OnNotFound(dest View) *Router{
	r.root.AddView(dest)

	r.root.Element().Watch("navigation","notfound",r.root.Element(), NewMutationHandler(func(evt MutationEvent)bool{
			r.GoTo(r.BaseURL + "/" + View.Name())
			return false
	}))
	return r
}

// OnUnAuthorized enables the addition of a special view to the root ViewElement.
// The router should navigate toward it when access to an input route is not granted
// due to insufficient rights.
func(r *Router) OnUnAuthorized(dest View) *Router{
	r.root.AddView(dest)

	r.root.Element().Watch("navigation","unauthorized",r.root.Element(), NewMutationHandler(func(evt MutationEvent)bool{
			r.GoTo(r.BaseURL + "/" + View.Name())
			return false
	}))
	return r
}

// OnAppFailure enables the addition of a special view to the root ViewElement.
// The router should navigate toward it when a malfunction occured.
func(r *Router) OnAppFailure(dest View) *Router{
	r.root.AddView(dest)

	r.root.Element().Watch("navigation","appfailure",r.root.Element(), NewMutationHandler(func(evt MutationEvent)bool{
			r.GoTo(r.BaseURL + "/" + View.Name())
			return false
	}))
	return r
}

// mount stores the route of an Element (it is also the route of the view if the Element belongs to one)
// A route URI represents a serie of states for the User-Interface that needs to be
// activated for an element to show up on screen.
func (r *Router) mount(element *Element, middleware ...RouteChangeHandler) {
	if element.ViewAccessPath == nil || len(element.ViewAccessPath.Nodes) == 0 {
		return
	}
	route := element.Route()
	r.Routes.Insert(route, newRegistree(element, middleware...))
}

// handler returns a mutation handler which deals with route change.
func (r *Router) handler() *MutationHandler {
	mh := NewMutationHandler(func(evt MutationEvent) bool {
		nroute, ok := evt.NewValue().(String)
		if !ok {
			log.Print("route mutation has wrong type... something must be wrong", evt.NewValue())
			r.root.Element().Set("navigation","appfailure",Bool(true)) // TODO document should watch itself and Activate the relevant view
			return true
		}
		newroute:=string(nroute)
		if !r.LeaveTrailingSlash {
			if newroute[len(newroute)-1:] == "/" {
				newroute = newroute[:len(newroute)-1]
			}
		}
		newroute = strings.TrimPrefix(newroute, r.BaseURL)

		Target, ApplyRouteFn := r.Routes.Match(newroute, nil)
		if ApplyRouteFn == nil {
			// No route match
			r.root.Element().Set("navigation","notfound",Bool(true)) // TODO document should watch itself and Activate the relevant view
			return true
		}
		for _, glmiddleware := range r.GlobalHandlers {
			stop := glmiddleware.Handle(Target)
			if stop {
				r.root.Element().Set("navigation","unauthorized",Bool(true)) // TODO document should watch itself and Activate the relevant view
				return stop
			}
		}
		err := ApplyRouteFn()
		if err != nil {
			r.root.Element().Set("navigation","unauthorized",Bool(true)) // TODO document should watch itself and Activate the relevant view
			return true
		}
		r.History.Push(nroute) // TODO doe sit belong there? does it not interfere with back and forward?
		return false // todo  i'm wondering if this is the right value to return.
	})
	return mh
}

// ListenAndServe registers a listener for route change.
// It should only be called after the app structure has been fully built.
//
// Example of JS bridging : the nativeEventBridge should add a popstate event listener to window
// It should also dispatch a RouteChangeEvent to bridge browser url mutation into the Go side
// after receiving notice of popstate event firing.
func (r *Router) ListenAndServe(nativebinding NativeEventBridge) {
	r.verifyLinkActivation()
	root := r.root
	routeChangeHandler := NewEventHandler(func(evt Event) bool {
		event, ok := evt.(RouteChangeEvent)
		if !ok {
			log.Print("Event of wrong type. Expected a RouteChangeEvent firing")
			root.Element().Set("navigation","appfailure", String("500: RouteChangeEvent of wrong type."))
			return true // means that event handling has to stop
		}
		// the target element route should be changed to the event NewRoute value.
		root.Element().SyncUISetData("currentroute", String(event.NewRoute()), false)
		return false
	})

	root.Element().AddEventListener("routechange", routeChangeHandler, nativebinding)
	root.Element().Watch("data", "currentroute", root.Element(), r.handler())
	r.serve()
}

func(r *Router) verifyLinkActivation(){
	for _,l:=range rLinks{
		_,ok:= l.Get("event","activated")
		if !ok{
			panic("Link activation failure: " + l.URI())
		}
	}
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

/*

   router nodes


*/
// A rnode is a router node. It holds information about the viewElement,
// the value field holding the viewid for the corresponding ViewElement,
// and a map of the potential children ViewElements classified by views (via viewnames)
type rnode struct{
	root *rnode
	value string // just a copy of the ViewElement.Element().ID
	ViewElement
	next map[string]map[string]*rnode // Each rnode has a list of views and each view may link to multiple same level ViewElement map[viewname]map[viewid]rnode
}

func newchildrnode(v ViewElement, root *rnode) *rnode{
	m:=make(map[string]map[string]*rnode)
	for k,_:=range v.Element().InactiveViews{
		m[k]= nil
	}
	if a:=v.Element().ActiveView;a!= ""{
		m[a] = nil
	}
	return &rnode{root,v.Element().ID,v,nil}
}

func newrootrnode(v ViewElement) *rnode{
	r:= newchildrnode(v)
	r.root = r
	return r
}

// insert  adds an arbitrary rnode to the rnode trie if  possible (the root
// ViewElement of the rnode ViewAccessPath should be that of the root rnode )
func(rn *rnode) insert(nrn *rnode){
	v:= nrn.ViewElement

	viewpath:= v.Element().ViewAccessPath
	if viewpathnodes == nil{
		return
	}
	viewpathnodes:= viewpath.Nodes
	if viewpathnodes[0].Element.ID != rn.root.ViewElement.Element().ID{
		return
	}
	l:= len(viewpathnodes)
	// attach iteratively the rnodes
	refnode := rn
	for i,node:=range viewpathnodes{
		if i+1 < l{
			// each view should be a rootnode and should be attached in succession. The end node is our argument.
			view:= ViewElement(viewpathnodes[i+1].Element)
			nr:=newchildrnode(view,rn)
			refnode.attach(node.View.Name(),nr)
			refnode = nr
		}
	}
	refnode.attach(nrn)
}

// attach links to rnodes that corresponds to viewElements that succeeds each other
func(r *rnode) attach(targetviewname string, nr *rnode){
	m,ok:=r.next[targetviewname]
	if !ok{
		m=make(map[string]*rnode)
		r.next[targetviewname] = m
	}
	r,ok:=m[nr.ViewElement.Element().ID]
	if !ok{
		m[nr.ViewElement.Element().ID]=nr
	} // else it has already been attached
}

// match verifies that a route passed as arguments corresponds to a given view state.
func(r *rnode) match(route string) error{
	route = strings.TrimPrefix(route,"/")
	segments := strings.SplitAfter(route, "/")
	ls:= len(segment)

	if ls==0{
		return ErrNotFound
	}


	var param string
	m,ok:= r.next[segments[0]] // 0 is the index of the viewname at the root ViewElement m is of type map[string]*rnode
	if !ok{
		// Let's see if the ViewElement has a parameterizable view
		param,ok= r.ViewElement.hasParameterizedView()
		r.ViewElement.Element().Set("navigation",param,String(segment[0]))
		if ls !=1{ // we get the next rnodes by uding the segment[1]
			m,ok=r.next[param]
			if !ok{
				return ErrFrameworkFailure
			}
		}
	}

	// Does other children views need activation? Let's check for it.
	if ls==1{
		// check authorization
		if param != ""{
			if r.ViewElement.isViewAuthorized(param){
				return nil
			}
			return ErrUnauthorized
		}
		if r.ViewElement.isViewAuthorized(segment[0]){
			return nil
		}
		return ErrUnauthorized
	}

	if ls%2 != 1 {
		return false
	}

	viewcount:= (ls - ls%2)/2

	// Let's get the next rnode and check that the view mentionned in the route exists (segment[2i+2])

	for i:=1;i<=viewcount;i++{
		routesegment:= segments[2i]
		nextroutesegment[2i+1]
		r,ok= m[routesegment]
		if !ok{
			return ErrNotFound
		}

		if r.value != routesegment{
			return ErrNotFound
		}

		// Now that we have the rnode, we can try to see if the nextroutesegment holding the viewname
		// is in the r.next. If not, we check whether the viewElement can be parameterized
		// and the new map pf next rnode is then retrieved if possible.
		m,ok=r.next[nextroutesegment]
		if !ok{
			// Let's see if the ViewElement has a parameterizable view
			param,ok= r.ViewElement.hasParameterizedView()
			if !r.ViewElement.isViewAuthorized(param){
				return ErrUnauthorized
			}
			r.ViewElement.Element().Set("navigation",param,String(segment[0]))
			if r.next{ // we get the next rnodes by using the segment[1]
				m,ok=r.next[param]
				if !ok{
					return ErrFrameworkFailure
				}

			}
		}
		if !r.ViewElement.isViewAuthorized(nextroutesegment){
			return ErrUnauthorized
		}
	}
}




/*

		Navigation link creation

*/

// Link holds the representation (under the form of an URI) of the application state
// required for the target View to be available for display on screen.
type Link struct{
	Raw *Element

	Target ViewElement
	ViewName string

	Router *Router
}

func(l Link) URI() string{
	return l.Target.Element().Route() +"/" + l.Target.ID +"/" + l.ViewName
}

func(l Link) Activate(){
	l.Router.GoTo(l.URI())
}

func(r *Router) NewLink(target ViewElement, viewname string) Link{
	// If previously created, it has been memoized. let's retrieve it then. otherwise,
	// let's create it.
	l,ok:= r.Links[target.Element().ID+"/"+viewname]
	if ok{
		return l
	}

	e:= NewElement(viewname,target.Element().ID+"/"+viewname,r.documentroot.Element().DocType)
	nh:= NewMutationHandler(func(evt MutationEvent)bool{
			e.Set("event","activated",Bool(true))
	})
	e.Watch("event","mounted",target.Element(), nh)
	l= Link{e,target,viewname,r}
	r.Links[target.Element().ID+"/"+viewname] = l

	return l
}

/*

    Navigation History

*/

// NavHistory holds the Navigation History. (aka NavStack)
type NavHistory struct{
	Stack []string
	Cursor int
}

func NewNavigationHistory() *NavHistory{
	return &NavHistory{make([]string,0,300),0}
}

func(n *NavHistory) Push(URI string) *NavHistory{
	n.Stack = append(n.Stack[:n.Cursor],URI)
	n.Cursor++
	return n
}

func(n *NavHistory) Back() string{
	if BackAllowed(){
		n.Cursor--
	}
	return n.Stack[n.Cursor]
}

func(n *NavHistory) Forward() string{
	if ForwardAllowed(){
		n.Cursor++
	}
	return n.Stack[n.Cursor]
}

func(n *NavHistory) BackAllowed() bool{
	return n.Cursor>0
}

func(n *NavHistory) ForwardAllowed() bool{
	return n.Cursor < len(n.Stack)-1
}
