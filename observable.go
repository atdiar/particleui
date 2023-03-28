package ui

import(
	"strings"
)

type Watchable interface {
	watchable()
	AsElement() *Element
}

type Observable struct {
	UIElement *Element
}

func (o Observable) AsElement() *Element {
	return o.UIElement
}

func newObservable(id string) Observable {
	if strings.Contains(id, "/") {
		panic("An id may not use a slash: " + id + " is not valid.")
	}
	e := &Element{
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		"",
		id,
		"observable",
		NewPropertyStore(),
		NewMutationCallbacks(),
		EventListeners{},
		NativeEventUnlisteners{},
		nil,
		"",
		nil,
		nil,
		nil,
		nil,
		nil,
	}

	e.OnDeleted(NewMutationHandler(func(evt MutationEvent)bool{
		unregisterElement(evt.Origin().Root(),e)
		return false
	}).RunOnce())

	

	e.enablefetching()

	return Observable{e}
}

func (o Observable) Get(category, propname string) (Value, bool) {
	return o.AsElement().Get(category, propname)
}

func (o Observable) Set(category string, propname string, value Value) {
	o.AsElement().Set(category, propname, value)
}

func (o Observable) Watch(category string, propname string, owner Watchable, h *MutationHandler) Observable {
  o.AsElement().Watch(category,propname,owner,h)
  return o
}

func (o Observable) Unwatch(category string, propname string, owner Watchable) Observable {
  o.AsElement().Unwatch(category,propname,owner)
  return o
}

func (o Observable) watchable() {}
