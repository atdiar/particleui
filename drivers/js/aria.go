package doc


import (

	"github.com/atdiar/particleui"

)

var AriaChangeAnnouncer =  defaultAnnouncer()

func defaultAnnouncer() DivElement{
	a:=Div.WithID("announcer")
	SetAttribute(a.AsElement(),"aria-live","polite")
	SetAttribute(a.AsElement(),"aria-atomic","true")
	SetInlineCSS(a.AsElement(),"clip:rect(0 0 0 0); clip-path:inset(50%); height:1px; overflow:hidden; position:absolute;white-space:nowrap;width:1px;")

	a.AsElement().Watch("event","mounted",a,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		w:= GetDocument(evt.Origin()).Window().AsElement()
		evt.Origin().Watch("ui","title",w,ui.NewMutationHandler(func(tevt ui.MutationEvent)bool{
			title:= string(tevt.NewValue().(ui.String))
			DivElement{ui.BasicElement{evt.Origin()}}.SetText(title)
			return false
		}).RunASAP().RunOnce())
		return false
	}))
	return a
}


func AriaMakeAnnouncement(message string){
	AriaChangeAnnouncer.SetText(message)
}