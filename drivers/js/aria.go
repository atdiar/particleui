package doc

import (

	"github.com/atdiar/particleui"

)

var AriaChangeAnnouncer =  defaultAnnouncer()

func defaultAnnouncer() Div{
	a:=NewDiv("announcer")
	SetAttribute(a.AsElement(),"aria-live","polite")
	SetAttribute(a.AsElement(),"aria-atomic","true")
	SetInlineCSS(a.AsElement(),"clip:rect(0 0 0 0); clip-path:inset(50%); height:1px; overflow:hidden; position:absolute;white-space:nowrap;width:1px;")

	a.AsElement().OnFirstTimeMounted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		w:= GetWindow().AsElement()
		evt.Origin().Watch("ui","title",w,ui.NewMutationHandler(func(tevt ui.MutationEvent)bool{
			title:= string(tevt.NewValue().(ui.String))
			Div{ui.BasicElement{evt.Origin()}}.SetText(title)
			return false
		}))
		return false
	}))
	return a
}


func AriaMakeAnnouncement(message string){
	AriaChangeAnnouncer.SetText(message)
}