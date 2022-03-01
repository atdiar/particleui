package ui

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

func NewObservable(id string) Observable {
	e := &Element{
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		id,
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
	}
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

func (o Observable) WatchGroup(category string, target Watchable, h *MutationHandler) Observable {
  o.AsElement().WatchGroup(category,target,h)
  return o
}

func (o Observable) UnwatchGroup(category string, owner *Element) Observable {
  o.AsElement().UnwatchGroup(category,owner)
  return o
}

func (o Observable) watchable() {}