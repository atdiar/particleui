package ui

func New(e AnyElement, modifiers ...func(*Element) *Element) *Element {
	re := e.AsElement()
	for _, mod := range modifiers {
		re = mod(re)
	}
	return re
}

// Children is an *Element modifier which can be used to set an Elements children.
// It is used in the declarative specification of a UI tree.
func Children(children ...*Element) func(*Element) *Element {
	return func(e *Element) *Element {
		e.SetChildren(children...)
		return e
	}
}

// Append is an *Element modifier which can be used to append children to an Element.
// It is used in the declarative specification of a UI tree.
func AppendChildren(children ...*Element) func(*Element) *Element {
	return func(e *Element) *Element {
		for _, child := range children {
			e.AppendChild(child)
		}
		return e
	}
}

// Listen is an *Element modifier that enables an element to listen to a specific event and handle to it.
func Listen(event string, h *EventHandler) func(*Element) *Element {
	return func(e *Element) *Element {
		return e.AddEventListener(event, h)
	}
}

// InitRouter is an *Element modifier that applies to an element that should also be a ViewElement.
// It defines a starting point for the navigation.
func InitRouter(options ...func(*Router) *Router) func(*Element) *Element {
	return func(e *Element) *Element {
		e.OnMounted(NewMutationHandler(func(evt MutationEvent) bool {
			v, ok := evt.Origin().AsViewElement()
			if !ok {
				panic("Router cannot be instantiated with non-ViewElement objects")
			}
			NewRouter(v, options...)
			return false
		}).RunASAP().RunOnce())
		return e
	}
}

// Ref is an *Element modifier that assigns the *Element value to the referenced variable (vref).
// It allows to refer to UI tree elements.
// Typically useful for property mutation observing between elements.
//
// TODO once the generic implementation will allow constraints that specify fields, this will be
// updated to allow for more specific types.
// WIll also require alias to type parametered objects.
func Ref(vref **Element) func(*Element) *Element {
	return func(e *Element) *Element {
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
func Switch(category, propname string) elementSwitch {
	return elementSwitch{category: category, propname: propname, cases: []Value{}, elements: []*Element{}, boundDataProps: []string{}}
}

type elementSwitch struct {
	category       string
	propname       string
	cases          []Value
	elements       []*Element
	boundDataProps []string
}

func (e elementSwitch) Case(val Value, elem *Element) elementSwitch {
	e.cases = append(e.cases, val)
	e.elements = append(e.elements, elem)
	return e
}

func (e elementSwitch) WithSharedData(propname string) elementSwitch {
	e.boundDataProps = append(e.boundDataProps, propname)
	return e
}

func (e elementSwitch) Default(elem *Element) func(*Element) *Element {
	for i := 0; i < len(e.cases); i++ {
		if Equal(e.cases[i], (String("zui-default"))) {
			panic("Default case already defined")
		}
	}
	e.cases = append(e.cases, String("zui-default"))
	e.elements = append(e.elements, elem)
	return func(el *Element) *Element {
		for i := 0; i < len(e.cases); i++ {
			el.Watch(e.category, e.propname, el, NewMutationHandler(func(evt MutationEvent) bool {
				if Equal(evt.NewValue(), e.cases[i]) {
					el.SetChildren(e.elements[i])
				}
				return false
			}))

			e.elements[i].ShareLifetimeOf(el)
		}

		for _, propname := range e.boundDataProps {
			el.Watch(dataNS, propname, el, NewMutationHandler(func(evt MutationEvent) bool {
				for _, elm := range e.elements {
					elm.SetDataSetUI(propname, evt.NewValue())
				}
				return false
			}))

			for _, elm := range e.elements {
				el.Watch(dataNS, propname, elm, NewMutationHandler(func(evt MutationEvent) bool {
					for _, element := range e.elements {
						if element.ID == elm.ID {
							continue
						}
						element.SetDataSetUI(propname, evt.NewValue())
					}
					el.SyncUISetData(propname, evt.NewValue())

					return false
				}))
			}
		}

		return el
	}
}

/* Example of a reactive datepicker Element implemented using the Switch modifier:

var smallDatepicker *Element /// Reference to the small datepicker element, here just to provide a more realistic example

E(document.Div.WithID("uuid-responsiveDatepicker"),
	Switch("ui","display").
		Case(String("small"), DatePickerSmall("uuid-dp-s", Ref(&smallDatepicker))).
		Case(String("large"), DatePickerLarge("uuid-dp-xl"))).
		Case(String("medium"), DatePickerMedium("uuid-dp-m")).
		WithSharedData("date").
	Default(nil),
)


Important to note that the the value of "ui,display" is observed on the parent div of id "uuid-datepicker".
This is potentially the same value that is being set by the EnableResponsiveUI constructor option, i.e. the value
for the document "ui" property "display"
That should allow for the tree to be reactive to display changes.


*/

// ForEachIn allows to parse through a list of values when it changes and create a new Element for each value.
// If a custom link is provided, it is called once the child Element has been created to allow for custom
// It is useful in simple cases.
// Some other cases such as when the elemnts need some complex initialization that should only happen
// once at element creation need to be handled with care.
// (we don't want to register some watchers each time the list change partially.
// Already existing elements would not need it) for instance.
func ForEachIn(uiprop string, f func(int, Value) *Element) func(*Element) *Element {
	return func(e *Element) *Element {
		e.Watch(uiNS, uiprop, e, NewMutationHandler(func(evt MutationEvent) bool {
			v := evt.NewValue()
			switch v := v.(type) {
			case List:
				w := v.Unwrap()
				length := len(w)
				var newchildren = make([]*Element, length)
				g := func(k int, val Value) bool {
					ne := f(k, val)
					newchildren[k] = ne
					return false
				}
				v.Range(g)
				e.SetChildren(newchildren...)
			default:
				el := f(0, v)
				e.SetChildren(el)
			}
			return false
		}).RunASAP())

		return e
	}
}
