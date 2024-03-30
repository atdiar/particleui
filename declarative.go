package ui


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
		e.SetChildren(children...)
		return e
	}
}

// Append is an *Element modifier which can be used to append children to an Element.
// It is used in the declarative specification of a UI tree.
func AppendChildren(children ...*Element) func(*Element)*Element{
	return func(e *Element) *Element{
		for _,child:= range children{
			e.AppendChild(child)
		}
		return e
	}
}

// Listen is an *Element modifier that enables an element to listen to a specific event and handle to it.
func Listen(event string, h *EventHandler) func(*Element)*Element{
	return func(e *Element) *Element{
		return e.AddEventListener(event,h)
	}
}

// InitRouter is an *Element modifier that applies to an element that should also be a ViewElement.
// It defines a starting point for the navigation.
func InitRouter(options ...func(*Router)*Router) func(*Element)*Element{
	return func(e *Element) *Element{
		e.OnMounted(NewMutationHandler(func(evt MutationEvent)bool{
			v,ok:= evt.Origin().AsViewElement()
			if !ok{
				panic("Router cannot be instantiated with non-ViewElement objects")
			}
			NewRouter(v ,options...)
			return false
		}).RunASAP().RunOnce())
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
// Switch is an *Element modifier that applies to an element and allows to conditionally set its children
// based on the value of a property:
// 
//	Switch("ui","display").
//		Case(String("small"), A).
//		Case(String("large"), B).
//		Case(String("medium"), C).
//	Default(nil),
//
// In the above example, the children of the element will be set to A if the value of the property "display" is "small",
// to B if it is "large", to C if it is "medium" and to nothing if it is anything else.
// The Default method is used to set the children of the element when the property value does not match any of the cases.
func Switch(category, propname string) elementSwitch{
	return elementSwitch{category:category, propname:propname, cases: []Value{}, elements: []*Element{}}
}

type elementSwitch struct{
	category string
	propname string
	cases []Value
	elements []*Element
}

func(e elementSwitch) Case(val Value, elem *Element) elementSwitch{
	e.cases = append(e.cases,val)
	e.elements = append(e.elements,elem)
	return e
}

func (e elementSwitch) Default(elem *Element) func(*Element)*Element{
	for i:=0;i<len(e.cases);i++{
		if Equal(e.cases[i],(String("zui-default"))){
			panic("Default case already defined")
		}
	}
	e.cases= append(e.cases,String("zui-default"))
	e.elements= append(e.elements,elem)
	return func(el *Element) *Element{
		for i:=0;i<len(e.cases);i++{
			el.Watch(e.category,e.propname,el, NewMutationHandler(func(evt MutationEvent)bool{
				if Equal(evt.NewValue(),e.cases[i]){
					el.SetChildren(e.elements[i])
				}
				return false
			}))
		}
		return el
	}
}

