//go:build !server

package js

import "syscall/js"

// Re-exporting types from syscall/js
type Value = js.Value
type Error = js.Error
type Type = js.Type
type Func = js.Func
type ValueError = js.ValueError

// Re-exporting variables from syscall/js
var (
	Global    = js.Global
	Null      = js.Null
	Undefined = js.Undefined
)

// Re-exporting functions from syscall/js
var (
	ValueOf       = js.ValueOf
	CopyBytesToGo = js.CopyBytesToGo
	CopyBytesToJS = js.CopyBytesToJS
	FuncOf        = js.FuncOf
)
