
package main

import (
	"github.com/atdiar/particleui"
	"github.com/atdiar/particleui/drivers/js"
	. "github.com/atdiar/particleui/drivers/js/declarative"
)

func App() doc.Document {

	document:= doc.NewDocument("HelloWorld", doc.EnableScrollRestoration()).EnableWasm()
	var input *ui.Element 
	var paragraph *ui.Element


	E(document.Body(),
		Children(
			E(document.Input.WithID("input", "text").SetAttribute("type","text"),
				Ref(&input),
				doc.SyncValueOnChange(),
			),
			E(document.Label().For(input.AsElement()).SetText("What's your name?")),
			E(document.Paragraph().SetText("Hello!"),
				Ref(&paragraph),
			),
		),
	)

	// The document observes the input for changes and update the paragraph accordingly.
	document.Watch("data","text",input, ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		doc.ParagraphElement{paragraph}.SetText("Hello, "+evt.NewValue().(ui.String).String()+"!")
		return false
	}))
	return document
}

func main(){
	ListenAndServe := doc.NewBuilder(App)
	ListenAndServe(nil)
}


