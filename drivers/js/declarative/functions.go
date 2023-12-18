package functions

import(
	"github.com/atdiar/particleui"
	"github.com/atdiar/particleui/drivers/js"
)

var Children = ui.Children
var AppendChilden = ui.AppendChildren
var E = ui.New
var Listen = ui.Listen
var Class = func(classes ...string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element) *ui.Element{
		for _,class:= range classes{
			doc.AddClass(e,class)
		}
		return e
	}
}
var Ref = ui.Ref
var InitRouter = ui.InitRouter
var Hijack = ui.Hijack

func SetAttribute(name,value string) func(*ui.Element) *ui.Element{
	return func(e *ui.Element) *ui.Element{
		doc.SetAttribute(e,name,value)
		return e
	}
}

func WithStrConv(val ui.Value) ui.Value{
	return val.(ui.Object).MustGetString("value")
}





