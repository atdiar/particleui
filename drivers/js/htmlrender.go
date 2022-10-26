package doc

import (
	"io"
	"strings"

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
	
	n := &html.Node{}
	n.Type = nodetype
	n.Data = data

	attrs, ok := e.GetData("attrs")
	if !ok {
		return n
	}
	tattrs, ok := attrs.(ui.Object)
	if !ok {
		panic("attributes is supposed to be a ui.Object type")
	}
	for k, v := range tattrs {
		a := html.Attribute{"", k, string(v.(ui.String))}
		n.Attr = append(n.Attr, a)
	}

	
	// Element state should be stored serialized in script Element and hydration attribute should be set
	// on the Node
	n.Attr = append(n.Attr,html.Attribute{"",HydrationAttrName,"true"})



	return n
}

func NewHTMLTree(document Document) *html.Node {
	doc := document.AsBasicElement()
	indexmap:= make(map[string]*html.Node)
	return newHTMLTree(doc.AsElement(), indexmap)
}

func newHTMLTree(e *ui.Element, index map[string]*html.Node) *html.Node {
	d := NewHTMLNode(e)
	statescriptnode := generateStateInScriptNode((e))
	index[e.ID+ SSRStateSuffix] = statescriptnode

	if e.ID == GetDocument().Body().ID{
		index["body"] = d
	}
	if e.Children != nil && e.Children.List != nil {
		for _, child := range e.Children.List {
			c := newHTMLTree(child,index)
			d.AppendChild(c)
		}
	}

	bodyNode:= index["body"]
	delete(index, "body")
	for _,v:= range index{
		bodyNode.AppendChild(v)
	}

	return d
}

func generateStateInScriptNode(e *ui.Element) *html.Node{
	state:=  SerializeProps(e)
	script:= `<script id='` + e.ID+SSRStateSuffix+`'>
	` + state + `
	<script>`
	scriptNode, err:= html.Parse(strings.NewReader(script))
	if err!= nil{
		panic(err)
	}
	return scriptNode
}