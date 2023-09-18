package term

import(
	"github.com/atdiar/particleui"
	"github.com/gdamore/tcell/v2"
)

func StyleAsObject(s tcell.Style) ui.Object{
	o:= ui.NewObject()
	fg,bg,attrs:= s.Decompose()
	o.Set("fg",ui.Number(fg))
	o.Set("bg",ui.Number(bg))
	o.Set("attrs",ui.Number(attrs))
	return o.Commit()
}

