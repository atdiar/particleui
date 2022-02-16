package main

import (
	"log"

	"github.com/atdiar/particleui"
	"github.com/atdiar/particleui/drivers/js"
	"strconv"
)

// GOOS=js GOARCH=wasm go build -o  ../../app.wasm


func main() {
	// 1. Create a new document
	root2 := doc.NewDocument("TestAppID2")

	// 2. Create an Input box that will  allow to create new todos
	todosinput := doc.NewInput("text", "todo", "newtodo", doc.EnableSessionPersistence())
	doc.SetAttribute(todosinput.Element(), "placeholder", "What needs to be done?")
	doc.SetAttribute(todosinput.Element(), "autofocus", "")
	doc.SetAttribute(todosinput.Element(), "onfocus", "this.value=''")
	root2.Element().AppendChild(todosinput.Element())

	// 3. TODO definition
	type Todo = ui.Object
	NewTodo := func(title ui.String) Todo {
		o := ui.NewObject()
		o.Set("id", ui.String(doc.NewID()))
		o.Set("completed", ui.Bool(false))
		o.Set("title", ui.String(title))
		return o
	}

	// 4. List
	l := doc.NewUnorderedList("todoslist", "todoslist", doc.EnableSessionPersistence())
	root2.Element().AppendChild(l.Element())

	// 5. Handle list change, for instance, on new todo insertion
	h := ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		list, ok := evt.NewValue().(ui.List)
		if !ok {
			return true
		}

		for i, v := range list {
			// Let's get each todo
			o, ok := v.(ui.Object)
			if !ok {
				continue
			}
			rv, ok := o.Get("title")
			if !ok {
				continue
			}
			v, ok := rv.(ui.String)
			if !ok {
				continue
			}

			item := doc.Elements.GetByID(evt.Origin().ID + "-item-" + strconv.Itoa(i))
			if item != nil {
				doc.ListItem{item}.SetValue(v) // tooo insert an element instead
			} else {
				item = doc.NewListItem(evt.Origin().Name+"-item", evt.Origin().ID+"-item-"+strconv.Itoa(i)).SetValue(v).Element()
			}

			evt.Origin().Mutate(ui.AppendChildCommand(item.Element()))
		}
		return false
	})
	l.Element().Watch("ui", "todoslist", l.Element(), h)

	// 6. Watch for new todos to insert
	root2.Element().Watch("data", "newtodo", todosinput.Element(), ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		var tdl ui.List
		todoslist, ok := l.Element().Get("data", "todoslist")
		if !ok {
			tdl = ui.NewList()
		}
		tdl, ok = todoslist.(ui.List)
		if !ok {
			tdl = ui.NewList()
		}

		s, ok := evt.NewValue().(ui.String)
		if !ok || s == "" {
			return true
		}
		if s != "" {
			t := NewTodo(s)
			tdl = append(tdl, t)
			log.Print(tdl)
			l.Element().SetDataSyncUI("todoslist", tdl) // todo SetData only completed require another step
		}

		return false
	}))

	// UI event handlers
	todosinput.Element().AddEventListener("change", ui.NewEventHandler(func(evt ui.Event) bool {
		s := ui.String(evt.Value())
		todosinput.Element().SyncUISetData("value", s)
		return false
	}), doc.NativeEventBridge)

	todosinput.Element().AddEventListener("keyup", ui.NewEventHandler(func(evt ui.Event) bool {
		if evt.Value() == "Enter" {
			evt.PreventDefault()
			if todosinput.Value() != ""{
				todosinput.Element().SyncUISetData("newtodo", todosinput.Value())
			}
			todosinput.Element().SyncUISetData("value", ui.String(""))
			todosinput.Blur()
		}
		return false
	}), doc.NativeEventBridge)

	c := make(chan struct{}, 0)
	<-c
}


/*

func main() {
	// 1. Create a new document
	root2 := doc.NewDocument("test2", "TestAppID2")

	// 2. Create an Input box that will  allow to create new todos
	todosinput := doc.NewInput("text", "todos", "todos", doc.EnableSessionPersistence())
	doc.SetAttribute(todosinput.Element(), "placeholder", "Enter some text...")
	root2.Element().AppendChild(todosinput.Element())

	// 4. List
	l := doc.NewUnorderedList("todoslist", "todoslist",doc.EnableSessionPersistence())


	root2.Element().AppendChild(l.Element())

	root2.Element().Watch("data", "newtodo", todosinput.Element(), ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		v:= l.Values()
		log.Println(len(v),v)
		v= append(v,evt.NewValue())
		l.FromValues(v...)


		return false
	}))

	todosinput.Element().AddEventListener("change", ui.NewEventHandler(func(evt ui.Event) bool {
		s := ui.String(evt.Value())
		todosinput.Element().SyncUISetData("value", s)
		return false
	}), doc.NativeEventBridge)

	todosinput.Element().AddEventListener("keyup", ui.NewEventHandler(func(evt ui.Event) bool {
		if evt.Value() == "Enter" {
			evt.PreventDefault()

			todosinput.Element().SyncUISetData("newtodo", todosinput.Value())

			todosinput.Blur()
			//log.Println(todosinput.Value())

		}

		return false
	}), doc.NativeEventBridge)

	v, ok := todosinput.Element().GetData("newtodo")
	log.Println(v, ok)

	c := make(chan struct{}, 0)
	<-c
}

*/

/*

func main() {
	root2 := doc.NewDocument("test2", "TestAppID2")
	todosinput := doc.NewInput("text", "todos", "todos")
	todosinput.Element().AddEventListener("keypress", ui.NewEventHandler(func(evt ui.Event) bool {
		if evt.Value() == "enter" {
			evt.PreventDefault()
		}
		native, ok := evt.Native().(doc.NativeElement)
		if !ok {
			panic("native element should be of doc.NativeELement type")
		}
		native.Value.Call("blur")
		todosinput.Element().SetData("newtodo", todosinput.Value())
		return false
	}), doc.NativeEventBridge)

	root2.Element().AppendChild(todosinput.Element())
}
*/
/*
func main() {

	root := doc.NewDocument("test3", "TestAppID")

	rd := doc.NewDiv("test", "rootview") //.SetText("This is the view at initialization...")

	rd1 := doc.NewDiv("test", "d1").SetText("x top View A")
	rd2 := doc.NewDiv("test", "d2").SetText("x top View B")
	view1 := ui.NewView("view1", rd1.Element())
	view2 := ui.NewView("view2", rd2.Element())
	v := ui.NewViewElement(rd.Element(), view1, view2)

	rd3 := doc.NewDiv("test", "d3")
	rd4 := doc.NewDiv("test", "d4").SetText("xxxx    nested viewA")
	rd5 := doc.NewDiv("test", "d5").SetText("xxxx    nested viewB")
	rd6 := doc.NewDiv("test", "d6").SetText("xxxx    nested viewC")
	view4 := ui.NewView("nested1", rd4.Element())
	view5 := ui.NewView("nested2", rd5.Element())
	view6 := ui.NewView("nested3", rd6.Element())
	v2 := ui.NewViewElement(rd3.Element(), view4, view5, view6)

	rd2.Element().AppendChild(v2.Element())

	// By construction v2 is nested in v

	root.Element().AppendChild(rd.Element())

	router := ui.NewRouter("/", v)
	nd := doc.NewDiv("notfound", "divnotfound").SetText("notfound")
	router.OnNotfound(ui.NewView("notfound", nd.Element()))

	n := 0

	eh := ui.NewEventHandler(func(evt ui.Event) bool {
		n++
		//router.GoTo("/test"+ strconv.Itoa(n%3+1)+"/nested" + strconv.Itoa(n%2+1))
		router.GoTo("/view2/d3/nested" + strconv.Itoa(n%3+1))
		return false
	})

	root.Element().AddEventListener("click", eh, doc.NativeEventBridge)

	router.ListenAndServe("popstate", doc.GetWindow().Element(), doc.NativeEventBridge)

	c := make(chan struct{}, 0)
	<-c
}

*/

/*

//=========================== Event+ View + Routing test =======================

func main() {

	root := doc.NewDocument("test", "TestAppID")

	rd:= doc.NewDiv("test","divtest") //.SetText("This is the view at initialization...")

	rd2:= doc.NewDiv("test","divtest2").SetText("this is but a test nB02...")
	rd3:= doc.NewDiv("test","divtes3t").SetText("this is but a test nB03...")

	view2:=ui.NewView("test2",rd2.Element())
	view3:= ui.NewView("test3",rd3.Element())
	v:= ui.NewViewElement(rd.Element(),view2).AddView(view3)

	root.Element().AppendChild(v.Element())

	router := ui.NewRouter("/",v)
	log.Println("v route: ",v.Element().Route())
	nd:= doc.NewDiv("notfound","divnotfound").SetText("notfound")
	router.OnNotfound(ui.NewView("notfound",nd.Element()))

	n:=0

	eh:= ui.NewEventHandler(func(evt ui.Event)bool{
		n++
		router.GoTo("/test"+ strconv.Itoa(n%3+1))
		log.Print("click")
		return false
	})

	root.Element().AddEventListener("click",eh,doc.NativeEventBridge)

	router.ListenAndServe("popstate",doc.GetWindow().Element(),doc.NativeEventBridge)

	c := make(chan struct{}, 0)
	<-c
}

*/

/*
=========================== Mutation test ======================================


func main() {

	root := doc.NewDocument("test", "TestAppID")
	div := doc.NewDiv("someDiv", "someDiv", doc.EnableSessionPersistence())


	root.AppendChild(div.Element())

	div.Element().Watch("ui", "mutatediv", div.Element(), ui.NewMutationHandler(func(evt ui.MutationEvent) bool {
		s := evt.NewValue()
		b, ok := s.(ui.Bool)
		if !ok {
			return true
		}
		if b {
			log.Print("greeting....")
			div.SetText("Hello, Earthlings!")
		}else{
			log.Print("Byes...")
			text2 := doc.NewTextNode().SetValue("Bye, noobs!")
			//div.Element().Mutate(ui.AppendChildCommand(text2.Element()))
			div.Element().AppendChild(text2.Element())
		}
		return false
	}))
	v,ok := div.Element().GetData("mutatediv")
	log.Print("get data value mutatediv")
	if !ok {
		log.Print("mutatediv is not present in persistent storage.")
		div.Element().SetDataSyncUI("mutatediv", ui.Bool(true))
	} else{
		log.Print("data/mutatediv exists")
		b, ok := v.(ui.Bool)
		if !ok {
			log.Print("wrong type , expected ui.Bool")
			div.Element().SetDataSyncUI("mutatediv", ui.Bool(true))
		}
		div.Element().SetDataSyncUI("mutatediv", !b)
	}


	c := make(chan struct{}, 0)
	<-c
}
*/

/* =========================== Session Storage -- to simplify =====================



func main() {

	root := doc.NewDocument("test", "TestAppID")

	div:=doc.NewDiv("someDiv","someDiv",doc.EnableSessionPersistence())
	log.Print(div.Element().GetData("mutatediv"))
	text := doc.NewTextNode().SetValue("Hello, Earthlings!")

	// storage Test
	wd := doc.GetWindow()
	nwd,ok:=wd.Element().Native.(doc.NativeElement)
	if !ok{
		log.Print("unable to retrieve native window")
		return
	}
	nwd.JSValue().Get("sessionStorage").Set("test",true)
	root.AppendChild(div.Element())


	div.Element().Watch("data","mutatediv",div.Element(),ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		s:=evt.NewValue()
		b,ok:= s.(ui.Bool)
		if !ok{
			return true
		}
		if b{
			div.Element().AppendChild(text.Element())
			return false
		}
		div.Element().RemoveChild(text.Element())
		return false
	}))
	v,ok:= div.Element().GetData("mutatediv")
	log.Print(div.Element().Properties)
	if !ok{
		div.Element().SetData("mutatediv",ui.Bool(true))
	}

	b,ok:=v.(ui.Bool)
	if !ok{
		div.Element().SetData("mutatediv",ui.Bool(true))
	}
	div.Element().SetData("mutatediv",!b)

	c := make(chan struct{}, 0)
	<-c
}
*/

/* ================================================================================

func main() {

	root := doc.NewDocument("test", "TestAppID")

	div:=doc.NewDiv("someDiv","someDiv")
	text := doc.NewTextNode().SetValue("Hello, Earthlings!")
	div.Element().AppendChild(text.Element())

	root.AppendChild(div.Element())


	div.Element().Watch("data","somevalue",div.Element(),ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		s:=evt.NewValue()
		str,ok:= s.(ui.String)
		if !ok{
			return true
		}
		text.SetValue(str)
		return false
	}))

	div.Element().SetData("somevalue",ui.String("Bye, Earthlings !! With love!"))

	c := make(chan struct{}, 0)
	<-c
}
*/
