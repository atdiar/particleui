package ui

// TODO use AnyElement interface
func New(e AnyElement, modifiers ...func(*Element)*Element) *Element{
	re:= e.AsElement()
	for _,mod:= range modifiers{
		re=mod(re)
	}
	return re
}

// Children is an *Element modifier which can be used to set an Elements children.
// It is used in the declarative specification of a UI tree.
func Children(children ...*Element) func(*Element)*Element{
	return func(e *Element) *Element{
		e.SetChildrenElements(children...)
		return e
	}
}
// Listen is an *Element modifier that enables an element to listen to a specific event and handle to it.
func Listen(event string, h *EventHandler, nativebinding NativeEventBridge) func(*Element)*Element{
	return func(e *Element) *Element{
		return e.AddEventListener(event,h,nativebinding)
	}
}

// InitRouter is an *Element modifier that applies to an element that should also be a ViewElement.
// It defines a starting point for the navigation.
func InitRouter(options ...func(*Router)*Router) func(*Element)*Element{
	return func(e *Element) *Element{
		e.OnFirstTimeMounted(NewMutationHandler(func(evt MutationEvent)bool{
			v,ok:= evt.Origin().AsViewElement()
			if !ok{
				panic("Router cannot be instantiated with non-ViewElement objects")
			}
			NewRouter("/",v ,options...)
			return false
		}))
		return e
	}
}

// Ref is an *Element modifier that assigns the *Element value to the referenced variable (vref).
// It allows to refer to UI tree elements.
// Typically useful for property mutation observing between elements.
func Ref(vref **Element) func(*Element) *Element{
	return func(e *Element)*Element{
		*vref = e
		return e
	}
}