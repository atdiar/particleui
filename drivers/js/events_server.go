//go:build server

// Package doc defines the default set of Element constructors, native interfaces,
// events and event handlers, and animation properties used to build js-based UIs.
package doc

import (

	"fmt"
	"log"
	"runtime"

	"github.com/atdiar/particleui"
)



var DEBUG = log.Print // DEBUG

func SDEBUG(){
	pc := make([]uintptr, 30)
	n := runtime.Callers(0, pc)
	DEBUG(n)
	if n == 0{
		return
	}
	pc = pc[:n] // pass only valid pcs to runtime.CallersFrames
	frames := runtime.CallersFrames(pc)

	for {
		frame, more := frames.Next()

		fmt.Printf("%s\n", frame.Function)

		// Check whether there are more frames to process after this one.
		if !more {
			break
		}
	}


}




// NativeDispatch allows for the propagation of a JS event created in Go.
func NativeDispatch(evt ui.Event){}


var NativeEventBridge = func(NativeEventName string, listener *ui.Element, capture bool) {}






