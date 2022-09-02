// Package ui is a library of functions for simple, generic gui development.
package ui

import (
	"strings"
)

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
	mhs.list = make([]*MutationHandler, 0,10)
	return m
}

func (m *MutationCallbacks) DispatchEvent(evt MutationEvent) {
	key := evt.ObservedKey()
	shards := strings.Split(strings.TrimPrefix(key, "/"), "/")
	if len(shards) == 2 {
		observableID := shards[0]
		category := shards[1]
		grouphandlerAdress := observableID + "/" + category + "/" + "existifallpropertieswatched"
		gmhs, ok := m.list[grouphandlerAdress]
		if ok {
			gmhs.Handle(evt)
		}
	}

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
	index := -1
	for k, v := range m.list {
		if v != h {
			continue
		}
		index = k
		break
	}
	if index >= 0 {
		m.list = append(m.list[:index], m.list[index+1:]...)
	}
	return m
}

func (m *mutationHandlers) Handle(evt MutationEvent) {
	for _, h := range m.list {
		b := h.Handle(evt)
		if b {
			return
		}
	}
}

// MutationHandler is a wrapper type around a callback function run after a mutation
// event occured.
type MutationHandler struct {
	Fn func(MutationEvent) bool
	Once bool
	ASAP bool
}

func NewMutationHandler(f func(evt MutationEvent) bool) *MutationHandler {
	return &MutationHandler{f,false,false}
}

// RunOnce indicates that the handler will run only for the next occurence of a mutation event. 
// It will unregister right after.
// The returned mutation handler is a copy that holds a reference to the same handling function.
func(m *MutationHandler) RunOnce() *MutationHandler{
	n:= NewMutationHandler(m.Fn)
	n.Once = true
	return n
}

// RunASAP will run the event handler immediately if a mutation has already occured even if before
// the mutation ws registered. It is useful when a handler must be run as long as an event occured.
// E.g. if something must bew run the first time an Element is mounted (firsttimemounted event side-effect)
// The returned mutation handler is a copy that holds a reference to the same handling function.
func(m *MutationHandler) RunASAP() *MutationHandler{
	n:= NewMutationHandler(m.Fn)
	n.ASAP = true
	return n
}

func (m *MutationHandler) Handle(evt MutationEvent) bool {
	return m.Fn(evt)
}

// MutationEvent defines a common interface for mutation notifying events.
type MutationEvent interface {
	ObservedKey() string
	Type() string
	Origin() *Element
	NewValue() Value
	OldValue() Value
}

// Mutation defines a basic implementation for Mutation Events.
type Mutation struct {
	KeyName string
	typ     string
	NewVal  Value
	OldVal  Value
	Src     *Element
}

func (m Mutation) ObservedKey() string { return m.KeyName }
func (m Mutation) Origin() *Element    { return m.Src }
func (m Mutation) Type() string        { return m.typ }
func (m Mutation) NewValue() Value     { return m.NewVal }
func (m Mutation) OldValue() Value     { return m.OldVal }

func (e *Element) NewMutationEvent(category string, propname string, newvalue Value, oldvalue Value) Mutation {
	return Mutation{e.ID + "/" + category + "/" + propname, category, newvalue, oldvalue, e}
}
