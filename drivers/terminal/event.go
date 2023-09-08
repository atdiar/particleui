package term

import (
	//"context"
	"fmt"
	//"log"
	"strings"


	"github.com/atdiar/particleui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TODO check whihc elements may bubble and whcih may be cancelable

// CreateKeyEvent returns a new KeyEvent.
func CreateKeyEvent(target *ui.Element, evt *tcell.EventKey) ui.Event {
	e:= ui.NewEvent("key",false,true,target,target,evt,nil)
	return e
	
}


// CreateMouaseEvent returns a new MouseEvent.
func CreateMouseEvent(target *ui.Element, evt *tcell.EventMouse) ui.Event {
	return ui.NewEvent("mouseevent",true,true,target,target,evt,nil)
}

func defaultGoEventTranslator(evt ui.Event) tcell.Event{
	switch k:=evt.Native().(type) {
	case *tcell.EventKey:
		return k
	case *tcell.EventMouse:
		return k
	default:
		panic("unknown terminal synthetic event")
	}
}

// NativeDispatch allows for the propagation of a JS event created in Go.
func NativeDispatch(evt ui.Event){
	nevt:= defaultGoEventTranslator(evt)
	GetApplication(evt.CurrentTarget()).NativeElement().QueueEvent(nevt)
}



var NativeEventBridge = func(NativeEventName string, listener *ui.Element, capture bool) {
	var apprecovery = func(){
		if r := recover(); r != nil {
			app:= GetApplication(listener)
			t:= tview.NewModal()
			app.NativeElement().ResizeToFullScreen(t)

			t.SetText("An error occured in the application. \n"+ fmt.Sprint(r)).
			AddButtons([]string{"Quit"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string){
				app.NativeElement().Stop()
			})
			app.Draw()	
		}
	}

	k:=listener.Native.(NativeElement).Value

	app:= GetApplication(listener)
	app.QueueUpdateDraw(func(){	
		// Switch on the event name since the callback signature will be different

		// input event
		if strings.EqualFold(NativeEventName,"input"){
			k:=k.(*tview.Box)
			oldcapture:= k.GetInputCapture()
			k.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
				defer apprecovery()
				var done bool
				rawevt := newInputEvent(event)
				evt := ui.NewEvent("input",false,true,listener,listener,rawevt,rawevt.Value())
				
				ui.DoSync(func(){
					done = listener.DispatchEvent(evt)
				})
				if done{
					return nil
				}
				if oldcapture != nil{
					return oldcapture(event)
				}
				return event
			})
		}

		// Blur event
		if strings.EqualFold(NativeEventName,"blur"){
			listener.Native.(NativeElement).Value.(*tview.Box).SetBlurFunc(func(){
				defer apprecovery()
				evt := ui.NewEvent("blur",false,true,listener,listener,nil,nil)
				
				ui.DoSync(func(){
					listener.DispatchEvent(evt)
				})
			})

			if listener.NativeEventUnlisteners.List == nil {
				listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
			}
			listener.NativeEventUnlisteners.Add(NativeEventName, func() {
				listener.Native.(NativeElement).Value.(*tview.Box).SetBlurFunc(nil)
			})
		}

		// Focus event
		if strings.EqualFold(NativeEventName,"focus"){
			listener.Native.(NativeElement).Value.(*tview.Box).SetFocusFunc(func(){
				defer apprecovery()
				evt := ui.NewEvent("focus",false,true,listener,listener,nil,nil)
				
				ui.DoSync(func(){
					listener.DispatchEvent(evt)
				})
			})
			if listener.NativeEventUnlisteners.List == nil {
				listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
			}
			listener.NativeEventUnlisteners.Add(NativeEventName, func() {
				listener.Native.(NativeElement).Value.(*tview.Box).SetFocusFunc(nil)
			})
			return
		}
		// Draw event
		if strings.EqualFold(NativeEventName,"draw"){
			listener.Native.(NativeElement).Value.(*tview.Box).SetDrawFunc(func(screen tcell.Screen, x,y,width,height int) (int,int,int,int){
				defer apprecovery()
				n:= NativeDrawEvent{screen,x,y,width,height}
				evt := ui.NewEvent("draw",false,true,listener,listener,n,nil)
				
				var ix,iy,iw,ih int
				ui.DoSync(func(){
					listener.DispatchEvent(evt)
					ix,iy,iw,ih = listener.Native.(NativeElement).Value.(*tview.Box).GetInnerRect()
				})
				return ix,iy,iw,ih
			})
			if listener.NativeEventUnlisteners.List == nil {
				listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
			}
			listener.NativeEventUnlisteners.Add(NativeEventName, func() {
				listener.Native.(NativeElement).Value.(*tview.Box).SetDrawFunc(nil)
			})
			return 
		}

		// Exit event
		if strings.EqualFold(NativeEventName,"exit"){
			switch k:=listener.Native.(NativeElement).Value; k.(type) {
			case *tview.Button:
				k.(*tview.Button).SetExitFunc(func(key tcell.Key){
					defer apprecovery()
					evt := ui.NewEvent("exit",false,true,listener,listener,key,nil)
					
					ui.DoSync(func(){
						listener.DispatchEvent(evt)
					})
				})

				if listener.NativeEventUnlisteners.List == nil {
					listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
				}
				listener.NativeEventUnlisteners.Add(NativeEventName, func() {
					k.(*tview.Button).SetExitFunc(nil)
				})
			}
		}

			// Selected event
		if strings.EqualFold(NativeEventName,"selected"){
			switch k:= k.(type) {
			case *tview.Button:
				k.SetSelectedFunc(func(){
					defer apprecovery()
					evt := ui.NewEvent("selected",false,true,listener,listener,nil,nil)
					
					ui.DoSync(func(){
						listener.DispatchEvent(evt)
					})
					 
				})

				if listener.NativeEventUnlisteners.List == nil {
					listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
				}
				listener.NativeEventUnlisteners.Add(NativeEventName, func() {
					k.SetSelectedFunc(nil)
				})
			case *tview.DropDown:
				k.SetSelectedFunc(func(text string, index int){
					defer apprecovery()
					rawevt:=newDropDownSelectedEvent(text,index)
					evt := ui.NewEvent("selected",false,true,listener,listener,rawevt,rawevt.Value())
					
					ui.DoSync(func(){
						listener.DispatchEvent(evt)
					})
					
				})
				if listener.NativeEventUnlisteners.List == nil {
					listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
				}
				listener.NativeEventUnlisteners.Add(NativeEventName, func() {
					k.SetSelectedFunc(nil)
				}) 
			case *tview.List:
				k.SetSelectedFunc(func(index int, mainText string, secondaryText string, shorcut rune){
					defer apprecovery()
					rawevt:=newListSelectedEvent(index,mainText,secondaryText,shorcut)
					evt := ui.NewEvent("selected",false,true,listener,listener,rawevt,rawevt.Value())
					
					ui.DoSync(func(){
						listener.DispatchEvent(evt)
					})
					
				})
				if listener.NativeEventUnlisteners.List == nil {
					listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
				}
				listener.NativeEventUnlisteners.Add(NativeEventName, func() {
					k.SetSelectedFunc(nil)
				}) 
			case *tview.Table:
				k.SetSelectedFunc(func(row int, column int){
					defer apprecovery()
					rawevt:=newTableSelectedEvent(row,column)
					evt := ui.NewEvent("selected",false,true,listener,listener,rawevt,rawevt.Value())
					
					ui.DoSync(func(){
						listener.DispatchEvent(evt)
					})
					 
				})
				if listener.NativeEventUnlisteners.List == nil {
					listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
				}
				listener.NativeEventUnlisteners.Add(NativeEventName, func() {
					k.SetSelectedFunc(nil)
				})
			case *tview.TreeView:
				k.SetSelectedFunc(func(node *tview.TreeNode){
					defer apprecovery()
					rawevt:=newTreeViewSelectedEvent(node)
					evt := ui.NewEvent("selected",false,true,listener,listener,rawevt,nil)
					
					ui.DoSync(func(){
						listener.DispatchEvent(evt)
					})
					
				})
				if listener.NativeEventUnlisteners.List == nil {
					listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
				}
				listener.NativeEventUnlisteners.Add(NativeEventName, func() {
					k.SetSelectedFunc(nil)
				})
			case *tview.TreeNode:
				k.SetSelectedFunc(func(){
					defer apprecovery()
					evt := ui.NewEvent("selected",false,true,listener,listener,nil,nil)
					
					ui.DoSync(func(){
						listener.DispatchEvent(evt)
					})
					
				})
				if listener.NativeEventUnlisteners.List == nil {
					listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
				}
				listener.NativeEventUnlisteners.Add(NativeEventName, func() {
					k.SetSelectedFunc(nil)
				})
			}
		}

		// Done Event
		if strings.EqualFold(NativeEventName,"done") {
			switch k:= k.(type) {
			case *tview.InputField:
				k.SetDoneFunc(func(key tcell.Key){
					defer apprecovery()
					rawevt:=newDoneEvent(key,-1,"")
					evt := ui.NewEvent("done",false,true,listener,listener,rawevt,rawevt.Value())
					
					ui.DoSync(func(){
						listener.DispatchEvent(evt)
					})
					
				})
				if listener.NativeEventUnlisteners.List == nil {
					listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
				}
				listener.NativeEventUnlisteners.Add(NativeEventName, func() {
					k.SetDoneFunc(nil)
				})
			case *tview.List:
				k.SetDoneFunc(func(){
					defer apprecovery()
					rawevt:=newDoneEvent(tcell.KeyESC,-1,"")
					evt := ui.NewEvent("done",false,true,listener,listener,rawevt,nil)
					
					ui.DoSync(func(){
						listener.DispatchEvent(evt)
					})
					
				})
				if listener.NativeEventUnlisteners.List == nil {
					listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
				}
				listener.NativeEventUnlisteners.Add(NativeEventName, func() {
					k.SetDoneFunc(nil)
				})
			case *tview.Table:
				k.SetDoneFunc(func(key tcell.Key){
					defer apprecovery()
					rawevt:=newDoneEvent(key,-1,"")
					evt := ui.NewEvent("done",false,true,listener,listener,rawevt,rawevt.Value())
					
					ui.DoSync(func(){
						listener.DispatchEvent(evt)
					})
					
				})
				if listener.NativeEventUnlisteners.List == nil {
					listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
				}
				listener.NativeEventUnlisteners.Add(NativeEventName, func() {
					k.SetDoneFunc(nil)
				})
			case *tview.TreeView:
				k.SetDoneFunc(func(key tcell.Key){
					defer apprecovery()
					rawevt:=newDoneEvent(key,-1,"")
					evt := ui.NewEvent("done",false,true,listener,listener,rawevt,rawevt.Value())
					
					ui.DoSync(func(){
						listener.DispatchEvent(evt)
					})
					
				})
				if listener.NativeEventUnlisteners.List == nil {
					listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
				}
				listener.NativeEventUnlisteners.Add(NativeEventName, func() {
					k.SetDoneFunc(nil)
				})
			case *tview.DropDown:
				k.SetDoneFunc(func(key tcell.Key){
					defer apprecovery()
					rawevt:=newDoneEvent(key,-1,"")
					evt := ui.NewEvent("done",false,true,listener,listener,rawevt,rawevt.Value())
					
					ui.DoSync(func(){
						listener.DispatchEvent(evt)
					})
					
				})
				if listener.NativeEventUnlisteners.List == nil {
					listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
				}
				listener.NativeEventUnlisteners.Add(NativeEventName, func() {
					k.SetDoneFunc(nil)
				})
			case *tview.TextView:
				k.SetDoneFunc(func(key tcell.Key){
					defer apprecovery()
					rawevt:=newDoneEvent(key,-1,"")
					evt := ui.NewEvent("done",false,true,listener,listener,rawevt,rawevt.Value())
					
					ui.DoSync(func(){
						listener.DispatchEvent(evt)
					})
					
				})
				if listener.NativeEventUnlisteners.List == nil {
					listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
				}
				listener.NativeEventUnlisteners.Add(NativeEventName, func() {
					k.SetDoneFunc(nil)
				})
			case *tview.Modal:
				k.SetDoneFunc(func(buttonIndex int, buttonLabel string){
					defer apprecovery()
					rawevt:=newDoneEvent(0,buttonIndex,buttonLabel)
					if buttonIndex <0 && buttonLabel == ""{
						rawevt.Key = tcell.KeyEscape
					}
					evt := ui.NewEvent("done",false,true,listener,listener,rawevt,rawevt.Value())
					
					ui.DoSync(func(){
						listener.DispatchEvent(evt)
					})
					
				})
				if listener.NativeEventUnlisteners.List == nil {
					listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
				}
				listener.NativeEventUnlisteners.Add(NativeEventName, func() {
					k.SetDoneFunc(nil)
				})
			}
		}

		// Changed event
		if strings.EqualFold(NativeEventName,"changed"){
			switch k:=k.(type){
			case *tview.InputField:
				k.SetChangedFunc(func(text string){
					defer apprecovery()
					evt := ui.NewEvent("changed",false,true,listener,listener,text,ui.String(text))
					
					ui.DoSync(func(){
						listener.DispatchEvent(evt)
					})
					
				})
				if listener.NativeEventUnlisteners.List == nil {
					listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
				}
				listener.NativeEventUnlisteners.Add(NativeEventName, func() {
					k.SetChangedFunc(nil)
				})
			case *tview.TextView:
				k.SetChangedFunc(func(){
					defer apprecovery()
					evt := ui.NewEvent("changed",false,true,listener,listener,nil,nil)
					
					ui.DoSync(func(){
						listener.DispatchEvent(evt)
					})
					
				})
				if listener.NativeEventUnlisteners.List == nil {
					listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
				}
				listener.NativeEventUnlisteners.Add(NativeEventName, func() {
					k.SetChangedFunc(nil)
				})
			case *tview.Checkbox:
				k.SetChangedFunc(func(checked bool){
					defer apprecovery()
					evt := ui.NewEvent("changed",false,true,listener,listener,checked,ui.Bool(checked))
					
					ui.DoSync(func(){
						listener.DispatchEvent(evt)
					})
					
				})
				if listener.NativeEventUnlisteners.List == nil {
					listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
				}
				listener.NativeEventUnlisteners.Add(NativeEventName, func() {
					k.SetChangedFunc(nil)
				})
			case *tview.List:
				k.SetChangedFunc(func(index int,mainText,secondaryText string,shortcut rune){
					defer apprecovery()
					rawevt:=newListChangedEvent(index,mainText,secondaryText,shortcut)
					evt := ui.NewEvent("changed",false,true,listener,listener,rawevt,rawevt.Value())
					
					ui.DoSync(func(){
						listener.DispatchEvent(evt)
					})
					
				})
				if listener.NativeEventUnlisteners.List == nil {
					listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
				}
				listener.NativeEventUnlisteners.Add(NativeEventName, func() {
					k.SetChangedFunc(nil)
				})
			case *tview.Pages:
				k.SetChangedFunc(func(){
					defer apprecovery()
					evt := ui.NewEvent("changed",false,true,listener,listener,nil,nil)
					
					ui.DoSync(func(){
						listener.DispatchEvent(evt)
					})
					
				})
				if listener.NativeEventUnlisteners.List == nil {
					listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
				}
				listener.NativeEventUnlisteners.Add(NativeEventName, func() {
					k.SetChangedFunc(nil)
				})
			case *tview.TextArea:
				k.SetChangedFunc(func(){
					defer apprecovery()
					evt := ui.NewEvent("changed",false,true,listener,listener,nil,nil)
					
					ui.DoSync(func(){
						listener.DispatchEvent(evt)
					})
					
				})
				if listener.NativeEventUnlisteners.List == nil {
					listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
				}
				listener.NativeEventUnlisteners.Add(NativeEventName, func() {
					k.SetChangedFunc(nil)
				})

			}
		}

		// Autocompleted event
		if strings.EqualFold(NativeEventName,"autocompleted"){
			k := k.(*tview.InputField)
			k.SetAutocompletedFunc(func(text string, index int, source int) bool{
				defer apprecovery()
				rawevt:=newInputFieldAutocompletedEvent(text,index,source)
				evt := ui.NewEvent("autocompleted",false,true,listener,listener,rawevt,rawevt.Value())
				
				ui.DoSync(func(){
					listener.DispatchEvent(evt)
				})
				return true
			})
			if listener.NativeEventUnlisteners.List == nil {
				listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
			}
			listener.NativeEventUnlisteners.Add(NativeEventName, func() {
				k.SetAutocompletedFunc(nil)
			})
		}

		// SelectionChanged event
		if strings.EqualFold(NativeEventName,"selectionchanged"){
			k := k.(*tview.Table)
			k.SetSelectionChangedFunc(func(start, end int){
				defer apprecovery()
				rawevt:=newTableSelectionChangedEvent(start,end)
				evt := ui.NewEvent("selectionchanged",false,true,listener,listener,rawevt,rawevt.Value())
				
				ui.DoSync(func(){
					listener.DispatchEvent(evt)
				})
			})
			if listener.NativeEventUnlisteners.List == nil {
				listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
			}
			listener.NativeEventUnlisteners.Add(NativeEventName, func() {
				k.SetSelectionChangedFunc(nil)
			})
		}

		// Clicked event
		if strings.EqualFold(NativeEventName,"clicked"){
			k := k.(*tview.TableCell)
			k.SetClickedFunc(func() bool{
				defer apprecovery()
				evt := ui.NewEvent("clicked",false,true,listener,listener,nil,nil)
				
				ui.DoSync(func(){
					listener.DispatchEvent(evt)
				})
				return false
			})
			if listener.NativeEventUnlisteners.List == nil {
				listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
			}
			listener.NativeEventUnlisteners.Add(NativeEventName, func() {
				k.SetClickedFunc(nil)
			})
		}

		// Cancel event
		if strings.EqualFold(NativeEventName,"cancel"){
			if kk,ok:= k.(*tview.Form);ok{
				kk.SetCancelFunc(func(){
					defer apprecovery()
					evt := ui.NewEvent("cancel",false,true,listener,listener,nil,nil)
					
					ui.DoSync(func(){
						listener.DispatchEvent(evt)
					})
					
				})
				if listener.NativeEventUnlisteners.List == nil {
					listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
				}
				listener.NativeEventUnlisteners.Add(NativeEventName, func() {
					kk.SetCancelFunc(nil)
				}) 
			}
		}

		// Finished event
		if strings.EqualFold(NativeEventName,"finished"){
			switch k:= k.(type){
			case *tview.TextArea:
				k.SetFinishedFunc(func(key tcell.Key){
					defer apprecovery()
					rawevt:=newFinishedEvent(key)
					evt := ui.NewEvent("finished",false,true,listener,listener,rawevt,rawevt.Value())
					
					ui.DoSync(func(){
						listener.DispatchEvent(evt)
					})
					
				})
				if listener.NativeEventUnlisteners.List == nil {
					listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
				}
				listener.NativeEventUnlisteners.Add(NativeEventName, func() {
					k.SetFinishedFunc(nil)
				}) 
			case *tview.Checkbox:
				k.SetFinishedFunc(func(key tcell.Key){
					defer apprecovery()
					rawevt:=newFinishedEvent(key)
					evt := ui.NewEvent("finished",false,true,listener,listener,rawevt,rawevt.Value())
					
					ui.DoSync(func(){
						listener.DispatchEvent(evt)
					})
					
				})
				if listener.NativeEventUnlisteners.List == nil {
					listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
				}
				listener.NativeEventUnlisteners.Add(NativeEventName, func() {
					k.SetFinishedFunc(nil)
				})
			case *tview.DropDown:
				k.SetFinishedFunc(func(key tcell.Key){
					defer apprecovery()
					rawevt:=newFinishedEvent(key)
					evt := ui.NewEvent("finished",false,true,listener,listener,rawevt,rawevt.Value())
					
					ui.DoSync(func(){
						listener.DispatchEvent(evt)
					})
					
				})
				if listener.NativeEventUnlisteners.List == nil {
					listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
				}
				listener.NativeEventUnlisteners.Add(NativeEventName, func() {
					k.SetFinishedFunc(nil)
				})
			case *tview.Image:
				k.SetFinishedFunc(func(key tcell.Key){
					defer apprecovery()
					rawevt:=newFinishedEvent(key)
					evt := ui.NewEvent("finished",false,true,listener,listener,rawevt,rawevt.Value())
					
					ui.DoSync(func(){
						listener.DispatchEvent(evt)
					})
					
				})
				if listener.NativeEventUnlisteners.List == nil {
					listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
				}
				listener.NativeEventUnlisteners.Add(NativeEventName, func() {
					k.SetFinishedFunc(nil)
				})
			case *tview.InputField:
				k.SetFinishedFunc(func(key tcell.Key){
					defer apprecovery()
					rawevt:=newFinishedEvent(key)
					evt := ui.NewEvent("finished",false,true,listener,listener,rawevt,rawevt.Value())
					
					ui.DoSync(func(){
						listener.DispatchEvent(evt)
					})
					
				})
				if listener.NativeEventUnlisteners.List == nil {
					listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
				}
				listener.NativeEventUnlisteners.Add(NativeEventName, func() {
					k.SetFinishedFunc(nil)
				})
			case *tview.TextView:
				k.SetFinishedFunc(func(key tcell.Key){
					defer apprecovery()
					rawevt:=newFinishedEvent(key)
					evt := ui.NewEvent("finished",false,true,listener,listener,rawevt,rawevt.Value())
					
					ui.DoSync(func(){
						listener.DispatchEvent(evt)

					})
					
				})
				if listener.NativeEventUnlisteners.List == nil {
					listener.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
				}
				listener.NativeEventUnlisteners.Add(NativeEventName, func() {
					k.SetFinishedFunc(nil)
				})
			}
			
		}

	})
}


type InputEvent struct{
	Key *tcell.EventKey
}

func newInputEvent(key *tcell.EventKey) InputEvent{
	return InputEvent{key}
}

func(e InputEvent) Value() ui.Value{
	v:= ui.NewObject()
	v.Set("key",ui.String(e.Key.Name()))
	v.Set("rune",ui.String(string(e.Key.Rune())))
	v.Set("modifiers",ui.Number(e.Key.Modifiers()))
	v.Set("time",ui.String(e.Key.When().UTC().String()))
	return v.Commit()
}

type NativeDrawEvent struct {
	Screen tcell.Screen
	X int
	Y int
	Width int
	Height int
}

type DropDownSelectedEvent struct{
	Text string
	Index int
}

func newDropDownSelectedEvent(text string, index int) DropDownSelectedEvent{
	return DropDownSelectedEvent{text,index}
}

func(e DropDownSelectedEvent) Value() ui.Value{
	v:= ui.NewObject()
	v.Set("text",ui.String(e.Text))
	v.Set("index",ui.Number(e.Index))
	return v.Commit()
}

type ListSelectedEvent struct{
	Index int
	MainText string
	SecondaryText string
	Shortcut rune
}

func newListSelectedEvent(index int, mainText string, secondaryText string, shorcut rune) ListSelectedEvent{
	return ListSelectedEvent{index,mainText,secondaryText,shorcut}
}

func(e ListSelectedEvent) Value() ui.Value{
	v:= ui.NewObject()
	v.Set("index",ui.Number(e.Index))
	v.Set("mainText",ui.String(e.MainText))
	v.Set("secondaryText",ui.String(e.SecondaryText))
	v.Set("shortcut",ui.String(string(e.Shortcut)))
	return v.Commit()
}

type ListChangedEvent struct{
	Index int
	MainText string
	SecondaryText string
	Shortcut rune
}

func newListChangedEvent(index int, mainText string, secondaryText string, shorcut rune) ListChangedEvent{
	return ListChangedEvent{index,mainText,secondaryText,shorcut}
}

func(e ListChangedEvent) Value() ui.Value{
	v:= ui.NewObject()
	v.Set("index",ui.Number(e.Index))
	v.Set("mainText",ui.String(e.MainText))
	v.Set("secondaryText",ui.String(e.SecondaryText))
	v.Set("shortcut",ui.String(string(e.Shortcut)))
	return v.Commit()
}


type TreeViewSelectedEvent struct{
	Node *tview.TreeNode
}

func newTreeViewSelectedEvent(node *tview.TreeNode) TreeViewSelectedEvent{
	return TreeViewSelectedEvent{node}
}


type TableSelectedEvent struct{
	Row int
	Column int
}

func newTableSelectedEvent(row int, column int) TableSelectedEvent{
	return TableSelectedEvent{row,column}
}

func(e TableSelectedEvent) Value() ui.Value{
	v:= ui.NewObject()
	v.Set("row",ui.Number(e.Row))
	v.Set("column",ui.Number(e.Column))
	return v.Commit()
}

type DoneEvent struct{
	// For a list, this is the Escape key.
	Key tcell.Key

	// Modal only: The button index that was pressed unless Escape was pressed.
	ButtonIndex int
	ButtonLabel string
}

func newDoneEvent(key tcell.Key, buttonIndex int, buttonLabel string) DoneEvent{
	return DoneEvent{key,buttonIndex,buttonLabel}
}

func(e DoneEvent) Value() ui.Value{
	v:= ui.NewObject()
	v.Set("key",ui.Number(e.Key))
	v.Set("buttonIndex",ui.Number(e.ButtonIndex))
	v.Set("buttonLabel",ui.String(e.ButtonLabel))
	return v.Commit()
}

type FinishedEvent struct{
	Key tcell.Key
}

func newFinishedEvent(key tcell.Key) FinishedEvent{
	return FinishedEvent{key}
}

func(e FinishedEvent) Value() ui.Value{
	v:= ui.NewObject()
	v.Set("key",ui.Number(e.Key))
	return v.Commit()
}


type InputFieldAutocompletedEvent struct{
	Text string
	Index int
	Source int
}

func newInputFieldAutocompletedEvent(text string, index int, source int) InputFieldAutocompletedEvent{
	return InputFieldAutocompletedEvent{text,index,source}
}

func(e InputFieldAutocompletedEvent) Value() ui.Value{
	v:= ui.NewObject()
	v.Set("text",ui.String(e.Text))
	v.Set("index",ui.Number(e.Index))
	v.Set("source",ui.Number(e.Source))
	return v.Commit()
}


type TableSelectionChangedEvent struct{
	Row int
	Column int
}

func newTableSelectionChangedEvent(row int, column int) TableSelectionChangedEvent{
	return TableSelectionChangedEvent{row,column}
}

func(e TableSelectionChangedEvent) Value() ui.Value{
	v:= ui.NewObject()
	v.Set("row",ui.Number(e.Row))
	v.Set("column",ui.Number(e.Column))
	return v.Commit()
}