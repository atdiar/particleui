package ui

import (
	"bytes"
	"context"
	"encoding/base32"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

const(
	runningFetch = "started"
	successfulFetch = "successful"
	abortedFetch = "aborted"
	failedFetch	= "failed"

	// ReplayNavMode is a navigation mode. When navigation is in replay mode, it means that
	// all navocontext-aware async goroutines are cancelled icnludind outbound fetch calls.
	ReplayNavMode = "replay"
	NormalNavMode = "normal"
)

func init(){
	SetHttpClient(HttpClient)
}

var HttpClient = http.DefaultClient
var CookieJar *cookiejar.Jar
var  PrefetchMaxAge = 5 * time.Second

// SetHttpClient allows for the use of a custom http client.
// It changes the value of HttpClient whose default value is the default Go http Client.
func SetHttpClient(c *http.Client){
	c.Jar = CookieJar
	HttpClient = c
}


// WorkQueue is a queue of UI mutating function that can be built from multiple goroutines.
// Only the UI thread read from this to do work on the UI tree.
var WorkQueue = make(chan func())

// DoSync sends a function to the main goroutine that is in charge of the UI to be run.
// Goroutines launched from the main thread that need access to the main UI tree must use it.
// Only a sincgle DOSync must be used within a DoAsync.
func DoSync(fn func()){
	go func(){
		WorkQueue <- fn
	}()
}

// DoAsync pushes a function onto a goroutine for execution, as long as navigation is still valid. 
// Instead of launching raw goroutines, one should use this wrapper for any concurrent processing that 
// is tied to navigation. For example, when triggering  http.Requests to fetch data for a given route.
// Elements must not be accessed in a DoAsync unless a DoSync is used to push changes back to the main
// goroutine.
func DoAsync(f func()){
	go func(){
		select{
		case <-NavContext.Done():
			return 
		default:
			f()
		}
	}()
}

// NewDataFetcher allows an element to retrieve data by sending a http Get request as soon as it gets mounted.
// It accepts a function as argument that is tasked with converting the *http.Response into 
// a Value that can be stored as an element property.
// Unless stated otherwise, the data is made prefetchable as well.
// The data is set asynchronously.
//
// The fetching occurs during the "fetch" event ("event","fetch") that is triggered each time an element
// is mounted.
func(e *Element) NewDataFetcher(propname string, req *http.Request, responsehandler func(*http.Response)(Value,error), noprefetch ...bool) {
	e.RemoveDataFetcher(propname)
	prefetchable:= true
	if noprefetch != nil{
		prefetchable = false
	}
	

	if prefetchable{
		if !e.isPrefetchable(propname){
			e.makePrefetchable(propname)
			
			

			prefetchhandler := NewMutationHandler(func(evt MutationEvent)bool{
				if  evt.Origin().isFetchedDataValid(propname){
					return false
				}
				if evt.Origin().isPrefetchedDataValid(propname){
					return false
				}
				
				r:= cloneReq(req)
				ctx,cancelFn:= context.WithCancel(r.Context())
				r = r.WithContext(ctx)

				evt.Origin().OnDeleted(NewMutationHandler(func(event MutationEvent)bool{
					cancelFn()
					return false
				}).RunOnce())

				oncancelprefetch:= NewMutationHandler(func(event MutationEvent)bool{
					cancelFn()
					return false
				}).RunOnce()

				var fetchcancelerremover *MutationHandler
				fetchcancelerremover= NewMutationHandler(func(event MutationEvent)bool{
					p:= string(event.NewValue().(String))
					if p != propname{
						return true
					}
					event.Origin().RemoveMutationHandler("event","cancelprefetchrequests",event.Origin(),oncancelprefetch)
					event.Origin().RemoveMutationHandler("event","removedatafetcher",event.Origin(),fetchcancelerremover)
					return false
				})
				evt.Origin().WatchEvent("removedatafetcher", evt.Origin(), fetchcancelerremover)

				evt.Origin().Watch("event","cancelprefetchrequests",evt.Origin(),oncancelprefetch)


				// After a new http.Request has been launched and a response has been returned, cancel and refetch
				// the data corresponding to the req.URL
				evt.Origin().WatchEvent(newRequestEventName("end",r.URL.String()),evt.Origin().ElementStore.Global,NewMutationHandler(func(event MutationEvent)bool{
					//event.Origin().invalidatePrefetch()
					cancelFn()
					return false
				}).RunOnce())

				evt.Origin().fetchData(propname,r,responsehandler,true)
				return false
			})

			var dataprefetcherremover *MutationHandler
			dataprefetcherremover=NewMutationHandler(func(evt MutationEvent)bool{
				p:= string(evt.NewValue().(String))
				if p != propname{
					return true
				}
				evt.Origin().RemoveMutationHandler("event","prefetch",evt.Origin(),prefetchhandler)
				evt.Origin().RemoveMutationHandler("event","removedatafetcher",evt.Origin(),dataprefetcherremover)
				return false
			})
			e.WatchEvent("removedatafetcher", e, dataprefetcherremover)

			e.Watch("event","prefetch",e,prefetchhandler)
		}
	}

	fetchhandler:= NewMutationHandler(func(evt MutationEvent) bool{
		if evt.Origin().isFetchedDataValid(propname){
			return false
		}
		if evt.Origin().isPrefetchedDataValid(propname){
			evt.Origin().fetchCompleted(propname,true)
			return false
		}

		r:= cloneReq(req)
		ctx,cancelFn:= context.WithCancel(r.Context())
		r = r.WithContext(ctx)

		oncancelfetch:= NewMutationHandler(func(event MutationEvent)bool{
			cancelFn()
			return false
		}).RunOnce()

		evt.Origin().OnDeleted(NewMutationHandler(func(event MutationEvent)bool{
			cancelFn()
			return false
		}).RunOnce())

		var fetchcancelerremover *MutationHandler
		fetchcancelerremover=NewMutationHandler(func(event MutationEvent)bool{
			p:= string(event.NewValue().(String))
			if p != propname{
				return false
			}
			event.Origin().RemoveMutationHandler("event","cancelfetchrequests",event.Origin(),oncancelfetch)
			event.Origin().RemoveMutationHandler("event","removedatafetcher",event.Origin(),fetchcancelerremover)
			return false
		})

		evt.Origin().WatchEvent("removedatafetcher", evt.Origin(), fetchcancelerremover)

		evt.Origin().Watch("event","cancelfetchrequests",evt.Origin(),oncancelfetch)

		// After a new http.Request has been launched and a response has been returned, cancel and refetch
		// the data corresponding to the req.URL
		evt.Origin().WatchEvent(newRequestEventName("end",r.URL.String()),evt.Origin().ElementStore.Global,NewMutationHandler(func(event MutationEvent)bool{
			//event.Origin().invalidatePrefetch()
			e.InvalidateFetch(propname)
			cancelFn()
			e.Fetch()
			return false
		}).RunOnce())


		evt.Origin().fetchData(propname,r,responsehandler,false)
		return false
	})

	var datafetcherremover *MutationHandler
	datafetcherremover=NewMutationHandler(func(evt MutationEvent)bool{
		p:= string(evt.NewValue().(String))
		if p != propname{
			return true
		}
		evt.Origin().RemoveMutationHandler("event","fetch",evt.Origin(),fetchhandler)
		evt.Origin().RemoveMutationHandler("event","removedatafetcher",evt.Origin(),datafetcherremover)
		return false
	})
	e.WatchEvent("removedatafetcher", e, datafetcherremover)

	e.OnFetch(fetchhandler)             

}

func(e *Element) NewURLDataFetcher(propname string, url string, responsehandler func(*http.Response)(Value,error), noprefetch ...bool){
	req,err:= http.NewRequestWithContext(NavContext,"GET",url,nil)
	if err!= nil{
		panic(url + " is malformed most likely. Unable to create new request")
	}
	e.NewDataFetcher(propname,req,responsehandler,noprefetch...)
}

func(e *Element) RemoveDataFetcher(propname string){
	e.TriggerEvent("removedatafetcher",String(propname))
}

// CancelFetch will abort ongoing fetch requests.
func(e *Element) CancelFetch(){
	e.TriggerEvent("cancelfetchrequests")
	e.Set("fetchstatus","cancelled",Bool(true))
}

func(e *Element) cancelPrefetch(){
	e.TriggerEvent("cancelprefetchrequests")
}


// CancelFetchOnError is an Element modifier that automatically aborts all ongoing fetches as soon 
// as one failed.
// It is not the default so as to leave the possibility to implement retries.
func CancelFetchOnError(e *Element) *Element{
	e.OnFetched(NewMutationHandler(func(evt MutationEvent)bool{
		oldv:= evt.OldValue()
		o,ok:= oldv.(Bool)
		if !ok{
			if oldv == nil{
				o = true
			}
			panic("OldValue for Fetched event of unexpected type. For a first time fetch, should be nil")
		}
		n:= evt.NewValue().(Bool)
		if !n && o{
			e.TriggerEvent("cancelfetchrequests")
		}
		return false
	}))

	return e
}

// WasFetchCancelled answers the question of whether a fecth was cancelled or not.
// It can be used when handling a "fetched" event (OnFetched) to differentiate fetching failure
// from fetching cancellation.
func(e *Element) WasFetchCancelled() bool{
	_,ok:= e.Get("fetchstatus","cancelled")
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

	
	DoAsync(func(){

		res, err:= HttpClient.Do(req)
		if err!= nil{
			DoSync(func() {
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
		if responsehandler == nil{
			return
		}
		v,err:= responsehandler(res)
		if err!= nil{
			DoSync(func() {
				if prefetching{
					e.prefetchCompleted(propname,false)
				}else{
					e.pushFetchError(propname,err)
					e.fetchCompleted(propname,false)
				}	
			})
			return
		}
		DoSync(func() {
			e.SetData(propname,v)
			if prefetching{
				e.prefetchCompleted(propname,true)
			}else{
				e.fetchCompleted(propname,true)
			}
		})
	})
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
	e.TriggerEvent("prefetch")
	
}

/*
type fetchoption Object
func(f fetchoption) Force() bool{
	return bool(Object(f).MustGetBool("force"))
}

func(f fetchoption) FetchAll() bool{
	l:= Object(f).MustGetList("props")
	return len(l) == 0
}

func(f fetchoption) PropList() []string{
	l:= Object(f).MustGetList("props")
	if len(l) == 0{
		return nil
	}
	v:= make([]string, len(l))
	for _,p:= range l{
		v = append(v,string(p.(String)))
	}
	return v
}

// FetchOption returns a Value that can further specify the fetch behavior.
// if forced is true, Fetch will be triggered even if the data is already present. (refetch)
// I f a property name is passed, fetch will only attempt to fetch the data for that string 
// if it has been registered for fetching via one of the WIthFetcheddData... functions.
func FetchOption(forced bool,proplist ...string) fetchoption{
	v:= NewObject()
	v.Set("force",Bool(forced))
	l:=NewList()
	for _,p:= range proplist{
		l = append(l,String(p))
	}
	v.Set("props",l)
	return fetchoption(v)
}

func(e *Element) Fetch(options ...fetchoption){
	e.Properties.Delete("runtime","fetcherrors")
	e.Properties.Delete("fetchstatus","cancelled")

	for _,o:= range options{
		e.TriggerEvent("fetch",Object(o))
	}
	
}
*/


func(e *Element) Fetch(){
	e.Properties.Delete("runtime","fetcherrors")
	e.Properties.Delete("fetchstatus","cancelled")

	e.TriggerEvent("fetch")
	
}

func(e *Element) OnFetch(h *MutationHandler){
	e.WatchEvent("fetch",e,h)
}

func(e *Element) OnFetched(h *MutationHandler){
	e.Watch("event","fetched",e,h)
}

func(e *Element) InvalidateFetch(propname string){
	e.invalidatePrefetch(propname)
	l,ok:=e.Get("runtime","fetchlist")
	if !ok{
		return
	}
	r:= l.(Object)
	fs,ok:= r.Get(propname)
	if !ok{
		return
	}
	s:= fs.(Object)
	s.Set("stale",Bool(true))
	r.Set(propname,s)

	e.Set("runtime", "fetchlist",r)
}

func(e *Element)InvalidateAllFetches(){
	l,ok:=e.Get("runtime","fetchlist")
	if !ok{
		return
	}
	fl:= l.(Object)
	for _,pname:= range fl{
		prop,ok:= pname.(string)
		if ok{
			e.InvalidateFetch(prop)
		}
	}	
}

/*
func fetchEnabled(e *Element) bool{
	v,ok:= e.Get("internals","fetchingenabled")
	if !ok{
		return false
	}
	b:= bool(v.(Bool))
	return b
}
*/

// GetFetchErrors returns, if it exists, a map where each propname key whose fetch failed has a corresponding
// error. Useful to implement retries.
func GetFetchErrors(e *Element) (map[string]error,bool){
	v,ok:= e.Get("runtime","fetcherrors")
	if !ok{
		return nil,ok
	}
	m:= make(map[string]error)
	for k,val:= range v.(Object){
		m[k]= errors.New(string(val.(String)))
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

func(e *Element) OnFetchError(h *MutationHandler){
	e.Watch("runtime","fetcherrors",e,h)
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

func(e *Element) invalidatePrefetch(propname string){
	l,ok:=e.Get("runtime","prefetchlist")
	if !ok{
		panic("Failed to find list of initiated prefetches.")
	}
	r:= l.(Object)
	fs,ok:= r.Get(propname)
	if ok{
		s:= fs.(Object)
		s.Set("status", String("stale"))
		r.Set(propname,s)	
	}
	e.Set("runtime","prefetchlist",r)
}

func(e *Element) fetchCompleted(propname string, successfully bool){
	l,ok:=e.Get("runtime","fetchlist")
	if !ok{
		panic("Failed to find list of initiated fetches.")
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
		delete(s,"stale")
	}
	e.Set("runtime", "fetchlist",r)
}

func(e *Element) isFetchedDataValid(propname string) bool{
	l,ok:=e.Get("runtime","fetchlist")
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

	stale,ok:= s.Get("stale")
	if !ok{
		return true
	}
	return !bool(stale.(Bool))
}

// An Element should also be able to sned requests to a remote server besides retrieving data
// via GET (POST, PUT, PATCH,  UPDATE, DELETE)
// When such a request is made to an endpoint, the Data Fetched should be invalidated and refetched.

// NewRequest makes a http Request using the default client
func(e *Element) NewRequest(req *http.Request, responsehandler func(*http.Response)(Value,error)){
	
	e.TriggerEvent(newRequestEventName("start",req.URL.String()),newRequestStateObject(nil,nil))
	
	
	r:= cloneReq(req)
	ctx,cancelFn:= context.WithCancel(r.Context())
	r = r.WithContext(ctx)

	e.WatchEvent(newRequestEventName("start",req.URL.String()),e,NewMutationHandler(func(evt MutationEvent)bool{
		e.CancelRequest(r)
		return false
	}).RunOnce())

	e.WatchEvent(newRequestEventName("cancel",req.URL.String()),e,NewMutationHandler(func(evt MutationEvent)bool{
		cancelFn()
		return false
	}).RunOnce())

	e.OnDeleted(NewMutationHandler(func(evt MutationEvent)bool{
		e.CancelRequest(r)
		return false
	}).RunOnce())

	e.WatchEvent(newRequestEventName("end",req.URL.String()),e,NewMutationHandler(func(evt MutationEvent)bool{
		evt.Origin().ElementStore.Global.TriggerEvent(newRequestEventName("end",req.URL.String()), evt.NewValue())
		return false
	}))
	
	DoAsync(func() {
		res, err:= HttpClient.Do(req)
		if err!= nil{
			DoSync(func(){
				e.TriggerEvent(newRequestEventName("end",req.URL.String()),newRequestStateObject(nil,err))
			})
			return
		}
		defer res.Body.Close()
		if responsehandler == nil{
			return
		}
		v,err:= responsehandler(res)
		if err!= nil{
			DoSync(func(){
				e.TriggerEvent(newRequestEventName("end",req.URL.String()),newRequestStateObject(nil,err))
			})
			return
		}
		DoSync(func(){
			e.TriggerEvent(newRequestEventName("end",req.URL.String()),newRequestStateObject(v,nil))
		})
	})
}

func(e *Element)CancelRequest(req *http.Request){
	e.TriggerEvent(newRequestEventName("cancel",req.URL.String()))
}

func newRequestStateObject(value Value, err error) Object{
	r:= NewObject()
	r.Set("value",value)
	if err != nil{
		r.Set("error",String(err.Error()))
	}

	return r
}

func newRequestEventName(typ string, URL string) string{
	var prefix string = "request"
	prefix+= typ+"_"
	u,err:= url.Parse(URL)
	if err!= nil{
		panic(err)
	}
	u.RawQuery=""
	eurl:= base32.StdEncoding.EncodeToString([]byte(u.String()))
	return prefix+eurl

}

func(e *Element) OnRequestStart(req *http.Request,h *MutationHandler){
	e.WatchEvent(newRequestEventName("start",req.URL.String()),e,h)
}

func(e *Element) OnRequestEnd(req *http.Request,h *MutationHandler){
	e.WatchEvent(newRequestEventName("end",req.URL.String()),e,h)
}


// GetResponse returns the response ot a request as an interface. If the request failed,
// the error returned by the Error method is non-nil.
// If the response does not exist yet, the interface is nil.
func GetResponse(e *Element, req *http.Request) interface{Value() Value; Error() error}{
	v,ok:= e.Get("event", newRequestEventName("end",req.URL.String()))
	if !ok{
		return nil
	}
	return newResponseObject(v)
}

type responseObject struct{
	Val Value
	Err error
}

func(r responseObject) Value() Value{
	return r.Val
}

func(r responseObject) Error() error{
	return r.Err
}

func newResponseObject(u Value) (responseObject){
	o,ok:= u.(Object)
	if !ok{
		panic("value used as response object should be of type Object")
	}
	rv,ok:= o.Get("value")
	if !ok{
		panic(" expected value field in response object")
	}
	es,ok:= o.Get("error")
	if !ok{
		return responseObject{rv,nil}
	}
	e:= errors.New(string(es.(String)))
	return responseObject{rv,e}
}
