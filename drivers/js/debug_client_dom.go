//go:build !server

package doc

import "golang.org/x/net/html"

// Stubs for client-side to avoid compilation errors
func CountChildrenNode(node *html.Node) int                  { return 0 }
func DebugPrintNodeTree(label string, node *html.Node)       { /* no-op */ }
func DebugPrintGlobalDocumentTree(d *Document, label string) { /* no-op */ }
