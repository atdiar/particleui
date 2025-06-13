package term

import (
	//"context"
	"fmt"
	//"log"
	//"strings"

	ui "github.com/atdiar/particleui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func withEventSupport[T ui.AnyElement](any T) T {
	e := any.AsElement()
	kk := e.Native.(NativeElement).Value
	switch kk := kk.(type) {
	case Application:
		k := kk.v
		// beforedraw event
		k.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
			beforedraw := ui.NewEvent("beforedraw", false, false, e, e, screen, nil)
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					k.ResizeToFullScreen(t)

					t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							k.Stop()
						})
				}
			}()

			var b bool
			ui.DoSync(e, func() {
				b = e.DispatchEvent(beforedraw)
			})
			return b
		})

		// afterdraw event
		k.SetAfterDrawFunc(func(screen tcell.Screen) {
			afterdraw := ui.NewEvent("afterdraw", false, false, e, e, screen, nil)
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					k.ResizeToFullScreen(t)

					t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							k.Stop()
						})
				}
			}()

			ui.DoSync(e, func() {
				e.DispatchEvent(afterdraw)
			})
		})

	case Box:
		k := kk.v
		// afterdraw event
		k.SetDrawFunc(func(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {

			o := ui.NewObject()
			o.Set("x", ui.Number(x))
			o.Set("y", ui.Number(y))
			o.Set("width", ui.Number(width))
			o.Set("height", ui.Number(height))

			afterdraw := ui.NewEvent("afterdraw", false, false, e, e, nil, o.Commit())
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					app := GetApplication(e).NativeElement()
					app.ResizeToFullScreen(t)

					t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							app.Stop()
						})
				}
			}()

			ui.DoSync(e, func() {
				e.DispatchEvent(afterdraw)
			})
			return k.GetInnerRect()
		})

		// focus event
		k.SetFocusFunc(func() {
			focus := ui.NewEvent("focus", false, false, e, e, nil, nil)
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					app := GetApplication(e).NativeElement()
					app.ResizeToFullScreen(t)

					t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							app.Stop()
						})
				}
			}()

			ui.DoSync(e, func() {
				e.DispatchEvent(focus)
			})
		})

		// blur event
		k.SetBlurFunc(func() {
			blur := ui.NewEvent("blur", false, false, e, e, nil, nil)
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					app := GetApplication(e).NativeElement()
					app.ResizeToFullScreen(t)

					t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							app.Stop()
						})
				}
			}()

			ui.DoSync(e, func() {
				e.DispatchEvent(blur)
			})
		})

	case Button:
		k := kk.v
		// exit event
		k.SetExitFunc(func(key tcell.Key) {

			o := ui.NewObject()
			o.Set("key", ui.Number(key))

			exit := ui.NewEvent("exit", false, false, e, e, key, o.Commit())
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					app := GetApplication(e).NativeElement()
					app.ResizeToFullScreen(t)

					t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							app.Stop()
						})
				}
			}()

			ui.DoSync(e, func() {
				e.DispatchEvent(exit)
			})
		})

		// selected event
		k.SetSelectedFunc(func() {
			selected := ui.NewEvent("selected", false, false, e, e, nil, nil)
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					app := GetApplication(e).NativeElement()
					app.ResizeToFullScreen(t)

					t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							app.Stop()
						})
				}
			}()

			ui.DoSync(e, func() {
				e.DispatchEvent(selected)
			})
		})

	case CheckBox:
		k := kk.v
		// changed event
		k.SetChangedFunc(func(checked bool) {
			changed := ui.NewEvent("changed", false, false, e, e, checked, ui.Bool(checked))
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					app := GetApplication(e).NativeElement()
					app.ResizeToFullScreen(t)

					t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							app.Stop()
						})
				}
			}()

			ui.DoSync(e, func() {
				e.DispatchEvent(changed)
			})
		})

		// Done event
		k.SetDoneFunc(func(key tcell.Key) {

			o := ui.NewObject()
			o.Set("key", ui.Number(key))

			done := ui.NewEvent("done", false, false, e, e, key, o.Commit())
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					app := GetApplication(e).NativeElement()
					app.ResizeToFullScreen(t)

					t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							app.Stop()
						})
				}
			}()

			ui.DoSync(e, func() {
				e.DispatchEvent(done)
			})
		})

		// Finished event
		k.SetFinishedFunc(func(key tcell.Key) {

			o := ui.NewObject()
			o.Set("key", ui.Number(key))

			finished := ui.NewEvent("finished", false, false, e, e, key, o.Commit())
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					app := GetApplication(e).NativeElement()
					app.ResizeToFullScreen(t)

					t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							app.Stop()
						})
				}
			}()

			ui.DoSync(e, func() {
				e.DispatchEvent(finished)
			})
		})

	case DropDown:
		k := kk.v
		// Done event
		k.SetDoneFunc(func(key tcell.Key) {

			o := ui.NewObject()
			o.Set("key", ui.Number(key))

			done := ui.NewEvent("done", false, false, e, e, key, o.Commit())
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					app := GetApplication(e).NativeElement()
					app.ResizeToFullScreen(t)

					t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							app.Stop()
						})
				}
			}()

			ui.DoSync(e, func() {
				e.DispatchEvent(done)
			})
		})

		// Finished event
		k.SetFinishedFunc(func(key tcell.Key) {

			o := ui.NewObject()
			o.Set("key", ui.Number(key))

			finished := ui.NewEvent("finished", false, false, e, e, key, o.Commit())
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					app := GetApplication(e).NativeElement()
					app.ResizeToFullScreen(t)

					t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							app.Stop()
						})
				}
			}()

			ui.DoSync(e, func() {
				e.DispatchEvent(finished)
			})
		})

		// selected event
		k.SetSelectedFunc(func(text string, index int) {

			o := ui.NewObject()
			o.Set("text", ui.String(text))
			o.Set("index", ui.Number(index))

			selected := ui.NewEvent("selected", false, false, e, e, struct {
				string
				int
			}{text, index}, o.Commit())
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					app := GetApplication(e).NativeElement()
					app.ResizeToFullScreen(t)

					t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							app.Stop()
						})
				}
			}()

			ui.DoSync(e, func() {
				e.DispatchEvent(selected)
			})
		})

	case Form:
		k := kk.v
		// cancel event
		k.SetCancelFunc(func() {

			cancel := ui.NewEvent("cancel", false, false, e, e, nil, nil)
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					app := GetApplication(e).NativeElement()
					app.ResizeToFullScreen(t)

					t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							app.Stop()
						})
				}
			}()

			ui.DoSync(e, func() {
				e.DispatchEvent(cancel)
			})
		})

	case Frame:
	case Grid:
	case Image:
		k := kk.v
		// Finished event
		k.SetFinishedFunc(func(key tcell.Key) {

			o := ui.NewObject()
			o.Set("key", ui.Number(key))

			finished := ui.NewEvent("finished", false, false, e, e, key, o.Commit())
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					app := GetApplication(e).NativeElement()
					app.ResizeToFullScreen(t)

					t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							app.Stop()
						})
				}
			}()

			ui.DoSync(e, func() {
				e.DispatchEvent(finished)
			})
		})
	case InputField:
		k := kk.v
		// acceptance event
		k.SetAcceptanceFunc(func(textToCheck string, lastChar rune) bool {
			o := ui.NewObject()
			o.Set("textToCheck", ui.String(textToCheck))
			o.Set("lastChar", ui.String(string(lastChar)))

			acceptance := ui.NewEvent("acceptance", false, false, e, e, struct {
				textToCheck string
				lastChar    rune
			}{textToCheck, lastChar}, o.Commit())
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					app := GetApplication(e).NativeElement()
					app.ResizeToFullScreen(t)

					t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(
							func(buttonIndex int, buttonLabel string) {
								app.Stop()
							})
				}
			}()

			var b bool
			ui.DoSync(e, func() {
				b = e.DispatchEvent(acceptance)
			})
			return b
		})

		// done event
		k.SetDoneFunc(func(key tcell.Key) {

			o := ui.NewObject()
			o.Set("key", ui.Number(key))

			done := ui.NewEvent("done", false, false, e, e, key, o.Commit())
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					app := GetApplication(e).NativeElement()
					app.ResizeToFullScreen(t)

					t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							app.Stop()
						})
				}
			}()

			ui.DoSync(e, func() {
				e.DispatchEvent(done)
			})
		})

		// finished event (form)
		k.SetFinishedFunc(func(key tcell.Key) {

			o := ui.NewObject()
			o.Set("key", ui.Number(key))

			finished := ui.NewEvent("finished", false, false, e, e, key, o.Commit())
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					app := GetApplication(e).NativeElement()
					app.ResizeToFullScreen(t)

					t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							app.Stop()
						})
				}
			}()

			ui.DoSync(e, func() {
				e.DispatchEvent(finished)
			})
		})

	case List:
		k := kk.v
		// changed event
		k.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
			changed := ui.NewEvent("changed", false, false, e, e, struct {
				index     int
				main      string
				secondary string
				shortcut  rune
			}{index, mainText, secondaryText, shortcut}, nil)
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					app := GetApplication(e).NativeElement()
					app.ResizeToFullScreen(t)

					t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(
							func(buttonIndex int, buttonLabel string) {
								app.Stop()
							})
				}
			}()

			ui.DoSync(e, func() {
				e.DispatchEvent(changed)
			})
		})

		// done event
		k.SetDoneFunc(func() {
			done := ui.NewEvent("done", false, false, e, e, nil, nil)
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					app := GetApplication(e).NativeElement()
					app.ResizeToFullScreen(t)

					t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							app.Stop()
						})
				}
			}()

			ui.DoSync(e, func() {
				e.DispatchEvent(done)
			})
		})

		// selected event
		k.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {

			o := ui.NewObject()
			o.Set("index", ui.Number(index))
			o.Set("mainText", ui.String(mainText))
			o.Set("secondaryText", ui.String(secondaryText))
			o.Set("shortcut", ui.String(string(shortcut)))

			selected := ui.NewEvent("selected", false, false, e, e, struct {
				index     int
				main      string
				secondary string
				shortcut  rune
			}{index, mainText, secondaryText, shortcut}, o.Commit())
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					app := GetApplication(e).NativeElement()
					app.ResizeToFullScreen(t)

					t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							app.Stop()
						})
				}

			}()

			ui.DoSync(e, func() {
				e.DispatchEvent(selected)
			})
		})

	case Modal:
		k := kk.v
		// done event
		k.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			o := ui.NewObject()
			o.Set("buttonIndex", ui.Number(buttonIndex))
			o.Set("buttonLabel", ui.String(buttonLabel))

			done := ui.NewEvent("done", false, false, e, e, struct {
				buttonIndex int
				buttonLabel string
			}{buttonIndex, buttonLabel}, o.Commit())
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					app := GetApplication(e).NativeElement()
					app.ResizeToFullScreen(t)

					t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							app.Stop()
						})
				}

			}()

			ui.DoSync(e, func() {
				e.DispatchEvent(done)
			})
		})

	case Pages:
		k := kk.v
		// changed event
		k.SetChangedFunc(func() {
			changed := ui.NewEvent("changed", false, false, e, e, nil, nil)
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					app := GetApplication(e).NativeElement()
					app.ResizeToFullScreen(t)

					t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							app.Stop()
						})
				}

			}()

			ui.DoSync(e, func() {
				e.DispatchEvent(changed)
			})
		})

	case Table:
		k := kk.v
		// Done event
		k.SetDoneFunc(func(key tcell.Key) {

			o := ui.NewObject()
			o.Set("key", ui.Number(key))

			done := ui.NewEvent("done", false, false, e, e, key, o.Commit())
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					app := GetApplication(e).NativeElement()
					app.ResizeToFullScreen(t)

					t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							app.Stop()
						})
				}
			}()

			ui.DoSync(e, func() {
				e.DispatchEvent(done)
			})
		})

		// selected event
		k.SetSelectedFunc(func(row int, column int) {

			o := ui.NewObject()
			o.Set("row", ui.Number(row))
			o.Set("column", ui.Number(column))

			selected := ui.NewEvent("selected", false, false, e, e, struct {
				row    int
				column int
			}{row, column}, o.Commit())
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					app := GetApplication(e).NativeElement()
					app.ResizeToFullScreen(t)

					t.SetText("An error occured. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							app.Stop()
						})
				}
			}()

			ui.DoSync(e, func() {
				e.DispatchEvent(selected)
			})
		})

		// selectinchanged event
		k.SetSelectionChangedFunc(func(row int, column int) {

			o := ui.NewObject()
			o.Set("row", ui.Number(row))
			o.Set("column", ui.Number(column))

			selectionchanged := ui.NewEvent("selected", false, false, e, e, struct {
				row    int
				column int
			}{row, column}, o.Commit())
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					app := GetApplication(e).NativeElement()
					app.ResizeToFullScreen(t)

					t.SetText("An error occured. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							app.Stop()
						})
				}
			}()

			ui.DoSync(e, func() {
				e.DispatchEvent(selectionchanged)
			})
		})

	case TextArea:
		k := kk.v
		// changed event
		k.SetChangedFunc(func() {
			changed := ui.NewEvent("changed", false, false, e, e, nil, nil)
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					app := GetApplication(e).NativeElement()
					app.ResizeToFullScreen(t)

					t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							app.Stop()
						})
				}

			}()

			ui.DoSync(e, func() {
				e.DispatchEvent(changed)
			})
		})

		// clipboard event

		k.SetClipboard(nil, nil)

		// finished event
		k.SetFinishedFunc(func(key tcell.Key) {

			o := ui.NewObject()
			o.Set("key", ui.Number(key))

			finished := ui.NewEvent("finished", false, false, e, e, key, o.Commit())
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					app := GetApplication(e).NativeElement()
					app.ResizeToFullScreen(t)

					t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							app.Stop()
						})
				}
			}()

			ui.DoSync(e, func() {
				e.DispatchEvent(finished)
			})
		})

		// moved event
		k.SetMovedFunc(func() {

			moved := ui.NewEvent("moved", false, false, e, e, nil, nil)
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					k := GetApplication(e).NativeElement()
					k.ResizeToFullScreen(t)

					t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(
							func(buttonIndex int, buttonLabel string) {
								k.Stop()
							})
				}
			}()

			ui.DoSync(e, func() {
				e.DispatchEvent(moved)
			})
		})

	case TextView:
		k := kk.v
		// changed event
		k.SetChangedFunc(func() {
			changed := ui.NewEvent("changed", false, false, e, e, nil, nil)
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					app := GetApplication(e).NativeElement()
					app.ResizeToFullScreen(t)

					t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							app.Stop()
						})
				}

			}()

			ui.DoSync(e, func() {
				e.DispatchEvent(changed)
			})
		})

		// done event
		k.SetDoneFunc(func(key tcell.Key) {

			o := ui.NewObject()
			o.Set("key", ui.Number(key))

			done := ui.NewEvent("done", false, false, e, e, key, o.Commit())
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					app := GetApplication(e).NativeElement()
					app.ResizeToFullScreen(t)

					t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							app.Stop()
						})
				}
			}()

			ui.DoSync(e, func() {
				e.DispatchEvent(done)
			})
		})

		// finished event (form)
		k.SetFinishedFunc(func(key tcell.Key) {

			o := ui.NewObject()
			o.Set("key", ui.Number(key))

			finished := ui.NewEvent("finished", false, false, e, e, key, o.Commit())
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					app := GetApplication(e).NativeElement()
					app.ResizeToFullScreen(t)

					t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							app.Stop()
						})
				}
			}()

			ui.DoSync(e, func() {
				e.DispatchEvent(finished)
			})
		})

	case TreeView:
		k := kk.v
		// changed event
		k.SetChangedFunc(func(node *tview.TreeNode) {
			changed := ui.NewEvent("changed", false, false, e, e, node, nil)
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					app := GetApplication(e).NativeElement()
					app.ResizeToFullScreen(t)

					t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							app.Stop()
						})
				}
			}()

			ui.DoSync(e, func() {
				e.DispatchEvent(changed)
			})
		})

		// Done event
		k.SetDoneFunc(func(key tcell.Key) {

			o := ui.NewObject()
			o.Set("key", ui.Number(key))

			done := ui.NewEvent("done", false, false, e, e, key, o.Commit())
			defer func() {
				if r := recover(); r != nil {
					t := tview.NewModal()
					app := GetApplication(e).NativeElement()
					app.ResizeToFullScreen(t)

					t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
						AddButtons([]string{"Quit"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							app.Stop()
						})
				}
			}()

			ui.DoSync(e, func() {
				e.DispatchEvent(done)
			})
		})
	}

	return any
}
