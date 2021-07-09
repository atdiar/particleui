// Package ui is a library of functions for simple, generic gui development.
package ui

import (
	"log"
	"time"
	//"strings"
)

// Command defines a type used to represent a UI mutation request.
//
// These commands can be logged in an append-only manner so that they are replayable
// in the order they were registered to recover UI state.
//
// In order to register a command, one just needs to Set the "command" property
// of the "ui" namespace of an Element.
//  Element.Set("ui","command",Command{...})
// As such, the Command type implements the Value interface.
type Command Object

func (c Command) discriminant() discriminant { return "particleui" }
func (c Command) ValueType() string          { return Object(c).ValueType() }
func (c Command) RawValue() Object           { return Object(c).RawValue() }

func (c Command) Name(s string) Command {
	Object(c).Set("name", String(s))
	return c
}

func (c Command) SourceID(s string) Command {
	// log.Print("source: ", s) // DEBUG
	Object(c).Set("sourceid", String(s))
	return c
}

func (c Command) TargetID(s string) Command {
	Object(c).Set("targetid", String(s))
	return c
}

func (c Command) Position(p int) Command {
	Object(c).Set("position", Number(p))
	return c
}

func (c Command) Timestamp(t time.Time) Command {
	Object(c).Set("timestamp", String(t.String()))
	return c
}

func NewUICommand() Command {
	c := Command(NewObject().SetType("Command"))
	return c.Timestamp(time.Now().UTC())
}

func AppendChildCommand(child *Element) Command {
	return NewUICommand().Name("appendchild").SourceID(child.ID)
}

func PrependChildCommand(child *Element) Command {
	return NewUICommand().Name("prependchild").SourceID(child.ID)
}

func InsertChildCommand(child *Element, index int) Command {
	return NewUICommand().Name("insertchild").SourceID(child.ID).Position(index)
}

func ReplaceChildCommand(old *Element, new *Element) Command {
	return NewUICommand().Name("replacechild").SourceID(new.ID).TargetID(old.ID)
}

func RemoveChildCommand(child *Element) Command {
	return NewUICommand().Name("removechild").SourceID(child.ID)
}

func RemoveChildrenCommand() Command {
	return NewUICommand().Name("removechildren")
}

func ActivateViewCommand(viewname string) Command {
	return NewUICommand().Name("activateview").SourceID(viewname)
}

// Mutate allows to send a command that aims to change an element, modifying the
// underlying User Interface.
// The default commands allow to change the ActiveView, AppendChild, PrependChild,
// InsertChild, ReplaceChild, RemoveChild, RemoveChildren.
//
// Why not simply use the Element methods?
//
// For the simple reason that commands can be stored to be replayed later whereas
// using the commands directly would not be a recordable action.
func Mutate(e *Element, command Command) {
	e.SetUI("command", command)
}

func (e *Element) Mutate(command Command) *Element {
	Mutate(e, command)
	return e
}

var DefaultCommandHandler = NewMutationHandler(func(evt MutationEvent) bool {
	command, ok := evt.NewValue().(Command)
	if !ok || (command.ValueType() != "Command") {
		log.Print("Wrong format for command property value ")
		return false // returning false so that handling may continue. E.g. a custom Command object was created and a handler for it is registered further down the chain
	}

	commandname, ok := Object(command).Get("name")
	if !ok {
		log.Print("Command is invalid. Missing command name")
		return true
	}
	cname, ok := commandname.(String)
	if !ok {
		log.Print("Command is invalid. Wrong type for command name value")
		return true
	}

	switch string(cname) {
	case "appendchild":
		sourceid, ok := command["sourceid"]
		if !ok {
			log.Print("Command malformed. Missing source id to append to")
			return true
		}
		e := evt.Origin()
		sid, ok := sourceid.(String)
		if !ok {
			log.Print("Error sourceid is not a string ?!")
			return true
		}
		child := e.ElementStore.GetByID(string(sid))
		log.Print(string(sid))
		if child == nil {
			log.Print("could not find item in element store") // DEBUG
			return true
		}
		e.appendChild(child)
		return false
	case "prependchild":
		sourceid, ok := command["sourceid"]
		if !ok {
			log.Print("Command malformed. Missing source id to append to")
			return true
		}
		e := evt.Origin()
		sid, ok := sourceid.(String)
		if !ok {
			log.Print("Error sourceid is not a string ?!")
			return true
		}
		child := e.ElementStore.GetByID(string(sid))
		if child == nil {
			return true
		}
		e.prependChild(child)
		return false
	case "insertChild":
		sourceid, ok := command["sourceid"]
		if !ok {
			log.Print("Command malformed. Missing source id to append to")
			return true
		}
		pos, ok := command["position"]
		if !ok {
			log.Print("Command malformed. Missing insertion positiob.")
			return true
		}
		commandpos, ok := pos.(Number)
		if !ok {
			log.Print("position to insert at is not stored as a valid numeric type")
			return true
		}
		if commandpos < 0 {
			return true
		}
		e := evt.Origin()
		sid, ok := sourceid.(String)
		if !ok {
			log.Print("Error sourceid is not a string ?!")
			return true
		}
		child := e.ElementStore.GetByID(string(sid))
		if child == nil {
			return true
		}
		e.insertChild(child, int(commandpos))
		return false
	case "replacechild":
		sourceid, ok := command["sourceid"]
		if !ok {
			log.Print("Command malformed. Missing source id to append to")
			return true
		}
		targetid, ok := command["targetid"]
		if !ok {
			log.Print("Command malformed. Missing id of target that should be replaced")
			return true
		}
		e := evt.Origin()
		sid, ok := sourceid.(String)
		if !ok {
			log.Print("Error sourceid is not a string ?!")
			return true
		}
		tid, ok := targetid.(String)
		if !ok {
			log.Print("Error targetid is not a string ?!")
			return true
		}
		newc := e.ElementStore.GetByID(string(sid))
		oldc := e.ElementStore.GetByID(string(tid))
		if newc == nil || oldc == nil {
			return true
		}
		e.replaceChild(oldc, newc)
		return false
	case "removechild":
		sourceid, ok := command["sourceid"]
		if !ok {
			log.Print("Command malformed. Missing id of source of mutation")
			return true
		}
		e := evt.Origin()
		sid, ok := sourceid.(String)
		if !ok {
			log.Print("Error sourceid is not a string ?!")
			return true
		}
		child := e.ElementStore.GetByID(string(sid))
		if child == nil {
			return true
		}
		e.removeChild(child)
		return false
	case "removechildren":
		evt.Origin().removeChildren()
		return false
	case "activateview":
		sourceid, ok := command["sourceid"]
		if !ok {
			log.Print("Command malformed. Missing viewname to activate, stored in sourceid")
			return true
		}
		viewname, ok := sourceid.(String)
		if !ok {
			log.Print("Error viewname/sourceid is not a string ?!")
			return true
		}
		err := evt.Origin().activateView(string(viewname))
		if err != nil {
			log.Print(err)
		}
		return false
	default:
		return true
	}
})
