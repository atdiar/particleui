//go:build server && csr

package doc

import (
	"fmt"
	"strings"

	"golang.org/x/net/html" // This import is only used on server side
)

// dumpNode recursively prints the HTML node tree for debugging.
func dumpNode(node *html.Node, indent int) {
	if node == nil {
		return
	}
	prefix := strings.Repeat("  ", indent)
	nodeTypeStr := ""
	switch node.Type {
	case html.DocumentNode:
		nodeTypeStr = "Document"
	case html.ElementNode:
		nodeTypeStr = "Element"
	case html.TextNode:
		nodeTypeStr = "Text"
	case html.CommentNode:
		nodeTypeStr = "Comment"
	case html.DoctypeNode:
		nodeTypeStr = "Doctype"
	case html.RawNode:
		nodeTypeStr = "Raw"
	default:
		nodeTypeStr = fmt.Sprintf("Unknown(%v)", node.Type)
	}

	parentData := "N/A"
	if node.Parent != nil {
		parentData = node.Parent.Data
	}
	nextSiblingData := "N/A"
	if node.NextSibling != nil {
		nextSiblingData = node.NextSibling.Data
	}
	prevSiblingData := "N/A"
	if node.PrevSibling != nil {
		prevSiblingData = node.PrevSibling.Data
	}

	fmt.Printf("%s[%s] %s (Addr:%p) Attrs:%+v (Parent: %s, NextSibling: %s, PrevSibling: %s)\n",
		prefix, nodeTypeStr, node.Data, node, node.Attr, parentData, nextSiblingData, prevSiblingData)

	// For element nodes, also show FirstChild and LastChild directly
	if node.Type == html.ElementNode || node.Type == html.DocumentNode {
		fmt.Printf("%s  FirstChild: %p, LastChild: %p\n", prefix, node.FirstChild, node.LastChild)
	}

	for c := node.FirstChild; c != nil; c = c.NextSibling {
		dumpNode(c, indent+1)
	}
}

// CountChildrenNode is a helper to count children of an *html.Node.
// This function needs to be server-only as it operates on *html.Node.
func CountChildrenNode(node *html.Node) int {
	count := 0
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		count++
	}
	return count
}

// DebugPrintNodeTree prints the entire tree from a given root *html.Node.
// This function is intended to be called from code that already has access to *html.Node,
// like your Document.Render method, or internal Value methods.
func DebugPrintNodeTree(label string, node *html.Node) {
	fmt.Printf("\n--- DEBUG (%s): Node Tree ---\n", label)
	dumpNode(node, 0)
	fmt.Println("--- END DEBUG ---")
}

// DebugPrintGlobalDocumentTree prints the entire document tree using the global reference.
// This is safe to call from any server-side code.
func DebugPrintGlobalDocumentTree(doc *Document, label string) {
	if doc != nil {
		n := newHTMLDocument(doc).Parent
		for n.Parent != nil {
			n = n.Parent
		}
		DebugPrintNodeTree(label+" (Global Document Node)", n)
	} else {
		fmt.Printf("\n--- DEBUG (%s): Global Document Node NOT SET ---\n", label)
	}
}
