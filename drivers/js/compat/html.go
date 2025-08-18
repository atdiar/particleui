//go:build server

package js

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type Type int

const (
	TypeNull Type = iota
	TypeBoolean
	TypeNumber
	TypeString
	TypeSymbol
	TypeObject
	TypeFunction
	TypeUndefined
)

func (t Type) String() string {
	switch t {
	case TypeUndefined:
		return "undefined"
	case TypeNull:
		return "null"
	case TypeBoolean:
		return "boolean"
	case TypeNumber:
		return "number"
	case TypeString:
		return "string"
	case TypeSymbol:
		return "symbol"
	case TypeObject:
		return "object"
	case TypeFunction:
		return "function"
	default:
		panic("bad type")
	}
}

type stubwindow struct {
	document *stubdocument
}

type stubdocument struct {
	implementation *documentImplementation
}

// documentImplementation is a dummy struct to represent the DOMImplementation interface.
// It provides methods like createHTMLDocument.
type documentImplementation struct {
	// No fields needed, as it's a stateless factory
}

// defaultDocumentImplementation is the singleton instance of DOMImplementation.
var defaultDocumentImplementation *documentImplementation

// stubWindow is a stub implementation of the global window object.
var stubWindow *stubwindow = &stubwindow{
	document: &stubdocument{
		implementation: &documentImplementation{},
	},
}

func Global() Value {
	return ValueOf(stubWindow) // Return the global window object as a Value
}

type Value struct {
	data interface{}
}

func ValueOf(i interface{}) Value {
	if v, ok := i.(Value); ok {
		return v // Already a Value, return it directly
	}
	return Value{data: i}
}

// Node returns the underlying *html.Node or panics if the Value is not a node.
func (v Value) Node() *html.Node {
	if v.data == nil || v.IsUndefined() {
		// If nil or undefined, panic because a node is expected.
		panic("Value does not represent a valid *html.Node (it is nil or undefined)")
	}

	if node, ok := v.data.(*html.Node); ok {
		return node
	}
	panic(fmt.Sprintf("Value is not a *html.Node (type: %T)", v.data))
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
	return fmt.Sprintf("%v", v.data)
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

func Null() Value {
	return ValueOf(nil)
}

func Undefined() Value {
	return Value{data: TypeUndefined}
}

func (v Value) Call(method string, args ...interface{}) Value {
	if v.data == nil || v.IsUndefined() {
		return Undefined()
	}

	if _, ok := v.data.(*stubwindow); ok {
		// If the Value is a stubwindow, we handle its methods.
		switch method {
		case "document":
			return ValueOf(v.data.(*stubwindow).document) // Return the document object
		default:
			//panic(fmt.Sprintf("method %s not implemented for stubwindow", method))
			// could return Undefined but for now, trying to find bugs
			return Undefined() // Return Undefined for unimplemented methods
		}
	}

	if doc, ok := v.data.(*stubdocument); ok {
		// If the Value is a stubdocument, we handle its methods.
		switch method {
		case "implementation":
			return ValueOf(doc.implementation) // Return the document implementation
		default:
			// Other methods on document are not emulated
			// panic(fmt.Sprintf("method %s not implemented for stubdocument", method))
			// could return Undefined but for now, trying to find bugs
			return Undefined() // Return Undefined for unimplemented methods
		}
	}

	// Specific behavior for documentImplementation methods
	if _, ok := v.data.(*documentImplementation); ok {
		switch method {
		case "createHTMLDocument":
			// Browser spec: createHTMLDocument can take 0, 1, or 3 arguments (title, doctype, body)
			// For lightweight emulation, we'll ignore args for now and always create a standard empty doc.
			doc, err := html.Parse(strings.NewReader(`<!DOCTYPE html><html><head></head><body></body></html>`))
			if err != nil {
				panic(fmt.Errorf("failed to create new HTML document via createHTMLDocument: %w", err))
			}
			// Find the <html> node within the parsed document
			var htmlNode *html.Node
			for c := doc.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode && c.Data == "html" {
					htmlNode = c
					break
				}
			}

			if htmlNode == nil {
				// This case should ideally not happen for `<html></html>` string,
				// but it's good to be robust.
				panic("failed to find <html> node in newly created document")
			}

			// Return the <html> node, not the document node
			return ValueOf(htmlNode)
		default:
			// Other methods on document.implementation are not emulated
			//panic(fmt.Sprintf("method %s not implemented for documentImplementation", method))
			// return Undefined()
			// could return Undefined but for now, trying to find bugs
			return Undefined() // Return Undefined for unimplemented methods
		}
	}

	// If the Value is not a *html.Node, it will panic. (it should not be a stub for instance).
	v.Node()

	// Conversion to Value if needed
	jsArgs := make([]Value, len(args))
	for i, arg := range args {
		jsArgs[i] = ValueOf(arg)
	}

	// Specific behaviors for HTML nodes
	if node, ok := v.data.(*html.Node); ok {
		if node.Type == html.ElementNode && node.Data == "html" && node.Parent != nil && node.Parent.Type == html.DocumentNode {
			docNode := node.Parent // The true DocumentNode, used for global searches

			switch method {
			case "getElementById": // Typically called on the document
				if len(args) == 1 && ValueOf(args[0]).Type() == TypeString {
					id := ValueOf(args[0]).String()
					foundNode := findNodeByID(docNode, id) // Search from the true DocumentNode
					if foundNode != nil {
						return ValueOf(foundNode)
					}
				}
				return Null() // getElementById returns null if not found

			case "querySelector":
				if len(args) == 1 && ValueOf(args[0]).Type() == TypeString {
					selector := ValueOf(args[0]).String()
					foundNode := findFirstElement(docNode, selector) // Search from the true DocumentNode
					if foundNode != nil {
						return ValueOf(foundNode)
					}
				}
				return Null()

			case "querySelectorAll":
				if len(args) == 1 && ValueOf(args[0]).Type() == TypeString {
					selector := ValueOf(args[0]).String()
					foundNodes := findAllElements(docNode, selector) // Search from the true DocumentNode
					result := make([]Value, len(foundNodes))
					for i, n := range foundNodes {
						result[i] = ValueOf(n)
					}
					return ValueOf(result)
				}
				return ValueOf([]Value{})

			case "createElement":
				if len(args) == 1 && ValueOf(args[0]).Type() == TypeString {
					tagName := ValueOf(args[0]).String()
					newNode := &html.Node{
						Type:     html.ElementNode,
						Data:     tagName,
						DataAtom: atom.Lookup([]byte(tagName)),
					}
					return ValueOf(newNode)
				}
				return Undefined()

			case "createTextNode":
				if len(args) == 1 && ValueOf(args[0]).Type() == TypeString {
					text := ValueOf(args[0]).String()
					newNode := &html.Node{
						Type: html.TextNode,
						Data: text,
					}
					return ValueOf(newNode)
				}
				return Undefined()

				// Add other methods that typically live on document here, e.g., createDocumentFragment, createComment etc.
			}
		}

		switch method {
		case "appendChild":
			if len(jsArgs) == 1 {
				childNode := jsArgs[0].Node()
				targetNode := node // Default target for appending

				// Rule 1: Auto-correction for <html> appending special elements (head/body)
				if node.Type == html.ElementNode && node.Data == "html" {
					if isSpecialElement(childNode, "head") {
						ensureSpecialElement(node, "head", childNode)
						return Undefined()
					}
					if isSpecialElement(childNode, "body") {
						ensureSpecialElement(node, "body", childNode)
						return Undefined()
					}
					// If a non-head/body element is appended directly to <html>, it usually goes to body
					if childNode.Type == html.ElementNode && !(childNode.Data == "head" || childNode.Data == "body") {
						targetNode = ensureSpecialElement(node, "body", nil) // Ensure body exists and set as target
					}
				}

				// Rule 2: Implicit <tbody> for <table> content
				if targetNode.Type == html.ElementNode && targetNode.Data == "table" {
					if isTableRelatedElement(childNode) {
						if childNode.Data == "tr" || isTableCellElement(childNode) { // tr, td, th must go into tbody
							tbody := findChildElement(targetNode, "tbody")
							if tbody == nil {
								tbody = &html.Node{Type: html.ElementNode, Data: "tbody"}
								targetNode.AppendChild(tbody)
							}
							if isTableCellElement(childNode) { // td/th must go into tr
								tr := findChildElement(tbody, "tr")
								if tr == nil { // Create tr if not exists in tbody
									tr = &html.Node{Type: html.ElementNode, Data: "tr"}
									tbody.AppendChild(tr)
								}
								tr.AppendChild(childNode)
							} else { // tr, tfoot, thead
								tbody.AppendChild(childNode) // Append tr directly into tbody
							}
							return Undefined() // Handled auto-correction, skip default append
						} else { // caption, colgroup, col can go directly into table
							targetNode.AppendChild(childNode)
							return Undefined()
						}
					}
				}

				// Rule 3: Content model enforcement for <head>
				if targetNode.Type == html.ElementNode && targetNode.Data == "head" {
					if !isMetadataContent(childNode) {
						// If non-metadata content is appended to <head>, redirect to <body>
						body := ensureSpecialElement(node.Parent, "body", nil) // Assuming head's parent is html
						body.AppendChild(childNode)
						return Undefined()
					}
				}

				// Default append if no specific auto-correction applies
				fmt.Println(childNode) // DEBUG
				targetNode.AppendChild(childNode)
			}
			return Undefined()

		case "insertBefore":
			if len(jsArgs) == 2 {
				newChild := jsArgs[0].Node()
				referenceChild := jsArgs[1].Node()
				targetNode := node // Default target for insertion

				// Rule 1: Auto-correction for <html> inserting special elements (head/body)
				if node.Type == html.ElementNode && node.Data == "html" {
					if isSpecialElement(newChild, "head") {
						ensureSpecialElement(node, "head", newChild)
						return Undefined()
					}
					if isSpecialElement(newChild, "body") {
						ensureSpecialElement(node, "body", newChild)
						return Undefined()
					}
					// If a non-head/body element is inserted directly into <html>, it usually goes to body
					if newChild.Type == html.ElementNode && !(newChild.Data == "head" || newChild.Data == "body") {
						targetNode = ensureSpecialElement(node, "body", nil) // Ensure body exists and set as target
					}
				}

				// Rule 2: Implicit <tbody> for <table> content (similar to appendChild)
				if targetNode.Type == html.ElementNode && targetNode.Data == "table" {
					if isTableRelatedElement(newChild) {
						if newChild.Data == "tr" || isTableCellElement(newChild) {
							tbody := findChildElement(targetNode, "tbody")
							if tbody == nil {
								tbody = &html.Node{Type: html.ElementNode, Data: "tbody"}
								targetNode.InsertBefore(tbody, referenceChild) // Insert tbody before reference
							}
							if isTableCellElement(newChild) {
								tr := findChildElement(tbody, "tr")
								if tr == nil {
									tr = &html.Node{Type: html.ElementNode, Data: "tr"}
									tbody.InsertBefore(tr, newChild) // Insert tr before newChild
								}
								tr.InsertBefore(newChild, referenceChild) // Insert cell into tr before reference
							} else {
								tbody.InsertBefore(newChild, referenceChild) // Insert tr/tbody/thead directly
							}
							return Undefined()
						} else {
							targetNode.InsertBefore(newChild, referenceChild)
							return Undefined()
						}
					}
				}

				// Rule 3: Content model enforcement for <head> (similar to appendChild)
				if targetNode.Type == html.ElementNode && targetNode.Data == "head" {
					if !isMetadataContent(newChild) {
						body := ensureSpecialElement(node.Parent, "body", nil)
						body.InsertBefore(newChild, referenceChild)
						return Undefined()
					}
				}

				// Default insert if no specific auto-correction applies
				targetNode.InsertBefore(newChild, referenceChild)
			}
			return Undefined()

		case "removeChild":
			if len(jsArgs) == 1 {
				node.RemoveChild(jsArgs[0].Node())
			}
			return Undefined()

		case "replaceChild":
			if len(jsArgs) == 2 {
				newChild := jsArgs[0].Node()
				oldChild := jsArgs[1].Node()
				targetNode := node // Default target for replacement

				// Rule 1: Auto-correction for <html> replacing special elements (head/body)
				if node.Type == html.ElementNode && node.Data == "html" {
					if isSpecialElement(newChild, "head") {
						ensureSpecialElement(node, "head", newChild)
						node.RemoveChild(oldChild) // Still remove the old child
						return Undefined()
					}
					if isSpecialElement(newChild, "body") {
						ensureSpecialElement(node, "body", newChild)
						node.RemoveChild(oldChild) // Still remove the old child
						return Undefined()
					}
					// If a non-head/body element replaces a child in <html>, it usually goes to body
					if newChild.Type == html.ElementNode && !(newChild.Data == "head" || newChild.Data == "body") {
						targetNode = ensureSpecialElement(node, "body", nil) // Ensure body exists and set as target
					}
				}

				// Rule 2: Implicit <tbody> for <table> content (similar to appendChild/insertBefore)
				if targetNode.Type == html.ElementNode && targetNode.Data == "table" {
					if isTableRelatedElement(newChild) {
						if newChild.Data == "tr" || isTableCellElement(newChild) {
							tbody := findChildElement(targetNode, "tbody")
							if tbody == nil {
								tbody = &html.Node{Type: html.ElementNode, Data: "tbody"}
								targetNode.InsertBefore(tbody, oldChild) // Insert tbody before oldChild
							}
							if isTableCellElement(newChild) {
								tr := findChildElement(tbody, "tr")
								if tr == nil {
									tr = &html.Node{Type: html.ElementNode, Data: "tr"}
									tbody.InsertBefore(tr, newChild) // Insert tr before newChild
								}
								tr.InsertBefore(newChild, oldChild) // Insert cell into tr before oldChild
							} else {
								tbody.InsertBefore(newChild, oldChild) // Insert tr/tbody/thead directly
							}
							node.RemoveChild(oldChild) // Remove old child from original parent
							return Undefined()
						} else {
							targetNode.InsertBefore(newChild, oldChild)
							node.RemoveChild(oldChild)
							return Undefined()
						}
					}
				}

				// Rule 3: Content model enforcement for <head> (similar to appendChild)
				if targetNode.Type == html.ElementNode && targetNode.Data == "head" {
					if !isMetadataContent(newChild) {
						body := ensureSpecialElement(node.Parent, "body", nil)
						body.InsertBefore(newChild, oldChild)
						node.RemoveChild(oldChild)
						return Undefined()
					}
				}

				// Default replacement if no specific auto-correction applies
				targetNode.InsertBefore(newChild, oldChild)
				node.RemoveChild(oldChild)
			}
			return Undefined()

		case "createElement":
			if len(jsArgs) == 1 {
				tagName := jsArgs[0].String()
				newNode := &html.Node{
					Type:     html.ElementNode,
					Data:     tagName,
					DataAtom: atom.Lookup([]byte(tagName)),
				}
				return ValueOf(newNode)
			}
			return Undefined()

		case "createTextNode":
			if len(jsArgs) == 1 {
				text := jsArgs[0].String()
				newNode := &html.Node{
					Type: html.TextNode,
					Data: text,
				}
				return ValueOf(newNode)
			}
			return Undefined()

		case "cloneNode":
			// Implementation of cloneNode, with deep cloning support
			deep := false
			if len(jsArgs) == 1 {
				deep = jsArgs[0].Bool()
			}
			return ValueOf(cloneNodeInternal(node, deep))

		case "getAttribute":
			if len(jsArgs) == 1 {
				attrName := jsArgs[0].String()
				for _, a := range node.Attr {
					if a.Key == attrName {
						return ValueOf(a.Val)
					}
				}
			}
			return Null() // getAttribute returns null if the attribute is not found

		case "setAttribute":
			if len(jsArgs) == 2 {
				attrName := jsArgs[0].String()
				attrValue := jsArgs[1].String()
				setAttribute(node, attrName, attrValue)
			}
			return Undefined()

		case "removeAttribute":
			if len(jsArgs) == 1 {
				attrName := jsArgs[0].String()
				removeAttribute(node, attrName)
			}
			return Undefined()

		case "append": // Modern method (Node.append)
			for _, arg := range jsArgs {
				// arg.Node() implies arg is already a Value wrapping an *html.Node in SSR.
				childNode := arg.Node()

				// If childNode is nil, something went wrong before the append.
				if childNode == nil {
					panic("Attempted to append a nil childNode.")
				}

				targetNode := node // The node on which 'append' was called (e.g., the <head> node)

				// Rule 1: Auto-correction for <html> appending special elements (head/body)
				if node.Type == html.ElementNode && node.Data == "html" {
					if isSpecialElement(childNode, "head") {
						ensureSpecialElement(node, "head", childNode)
						continue // Skip default append
					}
					if isSpecialElement(childNode, "body") {
						ensureSpecialElement(node, "body", childNode)
						continue // Skip default append
					}
					// If a non-head/body element is appended to <html>, it usually goes to body
					if childNode.Type == html.ElementNode && !(childNode.Data == "head" || childNode.Data == "body") {
						targetNode = ensureSpecialElement(node, "body", nil) // Ensure body exists and set as target
						// No continue here, it flows to the default append (or table/head rules)
					}
				}

				// Rule 2: Implicit <tbody> for <table> content
				if targetNode.Type == html.ElementNode && targetNode.Data == "table" {
					if isTableRelatedElement(childNode) {
						if childNode.Data == "tr" || isTableCellElement(childNode) {
							fmt.Printf("DEBUG: append: TABLE appending TR/TD, handling tbody/tr creation. Child: %s (Addr:%p)\n", childNode.Data, childNode)
							tbody := findChildElement(targetNode, "tbody")
							if tbody == nil {
								tbody = &html.Node{Type: html.ElementNode, Data: "tbody"}
								targetNode.AppendChild(tbody)
							}
							if isTableCellElement(childNode) {
								tr := findChildElement(tbody, "tr")
								if tr == nil {
									tr = &html.Node{Type: html.ElementNode, Data: "tr"}
									tbody.AppendChild(tr)
								}
								tr.AppendChild(childNode)
							} else {
								tbody.AppendChild(childNode)
							}
							continue // Handled auto-correction, skip default append
						} else {
							targetNode.AppendChild(childNode) // Append other table-related element (e.g., caption, colgroup)
							continue
						}
					}
				}

				// Rule 3: Content model enforcement for <head>
				if targetNode.Type == html.ElementNode && targetNode.Data == "head" {
					if !isMetadataContent(childNode) {
						body := ensureSpecialElement(node.Parent, "body", nil) // Assuming head's parent is html
						body.AppendChild(childNode)
						continue // Skip default append
					}
					// No continue here, it flows to the default append
				}
				targetNode.AppendChild(childNode) // Default append if no specific auto-correction applies
			}
			return Undefined()
		case "prepend": // Modern method (Node.prepend)
			for i := len(jsArgs) - 1; i >= 0; i-- { // Insert in reverse order for prepend
				childNode := jsArgs[i].Node()
				targetNode := node // Default target

				// Rule 1: Auto-correction for <html> prepending special elements (head/body)
				if node.Type == html.ElementNode && node.Data == "html" {
					if isSpecialElement(childNode, "head") {
						ensureSpecialElement(node, "head", childNode)
						continue // Skip default prepend
					}
					if isSpecialElement(childNode, "body") {
						ensureSpecialElement(node, "body", childNode)
						continue // Skip default prepend
					}
					// If a non-head/body element is prepended to <html>, it usually goes to body
					if childNode.Type == html.ElementNode && !(childNode.Data == "head" || childNode.Data == "body") {
						body := ensureSpecialElement(node, "body", nil) // Ensure body exists and set as target
						body.InsertBefore(childNode, body.FirstChild)
						continue // Skip default prepend
					}
				}

				// Rule 2: Implicit <tbody> for <table> content (similar to prepend)
				if targetNode.Type == html.ElementNode && targetNode.Data == "table" {
					if isTableRelatedElement(childNode) {
						if childNode.Data == "tr" || isTableCellElement(childNode) {
							tbody := findChildElement(targetNode, "tbody")
							if tbody == nil {
								tbody = &html.Node{Type: html.ElementNode, Data: "tbody"}
								targetNode.InsertBefore(tbody, targetNode.FirstChild) // Insert tbody at the beginning
							}
							if isTableCellElement(childNode) {
								tr := findChildElement(tbody, "tr")
								if tr == nil {
									tr = &html.Node{Type: html.ElementNode, Data: "tr"}
									tbody.InsertBefore(tr, tbody.FirstChild) // Insert tr at beginning of tbody
								}
								tr.InsertBefore(childNode, tr.FirstChild) // Insert cell into tr at beginning
							} else {
								tbody.InsertBefore(childNode, tbody.FirstChild) // Insert tr/tbody/thead directly
							}
							continue
						} else {
							targetNode.InsertBefore(childNode, targetNode.FirstChild)
							continue
						}
					}
				}

				// Rule 3: Content model enforcement for <head> (similar to prepend)
				if targetNode.Type == html.ElementNode && targetNode.Data == "head" {
					if !isMetadataContent(childNode) {
						body := ensureSpecialElement(node.Parent, "body", nil)
						body.InsertBefore(childNode, body.FirstChild)
						continue
					}
				}
				node.InsertBefore(childNode, node.FirstChild) // Default prepend
			}
			return Undefined()
		case "remove": // Modern method (Element.remove)
			// If the Value represents a collection (e.g., from querySelectorAll)
			if nodes := v.nodes(); nodes != nil {
				for _, n := range nodes {
					if n.Parent != nil {
						n.Parent.RemoveChild(n)
					}
				}
				return Undefined()
			}
			// If the Value represents a single node
			if node.Parent != nil {
				node.Parent.RemoveChild(node)
			}
			return Undefined()
		case "replaceWith": // Modern method (Element.replaceWith)
			if node.Parent == nil { // Cannot replace a node without a parent
				panic("cannot replace a node without a parent with replaceWith")
			}
			for _, arg := range jsArgs {
				newNode := arg.Node()
				// Element.replaceWith behavior for special elements needs auto-correction
				// if the replaced node is html, and the new child is head/body, etc.
				if node.Parent.Type == html.ElementNode && node.Parent.Data == "html" { // If replacing a child of <html>
					if isSpecialElement(newNode, "head") {
						ensureSpecialElement(node.Parent, "head", newNode)
						// No need to insert newNode directly here, it's handled by ensureSpecialElement
					} else if isSpecialElement(newNode, "body") {
						ensureSpecialElement(node.Parent, "body", newNode)
						// No need to insert newNode directly here
					} else if newNode.Type == html.ElementNode && !(newNode.Data == "head" || newNode.Data == "body") {
						body := ensureSpecialElement(node.Parent, "body", nil) // Ensure body exists
						body.InsertBefore(newNode, node)                       // Insert into body before old node
					} else {
						node.Parent.InsertBefore(newNode, node) // Default insert
					}
				} else if node.Type == html.ElementNode && node.Data == "table" { // If replacing the table itself
					// This case is complex if replacing the table with table-related elements.
					// For simplicity in replaceWith, we assume general element replacement.
					node.Parent.InsertBefore(newNode, node)
				} else if node.Parent.Type == html.ElementNode && node.Parent.Data == "table" { // If replacing a child of table
					if isTableRelatedElement(newNode) {
						if newNode.Data == "tr" || isTableCellElement(newNode) {
							tbody := findChildElement(node.Parent, "tbody")
							if tbody == nil { // If no tbody exists in parent table
								tbody = &html.Node{Type: html.ElementNode, Data: "tbody"}
								node.Parent.InsertBefore(tbody, node) // Insert tbody before the node being replaced
							}
							if isTableCellElement(newNode) {
								tr := findChildElement(tbody, "tr")
								if tr == nil {
									tr = &html.Node{Type: html.ElementNode, Data: "tr"}
									tbody.InsertBefore(tr, newNode)
								}
								tr.InsertBefore(newNode, node)
							} else {
								tbody.InsertBefore(newNode, node)
							}
						} else {
							node.Parent.InsertBefore(newNode, node)
						}
					} else {
						node.Parent.InsertBefore(newNode, node) // Default insert
					}
				} else if node.Parent.Type == html.ElementNode && node.Parent.Data == "head" { // If replacing a child of head
					if !isMetadataContent(newNode) {
						body := ensureSpecialElement(node.Parent.Parent, "body", nil) // Head's parent is html
						body.InsertBefore(newNode, node)
					} else {
						node.Parent.InsertBefore(newNode, node)
					}
				} else { // Generic replacement
					node.Parent.InsertBefore(arg.Node(), node)
				}
			}
			node.Parent.RemoveChild(node)
			return Undefined()

		case "addEventListener", "removeEventListener":
			// No-op for a server-side context. These functions have no meaning here.
			return Undefined()

		default:
			// If the method is not recognized on an HTML node
			return Undefined()
		}
	}

	// Default behaviors for other Value types (e.g., Go functions or objects)
	if _, ok := v.data.(Func); ok {
		// If the Value is a Func, attempt to invoke it.
		// For now, 'Invoke' handles the direct function execution.
		return Undefined()
	}

	// For other Value types, most method calls have no meaning.
	return Undefined()
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

func DEBUG(msg string, args ...interface{}) {
	// This is a placeholder for debugging output.
	// In a real application, you might use log.Printf or fmt.Printf.
	fmt.Printf(msg+"\n", args...)
}

// findChildElement finds the first child of a node with the given tag name.
// It searches only direct children that are html.ElementNode.
func findChildElement(parent *html.Node, tagName string) *html.Node {
	for c := parent.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tagName {
			return c
		}
	}
	return nil
}

// findFirstElement is a simplified implementation of querySelector for the example.
// It currently only handles simple selectors by tag name or by ID (prefixed with #).
func findFirstElement(root *html.Node, selector string) *html.Node {
	var found *html.Node
	var traverse func(*html.Node) bool

	traverse = func(n *html.Node) bool {
		if n.Type == html.ElementNode {
			if strings.HasPrefix(selector, "#") { // ID selector
				id := strings.TrimPrefix(selector, "#")
				for _, attr := range n.Attr {
					if attr.Key == "id" && attr.Val == id {
						found = n
						return true
					}
				}
			} else if n.Data == selector { // Tag name selector
				found = n
				return true
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if traverse(c) {
				return true
			}
		}
		return false
	}

	traverse(root)
	return found
}

// findAllElements is a simplified implementation of querySelectorAll for the example.
// It currently only handles simple selectors by tag name or by ID (prefixed with #).
func findAllElements(root *html.Node, selector string) []*html.Node {
	var results []*html.Node
	var traverse func(*html.Node)

	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if strings.HasPrefix(selector, "#") { // ID selector
				id := strings.TrimPrefix(selector, "#")
				for _, attr := range n.Attr {
					if attr.Key == "id" && attr.Val == id {
						results = append(results, n)
						break // ID is unique, so no need to continue for this node
					}
				}
			} else if n.Data == selector { // Tag name selector
				results = append(results, n)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(root)
	return results
}

// findNodeByID is a helper function for getElementById.
func findNodeByID(root *html.Node, id string) *html.Node {
	var found *html.Node
	var traverse func(*html.Node) bool

	traverse = func(n *html.Node) bool {
		if n.Type == html.ElementNode {
			for _, attr := range n.Attr {
				if attr.Key == "id" && attr.Val == id {
					found = n
					return true
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if traverse(c) {
				return true
			}
		}
		return false
	}

	traverse(root)
	return found
}

// cloneNodeInternal is a helper function for the recursive cloning of nodes.
func cloneNodeInternal(n *html.Node, deep bool) *html.Node {
	if n == nil {
		return nil
	}
	clone := &html.Node{
		Type:      n.Type,
		DataAtom:  n.DataAtom,
		Data:      n.Data,
		Namespace: n.Namespace,
		Attr:      append([]html.Attribute(nil), n.Attr...), // Copies attributes
	}

	if deep {
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			clone.AppendChild(cloneNodeInternal(child, true))
		}
	}
	return clone
}

// isSpecialElement checks if a node is a special HTML element (head or body).
func isSpecialElement(node *html.Node, tagName string) bool {
	return node != nil && node.Type == html.ElementNode && (node.Data == "head" || node.Data == "body") && node.Data == tagName
}

// ensureSpecialElement finds an existing special element (head/body) within a parent,
// or creates it if missing. It then moves children from nodeToMerge into this ensured element.
// Returns the actual element where content was merged.
func ensureSpecialElement(parent *html.Node, targetTagName string, nodeToMerge *html.Node) *html.Node {
	// 1. Try to find an existing special element
	existingElement := findChildElement(parent, targetTagName)

	// 2. If not found, create and append it
	if existingElement == nil {
		existingElement = &html.Node{Type: html.ElementNode, Data: targetTagName}
		// Ensure correct order: head before body
		if targetTagName == "body" {
			head := findChildElement(parent, "head")
			if head != nil {
				parent.InsertBefore(existingElement, head.NextSibling) // Insert after head
			} else {
				parent.AppendChild(existingElement) // If no head, just append
			}
		} else { // targetTagName == "head"
			body := findChildElement(parent, "body")
			if body != nil {
				parent.InsertBefore(existingElement, body) // Insert before body
			} else {
				parent.AppendChild(existingElement) // If no body, just append
			}
		}
	}

	// If nodeToMerge has attributes, apply them to existingElement.
	// This will overwrite existing attributes with the same key, and add new ones.
	if nodeToMerge != nil {
		// Clear existing attributes on existingElement first if you want nodeToMerge's attrs to be canonical.
		// Or, merge them. Let's do a merge (setAttribute for each).
		for _, attr := range nodeToMerge.Attr {
			setAttribute(existingElement, attr.Key, attr.Val) // Use your existing setAttribute helper
		}
	}

	// 3. Move children from nodeToMerge into the existing/created element
	if nodeToMerge != nil {
		for child := nodeToMerge.FirstChild; child != nil; {
			next := child.NextSibling

			nodeToMerge.RemoveChild(child) // Ensure child is detached from nodeToMerge

			// You might even check if child.Parent is still non-nil here and panic
			if child.Parent != nil {
				panic(fmt.Sprintf("CRITICAL ERROR: Child %s still attached to %v after RemoveChild!", child.Data, child.Parent.Data))
			}
			existingElement.AppendChild(child) // Append to the existing/newly created head/body
			child = next
		}
	}
	return existingElement
}

// isTableRelatedElement checks if a node is an element typically found directly within a <table>.
func isTableRelatedElement(node *html.Node) bool {
	if node == nil || node.Type != html.ElementNode {
		return false
	}
	switch node.Data {
	case "caption", "colgroup", "col", "thead", "tfoot", "tbody", "tr":
		return true
	default:
		return false
	}
}

// isTableCellElement checks if a node is a <td> or <th>.
func isTableCellElement(node *html.Node) bool {
	if node == nil || node.Type != html.ElementNode {
		return false
	}
	return node.Data == "td" || node.Data == "th"
}

// isMetadataContent checks if a node is considered metadata content (valid inside <head>).
func isMetadataContent(node *html.Node) bool {
	if node == nil || node.Type != html.ElementNode {
		return false
	}
	switch node.Data {
	case "base", "link", "meta", "noscript", "script", "style", "template", "title":
		return true
	default:
		return false
	}
}

func (v Value) Get(property string) Value {
	if v.data == nil {
		return Undefined()
	}

	// handling stub elements
	if _, ok := v.data.(*stubwindow); ok {
		if property == "document" {
			return ValueOf(stubWindow.document) // Return the default document for stub windows
		} else {
			panic(fmt.Sprintf("Property %s not found on stub window", property))
			// return Undefined() // For other properties, return Undefined
		}
	}

	if _, ok := v.data.(*stubdocument); ok {
		if property == "implementation" {
			return ValueOf(stubWindow.document.implementation) // Return the default document implementation for stub documents
		} else {
			panic(fmt.Sprintf("Property %s not found on stub document", property))
		}
	}

	if v.IsUndefined() || v.IsNull() {
		return Undefined() // If the Value is undefined or null, return Undefined
	}

	node := v.Node()

	if property == "isConnected" {
		// isConnected returns true if the node is part of the document tree.
		// This is a simplified check; in a real DOM, you'd traverse up to find the Document node.
		return ValueOf(node.Parent != nil && node.Parent.Type == html.DocumentNode)
	}

	if property == "parentElement" {
		// Returns the parent element if it exists, otherwise returns null.
		if node.Parent != nil && node.Parent.Type == html.ElementNode {
			return ValueOf(node.Parent)
		}
		return Null() // If no parent element, return Null
	}
	// Special cases for the Document node.
	if node.Type == html.DocumentNode { // Check if it's a Document node
		switch property {
		case "documentElement":
			htmlNode := findChildElement(node, "html")
			if htmlNode != nil {
				return ValueOf(htmlNode)
			}
			return Undefined()
		case "head":
			docElem := v.Get("documentElement").Node() // Using unexported Node()
			if docElem != nil {
				headNode := findChildElement(docElem, "head")
				if headNode != nil {
					return ValueOf(headNode)
				}
			}
			return Undefined()
		case "body":
			docElem := v.Get("documentElement").Node() // Using unexported Node()
			if docElem != nil {
				bodyNode := findChildElement(docElem, "body")
				if bodyNode != nil {
					return ValueOf(bodyNode)
				}
			}
			return Undefined()
		case "implementation": // document.implementation
			return ValueOf(defaultDocumentImplementation)
		case "defaultView": // When called on the document, this refers to the containing window.
			// In browser, this returns the Window. Here, since our document is not in a *real* browsing context,
			// it should strictly be null. However, if we want to provide *some* window-like object,
			// it would be the DefaultWindow. Sticking to Null() for strict DOM spec compliance outside browsing context.
			return Null() // Or ValueOf(DefaultWindow) if you prefer a non-null but dummy Window object.
		}
	}

	if node.Type == html.ElementNode && node.Data == "html" && node.Parent != nil && node.Parent.Type == html.DocumentNode {
		// docNode is the actual html.DocumentNode parent, used for global searches
		docNode := node.Parent

		switch property {
		case "head", "body", "documentElement", "implementation", "defaultView":
			return ValueOf(docNode).Get(property)
		case "nodeName": // For document, nodeName is "#document"
			return ValueOf("#document")
		case "nodeType": // For document, nodeType is html.DocumentNode
			return ValueOf(html.DocumentNode) // Return the conceptual DocumentNode type
		}
	}

	// DEBUG TODO handle .Get("href") for anchor elements?

	switch property {
	case "ownerDocument":
		// For any node, ownerDocument returns the Document node it belongs to.
		if node.Type == html.DocumentNode {
			return Null() // ValueOf(node) // If it's already a Document node, return null
		}
		for p := node.Parent; p != nil; p = p.Parent {
			if p.Type == html.DocumentNode {
				return ValueOf(p) // Return the Document node
			}
		}
		return Undefined() // If no Document node found, return Undefined
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
		for _, attr := range v.Node().Attr {
			attrs[attr.Key] = attr.Val
		}
		return ValueOf(attrs)

	case "tagName":
		return ValueOf(v.Node().Data)

	case "childElementCount":
		count := 0
		for c := v.Node().FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode {
				count++
			}
		}
		return ValueOf(count)

	case "firstElementChild":
		for c := v.Node().FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode {
				return ValueOf(c)
			}
		}
		return Undefined()

	case "lastElementChild":
		var lastChild *html.Node
		for c := v.Node().FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode {
				lastChild = c
			}
		}
		return ValueOf(lastChild)

	case "nextElementSibling":
		for c := v.Node().NextSibling; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode {
				return ValueOf(c)
			}
		}
		return Undefined()

	case "previousElementSibling":
		for c := v.Node().PrevSibling; c != nil; c = c.PrevSibling {
			if c.Type == html.ElementNode {
				return ValueOf(c)
			}
		}
		return Undefined()

	case "nodeName":
		return ValueOf(v.Node().Data)

	case "nodeType":
		return ValueOf(v.Node().Type)
	case "parentNode":
		// Returns the parent node of the current node.
		if v.Node().Parent != nil {
			return ValueOf(v.Node().Parent)
		}
		return Null() // If no parent, return Null (consistent with DOM spec)
	case "nodeValue":
		// Returns the node's value, which is relevant for Text, Comment, and ProcessingInstruction nodes.
		switch v.Node().Type {
		case html.TextNode, html.RawNode, html.CommentNode:
			return ValueOf(v.Node().Data) // Return the node's data as its value
		case html.DocumentNode, html.DoctypeNode:
			return Null() // For Document and Doctype nodes, nodeValue is null
		default:
			return Undefined() // For other node types, return Undefined
		}
	case "nextSibling":
		// Returns the next sibling node of the current node.
		if v.Node().NextSibling != nil {
			return ValueOf(v.Node().NextSibling)
		}
		return Null() // If no next sibling, return Null (consistent with DOM spec)
	case "previousSibling":
		// Returns the previous sibling node of the current node.
		if v.Node().PrevSibling != nil {
			return ValueOf(v.Node().PrevSibling)
		}
		return Null() // If no previous sibling, return Null (consistent with DOM spec)
	case "firstChild":
		// Returns the first child node of the current node.
		if v.Node().FirstChild != nil {
			return ValueOf(v.Node().FirstChild)
		}
		return Null() // If no first child, return Null (consistent with DOM spec)
	case "lastChild":
		// Returns the last child node of the current node.
		if v.Node().LastChild != nil {
			return ValueOf(v.Node().LastChild)
		}
		return Null() // If no last child, return Null (consistent with DOM spec)
	case "textContent":
		node := v.Node() // Panics if not a Node, consistent with JS runtime errors
		// If the node is a document or a doctype, textContent returns null.
		if node.Type == html.DocumentNode || node.Type == html.DoctypeNode {
			return Null()
		}
		// For other node types, use the helper
		return ValueOf(getTextContent(node))
	case "innerHTML":
		node := v.Node() // Panics if not a Node, consistent with JS runtime errors
		if node.Type == html.DocumentNode || node.Type == html.DoctypeNode {
			return Null() // innerHTML on Document/Doctype returns null
		}
		var b strings.Builder
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode || c.Type == html.TextNode || c.Type == html.RawNode {
				// For Element, Text, and Raw nodes, we can serialize their HTML.
				html.Render(&b, c) // Serializes the node to HTML
			} else if c.Type == html.CommentNode {
				// For Comment nodes, we can directly append their data.
				b.WriteString("<!--" + c.Data + "-->")
			}
		}
		return ValueOf(b.String())
	case "OuterHTML":
		node := v.Node() // Panics if not a Node, consistent with JS runtime errors
		if node.Type == html.DocumentNode || node.Type == html.DoctypeNode {
			return Null() // outerHTML on Document/Doctype returns null
		}
		var b strings.Builder
		html.Render(&b, node) // Serializes the entire node to HTML
		return ValueOf(b.String())
	default:
		return Undefined()
	}
}

// getTextContent retrieves the concatenated text content of a node's descendants.
//
// Rules:
//   - For Text or CDATA nodes: returns n.Data.
//   - For Comment or ProcessingInstruction nodes: returns n.Data (their own textContent).
//   - For Document or Doctype nodes: returns "" (Value.Get will convert to Null()).
//   - For Element or DocumentFragment nodes: returns the concatenation of their children's textContent,
//     EXCLUDING Comment and ProcessingInstruction children.
func getTextContent(n *html.Node) string {
	if n == nil {
		return ""
	}

	switch n.Type {
	case html.TextNode, html.RawNode:
		return n.Data
	case html.CommentNode:
		return n.Data // A CommentNode's own textContent is its data.
	case html.DocumentNode, html.DoctypeNode:
		return "" // Will be Null() in Value.Get, conforming to DOM spec.
	case html.ElementNode: // Handles all HTML elements.
		var b strings.Builder
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			// EXCLUDE comment nodes when concatenating children for the parent's text content.
			if c.Type != html.CommentNode {
				b.WriteString(getTextContent(c)) // Recursive call.
			}
		}
		return b.String()
	default:
		// This 'default' case primarily covers html.ErrorNode, which has no meaningful text content,
		// or any other unhandled/less common html.NodeType values.
		return ""
	}
}

// toString converts any interface{} to its string representation,
// mimicking JavaScript's string coercion.
// If the input is a js.Value, it calls its String() method.
// Otherwise, it uses fmt.Sprintf("%v", i).
func toString(i interface{}) string {
	if v, ok := i.(Value); ok {
		return v.String() // Value.String() handles its underlying data
	}
	// Handles raw Go types like int, bool, float64, nil
	return fmt.Sprintf("%v", i)
}

func clearChildren(node *html.Node) {
	if node == nil {
		panic("clearChildren called with nil node")
	}
	for child := node.FirstChild; child != nil; {
		next := child.NextSibling
		node.RemoveChild(child)
		child = next
	}
}

func setChildren(node *html.Node, children ...*html.Node) {
	if node == nil {
		panic("setChildren called with nil node")
	}
	// Clear existing children
	clearChildren(node)
	// Append new children
	for _, child := range children {
		node.AppendChild(child)
	}
}

func (v Value) Set(property string, value interface{}) {
	if v.data == nil {
		return
	}

	switch v.data.(type) {
	case *stubwindow:
		DEBUG("Cannot set properties on a stub window directly. Use stubWindow.document or stubWindow.defaultView.")
		return
	case *stubdocument:
		DEBUG("Cannot set properties on a stub document directly. Use stubDocument.documentElement or stubDocument.body.")
		return
	case *documentImplementation:
		DEBUG("Cannot set properties on a document implementation directly. Use stubDocument.documentElement or stubDocument.body.")
		return
	}

	node := v.Node() // This will panic if v.data is not a *html.Node
	// Handle document object specific properties setting
	if node.Type == html.ElementNode && node.Data == "html" && node.Parent != nil && node.Parent.Type == html.DocumentNode {
		// Special case for setting properties on the <html> element
		if property == "lang" {
			setAttribute(node, "lang", toString(value))
			return
		} else if property == "dir" {
			setAttribute(node, "dir", toString(value))
			return
		} else if property == "style" {
			if styleStr, ok := value.(string); ok {
				setAttribute(node, "style", styleStr)
				return
			}
			panic("Invalid value for 'style', expected string")
		} else if property == "title" {
			if headNode := findChildElement(node, "head"); headNode != nil {
				if titleNode := findChildElement(headNode, "title"); titleNode != nil {
					setChildren(titleNode, &html.Node{Type: html.TextNode, Data: toString(value)})
				} else {
					// Create <title> if it doesn't exist
					newTitle := &html.Node{Type: html.ElementNode, Data: "title"}
					newTitle.AppendChild(&html.Node{Type: html.TextNode, Data: toString(value)})
					headNode.AppendChild(newTitle)
				}
				return
			} else {
				panic("Cannot set title without a <head> element")
			}

		} else if property == "charset" {
			if headNode := findChildElement(node, "head"); headNode != nil {
				// Find existing <meta charset> or create one
				var metaCharsetNode *html.Node
				for c := headNode.FirstChild; c != nil; c = c.NextSibling {
					if c.Type == html.ElementNode && c.Data == "meta" {
						for _, attr := range c.Attr {
							if attr.Key == "charset" {
								metaCharsetNode = c
								break
							}
						}
					}
					if metaCharsetNode != nil {
						break
					}
				}

				if metaCharsetNode != nil {
					// Update existing charset attribute
					setAttribute(metaCharsetNode, "charset", toString(value))
				} else {
					// Create new <meta charset> tag
					newMeta := &html.Node{Type: html.ElementNode, Data: "meta"}
					newMeta.Attr = []html.Attribute{
						{Key: "charset", Val: toString(value)},
					}
					headNode.AppendChild(newMeta)
				}
				return
			} else {
				panic("Cannot set charset without a <head> element") // Still a good panic
			}
		}
	}

	// if the element is an anchor link, handle text property specifically
	if node.Type == html.ElementNode && node.Data == "a" && property == "text" {
		// If the anchor tag has a textContent, we set it as the link text.
		if text, ok := value.(string); ok {
			// Set the textContent of the anchor element
			setChildren(node, &html.Node{Type: html.TextNode, Data: text})
			return
		} else {
			panic("Invalid value for 'text', expected string")
		}
	}

	switch property {
	case "id", "class", "name", "type", "value", "href", "src": // Example attributes
		valStr := toString(value)
		setAttribute(v.Node(), property, valStr)

	case "textContent", "innerText":
		node := v.Node()
		// Setting textContent on Document or Doctype has no effect.
		if node.Type == html.DocumentNode || node.Type == html.DoctypeNode {
			return // No-op
		}

		var textToSet string

		// If the value is a Value type
		if val, ok := value.(Value); ok {
			// Check if it's explicitly a TextNode
			if valNode, isNode := val.data.(*html.Node); isNode && valNode.Type == html.TextNode {
				// If it's already a text node Value, we set its data directly
				textToSet = valNode.Data
			} else {
				// If it's a Value but not a TextNode, get its textContent.
				// This handles cases like: element.textContent = anotherElementValue
				// where anotherElementValue's textContent should be used.
				textToSet = getTextContent(val.Node()) // Assuming val.Node() will panic if not a node
			}
		} else {
			// For all other types (string, int, float, bool, etc.),
			// use the generic toString helper.
			textToSet = toString(value)
		}

		// For CDATA, Comment, PI, or Text nodes, set their Node.nodeValue (n.Data).
		switch node.Type {
		case html.TextNode, html.CommentNode:
			node.Data = textToSet
			// Ensure no children for these types if they somehow had any
			clearChildren(node)
			return
		}

		// For other node types (Element, DocumentFragment etc.),
		// remove all existing children and append a single new text node.
		textNode := &html.Node{
			Type: html.TextNode,
			Data: textToSet,
		}
		setChildren(node, textNode)
	case "innerHTML":
		node := v.Node()
		val := toString(value)
		// Clear existing children
		clearChildren(node)

		// Check if it's a script or style tag (or potentially others that contain plain text)
		// Use DataAtom for efficient comparison
		if node.Type == html.ElementNode && (node.DataAtom == atom.Script || node.DataAtom == atom.Style) {
			// For script and style, content is plain text, not HTML.
			// Insert it as a single TextNode.
			newTextNode := &html.Node{
				Type: html.TextNode,
				Data: val,
			}
			node.AppendChild(newTextNode)
		} else {
			// For all other elements, parse as HTML fragment
			parsedHTML, err := html.ParseFragment(strings.NewReader(val), node)
			if err != nil {
				fmt.Printf("ParseFragment error for %s (val: %q): %v\n", node.Data, val, err)
				panic(err) // Or handle more gracefully, e.g., log and return
			}
			for _, newNode := range parsedHTML {
				node.AppendChild(newNode)
			}
		}
	case "outerHTML":
		node := v.Node()
		val := toString(value)

		if node.Parent == nil {
			// Special case: If the node is the root (e.g., the <html> node or document itself),
			// outerHTML behavior is usually undefined or replaces the content of the root.
			// The *html.Node argument to html.Parse() is the Document node.
			// If 'node' is the Document node, replacing its 'outerHTML' means replacing its children.
			// If 'node' is the <html> node, its outerHTML replacement means replacing the root element itself.
			// This is complex and might depend on your specific library's semantics for the root.
			// For simplicity, let's assume it replaces children if it's the document node.
			// If it's the <html> node, you'd typically replace the *document's* child.

			// If 'node' is the document root and you're treating it as the parent,
			// this is like innerHTML for the document.
			// Clear children and then parse fragment into it.
			// This behavior for outerHTML on the document root is non-standard browser-wise.
			clearChildren(node)
			parsedHTML, err := html.ParseFragment(strings.NewReader(val), node) // Use node itself as context
			if err != nil {
				fmt.Println("parse error for outerHTML on root:", val, node)
				panic(err)
			}
			for _, newNode := range parsedHTML {
				node.AppendChild(newNode)
			}

		} else {
			// Normal case: node has a parent
			// Use node.Parent as context for parsing
			parsedHTML, err := html.ParseFragment(strings.NewReader(val), node.Parent)
			if err != nil {
				fmt.Println("parse error for outerHTML:", val, node)
				panic(err) // Handle error appropriately
			}

			// Replace the current node with the new one(s)
			for _, newNode := range parsedHTML {
				node.Parent.InsertBefore(newNode, node)
			}
			// After inserting all new nodes, remove the old node
			node.Parent.RemoveChild(node)
		}
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

// Float conversion in server context.
func (v Value) Float() float64 {
	if f, ok := v.data.(float64); ok {
		return f
	}
	if i, ok := v.data.(int); ok {
		return float64(i)
	}
	if s, ok := v.data.(string); ok {
		// Attempt to parse string to float64, like JS does
		f, err := strconv.ParseFloat(s, 64)
		if err == nil {
			return f
		}
		// If parsing fails, JS often results in NaN
		return math.NaN()
	}
	// For other non-numeric types, JS often results in NaN or 0 depending on context.
	// Panicking is also an option if you want strictness for non-numeric types.
	return math.NaN() // or 0.0, or panic("Value is not a number")
}

// Index in server context.Useful to implement query functions such as `document.querySelectorAll` or element.Children.
// Typically when we have to deal with list of nodes.
func (v Value) Index(i int) Value {
	if nodes, ok := v.data.([]*html.Node); ok { // If Value wraps a slice of HTML nodes
		if i >= 0 && i < len(nodes) {
			return ValueOf(nodes[i])
		}
		return Undefined() // Out of bounds for slice of nodes
	}
	if vals, ok := v.data.([]Value); ok { // If Value wraps a slice of Values
		if i >= 0 && i < len(vals) {
			return vals[i]
		}
		return Undefined() // Out of bounds for slice of Values
	}
	// If it's a string, index returns a Value wrapping the character
	if s, ok := v.data.(string); ok {
		if i >= 0 && i < len(s) {
			return ValueOf(string(s[i]))
		}
		return Undefined() // Out of bounds for string
	}
	// For non-indexable types, JS often returns undefined.
	return Undefined()
}

// InstanceOf as a limited version in server context.
func (v Value) InstanceOf(t Value) bool {
	// Very simplified: check if both are HTML Nodes of the same type.
	// This is not a full JS InstanceOf, but a practical server-side approximation.
	vNode, vIsNode := v.data.(*html.Node)
	tNode, tIsNode := t.data.(*html.Node)

	if vIsNode && tIsNode {
		// Example: element.InstanceOf(document.createElement('div'))
		// Or if you pass a special 'HTMLElement' Value
		return vNode.Type == tNode.Type && vNode.Data == tNode.Data // Crude, but type-aware
	}
	// For primitives, it would always be false against an 'object' type.
	return false
}

// Invoke is calls the registered function with the arguments converted to Value types in server context.
func (v Value) Invoke(args ...interface{}) Value {
	if f, ok := v.data.(Func); ok {
		// If your Func type stores the original Go function:
		if f.f != nil { // Assuming Func struct has a field `f` holding the func(this Value, args []Value) interface{}
			// Convert args to []Value
			jsArgs := make([]Value, len(args))
			for i, arg := range args {
				jsArgs[i] = ValueOf(arg)
			}
			// Call the Go function
			result := f.f(v, jsArgs) // 'this' is 'v'
			return ValueOf(result)
		}
	}
	// If not a callable function, return Undefined()
	return Undefined()
}

// IsNaN always returns false in server context.
func (v Value) IsNaN() bool {
	if f, ok := v.data.(float64); ok {
		return math.IsNaN(f)
	}
	return false // Only floats can be NaN
}

// IsNull checks if the Value is null.
func (v Value) IsNull() bool {
	return v.data == nil
}

// IsUndefined checks if the Value is undefined.
func (v Value) IsUndefined() bool {
	return v.data == TypeUndefined
}

// Length is a no-op in server context.
func (v Value) Length() int {
	if s, ok := v.data.(string); ok {
		return len(s)
	}
	if nodes, ok := v.data.([]*html.Node); ok {
		return len(nodes)
	}
	// If you return `[]Value` for children, then that needs handling too
	if vals, ok := v.data.([]Value); ok {
		return len(vals)
	}
	// Default for non-array-like types
	return 0
}

// New is a no-op in server context.
func (v Value) New(args ...interface{}) Value {
	return Undefined()
}

// SetIndex is a no-op in server context.
func (v Value) SetIndex(i int, x interface{}) {
	if nodes, ok := v.data.([]*html.Node); ok {
		if i >= 0 && i < len(nodes) {
			if htmlNode, ok := x.(*html.Node); ok { // Direct HTML node
				nodes[i] = htmlNode
			} else if val, ok := x.(Value); ok { // Value wrapping HTML node
				if htmlNode, ok := val.data.(*html.Node); ok {
					nodes[i] = htmlNode
				} else {
					// Value is not an HTML node, panic or log error
					panic(fmt.Sprintf("cannot set non-HTML node Value at index %d for *html.Node slice", i))
				}
			} else {
				// Not an HTML node or Value, panic or log error
				panic(fmt.Sprintf("cannot set non-HTML node type %T at index %d for *html.Node slice", x, i))
			}
			return
		}
		// Out of bounds is often a no-op or ignored in JS for SetIndex on arrays
		return
	}
	if vals, ok := v.data.([]Value); ok { // If Value wraps a slice of Values
		if i >= 0 && i < len(vals) {
			vals[i] = ValueOf(x) // Wrap the new value
			return
		}
		return
	}
	// For other types, SetIndex is typically a no-op or error
}

// Truthy is a no-op in server context.
func (v Value) Truthy() bool {
	if v.IsNull() || v.IsUndefined() {
		return false
	}
	switch val := v.data.(type) {
	case *stubdocument:
		return false // this is a stub, not a real object
	case *stubwindow:
		return false // this is a stub, not a real object
	case bool:
		return val
	case string:
		return len(val) > 0
	case int:
		return val != 0
	case float64:
		return val != 0.0 && !v.IsNaN() // Check for 0 and NaN
	case *html.Node, []*html.Node, map[string]string, Func: // Objects are truthy if not null/undefined
		return true
	default:
		return true // Default for other non-nil types
	}
}

// Type returns the type of the Value.
func (v Value) Type() Type {
	if v.IsNull() {
		return TypeNull
	}
	if v.IsUndefined() { // Requires TypeUndefined
		return TypeUndefined
	}
	switch v.data.(type) {
	case bool:
		return TypeBoolean
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return TypeNumber
	case string:
		return TypeString
	case *html.Node, []*html.Node, map[string]string:
		return TypeObject
	case Func:
		return TypeFunction
	default:
		return TypeObject
	}
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
	// You'll need to store the actual Go function here
	f func(this Value, args []Value) interface{}
	// You might also need an ID for mapping if you manage a registry of functions
	id uintptr // For Release management, if needed
}

// FuncOf returns a new Func that wraps the given Go function.
// In a server-side context, this function is primarily useful if you
// intend to allow "JavaScript callbacks" within your Go application
// that resolve to actual Go functions.
func FuncOf(f func(this Value, args []Value) interface{}) Func {
	// For server-side, you might just store the function directly.
	// In syscall/js, it registers it with the JS runtime.
	// A simple implementation for server-side can just store the function:
	return Func{f: f}
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

// DEBUG
func CountChildrenNode(node *html.Node) int {
	count := 0
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		count++
	}
	return count
}
