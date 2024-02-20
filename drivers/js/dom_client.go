//go:build !server

// Package doc defines the default set of Element constructors, native interfaces,
// events and event handlers, and animation properties used to build js-based UIs.

package doc

import (
	"context"
	"github.com/atdiar/particleui"
	"log"
	"net/url"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"github.com/atdiar/particleui/drivers/js/compat"
)

// TODO on init, Apply EnableMutationCapture to Elements if ldlflags -X tag is set for the buildtype variable to "dev"
// Also, the mutationtrace should be stored in the sessionstorage or localstorage
// And the mutationtrace should replay once the document is ready.

func init() {
	Elements.EnableMutationReplay()
	if DevMode != "false" && HMRMode != "false" {
		Elements.EnableMutationCapture()
	}
}

func (n NativeElement) SetChildren(children ...*ui.Element) {
	if n.typ == "HTMLElement" {
		fragment := js.Global().Get("document").Call("createDocumentFragment")
		for _, child := range children {
			v, ok := child.Native.(NativeElement)
			if !ok {
				panic("wrong format for native element underlying objects.Cannot append " + child.ID)
			}
			fragment.Call("append", v.Value)
		}
		n.Value.Call("append", fragment)
	}
}

func (n NativeElement) BatchExecute(parentid string, opslist string) {
	if n.typ == "HTMLElement" {
		js.Global().Call("applyBatchOperations", parentid, opslist)
	}
}

// NewBuilder accepts a function that builds a document as a UI tree and returns a function
// that enables event listening for this document.
// On the client, these are client side javascrip events.
// On the server, these events are http requests to a given UI endpoint, translated then in a navigation event
// for the document.
func NewBuilder(f func() Document, buildEnvModifiers ...func()) (ListenAndServe func(context.Context)) {
	for _, mod := range buildEnvModifiers {
		mod()
	}

	return func(ctx context.Context) {
		// GC is triggered only when the browser is idle.
		debug.SetGCPercent(-1)
		js.Global().Set("triggerGC", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			runtime.GC()
			return nil
		}))

		d := f()
		withNativejshelpers(&d)

		scrIdleGC := d.Script.WithID("idleGC").SetInnerHTML(`
			let lastGC = Date.now();

			function runGCDuringIdlePeriods(deadline) {

				if (deadline.didTimeout || !deadline.timeRemaining()) {
					setTimeout(() => window.requestIdleCallback(runGCDuringIdlePeriods), 120000); // Schedule next idle callback in 2 minutes
					return;
				}
				
				let now = Date.now();
				if (now - lastGC >= 120000) { // Check if at least 2 minutes passed since last GC
					window.triggerGC(); // Trigger GC
					lastGC = now;
				}

				// Schedule a new callback for the next idle time, but not sooner than 2 minutes from now
				setTimeout(() => window.requestIdleCallback(runGCDuringIdlePeriods), 120000); // Schedule next idle callback in 2 minutes
			}

			// Start the loop
			window.requestIdleCallback(runGCDuringIdlePeriods);
	
		`)
		d.Head().AppendChild(scrIdleGC)

		d.AfterEvent("document-loaded", d, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			js.Global().Call("onWasmDone")
			return false
		}))

		// sse support if hmr is enabled
		if HMRMode != "false" {
			d.Head().AppendChild(d.Script.WithID("ssesupport").SetInnerHTML(SSEscript))
		}

		d.OnReady(ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			// let's recover the baseURL from the document
			baseURI := js.Global().Get("document").Get("baseURI").String()
			bpath, err := url.Parse(baseURI)
			if err != nil {
				panic(err)
			}
			BasePath = bpath.Path
			err = d.mutationRecorder().Replay()
			if err != nil {
				d.mutationRecorder().Clear()
				// Should reload the page
				log.Println(err)
				d.Window().Reload()
				return false
			}
			d.mutationRecorder().Capture()
			return false
		}).RunOnce())

		if !InBrowser() { // SSR Mode only
			err := CreateSitemap(d, filepath.Join(".", "sitemap.xml"))
			if err != nil {
				log.Print(err)
			}
		}

		d.ListenAndServe(ctx)
	}
}
