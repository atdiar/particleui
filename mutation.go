// Package ui is a library of functions for simple, generic gui development.
package ui

import(
	"time"
	"strings"
)


var newEventID = newIDgenerator(5,time.Now().UnixNano())

type MutationCallbacks struct {
	list map[string]*mutationHandlers
}

func NewMutationCallbacks() *MutationCallbacks {
	return &MutationCallbacks{make(map[string]*mutationHandlers, 10)}
}

func (m *MutationCallbacks) Add(key string, h *MutationHandler) *MutationCallbacks {
	mhs, ok := m.list[key]
	if !ok {
		mhs = newMutationHandlers().Add(h)
		m.list[key] = mhs
		return m
	}
	mhs.Add(h)
	return m
}

func (m *MutationCallbacks) Remove(key string, h *MutationHandler) *MutationCallbacks {
	mhs, ok := m.list[key]
	if !ok {
		return m
	}
	mhs.Remove(h)
	return m
}

func (m *MutationCallbacks) RemoveAll(key string) *MutationCallbacks {
	mhs, ok := m.list[key]
	if !ok {
		return m
	}
	for i:=0;i<len(mhs.list);i++{
		mhs.list[i] = nil
	}
	mhs.list= mhs.list[:0]
	return m
}

func (m *MutationCallbacks) DispatchEvent(evt MutationEvent) {
	mhs, ok := m.list[evt.ObservedKey()]
	if !ok {
		return
	}
	mhs.Handle(evt)
}

type mutationHandlers struct {
	list []*MutationHandler
}

func newMutationHandlers() *mutationHandlers {
	return &mutationHandlers{make([]*MutationHandler, 0,10)}
}

func (m *mutationHandlers) Add(h *MutationHandler) *mutationHandlers {
	m.list = append(m.list, h)
	return m
}

func (m *mutationHandlers) Remove(h *MutationHandler) *mutationHandlers {

	for i:=0;i<len(m.list);i++{
		v:= m.list[i]
		if v == nil{
			continue
		}
		if v == h {
			m.list[i]= nil
		}
	}
	return m
}

func (m *mutationHandlers) Handle(evt MutationEvent) {
	var needcleanup bool
	var index int
	list:= m.list[:0]
	var handle = true
	
	for i:=0;i<len(m.list);i++{
		h:= m.list[i]
		if h == nil{
			if !needcleanup{
				list =m.list[:i]
				index = i+1
				needcleanup = true
			}
			continue
		}
		if handle{
			b := h.Handle(evt)
			if b {
				handle = false
				if !needcleanup{
					return
				}
			}
		}
		if needcleanup{
			list = append(list,h)
			index++
		}
		
	}

	if needcleanup{
		l:= len(m.list)
		for i:= index;i<l;i++{
			m.list[i]= nil
		}
		m.list = list[:index]
	}
	
}

// MutationHandler is a wrapper type around a callback function run after a mutation
// event occured.
type MutationHandler struct {
	Fn func(MutationEvent) bool
	Once bool
	ASAP bool
	binding bool
	fetching bool
}

func NewMutationHandler(f func(evt MutationEvent) bool) *MutationHandler {
	return &MutationHandler{f,false,false,false, false}
}

// RunOnce indicates that the handler will run only for the next occurence of a mutation event. 
// It will unregister right after.
// The returned mutation handler is a copy that holds a reference to the same handling function.
func(m *MutationHandler) RunOnce() *MutationHandler{
	if m.Once{
		return m
	}
	n:= NewMutationHandler(m.Fn)
	n.ASAP = m.ASAP
	n.binding = m.binding
	n.fetching = m.fetching
	n.Once = true
	return n
}

// RunASAP will run the event handler immediately if a mutation has already occured even if before
// the mutation ws registered. It is useful when a handler must be run as long as an event occured.
// E.g. if something must bew run the first time an Element is mounted (firsttimemounted event side-effect)
// The returned mutation handler is a copy that holds a reference to the same handling function.
func(m *MutationHandler) RunASAP() *MutationHandler{
	if m.ASAP{
		return m
	}
	n:= NewMutationHandler(m.Fn)
	n.Once = m.Once
	n.binding = m.binding
	n.fetching = m.fetching
	n.ASAP = true
	return n
}

func(m *MutationHandler) binder() *MutationHandler{
	if m.binding{
		return m
	}
	n:= NewMutationHandler(m.Fn)
	n.Once = m.Once
	n.ASAP = m.ASAP
	n.fetching = m.fetching
	n.binding = true
	return n
}

func(m *MutationHandler) fetcher() *MutationHandler{
	if m.binding{
		return m
	}
	n:= NewMutationHandler(m.Fn)
	n.Once = m.Once
	n.ASAP = m.ASAP
	n.binding = m.binding
	n.fetching = true
	return n
}

func (m *MutationHandler) Handle(evt MutationEvent) bool {
	return m.Fn(evt)
}

// MutationEvent defines a common interface for mutation notifying events.
type MutationEvent interface {
	ObservedKey() string
	Category() string
	Property() string
	Origin() *Element
	NewValue() Value
	OldValue() Value
}

// Mutation defines a basic implementation for Mutation Events.
type Mutation struct {
	KeyName string
	category string
	propname string
	NewVal  Value
	OldVal  Value
	Src     *Element
}

func (m Mutation) ObservedKey() string { return m.KeyName }
func (m Mutation) Origin() *Element    { return m.Src }
func (m Mutation) Category() string        { return m.category}
func (m Mutation) Property() string        { return m.propname }
func (m Mutation) NewValue() Value     { 
	if m.category == "event"{
		e,ok:= m.NewVal.(Object).Get("value")
		if !ok{
			panic("event value not found")
		}
		return Copy(e)
	}

	return Copy(m.NewVal)
}

func (m Mutation) OldValue() Value     { 
	if m.category == "event"{
		if m.OldVal == nil{
			return nil
		}
		e,ok:= m.OldVal.(Object).Get("value")
		if !ok{
			panic("event value not found")
		}
		return Copy(e)
	}

	return Copy(m.OldVal)
}

func (e *Element) NewMutationEvent(category string, propname string, newvalue Value, oldvalue Value) Mutation {
	return Mutation{strings.Join([]string{e.ID,category,propname},"/"), category,propname, Copy(newvalue), Copy(oldvalue), e}
}


var NoopMutationHandler = NewMutationHandler(func(evt MutationEvent)bool{
	return false
})