//go:build server

package js

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

type Type int

const (
	TypeUndefined Type = iota
	TypeNull
	TypeBoolean
	TypeNumber
	TypeString
	TypeSymbol
	TypeObject
	TypeFunction
)

var global = &html.Node{}

type Value struct {
	data interface{}
}

func ValueOf(i interface{}) Value {
	return Value{data: i}
}

func (v Value) Node() *html.Node {
	if v.data == nil {
		return &html.Node{} // DEBUG
	}

	if node, ok := v.data.(*html.Node); ok {
		return node
	}
	fmt.Println("Value is not a *html.Node", v, v.data) // DEBUG
	panic("Value is not a *html.Node")
}

func (v Value) Int() int {
	if i, ok := v.data.(int); ok {
		return i
	}
	panic("Value is not an int")
}

func (v Value) String() string {
	if s, ok := v.data.(string); ok {
		return s
	}
	panic("Value is not a string")
}

func (v Value) Bool() bool {
	if b, ok := v.data.(bool); ok {
		return b
	}
	panic("Value is not a bool")
}

func (v Value) nodes() []*html.Node {
	if nodes, ok := v.data.([]*html.Node); ok {
		return nodes
	}
	return nil
}

func Global() Value {
	return ValueOf(global)
}

func Null() Value {
	return ValueOf(nil)
}

func Undefined() Value {
	return Value{data: nil}
}

func (v Value) Call(method string, args ...interface{}) Value {
	if v.data == nil {
		return Undefined()
	}
	switch method {
	case "appendChild":
		node := v.Node() // This will panic if v is not a Node
		if len(args) == 1 {
			if child, ok := args[0].(Value); ok {
				node.AppendChild(child.Node())
			}
		}
		return Undefined()

	case "insertBefore":
		node := v.Node() // This will panic if v is not a Node
		if len(args) == 2 {
			if newChild, ok := args[0].(Value); ok {
				if referenceChild, ok := args[1].(Value); ok {
					node.InsertBefore(newChild.Node(), referenceChild.Node())
				}
			}
		}
		return Undefined()

	case "removeChild":
		node := v.Node() // This will panic if v is not a Node
		if len(args) == 1 {
			if child, ok := args[0].(Value); ok {
				node.RemoveChild(child.Node())
			}
		}
		return Undefined()

	case "replaceChild":
		node := v.Node() // This will panic if v is not a Node
		if len(args) == 2 {
			if newChild, ok := args[0].(Value); ok {
				if oldChild, ok := args[1].(Value); ok {
					node.InsertBefore(newChild.Node(), oldChild.Node())
					node.RemoveChild(oldChild.Node())
				}
			}
		}
		return Undefined()

	case "createElement":
		if len(args) == 1 {
			if tagName, ok := args[0].(string); ok {
				newNode := &html.Node{
					Type: html.ElementNode,
					Data: tagName,
				}
				return ValueOf(newNode)
			}
		}
		return Undefined()

	case "createTextNode":
		if len(args) == 1 {
			if text, ok := args[0].(string); ok {
				newNode := &html.Node{
					Type: html.TextNode,
					Data: text,
				}
				return ValueOf(newNode)
			}
		}
		return Undefined()

	case "cloneNode":
		node := v.Node() // This will panic if v is not a Node
		clone := &html.Node{
			Type:      node.Type,
			DataAtom:  node.DataAtom,
			Data:      node.Data,
			Namespace: node.Namespace,
			Attr:      append([]html.Attribute(nil), node.Attr...),
		}
		return ValueOf(clone)

	case "getAttribute":
		node := v.Node() // This will panic if v is not a Node
		if len(args) == 1 {
			if attrName, ok := args[0].(string); ok {
				for _, a := range node.Attr {
					if a.Key == attrName {
						return ValueOf(a.Val)
					}
				}
			}
		}
		return Undefined()

	case "setAttribute":
		node := v.Node() // This will panic if v is not a Node
		if len(args) == 2 {
			if attrName, ok := args[0].(string); ok {
				if attrValue, ok := args[1].(string); ok {
					setAttribute(node, attrName, attrValue)
				}
			}
		}
		return Undefined()

	case "removeAttribute":
		node := v.Node() // This will panic if v is not a Node
		if len(args) == 1 {
			if attrName, ok := args[0].(string); ok {
				removeAttribute(node, attrName)
			}
		}
		return Undefined()

	case "innerHTML":
		node := v.Node() // This will panic if v is not a Node
		if len(args) == 1 {
			if innerHTML, ok := args[0].(string); ok {
				parsedHTML, err := html.Parse(strings.NewReader(innerHTML))
				if err != nil {
					panic(err) // Or handle the error as appropriate
				}
				for node.FirstChild != nil {
					node.RemoveChild(node.FirstChild)
				}
				node.AppendChild(parsedHTML)
			}
		}
		return Undefined()

	case "outerHTML":
		node := v.Node() // This will panic if v is not a Node
		if len(args) == 1 {
			if outerHTML, ok := args[0].(string); ok {
				parsedHTML, err := html.ParseFragment(strings.NewReader(outerHTML), node.Parent)
				if err != nil {
					panic(err) // Or handle the error as appropriate
				}
				node.Parent.InsertBefore(parsedHTML[0], node)
				node.Parent.RemoveChild(node)
			}
		}
		return Undefined()

	case "textContent":
		node := v.Node() // This will panic if v is not a Node
		if len(args) == 1 {
			if text, ok := args[0].(string); ok {
				for child := node.FirstChild; child != nil; {
					next := child.NextSibling
					if child.Type == html.TextNode {
						node.RemoveChild(child)
					}
					child = next
				}
				textNode := &html.Node{
					Type: html.TextNode,
					Data: text,
				}
				node.AppendChild(textNode)
			}
		}
		return Undefined()
	case "append":
		node := v.Node() // This will panic if v is not a Node
		for _, arg := range args {
			if child, ok := arg.(Value); ok {
				node.AppendChild(child.Node())
			}
		}
		return Undefined()
	case "prepend":
		node := v.Node() // This will panic if v is not a Node
		for _, arg := range args {
			if child, ok := arg.(Value); ok {
				node.InsertBefore(child.Node(), node.FirstChild)
			}
		}
		return Undefined()
	case "remove":
		nodes := v.nodes()
		if nodes != nil {
			for _, node := range nodes {
				if node.Parent != nil {
					node.Parent.RemoveChild(node)
				}
			}
			return Undefined()
		}
		node := v.Node() // This will panic if v is not a Node
		node.Parent.RemoveChild(node)
		return Undefined()
	case "replaceWith":
		node := v.Node() // This will panic if v is not a Node
		for _, arg := range args {
			if child, ok := arg.(Value); ok {
				node.Parent.InsertBefore(child.Node(), node)
			}
		}
		node.Parent.RemoveChild(node)
		return Undefined()

	default:
		return Undefined()
	}
}

func setAttribute(n *html.Node, name, value string) {
	var found bool
	for i, attr := range n.Attr {
		if attr.Key == name {
			n.Attr[i].Val = value
			found = true
			break
		}
	}
	if !found {
		n.Attr = append(n.Attr, html.Attribute{Key: name, Val: value})
	}
}

func removeAttribute(n *html.Node, name string) {
	for i, attr := range n.Attr {
		if attr.Key == name {
			n.Attr = append(n.Attr[:i], n.Attr[i+1:]...)
			break
		}
	}
}

func (v Value) Get(property string) Value {
	if v.data == nil {
		return Undefined()
	}

	if property == "defaultView" {
		node := &html.Node{
			Type: html.ElementNode,
		}
		return ValueOf(node)
	}

	fmt.Println("Get", property) // DEBUG
	fmt.Println("Value", v)      // DEBUG
	node := v.Node()             // Ensure that v is a Node; this will panic if not

	switch property {
	case "children":
		children := make([]Value, 0)
		for child := v.Node().FirstChild; child != nil; child = child.NextSibling {
			children = append(children, ValueOf(child))
		}
		return ValueOf(children)

	case "length":
		children, ok := v.data.([]*html.Node)
		if !ok {
			return Undefined()
		}
		return ValueOf(len(children))

	case "attributes":
		attrs := make(map[string]string)
		for _, attr := range node.Attr {
			attrs[attr.Key] = attr.Val
		}
		return ValueOf(attrs)

	case "tagName":
		return ValueOf(node.Data)

	case "childElementCount":
		count := 0
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode {
				count++
			}
		}
		return ValueOf(count)

	case "firstElementChild":
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode {
				return ValueOf(c)
			}
		}
		return Undefined()

	case "lastElementChild":
		var lastChild *html.Node
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode {
				lastChild = c
			}
		}
		return ValueOf(lastChild)

	case "nextElementSibling":
		for c := node.NextSibling; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode {
				return ValueOf(c)
			}
		}
		return Undefined()

	case "previousElementSibling":
		for c := node.PrevSibling; c != nil; c = c.PrevSibling {
			if c.Type == html.ElementNode {
				return ValueOf(c)
			}
		}
		return Undefined()

	case "nodeName":
		return ValueOf(node.Data)

	case "nodeType":
		return ValueOf(node.Type)
	case "document":
		node := &html.Node{
			Type: html.DocumentNode,
		}
		return ValueOf(node)
	case "documentElement":
		node := &html.Node{
			Type: html.ElementNode,
		}
		return ValueOf(node)
	default:
		return Undefined()
	}
}

func (v Value) Set(property string, value interface{}) {
	if v.data == nil {
		return
	}
	node := v.Node() // This will panic if v is not a Node

	switch property {
	case "id", "class", "name", "type": // Example attributes
		if val, ok := value.(string); ok {
			setAttribute(node, property, val)
		}

	// Handle other properties as needed

	default:
		// Maybe log an error or handle cases where the property is not valid
	}
}

// Add remaining methods as in your original implementation...
// Delete, Equal, Float, Index, InstanceOf, Int, Invoke, IsNaN, IsNull,
// IsUndefined, Length, New, Set, SetIndex, String, Truthy, Type, Error,
// Func, FuncOf, Release, CopyBytesToGo, CopyBytesToJS, ValueError,
// and Error methods for ValueError...

// Delete is a no-op in server context.
func (v Value) Delete(p string) {}

// Equal checks if two Values are equal. Simplified implementation for server.
func (v Value) Equal(w Value) bool {
	return v.data == w.data
}

// Float is a no-op in server context.
func (v Value) Float() float64 {
	return 0.0
}

// Index is a no-op in server context.
func (v Value) Index(i int) Value {
	return Undefined()
}

// InstanceOf is a no-op in server context.
func (v Value) InstanceOf(t Value) bool {
	return false
}

// Invoke is a no-op in server context.
func (v Value) Invoke(args ...interface{}) Value {
	return Undefined()
}

// IsNaN always returns false in server context.
func (v Value) IsNaN() bool {
	return false
}

// IsNull checks if the Value is null.
func (v Value) IsNull() bool {
	return v.data == nil
}

// IsUndefined checks if the Value is undefined.
func (v Value) IsUndefined() bool {
	return v.data == nil
}

// Length is a no-op in server context.
func (v Value) Length() int {
	return 0
}

// New is a no-op in server context.
func (v Value) New(args ...interface{}) Value {
	return Undefined()
}

// SetIndex is a no-op in server context.
func (v Value) SetIndex(i int, x interface{}) {}

// Truthy is a no-op in server context.
func (v Value) Truthy() bool {
	if v.data == nil {
		return false
	}
	return true
}

// Type returns the type of the Value.
func (v Value) Type() Type {
	if v.data == nil {
		return TypeNull
	}
	// TODO needs to type switch for determination logic
	switch v.data.(type) {
	case bool:
		return TypeBoolean
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return TypeNumber
	case string:
		return TypeString
	}
	// Add more detailed type determination logic if needed.
	return TypeObject
}

// Error wraps a JavaScript error.
type Error struct {
	Value
}

// Error returns a string representing the Error.
func (e Error) Error() string {
	return "Rendering error"
}

// Func is a wrapped Go function to be called by JavaScript.
type Func struct {
	Value // the JavaScript function that invokes the Go function
}

// FuncOf returns a function to be used by JavaScript.
func FuncOf(fn func(this Value, args []Value) interface{}) Func {
	return Func{}
}

// Release frees up resources allocated for the function. No-op in server.
func (c Func) Release() {}

// CopyBytesToGo is a no-op in server context.
func CopyBytesToGo(dst []byte, src Value) int {
	return 0
}

// CopyBytesToJS is a no-op in server context.
func CopyBytesToJS(dst Value, src []byte) int {
	return 0
}

// ValueError represents an error concerning a Value object.
type ValueError struct {
	Value
}

// Error returns a string representing the ValueError.
func (e *ValueError) Error() string {
	return "Value error"
}
