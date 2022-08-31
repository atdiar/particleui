package doc

import (

	"syscall/js"
	"strings"
	"github.com/atdiar/particleui"

)


type TextNode struct {
	UIElement *ui.Element
}

func (t TextNode) Element() *ui.Element {
	return t.UIElement
}
func (t TextNode) SetValue(s ui.String) TextNode {
	t.Element().SetDataSetUI("text", s)
	return t
}

func (t TextNode) Value() ui.String {
	v, ok := t.Element().Get("data", "text")
	if !ok {
		return ""
	}
	s, ok := v.(ui.String)
	if !ok {
		return ""
	}
	return s
}


var newTextNode = Elements.NewConstructor("text", func(id string) *ui.Element {
	e := ui.NewElement(id, Elements.DocType)
	htmlTextNode := js.Global().Get("document").Call("createTextNode", "")
	n := NewNativeElementWrapper(htmlTextNode)
	e.Native = n

	e.Watch("ui", "text", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		if s, ok := evt.NewValue().(ui.String); ok { // if data.text is deleted, nothing happens, so no check for nil of  evt.NewValue() TODO handkle all the Value types
			htmlTextNode.Set("nodeValue", string(s))
		}

		return false
	}))

	return e
}, AllowSessionStoragePersistence, AllowAppLocalStoragePersistence)

// NewTextNode creates a text node.
//
func NewTextNode() TextNode {
	return TextNode{newTextNode("textnode", NewID())}
}

type TemplatedTextNode struct {
	*ui.Element
}

func (t TemplatedTextNode) AsElement() *ui.Element {
	return t.Element
}

func (t TemplatedTextNode) SetParam(paramName string, value ui.String) TemplatedTextNode {
	params, ok := t.AsElement().GetData("listparams")
	if !ok {
		return t
	}
	paramslist, ok := params.(ui.List)
	if !ok {
		return t
	}
	for _, pname := range paramslist {
		p, ok := pname.(ui.String)
		if !ok {
			continue
		}
		if paramName == string(p) {
			t.Element.SetData(paramName, value)
		}
	}
	return t
}

func (t TemplatedTextNode) Value() ui.String {
	v, ok := t.Element.Get("data", "text")
	if !ok {
		return ""
	}
	s, ok := v.(ui.String)
	if !ok {
		return ""
	}
	return s
}

// NewTemplatedText returns a templated textnode.
// Using SetParam allows to specify a value for the string parameters.
// Checking that all the parameters have been set before appending the textnode
//  is left at the discretion of the user.
func NewTemplatedText(template string) TemplatedTextNode {
	nt := NewTextNode()
	// nt.Element().Set("internals","template", ui.String(template))

	strmuthandlerFn := func(name string) *ui.MutationHandler {
		m := ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			o, ok := evt.Origin().GetData("listparams")
			if !ok {
				return true
			}
			listparams, ok := o.(ui.Object)
			if !ok {
				return true
			}
			var res string

			for k := range listparams {
				if k != "pui_object_typ" { // review TODO used to be "typ"
					s, ok := evt.Origin().GetData(k)
					if !ok {
						continue
					}
					str, ok := s.(ui.String)
					if !ok {
						continue
					}
					param := "$(" + k + "}"
					res = strings.ReplaceAll(template, param, string(str))
				}
			}
			evt.Origin().SetDataSetUI("text", ui.String(res))
			return false
		})
		return m
	}

	paramnames := parse(template, "${", "}")
	nt.Element().SetData("listparams", paramnames)

	for paramname := range paramnames {
		nt.Element().Watch("data", paramname, nt.Element(), strmuthandlerFn(paramname))
	}

	return TemplatedTextNode{nt.Element()}
}

func parse(input string, tokenstart string, tokenend string) ui.Object {
	result := ui.NewObject()
	ns := input

	startcursor := strings.Index(ns, tokenstart)
	if startcursor == -1 {
		return result
	}
	ns = ns[startcursor:]
	ns = strings.TrimPrefix(ns, tokenstart)

	endcursor := strings.Index(ns, tokenend)
	if endcursor < 1 {
		return result
	}
	tail := ns[endcursor:]
	p := strings.TrimSuffix(ns, tail)
	_, ok := result.Get(p)
	if !ok {
		result.Set(p, nil)
	}

	subresult := parse(strings.TrimPrefix(tail, tokenend), tokenstart, tokenend)
	for k, v := range subresult {
		_, ok := result.Get(k)
		if !ok {
			str, ok := v.(ui.String)
			if !ok {
				continue
			}
			result.Set(k, str)
		}
	}
	return result
}
