package ui

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"sync"
	"time"
)

const(
	runningFetch = "started"
	successfulFetch = "successful"
	abortedFetch = "aborted"
	failedFetch	= "failed"
)

var HttpClient = http.DefaultClient
var  PrefetchMaxAge = 5 * time.Second

// SetHttpClient allows for the use of a custom http client.
// It changes the value of HttpClient whose default value is the default Go http Client.
func SetHttpClient(c *http.Client){
	HttpClient = c
}


// Lock protects against unserialized concurrent UI tree access. It should enable the preservation 
// of the ordering of mutations.
var Lock = &sync.Mutex{}

// WorkQueue is a queue of UI mutating function that can be built from multiple goroutines.
// Only the UI thread read from this to do work on the UI tree.
var WorkQueue = make(chan func())

// Do sends a function to the main goroutine that is in charge of the UI to be run.
// Goroutines launched from the main thread that need access to the main UI tree should use it.
func Do(fn func()){
	go func(){
		WorkQueue <- fn
	}()
}

// NewCriticalSection returns a special function used to run another function.
// It is special because it ensures that the function being run has sole access to the UI tree.
// It essentially disallows concurrent mutations of the UI.
// Typically, it should be used in goroutines that need to modify a Ui ELement by setting data 
// for instance.
// A critical function can only be called once. After being called, it turns into a noop.
// This is consistent with the fact that a goroutine should only use one critical section.
// Of course, it is still possible to create another critical section function: don't do that.
func NewCriticalSection() func(func()){
	var once sync.Once
	cs:= func(f func()){
		Lock.Lock()
		once.Do(f)
		Lock.Unlock()
	}
	return cs
}


// WithFetchedData allows an element to retrieve data by sending a http Get request as soon as it gets mounted.
// It accepts a function as argument that is tasked with converting the *http.Response into 
// a Value that can be stored as an element property.
// Unless stated otherwise, the data is made prefetchable as well.
// The data is set asynchronously.
//
// The fetching occurs during the "fetch" event ("event","fetch") that is triggered each time an element
// is mounted.
func(e *Element) WithFetchedData(propname string, req *http.Request, responsehandler func(*http.Response)(Value,error), noprefetch ...bool) {
	prefetchable:= true
	if noprefetch != nil{
		prefetchable = false
	}
	

	if prefetchable{
		if !e.isPrefetchable(propname){
			e.makePrefetchable(propname)
			e.Watch("event","prefetch",e,NewMutationHandler(func(evt MutationEvent)bool{
				r:= cloneReq(req)
				ctx,cancelFn:= context.WithCancel(r.Context())
				r = r.WithContext(ctx)

				e.Watch("event","cancelprefetchrequests",e,NewMutationHandler(func(evt MutationEvent)bool{
					cancelFn()
					return false
				}).RunOnce())

				evt.Origin().fetchData(propname,r,responsehandler,true)
				return false
			}))
		}
	}
	e.OnFetch(NewMutationHandler(func(evt MutationEvent) bool{
		r:= cloneReq(req)
		ctx,cancelFn:= context.WithCancel(r.Context())
		r = r.WithContext(ctx)

		e.Watch("event","cancelfetchrequests",e,NewMutationHandler(func(evt MutationEvent)bool{
			cancelFn()
			return false
		}).RunOnce())

		e.fetchData(propname,r,responsehandler,false)
		return false
	}))             

}

func(e *Element) WithFetchedDataFromURL(propname string, url string, responsehandler func(*http.Response)(Value,error), noprefetch ...bool){
	req,err:= http.NewRequestWithContext(NavContext,"GET",url,nil)
	if err!= nil{
		panic(url + " is malformed most likely. Unable to create new request")
	}
	e.WithFetchedData(propname,req,responsehandler,noprefetch...)
}

// CancelFetch will abort ongoing fetch requests.
func(e *Element) CancelFetch(){
	e.Set("event","cancelfetchrequests", Bool(true))
	e.Set("event","fetchcancelled",Bool(true))
}

func(e *Element) cancelPrefetch(){
	e.Set("event","cancelprefetchrequests",Bool(true))
}


// CancelFetchOnError is an Element modifier that automatically aborts all ongoing fetches as soon 
// as one failed.
// It is not the default so as to leave the possibility to implement retries.
func CancelFetchOnError(e *Element) *Element{
	e.OnFetched(NewMutationHandler(func(evt MutationEvent)bool{
		o:= evt.OldValue().(Bool)
		n:= evt.NewValue().(Bool)
		if !n && o{
			e.Set("event","cancelfetchrequests", Bool(true))
		}
		return false
	}))

	return e
}

// WasFetchCancelled answers the question of whether a fecth was cancelled or not.
// It can be used when handling a "fetched" event (OnFetched) to differentiate fetching failure
// from fetching cancellation.
func(e *Element) WasFetchCancelled() bool{
	_,ok:= e.Get("event","fetchcancelled")
	return ok
}


func(e *Element) fetchData(propname string, req *http.Request, responsehandler func(*http.Response) (Value,error), prefetch bool){
	var prefetching = prefetch

	if e.isPrefetchedDataValid(propname){
		if !prefetching{
			e.fetchCompleted(propname,true)
		}
		return
	} else if !prefetching{
		e.cancelPrefetch()
	}

	// Register new fetch in fetchlist unless fetch is already running
	startfetch:= e.initFetch(propname)

	if !startfetch{
		return
	}

	
	go func(){
		res, err:= HttpClient.Do(req)
		if err!= nil{
			Do(func() {
				if prefetching{
					e.prefetchCompleted(propname,false)
				}else{
					e.pushFetchError(propname,err)
					e.fetchCompleted(propname,false)
				}	
			})
			return
		}
		defer res.Body.Close()

		v,err:= responsehandler(res)
		if err!= nil{
			Do(func() {
				if prefetching{
					e.prefetchCompleted(propname,false)
				}else{
					e.pushFetchError(propname,err)
					e.fetchCompleted(propname,false)
				}	
			})
			return
		}
		Do(func() {
			e.SetData(propname,v)
			if prefetching{
				e.prefetchCompleted(propname,true)
			}else{
				e.fetchCompleted(propname,true)
			}
		})
	}()
}

func cloneReq(req *http.Request) (*http.Request){
	r:= req.Clone(req.Context())
	if req.Body == io.ReadCloser(nil){
		return r
	}
	body,err:= io.ReadAll(req.Body)
	if err!= nil{
		panic(err)
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	r.Body = io.NopCloser(bytes.NewReader(body))
	return r
}

func(e *Element) Prefetch(){
	e.Set("event","prefetch",Bool(true))
}

func(e *Element) Fetch(){
	e.Properties.Delete("runtime","fetcherrors")
	e.Properties.Delete("event","fetchcancelled")
	e.Set("event","fetch",Bool(true))
}

func(e *Element) OnFetch(h *MutationHandler){
	e.Watch("event","fetch",e,h)
}

func(e *Element) OnFetched(h *MutationHandler){
	e.Watch("event","fetched",e,h)
}


// FetchErrors returns, if it exists, a map where each propname key whose fetch failed has a corresponding
// error string.
func FetchErrors(e *Element) (map[string]string,bool){
	v,ok:= e.Get("runtime","fetcherrors")
	if !ok{
		return nil,ok
	}
	m:= make(map[string]string)
	for k,val:= range v.(Object){
		m[k]= string(val.(String))
	}
	return m,ok
}



func(e *Element) pushFetchError(propname string, err error){
	var errlist Object
	v,ok:= e.Get("runtime","fetcherrors")
	if !ok{
		errlist= NewObject().Set(propname, String(err.Error()))
	} else{
		errlist= v.(Object).Set(propname, String(err.Error()))
	}
	e.Set("runtime","fetcherrors",errlist)
}

func(e *Element) initFetch(propname string )bool{
	var fetchlist Object
	var fetchindex List

	o,ok:= e.Get("runtime","fetchlist")
	if ok{
		fetchlist= o.(Object)
		fi,ok:= fetchlist.Get("fetch_index")
		if !ok{
			panic("Framework error: fetch index missing")
		}
		fetchindex= fi.(List)
		fso,ok:= fetchlist.Get(propname)
		if ok{
			fs,ok:= fso.(Object).Get("status")
			if ok{
				if string(fs.(String)) == runningFetch{
					return false
				}
			}
		}
	}else{
		fetchlist = NewObject()
		fetchindex = NewList()
	}
	f:= NewObject()
	f.Set("status",String("started"))
	
	fetchindex = append(fetchindex,String(propname))
	fetchlist.Set("fetch_index",fetchindex)
	fetchlist.Set(propname,f)
	e.Set("runtime","fetchlist",fetchlist)
	return true
}

func(e *Element) checkFetchCompletion(){
	completed := true

	cl,ok:= e.Get("runtime","fetchlist")
	if !ok{
		panic("FAIL: fetchlist object should be present")
	}
	l:= cl.(Object)
	rindex,ok:= l.Get("fetch_index")
	if !ok{
		panic("Framework error: fetch index missing")
	}
	index:= rindex.(List)
	for _,prop:= range index{
		propname := string(prop.(String))
		status,ok:= l.Get(propname)
		if !ok{
			panic("FAIL: a listed propname does not have a status object. This is a error of particleui")
		}
		s:= status.(Object)
		st,ok:= s.Get("status")
		if !ok{
			panic("FAIL: missing status field for the fetch object. This is a framwework error")
		}
		if stat:= string(st.(String)); stat != runningFetch{
			if stat != successfulFetch{
				e.Set("event","fetched", Bool(false))
				return
			}
			continue
		}
		completed = false
	}
	if completed{
		e.Set("event","fetched", Bool(completed))
	}
}


func(e *Element) makePrefetchable(propname string){
	l:= NewObject()
	p:= NewObject()
	l.Set(propname,p)
	e.Set("runtime","prefetchlist",l)
}

func(e *Element)isPrefetchable(propname string) bool{
	l,ok:=e.Get("runtime","prefetchlist")
	if !ok{
		return false
	}
	r:= l.(Object)
	_,ok= r.Get(propname)
	return ok
}

func(e *Element) isPrefetchedDataValid(propname string) bool{
	l,ok:=e.Get("runtime","prefetchlist")
	if !ok{
		return false
	}
	r:= l.(Object)
	state,ok:= r.Get(propname)
	if !ok{
		return false
	}
	s:= state.(Object)
	status,ok:= s.Get("status")
	if !ok{
		return false
	}
	if sts:= string(status.(String)); sts != "successful"{
		return false
	}
	t,ok:= s.Get("timestamp")
	if !ok{
		return false
	}
	ts:= string(t.(String))
	temps,err:= time.Parse(time.RFC3339, ts)
	if err!= nil{
		return false
	}
	if time.Now().UTC().After(temps.UTC().Add(PrefetchMaxAge)){
		return false
	}
	return true
}

func(e *Element) prefetchCompleted(propname string, successfully bool){
	l,ok:=e.Get("runtime","prefetchlist")
	if !ok{
		panic("Failed to find list of initiated prefetches.")
	}
	r:= l.(Object)
	fs,ok:= r.Get(propname)
	if ok{
		s:= fs.(Object)
		if !successfully{
			s.Set("status", String("failed"))
		} else{
			s.Set("status", String("successful"))
			s.Set("timestamp", String(time.Now().UTC().Format(time.RFC3339)))
		}		
	}
	e.Set("runtime","prefetchlist",r)
}

func(e *Element) fetchCompleted(propname string, successfully bool){
	l,ok:=e.Get("runtime","fetchlist")
	if !ok{
		panic("Failed to find list of initiated prefetches.")
	}
	r:= l.(Object)
	fs,ok:= r.Get(propname)
	if ok{
		s:= fs.(Object)
		if !successfully{
			s.Set("status", String("failed"))
		} else{
			s.Set("status", String("successful"))
		}
	}
	e.Set("runtime", "fetchlist",r)
}
