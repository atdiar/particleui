// go:build js && wasm
package doc

import (

	"syscall/js"
	"github.com/atdiar/particleui"

)


// Tooltip defines the type implementing the interface of a tooltip ui.Element.
// The default ui.Element interface is reachable via a call to the   AsBasicElement() method.
type Tooltip struct {
	ui.BasicElement
}

// SetContent sets the content of the tooltip.
func (t Tooltip) SetContent(content ui.BasicElement) Tooltip {
	t.AsElement().SetData("content", content.AsElement())
	return t
}

// SetContent sets the content of the tooltip.
func (t Tooltip) SetText(content string) Tooltip {
	t.AsElement().SetData("content", ui.String(content))
	return t
}

var tooltipConstructor = Elements.NewConstructor("tooltip", func(id string) *ui.Element {
	e := ui.NewElement(id, Elements.DocType)
	e.Set("internals", "tag", ui.String("div"))
	e = enableClasses(e)

	htmlTooltip := js.Global().Get("document").Call("getElementById", id)
	exist := !htmlTooltip.IsNull()

	if !exist {
		htmlTooltip = js.Global().Get("document").Call("createElement", "div")
	} else {
		htmlTooltip = reset(htmlTooltip)
	}

	n := NewNativeElementWrapper(htmlTooltip)
	e.Native = n
	SetAttribute(e, "id", id)
	AddClass(e, "tooltip")

	h := ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		content, ok := evt.NewValue().(*ui.Element)
		if ok {
			tooltip := evt.Origin()

			tooltip.AsElement().SetChildren(ui.BasicElement{content})

			return false
		}
		strcontent, ok := evt.NewValue().(ui.String)
		if !ok {
			return true
		}

		tooltip := evt.Origin()
		tooltip.RemoveChildren()

		htmlTooltip.Set("textContent", strcontent)

		return false
	})
	e.Watch("data", "content", e, h)

	return e
}, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

func HasTooltip(target *ui.Element) (Tooltip, bool) {
	v := target.ElementStore.GetByID(target.ID + "-tooltip")
	if v == nil {
		return Tooltip{ui.BasicElement{v}}, false
	}
	return Tooltip{ui.BasicElement{v}}, true
}

// EnableTooltip, when passed to a constructor which has the AllowTooltip option,
// creates a tootltip html div element (for a given target ui.Element)
// The content of the tooltip can be directly set by  specifying a value for
// the ("data","content") (category,propertyname) Element datastore entry.
// The content value can be a string or another ui.Element.
// The content of the tooltip can also be set by modifying the ("tooltip","content")
// property
func EnableTooltip() string {
	return "AllowTooltip"
}

var AllowTooltip = ui.NewConstructorOption("AllowTooltip", func(target *ui.Element) *ui.Element {
	e := LoadFromStorage(tooltipConstructor(target.ID+"-tooltip"))
	// Let's observe the target element which owns the tooltip too so that we can
	// change the tooltip automatically from there.
	h := ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		e.Set("data", "content", evt.NewValue(), false)
		return false
	})
	target.Watch("tooltip", "content", target, h)

	target.AppendChild(ui.BasicElement{e})

	return target
})