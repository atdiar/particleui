// Package ui is a library of functions for simple, generic gui development.
package ui

type MutationHandlers struct {
	List []func(MutationEvent)
}

type MutationEvent struct {
	KeyName string
	Type    bool //"ui" or "data"
	Value   interface{}
	Watcher *Element
	Mutated *Element
}

func (m MutationEvent) Name() string     { return m.KeyName }
func (m MutationEvent) Target() *Element { return m.Mutated }
func (m MutationEvent) Type() string {
	if m.Type {
		return "ui"
	}
	return "data"
}
func (m MutationEvent) Bubbles() bool      { return false }
func (m MutationEvent) Value() interface{} { return m.Value }

func NewDataMutationEvent(datalabel string, newvalue interface{}, mutated *Element, watcher *Element) MutationEvent {
	return MutationEvent{datalabel, false, newvalue, watcher, mutated}
}

func NewUIMutationEvent(uipropname string, newvalue interface{}, mutated *Element, watcher *Element) MutationEvent {
	return MutationEvent{uipropname, true, newvalue, watcher, mutated}
}
