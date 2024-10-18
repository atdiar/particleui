package doc

import (
	"github.com/atdiar/particleui"
)

var Children = ui.Children
var E = ui.New
var Listen = ui.Listen
var Class = func(classes ...string) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		for _, class := range classes {
			AddClass(e, class)
		}
		return e
	}
}

var Ref = ui.Ref
var InitRouter = ui.InitRouter
var Hijack = ui.Hijack

// WithStrConv is a utility function that is used to retrieve the value stored in an event object
// whne it is suppoosed to be a string.
func WithStrConv(val ui.Value) ui.Value {
	return val.(ui.Object).MustGetString("value")
}

// EnableResponsive is a Constructor function that enables observation of a ui property on the document by an Element
// when applied as a global option.
// It binds the same named property of the "ui" category/namespace to every Element.
// It allows any Element to be responsive to changes in the document property by watching itself.
//
// Example: EnableResponsiveUI("display") will enable the Element to watch the "display" property on the document.
// Along with a Switch modifier, the Element can conditionally set its children based on the value of the "display" property.
func EnableResponsiveUI(prop string) ui.ConstructorOption {
	return ui.NewConstructorOption("responsiveUI", func(e *ui.Element) *ui.Element {
		d := GetDocument(e)
		e.Watch("ui", prop, d, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			e.SetUI(prop, evt.NewValue())
			return false
		}).RunASAP())

		return e
	})
}
