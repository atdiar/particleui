// Package ui is a library of functions for simple, generic gui development.
package ui

type MutationCallbacks struct {
	list map[string]*mutationHandlers
}

func NewMutationCallbacks() *MutationCallbacks {
	return &MutationCallbacks{make(map[string]*mutationHandlers, 0)}
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

func (m *MutationCallbacks) DispatchEvent(evt MutationEvent) {
	mhs, ok := m.list[evt.Origin().ID+"/"+evt.ObservedKey()]
	if !ok {
		return
	}
	mhs.Handle(evt)
}

type mutationHandlers struct {
	list []*MutationHandler
}

func newMutationHandlers() *mutationHandlers {
	return &mutationHandlers{make([]*MutationHandler, 0)}
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
}

func NewMutationHandler(f func(evt MutationEvent) bool) *MutationHandler {
	return &MutationHandler{f}
}

func (m *MutationHandler) Handle(evt MutationEvent) bool {
	return m.Fn(evt)
}

// MutationEvent defines a common interface for mutation notifying events.
type MutationEvent interface {
	ObservedKey() string
	Type() string
	Origin() *Element
	NewValue() interface{}
}

// Mutation defines a basic implementation for Mutation Events .
type Mutation struct {
	KeyName string
	typ     string //"ui" or "data"
	Value   interface{}
	Src     *Element
}

func (m Mutation) ObservedKey() string   { return m.KeyName }
func (m Mutation) Origin() *Element      { return m.Src }
func (m Mutation) Type() string          { return m.typ }
func (m Mutation) NewValue() interface{} { return m.Value }

func (e *Element) NewMutationEvent(elementPropertyName string, newvalue interface{}) Mutation {
	return Mutation{e.ID + "/" + elementPropertyName, "", newvalue, e}
}

func (m Mutation) UI() Mutation {
	m.typ = "ui"
	return m
}

func (m Mutation) Data() Mutation {
	m.typ = "data"
	return m
}
