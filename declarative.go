package ui

// TODO use AnyElement interface
func New(e AnyElement, modifiers ...func(*Element)*Element) *Element{
	re:= e.AsElement()
	for _,mod:= range modifiers{
		re=mod(re)
	}
	return re
}

func Children(children ...*Element) func(*Element)*Element{
	return func(e *Element) *Element{
		e.SetChildrenElements(children...)
		return e
	}
}

func Listen(event string, h *EventHandler, nativebinding NativeEventBridge) func(*Element)*Element{
	return func(e *Element) *Element{
		return e.AddEventListener(event,h,nativebinding)
	}
}