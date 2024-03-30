package ui

import (
	"context"
	"encoding/base32"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var PrefetchMaxAge = 5 * time.Second

// WorkQueue is a queue of UI mutating function that can be built from multiple goroutines.
// Only the UI thread read from this to do work on the UI tree.
var WorkQueue = make(chan func())

// DoSync sends a function to the main goroutine that is in charge of the UI to be run.
// Goroutines launched from the main thread that need access to the main UI tree must use it.
// Only a single DoSync must be used within a DoAsync.
func DoSync(fn func()) {
	ch := make(chan struct{})
	go func() {
		WorkQueue <- newwork(fn, ch)
	}()
	<-ch
}

func newwork(f func(), signalDone chan struct{}) func() {
	return func() {
		f()
		close(signalDone)
	}
}

// DoAsync pushes a function onto a goroutine for execution.
// Instead of launching goroutines raw by using the 'go' statement, one should use this wrapper for
//
//	any concurrent processing.
//
// If nil is passed as the *ui.Element argument, the function will not be cancelled if the navigation
// changes. Otherwise, async function execution is cancellable by new navigation events.
//
// e.g. a fetch might be cancelled if it's not done before the user navigates away from the current route.
// meaning that each time a new navigation event is triggered, unfinished fetch for the document
// are cancelled.
//
// If after the concurrent task has been handled, the UI tree needs to be updated,
// the function should employ a DoSync to push the changes back to the main goroutine.
//
// It is necessary to use only one DoSync per such function, in order for every
// goroutine to see the fully updated tree and not a partially updated one.
// (reminder that a DoSync represent a critical section for the UI tree and in-between calls
// the UI goroutine is preemptable).
func DoAsync(e *Element, f func(context.Context)) {
	var executionCtx context.Context
	var cancel context.CancelFunc

	var ctxchan <-chan struct{}

	if e != nil {
		if e.Root != nil {
			if r := e.Root.router; r != nil {
				ctxchan = r.NavContext.Done()
				executionCtx, cancel = context.WithCancel(r.NavContext)
			} else {
				executionCtx = context.Background()
				cancel = func() {}
			}
		} else {
			executionCtx = context.Background()
			cancel = func() {}
			return
		}
	}

	go func() {
		select {
		case <-ctxchan:
			cancel()
		default:
			f(executionCtx)
		}
	}()
}

// DoAfter schedules a function to be executed after a certain duration.
// Similarly to DoAfter, if attached to a regsitered Element,
// the function is cancellable when navigating.
// When f is supposed to update the UI tree, it needs to use a single DoSync call when it does so.
func DoAfter(d time.Duration, e *Element, f func(ectx context.Context)) {
	t := time.NewTimer(d)

	var executionCtx context.Context
	var cancel context.CancelFunc

	var ctxchan <-chan struct{}

	if e != nil {
		if e.Root != nil {
			if r := e.Root.router; r != nil {
				ctxchan = r.NavContext.Done()
				executionCtx, cancel = context.WithCancel(r.NavContext)
			} else {
				executionCtx = context.Background()
				cancel = func() {}
			}
		} else {
			executionCtx = context.Background()
			cancel = func() {}
			return
		}
	}

	go func() {
		select {
		case <-ctxchan:
			cancel()
			return
		case <-t.C:
			f(executionCtx)
		}
	}()
}

func (e *Element) setDataPrefetcher(propname string, reqfunc func(e *Element) *http.Request, responsehandler func(*http.Response) (Value, error)) {
	// TODO panic if data fetcher already exists for this propname
	if e.fetching(prefetchTxName(propname, "start")) { // todo this is not the right propname to check. should use the fetching transition prop name
		panic("a data fetcher has already been set for this element and property")
	}

	e.registerPrefetch(propname)
	var ctx context.Context
	var cancelFn context.CancelFunc

	prefetch := NewMutationHandler(func(evt MutationEvent) bool {
		if evt.Origin().isFetchedDataValid(propname) {
			evt.Origin().endprefetchTransition(propname)
			return false
		}
		if evt.Origin().isPrefetchedDataValid(propname) {
			evt.Origin().endprefetchTransition(propname)
			return false
		}

		r := reqfunc(e)
		ctx, cancelFn = context.WithCancel(r.Context())
		r = r.WithContext(ctx)

		evt.Origin().fetchData(propname, r, cancelFn, responsehandler, true)
		return false
	}).fetcher()

	cancel := NewMutationHandler(func(event MutationEvent) bool {
		cancelFn()
		return false
	}).RunOnce()

	end := NewMutationHandler(func(evt MutationEvent) bool {
		switch t := evt.NewValue().(type) {
		case String:
			if t.String() == "cancelled" {
				e.prefetchCompleted(propname, false)
			} else {
				panic("unexpected prefetch transition end value")
			}
		case Bool:
			if t.Bool() {
				evt.Origin().prefetchCompleted(propname, true)
			} else {
				evt.Origin().prefetchCompleted(propname, false)
			}
		}
		return false
	}).RunOnce()

	e.newPrefetchTransition(propname, prefetch, nil, cancel, end)

}

// SetDataFetcher allows an element to retrieve data by sending a http Get request as soon as it gets mounted.
// It accepts a function as argument that is tasked with converting the *http.Response into
// a Value that can be stored as an element property.
// Unless stated otherwise, the data is made prefetchable as well.
// The data is set asynchronously.
//
// The fetching occurs during the "fetch" event ("event","fetch") that is triggered each time an element
// is mounted.
func (e *Element) SetDataFetcher(propname string, reqfunc func(e *Element) *http.Request, responsehandler func(*http.Response) (Value, error), prefetchable bool) {
	_, ok := e.Get("internals", "fetching")
	if !ok {
		e.enablefetching()
		e.Set("internals", "fetching", Bool(true))
	}
	// TODO panic if data fetcher already exists for this propname
	if e.fetching(fetchTxName(propname, "start")) { // todo this is not the right propname to check. should use the fetching transition prop name
		panic("a data fetcher has already been set for this element and property")
	}

	e.registerfetch(propname)

	if prefetchable {
		e.setDataPrefetcher(propname, reqfunc, responsehandler)
	}

	var ctx context.Context
	var cancelFn context.CancelFunc

	reqmonitor := NewMutationHandler(func(ev MutationEvent) bool {
		if strings.EqualFold(verb(ev.NewValue()), "GET") {
			return false
		}
		ev.Origin().invalidatePrefetch(propname)
		ev.Origin().InvalidateFetch(propname)
		ev.Origin().Fetch(propname)
		return false
	})

	fetch := NewMutationHandler(func(evt MutationEvent) bool {
		if evt.Origin().isFetchedDataValid(propname) {
			return false
		}
		if evt.Origin().isPrefetchedDataValid(propname) {
			evt.Origin().endfetchTransition(propname)
			return false
		}

		evt.Origin().cancelPrefetch(propname)

		r := reqfunc(e)
		ctx, cancelFn = context.WithCancel(r.Context())
		r = r.WithContext(ctx)

		// After a new http.Request has been launched and a response has been returned, cancel and refetch
		// the data corresponding to the r.URL
		evt.Origin().OnRegistered(NewMutationHandler(func(event MutationEvent) bool {
			event.Origin().RemoveMutationHandler("event", "request-"+requestID(r), event.Origin().Root, reqmonitor)
			event.Origin().WatchEvent("request-"+requestID(r), event.Origin().Root, reqmonitor)
			return false
		}).RunOnce().RunASAP())
		evt.Origin().fetchData(propname, r, cancelFn, responsehandler, false)
		return false
	}).fetcher()

	cancel := NewMutationHandler(func(event MutationEvent) bool {
		cancelFn()
		return false
	}).RunOnce()

	end := NewMutationHandler(func(evt MutationEvent) bool {
		switch t := evt.NewValue().(type) {
		case String:
			if m := t.String(); m == "cancelled" || m == "error" {
				e.fetchCompleted(propname, false)
			}
			// TODO what is the string value is somethign else? Would it have been intercepted by
			// a transition handler for the end phase?
		case Bool:
			if t.Bool() {
				evt.Origin().fetchCompleted(propname, true)
			} else {
				evt.Origin().fetchCompleted(propname, false)
			}
		}
		return false
	}).RunOnce()

	e.newFetchTransition(propname, fetch, nil, cancel, end)

}

func (e *Element) SetURLDataFetcher(propname string, url string, responsehandler func(*http.Response) (Value, error), prefetchable bool) {
	router := GetRouter(e.Root)
	r, err := http.NewRequestWithContext(router.NavContext, "GET", url, nil)
	if err != nil {
		panic(url + " might be malformed. Unable to create new request")
	}
	e.SetDataFetcher(propname, func(*Element) *http.Request { return r }, responsehandler, prefetchable)
}

// CancelFetch will abort ongoing fetch requests.
func (e *Element) CancelAllFetches() {
	// iterate through alln props registered for fetching (runtime fetchlist)
	// cancel the fetch transition for each of them
	f, ok := e.Get("runtime", "fetchlist")
	if !ok {
		return
	}
	fetchlist := f.(Object).MustGetList("fetch_index")
	for _, propname := range fetchlist.UnsafelyUnwrap() {
		e.CancelFetch(propname.(String).String())
	}
}

func (e *Element) CancelFetch(propname string) {
	e.cancelfetchTransition(propname)
}

func (e *Element) cancelPrefetch(propname string) {
	e.cancelprefetchTransition(propname)
}

// WasFetchCancelled answers the question of whether a fecth was cancelled or not.
// It can be used when handling a "fetched" event (OnFetched) to differentiate fetching failure
// from fetching cancellation.
func (e *Element) WasFetchCancelled() bool {
	_, ok := e.Get("fetchstatus", "cancelled")
	return ok
}

func verb(v Value) string {
	switch t := v.(type) {
	case String:
		return t.String()
	case Object:
		return t.MustGetString("verb").String()
	default:
		return "unknown"
	}
}

func (e *Element) fetchData(propname string, r *http.Request, cancelFn context.CancelFunc, responsehandler func(*http.Response) (Value, error), prefetch bool) {
	if responsehandler == nil {
		panic("response handler is not specified. Will not be able to process the data fetching request")
	}
	var prefetching = prefetch

	if e.isPrefetchedDataValid(propname) {
		if !prefetching {
			e.fetchCompleted(propname, true)
		}
		return
	} else if !prefetching {
		e.cancelPrefetch(propname)
	}

	DoAsync(e, func(execution context.Context) {

		DoSync(func() {
			go func() {
				select {
				case <-execution.Done():
					cancelFn()
				case <-r.Context().Done():
					// the goroutine should be done now
				}
			}()
		})

		res, err := e.Root.HttpClient.Do(r)
		if err != nil {
			DoSync(func() {
				if prefetching {
					e.endprefetchTransition(propname, Bool(false))
				} else {
					e.pushFetchError(propname, err)
					e.errorfetchTransition(propname)
				}
			})
			return
		}
		defer res.Body.Close()

		v, err := responsehandler(res)
		if err != nil {
			DoSync(func() {
				if prefetching {
					e.endprefetchTransition(propname, Bool(false))
				} else {
					e.pushFetchError(propname, err)
					e.errorfetchTransition(propname)
				}
			})
			return
		}
		DoSync(func() {
			e.SetData(propname, v)
			if prefetching {
				e.endprefetchTransition(propname)
			} else {
				e.endfetchTransition(propname)
			}
		})
	})
}

/*
func cloneReq(r *http.Request) (*http.Request){
	r:= r.Clone(r.Context())
	if r.Body == io.ReadCloser(nil){
		return r
	}
	body,err:= io.ReadAll(r.Body)
	if err!= nil{
		panic(err)
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	r.Body = io.NopCloser(bytes.NewReader(body))
	return r
}
*/

// enablefetching adds fetch transition support to UI elements.
func (e *Element) enablefetching() *Element {
	prefetch := NewMutationHandler(func(evt MutationEvent) bool {
		// prefetchstart by iterating on prefetchlist
		fl, ok := evt.Origin().Get("runtime", "prefetchlist")
		if !ok {
			e.EndTransition("prefetch")
			return false
		}
		prefetchlist, ok := fl.(Object)
		if !ok {
			panic("unexpected prefetchlist type")
		}

		if prefetchlist.Size() == 0 {
			e.EndTransition("prefetch")
		}

		prefetchlist.Range(func(propname string, v Value) bool {
			e.startprefetchTransition(propname)
			return false
		})

		return false
	})

	fetch := NewMutationHandler(func(evt MutationEvent) bool {
		// fetchstart by iterating on fetchlist
		fl, ok := evt.Origin().Get("runtime", "fetchlist")
		if !ok {
			e.EndTransition("fetch")
			return false
		}
		fetchlist := fl.(Object).MustGetList("fetch_index")
		if len(fetchlist.UnsafelyUnwrap()) == 0 {
			e.EndTransition("fetch")
		}
		for _, v := range fetchlist.UnsafelyUnwrap() {
			propname := v.(String).String()
			e.OnTransitionError(strings.Join([]string{"fetch", propname}, "-"), NewMutationHandler(func(evt MutationEvent) bool {
				e.errorfetchTransition(propname)
				return false
			}))
			e.startfetchTransition(propname)
		}

		return false
	})

	cancel := NewMutationHandler(func(evt MutationEvent) bool {
		evt.Origin().CancelAllFetches()
		return false
	})

	e.DefineTransition("prefetch", prefetch, nil, nil, nil)
	e.DefineTransition("fetch", fetch, nil, cancel, nil)

	e.OnMounted(NewMutationHandler(func(evt MutationEvent) bool {
		evt.Origin().Fetch()
		return false
	}))

	return e
}

func (e *Element) Prefetch() {
	if !e.Registered() {
		panic("Prefetch can only be called on registered elements")
	}
	e.Root.WatchEvent("document-loaded", e.Root, NewMutationHandler(func(evt MutationEvent) bool {
		// should start the prefetching process by triggering the prefetch transitions that have been registered
		e.StartTransition("prefetch")
		return false
	}).RunASAP().RunOnce())
}

func (e *Element) Fetch(props ...string) {
	if !e.Registered() {
		panic("Fetch can only be called on registered elements. Error for " + e.ID)
	}

	// The Fetch will only proceed once a document tree is fully created, i.e. once the document-loaded event
	// has fired
	e.Root.WatchEvent("document-loaded", e.Root, NewMutationHandler(func(evt MutationEvent) bool {
		if len(props) == 0 {
			e.Properties.Delete("runtime", "fetcherrors")
			//e.Properties.Delete("fetchstatus","cancelled")

			// should start the fetching process by triggering the fetch transitions that have been registered
			e.StartTransition("fetch")
			return false
		}
		for _, prop := range props {
			e.startfetchTransition(prop)
		}

		return false
	}).RunASAP().RunOnce())
}

func (e *Element) ForceFetch() {
	e.InvalidateAllFetches()
	e.Fetch()
}

func (e *Element) OnFetch(h *MutationHandler) {
	e.OnTransitionStart("fetch", h)
}

func (e *Element) OnFetchCancel(h *MutationHandler) {
	e.OnTransitionCancel("fetch", h)
}

func (e *Element) OnFetched(h *MutationHandler) {
	e.OnTransitionEnd("fetch", h)
}

func (e *Element) OnFetchError(h *MutationHandler) {
	e.OnTransitionError("fetch", h)
}

func (e *Element) InvalidateFetch(propname string) {
	if e.Registered() {
		e.invalidatePrefetch(propname)
		l, ok := e.Get("runtime", "fetchlist")
		if !ok {
			return
		}
		r := l.(Object)
		fs, ok := r.Get(propname)
		if !ok {
			return
		}
		s := fs.(Object)
		s = s.MakeCopy().Set("stale", Bool(true)).Commit()
		r = r.MakeCopy().Set(propname, s).Commit()

		e.Set("runtime", "fetchlist", r)
		e.CancelFetch(propname)
	}
}

func (e *Element) InvalidateAllFetches() {
	if e.Registered() {
		l, ok := e.Get("runtime", "fetchlist")
		if !ok {
			return
		}
		fl := l.(Object)

		fl.Range(func(propname string, v Value) bool{
			e.InvalidateFetch(propname)
			return false
		})
	}
}

// GetFetchErrors returns, if it exists, a map where each propname key whose fetch failed has a corresponding
// error. Useful to implement retries.
func GetFetchErrors(e *Element) (map[string]error, bool) {
	v, ok := e.Get("runtime", "fetcherrors")
	if !ok {
		return nil, ok
	}
	m := make(map[string]error)

	v.(Object).Range(func(propname string, v Value) bool {
		m[propname] = errors.New(string(v.(String)))
		return false
	})
	return m, ok

}

func (e *Element) pushFetchError(propname string, err error) {
	var errlist Object
	v, ok := e.Get("runtime", "fetcherrors")
	if !ok {
		errlist = NewObject().Set(propname, String(err.Error())).Commit()
	} else {
		r := v.(Object)
		errlist = r.MakeCopy().Set(propname, String(err.Error())).Commit()
	}
	e.Set("runtime", "fetcherrors", errlist)
}

func (e *Element) registerfetch(propname string) {
	var fetchlist = NewObject()
	var fetchindex = NewList()

	o, ok := e.Get("runtime", "fetchlist")
	if ok {
		fetchlist = o.(Object).MakeCopy()
		_, ok := fetchlist.Get("fetch_index")
		if !ok {
			panic("Framework error: fetch index missing")
		}
	}

	no := NewObject().Commit()

	fetchindex = fetchindex.Append(String(propname))
	fetchlist = fetchlist.Set("fetch_index", fetchindex.Commit()).Set(propname, no)
	e.Set("runtime", "fetchlist", fetchlist.Commit())
}

// Note that if an error occured during the fetching process (e.g. on of the fetch failed),
// the process is not considered of having completed.
// The fetch transition is not ended to allow for retries in the error state.

func (e *Element) checkFetchCompletion() {
	completed := true

	cl, ok := e.Get("runtime", "fetchlist")
	if !ok {
		panic("FAIL: fetchlist object should be present")
	}
	l := cl.(Object)
	rindex, ok := l.Get("fetch_index")
	if !ok {
		panic("Framework error: fetch index missing")
	}
	index := rindex.(List)
	for _, prop := range index.UnsafelyUnwrap() {
		propname := string(prop.(String))
		status, ok := l.Get(propname)
		if !ok {
			return
		}
		s := status.(Object)
		st, ok := s.Get("status")
		if !ok {
			// this is not completed
			return
		}
		stat := st.(String).String()
		if stat == "successful" {
			// this is not completed, this is in a failed state.. the transietion is not ended.
			continue
		}

		completed = false
		break
	}
	if completed {
		e.EndTransition("fetch")
	}
}

func (e *Element) registerPrefetch(propname string) {
	l := NewObject()
	p := NewObject()
	e.Set("runtime", "prefetchlist", l.Set(propname, p.Commit()).Commit())
}

func (e *Element) isPrefetchedDataValid(propname string) bool {
	l, ok := e.Get("runtime", "prefetchlist")
	if !ok {
		return false
	}
	r := l.(Object)
	state, ok := r.Get(propname)
	if !ok {
		return false
	}
	s := state.(Object)
	status, ok := s.Get("status")
	if !ok {
		return false
	}
	if sts := string(status.(String)); sts != "successful" {
		return false
	}
	t, ok := s.Get("timestamp")
	if !ok {
		return false
	}
	ts := string(t.(String))
	temps, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return false
	}
	if time.Now().UTC().After(temps.UTC().Add(PrefetchMaxAge)) {
		return false
	}
	return true
}

func (e *Element) prefetchCompleted(propname string, successfully bool) {
	l, ok := e.Get("runtime", "prefetchlist")
	if !ok {
		panic("Failed to find list of initiated prefetches.")
	}
	r := l.(Object)
	fs, ok := r.Get(propname)
	if ok {
		s := fs.(Object)
		if !successfully {
			s = s.MakeCopy().
				Set("status", String("failed")).
				Commit()
			r = r.MakeCopy().Set(propname, s).Commit()
		} else {
			s = s.MakeCopy().
				Set("status", String("successful")).
				Set("timestamp", String(time.Now().UTC().Format(time.RFC3339))).
				Commit()
			r = r.MakeCopy().Set(propname, s).Commit()
		}
	}

	e.Set("runtime", "prefetchlist", r)
}

func (e *Element) invalidatePrefetch(propname string) {
	l, ok := e.Get("runtime", "prefetchlist")
	if !ok {
		panic("Failed to find list of initiated prefetches.")
	}
	r := l.(Object)
	fs, ok := r.Get(propname)
	if ok {
		s := fs.(Object)
		s = s.MakeCopy().Set("status", String("stale")).Commit()
		r = r.MakeCopy().Set(propname, s).Commit()
	}
	e.Set("runtime", "prefetchlist", r)
	e.cancelPrefetch(propname)
}

func (e *Element) fetchCompleted(propname string, successfully bool) {
	l, ok := e.Get("runtime", "fetchlist")
	if !ok {
		panic("Failed to find list of initiated fetches.")
	}
	r := l.(Object)
	fs, ok := r.Get(propname)
	if ok {
		s := fs.(Object)
		ts := s.MakeCopy()
		if !successfully {
			ts.Set("status", String("failed"))
		} else {
			ts.Set("status", String("successful"))
		}
		ts.Delete("stale")
		r = r.MakeCopy().
			Set(propname, ts.Commit()).
			Commit()
	}

	e.Set("runtime", "fetchlist", r)
	e.checkFetchCompletion()
}

func (e *Element) isFetchedDataValid(propname string) bool {
	l, ok := e.Get("runtime", "fetchlist")
	if !ok {
		return false
	}
	r := l.(Object)
	state, ok := r.Get(propname)
	if !ok {
		return false
	}
	s := state.(Object)
	status, ok := s.Get("status")
	if !ok {
		return false
	}
	if sts := string(status.(String)); sts != "successful" {
		return false
	}

	stale, ok := s.Get("stale")
	if !ok {
		return true
	}
	return !bool(stale.(Bool))
}

// Fetch transition helpers

func (e *Element) newFetchTransition(propname string, onstart, onerror, oncancel, onend *MutationHandler) {
	e.DefineTransition(strings.Join([]string{"fetch", propname}, "-"), onstart, onerror, oncancel, onend)
}

func (e *Element) newPrefetchTransition(propname string, onstart, onerror, oncancel, onend *MutationHandler) {
	e.DefineTransition(strings.Join([]string{"prefetch", propname}, "-"), onstart, onerror, oncancel, onend)
}

func (e *Element) startfetchTransition(propname string) {
	e.StartTransition(strings.Join([]string{"fetch", propname}, "-"))
}

func (e *Element) startprefetchTransition(propname string) {
	e.StartTransition(strings.Join([]string{"prefetch", propname}, "-"))
}

func (e *Element) errorfetchTransition(propname string) {
	e.ErrorTransition(strings.Join([]string{"fetch", propname}, "-"))
}

func (e *Element) cancelfetchTransition(propname string) {
	e.CancelTransition(strings.Join([]string{"fetch", propname}, "-"))
}

func (e *Element) cancelprefetchTransition(propname string) {
	e.CancelTransition(strings.Join([]string{"prefetch", propname}, "-"))
}

func (e *Element) endprefetchTransition(propname string, values ...Value) {
	e.EndTransition(strings.Join([]string{"prefetch", propname}, "-"), values...)
}

func (e *Element) endfetchTransition(propname string, values ...Value) {
	e.EndTransition(strings.Join([]string{"fetch", propname}, "-"), values...)
}

func fetchTxName(propname, phase string) string {
	return transition(strings.Join([]string{"fetch", propname}, "-"), phase)
}

func prefetchTxName(propname, phase string) string {
	return transition(strings.Join([]string{"prefetch", propname}, "-"), phase)
}

//  Making requests at the Element level

// An Element should also be able to send requests to a remote server besides retrieving data
// via GET (POST, PUT, PATCH,  UPDATE, DELETE)
// When such a request is made to an endpoint, the Data Fetched should be invalidated and refetched.
// This is because the data may have changed on the server side.
//
// The Element should be part of a document i.e. registered.
// NewRequest makes a http Request using the default client
// If the request needs to be cancelled on navigation, the context
// on the request value should be set to be a navigation context
// such as the one stored on the document object.
func (e *Element) NewRequest(r *http.Request, responsehandler func(*http.Response) (Value, error)) {
	if !e.Registered() {
		panic("Element is not registered. Cannot process request")
	}

	e.Root.WatchEvent("document-loaded", e.Root, NewMutationHandler(func(evt MutationEvent) bool {
		var ctx context.Context
		var cancelFn context.CancelFunc

		onstart := NewMutationHandler(func(evt MutationEvent) bool {
			ctx, cancelFn = context.WithCancel(r.Context())
			r = r.WithContext(ctx)

			e.Properties.Delete("event", "request-error-"+requestID(r))

			DoAsync(e, func(execution context.Context) {

				DoSync(func() {
					go func() {
						select {
						case <-execution.Done():
							cancelFn()
						case <-ctx.Done():
						}
					}()
				})

				res, err := e.Root.HttpClient.Do(r)
				if err != nil {
					DoSync(func() {
						e.TriggerEvent("request-error-"+requestID(r), newRequestStateObject(nil, err))
					})
					return
				}
				defer res.Body.Close()
				if responsehandler == nil {
					return
				}
				v, err := responsehandler(res)
				if err != nil {
					DoSync(func() {
						e.TriggerEvent("request-error-"+requestID(r), newRequestStateObject(nil, err))
					})
					return
				}
				DoSync(func() {
					e.endrequestTransition(r.URL.String(), newRequestStateObject(v, nil))
				})
			})
			return false
		}).RunOnce()

		onerror := NewMutationHandler(func(evt MutationEvent) bool {
			return false
		}).RunOnce()

		oncancel := NewMutationHandler(func(evt MutationEvent) bool {
			cancelFn()
			return false
		}).RunOnce()

		onend := NewMutationHandler(func(evt MutationEvent) bool {
			// initially thought that we could do nothing if req was canceleld or on error
			// but in fact it doesn't matter because a request in flight may still have mutated data on
			// the serveer
			// the clien only controls the request.
			evt.Origin().Root.TriggerEvent("request-"+requestID(r), String(r.Method))
			return false
		}).RunOnce()

		e.newRequestTransition(requestID(r), onstart, onerror, oncancel, onend)

		e.OnRequestError(r, NewMutationHandler(func(evt MutationEvent) bool {
			evt.Origin().OnRequestError(r, NewMutationHandler(func(event MutationEvent) bool {
				event.Origin().endrequestTransition(r.URL.String(), event.NewValue())
				return false
			}).RunOnce())
			return false
		}).RunOnce())

		e.startrequestTransition(requestID(r))

		return false
	}).RunOnce().RunASAP())

}

func (e *Element) CancelRequest(r *http.Request) {
	e.cancelrequestTransition(r.URL.String())
}

func newRequestStateObject(value Value, err error) Object {
	r := NewObject()
	r.Set("value", value)
	if err != nil {
		r.Set("error", String(err.Error()))
	}

	return r.Commit()
}

// requestID provides a base32 encoding of an URL, after removal of the query string
func requestID(r *http.Request) string {
	u, err := url.Parse(r.URL.String())
	if err != nil {
		panic(err)
	}
	u.RawQuery = ""
	eurl := base32.StdEncoding.EncodeToString([]byte(u.String()))
	return eurl
}

func (e *Element) OnRequestStart(r *http.Request, h *MutationHandler) {
	e.OnTransitionStart("request-"+requestID(r), h)
}

func (e *Element) OnRequestCancel(r *http.Request, h *MutationHandler) {
	e.OnTransitionCancel("request-"+requestID(r), h)
}

func (e *Element) OnRequestEnd(r *http.Request, h *MutationHandler) {
	e.OnTransitionEnd("request-"+requestID(r), h)
}

func (e *Element) OnRequestError(r *http.Request, h *MutationHandler) {
	e.WatchEvent("request-error-"+requestID(r), e, h)
}

// RetrieveResponse returns the response received for a request if it exists.
// Otherwise it returns nil.
// It is typically used when handling OnRequestEnd.
func RetrieveResponse(e *Element, r *http.Request) (Value, error) {
	v, ok := e.Get("event", transition("request-"+requestID(r), "end"))
	if !ok {
		return nil, nil
	}
	return newResponseObject(v)
}

func newResponseObject(u Value) (Value, error) {
	o, ok := u.(Object)
	if !ok {
		panic("value used as response object should be of type Object")
	}
	rv, ok := o.Get("value")
	if !ok {
		panic(" expected value field in response object")
	}
	es, ok := o.Get("error")
	if !ok {
		return rv, nil
	}
	err := errors.New(string(es.(String)))
	return rv, err
}

// SyncUISyncDataOptimistically sets a data property optimistically on transition start.
// If the transition doesn't end successfully (it was cancelled or errored out) the property is reverted
// to its former value.
// Since a new request always cancels the previous one, there should not be any clobbering of data updates.

func (e *Element) SyncUISyncDataOptimistically(propname string, value Value, r *http.Request, responsehandler ...func(*http.Response) (Value, error)) {
	oldv, _ := e.GetData(propname)
	if Equal(oldv, value) {
		return
	}
	e.SyncUI(propname, value)

	e.OnRequestError(r, NewMutationHandler(func(evt MutationEvent) bool {
		e.SetUI(propname, oldv)
		err := NewObject().Set("prop", String(propname)).Set("value", value).Commit()
		e.TriggerEvent("optimisticmutationerror", err)
		return false
	}).RunOnce())

	e.OnRequestCancel(r, NewMutationHandler(func(evt MutationEvent) bool {
		e.SetUI(propname, oldv)
		return false
	}).RunOnce())

	e.OnRequestEnd(r, NewMutationHandler(func(evt MutationEvent) bool {
		truev, _ := e.GetUI(propname)
		e.SetData(propname, truev)
		return false
	}).RunOnce())

	if responsehandler != nil {
		e.NewRequest(r, responsehandler[0])
	} else {
		e.NewRequest(r, nil)
	}
}

func (e *Element) OnOptimisticMutationError(h *MutationHandler) {
	e.WatchEvent("optimisticmutationerror", e, h)
}

func (e *Element) newRequestTransition(requestID string, onstart, onerror, oncancel, onend *MutationHandler) {
	e.DefineTransition("request-"+requestID, onstart, onerror, oncancel, onend)
}

func (e *Element) startrequestTransition(requestID string) {
	e.StartTransition("request-" + requestID)
}

func (e *Element) cancelrequestTransition(requestID string) {
	e.CancelTransition("request-" + requestID)
}

func (e *Element) endrequestTransition(requestID string, values ...Value) {
	e.EndTransition("request-"+requestID, values...)
}
