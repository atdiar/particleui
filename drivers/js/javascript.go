Tooltip// +build js,wasm

// Package javascript defines the default set of Element constructors, native interfaces,
// events and event handlers, and animation properties used to build js-based UIs.
package javascript

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"syscall/js"

	"github.com/atdiar/particleui"
)

var (
	// DOCTYPE holds the document doctype.
	DOCTYPE = "html/js"
	// Elements stores wasm-generated HTML ui.Element constructors.
	Elements           = ui.NewElementStore("default",DOCTYPE)
	EventTable         = NewEventTranslationTable()
	DefaultWindowTitle = "Powered by ParticleUI"
)

// MutationCaptureMode describes how a Go App may capture textarea value changes
// that happen in native javascript. For instance, when a blur event is dispatched
// or when any mutation is observed via the MutationObserver API.
type MutationCaptureMode int

var (
	OnBlur    MutationCaptureMode = iota
	OnInput MutationCaptureMode //implemented via mutation observers
)

// Window is a ype that represents a browser window
type Window struct {
	*ui.Element
}

func (w Window) SetTitle(title string) {
	w.Set("ui", "title", title, false)
}

// TODO see if can get height width of window view port, etc.

func getWindow() Window {
	e := ui.NewElement("window", DefaultWindowTitle, DOCTYPE)
	e.Native = NewNativeElementWrapper(js.Global())

	h := ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		target := evt.Origin()
		newtitle, ok := evt.NewValue().(string)
		if !ok {
			return true
		}
		win, ok := target.Native.(Window)
		if !ok {
			return true
		}
		nat, ok := win.Element.NativeElement.(js.Wrapper)
		if !ok {
			return true
		}
		jswindow := nat.JSValue()
		jswindow.Get("document").Set("title", newtitle)
		return false
	})

	e.Watch("ui", "title", e, h)
	e.Set("ui", "title", DefaultWindowTitle, false)
	return Window{e}
}

var DefaultWindow Window = getWindow()

// NativeElement defines a wrapper around a js.Value that implements the
// ui.NativeElementWrapper interface.
type NativeElement struct {
	js.Value
}

func NewNativeElementWrapper(v js.Value) NativeElement {
	return NativeElement{v}
}

func (n NativeElement) AppendChild(child *ui.Element) {
	v, ok := child.Native.(NativeElement)
	if !ok {
		log.Print("wrong format for native element underlying objects.Cannot append " + child.Name)
		return
	}
	n.Value.Call("append", v.Value)
}

func (n NativeElement) PrependChild(child *ui.Element) {
	v, ok := child.Native.(NativeElement)
	if !ok {
		log.Print("wrong format for native element underlying objects.Cannot prepend " + child.Name)
		return
	}
	n.Value.Call("prepend", v.Value)
}

func (n NativeElement) InsertChild(child *ui.Element, index int) {
	v, ok := child.Native.(NativeElement)
	if !ok {
		log.Print("wrong format for native element underlying objects.Cannot insert " + child.Name)
		return
	}
	childlist := n.Value.Get("children")
	length := childlist.Get("length").Int()
	if index >= length {
		log.Print("insertion attempt out of bounds.")
		return
	}
	r := childlist.Call("item", index)
	n.Value.Call("insertBefore", v, r)
}

func (n NativeElement) ReplaceChild(old *ui.Element, new *ui.Element) {
	nold, ok := old.Native.(NativeElement)
	if !ok {
		log.Print("wrong format for native element underlying objects.Cannot replace " + old.Name)
		return
	}
	nnew, ok := new.Native.(NativeElement)
	if !ok {
		log.Print("wrong format for native element underlying objects.Cannot replace with " + new.Name)
		return
	}
	//nold.Call("replaceWith", nnew) also works
	n.Value.Call("replaceChild", nnew.Value, nold.Value)
}

func (n NativeElement) RemoveChild(child *ui.Element) {
	v, ok := child.Native.(NativeElement)
	if !ok {
		log.Print("wrong format for native element underlying objects.Cannot insert " + child.Name)
		return
	}
	n.JSValue().Call("removeChild", v.JSValue())
}

/*
//
//
// Element Constructors
//
//
//
*/

// TODO window should have its own type. it is not an element butits properties
// can be read and some such as title can be changed.
// Should be alos of type js.Wrapper-

// NewAppRoot creates a new app entry point. It is the top-most element
// in the tree of Elements that consitute the full document.
// It should be the element which is passed to a router to observe for route
// change.
// By default, it represents document.body. As such, it is different from the
// document which holds the head element for instance.
var NewAppRoot = Elements.NewConstructor("root", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)
	root := js.Global().Get("document").Get("body")
	n := NewNativeElementWrapper(root)
	e.Native = n
	return e
})

// NewDiv is a constructor for html div elements.
var NewDiv = Elements.NewConstructor("div", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)
	e = enableClasses(e)

	htmlDiv := js.Global().Get("document").Call("createElement", "div")
	n := NewNativeElementWrapper(htmlDiv)
	e.Native = n
	SetAttribute(e,"id",id)
	return e
}, EnableLayoutDispositionTracking)

var EnableLayoutDispositionTracking = ui.NewConstructorOption("EnableLayoutDispositionTracking",func(args ...interface{}) (func(*ui.Element)*ui.Element){
			return func(e *ui.Element) *ui.Element{
				if len(args)!= 2{
					return e
				}
				defdispo,ok := args[0].(string)
				if !ok{
					return e
				}
				muthandler,ok := args[1].(*ui.MutationHandler)
				if !ok{
					return e
				}
				e.Watch("ui", "disposition", e, muthandler)
				e.Set("ui", "disposition", defdispo, false)
				return e
			}
		})

func EnableLayoutDispositionTracking(defaultdisposition string, ondispositionchange *ui.MutationHandler) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		e.Watch("ui", "disposition", e, ondispositionchange)
		e.Set("ui", "disposition", defaultdisposition, false)
		return e
	}
}

// NewTooltip is a constructor for html div elements.
func NewTooltip(name string id string) *ui.Element{
	var TooltipConstructor = Elements.NewConstructor("tooltip", func(name string, id string) *ui.Element {
		e := ui.NewElement(name, id, Elements.DocType)
		e = enableClasses(e)

		htmlTooltip := js.Global().Get("document").Call("createElement", "div")
		n := NewNativeElementWrapper(htmlTooltip)
		e.Native = n
		SetAttribute(e,"id",id)

		h:= ui.NewMutationHanlder(func(evt ui.MutationEvent)bool{
			content,ok:= evt.NewValue().(*ui.Element)
			if ok{+
				tooltipdiv := evt.Origin()
				tooltipdiv.RemoveAll()
				tooltipdiv.AppendChild(NewSpan("tooltip-span",NewID())).AppendChild(content))
				return false
			}
			strcontent,ok := evt.NewValue().(string)
			if !ok{
				return true
			}

			tooltipdiv := evt.Origin()
			tooltipdiv.RemoveAll()
			tn:= NewTextNode()
			tn.Set("data","text", strcontent,false)
			tooltipdiv.AppendChild(NewSpan("tooltip-span",NewID()).AppendChild(tn)
			return false
		})
		e.Watch("data","content",e,h)
		return e
	})
	return TooltipConstructor(name,id)
}

func EnableTooltip(tooltipcontent interface{}) func(*ui.Element) *ui.Element{
	return func(e *ui.Element)*ui.Element{
		t:= NewTooltip()
		e.AppendChild(t)
		h:= NewMutationHandler(func(evt ui.MutationEvent)bool{
			t.Set("data","content",evt.NewValue(),false)
			return false
		})
		e.Watch("data","tooltipcontent",e,h)
		e.Set("data","tooltipcontent",tooltipcontent, false)
		return e
	}
}


// NewTextArea is a constructor for a textarea html element.
var NewTextArea = func(name string, id string, rows int, cols int, options ...func(*ui.Element)*uiu.Element) *ui.Element {
	return Elements.NewConstructor("textarea", func(ename string, eid string) *ui.Element {
		e := ui.NewElement(ename, eid, Elements.DocType)
		e = enableClasses(e)

		htmlTextArea := js.Global().get("document").Call("createElement", "textarea")


		e.Watch("data", "text", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			if s, ok := evt.NewValue().(string); ok {
				old := htmlTextArea.Get("value").String()
				if s != old {
					SetAttribute("value", s)
				}
			}
			return false
		}))


		n := NativeElementWrapper(htmlTextArea)
		e.Native = n
		SetAttribute(e,"name", ename)
		SetAttribute(e,"id", eid)
		SetAttribute(e,"rows", strconv.Itoa(row))
		SetAttribute(e,"cols", strconv.Itoa(cols))
	})(name, id,options...)
}

func EnableDataBinding(datacapturemode ...MutationCaptureMode) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		callback:= ui.NewEventHandler(func(evt ui.Event)bool{
			if evt.Target().ID != e.ID{
				return false // we do not stop the event propagation but do not handle the event either
			}
			n,ok:= e.Native.(NativeElementWrapper)
			if !ok{
				return true
			}
			nn:= n.Value()
			v:= nn.Get("value")
			ok = v.Truthy()
			if !ok{
				return true
			}
			s:= v.String()
			e.Set("data",'text',s,false)
			return false
		})


		if datacapturemode == nil || len(datacapturemode)>1 {
			e.AddEventListener("blur",callback, EventTable.NativeEventBridge())
			return e
		}
		mode:= datacapturemode[0]
		if  mode == OnInput{
			e.AddEventListener("input",callback, EventTable.NativeEventBridge())
			return e
		}

		// capture textarea value on blur by default
		e.AddEventListener("blur",callback, EventTable.NativeEventBridge())
		return e
	}
}

// TODO attribute setting functions such as Placeholder(val string) func(*ui.Element) *ui.Element to implement

// NewHeader is a constructor for a html header element.
var NewHeader = Elements.NewConstructor("header", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)
	e = enableClasses(e)

	htmlHeader := js.Global().Get("document").Call("createElement", "header")
	n := NewNativeElementWrapper(htmlHeader)
	e.Native = n
	SetAttribute(e,"id", id)
	return e
})

// NewFooter is a constructor for an html footer element.
var NewFooter = Elements.NewConstructor("footer", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)
	e = enableClasses(e)

	htmlFooter := js.Global().Get("document").Call("createElement", "footer")
	n := NewNativeElementWrapper(htmlFooter)
	e.Native = n
	SetAttribute(e,"id", id)
	return e
})

// NewSpan is a constructor for html div elements.
var NewSpan = Elements.NewConstructor("span", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)
	e = enableClasses(e)

	htmlSpan := js.Global().Get("document").Call("createElement", "span")
	n := NewNativeElementWrapper(htmlSpan)
	e.Native = n
	SetAttribute(e,"id", id)
	return e
})

// NewDiv is a constructor for html div elements.
var NewParagraph = Elements.NewConstructor("paragraph", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)
	e = enableClasses(e)

	htmlParagraph := js.Global().Get("document").Call("createElement", "p")
	n := NewNativeElementWrapper(htmlParagraph)
	e.Native = n
	SetAttribute(e,"id", id)
	return e
})

// NewNavMenu is a constructor for a html nav element.
var NewNavMenu = Elements.NewConstructor("nav", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)
	e = enableClasses(e)

	htmlNavMenu := js.Global().Get("document").Call("createElement", "nav")
	n := NewNativeElementWrapper(htmlNavMenu)
	e.Native = n
	SetAttribute(e,"id", id)
	return e
})

// NewAnchor creates an html anchor element which points to the object whose id is
// being passed as argument.
// If the object does not exist, it points to itself.
var NewAnchor = Elements.NewConstructor("link", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)
	e = enableClasses(e)

	htmlAnchor := js.Global().Get("document").Call("createElement", "a")
	baseid:= id
	id =id+"-link"
	// finds the element whose id has been passed as argument: if search returns nil
	// then the Link element references itself.
	lnkTarget := Elements.GetByID(baseid)
	if lnkTarget == nil {
		lnkTarget = e
		id = baseid
	}

	// Set a mutation Handler on lnkTarget which observes the tree insertion event (attach event)
	// At each attachment, we should rewrite href with the new route.
	lnkTarget.Watch("event", "attached", lnkTarget, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		if evt.ObservedKey() != "attached" || evt.Type() != "event" || evt.Origin() != lnkTarget {
			return true
		}

		SetAttribute(e, "href", e.Route())
		return false
	}))
	n := NewNativeElementWrapper(htmlAnchor)
	e.Native = n
	SetAttribute(e,"id", id)
	return e
})

var NewButton = func(name string, id string, typ string, options ...func(*ui.Element) *ui.Element) *ui.Element {
	f := Elements.NewConstructor("button", func(elementname string, elementid string) *ui.Element {
		e := ui.NewElement(elementname, elementid, Elements.DocType)
		e = enableClasses(e)

		htmlButton := js.Global().Get("document").Call("createElement", "button")
		n := NewNativeElementWrapper(htmlButton)
		e.Native = n
		SetAttribute(e,"name", elementname)
		SetAttribute(e,"id", elementid)
		SetAttribute(e,"type", typ)
		return e
	})
	return f(name, id, options...)
}

var NewInput = func(name string, id string, typ string, options ...func(*ui.Element) *ui.Element) *ui.Element {
	f := Elements.NewConstructor("input", func(elementname string, elementid string) *ui.Element {
		e := ui.NewElement(elementname, elementid, Elements.DocType)
		e = enableClasses(e)

		htmlInput := js.Global().Get("document").Call("createElement", "input")

		n := NewNativeElementWrapper(htmlInput)
		e.Native = n
		SetAttribute(e,"name", elementname)
		SetAttribute(e,"id", elementid)
		SetAttribute(e,"type", typ)
		return e
	})
	return f(name, id, options...)
}

var NewImage = func(src string, id string, altname string, options ...func(*ui.Element) *ui.Element) *ui.Element {
	return Elements.NewConstructor("image", func(name string, imgid string) *ui.Element {
		e := ui.NewElement(name, imgid, Elements.DocType)
		e = enableClasses(e)

		htmlImg := js.Global().Get("document").Call("createElement", "img")

		n := NewNativeElementWrapper(htmlImg)
		e.Native = n
		SetAttribute(e,"src", src)
		SetAttribute(e,"alt", name)
		SetAttribute(e,"id", imgid)
		return e
	})(altname, id, options...)
}

var NewAudio = Elements.NewConstructor("audio", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)
	e = enableClasses(e)

	htmlAudio := js.Global().Get("document").Call("createElement", "audio")

	n := NewNativeElementWrapper(htmlAudio)
	e.Native = n
	SetAttribute(e,"name", name)
	SetAttribute(e,"id", id)
	return e
})

var NewVideo = Elements.NewConstructor("video", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)
	e = enableClasses(e)

	htmlVideo := js.Global().Get("document").Call("createElement", "video")
	SetAttribute(e,"name", name)
	SetAttribute(e,"id", id)

	n := NewNativeElementWrapper(htmlVideo)
	e.Native = n
	return e
})

var NewMediaSource = func(src string, typ string, options ...func(*ui.Element)*uiu.Element) *ui.Element {
	return Elements.NewConstructor("source", func(name string, id string) *ui.Element {
		e := ui.NewElement(name, id, Elements.DocType)
		e = enableClasses(e)

		htmlVideo := js.Global().Get("document").Call("createElement", "video")

		n := NewNativeElementWrapper(htmlVideo)
		e.Native = n
		SetAttribute(e,"type", name)
		SetAttribute(e,"src", id)
		return e
	})(typ, src,options...)
}

func WithSources(sources ...*ui.Element) func(*ui.Element) *ui.Element {
	return func(mediaplayer *ui.Element) *ui.Element {
		for _, source := range sources {
			if source.Name != "source" {
				log.Print("cannot append non media source element to mediaplayer")
				continue
			}
			mediaplayer.AppendChild(source)
		}
		return mediaplayer
	}
}

// NewTextNode creates a text node for the Element whose id is passed as argument
// The id for the text Element is the id of its parent to which
// is suffixed "-txt-" and a random number.
// If the parent does not exist, a parent span is created whose id is the one
// passed as argument.
// To change the value of the text, one would Set the "text" property belonging
// to the "data" category/namespace. i.e. Set("data","text",value)
func NewTextNode() *ui.Element{
	var TextNode = Elements.NewConstructor("text", func(name string, id string) *ui.Element {
		e := ui.NewElement(name, id Elements.DocType)
		htmlTextNode := js.Global().Get("document").Call("createTextNode", "")
		n := NewNativeElementWrapper(htmlTextNode)
		e.Native = n

		e.Watch("data", "text", e, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			if s, ok := evt.NewValue().(string); ok { // if data.text is deleted, nothing happens, so no check for nil of  evt.NewValue()
				htmlTextNode.Set("nodeValue", s)
			}
			return false
		}))

		return e
	})
	return TextNode("textnode",NewID())
}



// NewTemplatedText returns either a textnode appended to the Element whose id
// is passed as argument, or a div wrapping a textnode if no ui.Element exists
// yet for the id.
// The template accepts a parameterized string as would be accepted by fmt.Sprint
// and the parameter should have their names passed as arguments.
// Done correctly, calling element.Set("data", paramname, stringvalue) will
// set the textnode with a new string value where the parameter whose name is
// `paramname` is set with the value `stringvalue`.
var NewTemplatedText = func(name string, id string, format string, paramsNames ...string) *ui.Element {
	nt := NewTextNode()

	formatter := func(tplt string, params ...string) string {
		v := make([]interface{}, len(params))
		for i, p := range params {
			val, ok := nt.Get("data", p)
			if ok {
				v[i] = val
			}
			continue
		}
		return fmt.Sprintf(tplt, v...)
	}
	params := make([]string, len(paramsNames))
	for i, p := range paramsNames {
		nt.Watch("data", p, nt, ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
			s, ok := evt.NewValue().(string)
			if ok {
				params[i] = s
			} else {
				params[i] = "???"
			}

			nt.Set("data", "text", formatter(format, params...), false)
			return false
		}))
	}
	return nt
}

var NewList = func(name string, id string,options ...func(*ui.Element)*uiu.Element) *ui.Element {
	elname := "ul"
	return Elements.NewConstructor(elname, func(ename, eid string) *ui.Element {
		e := ui.NewElement(ename, eid, Elements.DocType)
		e = enableClasses(e)

		htmlList := js.Global().Get("document").Call("createElement", elname)

		n := NewNativeElementWrapper(htmlList)
		e.Native = n
		SetAttribute(e,"name", ename)
		SetAttribute(e,"id", eid)
		return e
	})(name,id, options...)
}

var NewOrderedList = func(name string, id string, typ string, numberingstart int,options ...func(*ui.Element)*uiu.Element) *ui.Element {
	elname := "ol"
	return Elements.NewConstructor(elname, func(ename, eid string) *ui.Element {
		e := ui.NewElement(ename, eid, Elements.DocType)
		e = enableClasses(e)

		htmlList := js.Global().Get("document").Call("createElement", elname)

		n := NewNativeElementWrapper(htmlList)
		e.Native = n
		SetAttribute(e,"name", ename)
		SetAttribute(e,"id", eid)
		SetAttribute(e,"type", typ)
		SetAttribute(e,"start", strconv.Itoa(numberingstart))
		return e
	})(name,id,options...)
}

var NewListItem = Elements.NewConstructor("listitem", func(name string, id string) *ui.Element {
	e := ui.NewElement(name, id, Elements.DocType)
	e = enableClasses(e)

	htmlListItem := js.Global().Get("document").Call("createElement", "li")

	n := NewNativeElementWrapper(htmlListItem)
	e.Native = n
	SetAttribute(e,"name", name)
	SetAttribute(e,"id", id) // TODO define attribute setters optional functions

	ondatamutation:= ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		cat,ok:= evt.Type().(string)
		if !ok{
			return true
		}
		if cat != "data"{
			return false
		}
		propname,ok:= evt.ObservedKey().(string)
		if !ok{
			return true
		}
		if propname!= "content"{
			return false
		}
		evt.Origin().Set("ui",propname, evt.NewValue(),false)
		return false
	})
	e.Watch("data","content",e,ondatamutation)

	onuimutation = ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		cat,ok:= evt.Type().(string)
		if !ok{
			return true
		}
		if cat != "ui"{
			return true
		}
		propname,ok:= evt.ObservedKey().(string)
		if !ok{
			return true
		}
		if propname!= "content"{
			return true
		}
		// we apply the modifications to the UI
		v:= evt.NewValue()
		item,ok:=v.(*ui.Element)
		if !ok{
			str,ok:= v.(string)
			if !ok{
				return true
			}
			item = NewTextNode()
			item.Set("data","text",str,false)
		}
		evt.Origin().RemoveChildren().AppendChild(item)
		return false
	})
	e.Watch("ui","content",e,onuimutation)
	return e
})

type listValue struct{
	Index int
	Value interface{}
}

func newListValue(index int, value interface{}) listValue{
	return listValue{index,value}
}

func ListMutation(v interface{}) (index int, newvalue interface{},ok bool){
	res,ok:= v.(listValue)
	return res.Index, res.Value, ok
}


func ListAppend(list *ui.Element, values ...interface{}) *ui.Element{
	backinglist,ok := list.Get("internals",list.Name).([]interface{})
	if !ok{
		backinglist= make([]interface{},0)
	}
	length:= len(backinglist)

	backinglist = append(backinglist,values...)
	list.Set("internals",list.Name,backinglist,false)
	for i,value:= range values{
		list.Set(list.Name, "append",newListValue(i+length,value),false)
	}
	return list
}

func ListPrepend(list *ui.Element, values ...interface{}) *ui.Element{
	backinglist,ok := list.Get("internals",list.Name).([]interface{})
	if !ok{
		backinglist= make([]interface{},0)
	}
	length:= len(backinglist)

	backinglist = append(values,backinglist...)
	list.Set("internals",list.Name,backinglist,false)
	for i:= len(values)-1, i>=0, i--{
		list.Set(list.Name, "prepend",newListValue(i,values[i]),false)
	}
	return list
}

func ListInsertAt(list *ui.Element, offset int, values ...interface{}) *ui.Element{
	backinglist,ok := list.Get("internals",list.Name).([]interface{})
	if !ok{
		backinglist= make([]interface{},0)
	}
	length:= len(backinglist)
	if offset >= length || offset <=0{
		log.Print("Cannot insert element in list at that position.")
		return list
	}

	nel := make([]interface{}, 0)
	nel = append(nel, backinglist[:offset]...)
	nel = append(nel, values...)
	nel = append(nel, backinglist[offset:]...)
	backinglist = nel
	list.Set("internals",list.Name,backinglist,false)
	for i,value:= range values{
		list.Set(list.Name, "insert",newListValue(offset+i,value),false)
	}
	return list
}

func ListDelete(list *ui.Element,offset int)*ui.Element{
	backinglist,ok := list.Get("internals",list.Name).([]interface{})
	if !ok{
		return list
	}
	length:= len(backinglist)
	if offset >= length || offset <=0{
		log.Print("Cannot insert element in list at that position.")
		return list
	}
	backinglist = append(backinglist[:offset], backinglist[offset+1:])
	list.Set("internals",list.Name,backinglist,false)
	list.Set(list.Name,"delete",newListValue(offset,nil),false)
	return list
}

// AutoSyncList enables to set a mutation handler which is called each time
// a change occurs in the chosen namespace/category of a list Element.
func AutoSyncList() func(*ui.Element)*ui.Element{
	h:= ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		i,v,ok:= ListMutation(evt.NewValue())
		if !ok{
			return true
		}

		if evt.ObservedKey() == "append"{
			id:= NewID()
			n:= NewListItem(evt.Origin().Name+"-item",id)
			item,ok:=v.(*ui.Element)
			if !ok{
				str,ok:= v.(string)
				if !ok{
					return true
				}
			  item = NewTextNode()
				item.Set("data","text",str,false)
			}
			n.Set("data","content",item, false)

			evt.Origin().AppendChild(n)
		}

		if evt.ObservedKey() == "prepend"{
			id:= NewId()
			n:= NewListItem(evt.Origin().Name+"-item",id)
			item,ok:=v.(*ui.Element)
			if !ok{
				str,ok:= v.(string)
				if !ok{
					return true
				}
			  item = NewTextNode()
				item.Set("data","text",str,false)
			}
			n.Set("data","content",item, false)

			evt.Origin().PrependChild(n)
		}

		if evt.ObservedKey() == "insert"{
			id:= NewId()
			n:= NewListItem(evt.Origin().Name+"-item",id)
			item,ok:=v.(*ui.Element)
			if !ok{
				str,ok:= v.(string)
				if !ok{
					return true
				}
			  item = NewTextNode()
				item.Set("data","text",str,false)
			}
			n.Set("data","content",item, false)

			evt.Origin().InsertChild(n,i)
		}

		return false
	})

	if evt.ObservedKey() == "delete"{
		target := evt.Origin()
		deletee:= target.Children.AtIndex(i)
		if deletee != nil{
			target.RemoveChild(deletee)
		}
	}

	return func(e*ui.Element) *ui.Element{
		e.WatchGroup(e.Name,h)
		return e
	}
}

var NewID = ui.NewIDgenerator(5681325)

// StopAutoSyncList interrupts the monitoring of changes made to a list
// ui.Element for the given category. The list will no longer react to changes on
// itself.
func StopAutoSyncList(list *ui.Element){
	list.UnwatchGroup(list.Name, list)
}

type EventTranslationTable struct {
	FromJS          map[string]func(evt js.Value) ui.Event
	ToJS            map[string]func(evt ui.Event) js.Value
	nameTranslation map[nameTranslation]string
}

type nameTranslation struct {
	Event  string
	Native bool
}

func translationKey(evtname string, js bool) nameTranslation {
	return nameTranslation{evtname, js}
}

func NewEventTranslationTable() EventTranslationTable {
	return EventTranslationTable{make(map[string]func(evt js.Value) ui.Event), make(map[string]func(evt ui.Event) js.Value), make(map[nameTranslation]string)}
}

// Register enables the storage of an event translation function which is used
// by ui.Element to listen to events that are actually dispatched from the
// underlying javascript target.
func (e EventTranslationTable) GoEventTranslator(goEventName string, nativeEventName string, toJS func(ui.Event) js.Value) {
	e.ToJS[goEventName] = toJS
	e.nameTranslation[translationKey(goEventName, false)] = nativeEventName
}

func (e EventTranslationTable) JSEventTranslator(nativeEventName string, goEventName string, fromJS func(js.Value) ui.Event) {
	e.FromJS[nativeEventName] = fromJS
	e.nameTranslation[translationKey(nativeEventName, true)] = goEventName
}

func (e EventTranslationTable) TranslateEventName(evt string, jsNative bool) string {
	res, ok := e.nameTranslation[translationKey(evt, jsNative)]
	if !ok {
		return evt
	}
	return res
}

func (e EventTranslationTable) NativeEventBridge() ui.NativeEventBridge {
	return func(evt string, target *ui.Element) {
		translate, ok := e.FromJS[evt]
		NativeEventName := e.nameTranslation[translationKey(evt, false)]
		if !ok {
			translate = DefaultJSEventTranslator
			NativeEventName = evt
		}
		// Let's create the callback that will be called from the js side
		cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			nativeEvent := args[0]
			nativeEvent.Call("stopPropagation")
			goevt := translate(nativeEvent)
			target.DispatchEvent(goevt, nil)
			return nil
		})

			js.Global().Get("document").Call("getElementById", target.ID).Call("addEventListener", NativeEventName, cb)
			if target.NativeEventUnlisteners.List == nil {
				target.NativeEventUnlisteners = ui.NewNativeEventUnlisteners()
			}
			target.NativeEventUnlisteners.Add(NativeEventName, func() {
				js.Global().Get("document").Call("getElementById", target.ID).Call("removeEventListener", NativeEventName, cb)
				cb.Release()
			})
	}
}

func (e EventTranslationTable) NativeDispatcher() ui.NativeDispatch {
	return func(evt ui.Event) {
		translate, ok := e.ToJS[evt.Type()]
		if !ok {
			translate = DefaultGoEventTranslator
		}
		nativeevent := translate(evt)
		nelmt, ok := evt.Target().Native.(NativeElement)
		if !ok {
			log.Print("Unable to dispatch event for non-javascript html element")
			return
		}
		nelmt.Value.Call("dispatchEvent", nativeevent)
	}
}

func (e EventTranslationTable) EventFromJS(evt js.Value) ui.Event {
	typ := evt.Get("type").String()
	translate, ok := e.FromJS[typ]
	if !ok {
		translate = DefaultJSEventTranslator
	}
	return translate(evt)
}

func (e EventTranslationTable) EventToJS(evt ui.Event) js.Wrapper {
	translate, ok := e.ToJS[evt.Type()]
	if !ok {
		translate = DefaultGoEventTranslator
	}
	return translate(evt)
}

func AddClass(target *ui.Element, classname string) {
	category := "css"
	classes, ok := target.Get(category, "class")
	if ok {
		c, ok := classes.(string)
		if !ok {
			target.Set(category, "class", classname, false)
			return
		}
		if !strings.Contains(c, classname) {
			c = c + " " + classname
			target.Set(category, "class", c, false)
		}
		return
	}
	target.Set(category, "class", classname, false)
}

func RemoveClass(target *ui.Element, classname string) {
	category := "css"
	classes, ok := target.Get(category, "class")
	if !ok {
		return
	}
	c, ok := classes.(string)
	if !ok {
		return
	}
	c = strings.TrimPrefix(c, classname)
	c = strings.TrimPrefix(c, " ")
	c = strings.ReplaceAll(c, classname+" ", " ")
	target.Set(category, "class", c, false)
}

func Classes(target *ui.Element) []string {
	category := "css"
	classes, ok := target.Get(category, "class")
	if !ok {
		return nil
	}
	c, ok := classes.(string)
	if !ok {
		return nil
	}
	return strings.Split(c, " ")
}

func enableClasses(e *ui.Element) *ui.Element {
	h := ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		target := evt.Origin()
		native, ok := target.Native.(NativeElement)
		if !ok {
			log.Print("wrong type for native element or native element does not exist")
			return true
		}
		classes, ok := evt.NewValue().(string)
		if !ok {
			log.Print("new value of non-string type. Unable to use as css class(es)")
			return true
		}
		native.Value.Call("setAttribute","class", classes)
		return false
	})
	e.Watch("css", "class", e, h)
	return e
}

// TODO check that the string is well formatted style
func SetInlineCSS(target *ui.Element, str string) {
	SetAttribute(target, "style", str)
}

func GetInlineCSS(target *ui.Element) string {
	return GetAttribute(target, "style")
}

func AppendInlineCSS(target *ui.Element, str string) {
	css := GetInlineCSS(target)
	css = css + str
	SetInlineCSS(target, css)
}

func GetAttribute(target *ui.Element, name string) string {
	native, ok := target.Native.(NativeElement)
	if !ok {
		log.Print("Cannot retrieve Attribute on non-expected wrapper type")
		return ""
	}
	return native.Value.Call("getAttribute", "name").String()
}

func SetAttribute(target *ui.Element, name string, value string) {
	target.Set("attrs", name, value, false)
	native, ok := target.Native.(NativeElement)
	if !ok {
		log.Print("Cannot set Attribute on non-expected wrapper type")
		return
	}
	native.Value.Call("setAttribute", name, value)
}

func RemoveAttribute(target *ui.Element, name string) {
	target.Delete("attrs", name)
	native, ok := target.Native.(NativeElement)
	if !ok {
		log.Print("Cannot delete Attribute using non-expected wrapper type")
		return
	}
	native.Value.Call("removeAttribute", name)
}
