// Package javascript defines the default set of Element constructors, native interfaces,
// events and event handlers, and animation properties used to build js-based UIs.
package javascript

// +build js,wasm

import (
	"syscall/js"

	"github.com/atdiar/particleui"
)

var (
	DOCTYPE     = "html/js"
	ElementList = ui.NewElementStore(DOCTYPE)
)

var NewDiv = ElementList.NewConstructor("div", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, ElementList.DocType)
	native := js.Global()
	return e
})
