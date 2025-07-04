//go:build terminal

// Package term defines the default set of Element constructors, native interfaces,
// events and event handlers, and animation properties used to build Terminal UIs.
package term

import (
	"context"
	"fmt"
	"image"
	"log"
	"os/exec"
	"strings"
	"time"

	// crand "crypto/rand"
	"golang.org/x/exp/rand"

	ui "github.com/atdiar/particleui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func init() {
	ui.NativeEventBridge = nil // TODO
	ui.NativeDispatch = nil
}

var (
	// DOCTYPE holds the document doctype.
	DOCTYPE = "terminal"
	// Elements stores wasm-generated HTML ui.Element constructors.
	Elements = ui.NewConfiguration("default", DOCTYPE).WithGlobalConstructorOption(allowdatapersistence).
			AddPersistenceMode("diskpersistence", load, store, clear)

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
		l := len(charset)
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

func (w ApplicationElement) running() bool {
	var ok bool
	ui.DoSync(func() {
		_, ok = w.Raw.AsElement().Get("event", "running")
	})
	return ok
}

func (w ApplicationElement) GetFocus() *ui.Element {
	fid, ok := w.Raw.AsElement().Get("ui", "focus")
	if !ok {
		return nil
	}
	tfid := fid.(ui.String)
	return document.GetElementById(string(tfid))
}

func (w ApplicationElement) NativeElement() *tview.Application {
	return w.AsElement().Native.(NativeElement).Value.(Application).v
}

// Run calls for the terminal app startup after having triggered a "running" event on the application.
func (w ApplicationElement) Run() {
	w.NativeElement().Run()
	w.Raw.TriggerEvent("running")
}

// Stop stops the application, causing Run() to return.
func (w ApplicationElement) Stop() {
	w.Raw.TriggerEvent("before-unactive")
	w.NativeElement().Stop()
}

// Suspend temporarily suspends the application by exiting terminal UI mode and invoking the provided
// function "f". When "f" returns, terminal UI mode is entered again and the application resumes.
//
// A return value of true indicates that the application was suspended and "f" was called. If false
// is returned, the application was already suspended, terminal UI mode was not exited, and "f"
// was not called.
func (w ApplicationElement) Suspend(f func()) bool {
	w.Raw.TriggerEvent("before-unactive")
	return w.NativeElement().Suspend(f)
}

// Sync forces a full re-sync of the screen buffer with the actual screen during the next event cycle.
// This is useful for when the terminal screen is corrupted so you may want to offer your users a
// keyboard shortcut to refresh the screen.
func (w ApplicationElement) Sync() {
	w.NativeElement().Sync()
}

var newApplication = Elements.NewConstructor("application", func(id string) *ui.Element {
	e := ui.NewElement(id, DOCTYPE)
	e.Set("event", "mounted", ui.Bool(true))
	e.Set("event", "mountable", ui.Bool(true))

	e.Configuration = Elements
	e.Parent = e
	raw := tview.NewApplication()
	e.Native = NewNativeElementWrapper(raw)

	Screen, err := tcell.NewScreen()
	if err != nil {
		panic(err)
	}

	raw.SetScreen(Screen)

	raw.SetAfterDrawFunc(func(screen tcell.Screen) {
		afterdraw := ui.NewEvent("afterdraw", false, false, e, e, screen, nil)
		defer func() {
			if r := recover(); r != nil {
				app := raw
				t := tview.NewModal()
				app.ResizeToFullScreen(t)

				t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
					AddButtons([]string{"Quit"}).
					SetDoneFunc(func(buttonIndex int, buttonLabel string) {
						app.Stop()
					})
			}
		}()
		ui.DoSync(func() {
			e.DispatchEvent(afterdraw)
		})

	})

	raw.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		beforedraw := ui.NewEvent("beforedraw", false, false, e, e, screen, nil)
		defer func() {
			if r := recover(); r != nil {
				app := raw
				t := tview.NewModal()
				app.ResizeToFullScreen(t)

				t.SetText("An error occured in the application. \n" + fmt.Sprint(r)).
					AddButtons([]string{"Quit"}).
					SetDoneFunc(func(buttonIndex int, buttonLabel string) {
						app.Stop()
					})
			}
		}()

		var b bool
		ui.DoSync(func() {
			b = e.DispatchEvent(beforedraw)
		})
		return b

	})

	return e
})

func GetApplication(e *ui.Element) ApplicationElement {
	return GetDocument(e).Application()
}

func application(options ...string) ApplicationElement {
	e := newApplication("term-application", options...)
	return ApplicationElement{e}
}

// Constructor helpers
type idEnabler[T any] interface {
	WithID(id string, options ...string) T
}

type constiface[T any] interface {
	~func() T
	idEnabler[T]
}

type gconstructor[T ui.AnyElement, U constiface[T]] func() T

func (c *gconstructor[T, U]) WithID(id string, options ...string) T {
	var u U
	e := u.WithID(id, options...)
	d := c.owner()
	if d == nil {
		panic("constructor should have an owner")
	}
	ui.RegisterElement(d.AsElement(), e.AsElement())

	return withEventSupport(e)
}

func (c *gconstructor[T, U]) ownedBy(d *Document) {
	id := fmt.Sprintf("%v", *c)
	constructorDocumentLinker[id] = d
	d.Element.OnDeleted(ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		delete(constructorDocumentLinker, id)
		return false
	}))
}

func (c *gconstructor[T, U]) owner() *Document {
	return constructorDocumentLinker[fmt.Sprintf("%v", *c)]
}

// For ButtonElement: it has a dedicated Document linked constructor as it has an optional typ argument
type idEnablerButton[T any] interface {
	WithID(id string, label string, options ...string) T
}

type buttonconstiface[T any] interface {
	~func(label string) T
	idEnablerButton[T]
}

type buttongconstructor[T ui.AnyElement, U buttonconstiface[T]] func(label string) T

func (c *buttongconstructor[T, U]) WithID(id string, label string, options ...string) T {
	var u U
	e := u.WithID(id, label, options...)
	d := c.owner()
	if d == nil {
		panic("constructor should have an owner")
	}
	ui.RegisterElement(d.AsElement(), e.AsElement())

	return withEventSupport(e)
}

func (c *buttongconstructor[T, U]) ownedBy(d *Document) {
	id := fmt.Sprintf("%v", *c)
	constructorDocumentLinker[id] = d
}

func (c *buttongconstructor[T, U]) owner() *Document {
	return constructorDocumentLinker[fmt.Sprintf("%v", *c)]
}

// Since all UI elements are derived from the same base type *Box,
// we can define a generic wrapper with a method that erturns that base element.
// It should be useful when needing to access the underlying box of an element.
type universalWrapper[T any] struct {
	v T
}

func (w universalWrapper[T]) Box() *tview.Box {
	switch v := any(w.v).(type) {
	case *tview.Application:
		return nil
	case *tview.Box:
		return v
	case *tview.Button:
		return v.Box
	case *tview.Checkbox:
		return v.Box
	case *tview.DropDown:
		return v.Box
	case *tview.Frame:
		return v.Box
	case *tview.Form:
		return v.Box
	case *tview.Flex:
		return v.Box
	case *tview.Grid:
		return v.Box
	case *tview.InputField:
		return v.Box
	case *tview.List:
		return v.Box
	case *tview.Modal:
		return v.Box
	case *tview.Pages:
		return v.Box
	case *tview.TreeView:
		return v.Box
	case *tview.TextView:
		return v.Box
	case *tview.Table:
		return v.Box
	case *tview.TableCell:
		return nil
	case *tview.Image:
		return v.Box
	case *tview.TextArea:
		return v.Box
	default:
		panic("not a tview.Box")
	}
}

type Application = universalWrapper[*tview.Application]
type Box = universalWrapper[*tview.Box]
type Button = universalWrapper[*tview.Button]
type CheckBox = universalWrapper[*tview.Checkbox]
type DropDown = universalWrapper[*tview.DropDown]
type Frame = universalWrapper[*tview.Frame]
type Form = universalWrapper[*tview.Form]
type Flex = universalWrapper[*tview.Flex]
type Grid = universalWrapper[*tview.Grid]
type Image = universalWrapper[*tview.Image]
type InputField = universalWrapper[*tview.InputField]
type List = universalWrapper[*tview.List]
type Modal = universalWrapper[*tview.Modal]
type Pages = universalWrapper[*tview.Pages]
type TreeView = universalWrapper[*tview.TreeView]
type TextView = universalWrapper[*tview.TextView]
type Table = universalWrapper[*tview.Table]
type TextArea = universalWrapper[*tview.TextArea]

type Boxable interface {
	Box() *tview.Box
}

// NativeElement defines a wrapper around a native element that implements the
// ui.NativeElementWrapper interface.
type NativeElement struct {
	Value Boxable
}

func NewNativeElementWrapper[T any](v T) NativeElement {
	return NativeElement{universalWrapper[T]{v}}
}

func (n NativeElement) AppendChild(child *ui.Element) {
	if f, ok := n.Value.(Flex); ok {
		props, ok := child.Get("ui", "flex")
		if !ok {
			f.v.AddItem(child.Native.(NativeElement).Value.(tview.Primitive), 0, 1, false)
			return
		}
		fixedsize := int(props.RawValue().MustGetNumber("fixedsize"))
		proportion := int(props.RawValue().MustGetNumber("proportion"))
		focus := bool(props.RawValue().MustGetBool("focus"))
		f.v.AddItem(child.Native.(NativeElement).Value.(tview.Primitive), fixedsize, proportion, focus)
		return
	}

	if f, ok := n.Value.(Grid); ok {
		props, ok := child.Get("ui", "grid")
		if !ok {
			f.v.AddItem(child.Native.(NativeElement).Value.(tview.Primitive), 0, 0, 0, 0, 0, 0, false)
			return
		}
		row := int(props.RawValue().MustGetNumber("row"))
		column := int(props.RawValue().MustGetNumber("column"))
		rowSpan := int(props.RawValue().MustGetNumber("rowSpan"))
		columnSpan := int(props.RawValue().MustGetNumber("columnSpan"))
		minGridHeight := int(props.RawValue().MustGetNumber("minGridHeight"))
		minGridWidth := int(props.RawValue().MustGetNumber("minGridWidth"))
		focus := bool(props.RawValue().MustGetBool("focus"))
		f.v.AddItem(child.Native.(NativeElement).Value.(tview.Primitive), row, column, rowSpan, columnSpan, minGridHeight, minGridWidth, focus)
		return
	}

	if f, ok := n.Value.(Form); ok {
		if b, ok := child.Native.(NativeElement).Value.(Button); ok {
			l := b.v.GetLabel()
			f.v.AddButton(l, nil)
		}

		if c, ok := child.Native.(NativeElement).Value.(CheckBox); ok {
			f.v.AddFormItem(c.v)
		}

		if d, ok := child.Native.(NativeElement).Value.(DropDown); ok {
			f.v.AddFormItem(d.v)
		}

		if i, ok := child.Native.(NativeElement).Value.(Image); ok {
			f.v.AddFormItem(i.v)
		}

		if i, ok := child.Native.(NativeElement).Value.(InputField); ok {
			f.v.AddFormItem(i.v)
		}

		if l, ok := child.Native.(NativeElement).Value.(TextArea); ok {
			f.v.AddFormItem(l.v)
		}

		if l, ok := child.Native.(NativeElement).Value.(TextView); ok {
			f.v.AddFormItem(l.v)
		}
	}

}

func (n NativeElement) PrependChild(child *ui.Element) {
	if f, ok := n.Value.(Flex); ok {
		f.v.Clear()
		for _, c := range child.Parent.Children.List {
			props, ok := c.Get("ui", "flex")
			if !ok {
				f.v.AddItem(c.Native.(NativeElement).Value.(tview.Primitive), 0, 1, false)
				continue
			}
			fixedsize := int(props.RawValue().MustGetNumber("fixedsize"))
			proportion := int(props.RawValue().MustGetNumber("proportion"))
			focus := bool(props.RawValue().MustGetBool("focus"))
			f.v.AddItem(c.Native.(NativeElement).Value.(tview.Primitive), fixedsize, proportion, focus)
		}
		return
	}

	if f, ok := n.Value.(Grid); ok {
		f.v.Clear()
		for _, c := range child.Parent.Children.List {
			props, ok := c.Get("ui", "grid")
			if !ok {
				f.v.AddItem(c.Native.(NativeElement).Value.(tview.Primitive), 0, 0, 0, 0, 0, 0, false)
				continue
			}
			row := int(props.RawValue().MustGetNumber("row"))
			column := int(props.RawValue().MustGetNumber("column"))
			rowSpan := int(props.RawValue().MustGetNumber("rowSpan"))
			columnSpan := int(props.RawValue().MustGetNumber("columnSpan"))
			minGridHeight := int(props.RawValue().MustGetNumber("minGridHeight"))
			minGridWidth := int(props.RawValue().MustGetNumber("minGridWidth"))
			focus := bool(props.RawValue().MustGetBool("focus"))
			f.v.AddItem(c.Native.(NativeElement).Value.(tview.Primitive), row, column, rowSpan, columnSpan, minGridHeight, minGridWidth, focus)
		}
		return
	}

	if f, ok := n.Value.(Form); ok {
		f.v.Clear(true)
		for _, c := range child.Parent.Children.List {
			if b, ok := c.Native.(NativeElement).Value.(Button); ok {
				l := b.v.GetLabel()
				f.v.AddButton(l, nil)
			}

			if c, ok := c.Native.(NativeElement).Value.(CheckBox); ok {
				f.v.AddFormItem(c.v)
			}

			if d, ok := c.Native.(NativeElement).Value.(DropDown); ok {
				f.v.AddFormItem(d.v)
			}

			if i, ok := c.Native.(NativeElement).Value.(Image); ok {
				f.v.AddFormItem(i.v)
			}

			if i, ok := c.Native.(NativeElement).Value.(InputField); ok {
				f.v.AddFormItem(i.v)
			}

			if l, ok := c.Native.(NativeElement).Value.(TextArea); ok {
				f.v.AddFormItem(l.v)
			}

			if l, ok := c.Native.(NativeElement).Value.(TextView); ok {
				f.v.AddFormItem(l.v)
			}
		}
		return
	}
}

func (n NativeElement) InsertChild(child *ui.Element, index int) {
	if f, ok := n.Value.(Flex); ok {
		f.v.Clear()
		for _, c := range child.Parent.Children.List {
			props, ok := c.Get("ui", "flex")
			if !ok {
				f.v.AddItem(c.Native.(NativeElement).Value.(tview.Primitive), 0, 1, false)
				continue
			}
			fixedsize := int(props.RawValue().MustGetNumber("fixedsize"))
			proportion := int(props.RawValue().MustGetNumber("proportion"))
			focus := bool(props.RawValue().MustGetBool("focus"))
			f.v.AddItem(c.Native.(NativeElement).Value.(tview.Primitive), fixedsize, proportion, focus)
		}
		return
	}

	if f, ok := n.Value.(Grid); ok {
		f.v.Clear()
		for _, c := range child.Parent.Children.List {
			props, ok := c.Get("ui", "grid")
			if !ok {
				f.v.AddItem(c.Native.(NativeElement).Value.(tview.Primitive), 0, 0, 0, 0, 0, 0, false)
				continue
			}
			row := int(props.RawValue().MustGetNumber("row"))
			column := int(props.RawValue().MustGetNumber("column"))
			rowSpan := int(props.RawValue().MustGetNumber("rowSpan"))
			columnSpan := int(props.RawValue().MustGetNumber("columnSpan"))
			minGridHeight := int(props.RawValue().MustGetNumber("minGridHeight"))
			minGridWidth := int(props.RawValue().MustGetNumber("minGridWidth"))
			focus := bool(props.RawValue().MustGetBool("focus"))
			f.v.AddItem(c.Native.(NativeElement).Value.(tview.Primitive), row, column, rowSpan, columnSpan, minGridHeight, minGridWidth, focus)
		}
		return
	}

	if f, ok := n.Value.(Form); ok {
		f.v.Clear(true)
		for _, c := range child.Parent.Children.List {
			if b, ok := c.Native.(NativeElement).Value.(Button); ok {
				l := b.v.GetLabel()
				f.v.AddButton(l, nil)
			}

			if c, ok := c.Native.(NativeElement).Value.(CheckBox); ok {
				f.v.AddFormItem(c.v)
			}

			if d, ok := c.Native.(NativeElement).Value.(DropDown); ok {
				f.v.AddFormItem(d.v)
			}

			if i, ok := c.Native.(NativeElement).Value.(Image); ok {
				f.v.AddFormItem(i.v)
			}

			if i, ok := c.Native.(NativeElement).Value.(InputField); ok {
				f.v.AddFormItem(i.v)
			}

			if l, ok := c.Native.(NativeElement).Value.(TextArea); ok {
				f.v.AddFormItem(l.v)
			}

			if l, ok := c.Native.(NativeElement).Value.(TextView); ok {
				f.v.AddFormItem(l.v)
			}
		}
		return
	}
}

func (n NativeElement) ReplaceChild(old *ui.Element, new *ui.Element) {
	if f, ok := n.Value.(Flex); ok {
		f.v.Clear()
		for _, c := range new.Parent.Children.List {
			props, ok := c.Get("ui", "flex")
			if !ok {
				f.v.AddItem(c.Native.(NativeElement).Value.(tview.Primitive), 0, 1, false)
				continue
			}
			fixedsize := int(props.RawValue().MustGetNumber("fixedsize"))
			proportion := int(props.RawValue().MustGetNumber("proportion"))
			focus := bool(props.RawValue().MustGetBool("focus"))
			f.v.AddItem(c.Native.(NativeElement).Value.(tview.Primitive), fixedsize, proportion, focus)
		}
		return
	}

	if f, ok := n.Value.(Grid); ok {
		f.v.Clear()
		for _, c := range new.Parent.Children.List {
			props, ok := c.Get("ui", "grid")
			if !ok {
				f.v.AddItem(c.Native.(NativeElement).Value.(tview.Primitive), 0, 0, 0, 0, 0, 0, false)
				continue
			}
			row := int(props.RawValue().MustGetNumber("row"))
			column := int(props.RawValue().MustGetNumber("column"))
			rowSpan := int(props.RawValue().MustGetNumber("rowSpan"))
			columnSpan := int(props.RawValue().MustGetNumber("columnSpan"))
			minGridHeight := int(props.RawValue().MustGetNumber("minGridHeight"))
			minGridWidth := int(props.RawValue().MustGetNumber("minGridWidth"))
			focus := bool(props.RawValue().MustGetBool("focus"))
			f.v.AddItem(c.Native.(NativeElement).Value.(tview.Primitive), row, column, rowSpan, columnSpan, minGridHeight, minGridWidth, focus)
		}
		return
	}

	if f, ok := n.Value.(Form); ok {
		f.v.Clear(true)
		for _, c := range new.Parent.Children.List {
			if b, ok := c.Native.(NativeElement).Value.(Button); ok {
				l := b.v.GetLabel()
				f.v.AddButton(l, nil)
			}

			if c, ok := c.Native.(NativeElement).Value.(CheckBox); ok {
				f.v.AddFormItem(c.v)
			}

			if d, ok := c.Native.(NativeElement).Value.(DropDown); ok {
				f.v.AddFormItem(d.v)
			}

			if i, ok := c.Native.(NativeElement).Value.(Image); ok {
				f.v.AddFormItem(i.v)
			}

			if i, ok := c.Native.(NativeElement).Value.(InputField); ok {
				f.v.AddFormItem(i.v)
			}

			if l, ok := c.Native.(NativeElement).Value.(TextArea); ok {
				f.v.AddFormItem(l.v)
			}

			if l, ok := c.Native.(NativeElement).Value.(TextView); ok {
				f.v.AddFormItem(l.v)
			}
		}
		return
	}
}

func (n NativeElement) RemoveChild(child *ui.Element) {
	if f, ok := n.Value.(Flex); ok {
		f.v.RemoveItem(child.Native.(NativeElement).Value.(tview.Primitive))
	}

	if f, ok := n.Value.(Grid); ok {
		f.v.RemoveItem(child.Native.(NativeElement).Value.(tview.Primitive))
	}
}

func (n NativeElement) SetChildren(children ...*ui.Element) {
	if f, ok := n.Value.(Flex); ok {
		f.v.Clear()
		for _, c := range children {
			props, ok := c.Get("ui", "flex")
			if !ok {
				f.v.AddItem(c.Native.(NativeElement).Value.(tview.Primitive), 0, 1, false)
				continue
			}
			fixedsize := int(props.RawValue().MustGetNumber("fixedsize"))
			proportion := int(props.RawValue().MustGetNumber("proportion"))
			focus := bool(props.RawValue().MustGetBool("focus"))
			f.v.AddItem(c.Native.(NativeElement).Value.(tview.Primitive), fixedsize, proportion, focus)
		}
		return
	}

	if f, ok := n.Value.(Grid); ok {
		f.v.Clear()
		for _, c := range children {
			props, ok := c.Get("ui", "grid")
			if !ok {
				f.v.AddItem(c.Native.(NativeElement).Value.(tview.Primitive), 0, 0, 0, 0, 0, 0, false)
				continue
			}
			row := int(props.RawValue().MustGetNumber("row"))
			column := int(props.RawValue().MustGetNumber("column"))
			rowSpan := int(props.RawValue().MustGetNumber("rowSpan"))
			columnSpan := int(props.RawValue().MustGetNumber("columnSpan"))
			minGridHeight := int(props.RawValue().MustGetNumber("minGridHeight"))
			minGridWidth := int(props.RawValue().MustGetNumber("minGridWidth"))
			focus := bool(props.RawValue().MustGetBool("focus"))
			f.v.AddItem(c.Native.(NativeElement).Value.(tview.Primitive), row, column, rowSpan, columnSpan, minGridHeight, minGridWidth, focus)
		}
		return
	}

	if f, ok := n.Value.(Form); ok {
		f.v.Clear(true)
		for _, c := range children {
			if b, ok := c.Native.(NativeElement).Value.(Button); ok {
				l := b.v.GetLabel()
				f.v.AddButton(l, nil)
			}

			if c, ok := c.Native.(NativeElement).Value.(CheckBox); ok {
				f.v.AddFormItem(c.v)
			}

			if d, ok := c.Native.(NativeElement).Value.(DropDown); ok {
				f.v.AddFormItem(d.v)
			}

			if i, ok := c.Native.(NativeElement).Value.(Image); ok {
				f.v.AddFormItem(i.v)
			}

			if i, ok := c.Native.(NativeElement).Value.(InputField); ok {
				f.v.AddFormItem(i.v)
			}

			if l, ok := c.Native.(NativeElement).Value.(TextArea); ok {
				f.v.AddFormItem(l.v)
			}

			if l, ok := c.Native.(NativeElement).Value.(TextView); ok {
				f.v.AddFormItem(l.v)
			}
		}
		return
	}
}

// TODO implement file storage? json or csv ? zipped?

// LoadFromStorage will load an element properties.
func LoadFromStorage(e *ui.Element) *ui.Element {

	lb, ok := e.Get("event", "storesynced")
	if ok {
		if isSynced := lb.(ui.Bool); isSynced {
			return e
		}

	}
	pmode := ui.PersistenceMode(e)
	storage, ok := e.Configuration.PersistentStorer[pmode]
	if ok {
		err := storage.Load(e)
		if err != nil {
			log.Print(err)
			return e
		}
		e.Set("event", "storesynced", ui.Bool(true))
	}
	return e
}

// PutInStorage stores an element properties in storage
func PutInStorage(e *ui.Element) *ui.Element {
	pmode := ui.PersistenceMode(e)
	storage, ok := e.Configuration.PersistentStorer[pmode]
	if !ok {
		return e
	}
	for cat, props := range e.Properties.Categories {
		if cat != "event" {
			for prop, val := range props.Local {
				storage.Store(e, cat, prop, val)
			}
		}
	}
	e.Set("event", "storesynced", ui.Bool(true))
	return e
}

// ClearFromStorage will clear an element properties from storage.
func ClearFromStorage(e *ui.Element) *ui.Element {
	pmode := ui.PersistenceMode(e)
	storage, ok := e.Configuration.PersistentStorer[pmode]
	if ok {
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
	rng *rand.Rand
	src *rand.PCGSource

	Box        gconstructor[BoxElement, boxConstructor]
	Button     buttongconstructor[ButtonElement, buttonConstructor]
	CheckBox   gconstructor[CheckBoxElement, checkboxConstructor]
	DropDown   gconstructor[DropDownElement, dropdownConstructor]
	Frame      gconstructor[FrameElement, frameConstructor]
	Form       gconstructor[FormElement, formConstructor]
	Flex       gconstructor[FlexElement, flexConstructor]
	Grid       gconstructor[GridElement, gridConstructor]
	Image      gconstructor[ImageElement, imageConstructor]
	InputField gconstructor[InputFieldElement, inputfieldConstructor]
	List       gconstructor[ListElement, listConstructor]
	Modal      gconstructor[ModalElement, modalConstructor]
	Pages      gconstructor[PagesElement, pagesConstructor]
	TreeView   gconstructor[TreeViewElement, treeviewConstructor]
	TextView   gconstructor[TextViewElement, textviewConstructor]
	Table      gconstructor[TableElement, tableConstructor]
	TextArea   gconstructor[TextAreaElement, textareaConstructor]
}

func withStdConstructors(d Document) Document {
	// TODO add all constructors here
	d.Box = gconstructor[BoxElement, boxConstructor](func() BoxElement {
		e := BoxElement{newBox(d.newID())}
		ui.RegisterElement(d.Element, e.AsElement())
		return withEventSupport(e)
	})
	d.Box.ownedBy(&d)

	d.Button = buttongconstructor[ButtonElement, buttonConstructor](func(label string) ButtonElement {
		e := ButtonElement{newButton(d.newID(), label)}
		ui.RegisterElement(d.Element, e.AsElement())
		return withEventSupport(e)
	})
	d.Button.ownedBy(&d)

	d.CheckBox = gconstructor[CheckBoxElement, checkboxConstructor](func() CheckBoxElement {
		e := CheckBoxElement{newCheckBox(d.newID())}
		ui.RegisterElement(d.Element, e.AsElement())
		return withEventSupport(e)
	})
	d.CheckBox.ownedBy(&d)

	d.DropDown = gconstructor[DropDownElement, dropdownConstructor](func() DropDownElement {
		e := DropDownElement{newDropDown(d.newID())}
		ui.RegisterElement(d.Element, e.AsElement())
		return withEventSupport(e)
	})
	d.DropDown.ownedBy(&d)

	d.Frame = gconstructor[FrameElement, frameConstructor](func() FrameElement {
		e := FrameElement{newFrame(d.newID())}
		ui.RegisterElement(d.Element, e.AsElement())
		return withEventSupport(e)
	})
	d.Frame.ownedBy(&d)

	d.Form = gconstructor[FormElement, formConstructor](func() FormElement {
		e := FormElement{newForm(d.newID())}
		ui.RegisterElement(d.Element, e.AsElement())
		return withEventSupport(e)
	})
	d.Form.ownedBy(&d)

	d.Flex = gconstructor[FlexElement, flexConstructor](func() FlexElement {
		e := FlexElement{newFlex(d.newID())}
		ui.RegisterElement(d.Element, e.AsElement())
		return withEventSupport(e)
	})
	d.Flex.ownedBy(&d)

	d.Grid = gconstructor[GridElement, gridConstructor](func() GridElement {
		e := GridElement{newGrid(d.newID())}
		ui.RegisterElement(d.Element, e.AsElement())
		return withEventSupport(e)
	})
	d.Grid.ownedBy(&d)

	d.Image = gconstructor[ImageElement, imageConstructor](func() ImageElement {
		e := ImageElement{newImage(d.newID())}
		ui.RegisterElement(d.Element, e.AsElement())
		return withEventSupport(e)
	})
	d.Image.ownedBy(&d)

	d.InputField = gconstructor[InputFieldElement, inputfieldConstructor](func() InputFieldElement {
		e := InputFieldElement{newInputField(d.newID())}
		ui.RegisterElement(d.Element, e.AsElement())
		return withEventSupport(e)
	})
	d.InputField.ownedBy(&d)

	d.List = gconstructor[ListElement, listConstructor](func() ListElement {
		e := ListElement{newList(d.newID())}
		ui.RegisterElement(d.Element, e.AsElement())
		return withEventSupport(e)
	})
	d.List.ownedBy(&d)

	d.Modal = gconstructor[ModalElement, modalConstructor](func() ModalElement {
		e := ModalElement{newModal(d.newID())}
		ui.RegisterElement(d.Element, e.AsElement())
		return withEventSupport(e)
	})
	d.Modal.ownedBy(&d)

	d.Pages = gconstructor[PagesElement, pagesConstructor](func() PagesElement {
		e := PagesElement{newPages(d.newID())}
		ui.RegisterElement(d.Element, e.AsElement())
		return withEventSupport(e)
	})
	d.Pages.ownedBy(&d)

	d.TreeView = gconstructor[TreeViewElement, treeviewConstructor](func() TreeViewElement {
		e := TreeViewElement{newTreeView(d.newID())}
		ui.RegisterElement(d.Element, e.AsElement())
		return withEventSupport(e)
	})
	d.TreeView.ownedBy(&d)

	d.TextView = gconstructor[TextViewElement, textviewConstructor](func() TextViewElement {
		e := TextViewElement{newTextView(d.newID())}
		ui.RegisterElement(d.Element, e.AsElement())
		return withEventSupport(e)
	})
	d.TextView.ownedBy(&d)

	d.Table = gconstructor[TableElement, tableConstructor](func() TableElement {
		e := TableElement{newTable(d.newID())}
		ui.RegisterElement(d.Element, e.AsElement())
		return withEventSupport(e)
	})
	d.Table.ownedBy(&d)

	d.TextArea = gconstructor[TextAreaElement, textareaConstructor](func() TextAreaElement {
		e := TextAreaElement{newTextArea(d.newID())}
		ui.RegisterElement(d.Element, e.AsElement())
		return withEventSupport(e)
	})
	d.TextArea.ownedBy(&d)

	return d

}

func (d Document) GetElementById(id string) *ui.Element {
	return ui.GetById(d.AsElement(), id)
}

func (d Document) newID() string {
	return newID() // DEBUG
}

func (d Document) Application() ApplicationElement {
	w := d.GetElementById("term-application")
	if w != nil {
		return ApplicationElement{w}
	}
	app := application()
	ui.RegisterElement(d.AsElement(), app.Raw)
	app.Raw.TriggerEvent("mounted", ui.Bool(true))
	app.Raw.TriggerEvent("mountable", ui.Bool(true))

	return app
}

// NewObservable returns a new ui.Observable element after registering it for the document.
// If the observable alreadys exiswted for this id, it is returns as is.
// it is up to the caller to check whether an element already exist for this id and possibly clear
// its state beforehand.
func (d Document) NewObservable(id string, options ...string) ui.Observable {
	if e := d.GetElementById(id); e != nil {
		return ui.Observable{e}
	}
	o := d.AsElement().Configuration.NewObservable(id, options...).AsElement()

	ui.RegisterElement(d.AsElement(), o)

	return ui.Observable{o}
}

func (d Document) OnNavigationEnd(h *ui.MutationHandler) {
	d.AsElement().WatchEvent("navigation-end", d, h)
}

func (d Document) OnReady(h *ui.MutationHandler) {
	d.AsElement().WatchEvent("document-ready", d, h)
}

func (d Document) isReady() bool {
	_, ok := d.GetEventValue("document-ready")
	return ok
}

func (d Document) OnRouterMounted(h *ui.MutationHandler) {
	d.AsElement().WatchEvent("router-mounted", d, h)
}

func (d Document) OnBeforeUnactive(h *ui.MutationHandler) {
	d.AsElement().WatchEvent("before-unactive", d, h)
}

// Router returns the router associated with the document. It is nil if no router has been created.
func (d Document) Router() *ui.Router {
	return ui.GetRouter(d.AsElement())
}

func (d Document) Delete() { // TODO check for dangling references
	ui.DoSync(func() {
		e := d.AsElement()
		d.Router().CancelNavigation()
		ui.Delete(e)
	})
}

func (d Document) ListenAndServe(ctx context.Context) {
	if d.Element == nil {
		panic("document is missing")
	}

	a := d.Application()

	d.WatchEvent("running", a.Raw, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		d.TriggerEvent("document-ready")
		return false
	}).RunASAP())

	d.Router().ListenAndServe(ctx, "", a)
	a.Run()

}

func (d Document) NativeElement() tview.Primitive {
	return d.AsElement().Native.(NativeElement).Value.(tview.Primitive)
}

func GetDocument(e *ui.Element) Document {
	if document != nil {
		return *document
	}
	if e.Root == nil {
		panic("This element does not belong to any registered subtree of the Document. Root is nil. If root of a component, it should be declared as such by callling the NewComponent method of the document Element.")
	}
	return withStdConstructors(Document{Element: e.Root}) // TODO initialize document *Element constructors
}

func getDocumentRef(e *ui.Element) *Document {
	if document != nil {
		return document
	}
	return &Document{Element: e.Root}
}

var newDocument = Elements.NewConstructor("root", func(id string) *ui.Element {

	e := Elements.NewAppRoot(id).AsElement()

	root := tview.NewFlex().SetDirection(tview.FlexRowCSS).SetFullScreen(true)
	e.Native = NewNativeElementWrapper(root)

	w := GetApplication(e)

	err := w.NativeElement().SetRoot(root, true)
	if err != nil {
		panic(err)
	}

	return withEventSupport(e)
}, AllowDataStorage)

// NewDocument returns the root of a new terminal app. It is the top-most element
// in the tree of Elements that consitute the full document.
// Options such as the location of persisted data can be passed to the constructor of an instance.
func NewDocument(id string, options ...string) Document {
	d := Document{Element: newDocument(id, options...)}

	d = withStdConstructors(d)

	document = &d

	return d
}

// Document option that turns data storage on for a document
var EnableDataStorage = "enable-data-storage"
var AllowDataStorage = ui.NewConstructorOption("enable-data-storage", func(e *ui.Element) *ui.Element {
	if diskStorage != nil {
		return e
	}
	err := initDiskStorage("./datastore.json")
	if err != nil {
		panic(err)
	}
	d := getDocumentRef(e)
	d.OnBeforeUnactive(ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		if diskStorage == nil {
			return false
		}
		err := diskStorage.Close()
		if err != nil {
			log.Print(err) // TODO check output location
		}
		return false
	}))
	return e
})

// BoxElement
type BoxElement struct {
	*ui.Element
}

func (e BoxElement) NativeElement() *tview.Box {
	return e.AsElement().Native.(NativeElement).Value.Box()
}

func (e BoxElement) UnderlyingBox() BoxElement {
	return e
}

type boxModifier struct{}

// Modifier holds the modifiers for the BoxElement.
// These modifiers can be used for all UI elements that are built upon  a *tview.Box
var Modifier boxModifier

func (m boxModifier) BackgroundColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.Box().SetBackgroundColor(color)
		return e
	}
}

func (m boxModifier) ShowBorder(b bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.Box().SetBorder(b)
		return e
	}
}

func (m boxModifier) BorderColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.Box().SetBorderColor(color)
		return e
	}
}

func (m boxModifier) BorderPadding(left, top, right, bottom int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.Box().SetBorderPadding(left, top, right, bottom)
		return e
	}
}

func (m boxModifier) BorderAttributes(attr tcell.AttrMask) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.Box().SetBorderAttributes(attr)
		return e
	}
}

func (m boxModifier) BorderStyle(style tcell.Style) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.Box().SetBorderStyle(style)
		return e
	}
}

func (m boxModifier) SetTitle(title string) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.Box().SetTitle(title)
		return e
	}
}

func (m boxModifier) SetTitleAlign(align int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.Box().SetTitleAlign(align)
		return e
	}
}

func (m boxModifier) SetTitleColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.Box().SetTitleColor(color)
		return e
	}
}

func (m boxModifier) SetRect(x, y, width, height int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.Box().SetRect(x, y, width, height)
		return e
	}
}

func (m boxModifier) FlexItem(fixedsize int, proportion int, focus bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		prop := ui.NewObject()
		prop.Set("fixedsize", ui.Number(fixedsize))
		prop.Set("proportion", ui.Number(proportion))
		prop.Set("focus", ui.Bool(focus))
		e.SetUI("flex", prop.Commit())
		return e
	}
}

func (m boxModifier) GridItem(row, column int, rowSpan, columnSpan int, minGridHeight, minGridWidth int, focus bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		prop := ui.NewObject()
		prop.Set("row", ui.Number(row))
		prop.Set("column", ui.Number(column))
		prop.Set("rowSpan", ui.Number(rowSpan))
		prop.Set("columnSpan", ui.Number(columnSpan))
		prop.Set("minGridHeight", ui.Number(minGridHeight))
		prop.Set("minGridWidth", ui.Number(minGridWidth))
		prop.Set("focus", ui.Bool(focus))
		e.SetUI("grid", prop.Commit())
		return e
	}
}

var newBox = Elements.NewConstructor("box", func(id string) *ui.Element {

	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewBox())

	// TODO think about calling Draw OnMounted

	return e
}, AllowDataStorage)

type boxConstructor func() BoxElement

func (c boxConstructor) WithID(id string, options ...string) BoxElement {
	return BoxElement{newBox(id, options...)}
}

// ButtonElement
type ButtonElement struct {
	*ui.Element
}

func (e ButtonElement) NativeElement() *tview.Button {
	return e.AsElement().Native.(NativeElement).Value.(Button).v
}

func (e ButtonElement) UnderlyingBox() BoxElement {
	box := document.GetElementById(e.AsElement().ID + "-box")
	if box != nil {
		return BoxElement{box}
	}

	b := document.Box.WithID(e.AsElement().ID + "-box")
	b.AsElement().Native = NewNativeElementWrapper(e.NativeElement().Box)
	return b
}

var newButton = Elements.NewConstructor("button", func(id string) *ui.Element {

	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewButton(""))

	// TODO think about calling Draw OnMounted

	return e
}, AllowDataStorage)

type buttonConstructor func(label string) ButtonElement

func (c buttonConstructor) WithID(id string, label string, options ...string) ButtonElement {
	b := ButtonElement{newButton(id, options...)}
	b.NativeElement().SetLabel(label)
	return b
}

type buttonModifier struct{}

// ButtonModifier holds the modfiers for the ButtonElement.
var ButtonModifier buttonModifier

func (m buttonModifier) Label(label string) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Button).v.SetLabel(label)
		return e
	}
}

func (m buttonModifier) LabelColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Button).v.SetLabelColor(color)
		return e
	}
}

func (m buttonModifier) LabelColorActivated(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Button).v.SetLabelColorActivated(color)
		return e
	}
}

func (m buttonModifier) Disabled(b bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Button).v.SetDisabled(b)
		return e
	}
}

func (m buttonModifier) DisabledStyle(style tcell.Style) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Button).v.SetDisabledStyle(style)
		return e
	}
}

func (m buttonModifier) FlexItem(fixedsize int, proportion int, focus bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		prop := ui.NewObject()
		prop.Set("fixedsize", ui.Number(fixedsize))
		prop.Set("proportion", ui.Number(proportion))
		prop.Set("focus", ui.Bool(focus))
		e.SetUI("flex", prop.Commit())
		return e
	}
}

func (m buttonModifier) GridItem(row, column int, rowSpan, columnSpan int, minGridHeight, minGridWidth int, focus bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		prop := ui.NewObject()
		prop.Set("row", ui.Number(row))
		prop.Set("column", ui.Number(column))
		prop.Set("rowSpan", ui.Number(rowSpan))
		prop.Set("columnSpan", ui.Number(columnSpan))
		prop.Set("minGridHeight", ui.Number(minGridHeight))
		prop.Set("minGridWidth", ui.Number(minGridWidth))
		prop.Set("focus", ui.Bool(focus))
		e.SetUI("grid", prop.Commit())
		return e
	}
}

// CheckBoxElement
type CheckBoxElement struct {
	*ui.Element
}

func (e CheckBoxElement) NativeElement() *tview.Checkbox {
	return e.AsElement().Native.(NativeElement).Value.(CheckBox).v
}

func (e CheckBoxElement) UnderlyingBox() BoxElement {
	box := document.GetElementById(e.AsElement().ID + "-box")
	if box != nil {
		return BoxElement{box}
	}

	b := document.Box.WithID(e.AsElement().ID + "-box")
	b.AsElement().Native = NewNativeElementWrapper(e.NativeElement().Box)
	return b
}

var newCheckBox = Elements.NewConstructor("checkbox", func(id string) *ui.Element {

	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewCheckbox())
	return e
}, AllowDataStorage)

type checkboxConstructor func() CheckBoxElement

func (c checkboxConstructor) WithID(id string, options ...string) CheckBoxElement {
	e := CheckBoxElement{newCheckBox(id, options...)}
	return e
}

type checkBoxModifier struct{}

// CheckBoxModifier holds the modifiers for the CheckBoxElement.
var CheckBoxModifier checkBoxModifier

func (m checkBoxModifier) Checked(b bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(CheckBox).v.SetChecked(b)
		return e
	}
}

func (m checkBoxModifier) CheckedString(checked string) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(CheckBox).v.SetCheckedString(checked)
		return e
	}
}

func (m checkBoxModifier) Disabled(b bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(CheckBox).v.SetDisabled(b)
		return e
	}
}

func (m checkBoxModifier) FieldBackgroundColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(CheckBox).v.SetFieldBackgroundColor(color)
		return e
	}
}

func (m checkBoxModifier) FieldTextColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(CheckBox).v.SetFieldTextColor(color)
		return e
	}
}

func (m checkBoxModifier) FormAttributes(labelWidth int, labelColor, bgColor, fieldTextColor, fieldBgColor tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(CheckBox).v.SetFormAttributes(labelWidth, labelColor, bgColor, fieldTextColor, fieldBgColor)
		return e
	}
}

func (m checkBoxModifier) Label(label string) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(CheckBox).v.SetLabel(label)
		return e
	}
}

func (m checkBoxModifier) LabelColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(CheckBox).v.SetLabelColor(color)
		return e
	}
}

func (m checkBoxModifier) LabelWidth(width int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(CheckBox).v.SetLabelWidth(width)
		return e
	}
}

func (m checkBoxModifier) FlexItem(fixedsize int, proportion int, focus bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		prop := ui.NewObject()
		prop.Set("fixedsize", ui.Number(fixedsize))
		prop.Set("proportion", ui.Number(proportion))
		prop.Set("focus", ui.Bool(focus))
		e.SetUI("flex", prop.Commit())
		return e
	}
}

func (m checkBoxModifier) GridItem(row, column int, rowSpan, columnSpan int, minGridHeight, minGridWidth int, focus bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		prop := ui.NewObject()
		prop.Set("row", ui.Number(row))
		prop.Set("column", ui.Number(column))
		prop.Set("rowSpan", ui.Number(rowSpan))
		prop.Set("columnSpan", ui.Number(columnSpan))
		prop.Set("minGridHeight", ui.Number(minGridHeight))
		prop.Set("minGridWidth", ui.Number(minGridWidth))
		prop.Set("focus", ui.Bool(focus))
		e.SetUI("grid", prop.Commit())
		return e
	}
}

// DropDownElement
type DropDownElement struct {
	*ui.Element
}

func (e DropDownElement) NativeElement() *tview.DropDown {
	return e.AsElement().Native.(NativeElement).Value.(DropDown).v
}

func (e DropDownElement) UnderlyingBox() BoxElement {
	box := document.GetElementById(e.AsElement().ID + "-box")
	if box != nil {
		return BoxElement{box}
	}

	b := document.Box.WithID(e.AsElement().ID + "-box")
	b.AsElement().Native = NewNativeElementWrapper(e.NativeElement().Box)
	return b
}

func (e DropDownElement) OnSelected(h *ui.MutationHandler) {
	e.AsElement().WatchEvent("selected", e, h)
}

var newDropDown = Elements.NewConstructor("dropdown", func(id string) *ui.Element {

	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewDropDown())

	return e
}, AllowDataStorage)

type dropdownConstructor func() DropDownElement

func (c dropdownConstructor) WithID(id string, options ...string) DropDownElement {
	e := DropDownElement{newDropDown(id, options...)}
	return e
}

type dropDownModifier struct{}

var DropDownModifier dropDownModifier

func (m dropDownModifier) Options(options ...string) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(DropDown).v.SetOptions(options, nil)
		opt := ui.NewList()
		for _, o := range options {
			opt.Append(ui.String(o))
		}
		v, ok := e.GetUI("dropdown")
		if !ok {
			e.SetUI("dropdown", ui.NewObject().Set("options", opt.Commit()).Commit())
		}
		o := v.(ui.Object).MakeCopy().Set("options", opt.Commit()).Commit()
		e.SetUI("dropdown", o)

		return e
	}
}

func (m dropDownModifier) CurrentOption(index int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(DropDown).v.SetCurrentOption(index)
		return e
	}
}

func (m dropDownModifier) Label(label string) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(DropDown).v.SetLabel(label)
		return e
	}
}

func (m dropDownModifier) LabelColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(DropDown).v.SetLabelColor(color)
		return e
	}
}

func (m dropDownModifier) LabelWidth(width int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(DropDown).v.SetLabelWidth(width)
		return e
	}
}

func (m dropDownModifier) Disabled(b bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(DropDown).v.SetDisabled(b)
		return e
	}
}

func (m dropDownModifier) FieldBackgroundColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(CheckBox).v.SetFieldBackgroundColor(color)
		return e
	}
}

func (m dropDownModifier) FieldTextColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(CheckBox).v.SetFieldTextColor(color)
		return e
	}
}

func (m dropDownModifier) FieldWidth(width int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(DropDown).v.SetFieldWidth(width)
		return e
	}
}

func (m dropDownModifier) FormAttributes(labelWidth int, labelColor, bgColor, fieldTextColor, fieldBgColor tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(CheckBox).v.SetFormAttributes(labelWidth, labelColor, bgColor, fieldTextColor, fieldBgColor)
		return e
	}
}

func (m dropDownModifier) ListStyles(unselected, selected tcell.Style) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(DropDown).v.SetListStyles(unselected, selected)
		return e
	}
}

func (m dropDownModifier) PrefixTextColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(DropDown).v.SetPrefixTextColor(color)
		return e
	}
}

func (m dropDownModifier) TextOptions(prefix, suffix, currentPrefix, currentSuffix, noSelection string) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(DropDown).v.SetTextOptions(prefix, suffix, currentPrefix, currentSuffix, noSelection)
		return e
	}
}

func (m dropDownModifier) FlexItem(fixedsize int, proportion int, focus bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		prop := ui.NewObject()
		prop.Set("fixedsize", ui.Number(fixedsize))
		prop.Set("proportion", ui.Number(proportion))
		prop.Set("focus", ui.Bool(focus))
		e.SetUI("flex", prop.Commit())
		return e
	}
}

func (m dropDownModifier) GridItem(row, column int, rowSpan, columnSpan int, minGridHeight, minGridWidth int, focus bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		prop := ui.NewObject()
		prop.Set("row", ui.Number(row))
		prop.Set("column", ui.Number(column))
		prop.Set("rowSpan", ui.Number(rowSpan))
		prop.Set("columnSpan", ui.Number(columnSpan))
		prop.Set("minGridHeight", ui.Number(minGridHeight))
		prop.Set("minGridWidth", ui.Number(minGridWidth))
		prop.Set("focus", ui.Bool(focus))
		e.SetUI("grid", prop.Commit())
		return e
	}
}

// FlexElement
type FlexElement struct {
	*ui.Element
}

func (e FlexElement) NativeElement() *tview.Flex {
	return e.AsElement().Native.(NativeElement).Value.(Flex).v
}

func (e FlexElement) UnderlyingBox() BoxElement {
	box := document.GetElementById(e.AsElement().ID + "-box")
	if box != nil {
		return BoxElement{box}
	}

	b := document.Box.WithID(e.AsElement().ID + "-box")
	b.AsElement().Native = NewNativeElementWrapper(e.NativeElement().Box)
	return b
}

var newFlex = Elements.NewConstructor("flex", func(id string) *ui.Element {

	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewFlex())

	// TODO think about calling Draw OnMounted

	return e
}, AllowDataStorage)

type flexConstructor func() FlexElement

func (c flexConstructor) WithID(id string, options ...string) FlexElement {
	e := FlexElement{newFlex(id, options...)}
	return e
}

type flexModifier struct{}

var FlexModifier flexModifier

func (m flexModifier) Horizontal() func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Flex).v.SetDirection(tview.FlexRowCSS)
		return e
	}
}

func (m flexModifier) Vertical() func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Flex).v.SetDirection(tview.FlexColumnCSS)
		return e
	}
}

func (m flexModifier) FlexItem(fixedsize int, proportion int, focus bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		prop := ui.NewObject()
		prop.Set("fixedsize", ui.Number(fixedsize))
		prop.Set("proportion", ui.Number(proportion))
		prop.Set("focus", ui.Bool(focus))
		e.SetUI("flex", prop.Commit())
		return e
	}
}

func (m flexModifier) GridItem(row, column int, rowSpan, columnSpan int, minGridHeight, minGridWidth int, focus bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		prop := ui.NewObject()
		prop.Set("row", ui.Number(row))
		prop.Set("column", ui.Number(column))
		prop.Set("rowSpan", ui.Number(rowSpan))
		prop.Set("columnSpan", ui.Number(columnSpan))
		prop.Set("minGridHeight", ui.Number(minGridHeight))
		prop.Set("minGridWidth", ui.Number(minGridWidth))
		prop.Set("focus", ui.Bool(focus))
		e.SetUI("grid", prop.Commit())
		return e
	}
}

// FormElement
type FormElement struct {
	*ui.Element
}

func (e FormElement) NativeElement() *tview.Form {
	return e.AsElement().Native.(NativeElement).Value.(Form).v
}

func (e FormElement) UnderlyingBox() BoxElement {
	box := document.GetElementById(e.AsElement().ID + "-box")
	if box != nil {
		return BoxElement{box}
	}

	b := document.Box.WithID(e.AsElement().ID + "-box")
	b.AsElement().Native = NewNativeElementWrapper(e.NativeElement().Box)
	return b
}

var newForm = Elements.NewConstructor("form", func(id string) *ui.Element {

	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewForm())

	// TODO think about calling Draw OnMounted

	return e
}, AllowDataStorage)

type formConstructor func() FormElement

func (c formConstructor) WithID(id string, options ...string) FormElement {
	e := FormElement{newForm(id, options...)}
	return e
}

type formModifier struct{}

var FormModifier formModifier

func (m formModifier) Clear(exceptbuttons bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Form).v.Clear(!exceptbuttons)
		return e
	}
}

func (m formModifier) ButtonActivatedStyle(style tcell.Style) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Form).v.SetButtonActivatedStyle(style)
		return e
	}
}

func (m formModifier) ButtonBackgroundColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Form).v.SetButtonBackgroundColor(color)
		return e
	}
}

func (m formModifier) ButtonDisabledStyle(style tcell.Style) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Form).v.SetButtonDisabledStyle(style)
		return e
	}
}

func (m formModifier) ButtonStyle(style tcell.Style) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Form).v.SetButtonStyle(style)
		return e
	}
}

func (m formModifier) ButtonTextColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Form).v.SetButtonTextColor(color)
		return e
	}
}

func (m formModifier) ButtonsAlign(align int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Form).v.SetButtonsAlign(align)
		return e
	}
}

func (m formModifier) FieldBackgroundColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Form).v.SetFieldBackgroundColor(color)
		return e
	}
}

func (m formModifier) FieldTextColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Form).v.SetFieldTextColor(color)
		return e
	}
}

func (m formModifier) SetFocus(index int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Form).v.SetFocus(index)
		return e
	}
}

func (m formModifier) Horizontal() func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Form).v.SetHorizontal(true)
		return e
	}
}

func (m formModifier) ItemPadding(padding int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Form).v.SetItemPadding(padding)
		return e
	}
}

func (m formModifier) FlexItem(fixedsize int, proportion int, focus bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		prop := ui.NewObject()
		prop.Set("fixedsize", ui.Number(fixedsize))
		prop.Set("proportion", ui.Number(proportion))
		prop.Set("focus", ui.Bool(focus))
		e.SetUI("flex", prop.Commit())
		return e
	}
}

func (m formModifier) GridItem(row, column int, rowSpan, columnSpan int, minGridHeight, minGridWidth int, focus bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		prop := ui.NewObject()
		prop.Set("row", ui.Number(row))
		prop.Set("column", ui.Number(column))
		prop.Set("rowSpan", ui.Number(rowSpan))
		prop.Set("columnSpan", ui.Number(columnSpan))
		prop.Set("minGridHeight", ui.Number(minGridHeight))
		prop.Set("minGridWidth", ui.Number(minGridWidth))
		prop.Set("focus", ui.Bool(focus))
		e.SetUI("grid", prop.Commit())
		return e
	}
}

func (m formModifier) LabelColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Form).v.SetLabelColor(color)
		return e
	}
}

// FrameElement allows to render space around an element if provided, otherwise, just some space.
type FrameElement struct {
	*ui.Element
}

func (e FrameElement) NativeElement() *tview.Frame {
	return e.AsElement().Native.(NativeElement).Value.(Frame).v
}

func (e FrameElement) UnderlyingBox() BoxElement {
	box := document.GetElementById(e.AsElement().ID + "-box")
	if box != nil {
		return BoxElement{box}
	}

	b := document.Box.WithID(e.AsElement().ID + "-box")
	b.AsElement().Native = NewNativeElementWrapper(e.NativeElement().Box)
	return b
}

var newFrame = Elements.NewConstructor("frame", func(id string) *ui.Element {

	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewFrame(nil))

	// TODO think about calling Draw OnMounted

	return e
}, AllowDataStorage)

type frameConstructor func() FrameElement

func (c frameConstructor) WithID(id string, options ...string) FrameElement {
	return FrameElement{newFrame(id, options...)}
}

type frameModifier struct{}

var FrameModifier frameModifier

func (m frameModifier) AddText(text string, header bool, align int, color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Frame).v.AddText(text, header, align, color)
		return e
	}
}

func (m frameModifier) SetBorders(top, bottom, header, footer, left, right int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Frame).v.SetBorders(top, bottom, header, footer, left, right)
		return e
	}
}

func (m frameModifier) FlexItem(fixedsize int, proportion int, focus bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		prop := ui.NewObject()
		prop.Set("fixedsize", ui.Number(fixedsize))
		prop.Set("proportion", ui.Number(proportion))
		prop.Set("focus", ui.Bool(focus))
		e.SetUI("flex", prop.Commit())
		return e
	}
}

func (m frameModifier) GridItem(row, column int, rowSpan, columnSpan int, minGridHeight, minGridWidth int, focus bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		prop := ui.NewObject()
		prop.Set("row", ui.Number(row))
		prop.Set("column", ui.Number(column))
		prop.Set("rowSpan", ui.Number(rowSpan))
		prop.Set("columnSpan", ui.Number(columnSpan))
		prop.Set("minGridHeight", ui.Number(minGridHeight))
		prop.Set("minGridWidth", ui.Number(minGridWidth))
		prop.Set("focus", ui.Bool(focus))
		e.SetUI("grid", prop.Commit())
		return e
	}
}

// GridElement
type GridElement struct {
	*ui.Element
}

func (e GridElement) NativeElement() *tview.Grid {
	return e.AsElement().Native.(NativeElement).Value.(Grid).v
}

func (e GridElement) UnderlyingBox() BoxElement {
	box := document.GetElementById(e.AsElement().ID + "-box")
	if box != nil {
		return BoxElement{box}
	}

	b := document.Box.WithID(e.AsElement().ID + "-box")
	b.AsElement().Native = NewNativeElementWrapper(e.NativeElement().Box)
	return b
}

var newGrid = Elements.NewConstructor("grid", func(id string) *ui.Element {

	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewGrid())

	// TODO think about calling Draw OnMounted

	return e
}, AllowDataStorage)

type gridConstructor func() GridElement

func (c gridConstructor) WithID(id string, options ...string) GridElement {
	e := GridElement{newGrid(id, options...)}
	return e
}

type gridModifier struct{}

var GridModifier gridModifier

func (m gridModifier) Columns(columns ...int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Grid).v.SetColumns(columns...)
		return e
	}
}

func (m gridModifier) Rows(rows ...int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Grid).v.SetRows(rows...)
		return e
	}
}

func (m gridModifier) Borders(b bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Grid).v.SetBorders(b)
		return e
	}
}

func (m gridModifier) BordersColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Grid).v.SetBordersColor(color)
		return e
	}
}

func (m gridModifier) Gap(row, column int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Grid).v.SetGap(row, column)
		return e
	}
}

func (m gridModifier) MinSize(row, column int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Grid).v.SetMinSize(row, column)
		return e
	}
}

func (m gridModifier) Offset(row, column int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Grid).v.SetOffset(row, column)
		return e
	}
}

func (m gridModifier) Size(numRows, numColumn, rowSize, columnSize int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Grid).v.SetSize(numRows, numColumn, rowSize, columnSize)
		return e
	}
}

func (m gridModifier) FlexItem(fixedsize int, proportion int, focus bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		prop := ui.NewObject()
		prop.Set("fixedsize", ui.Number(fixedsize))
		prop.Set("proportion", ui.Number(proportion))
		prop.Set("focus", ui.Bool(focus))
		e.SetUI("flex", prop.Commit())
		return e
	}
}

func (m gridModifier) GridItem(row, column int, rowSpan, columnSpan int, minGridHeight, minGridWidth int, focus bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		prop := ui.NewObject()
		prop.Set("row", ui.Number(row))
		prop.Set("column", ui.Number(column))
		prop.Set("rowSpan", ui.Number(rowSpan))
		prop.Set("columnSpan", ui.Number(columnSpan))
		prop.Set("minGridHeight", ui.Number(minGridHeight))
		prop.Set("minGridWidth", ui.Number(minGridWidth))
		prop.Set("focus", ui.Bool(focus))
		e.SetUI("grid", prop.Commit())
		return e
	}
}

// ImageElement
type ImageElement struct {
	*ui.Element
}

func (e ImageElement) NativeElement() *tview.Image {
	return e.AsElement().Native.(NativeElement).Value.(Image).v
}

func (e ImageElement) UnderlyingBox() BoxElement {
	box := document.GetElementById(e.AsElement().ID + "-box")
	if box != nil {
		return BoxElement{box}
	}

	b := document.Box.WithID(e.AsElement().ID + "-box")
	b.AsElement().Native = NewNativeElementWrapper(e.NativeElement().Box)
	return b
}

var newImage = Elements.NewConstructor("image", func(id string) *ui.Element {

	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewImage())

	// TODO think about calling Draw OnMounted

	return e
}, AllowDataStorage)

type imageConstructor func() ImageElement

func (c imageConstructor) WithID(id string, options ...string) ImageElement {
	e := ImageElement{newImage(id, options...)}
	return e
}

type imageModifier struct{}

var ImageModifier imageModifier

func (m imageModifier) Align(vertical, horizontal int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Image).v.SetAlign(vertical, horizontal)
		return e
	}
}

func (m imageModifier) AspectRatio(aspectRatio float64) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Image).v.SetAspectRatio(aspectRatio)
		return e
	}
}

func (m imageModifier) Colors(colors int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Image).v.SetColors(colors)
		return e
	}
}

func (m imageModifier) Disabled(b bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Image).v.SetDisabled(b)
		return e
	}
}

func (m imageModifier) Dithering(d int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Image).v.SetDithering(d)
		return e
	}
}

func (m imageModifier) FormAttributes(labelWidth int, labelColor, bgColoro, fieldTextColor, fieldbgColor tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Image).v.SetFormAttributes(labelWidth, labelColor, bgColoro, fieldTextColor, fieldbgColor)
		return e
	}
}

func (m imageModifier) SetImage(image image.Image) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Image).v.SetImage(image)
		return e
	}
}

func (m imageModifier) Label(label string) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Image).v.SetLabel(label)
		return e
	}
}

func (m imageModifier) LabelStyle(style tcell.Style) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Image).v.SetLabelStyle(style)
		return e
	}
}

func (m imageModifier) LabelWidth(width int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Image).v.SetLabelWidth(width)
		return e
	}
}

func (m imageModifier) Size(rows, colums int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Image).v.SetSize(rows, colums)
		return e
	}
}

func (m imageModifier) FlexItem(fixedsize int, proportion int, focus bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		prop := ui.NewObject()
		prop.Set("fixedsize", ui.Number(fixedsize))
		prop.Set("proportion", ui.Number(proportion))
		prop.Set("focus", ui.Bool(focus))
		e.SetUI("flex", prop.Commit())
		return e
	}
}

func (m imageModifier) GridItem(row, column int, rowSpan, columnSpan int, minGridHeight, minGridWidth int, focus bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		prop := ui.NewObject()
		prop.Set("row", ui.Number(row))
		prop.Set("column", ui.Number(column))
		prop.Set("rowSpan", ui.Number(rowSpan))
		prop.Set("columnSpan", ui.Number(columnSpan))
		prop.Set("minGridHeight", ui.Number(minGridHeight))
		prop.Set("minGridWidth", ui.Number(minGridWidth))
		prop.Set("focus", ui.Bool(focus))
		e.SetUI("grid", prop.Commit())
		return e
	}
}

// InputFieldElement
type InputFieldElement struct {
	*ui.Element
}

func (e InputFieldElement) NativeElement() *tview.InputField {
	return e.AsElement().Native.(NativeElement).Value.(InputField).v
}

func (e InputFieldElement) UnderlyingBox() BoxElement {
	box := document.GetElementById(e.AsElement().ID + "-box")
	if box != nil {
		return BoxElement{box}
	}

	b := document.Box.WithID(e.AsElement().ID + "-box")
	b.AsElement().Native = NewNativeElementWrapper(e.NativeElement().Box)
	return b
}

var newInputField = Elements.NewConstructor("inputfield", func(id string) *ui.Element {

	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewInputField())
	return e
}, AllowDataStorage)

type inputfieldConstructor func() InputFieldElement

func (c inputfieldConstructor) WithID(id string, options ...string) InputFieldElement {
	e := InputFieldElement{newInputField(id, options...)}
	return e
}

type inputfieldModifier struct{}

var InputFieldModifier inputfieldModifier

func (m inputfieldModifier) FieldBackgroundColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(InputField).v.SetFieldBackgroundColor(color)
		return e
	}
}

func (m inputfieldModifier) FieldStyle(style tcell.Style) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(InputField).v.SetFieldStyle(style)
		return e
	}
}

func (m inputfieldModifier) FieldTextColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(InputField).v.SetFieldTextColor(color)
		return e
	}
}

func (m inputfieldModifier) FieldWidth(width int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(InputField).v.SetFieldWidth(width)
		return e
	}
}

func (m inputfieldModifier) FormAttributes(labelWidth int, labelColor, bgColoro, fieldTextColor, fieldbgColor tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(InputField).v.SetFormAttributes(labelWidth, labelColor, bgColoro, fieldTextColor, fieldbgColor)
		return e
	}
}

func (m inputfieldModifier) Label(label string) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(InputField).v.SetLabel(label)
		return e
	}
}

func (m inputfieldModifier) LabelColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(InputField).v.SetLabelColor(color)
		return e
	}
}

func (m inputfieldModifier) LabelStyle(style tcell.Style) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(InputField).v.SetLabelStyle(style)
		return e
	}
}

func (m inputfieldModifier) LabelWidth(width int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(InputField).v.SetLabelWidth(width)
		return e
	}
}

func (m inputfieldModifier) MaskCharacter(char rune) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(InputField).v.SetMaskCharacter(char)
		return e
	}
}

func (m inputfieldModifier) Placeholder(text string) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(InputField).v.SetPlaceholder(text)
		return e
	}
}

func (m inputfieldModifier) PlaceholderStyle(style tcell.Style) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(InputField).v.SetPlaceholderStyle(style)
		return e
	}
}

func (m inputfieldModifier) PlaceholderTextColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(InputField).v.SetPlaceholderTextColor(color)
		return e
	}
}

func (m inputfieldModifier) Text(text string) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(InputField).v.SetText(text)
		return e
	}
}

func (m inputfieldModifier) FlexItem(fixedsize int, proportion int, focus bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		prop := ui.NewObject()
		prop.Set("fixedsize", ui.Number(fixedsize))
		prop.Set("proportion", ui.Number(proportion))
		prop.Set("focus", ui.Bool(focus))
		e.SetUI("flex", prop.Commit())
		return e
	}
}

func (m inputfieldModifier) GridItem(row, column int, rowSpan, columnSpan int, minGridHeight, minGridWidth int, focus bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		prop := ui.NewObject()
		prop.Set("row", ui.Number(row))
		prop.Set("column", ui.Number(column))
		prop.Set("rowSpan", ui.Number(rowSpan))
		prop.Set("columnSpan", ui.Number(columnSpan))
		prop.Set("minGridHeight", ui.Number(minGridHeight))
		prop.Set("minGridWidth", ui.Number(minGridWidth))
		prop.Set("focus", ui.Bool(focus))
		e.SetUI("grid", prop.Commit())
		return e
	}
}

// CmdExec modifier turns the input field into a command executor by listening to the
// Done event and executing  the command string after splitting it using strings.Field.

func (m inputfieldModifier) ExecuteCommands() func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.AddEventListener("done", ui.NewEventHandler(func(ev ui.Event) bool {
			cmd := e.Native.(NativeElement).Value.(InputField).v.GetText()

			if cmd == "" {
				return false
			}

			e.Native.(NativeElement).Value.(InputField).v.SetText("")
			args := strings.Fields(cmd)
			if len(args) == 0 {
				return false
			}
			exec.Command(args[0], args[1:]...).Run()
			return false
		}))
		return e
	}
}

// ListElement
type ListElement struct {
	*ui.Element
}

func (e ListElement) NativeElement() *tview.List {
	return e.AsElement().Native.(NativeElement).Value.(List).v
}

func (e ListElement) UnderlyingBox() BoxElement {
	box := document.GetElementById(e.AsElement().ID + "-box")
	if box != nil {
		return BoxElement{box}
	}

	b := document.Box.WithID(e.AsElement().ID + "-box")
	b.AsElement().Native = NewNativeElementWrapper(e.NativeElement().Box)
	return b
}

var newList = Elements.NewConstructor("list", func(id string) *ui.Element {

	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewList())
	return e
}, AllowDataStorage)

type listConstructor func() ListElement

func (c listConstructor) WithID(id string, options ...string) ListElement {
	e := ListElement{newList(id, options...)}
	return e
}

type listModifier struct{}

var ListModifier listModifier

// Given the API for tview.List , we 'd like to generate the modifiers
// for the corresponding InputFieldElement i.e. inputfieldModifier.
// It should follow the same patterns as the other modifiers, especially
// wrt method signatures and naming scheme. A lot of the code is similar to what has already been written.
// Usually, the modifiers are the Setter methods of the corresponding tview type.
// except for the callback accepting methods such SetChangedFunc.
// Namely: AddItem, CurrentItem, HighlightFullLine, ItemText, MainTextColor, MainTextStyle, Offset, SecondaryTextColor, SecondaryTextStyle, SelectedBackgroundColor, SelectedFocusOnly, SelectedStyle, SelectedTextColor, ShortcutColor, ShortcutStyle, WrapAround, and finally ShowSecondaryText.

func (m listModifier) AddItem(text, secondaryText string, shortcut rune, selected func()) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(List).v.AddItem(text, secondaryText, shortcut, selected)
		return e
	}
}

func (m listModifier) CurrentItem(index int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(List).v.SetCurrentItem(index)
		return e
	}
}

func (m listModifier) HighlightFullLine(b bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(List).v.SetHighlightFullLine(b)
		return e
	}
}

func (m listModifier) ItemText(index int, main, secondary string) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(List).v.SetItemText(index, main, secondary)
		return e
	}
}

func (m listModifier) MainTextColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(List).v.SetMainTextColor(color)
		return e
	}
}

func (m listModifier) MainTextStyle(style tcell.Style) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(List).v.SetMainTextStyle(style)
		return e
	}
}

func (m listModifier) Offset(items, horizontal int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(List).v.SetOffset(items, horizontal)
		return e
	}
}

func (m listModifier) SecondaryTextColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(List).v.SetSecondaryTextColor(color)
		return e
	}
}

func (m listModifier) SecondaryTextStyle(style tcell.Style) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(List).v.SetSecondaryTextStyle(style)
		return e
	}
}

func (m listModifier) SelectedBackgroundColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(List).v.SetSelectedBackgroundColor(color)
		return e
	}
}

func (m listModifier) SelectedFocusOnly(b bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(List).v.SetSelectedFocusOnly(b)
		return e
	}
}

func (m listModifier) SelectedStyle(style tcell.Style) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(List).v.SetSelectedStyle(style)
		return e
	}
}

func (m listModifier) SelectedTextColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(List).v.SetSelectedTextColor(color)
		return e
	}
}

func (m listModifier) ShortcutColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(List).v.SetShortcutColor(color)
		return e
	}
}

func (m listModifier) ShortcutStyle(style tcell.Style) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(List).v.SetShortcutStyle(style)
		return e
	}
}

func (m listModifier) WrapAround(b bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(List).v.SetWrapAround(b)
		return e
	}
}

func (m listModifier) ShowSecondaryText(b bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(List).v.ShowSecondaryText(b)
		return e
	}
}

func (m listModifier) FlexItem(fixedsize int, proportion int, focus bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		prop := ui.NewObject()
		prop.Set("fixedsize", ui.Number(fixedsize))
		prop.Set("proportion", ui.Number(proportion))
		prop.Set("focus", ui.Bool(focus))
		e.SetUI("flex", prop.Commit())
		return e
	}
}

func (m listModifier) GridItem(row, column int, rowSpan, columnSpan int, minGridHeight, minGridWidth int, focus bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		prop := ui.NewObject()
		prop.Set("row", ui.Number(row))
		prop.Set("column", ui.Number(column))
		prop.Set("rowSpan", ui.Number(rowSpan))
		prop.Set("columnSpan", ui.Number(columnSpan))
		prop.Set("minGridHeight", ui.Number(minGridHeight))
		prop.Set("minGridWidth", ui.Number(minGridWidth))
		prop.Set("focus", ui.Bool(focus))
		e.SetUI("grid", prop.Commit())
		return e
	}
}

// ModalElement
type ModalElement struct {
	*ui.Element
}

func (e ModalElement) NativeElement() *tview.Modal {
	return e.AsElement().Native.(NativeElement).Value.(Modal).v
}

func (e ModalElement) UnderlyingBox() BoxElement {
	box := document.GetElementById(e.AsElement().ID + "-box")
	if box != nil {
		return BoxElement{box}
	}

	b := document.Box.WithID(e.AsElement().ID + "-box")
	b.AsElement().Native = NewNativeElementWrapper(e.NativeElement().Box)
	return b
}

var newModal = Elements.NewConstructor("modal", func(id string) *ui.Element {

	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewModal())
	return e
}, AllowDataStorage)

type modalConstructor func() ModalElement

func (c modalConstructor) WithID(id string, options ...string) ModalElement {
	e := ModalElement{newModal(id, options...)}
	return e
}

type modalModifier struct{}

var ModalModifier modalModifier

// Given the API for tview.List , we 'd like to generate the modifiers
// for the corresponding ModalElement i.e. modalModifier.
// It should follow the same patterns as the other modifiers, especially
// wrt method signatures and naming scheme. A lot of the code is similar to what has already been written.
// Usually, the modifiers are the Setter methods of the corresponding tview type.
// except for the callback accepting methods such SetDoneFunc.
// Namely: AddButtons, ClearButtons, SetBackgroundColor, SetButtonActivatedStyle, SetButtonBackgroundColor, SetButtonStyle, SetButtonTextColor, SetFocus, SetText and finally SetTextColor.

func (m modalModifier) Buttons(buttons ...string) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Modal).v.AddButtons(buttons)
		return e
	}
}

func (m modalModifier) ClearButtons() func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Modal).v.ClearButtons()
		return e
	}
}

func (m modalModifier) BackgroundColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Modal).v.SetBackgroundColor(color)
		return e
	}
}

func (m modalModifier) ButtonActivatedStyle(style tcell.Style) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Modal).v.SetButtonActivatedStyle(style)
		return e
	}
}

func (m modalModifier) ButtonBackgroundColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Modal).v.SetButtonBackgroundColor(color)
		return e
	}
}

func (m modalModifier) ButtonStyle(style tcell.Style) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Modal).v.SetButtonStyle(style)
		return e
	}
}

func (m modalModifier) ButtonTextColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Modal).v.SetButtonTextColor(color)
		return e
	}
}

func (m modalModifier) Focus(index int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Modal).v.SetFocus(index)
		return e
	}
}

func (m modalModifier) Text(text string) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Modal).v.SetText(text)
		return e
	}
}

func (m modalModifier) TextColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Modal).v.SetTextColor(color)
		return e
	}
}

func (m modalModifier) FlexItem(fixedsize int, proportion int, focus bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		prop := ui.NewObject()
		prop.Set("fixedsize", ui.Number(fixedsize))
		prop.Set("proportion", ui.Number(proportion))
		prop.Set("focus", ui.Bool(focus))
		e.SetUI("flex", prop.Commit())
		return e
	}
}

func (m modalModifier) GridItem(row, column int, rowSpan, columnSpan int, minGridHeight, minGridWidth int, focus bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		prop := ui.NewObject()
		prop.Set("row", ui.Number(row))
		prop.Set("column", ui.Number(column))
		prop.Set("rowSpan", ui.Number(rowSpan))
		prop.Set("columnSpan", ui.Number(columnSpan))
		prop.Set("minGridHeight", ui.Number(minGridHeight))
		prop.Set("minGridWidth", ui.Number(minGridWidth))
		prop.Set("focus", ui.Bool(focus))
		e.SetUI("grid", prop.Commit())
		return e
	}
}

// PagesElement
type PagesElement struct {
	*ui.Element
}

func (e PagesElement) NativeElement() *tview.Pages {
	return e.AsElement().Native.(NativeElement).Value.(Pages).v
}

func (e PagesElement) UnderlyingBox() BoxElement {
	box := document.GetElementById(e.AsElement().ID + "-box")
	if box != nil {
		return BoxElement{box}
	}

	b := document.Box.WithID(e.AsElement().ID + "-box")
	b.AsElement().Native = NewNativeElementWrapper(e.NativeElement().Box)
	return b
}

type pagesModifier struct{}

var PagesModifier pagesModifier

func (m pagesModifier) AsPagesElement(e *ui.Element) PagesElement {
	return PagesElement{e}
}

func (m pagesModifier) AddPage(name string, elements ...*ui.Element) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		b := document.Flex.WithID(e.ID + "-pg-" + name)
		page := b.AsElement().SetChildren(elements...)
		ui.NewViewElement(e, ui.NewView(name, page))
		p := PagesElement{e}
		p.NativeElement().AddPage(name, b.NativeElement(), true, false)

		ui.ViewElement{e}.OnActivated(name, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			p := PagesElement{evt.Origin()}.NativeElement()
			pname := string(evt.NewValue().(ui.String))

			if p.HasPage(pname) {
				p.SwitchToPage(pname)
			} else {
				if !p.HasPage("pagenotfound -- SYSERR") {
					p.AddAndSwitchToPage("pagenotfound -- SYSERR", tview.NewBox().SetBorder(true).SetTitle("Page Not Found -- SYSERR"), true)
				} else {
					p.SwitchToPage("pagenotfound -- SYSERR")
				}
			}

			return false
		}))
		return e
	}
}

func (m pagesModifier) FlexItem(fixedsize int, proportion int, focus bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		prop := ui.NewObject()
		prop.Set("fixedsize", ui.Number(fixedsize))
		prop.Set("proportion", ui.Number(proportion))
		prop.Set("focus", ui.Bool(focus))
		e.SetUI("flex", prop.Commit())
		return e
	}
}

func (m pagesModifier) GridItem(row, column int, rowSpan, columnSpan int, minGridHeight, minGridWidth int, focus bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		prop := ui.NewObject()
		prop.Set("row", ui.Number(row))
		prop.Set("column", ui.Number(column))
		prop.Set("rowSpan", ui.Number(rowSpan))
		prop.Set("columnSpan", ui.Number(columnSpan))
		prop.Set("minGridHeight", ui.Number(minGridHeight))
		prop.Set("minGridWidth", ui.Number(minGridWidth))
		prop.Set("focus", ui.Bool(focus))
		e.SetUI("grid", prop.Commit())
		return e
	}
}

var newPages = Elements.NewConstructor("pages", func(id string) *ui.Element {

	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewPages())
	return e
}, AllowDataStorage)

type pagesConstructor func() PagesElement

func (c pagesConstructor) WithID(id string, options ...string) PagesElement {
	e := PagesElement{newPages(id, options...)}

	return e
}

// TableElement
type TableElement struct {
	*ui.Element
}

func (e TableElement) NativeElement() *tview.Table {
	return e.AsElement().Native.(NativeElement).Value.(Table).v
}

func (e TableElement) UnderlyingBox() BoxElement {
	box := document.GetElementById(e.AsElement().ID + "-box")
	if box != nil {
		return BoxElement{box}
	}

	b := document.Box.WithID(e.AsElement().ID + "-box")
	b.AsElement().Native = NewNativeElementWrapper(e.NativeElement().Box)
	return b
}

func (e TableElement) NewTableCell(text string) *tview.TableCell {
	return tview.NewTableCell(text)
}

var newTable = Elements.NewConstructor("table", func(id string) *ui.Element {

	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewTable())
	return e
}, AllowDataStorage)

type tableConstructor func() TableElement

func (c tableConstructor) WithID(id string, options ...string) TableElement {
	e := TableElement{newTable(id, options...)}
	return e
}

type tableModifier struct{}

var TableModifier tableModifier

// Given the API for tview.Table , we 'd like to generate the modifiers
// for the corresponding TableElement i.e. tableModifier.
// It should follow the same patterns as the other modifiers, especially
// wrt method signatures and naming scheme. A lot of the code is similar to what has already been written.
// Usually, the modifiers are the Setter methods of the corresponding tview type.
// except for the callback accepting methods such as SetDoneFunc.
// Namely, SetBorders, SetBordersColor, SetCell, SetCellSimple, SetContent, SetEvaluateAllRows, SetFixed, SetOffset, SetSelectable, SetSelecetedStyle, SetSeparator, SetWrapSelection

func (m tableModifier) Borders(b bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Table).v.SetBorders(b)
		return e
	}
}

func (m tableModifier) BordersColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Table).v.SetBordersColor(color)
		return e
	}
}

func (m tableModifier) Cell(row, column int, cell *tview.TableCell) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Table).v.SetCell(row, column, cell)
		return e
	}
}

func (m tableModifier) CellSimple(row, column int, text string) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Table).v.SetCellSimple(row, column, text)
		return e
	}
}

func (m tableModifier) Content(content tview.TableContent) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Table).v.SetContent(content)
		return e
	}
}

func (m tableModifier) EvaluateAllRows(b bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Table).v.SetEvaluateAllRows(b)
		return e
	}
}

func (m tableModifier) Fixed(rows, columns int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Table).v.SetFixed(rows, columns)
		return e
	}
}

func (m tableModifier) Offset(row, column int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Table).v.SetOffset(row, column)
		return e
	}
}

func (m tableModifier) Selectable(rows, columns bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Table).v.SetSelectable(rows, columns)
		return e
	}
}

func (m tableModifier) SelectedStyle(style tcell.Style) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Table).v.SetSelectedStyle(style)
		return e
	}
}

func (m tableModifier) Separator(separator rune) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Table).v.SetSeparator(separator)
		return e
	}
}

func (m tableModifier) WrapSelection(vertical, horizontal bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(Table).v.SetWrapSelection(vertical, horizontal)
		return e
	}
}

// TextAreaElement
type TextAreaElement struct {
	*ui.Element
}

func (e TextAreaElement) NativeElement() *tview.TextArea {
	return e.AsElement().Native.(NativeElement).Value.(TextArea).v
}

func (e TextAreaElement) UnderlyingBox() BoxElement {
	box := document.GetElementById(e.AsElement().ID + "-box")
	if box != nil {
		return BoxElement{box}
	}

	b := document.Box.WithID(e.AsElement().ID + "-box")
	b.AsElement().Native = NewNativeElementWrapper(e.NativeElement().Box)
	return b
}

var newTextArea = Elements.NewConstructor("textarea", func(id string) *ui.Element {

	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewTextArea())
	return e
}, AllowDataStorage)

type textareaConstructor func() TextAreaElement

func (c textareaConstructor) WithID(id string, options ...string) TextAreaElement {
	e := TextAreaElement{newTextArea(id, options...)}
	return e
}

type textareaModifier struct{}

var TextAreaModifier textareaModifier

// Given the API for tview.TextArea , we 'd like to generate the modifiers
// for the corresponding TextAreaElement i.e. textareaModifier.
// It should follow the same patterns as the other modifiers, especially
// wrt method signatures and naming scheme. A lot of the code is similar to what has already been written.
// Usually, the modifiers are the Setter methods of the corresponding tview type.
// except for the callback accepting methods such as SetChangedFunc.
// Namely,SetDisabled, SetFormAttributes, SetLabel,SetLabelStyle, SetLabelWidth, SetMaxLength, SetOffset, SetPlaceholder, SetPlaceholderStyle, SetSize, SetText, SetWordWrap, SetWrap

func (m textareaModifier) Disabled(b bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TextArea).v.SetDisabled(b)
		return e
	}
}

func (m textareaModifier) FormAttributes(labelWidth int, labelColor, bgColor, fieldTextColor, fieldBgColor tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TextArea).v.SetFormAttributes(labelWidth, labelColor, bgColor, fieldTextColor, fieldBgColor)
		return e
	}
}

func (m textareaModifier) Label(label string) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TextArea).v.SetLabel(label)
		return e
	}
}

func (m textareaModifier) LabelStyle(style tcell.Style) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TextArea).v.SetLabelStyle(style)
		return e
	}
}

func (m textareaModifier) LabelWidth(width int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TextArea).v.SetLabelWidth(width)
		return e
	}
}

func (m textareaModifier) MaxLength(length int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TextArea).v.SetMaxLength(length)
		return e
	}
}

func (m textareaModifier) Offset(row, column int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TextArea).v.SetOffset(row, column)
		return e
	}
}

func (m textareaModifier) Placeholder(text string) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TextArea).v.SetPlaceholder(text)
		return e
	}
}

func (m textareaModifier) PlaceholderStyle(style tcell.Style) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TextArea).v.SetPlaceholderStyle(style)
		return e
	}
}

func (m textareaModifier) Size(rows, columns int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TextArea).v.SetSize(rows, columns)
		return e
	}
}

func (m textareaModifier) Text(text string, cursorAtTheEnd bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TextArea).v.SetText(text, cursorAtTheEnd)
		return e
	}
}

func (m textareaModifier) TextStyle(style tcell.Style) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TextArea).v.SetTextStyle(style)
		return e
	}
}

func (m textareaModifier) WordWrap(b bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TextArea).v.SetWordWrap(b)
		return e
	}
}

func (m textareaModifier) Wrap(b bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TextArea).v.SetWrap(b)
		return e
	}
}

// TextViewElement
type TextViewElement struct {
	*ui.Element
}

func (e TextViewElement) NativeElement() *tview.TextView {
	return e.AsElement().Native.(NativeElement).Value.(TextView).v
}

func (e TextViewElement) UnderlyingBox() BoxElement {
	box := document.GetElementById(e.AsElement().ID + "-box")
	if box != nil {
		return BoxElement{box}
	}

	b := document.Box.WithID(e.AsElement().ID + "-box")
	b.AsElement().Native = NewNativeElementWrapper(e.NativeElement().Box)
	return b
}

var newTextView = Elements.NewConstructor("textview", func(id string) *ui.Element {

	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewTextView())

	// TODO think about calling Draw OnMounted

	return e
}, AllowDataStorage)

type textviewConstructor func() TextViewElement

func (c textviewConstructor) WithID(id string, options ...string) TextViewElement {
	e := TextViewElement{newTextView(id, options...)}

	return e
}

type textviewModifier struct{}

var TextViewModifier textviewModifier

// Given the API for tview.TextView , we 'd like to generate the modifiers
// for the corresponding TextViewElement i.e. textviewModifier.
// It should follow the same patterns as the other modifiers, especially
// wrt method signatures and naming scheme. A lot of the code is similar to what has already been written.
// Usually, the modifiers are the Setter methods of the corresponding tview type.
// except for the callback accepting methods such as SetChangedFunc.
// Namely, SetBackgroundColor, SetDynamicColors, SetFormAttributes,  SetLabel, SetLabelWidth, SetMaxLines, SetRegions, SetScrollable, SetSize, SetTet, SetTextAlign, SetTextColor, SetTextStyle, SetToggleHighlights, SetWordWrap, SetWrap

func (m textviewModifier) BackgroundColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TextView).v.SetBackgroundColor(color)
		return e
	}
}

func (m textviewModifier) DynamicColors(b bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TextView).v.SetDynamicColors(b)
		return e
	}
}

func (m textviewModifier) FormAttributes(labelWidth int, labelColor, bgColor, fieldTextColor, fieldBgColor tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TextView).v.SetFormAttributes(labelWidth, labelColor, bgColor, fieldTextColor, fieldBgColor)
		return e
	}
}

func (m textviewModifier) Label(label string) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TextView).v.SetLabel(label)
		return e
	}
}

func (m textviewModifier) LabelWidth(width int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TextView).v.SetLabelWidth(width)
		return e
	}
}

func (m textviewModifier) MaxLines(lines int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TextView).v.SetMaxLines(lines)
		return e
	}
}

func (m textviewModifier) Regions(enable bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TextView).v.SetRegions(enable)
		return e
	}
}

func (m textviewModifier) Scrollable(b bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TextView).v.SetScrollable(b)
		return e
	}
}

func (m textviewModifier) Size(rows, columns int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TextView).v.SetSize(rows, columns)
		return e
	}
}

func (m textviewModifier) Text(text string) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TextView).v.SetText(text)
		return e
	}
}

func (m textviewModifier) TextAlign(align int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TextView).v.SetTextAlign(align)
		return e
	}
}

func (m textviewModifier) TextColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TextView).v.SetTextColor(color)
		return e
	}
}

func (m textviewModifier) TextStyle(style tcell.Style) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TextView).v.SetTextStyle(style)
		return e
	}
}

func (m textviewModifier) ToggleHighlights(b bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TextView).v.SetToggleHighlights(b)
		return e
	}
}

func (m textviewModifier) WordWrap(wraponwords bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TextView).v.SetWordWrap(wraponwords)
		return e
	}
}

func (m textviewModifier) Wrap(b bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TextView).v.SetWrap(b)
		return e
	}
}

// since *tview.TextView implements io.Writer, we can have a modifier that lets us use it as the cmd.Output
func (m textviewModifier) Stdout(cmd exec.Cmd) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		cmd.Stdout = e.Native.(NativeElement).Value.(TextView).v
		return e
	}
}

// TreeViewElement
type TreeViewElement struct {
	*ui.Element
}

func (e TreeViewElement) NativeElement() *tview.TreeView {
	return e.AsElement().Native.(NativeElement).Value.(TreeView).v
}

func (e TreeViewElement) UnderlyingBox() BoxElement {
	box := document.GetElementById(e.AsElement().ID + "-box")
	if box != nil {
		return BoxElement{box}
	}

	b := document.Box.WithID(e.AsElement().ID + "-box")
	b.AsElement().Native = NewNativeElementWrapper(e.NativeElement().Box)
	return b
}

func (e TreeViewElement) NewTreeNode(text string) *tview.TreeNode {
	return tview.NewTreeNode(text)
}

var newTreeView = Elements.NewConstructor("treeview", func(id string) *ui.Element {

	e := ui.NewElement(id, Elements.DocType)
	e.Native = NewNativeElementWrapper(tview.NewTreeView())
	return e
}, AllowDataStorage)

type treeviewConstructor func() TreeViewElement

func (c treeviewConstructor) WithID(id string, options ...string) TreeViewElement {
	e := TreeViewElement{newTreeView(id, options...)}
	return e
}

type treeviewModifier struct{}

var TreeViewModifier treeviewModifier

// Given the API for tview.TreeView , we 'd like to generate the modifiers
// for the corresponding TreeViewElement i.e. treeviewModifier.
// It should follow the same patterns as the other modifiers, especially
// wrt method signatures and naming scheme. A lot of the code is similar to what has already been written.
// Usually, the modifiers are the Setter methods of the corresponding tview type.
// except for the callback accepting methods such as SetChangedFunc.
// Namely SetAlign, SetCurrentNode, SetGraphics, SetGraphicsColor, SetPrefixes,  SetRoot, SetTopLevel.

func (m treeviewModifier) Align(align bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TreeView).v.SetAlign(align)
		return e
	}
}

func (m treeviewModifier) CurrentNode(node *tview.TreeNode) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TreeView).v.SetCurrentNode(node)
		return e
	}
}

func (m treeviewModifier) Graphics(showGraphics bool) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TreeView).v.SetGraphics(showGraphics)
		return e
	}
}

func (m treeviewModifier) GraphicsColor(color tcell.Color) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TreeView).v.SetGraphicsColor(color)
		return e
	}
}

func (m treeviewModifier) Prefixes(prefixes []string) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TreeView).v.SetPrefixes(prefixes)
		return e
	}
}

func (m treeviewModifier) Root(node *tview.TreeNode) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TreeView).v.SetRoot(node)
		return e
	}
}

func (m treeviewModifier) TopLevel(level int) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Native.(NativeElement).Value.(TreeView).v.SetTopLevel(level)
		return e
	}
}
