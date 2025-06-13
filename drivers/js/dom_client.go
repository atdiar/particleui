//go:build !server

// Package doc defines the default set of Element constructors, native interfaces,
// events and event handlers, and animation properties used to build js-based UIs.

package doc

import (
	"context"
	"log"
	"net/url"
	"path/filepath"
	"runtime"
	"runtime/debug"

	ui "github.com/atdiar/particleui"
	js "github.com/atdiar/particleui/drivers/js/compat"
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
func NewBuilder(f func() *Document, buildEnvModifiers ...func()) (ListenAndServe func(context.Context)) {
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
		withNativejshelpers(d)

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

		base := js.Global().Get("document").Call("getElementById", "base")
		if base.Truthy() {
			base.Set("href", BasePath)
		}
		// TODO implement reverse operation that targets a DOM element and creates its zui counterpart.
		// Usually we create a zui element and link/initialize its native counterpart.

		// sse support if hmr is enabled
		if HMRMode != "false" {
			d.Head().AppendChild(d.Script.WithID("ssesupport").SetInnerHTML(SSEscript))
		}

		d.OnTransitionStart("load", ui.OnMutation(func(evt ui.MutationEvent) bool {
			// let's recover the baseURL from the document
			buri := js.Global().Get("document").Get("baseURI")
			if buri.Truthy() {
				baseURI := buri.String()
				bpath, err := url.Parse(baseURI)
				if err != nil {
					panic(err)
				}
				BasePath = bpath.Path // TODO the router should be able to handle this and rewrite links DEBUG
			}
			// Otherwise we do nothing, the basepath uses the default value. This i si specific to this framework as
			// we set the basepath in the document head  .
			return false
		}).RunASAP())

		d.OnTransitionStart("replay", ui.OnMutation(func(evt ui.MutationEvent) bool {
			err := d.mutationRecorder().Replay()
			if err != nil {
				d.ErrorTransition("replay", ui.String(err.Error()))
				return true // DEBUG may want to return false, should check
			}
			d.EndTransition("replay")
			return false
		}))

		d.OnTransitionError("replay", ui.OnMutation(func(evt ui.MutationEvent) bool {
			d.mutationRecorder().Clear()
			// Should reload the page
			log.Println("replay error, we should reload: ", evt.NewValue())
			d.Window().Reload()
			return true // here true or false doesn't matter
		}))

		d.AfterTransition("load", ui.OnMutation(func(evt ui.MutationEvent) bool {
			js.Global().Call("onWasmDone")
			return false
		}))

		// Capture mutations
		d.AfterEvent("ui-ready", d, ui.OnMutation(func(evt ui.MutationEvent) bool {
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
