package doc

import ui "github.com/atdiar/particleui"

type AriaChangeAnnoucerElement struct {
	DivElement
}

func (a AriaChangeAnnoucerElement) MakeAnnouncement(message string) AriaChangeAnnoucerElement {
	a.DivElement.SetText(message)
	return a
}

func AriaChangeAnnouncerFor(d *Document) AriaChangeAnnoucerElement {
	return AriaChangeAnnoucerElement{defaultAnnouncer(d)}
}

func defaultAnnouncer(d *Document) DivElement {
	a := d.Div.WithID("announcer")
	SetAttribute(a.AsElement(), "aria-live", "polite")
	SetAttribute(a.AsElement(), "aria-atomic", "true")
	SetInlineCSS(a.AsElement(), "clip:rect(0 0 0 0); clip-path:inset(50%); height:1px; overflow:hidden; position:absolute;white-space:nowrap;width:1px;")

	a.AsElement().Watch("event", "mounted", a, ui.OnMutation(func(evt ui.MutationEvent) bool {
		w := GetDocument(evt.Origin()).Window().AsElement()
		evt.Origin().Watch("ui", "title", w, ui.OnMutation(func(tevt ui.MutationEvent) bool {
			title := string(tevt.NewValue().(ui.String))
			DivElement{evt.Origin()}.SetText(title)
			return false
		}).RunASAP().RunOnce())
		return false
	}))
	return a
}
