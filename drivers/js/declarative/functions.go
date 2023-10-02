package functions

import(
	"github.com/atdiar/particleui"
	"github.com/atdiar/particleui/drivers/js"
)

var Children = ui.Children
var E = ui.New
var Listen = ui.Listen
var CSS = func(classes ...string) func(*ui.Element)*ui.Element{
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


// Styling via CSS

func css(modifier string, property,value string) func(*ui.Element) *ui.Element{
	return func(e *ui.Element) *ui.Element{
		// Retrieve/Create a css ruleset object and store the new rule/declaration.
		var ruleset *ui.TempObject
		c,ok:= e.Get("css","ruleset")
		if !ok{
			ruleset = ui.NewObject()
		}
		ruleset = c.(ui.Object).MakeCopy()
		ruleobj,ok:= ruleset.Get(modifier)
		var rules *ui.TempObject
		if !ok{
			rules = ui.NewObject()
			rules.Set(property,ui.String(value))
			ruleset.Set(modifier,rules.Commit())
		} else{
			rules = ruleobj.(ui.Object).MakeCopy()
			rules.Set(property,ui.String(value))
			ruleset.Set(modifier,rules.Commit())
		}
		e.Set("css","ruleset",ruleset.Commit())
		return e
	}
}

func clearcss() func(*ui.Element) *ui.Element{
	return func(e *ui.Element) *ui.Element{
		e.Set("css","ruleset",ui.NewObject().Commit())
		return e
	}
}

