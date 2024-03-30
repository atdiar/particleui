package doc

import(
	"github.com/atdiar/particleui"
)

var Children = ui.Children
var E = ui.New
var Listen = ui.Listen
var Class = func(classes ...string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element) *ui.Element{
		for _,class:= range classes{
			AddClass(e,class)
		}
		return e
	}
}

var Ref = ui.Ref
var InitRouter = ui.InitRouter
var Hijack = ui.Hijack

func WithStrConv(val ui.Value) ui.Value{
	return val.(ui.Object).MustGetString("value")
}

// EnableResponsive is a modifier that enables observation of a ui property on the document by an Element
// It also binds the same named property to the Element.
// It allows the Element to be responsive to changes in the document property by watching itself.
//
// Example: EnableResponsive("display") will enable the Element to watch the "display" property on the document.
// Along with a Switch modifier, the Element can conditionally set its children based on the value of the "display" property.
func EnableResponsive(uiprop string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element) *ui.Element{
		d:= GetDocument(e)
		e.Watch("ui", uiprop, d, ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			e.SetUI(uiprop, evt.NewValue())
			return false
		}).RunASAP())

		return e
	}
}
// TODO
/*
func ForEachIn[R ui.List | ui.Object, K any, V ui.Value](rangeable R, f func(K,V)*ui.Element) func(*ui.Element) *ui.Element{
	return func(e *ui.Element) *ui.Element{
		switch v:=any(rangeable).(type){
		case ui.List:
			v.Range()
		}
		return e
	}

}
*/