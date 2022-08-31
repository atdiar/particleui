package doc

import (
	"io"
	"github.com/atdiar/particleui"

	"golang.org/x/net/html"
)

/*
 HTML rendering

*/

func (d Document) Render(w io.Writer) error {
	return html.Render(w, NewHTMLTree(d))
}

func NewHTMLNode(e *ui.Element) *html.Node {
	if e.DocType != Elements.DocType {
		panic("Bad Element doctype")
	}
	v, ok := e.Get("internals", "constructor")
	if !ok {
		return nil
	}
	tag, ok := v.(ui.String)
	if !ok {
		panic("constructor name should be a string")
	}
	data := string(tag)
	nodetype := html.RawNode
	if string(tag) == "root" {
		data = "body"
	}
	n := &html.Node{}
	n.Type = nodetype
	n.Data = data

	attrs, ok := e.GetData("attrs")
	if !ok {
		return n
	}
	tattrs, ok := attrs.(ui.Object)
	if !ok {
		panic("attributes is supossed to be a ui.Object type")
	}
	for k, v := range tattrs {
		val, ok := v.(ui.String)
		if !ok {
			continue // should panic probably instead
		}
		a := html.Attribute{"", k, string(val)}
		n.Attr = append(n.Attr, a)
	}
	return n
}

func NewHTMLTree(document Document) *html.Node {
	doc := document.AsBasicElement()
	return newHTMLTree(doc.AsElement())
}

func newHTMLTree(e *ui.Element) *html.Node {
	d := NewHTMLNode(e)
	if e.Children != nil && e.Children.List != nil {
		for _, child := range e.Children.List {
			c := newHTMLTree(child)
			d.AppendChild(c)
		}
	}
	return d
}