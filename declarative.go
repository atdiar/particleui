package ui

// TODO use AnyElement interface
func New(e AnyElement, modifiers ...func(*Element)*Element) *Element{
	re:= e.AsElement()
	for _,mod:= range modifiers{
		re=mod(re)
	}
	return re
}

// Children is an Element modifier which can be used to set an Elements children.
// It is used in the declarative specification of a UI tree.
func Children(children ...*Element) func(*Element)*Element{
	return func(e *Element) *Element{
		e.SetChildrenElements(children...)
		return e
	}
}

// Listen is an Element modifier that enables an element to listen to a specific event and handle to it.
func Listen(event string, h *EventHandler, nativebinding NativeEventBridge) func(*Element)*Element{
	return func(e *Element) *Element{
		return e.AddEventListener(event,h,nativebinding)
	}
}

// InitRouter is a modifier that applies to an element that should also be a ViewElement.
// It defines a starting point for the navigation.
func InitRouter(options ...func(*Router)*Router) func(*Element)*Element{
	return func(e *Element) *Element{
		e.OnFirstTimeMounted(NewMutationHandler(func(evt MutationEvent)bool{
			v,ok:= evt.Origin().AsViewElement()
			if !ok{
				panic("Router cannot be instantiated with non-ViewElement objects")
			}
			router := NewRouter("/",v ,options...)
			return false
		}))
		return e
	}
}

// Ref will assign an Element as value for a variable whose reference is passed as argument.
// It allows to refer to elements created in the UI tree from outside the UI tree.
// Typically useful for property mutation observing between elements.
func Ref(r **Element) func(*Element) *Element{
	return func(e *Element)*Element{
		*r = e
		return e
	}
}