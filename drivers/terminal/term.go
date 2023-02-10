// Package term defines the default set of Element constructors, native interfaces,
// events and event handlers, and animation properties used to build Terminal UIs.
package term 

import(
	"context"
	"log"

	"github.com/rivo/tview"
	"github.com/atdiar/particleui"
	"github.com/gdamore/tcell/v2"
)

var (
	// DOCTYPE holds the document doctype.
	DOCTYPE = "terminal"
	// Elements stores wasm-generated HTML ui.Element constructors.
	Elements = ui.NewElementStore("default", DOCTYPE)
	



	// DocumentInitializer is a Document specific modifier that is called on creation of a 
	// new document. By assigning a new value to this global function, we can hook new behaviors
	// into a NewDocument call.
	// That can be useful to pass specific properties to a new document object that will specialize 
	// construction of the document.
	DocumentInitializer func(Document) Document = func(d Document) Document{return d}
)



// =================================================================================================

// Terminal UIs are flat structures.Each box is given a non-hierarchical position on screen.
// However, the library can stil be used as a wrapper to enable Parent-CHild style data relationships
// and data reactivity.
// In effect, it provides a Document Object Model on top of raw rendered eklements instead of
// the copy of the DOM as would be the case for in-browser rendering.
// On change, the terminal view would be redrawn (maybe some optimizations can happen if needed)


// ApplicationElement is a type that represents a Terminal application
type ApplicationElement struct {
	UIElement ui.BasicElement
}

func (w ApplicationElement) AsBasicElement() ui.BasicElement {
	return w.UIElement
}

func (w ApplicationElement) AsElement() *ui.Element {
	return w.UIElement.AsElement()
}

func(w ApplicationElement) running() bool{
	var ok bool
	ui.DoSync(func() {
		_,ok= w.UIElement.AsElement().Get("event","running")
	})
	return ok
}

func(w ApplicationElement) Draw(){
	w.NativeElement().Draw()
}

func(w ApplicationElement) GetAfterDrawFunc() func(screen tcell.Screen){
	var f func(tcell.Screen)
	w.QueueUpdate(func(){
		f = w.NativeElement().GetAfterDrawFunc()
	})
	return f
}

func(w ApplicationElement) GetBeforeDrawFunc() (f func(screen tcell.Screen) bool){
	w.QueueUpdate(func(){
		f = w.NativeElement().GetBeforeDrawFunc()
	})
	return f
}

func(w ApplicationElement) GetFocus() tview.Primitive{
	var p tview.Primitive
	w.QueueUpdate(func(){
		p = w.NativeElement().GetFocus()
	})
	return p
}

func(w ApplicationElement) GetInputCapture() (f func(*tcell.EventKey)*tcell.EventKey){
	w.QueueUpdate(func(){
		f = w.NativeElement().GetInputCapture()
	})
	return f
}

func(w ApplicationElement) GetMouseCapture() (f func(event *tcell.EventMouse, action tview.MouseAction) (*tcell.EventMouse, tview.MouseAction)){
	w.QueueUpdate(func(){
		f = w.NativeElement().GetMouseCapture()
	})
	return f
}

func(w ApplicationElement) ResizeToFullScreen(e ui.AnyElement){
	w.QueueUpdate(func(){
		w.NativeElement().ResizeToFullScreen(e.AsElement().Native.(NativeElement).Value.(tview.Primitive))
	})
}

func(w ApplicationElement) NativeElement() *tview.Application{
	return w.AsElement().Native.(NativeElement).Value.(*tview.Application)
}

// Run calls for the terminal app startup after having triggered a "running" event on the application.
func(w ApplicationElement) Run(){
	ui.DoSync(func() {
		w.UIElement.AsElement().TriggerEvent("running")
	})
	w.NativeElement().Run()
}

// Stop stops the application, causing Run() to return. 
func(w ApplicationElement) Stop(){
	w.QueueUpdate(func(){
		w.NativeElement().Stop()
	})
}

// Suspend temporarily suspends the application by exiting terminal UI mode and invoking the provided 
// function "f". When "f" returns, terminal UI mode is entered again and the application resumes.
//
// A return value of true indicates that the application was suspended and "f" was called. If false 
// is returned, the application was already suspended, terminal UI mode was not exited, and "f" 
// was not called. 
func(w ApplicationElement) Suspend(f func())bool{
	var b bool
	w.QueueUpdate(func(){
		b = w.NativeElement().Suspend(f)
	})
	return b
}

// Sync forces a full re-sync of the screen buffer with the actual screen during the next event cycle. 
// This is useful for when the terminal screen is corrupted so you may want to offer your users a 
// keyboard shortcut to refresh the screen. 
func(w ApplicationElement) Sync(){
	w.QueueUpdate(func(){
		w.NativeElement().Sync()
	})
}


var newApplication= Elements.NewConstructor("application", func(id string) *ui.Element {
	e := ui.NewElement(id, DOCTYPE)
	e.Set("event", "mounted", ui.Bool(true))
	e.Set("event", "mountable", ui.Bool(true))

	e.ElementStore = Elements
	e.Parent = e
	e.Native = NewNativeElementWrapper(tview.NewApplication())

	return e
})

type appModifier struct{}
func(m appModifier) AsApplicationElement(e *ui.Element) ApplicationElement{
	return ApplicationElement{ui.BasicElement{e}}
}

func(m appModifier) EnableMNouse(enable bool) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		GetApplication().QueueUpdate(func(){
			m.AsApplicationElement(e).NativeElement().EnableMouse(enable)
		})
		return e
	}
}

func(m appModifier) SetAfterDrawFunc(handler func(screen tcell.Screen)) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		GetApplication().QueueUpdate(func(){
			m.AsApplicationElement(e).NativeElement().SetAfterDrawFunc(handler)
		})
		return e
	}
}

func(m appModifier) SetBeforeDrawFunc(handler func(screen tcell.Screen)bool) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		GetApplication().QueueUpdate(func(){
			m.AsApplicationElement(e).NativeElement().SetBeforeDrawFunc(handler)
		})
		return e
	}
}

func(m appModifier) SetFocus(p *ui.Element) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		GetApplication().QueueUpdate(func(){
			m.AsApplicationElement(e).NativeElement().SetFocus(p.Native.(NativeElement).Value.(tview.Primitive))
		})
		return e
	}
}

func(m appModifier) SetInputCapture(capture func(event *tcell.EventKey)*tcell.EventKey) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		GetApplication().QueueUpdate(func(){
			m.AsApplicationElement(e).NativeElement().SetInputCapture(capture)
		})
		return e
	}
}


func(m appModifier) SetMouseCapture(capture func(event *tcell.EventMouse, actrion tview.MouseAction) (*tcell.EventMouse,tview.MouseAction)) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		GetApplication().QueueUpdate(func(){
			m.AsApplicationElement(e).NativeElement().SetMouseCapture(capture)
		})
		return e
	}
}

func(m appModifier) SetRoot(p *ui.Element, fullscreen bool) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		GetApplication().QueueUpdate(func(){
			m.AsApplicationElement(e).NativeElement().SetRoot(p.Native.(NativeElement).Value.(tview.Primitive), fullscreen)
		})
		return e
	}
}


func application(options ...string) ApplicationElement {
	e:= newApplication("term-application", options...)
	return ApplicationElement{ui.BasicElement{LoadFromStorage(e)}}
}

func GetApplication() ApplicationElement {
	w := Elements.GetByID("term-application")
	if w ==nil{
		return application()
	}
	
	return ApplicationElement{ui.BasicElement{w}}
}

// NativeElement defines a wrapper around a js.Value that implements the
// ui.NativeElementWrapper interface.
type NativeElement struct {
	Value any
}

func NewNativeElementWrapper(v any) NativeElement {
	return NativeElement{v}
}

// TODO implement these methods by switching on the type of the Native Element
func (n NativeElement) AppendChild(child *ui.Element) {}

func (n NativeElement) PrependChild(child *ui.Element) {}

func (n NativeElement) InsertChild(child *ui.Element, index int) {}

func (n NativeElement) ReplaceChild(old *ui.Element, new *ui.Element) {}

func (n NativeElement) RemoveChild(child *ui.Element) {}

func (n NativeElement) SetChildren(children ...*ui.Element) {}

// TODO implement file storage? json or csv ? zipped?

// LoadFromStorage will load an element properties.
// If the corresponding native DOM Element is marked for hydration, by the presence of a data-hydrate
// atribute, the props are loaded from this attribute instead.
// abstractjs
func LoadFromStorage(e *ui.Element) *ui.Element {
	
	lb,ok:=e.Get("event","storesynced")
	if ok{
		if isSynced:=lb.(ui.Bool); isSynced{
			return e
		}
		
	}
	pmode := ui.PersistenceMode(e)
	storage, ok := e.ElementStore.PersistentStorer[pmode]
	if ok {
		err := storage.Load(e)
		if err != nil {
			log.Print(err)
			return e
		}
		e.Set("event","storesynced",ui.Bool(true))
	}
	return e
}

// PutInStorage stores an element properties in storage (localstorage or sessionstorage).
func PutInStorage(e *ui.Element) *ui.Element{
	pmode := ui.PersistenceMode(e)
	storage,ok:= e.ElementStore.PersistentStorer[pmode]
	if !ok{
		return e
	}
	for cat,props:= range e.Properties.Categories{
		if cat != "event"{
			for prop,val:= range props.Local{
				storage.Store(e,cat,prop,val)
			}
		}		
	}
	e.Set("event","storesynced",ui.Bool(true))
	return e
}

// ClearFromStorage will clear an element properties from storage.
func ClearFromStorage(e *ui.Element) *ui.Element{
	pmode:=ui.PersistenceMode(e)
	storage,ok:= e.ElementStore.PersistentStorer[pmode]
	if ok{
		storage.Clear(e)
		// reset the categories index/list for the element
		idx,ok:= e.Get("index","categories")
		if ok{
			index:=idx.(ui.List)[:0]
			e.Set("index","categories",index)
		}
	}
	return e
}

// =================================================================================================


// NewBuilder registers a new document building function.
func NewBuilder(f func()Document)(ListenAndServe func(context.Context)){
	return func(ctx context.Context){
		document:= f()
		go func(){
			document.ListenAndServe(ctx) // launches the UI thread
		}()
		GetApplication().Run()
		
	}
}

// QueueUpdate should be used to safely access Native Element.
// It should be used to wrap function calls on Native Objects for instance. (such as *Box.Blur())
// It is needed because the UI tree is updated in its own goroutine/thread which is different
// from the main application thread.
// So the two threads have to communicate by passing UI mutating functions.
//
// Note: the dual is that native event callbacks  should wrap all their UI tree mutating functions in 
// a siungle ui.DoSync. This is automatically done when registering an event handle via 
// *ui.Element.AddEventListener for instance.
func (w ApplicationElement) QueueUpdate(f func()){
	if !w.running(){
		f()
		return
	}
	w.NativeElement().QueueUpdate(f)
}

// QueueUpdateDraw is the same as QueueuUpdate with the difference that it refreshes the screen.
// It might be the more sensible option depending on the granularity of the UI change.
func (w ApplicationElement) QueueUpdateDraw(f func()){
	if !w.running(){
		f()
		w.NativeElement().Draw()
		return
	}
	w.NativeElement().QueueUpdateDraw(f)
}

type Document struct {
	ui.BasicElement
}


func (d Document) OnNavigationEnd(h *ui.MutationHandler){
	d.AsElement().Watch("event","navigationend", d, h)
}

func(d Document) OnReady(h *ui.MutationHandler){
	d.AsElement().Watch("navigation","ready",d,h)
}

func(d Document) Delete(){ // TODO check for dangling references
	ui.DoSync(func(){
		e:= d.AsElement()
		ui.CancelNav()
		e.DeleteChildren()
		Elements.Delete(e.ID)
	})
}

func (d Document) NativeElement() tview.Primitive{
	return d.AsElement().Native.(NativeElement).Value.(tview.Primitive)
}






// ListenAndServe is used to start listening to state changes to the document (aka navigation)
// coming from the browser such as popstate.
// It needs to run at the end, after the UI tree has been built.
//
// By construction, this is a blocking function.
func(d Document) ListenAndServe(ctx context.Context){
	ui.GetRouter().ListenAndServe(ctx,"", GetApplication())
}

func GetDocument(e *ui.Element) *Document{
	if e.Root() == nil{
		return nil
	}
	return &Document{ui.BasicElement{e.Root()}}
}

var newDocument = Elements.NewConstructor("root", func(id string) *ui.Element {

	e := Elements.NewAppRoot(id).AsElement()

	root:= tview.NewBox()
	e.Native = NewNativeElementWrapper(root)

	w:= GetApplication()

	err := w.NativeElement().SetRoot(root, true)
	if err!= nil{
		panic(err)
	}


	// makes ViewElements focusable (focus management support)
	e.Watch("internals", "views",e.Global,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		l:= evt.NewValue().(ui.List)
		view:= l[len(l)-1].(*ui.Element)
		e.Watch("ui","activeview",view,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			e.SetDataSetUI("focus",view)
			return false
		}))
		return false
	}))

	
	return e
})

// NewDocument returns the root of a new terminal app. It is the top-most element
// in the tree of Elements that consitute the full document.
// Options such as the location of persisted data can be passed to the constructor of an instance.
func NewDocument(id string, options ...string) Document {
	d:= Document{ui.BasicElement{LoadFromStorage(newDocument(id, options...))}}
	d = DocumentInitializer(d)
	return d
}


// BoxElement
type BoxElement struct{
	ui.BasicElement
}

func(e BoxElement) NativeElement() *tview.Box{
	return e.AsElement().Native.(NativeElement).Value.(*tview.Box)
}

func(e BoxElement) Blur(){
	GetApplication().QueueUpdateDraw(func() {
		e.NativeElement().Blur()
	})
}

func(e BoxElement) Focus(delegate func(p tview.Primitive)){
	GetApplication().QueueUpdateDraw(func() {
		e.NativeElement().Focus(delegate)
	})
}

func(e BoxElement) GetBakcgroundColor() tcell.Color{
	var c tcell.Color
	GetApplication().QueueUpdate(func() {
		c= e.NativeElement().GetBackgroundColor()
	})
	return c
}

func(e BoxElement) GetBorderAttributes() tcell.AttrMask{
	var c tcell.AttrMask
	GetApplication().QueueUpdate(func() {
		c= e.NativeElement().GetBorderAttributes()
	})
	return c
}

func(e BoxElement) GetBorderColor() tcell.Color{
	var c tcell.Color
	GetApplication().QueueUpdate(func() {
		c= e.NativeElement().GetBorderColor()
	})
	return c
}

func(e BoxElement) GetDrawFunc() (f func(screen tcell.Screen,x,y,width,height int)(int,int,int,int)){
	GetApplication().QueueUpdate(func() {
		f= e.NativeElement().GetDrawFunc()
	})
	return f
}

func (e BoxElement) GetInnerRect() (x0, y0, x1, y1 int) {
	GetApplication().QueueUpdate(func() {
		x0, y0, x1, y1 = e.NativeElement().GetInnerRect()
	})
	return x0, y0, x1, y1
}



// GetInputCapture returns the function that is called when the user presses a key.
func(e BoxElement) GetInputCapture() func(event *tcell.EventKey) *tcell.EventKey{
	var f func(event *tcell.EventKey) *tcell.EventKey
	GetApplication().QueueUpdate(func() {
		f= e.NativeElement().GetInputCapture()
	})
	return f
}

// GetMouseCapture returns the function that is called when the user presses a mouse button.
func(e BoxElement) GetMouseCapture() func(actiion tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse){
	var f func(actiion tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse)
	GetApplication().QueueUpdate(func() {
		f= e.NativeElement().GetMouseCapture()
	})
	return f
}

func (e BoxElement) GetRect() (x0, y0, x1, y1 int) {
	GetApplication().QueueUpdate(func() {
		x0, y0, x1, y1 = e.NativeElement().GetRect()
	})
	return x0, y0, x1, y1
}

func (e BoxElement) GetTitle() string {
	var t string
	GetApplication().QueueUpdate(func() {
		t = e.NativeElement().GetTitle()
	})
	return t
}

func (e BoxElement) HasFocus() bool {
	var t bool
	GetApplication().QueueUpdate(func() {
		t = e.NativeElement().HasFocus()
	})
	return t
}

func (e BoxElement) InRect(x,y int) bool {
	var t bool
	GetApplication().QueueUpdate(func() {
		t = e.NativeElement().InRect(x,y)
	})
	return t
}

func(e BoxElement) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	var f func(event *tcell.EventKey, setFocus func(p tview.Primitive))
	GetApplication().QueueUpdate(func() {
		f= e.NativeElement().InputHandler()
	})
	return f
}

func(e BoxElement) MouseHandler() (f func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consume bool, capture tview.Primitive)){
	GetApplication().QueueUpdate(func() {
		f= e.NativeElement().MouseHandler()
	})
	return f
}



type boxModifier struct{}
var BoxModifier boxModifier

func (b boxModifier) AsBoxElement(e *ui.Element) BoxElement{
	return BoxElement{ui.BasicElement{e}}
}

func(m boxModifier) SetBackgroundColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		GetApplication().QueueUpdateDraw(func() {
			m.AsBoxElement(e).NativeElement().SetBackgroundColor(color)
		})
		return e
	}
}

func(m boxModifier) SetBorder(border bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		GetApplication().QueueUpdate(func() {
			m.AsBoxElement(e).NativeElement().SetBorder(border)
		})
		return e
	}
}

func(m boxModifier) SetBorderColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		GetApplication().QueueUpdate(func() {
			m.AsBoxElement(e).NativeElement().SetBorderColor(color)
		})
		return e
	}
}

func(m boxModifier) SetBorderAttributes(attributes tcell.AttrMask) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		GetApplication().QueueUpdate(func() {
			m.AsBoxElement(e).NativeElement().SetBorderAttributes(attributes)
		})
		return e
	}
}

func(m boxModifier) SetTitle(title string) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		GetApplication().QueueUpdate(func() {
			m.AsBoxElement(e).NativeElement().SetTitle(title)
		})
		return e
	}
}

func(m boxModifier) SetTitleAlign(align int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		GetApplication().QueueUpdate(func() {
			m.AsBoxElement(e).NativeElement().SetTitleAlign(align)
		})
		return e
	}
}

func(m boxModifier) SetTitleColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		GetApplication().QueueUpdate(func() {
			m.AsBoxElement(e).NativeElement().SetTitleColor(color)
		})
		return e
	}
}


var newBox = Elements.NewConstructor("box",func(id string)*ui.Element{
	
	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewBox())

	// TODO think about calling Draw OnMounted

	return e
})


var Box = boxConstructor(func (options ...string) BoxElement {
	return BoxElement{ui.BasicElement{LoadFromStorage(newBox(Elements.NewID(), options...))}}
})

type boxConstructor func(...string) BoxElement
func(c boxConstructor) WithID(id string, options ...string)BoxElement{
	e:= BoxElement{ui.BasicElement{LoadFromStorage(newBox(id, options...))}}
	n:= e.NativeElement()
	n.SetTitle(id)
	return e
}


// ButtonElement
type ButtonElement struct{
	ui.BasicElement
}

func(e ButtonElement) NativeElement() *tview.Button{
	return e.AsElement().Native.(NativeElement).Value.(*tview.Button)
}


var newButton = Elements.NewConstructor("button",func(id string)*ui.Element{
	
	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewButton(""))

	// TODO think about calling Draw OnMounted

	return e
})


var Button = func (label string, options ...string) ButtonElement {
	e:= newButton(Elements.NewID(), options...)
	e.Native.(NativeElement).Value.(*tview.Button).SetLabel(label)
	return ButtonElement{ui.BasicElement{LoadFromStorage(e)}}
}

type buttonConstructor func(label string, options ...string) ButtonElement
func(c buttonConstructor) WithID(id string) func(label string, options ...string)ButtonElement{
	return func(label string, options ...string) ButtonElement{
		e:= ButtonElement{ui.BasicElement{LoadFromStorage(newButton(id, options...))}}
		e.NativeElement().SetLabel(label)
		return e
	}	
}


// GridElement
type GridElement struct{
	ui.BasicElement
}

func(e GridElement) NativeElement() *tview.Grid{
	return e.AsElement().Native.(NativeElement).Value.(*tview.Grid)
}


var newGrid = Elements.NewConstructor("grid",func(id string)*ui.Element{
	
	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewGrid())

	// TODO think about calling Draw OnMounted

	return e
})


var Grid = gridConstructor(func (options ...string) GridElement {
	return GridElement{ui.BasicElement{LoadFromStorage(newGrid(Elements.NewID(), options...))}}
})

type gridConstructor func(...string) GridElement
func(c gridConstructor) WithID(id string, options ...string)GridElement{
	e:= GridElement{ui.BasicElement{LoadFromStorage(newGrid(id, options...))}}
	n:= e.NativeElement()
	n.SetTitle(id)
	return e
}

// FlexElement
type FlexElement struct{
	ui.BasicElement
}

func(e FlexElement) NativeElement() *tview.Flex{
	return e.AsElement().Native.(NativeElement).Value.(*tview.Flex)
}


var newFlex = Elements.NewConstructor("flex",func(id string)*ui.Element{
	
	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewFlex())

	// TODO think about calling Draw OnMounted

	return e
})


var Flex = flexConstructor(func (options ...string) FlexElement {
	return FlexElement{ui.BasicElement{LoadFromStorage(newFlex(Elements.NewID(), options...))}}
})

type flexConstructor func(...string) FlexElement
func(c flexConstructor) WithID(id string, options ...string)FlexElement{
	e:= FlexElement{ui.BasicElement{LoadFromStorage(newFlex(id, options...))}}
	n:= e.NativeElement()
	n.SetTitle(id)
	return e
}

// PagesElement
type PagesElement struct{
	ui.BasicElement
}

func(e PagesElement) NativeElement() *tview.Pages{
	return e.AsElement().Native.(NativeElement).Value.(*tview.Pages)
}

type pagesModifier struct{}
var PagesModifier pagesModifier

func(m pagesModifier) AsPagesElement(e *ui.Element) PagesElement{
	return PagesElement{ui.BasicElement{e}}
}

func(m pagesModifier) AddPage(name string, elements ...ui.AnyElement) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		b:= Box()
		page:= b.AsElement().SetChildren(elements...)
		ui.NewViewElement(e,ui.NewView(name,page))
		p:= PagesElement{ui.BasicElement{e}}
		p.NativeElement().AddPage(name,b.NativeElement(), true, false)

		ui.ViewElement{e}.OnActivated(name, ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			p:= PagesElement{ui.BasicElement{evt.Origin()}}.NativeElement()
			pname:= string(evt.NewValue().(ui.String))
			
			GetApplication().QueueUpdateDraw(func(){
				if p.HasPage(pname){
					p.SwitchToPage(pname)
				} else{
					if !p.HasPage("pagenotfound -- SYSERR"){
						p.AddAndSwitchToPage("pagenotfound -- SYSERR",tview.NewBox().SetBorder(true).SetTitle("Page Not Found -- SYSERR"),true)
					} else{
						p.SwitchToPage("pagenotfound -- SYSERR")
					}
				}
			})
			
			return false
		}))
		return e
	}
}

func(m pagesModifier) HidePage(name string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		GetApplication().QueueUpdateDraw(func(){
			m.AsPagesElement(e).NativeElement().HidePage(name)
		})
		return e
	}
}

func(m pagesModifier) ShowPage(name string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		GetApplication().QueueUpdateDraw(func(){
			m.AsPagesElement(e).NativeElement().ShowPage(name)
		})
		return e
	}
}

// SetRect is used to set the position and dimensions of an element.
// It has not effect if part of a layout (flex or grid)
func(m pagesModifier) SetRect(x,y,width,height int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		GetApplication().QueueUpdateDraw(func(){
			tview.Primitive(m.AsPagesElement(e).NativeElement()).SetRect(x,y,width,height)
		})
		return e
	}
}


var newPages = Elements.NewConstructor("pages",func(id string)*ui.Element{
	
	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewPages())

	// TODO think about calling Draw OnMounted

	return e
})


var Pages = pagesConstructor(func (options ...string) PagesElement {
	return PagesElement{ui.BasicElement{LoadFromStorage(newPages(Elements.NewID(), options...))}}
})

type pagesConstructor func(...string) PagesElement
func(c pagesConstructor) WithID(id string, options ...string)PagesElement{
	e:= PagesElement{ui.BasicElement{LoadFromStorage(newPages(id, options...))}}
	n:= e.NativeElement()
	n.SetTitle(id)
	return e
}

// ModalElement
type ModalElement struct{
	ui.BasicElement
}

func(e ModalElement) NativeElement() *tview.Modal{
	return e.AsElement().Native.(NativeElement).Value.(*tview.Modal)
}


var newModal = Elements.NewConstructor("modal",func(id string)*ui.Element{
	
	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewModal())

	// TODO think about calling Draw OnMounted

	return e
})


var Modal = modalConstructor(func (options ...string) ModalElement {
	return ModalElement{ui.BasicElement{LoadFromStorage(newModal(Elements.NewID(), options...))}}
})

type modalConstructor func(...string) ModalElement
func(c modalConstructor) WithID(id string, options ...string)ModalElement{
	e:= ModalElement{ui.BasicElement{LoadFromStorage(newModal(id, options...))}}
	n:= e.NativeElement()
	n.SetTitle(id)
	return e
}

// FormElement
type FormElement struct{
	ui.BasicElement
}

func(e FormElement) NativeElement() *tview.Form{
	return e.AsElement().Native.(NativeElement).Value.(*tview.Form)
}


var newForm = Elements.NewConstructor("form",func(id string)*ui.Element{
	
	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewForm())

	// TODO think about calling Draw OnMounted

	return e
})


var Form = formConstructor(func (options ...string) FormElement {
	return FormElement{ui.BasicElement{LoadFromStorage(newForm(Elements.NewID(), options...))}}
})

type formConstructor func(...string) FormElement
func(c formConstructor) WithID(id string, options ...string)FormElement{
	e:= FormElement{ui.BasicElement{LoadFromStorage(newForm(id, options...))}}
	n:= e.NativeElement()
	n.SetTitle(id)
	return e
}

// ImageElement
type ImageElement struct{
	ui.BasicElement
}

func(e ImageElement) NativeElement() *tview.Image{
	return e.AsElement().Native.(NativeElement).Value.(*tview.Image)
}


var newImage = Elements.NewConstructor("image",func(id string)*ui.Element{
	
	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewImage())

	// TODO think about calling Draw OnMounted

	return e
})


var Image = imageConstructor(func (options ...string) ImageElement {
	return ImageElement{ui.BasicElement{LoadFromStorage(newImage(Elements.NewID(), options...))}}
})

type imageConstructor func(...string) ImageElement
func(c imageConstructor) WithID(id string, options ...string)ImageElement{
	e:= ImageElement{ui.BasicElement{LoadFromStorage(newImage(id, options...))}}
	n:= e.NativeElement()
	n.SetTitle(id)
	return e
}

// CheckboxElement
type CheckboxElement struct{
	ui.BasicElement
}

func(e CheckboxElement) NativeElement() *tview.Checkbox{
	return e.AsElement().Native.(NativeElement).Value.(*tview.Checkbox)
}


var newCheckbox = Elements.NewConstructor("checkbox",func(id string)*ui.Element{
	
	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewCheckbox())

	// TODO think about calling Draw OnMounted

	return e
})


var Checkbox = checkboxConstructor(func (options ...string) CheckboxElement {
	return CheckboxElement{ui.BasicElement{LoadFromStorage(newCheckbox(Elements.NewID(), options...))}}
})

type checkboxConstructor func(...string) CheckboxElement
func(c checkboxConstructor) WithID(id string, options ...string)CheckboxElement{
	e:= CheckboxElement{ui.BasicElement{LoadFromStorage(newCheckbox(id, options...))}}
	n:= e.NativeElement()
	n.SetTitle(id)
	return e
}

// DropDownElement
type DropDownElement struct{
	ui.BasicElement
}

func(e DropDownElement) NativeElement() *tview.DropDown{
	return e.AsElement().Native.(NativeElement).Value.(*tview.DropDown)
}


var newDropDown = Elements.NewConstructor("dropdown",func(id string)*ui.Element{
	
	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewDropDown())

	// TODO think about calling Draw OnMounted

	return e
})


var DropDown = dropdownConstructor(func (options ...string) DropDownElement {
	return DropDownElement{ui.BasicElement{LoadFromStorage(newDropDown(Elements.NewID(), options...))}}
})

type dropdownConstructor func(...string) DropDownElement
func(c dropdownConstructor) WithID(id string, options ...string)DropDownElement{
	e:= DropDownElement{ui.BasicElement{LoadFromStorage(newDropDown(id, options...))}}
	n:= e.NativeElement()
	n.SetTitle(id)
	return e
}

// InputFieldElement
type InputFieldElement struct{
	ui.BasicElement
}

func(e InputFieldElement) NativeElement() *tview.InputField{
	return e.AsElement().Native.(NativeElement).Value.(*tview.InputField)
}


var newInputField = Elements.NewConstructor("inputfield",func(id string)*ui.Element{
	
	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewInputField())

	// TODO think about calling Draw OnMounted

	return e
})


var InputField = inputfieldConstructor(func (options ...string) InputFieldElement {
	return InputFieldElement{ui.BasicElement{LoadFromStorage(newInputField(Elements.NewID(), options...))}}
})

type inputfieldConstructor func(...string) InputFieldElement
func(c inputfieldConstructor) WithID(id string, options ...string)InputFieldElement{
	e:= InputFieldElement{ui.BasicElement{LoadFromStorage(newInputField(id, options...))}}
	n:= e.NativeElement()
	n.SetTitle(id)
	return e
}

// ListElement
type ListElement struct{
	ui.BasicElement
}

func(e ListElement) NativeElement() *tview.List{
	return e.AsElement().Native.(NativeElement).Value.(*tview.List)
}


var newList = Elements.NewConstructor("list",func(id string)*ui.Element{
	
	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewList())

	// TODO think about calling Draw OnMounted

	return e
})


var List = listConstructor(func (options ...string) ListElement {
	return ListElement{ui.BasicElement{LoadFromStorage(newList(Elements.NewID(), options...))}}
})

type listConstructor func(...string) ListElement
func(c listConstructor) WithID(id string, options ...string)ListElement{
	e:= ListElement{ui.BasicElement{LoadFromStorage(newList(id, options...))}}
	n:= e.NativeElement()
	n.SetTitle(id)
	return e
}

// TreeViewElement
type TreeViewElement struct{
	ui.BasicElement
}

func(e TreeViewElement) NativeElement() *tview.TreeView{
	return e.AsElement().Native.(NativeElement).Value.(*tview.TreeView)
}


var newTreeView = Elements.NewConstructor("treeview",func(id string)*ui.Element{
	
	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewTreeView())

	// TODO think about calling Draw OnMounted

	return e
})


var TreeView = treeviewConstructor(func (options ...string) TreeViewElement {
	return TreeViewElement{ui.BasicElement{LoadFromStorage(newTreeView(Elements.NewID(), options...))}}
})

type treeviewConstructor func(...string) TreeViewElement
func(c treeviewConstructor) WithID(id string, options ...string)TreeViewElement{
	e:= TreeViewElement{ui.BasicElement{LoadFromStorage(newTreeView(id, options...))}}
	n:= e.NativeElement()
	n.SetTitle(id)
	return e
}

// TableElement
type TableElement struct{
	ui.BasicElement
}

func(e TableElement) NativeElement() *tview.Table{
	return e.AsElement().Native.(NativeElement).Value.(*tview.Table)
}


var newTable = Elements.NewConstructor("table",func(id string)*ui.Element{
	
	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewTable())

	// TODO think about calling Draw OnMounted

	return e
})


var Table = tableConstructor(func (options ...string) TableElement {
	return TableElement{ui.BasicElement{LoadFromStorage(newTable(Elements.NewID(), options...))}}
})

type tableConstructor func(...string) TableElement
func(c tableConstructor) WithID(id string, options ...string)TableElement{
	e:= TableElement{ui.BasicElement{LoadFromStorage(newTable(id, options...))}}
	n:= e.NativeElement()
	n.SetTitle(id)
	return e
}

// TextAreaElement
type TextAreaElement struct{
	ui.BasicElement
}

func(e TextAreaElement) NativeElement() *tview.TextArea{
	return e.AsElement().Native.(NativeElement).Value.(*tview.TextArea)
}


var newTextArea = Elements.NewConstructor("textarea",func(id string)*ui.Element{
	
	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewTextArea())

	// TODO think about calling Draw OnMounted

	return e
})


var TextArea = textareaConstructor(func (options ...string) TextAreaElement {
	return TextAreaElement{ui.BasicElement{LoadFromStorage(newTextArea(Elements.NewID(), options...))}}
})

type textareaConstructor func(...string) TextAreaElement
func(c textareaConstructor) WithID(id string, options ...string)TextAreaElement{
	e:= TextAreaElement{ui.BasicElement{LoadFromStorage(newTextArea(id, options...))}}
	n:= e.NativeElement()
	n.SetTitle(id)
	return e
}

// TextViewElement
type TextViewElement struct{
	ui.BasicElement
}

func(e TextViewElement) NativeElement() *tview.TextView{
	return e.AsElement().Native.(NativeElement).Value.(*tview.TextView)
}


var newTextView = Elements.NewConstructor("textview",func(id string)*ui.Element{
	
	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewTextView())

	// TODO think about calling Draw OnMounted

	return e
})


var TextView = textviewConstructor(func (options ...string) TextViewElement {
	return TextViewElement{ui.BasicElement{LoadFromStorage(newTextView(Elements.NewID(), options...))}}
})

type textviewConstructor func(...string) TextViewElement
func(c textviewConstructor) WithID(id string, options ...string)TextViewElement{
	e:= TextViewElement{ui.BasicElement{LoadFromStorage(newTextView(id, options...))}}
	n:= e.NativeElement()
	n.SetTitle(id)
	return e
}

