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