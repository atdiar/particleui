// Package term defines the default set of Element constructors, native interfaces,
// events and event handlers, and animation properties used to build Terminal UIs.
package term 

import(
	"context"
	"fmt"
	"log"
	"time"
	// crand "crypto/rand"
	"golang.org/x/exp/rand"

	"github.com/rivo/tview"
	"github.com/atdiar/particleui"
	"github.com/gdamore/tcell/v2"
)

func init(){
	ui.NativeEventBridge = nil // TODO
	ui.NativeDispatch = nil
}

var (
	// DOCTYPE holds the document doctype.
	DOCTYPE = "terminal"
	// Elements stores wasm-generated HTML ui.Element constructors.
	Elements = ui.NewElementStore("default", DOCTYPE)
	Screen tcell.Screen
	
	document *Document

)

// newIDgenerator returns a function used to create new IDs. It uses
// a Pseudo-Random Number Generator (PRNG) as it is desirable to generate deterministic sequences.
// Evidently, as users navigate the app differently and may create new Elements
func newIDgenerator(charlen int, seed uint64) func() string {
	source := rand.NewSource(seed)
	r := rand.New(source)
	return func() string {
		var charset = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
		l:= len(charset)
		b := make([]rune, charlen)
		for i := range b {
			b[i] = charset[r.Intn(l)]
		}
		return string(b)
	}
}

var newID = newIDgenerator(16, uint64(time.Now().UnixNano()))



// =================================================================================================

// Terminal UIs are flat structures.Each box is given a non-hierarchical position on screen.
// However, the library can stil be used as a wrapper to enable Parent-Child style data relationships
// and data reactivity.
// In effect, it provides a Document Object Model on top of raw rendered eklements instead of
// the copy of the DOM as would be the case for in-browser rendering.
// On change, the terminal view would be redrawn (maybe some optimizations can happen if needed)


// ApplicationElement is a type that represents a Terminal application
type ApplicationElement struct {
	Raw *ui.Element
}


func (w ApplicationElement) AsElement() *ui.Element {
	return w.Raw
}

func(w ApplicationElement) running() bool{
	var ok bool
	ui.DoSync(func() {
		_,ok= w.Raw.AsElement().Get("event","running")
	})
	return ok
}

func(w ApplicationElement) GetFocus() *ui.Element{
	fid,ok:= w.Raw.AsElement().Get("ui","focus")
	if !ok{
		return nil
	}
	tfid := fid.(ui.String)
	return document.GetElementById(string(tfid))
}

func(w ApplicationElement) NativeElement() *tview.Application{
	return w.AsElement().Native.(NativeElement).Value.(*tview.Application)
}

// Run calls for the terminal app startup after having triggered a "running" event on the application.
func(w ApplicationElement) Run(){
	w.NativeElement().Run()
	w.Raw.TriggerEvent("running")
}

// Stop stops the application, causing Run() to return. 
func(w ApplicationElement) Stop(){
	w.Raw.TriggerEvent("before-unactive")
	w.NativeElement().Stop()
}

// Suspend temporarily suspends the application by exiting terminal UI mode and invoking the provided 
// function "f". When "f" returns, terminal UI mode is entered again and the application resumes.
//
// A return value of true indicates that the application was suspended and "f" was called. If false 
// is returned, the application was already suspended, terminal UI mode was not exited, and "f" 
// was not called. 
func(w ApplicationElement) Suspend(f func())bool{
	w.Raw.TriggerEvent("before-unactive")
	return w.NativeElement().Suspend(f)
}

// Sync forces a full re-sync of the screen buffer with the actual screen during the next event cycle. 
// This is useful for when the terminal screen is corrupted so you may want to offer your users a 
// keyboard shortcut to refresh the screen. 
func(w ApplicationElement) Sync(){
	w.NativeElement().Sync()
}


var newApplication= Elements.NewConstructor("application", func(id string) *ui.Element {
	e := ui.NewElement(id, DOCTYPE)
	e.Set("event", "mounted", ui.Bool(true))
	e.Set("event", "mountable", ui.Bool(true))

	e.ElementStore = Elements
	e.Parent = e
	raw:= tview.NewApplication()
	e.Native = NewNativeElementWrapper(raw)

	Screen, err:= tcell.NewScreen()
	if err != nil{
		panic (err)
	}

	raw.SetScreen(Screen)
	
	raw.SetAfterDrawFunc(func(screen tcell.Screen) {
		afterdraw:= ui.NewEvent("afterdraw", false, false, e, e,screen, nil)
		defer func() {
			if r := recover(); r != nil {
				app:= raw
				t:= tview.NewModal()
				app.ResizeToFullScreen(t)

				t.SetText("An error occured in the application. \n"+ fmt.Sprint(r)).
				AddButtons([]string{"Quit"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string){
					app.Stop()
				})	
			}
		}()
		ui.DoSync(func() {
			e.DispatchEvent(afterdraw)
		})

	})

	raw.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		beforedraw:= ui.NewEvent("beforedraw", false, false, e, e,screen, nil)
		defer func() {
			if r := recover(); r != nil {
				app:= raw
				t:= tview.NewModal()
				app.ResizeToFullScreen(t)

				t.SetText("An error occured in the application. \n"+ fmt.Sprint(r)).
				AddButtons([]string{"Quit"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string){
					app.Stop()
				})	
			}
		}()

		var b bool 
		ui.DoSync(func() {
			b= e.DispatchEvent(beforedraw)
		})
		return b

	})

	e.Watch("ui","focus",e, ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		target:= GetDocument(evt.Origin()).GetElementById(evt.NewValue().(ui.String).String())
		if target == nil{
			return false
		}
		evt.Origin().Native.(NativeElement).Value.(*tview.Application).SetFocus(target.Native.(NativeElement).Value.(tview.Primitive))
		return false
	}))

	e.Watch("ui","mouseenabled",e, ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		evt.Origin().Native.(NativeElement).Value.(*tview.Application).EnableMouse(evt.NewValue().(ui.Bool).Bool())
		return false
	}))


	return e
})

func GetApplication(e *ui.Element) ApplicationElement{
	return GetDocument(e).Application()
}


type appModifier struct{}
func(m appModifier) AsApplicationElement(e *ui.Element) ApplicationElement{
	return ApplicationElement{e}
}


func(m appModifier) SetFocus(p *ui.Element) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.Set("ui","focus", ui.String(p.ID))
		return e
	}
}


func(m appModifier) EnableMouse(b bool) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		e.Set("ui","mouseenabled", ui.Bool(b))
		return e
	}
}


func application(options ...string) ApplicationElement {
	e:= newApplication("term-application", options...)
	return ApplicationElement{e}
}


// Constructor helpers
type  idEnabler [T any] interface{
	WithID(id string, options ...string) T
}

type constiface[T any] interface{
	~func() T
	idEnabler[T]
}

type gconstructor[T ui.AnyElement, U constiface[T]] func()T

func(c *gconstructor[T,U]) WithID(id string, options ...string) T{
	var u U
	e := u.WithID(id, options...)
	d:= c.owner()
	if d == nil{
		panic("constructor should have an owner")
	}
	ui.RegisterElement(d.AsElement(),e.AsElement())

	return e
}

func( c *gconstructor[T,U]) ownedBy(d *Document){
	id := fmt.Sprintf("%v", *c)
	constructorDocumentLinker[id] = d
	d.Element.OnDeleted(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		delete(constructorDocumentLinker,id)
		return false
	}))
}

func( c *gconstructor[T,U]) owner() *Document{
	return constructorDocumentLinker[fmt.Sprintf("%v", *c)]
}


// For ButtonElement: it has a dedicated Document linked constructor as it has an optional typ argument
type  idEnablerButton [T any] interface{
	WithID(id string, label string, options ...string) T
}

type buttonconstiface[T any] interface{
	~func(label string) T
	idEnablerButton[T]
}

type buttongconstructor[T ui.AnyElement, U buttonconstiface[T]] func(label string)T

func(c *buttongconstructor[T,U]) WithID(id string, label string,  options ...string) T{
	var u U
	e := u.WithID(id, label, options...)
	d:= c.owner()
	if d == nil{
		panic("constructor should have an owner")
	}
	ui.RegisterElement(d.AsElement(),e.AsElement())

	return e
}

func( c *buttongconstructor[T,U]) ownedBy(d *Document){
	id := fmt.Sprintf("%v", *c)
	constructorDocumentLinker[id] = d
}

func( c *buttongconstructor[T,U]) owner() *Document{
	return constructorDocumentLinker[fmt.Sprintf("%v", *c)]
}


// NativeElement defines a wrapper around a js.Value that implements the
// ui.NativeElementWrapper interface.
type NativeElement struct {
	Value any
}

func NewNativeElementWrapper(v any) NativeElement {
	return NativeElement{v}
}

// TODO implement these methods by switching on the type of the Native Element (Probably by drawing the leemnt on screen)
func (n NativeElement) AppendChild(child *ui.Element) {
	n.Value.(tview.Primitive).Draw(Screen)
	p:= child.Parent
	for _,c:= range p.Children.List{
		c.Native.(NativeElement).Value.(tview.Primitive).Draw(Screen)
	}
}

func (n NativeElement) PrependChild(child *ui.Element) {
	n.Value.(tview.Primitive).Draw(Screen)
	p:= child.Parent
	for _,c:= range p.Children.List{
		c.Native.(NativeElement).Value.(tview.Primitive).Draw(Screen)
	}
}

func (n NativeElement) InsertChild(child *ui.Element, index int) {
	n.Value.(tview.Primitive).Draw(Screen)
	p:= child.Parent
	for _,c:= range p.Children.List{
		c.Native.(NativeElement).Value.(tview.Primitive).Draw(Screen)
	}
}

func (n NativeElement) ReplaceChild(old *ui.Element, new *ui.Element) {
	n.Value.(tview.Primitive).Draw(Screen)
	p:= new.Parent
	for _,c:= range p.Children.List{
		c.Native.(NativeElement).Value.(tview.Primitive).Draw(Screen)
	}
}

func (n NativeElement) RemoveChild(child *ui.Element) {
	n.Value.(tview.Primitive).Draw(Screen)
	p:= child.Parent
	for _,c:= range p.Children.List{
		c.Native.(NativeElement).Value.(tview.Primitive).Draw(Screen)
	}
}

func (n NativeElement) SetChildren(children ...*ui.Element) {
	n.Value.(tview.Primitive).Draw(Screen)
	for _,c:= range children{
		c.Native.(NativeElement).Value.(tview.Primitive).Draw(Screen)
	}
}

// TODO implement file storage? json or csv ? zipped?

// LoadFromStorage will load an element properties.
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

// PutInStorage stores an element properties in storage
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
	}
	return e
}

// =================================================================================================

// Document i.e. the app...

// constructorDocumentLinker maps constructors id to the document they are created for.
// Since we do not have dependent types, it is used to  have access to the document within  
// WithID methods, for element registration purposes (functio types do not have ccessible settable state)
var constructorDocumentLinker = make(map[string]*Document)

type Document struct {
	*ui.Element

	// id generator with serializable state
	// used to generate unique ids for elements
	rng   *rand.Rand
	src *rand.PCGSource

	Box gconstructor[BoxElement, boxConstructor,]
	Button buttongconstructor[ButtonElement, buttonConstructor,]
	CheckBox gconstructor[CheckBoxElement, checkboxConstructor,]
	DropDown gconstructor[DropDownElement, dropdownConstructor,]
	Frame gconstructor[FrameElement, frameConstructor,]
	Form gconstructor[FormElement, formConstructor,]
	Flex gconstructor[FlexElement, flexConstructor,]
	Grid gconstructor[GridElement, gridConstructor,]
	Image gconstructor[ImageElement, imageConstructor,]
	InputField gconstructor[InputFieldElement, inputfieldConstructor,]
	List gconstructor[ListElement, listConstructor,]
	Modal gconstructor[ModalElement, modalConstructor,]
	Pages gconstructor[PagesElement, pagesConstructor,]
	TreeView gconstructor[TreeViewElement, treeviewConstructor,]
	TextView gconstructor[TextViewElement, textviewConstructor,]
	Table gconstructor[TableElement, tableConstructor,]
	TextArea gconstructor[TextAreaElement, textareaConstructor,]

}

func withStdConstructors(d Document)Document{
	// TODO add all constructors here
	d.Box = gconstructor[BoxElement, boxConstructor,](func() BoxElement{
		e := BoxElement{newBox(d.newID())}
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	d.Box.ownedBy(&d)

	d.Button = buttongconstructor[ButtonElement, buttonConstructor,](func(label string) ButtonElement{
		e := ButtonElement{newButton(d.newID(),label)}
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	d.Button.ownedBy(&d)

	d.CheckBox = gconstructor[CheckBoxElement, checkboxConstructor,](func() CheckBoxElement{
		e := CheckBoxElement{newCheckBox(d.newID())}
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	d.CheckBox.ownedBy(&d)

	d.DropDown = gconstructor[DropDownElement, dropdownConstructor,](func() DropDownElement{
		e := DropDownElement{newDropDown(d.newID())}
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	d.DropDown.ownedBy(&d)

	d.Frame = gconstructor[FrameElement, frameConstructor,](func() FrameElement{
		e := FrameElement{newFrame(d.newID())}
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	d.Frame.ownedBy(&d)

	d.Form = gconstructor[FormElement, formConstructor,](func() FormElement{
		e := FormElement{newForm(d.newID())}
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	d.Form.ownedBy(&d)

	d.Flex = gconstructor[FlexElement, flexConstructor,](func() FlexElement{
		e := FlexElement{newFlex(d.newID())}
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	d.Flex.ownedBy(&d)

	d.Grid = gconstructor[GridElement, gridConstructor,](func() GridElement{
		e := GridElement{newGrid(d.newID())}
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	d.Grid.ownedBy(&d)

	d.Image = gconstructor[ImageElement, imageConstructor,](func() ImageElement{
		e := ImageElement{newImage(d.newID())}
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	d.Image.ownedBy(&d)

	d.InputField = gconstructor[InputFieldElement, inputfieldConstructor,](func() InputFieldElement{
		e := InputFieldElement{newInputField(d.newID())}
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	d.InputField.ownedBy(&d)

	d.List = gconstructor[ListElement, listConstructor,](func() ListElement{
		e := ListElement{newList(d.newID())}
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	d.List.ownedBy(&d)

	d.Modal = gconstructor[ModalElement, modalConstructor,](func() ModalElement{
		e := ModalElement{newModal(d.newID())}
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	d.Modal.ownedBy(&d)

	d.Pages = gconstructor[PagesElement, pagesConstructor,](func() PagesElement{
		e := PagesElement{newPages(d.newID())}
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	d.Pages.ownedBy(&d)

	d.TreeView = gconstructor[TreeViewElement, treeviewConstructor,](func() TreeViewElement{
		e := TreeViewElement{newTreeView(d.newID())}
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	d.TreeView.ownedBy(&d)

	d.TextView = gconstructor[TextViewElement, textviewConstructor,](func() TextViewElement{
		e := TextViewElement{newTextView(d.newID())}
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	d.TextView.ownedBy(&d)

	d.Table = gconstructor[TableElement, tableConstructor,](func() TableElement{
		e := TableElement{newTable(d.newID())}
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	d.Table.ownedBy(&d)

	d.TextArea = gconstructor[TextAreaElement, textareaConstructor,](func() TextAreaElement{
		e := TextAreaElement{newTextArea(d.newID())}
		ui.RegisterElement(d.Element,e.AsElement())
		return e
	})
	d.TextArea.ownedBy(&d)

	return d

}

func (d Document)GetElementById(id string) *ui.Element{
	return ui.GetById(d.AsElement(),id)
}

func(d Document) newID() string{
	return newID() // DEBUG
}

func (d Document) Application() ApplicationElement {
	w:= d.GetElementById("term-application")
	if w != nil{
		return ApplicationElement{w}
	}
	app:= application()
	ui.RegisterElement(d.AsElement(),app.Raw)
	app.Raw.TriggerEvent("mounted", ui.Bool(true))
	app.Raw.TriggerEvent("mountable", ui.Bool(true))

	return app
}

// NewObservable returns a new ui.Observable element after registering it for the document.
// If the observable alreadys exiswted for this id, it is returns as is.
// it is up to the caller to check whether an element already exist for this id and possibly clear 
// its state beforehand.
func(d Document) NewObservable(id string, options ...string) ui.Observable{
	if e:=d.GetElementById(id); e != nil{
		return ui.Observable{e}
	}
	o:= d.AsElement().ElementStore.NewObservable(id,options...).AsElement()
	
	ui.RegisterElement(d.AsElement(),o)

	return ui.Observable{o}
}


func (d Document) OnNavigationEnd(h *ui.MutationHandler){
	d.AsElement().WatchEvent("navigation-end", d, h)
}

func(d Document) OnReady(h *ui.MutationHandler){
	d.AsElement().WatchEvent("document-ready",d,h)
}

func (d Document) isReady() bool{
	_, ok:= d.GetEventValue("document-ready")
	return ok
}

func(d Document) OnRouterMounted(h *ui.MutationHandler){
	d.AsElement().WatchEvent("router-mounted",d,h)
}

func(d Document) OnBeforeUnactive(h *ui.MutationHandler){
	d.AsElement().WatchEvent("before-unactive",d,h)
}

// Router returns the router associated with the document. It is nil if no router has been created.
func(d Document) Router() *ui.Router{
	return ui.GetRouter(d.AsElement())
}


func(d Document) Delete(){ // TODO check for dangling references
	ui.DoSync(func(){
		e:= d.AsElement()
		d.Router().NavCancel()
		ui.Delete(e)
	})
}

func(d Document) ListenAndServe(ctx context.Context){
	if d.Element ==nil{
		panic("document is missing")
	}

	a := d.Application()

	d.WatchEvent("running",a.Raw,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		d.TriggerEvent("document-ready")
		return false
	}).RunASAP())

	d.Router().ListenAndServe(ctx,"", a)
	a.Run()
	
}

func (d Document) NativeElement() tview.Primitive{
	return d.AsElement().Native.(NativeElement).Value.(tview.Primitive)
}




func GetDocument(e *ui.Element) Document{
	if document != nil{
		return *document
	}
	if e.Root == nil{
		panic("This element does not belong to any registered subtree of the Document. Root is nil. If root of a component, it should be declared as such by callling the NewComponent method of the document Element.")
	}
	return withStdConstructors(Document{Element:e.Root}) // TODO initialize document *Element constructors
}

var newDocument = Elements.NewConstructor("root", func(id string) *ui.Element {

	e := Elements.NewAppRoot(id).AsElement()

	root:= tview.NewBox()
	e.Native = NewNativeElementWrapper(root)

	w:= GetApplication(e)

	err := w.NativeElement().SetRoot(root, true)
	if err!= nil{
		panic(err)
	}
	
	return e
})

// NewDocument returns the root of a new terminal app. It is the top-most element
// in the tree of Elements that consitute the full document.
// Options such as the location of persisted data can be passed to the constructor of an instance.
func NewDocument(id string, options ...string) Document {
	d:= Document{Element:newDocument(id, options...)}
	
	d = withStdConstructors(d)

	document = &d

	return d
}


// BoxElement
type BoxElement struct{
	*ui.Element
}

func(e BoxElement) NativeElement() *tview.Box{
	return e.AsElement().Native.(NativeElement).Value.(*tview.Box)
}

func(e BoxElement) Blur(){
	GetApplication(document.AsElement()).QueueUpdateDraw(func() {
		e.NativeElement().Blur()
	})
}

func(e BoxElement) Focus(delegate func(p tview.Primitive)){
	GetApplication(document.AsElement()).QueueUpdateDraw(func() {
		e.NativeElement().Focus(delegate)
	})
}

func(e BoxElement) GetBakcgroundColor() tcell.Color{
	var c tcell.Color
	GetApplication(document.AsElement()).QueueUpdate(func() {
		c= e.NativeElement().GetBackgroundColor()
	})
	return c
}

func(e BoxElement) GetBorderAttributes() tcell.AttrMask{
	var c tcell.AttrMask
	GetApplication(document.AsElement()).QueueUpdate(func() {
		c= e.NativeElement().GetBorderAttributes()
	})
	return c
}

func(e BoxElement) GetBorderColor() tcell.Color{
	var c tcell.Color
	GetApplication(document.AsElement()).QueueUpdate(func() {
		c= e.NativeElement().GetBorderColor()
	})
	return c
}

func(e BoxElement) GetDrawFunc() (f func(screen tcell.Screen,x,y,width,height int)(int,int,int,int)){
	GetApplication(document.AsElement()).QueueUpdate(func() {
		f= e.NativeElement().GetDrawFunc()
	})
	return f
}

func (e BoxElement) GetInnerRect() (x0, y0, x1, y1 int) {
	GetApplication(document.AsElement()).QueueUpdate(func() {
		x0, y0, x1, y1 = e.NativeElement().GetInnerRect()
	})
	return x0, y0, x1, y1
}



// GetInputCapture returns the function that is called when the user presses a key.
func(e BoxElement) GetInputCapture() func(event *tcell.EventKey) *tcell.EventKey{
	var f func(event *tcell.EventKey) *tcell.EventKey
	GetApplication(document.AsElement()).QueueUpdate(func() {
		f= e.NativeElement().GetInputCapture()
	})
	return f
}

// GetMouseCapture returns the function that is called when the user presses a mouse button.
func(e BoxElement) GetMouseCapture() func(actiion tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse){
	var f func(actiion tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse)
	GetApplication(document.AsElement()).QueueUpdate(func() {
		f= e.NativeElement().GetMouseCapture()
	})
	return f
}

func (e BoxElement) GetRect() (x0, y0, x1, y1 int) {
	GetApplication(document.AsElement()).QueueUpdate(func() {
		x0, y0, x1, y1 = e.NativeElement().GetRect()
	})
	return x0, y0, x1, y1
}

func (e BoxElement) GetTitle() string {
	var t string
	GetApplication(document.AsElement()).QueueUpdate(func() {
		t = e.NativeElement().GetTitle()
	})
	return t
}

func (e BoxElement) HasFocus() bool {
	var t bool
	GetApplication(document.AsElement()).QueueUpdate(func() {
		t = e.NativeElement().HasFocus()
	})
	return t
}

func (e BoxElement) InRect(x,y int) bool {
	var t bool
	GetApplication(document.AsElement()).QueueUpdate(func() {
		t = e.NativeElement().InRect(x,y)
	})
	return t
}

func(e BoxElement) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	var f func(event *tcell.EventKey, setFocus func(p tview.Primitive))
	GetApplication(document.AsElement()).QueueUpdate(func() {
		f= e.NativeElement().InputHandler()
	})
	return f
}

func(e BoxElement) MouseHandler() (f func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consume bool, capture tview.Primitive)){
	GetApplication(document.AsElement()).QueueUpdate(func() {
		f= e.NativeElement().MouseHandler()
	})
	return f
}



type boxModifier struct{}
var BoxModifier boxModifier

func (b boxModifier) AsBoxElement(e *ui.Element) BoxElement{
	return BoxElement{e}
}

func(m boxModifier) SetBackgroundColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		GetApplication(document.AsElement()).QueueUpdateDraw(func() {
			m.AsBoxElement(e).NativeElement().SetBackgroundColor(color)
		})
		return e
	}
}

func(m boxModifier) SetBorder(border bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		GetApplication(document.AsElement()).QueueUpdate(func() {
			m.AsBoxElement(e).NativeElement().SetBorder(border)
		})
		return e
	}
}

func(m boxModifier) SetBorderColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		GetApplication(document.AsElement()).QueueUpdate(func() {
			m.AsBoxElement(e).NativeElement().SetBorderColor(color)
		})
		return e
	}
}

func(m boxModifier) SetBorderAttributes(attributes tcell.AttrMask) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		GetApplication(document.AsElement()).QueueUpdate(func() {
			m.AsBoxElement(e).NativeElement().SetBorderAttributes(attributes)
		})
		return e
	}
}

func(m boxModifier) SetTitle(title string) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		GetApplication(document.AsElement()).QueueUpdate(func() {
			m.AsBoxElement(e).NativeElement().SetTitle(title)
		})
		return e
	}
}

func(m boxModifier) SetTitleAlign(align int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		GetApplication(document.AsElement()).QueueUpdate(func() {
			m.AsBoxElement(e).NativeElement().SetTitleAlign(align)
		})
		return e
	}
}

func(m boxModifier) SetTitleColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		GetApplication(document.AsElement()).QueueUpdate(func() {
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


type boxConstructor func() BoxElement
func(c boxConstructor) WithID(id string, options ...string)BoxElement{
	return BoxElement{newBox(id, options...)}
}


// ButtonElement
type ButtonElement struct{
	*ui.Element
}

func(e ButtonElement) NativeElement() *tview.Button{
	return e.AsElement().Native.(NativeElement).Value.(*tview.Button)
}

func(e ButtonElement) UnderlyingBox() BoxElement{
	box:= document.GetElementById(e.AsElement().ID+"-box")
	if box!= nil{
		return BoxElement{box}
	}

	b:= document.Box.WithID(e.AsElement().ID+"-box")
	b.AsElement().Native = NewNativeElementWrapper(e.NativeElement().Box)
	return b
}


var newButton = Elements.NewConstructor("button",func(id string)*ui.Element{
	
	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewButton(""))

	// TODO think about calling Draw OnMounted

	return e
})

type buttonConstructor func(label string) ButtonElement
func(c buttonConstructor) WithID(id string, label string, options ...string)ButtonElement{
	b:= ButtonElement{newButton(id, options...)}
	b.NativeElement().SetLabel(label)
	return b
}

// FrameElement allows to render space around an element if provided, otherwise, just some space.
type FrameElement struct{
	*ui.Element
}

func(e FrameElement) NativeElement() *tview.Frame{
	return e.AsElement().Native.(NativeElement).Value.(*tview.Frame)
}

func(e FrameElement) UnderlyingBox() BoxElement{
	box:= document.GetElementById(e.AsElement().ID+"-box")
	if box!= nil{
		return BoxElement{box}
	}

	b:= document.Box.WithID(e.AsElement().ID+"-box")
	b.AsElement().Native = NewNativeElementWrapper(e.NativeElement().Box)
	return b
}

var newFrame = Elements.NewConstructor("frame",func(id string)*ui.Element{
	
	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewFrame(nil))

	// TODO think about calling Draw OnMounted

	return e
})

type frameConstructor func() FrameElement
func(c frameConstructor) WithID(id string, options ...string)FrameElement{
	return FrameElement{newFrame(id, options...)}
}

// GridElement
type GridElement struct{
	*ui.Element
}

func(e GridElement) NativeElement() *tview.Grid{
	return e.AsElement().Native.(NativeElement).Value.(*tview.Grid)
}

func(e GridElement) UnderlyingBox() BoxElement{
	box:= document.GetElementById(e.AsElement().ID+"-box")
	if box!= nil{
		return BoxElement{box}
	}

	b:= document.Box.WithID(e.AsElement().ID+"-box")
	b.AsElement().Native = NewNativeElementWrapper(e.NativeElement().Box)
	return b
}


var newGrid = Elements.NewConstructor("grid",func(id string)*ui.Element{
	
	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewGrid())

	// TODO think about calling Draw OnMounted

	return e
})


type gridConstructor func() GridElement
func(c gridConstructor) WithID(id string, options ...string)GridElement{
	e:= GridElement{newGrid(id, options...)}
	
	
	return e
}

// FlexElement
type FlexElement struct{
	*ui.Element
}

func(e FlexElement) NativeElement() *tview.Flex{
	return e.AsElement().Native.(NativeElement).Value.(*tview.Flex)
}

func(e FlexElement) UnderlyingBox() BoxElement{
	box:= document.GetElementById(e.AsElement().ID+"-box")
	if box!= nil{
		return BoxElement{box}
	}

	b:= document.Box.WithID(e.AsElement().ID+"-box")
	b.AsElement().Native = NewNativeElementWrapper(e.NativeElement().Box)
	return b
}

var newFlex = Elements.NewConstructor("flex",func(id string)*ui.Element{
	
	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewFlex())

	// TODO think about calling Draw OnMounted

	return e
})


type flexConstructor func() FlexElement
func(c flexConstructor) WithID(id string, options ...string)FlexElement{
	e:= FlexElement{newFlex(id, options...)}
	return e
}

// PagesElement
type PagesElement struct{
	*ui.Element
}

func(e PagesElement) NativeElement() *tview.Pages{
	return e.AsElement().Native.(NativeElement).Value.(*tview.Pages)
}

func(e PagesElement) UnderlyingBox() BoxElement{
	box:= document.GetElementById(e.AsElement().ID+"-box")
	if box!= nil{
		return BoxElement{box}
	}

	b:= document.Box.WithID(e.AsElement().ID+"-box")
	b.AsElement().Native = NewNativeElementWrapper(e.NativeElement().Box)
	return b
}

type pagesModifier struct{}
var PagesModifier pagesModifier

func(m pagesModifier) AsPagesElement(e *ui.Element) PagesElement{
	return PagesElement{e}
}

func(m pagesModifier) AddPage(name string, elements ...*ui.Element) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		b:= document.Box()
		page:= b.AsElement().SetChildren(elements...)
		ui.NewViewElement(e,ui.NewView(name,page))
		p:= PagesElement{e}
		p.NativeElement().AddPage(name,b.NativeElement(), true, false)

		ui.ViewElement{e}.OnActivated(name, ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			p:= PagesElement{evt.Origin()}.NativeElement()
			pname:= string(evt.NewValue().(ui.String))
			
			GetApplication(e).QueueUpdateDraw(func(){
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
		GetApplication(e).QueueUpdateDraw(func(){
			m.AsPagesElement(e).NativeElement().HidePage(name)
		})
		return e
	}
}

func(m pagesModifier) ShowPage(name string) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		GetApplication(e).QueueUpdateDraw(func(){
			m.AsPagesElement(e).NativeElement().ShowPage(name)
		})
		return e
	}
}

// SetRect is used to set the position and dimensions of an element.
// It has not effect if part of a layout (flex or grid)
func(m pagesModifier) SetRect(x,y,width,height int) func(*ui.Element)*ui.Element{
	return func(e *ui.Element)*ui.Element{
		GetApplication(e).QueueUpdateDraw(func(){
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


type pagesConstructor func() PagesElement
func(c pagesConstructor) WithID(id string, options ...string)PagesElement{
	e:= PagesElement{newPages(id, options...)}
	
	return e
}

// ModalElement
type ModalElement struct{
	*ui.Element
}

func(e ModalElement) NativeElement() *tview.Modal{
	return e.AsElement().Native.(NativeElement).Value.(*tview.Modal)
}

func(e ModalElement) UnderlyingBox() BoxElement{
	box:= document.GetElementById(e.AsElement().ID+"-box")
	if box!= nil{
		return BoxElement{box}
	}

	b:= document.Box.WithID(e.AsElement().ID+"-box")
	b.AsElement().Native = NewNativeElementWrapper(e.NativeElement().Box)
	return b
}


var newModal = Elements.NewConstructor("modal",func(id string)*ui.Element{
	
	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewModal())

	// TODO think about calling Draw OnMounted

	return e
})



type modalConstructor func() ModalElement
func(c modalConstructor) WithID(id string, options ...string)ModalElement{
	e:= ModalElement{newModal(id, options...)}
	
	
	return e
}

// FormElement
type FormElement struct{
	*ui.Element
}

func(e FormElement) NativeElement() *tview.Form{
	return e.AsElement().Native.(NativeElement).Value.(*tview.Form)
}

func(e FormElement) UnderlyingBox() BoxElement{
	box:= document.GetElementById(e.AsElement().ID+"-box")
	if box!= nil{
		return BoxElement{box}
	}

	b:= document.Box.WithID(e.AsElement().ID+"-box")
	b.AsElement().Native = NewNativeElementWrapper(e.NativeElement().Box)
	return b
}

var newForm = Elements.NewConstructor("form",func(id string)*ui.Element{
	
	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewForm())

	// TODO think about calling Draw OnMounted

	return e
})


type formConstructor func() FormElement
func(c formConstructor) WithID(id string, options ...string)FormElement{
	e:= FormElement{newForm(id, options...)}
	
	
	return e
}

// ImageElement
type ImageElement struct{
	*ui.Element
}

func(e ImageElement) NativeElement() *tview.Image{
	return e.AsElement().Native.(NativeElement).Value.(*tview.Image)
}

func(e ImageElement) UnderlyingBox() BoxElement{
	box:= document.GetElementById(e.AsElement().ID+"-box")
	if box!= nil{
		return BoxElement{box}
	}

	b:= document.Box.WithID(e.AsElement().ID+"-box")
	b.AsElement().Native = NewNativeElementWrapper(e.NativeElement().Box)
	return b
}

var newImage = Elements.NewConstructor("image",func(id string)*ui.Element{
	
	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewImage())

	// TODO think about calling Draw OnMounted

	return e
})


type imageConstructor func() ImageElement
func(c imageConstructor) WithID(id string, options ...string)ImageElement{
	e:= ImageElement{newImage(id, options...)}
	
	
	return e
}

var newCheckBox = Elements.NewConstructor("checkbox",func(id string)*ui.Element{
	
	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewCheckbox())

	// TODO think about calling Draw OnMounted

	return e
})

// CheckBoxElement
type CheckBoxElement struct{
	*ui.Element
}

func(e CheckBoxElement) NativeElement() *tview.Checkbox{
	return e.AsElement().Native.(NativeElement).Value.(*tview.Checkbox)
}

func(e CheckBoxElement) UnderlyingBox() BoxElement{
	box:= document.GetElementById(e.AsElement().ID+"-box")
	if box!= nil{
		return BoxElement{box}
	}

	b:= document.Box.WithID(e.AsElement().ID+"-box")
	b.AsElement().Native = NewNativeElementWrapper(e.NativeElement().Box)
	return b
}


var newCheckbox = Elements.NewConstructor("checkbox",func(id string)*ui.Element{
	
	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewCheckbox())

	// TODO think about calling Draw OnMounted

	return e
})


type checkboxConstructor func() CheckBoxElement
func(c checkboxConstructor) WithID(id string, options ...string)CheckBoxElement{
	e:= CheckBoxElement{newCheckbox(id, options...)}
	
	
	return e
}

// DropDownElement
type DropDownElement struct{
	*ui.Element
}

func(e DropDownElement) NativeElement() *tview.DropDown{
	return e.AsElement().Native.(NativeElement).Value.(*tview.DropDown)
}

func(e DropDownElement) UnderlyingBox() BoxElement{
	box:= document.GetElementById(e.AsElement().ID+"-box")
	if box!= nil{
		return BoxElement{box}
	}

	b:= document.Box.WithID(e.AsElement().ID+"-box")
	b.AsElement().Native = NewNativeElementWrapper(e.NativeElement().Box)
	return b
}


var newDropDown = Elements.NewConstructor("dropdown",func(id string)*ui.Element{
	
	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewDropDown())

	// TODO think about calling Draw OnMounted

	return e
})


type dropdownConstructor func() DropDownElement
func(c dropdownConstructor) WithID(id string, options ...string)DropDownElement{
	e:= DropDownElement{newDropDown(id, options...)}
	
	
	return e
}

// InputFieldElement
type InputFieldElement struct{
	*ui.Element
}

func(e InputFieldElement) NativeElement() *tview.InputField{
	return e.AsElement().Native.(NativeElement).Value.(*tview.InputField)
}

func(e InputFieldElement) UnderlyingBox() BoxElement{
	box:= document.GetElementById(e.AsElement().ID+"-box")
	if box!= nil{
		return BoxElement{box}
	}

	b:= document.Box.WithID(e.AsElement().ID+"-box")
	b.AsElement().Native = NewNativeElementWrapper(e.NativeElement().Box)
	return b
}


var newInputField = Elements.NewConstructor("inputfield",func(id string)*ui.Element{
	
	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewInputField())

	// TODO think about calling Draw OnMounted

	return e
})


type inputfieldConstructor func() InputFieldElement
func(c inputfieldConstructor) WithID(id string, options ...string)InputFieldElement{
	e:= InputFieldElement{newInputField(id, options...)}
	
	
	return e
}

// ListElement
type ListElement struct{
	*ui.Element
}

func(e ListElement) NativeElement() *tview.List{
	return e.AsElement().Native.(NativeElement).Value.(*tview.List)
}

func(e ListElement) UnderlyingBox() BoxElement{
	box:= document.GetElementById(e.AsElement().ID+"-box")
	if box!= nil{
		return BoxElement{box}
	}

	b:= document.Box.WithID(e.AsElement().ID+"-box")
	b.AsElement().Native = NewNativeElementWrapper(e.NativeElement().Box)
	return b
}


var newList = Elements.NewConstructor("list",func(id string)*ui.Element{
	
	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewList())

	// TODO think about calling Draw OnMounted

	return e
})


type listConstructor func() ListElement
func(c listConstructor) WithID(id string, options ...string)ListElement{
	e:= ListElement{newList(id, options...)}
	
	
	return e
}

// TreeViewElement
type TreeViewElement struct{
	*ui.Element
}

func(e TreeViewElement) NativeElement() *tview.TreeView{
	return e.AsElement().Native.(NativeElement).Value.(*tview.TreeView)
}

func(e TreeViewElement) UnderlyingBox() BoxElement{
	box:= document.GetElementById(e.AsElement().ID+"-box")
	if box!= nil{
		return BoxElement{box}
	}

	b:= document.Box.WithID(e.AsElement().ID+"-box")
	b.AsElement().Native = NewNativeElementWrapper(e.NativeElement().Box)
	return b
}


var newTreeView = Elements.NewConstructor("treeview",func(id string)*ui.Element{
	
	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewTreeView())

	// TODO think about calling Draw OnMounted

	return e
})


type treeviewConstructor func() TreeViewElement
func(c treeviewConstructor) WithID(id string, options ...string)TreeViewElement{
	e:= TreeViewElement{newTreeView(id, options...)}
	
	
	return e
}

// TableElement
type TableElement struct{
	*ui.Element
}

func(e TableElement) NativeElement() *tview.Table{
	return e.AsElement().Native.(NativeElement).Value.(*tview.Table)
}

func(e TableElement) UnderlyingBox() BoxElement{
	box:= document.GetElementById(e.AsElement().ID+"-box")
	if box!= nil{
		return BoxElement{box}
	}

	b:= document.Box.WithID(e.AsElement().ID+"-box")
	b.AsElement().Native = NewNativeElementWrapper(e.NativeElement().Box)
	return b
}


var newTable = Elements.NewConstructor("table",func(id string)*ui.Element{
	
	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewTable())

	// TODO think about calling Draw OnMounted

	return e
})


type tableConstructor func() TableElement
func(c tableConstructor) WithID(id string, options ...string)TableElement{
	e:= TableElement{newTable(id, options...)}
	
	
	return e
}

// TextAreaElement
type TextAreaElement struct{
	*ui.Element
}

func(e TextAreaElement) NativeElement() *tview.TextArea{
	return e.AsElement().Native.(NativeElement).Value.(*tview.TextArea)
}

func(e TextAreaElement) UnderlyingBox() BoxElement{
	box:= document.GetElementById(e.AsElement().ID+"-box")
	if box!= nil{
		return BoxElement{box}
	}

	b:= document.Box.WithID(e.AsElement().ID+"-box")
	b.AsElement().Native = NewNativeElementWrapper(e.NativeElement().Box)
	return b
}


var newTextArea = Elements.NewConstructor("textarea",func(id string)*ui.Element{
	
	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewTextArea())

	// TODO think about calling Draw OnMounted

	return e
})


type textareaConstructor func() TextAreaElement
func(c textareaConstructor) WithID(id string, options ...string)TextAreaElement{
	e:= TextAreaElement{newTextArea(id, options...)}
	
	
	return e
}

// TextViewElement
type TextViewElement struct{
	*ui.Element
}

func(e TextViewElement) NativeElement() *tview.TextView{
	return e.AsElement().Native.(NativeElement).Value.(*tview.TextView)
}

func(e TextViewElement) UnderlyingBox() BoxElement{
	box:= document.GetElementById(e.AsElement().ID+"-box")
	if box!= nil{
		return BoxElement{box}
	}

	b:= document.Box.WithID(e.AsElement().ID+"-box")
	b.AsElement().Native = NewNativeElementWrapper(e.NativeElement().Box)
	return b
}


var newTextView = Elements.NewConstructor("textview",func(id string)*ui.Element{
	
	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewTextView())

	// TODO think about calling Draw OnMounted

	return e
})


type textviewConstructor func() TextViewElement
func(c textviewConstructor) WithID(id string, options ...string)TextViewElement{
	e:= TextViewElement{newTextView(id, options...)}
	
	
	return e
}


