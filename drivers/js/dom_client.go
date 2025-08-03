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
		debug.SetMemoryLimit(int64(512 * 1024 * 1024))
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
				if (now - lastGC >= 12000) { // Check if at least 12 seconds passed since last GC
					window.triggerGC(); // Trigger GC
					lastGC = now;
				}

				// Schedule a new callback for the next idle time, but not sooner than 12 seconds from now
				setTimeout(() => window.requestIdleCallback(runGCDuringIdlePeriods), 12000); // Schedule next idle callback in 2 minutes
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

		// sse support if lr is enabled
		if LRMode != "false" {
			d.Head().AppendChild(d.Script.WithID("ssesupport").SetInnerHTML(SSEscript))

			// clearing mutationr ecorder idnexedDB backend storage via a button
			w, ok := JSValue(d.Window().AsElement())
			if !ok {
				return
			}
			w.Set("clearbuttonFn", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				d.mutationRecorder().Clear()
				return js.Null()
			}))

			d.Head().AppendChild(d.Script.WithID("iDBclearButton").SetInnerHTML(`
				// Inject CSS styles dynamically
				const styleElement = document.createElement('style');
				styleElement.textContent = ` + "`" + `
					/* Basic body font for consistency, if not already set by main CSS */

					/* Floating Button Styles */
					#floatingButton {
						position: fixed;
						top: 20px;
						right: 60px;
						background-color: white;
						color: 	rgb(57,255,20);
						padding: 10px 15px;
						border-radius: 50px;
						box-shadow: 0 4px 8px rgba(0, 0, 0, 0.2);
						cursor: grab;
						text-align: center;
						line-height: 1.2;
						user-select: none;
						transition: background-color 0.3s ease, box-shadow 0.3s ease;
						z-index: 1000;
						border: none;
						outline: none;
						display: flex;
						flex-direction: column;
						align-items: center;
						justify-content: center;
						width: 150px;
						height: 60px;
						font-family: 'Courier New', Courier, monospace;
					}

					#floatingButton:hover {
						background-color: #f0f0f0;
						box-shadow: 0 6px 12px rgba(0, 0, 0, 0.3);
					}

					#floatingButton:active {
						cursor: grabbing;
						background-color: #e0e0e0;
						box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
					}

					#floatingButton span {
						display: block;
						font-weight: bold;
						animation: blink 1s linear infinite alternate; /* Blinking animation */
					}

					#floatingButton span {
						display: block;
						font-weight: light;
					}

					#floatingButton span:first-child {
						font-size: 15px;
						margin-bottom: 3px;
					}

					#floatingButton span:last-child {
						font-size: 12px;
						color: #007bff;
						text-decoration: underline;
						cursor: pointer;
					}

					@keyframes blink {
						0% { opacity: 1; }
						50% { opacity: 0.5; } /* Slightly fade out */
						100% { opacity: 1; }
					}

					/* Message Box Styles */
					#messageBox {
						position: fixed;
						top: 50%;
						left: 50%;
						transform: translate(-50%, -50%);
						background-color: rgba(0, 0, 0, 0.8);
						color: white;
						padding: 20px 30px;
						border-radius: 8px;
						box-shadow: 0 5px 15px rgba(0, 0, 0, 0.3);
						z-index: 2000;
						display: none; /* Hidden by default */
						text-align: center;
						font-size: 10px;
					}
				` + "`" + `;
				document.head.appendChild(styleElement);

				// Create the floating button element
				const floatingButton = document.createElement('div');
				floatingButton.id = 'floatingButton';

				// Create the first line of text
				const line1 = document.createElement('span');
				line1.textContent = 'Mutation Capture is on!';
				floatingButton.appendChild(line1);

				// Create the second line of text (link-like)
				const line2 = document.createElement('span');
				line2.textContent = 'Click to clear mutation records';
				floatingButton.appendChild(line2);

				// Append the button to the body
				document.body.appendChild(floatingButton);

				let isDragging = false;
				let offsetX, offsetY;


				// Mouse down event to start dragging
				floatingButton.addEventListener('mousedown', (e) => {
					isDragging = true;
					// Calculate offset from mouse pointer to the button's top-left corner
					offsetX = e.clientX - floatingButton.getBoundingClientRect().left;
					offsetY = e.clientY - floatingButton.getBoundingClientRect().top;
					floatingButton.style.cursor = 'grabbing'; // Change cursor to grabbing
					// Prevent text selection while dragging
					e.preventDefault();
				});

				// Mouse move event to update button position
				document.addEventListener('mousemove', (e) => {
					if (!isDragging) return;

					// Calculate new position based on mouse coordinates and initial offset
					let newLeft = e.clientX - offsetX;
					let newTop = e.clientY - offsetY;

					// Keep button within viewport boundaries
					const maxX = window.innerWidth - floatingButton.offsetWidth;
					const maxY = window.innerHeight - floatingButton.offsetHeight;

					newLeft = Math.max(0, Math.min(newLeft, maxX));
					newTop = Math.max(0, Math.min(newTop, maxY));

					floatingButton.style.left = ` + "`" + `${newLeft}px` + "`" + `;
					floatingButton.style.top = ` + "`" + `${newTop}px` + "`" + `;
				});

				// Mouse up event to stop dragging
				document.addEventListener('mouseup', () => {
					isDragging = false;
					floatingButton.style.cursor = 'grab'; // Reset cursor
				});

				// Click event to clear IndexedDB
				floatingButton.addEventListener('click', async (e) => {
					// Prevent click event from firing if it was a drag operation
					// This is a common pattern to distinguish click from drag-and-release
					if (offsetX !== (e.clientX - floatingButton.getBoundingClientRect().left) ||
						offsetY !== (e.clientY - floatingButton.getBoundingClientRect().top)) {
						// If the mouse moved significantly, it was a drag, not a pure click
						return;
					}
					
					window.clearbuttonFn();

					// If it was a pure click, proceed to clear IndexedDB
					try {
						// Use the globally initialized instance directly
						await window.indexedDBSyncInstance.clear(); // Clear all data in the object store
						window.location.reload();
					} catch (error) {
						console.error('Failed to clear IndexedDB:', error);
					}	
				});
			`))

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
			// we set the basepath in the document head.
			return false
		}).RunASAP())

		d.OnTransitionStart("replay", ui.OnMutation(func(evt ui.MutationEvent) bool {
			err := d.mutationRecorder().Replay()
			if err != nil {
				DEBUG("replay error, sending to transition error handler... ", err)
				d.ErrorTransition("replay", ui.String(err.Error()))
				return true // DEBUG may want to return false, should check
			}
			d.EndTransition("replay")
			return false
		}))

		d.OnTransitionError("replay", ui.OnMutation(func(evt ui.MutationEvent) bool {
			DEBUG("replay transition error for the document: ", d.ID, " with error: ", evt.NewValue())
			d.mutationRecorder().Clear()
			// Should reload the page
			DEBUG("replay error, we should reload: ", evt.NewValue())
			// DEBUG
			d.Window().Reload()
			return true // here true or false doesn't matter
		}).RunASAP())

		d.AfterTransition("load", ui.OnMutation(func(evt ui.MutationEvent) bool {
			js.Global().Call("onWasmDone")
			return false
		}))

		// Capture mutations
		d.AfterEvent("ui-ready", d, ui.OnMutation(func(evt ui.MutationEvent) bool {
			lch := ui.NewLifecycleHandlers(evt.Origin())
			if !lch.MutationWillReplay() {
				d.mutationRecorder().Capture()
			} else {
				d.AfterEvent("mutation-replayed", d, ui.OnMutation(func(evt ui.MutationEvent) bool {
					d.mutationRecorder().Capture()
					return false
				}).RunASAP())
			}
			return false
		}).RunOnce())

		if !InBrowser() { // SSG Mode only
			err := CreateSitemap(d, filepath.Join(".", "sitemap.xml"))
			if err != nil {
				log.Print(err)
			}
		}

		d.ListenAndServe(ctx)
	}
}
