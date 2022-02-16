// Package ui is a library of functions for simple, generic gui development.
package ui

import (
	"log"
	"time"
	//"strings"
)

type MutationRecord Object

func (m MutationRecord) discriminant() discriminant { return "particleui" }
func (m MutationRecord) ValueType() string          { return "MutationRecord" }
func (m MutationRecord) RawValue() Object           { return Object(m).RawValue() }

func NewMutationRecord(category string, propname string, value Value) MutationRecord {
	mr := NewObject().SetType("MutationRecord")
	mr.Set("category", String(category))
	mr.Set("property", String(propname))
	mr.Set("value", value)
	mr.Set("timestamp", String(time.Now().UTC().String()))

	return MutationRecord(mr)
}

type discriminant string // just here to pin the definition of the Value interface to this package

// Value is the type for Element property values.
type Value interface {
	discriminant() discriminant
	RawValue() Object
	ValueType() string
}

func (e *Element) discriminant() discriminant { return "particleui" }
func (e *Element) ValueType() string          { return "Element" }
func (e *Element) RawValue() Object {
	o := NewObject().SetType("Element")

	o["id"] = String(e.ID)
	o["name"] = String(e.Name)
	constructoroptions, ok := e.Get("internals", "constructoroptions")
	if ok {
		o.Set("constructoroptions", constructoroptions)
	}

	constructorname, ok := e.Get("internals", "constructorname")
	if !ok {
		return nil
	}
	cname, ok := constructorname.(String)
	if !ok {
		return nil
	}
	o["constructorname"] = cname

	o["elementstoreid"] = String(e.ElementStore.Global.ID)
	return o.RawValue()
}

type Bool bool

func (b Bool) discriminant() discriminant { return "particleui" }
func (b Bool) RawValue() Object {
	o := NewObject()
	o["typ"] = "Bool"
	o["value"] = bool(b)
	return o.RawValue()
}
func (b Bool) ValueType() string { return "Bool" }

type String string

func (s String) discriminant() discriminant { return "particleui" }
func (s String) RawValue() Object {
	o := NewObject()
	o["typ"] = "String"
	o["value"] = string(s)
	return o.RawValue()
}
func (s String) ValueType() string { return "String" }

type Number float64

func (n Number) discriminant() discriminant { return "particleui" }
func (n Number) RawValue() Object {
	o := NewObject()
	o["typ"] = "Number"
	o["value"] = float64(n)
	return o.RawValue()
}
func (n Number) ValueType() string { return "Number" }

type Object map[string]interface{}

func (o Object) discriminant() discriminant { return "particleui" }

func (o Object) RawValue() Object {
	p := NewObject()
	for k, val := range o {
		v, ok := val.(Value)
		if ok {
			p[k] = map[string]interface{}(v.RawValue())
			continue
		}
		p[k] = val // typ should still be a plain string, calling RawValue twice in a row should be idempotent
		continue
	}
	return p
}

func (o Object) ValueType() string {
	t, ok := o.Get("typ")
	if !ok {
		return "undefined"
	}
	s, ok := t.(string)
	if !ok {
		return "undefined object"
	}
	return string(s)
}

func (o Object) Get(key string) (interface{}, bool) {
	v, ok := o[key]
	return v, ok
}

func (o Object) Set(key string, value Value) {
	o[key] = value
}
func (o Object) SetType(typ string) Object {
	o["typ"] = typ
	return o
}
func (o Object) Value() Value {
	switch o.ValueType() {
	case "Bool":
		v, ok := o.Get("value")
		if !ok {
			return nil
		}
		res, ok := v.(bool)
		if !ok {
			return nil
		}
		return Bool(res)
	case "String":
		v, ok := o.Get("value")
		if !ok {
			return nil
		}
		res, ok := v.(string)
		if !ok {
			return nil
		}
		return String(res)
	case "Number":
		v, ok := o.Get("value")
		if !ok {
			return nil
		}
		res, ok := v.(float64)
		if !ok {
			return nil
		}
		return Number(res)
	case "List":
		v, ok := o.Get("value")
		if !ok {
			return nil
		}
		l, ok := v.([]interface{})
		if !ok {
			return nil
		}
		m := NewList()
		for _, val := range l {
			r, ok := val.(map[string]interface{})
			if ok {
				v := Object(r).Value()
				m = append(m, v)
				continue
			} else {
				return nil
			}
		}
		return m
	case "Object":
		p := NewObject()
		for k, val := range o {
			v, ok := val.(Value)
			if !ok {
				m, ok := val.(map[string]interface{})
				if ok {
					obj := Object(m)
					p.Set(k, obj.Value())
					continue
				}
				p[k] = val
				continue
			}
			u, ok := v.(Object)
			if !ok {
				p.Set(k, v)
				continue
			}
			m, ok := val.(map[string]interface{})
			if ok {
				obj := Object(m)
				p.Set(k, obj.Value())
			}
			p.Set(k, u.Value())
		}
		return p
	/*case "Command":
		p := NewObject()
		for k, val := range o {
			v, ok := val.(Value)
			if !ok {
				m, ok := val.(map[string]interface{})
				if ok {
					obj := Object(m)
					p.Set(k, obj.Value())
					continue
				}
				p[k] = val
				continue
			}
			u, ok := v.(Object)
			if ok {
				p.Set(k, u.Value())
				continue
			}
			p.Set(k, v)
		}
		return Command(p)
	case "MutationRecord":
		p := NewObject()
		for k, val := range o {
			v, ok := val.(Value)
			if !ok {
				m, ok := val.(map[string]interface{})
				if ok {
					obj := Object(m)
					p.Set(k, obj.Value())
					continue
				}
				p[k] = val
				continue
			}
			u, ok := v.(Object)
			if ok {
				p.Set(k, u.Value())
				continue
			}
			p.Set(k, v)
		}
		return MutationRecord(p)*/
	case "Element":
		p := NewObject()
		for k, val := range o {
			v, ok := val.(Value)
			if !ok {
				m, ok := val.(map[string]interface{})
				if ok {
					obj := Object(m)
					p.Set(k, obj.Value())
					continue
				}
				p[k] = val
				continue
			}
			u, ok := v.(Object)
			if ok {
				p.Set(k, u.Value())
				continue
			}
			p.Set(k, v)
		}

		id, ok := p.Get("id")
		if !ok {
			return nil
		}
		name, ok := p.Get("name")
		if !ok {
			return nil
		}
		elementstoreid, ok := p.Get("elementstoreid")
		if !ok {
			return nil
		}
		constructorname, ok := p.Get("constructorname")
		if !ok {
			return nil
		}
		elstoreid, ok := elementstoreid.(String)
		if !ok {
			log.Print("Wrong type for ElementStore ID")
			return nil
		}
		// Let's get the elementstore
		elstore, ok := Stores.Get(string(elstoreid))
		if !ok {
			return nil
		}
		// Let's try to see if the element is in the ElementStore already
		elid, ok := id.(String)
		if !ok {
			log.Print("Wrong type for Element ID stored in ui.Value")
			return nil
		}
		element := elstore.GetByID(string(elid))
		if element != nil {
			return element
		}
		// Otherwise we construct it. (TODO: make sure that element constructors try to get the data in store)
		cname, ok := constructorname.(String)
		if !ok {
			log.Print("Wrong type for constructor name.")
			return nil
		}
		constructor, ok := elstore.Constructors[string(cname)]
		if !ok {
			log.Print("constructor not found at thhe recorded name from Element store. Cannot create Element " + elid + "from Value")
			return nil
		}
		ename, ok := name.(String)
		if !ok {
			log.Print("Element name in Value of wrong type.")
			return nil
		}

		coptions := make([]string, 0)
		constructoroptions, ok := p.Get("constructoroptions")
		if ok {
			objoptlist, ok := constructoroptions.(Object)
			if ok {
				voptlist := objoptlist.Value()
				optlist, ok := voptlist.(List)
				if ok {
					for _, opt := range optlist {
						sopt, ok := opt.(String)
						if !ok {
							return nil
						}
						coptions = append(coptions, string(sopt))
					}
				}
			}
		}
		return constructor(string(ename), string(elid), coptions...)

	default:
		return o
	}
}

func NewObject() Object {
	o := Object(make(map[string]interface{}))
	o["typ"] = "Object"
	return o
}

type List []Value

func (l List) discriminant() discriminant { return "particleui" }
func (l List) RawValue() Object {
	o := NewObject().SetType("List")

	raw := make([]interface{}, 0)
	for _, v := range l {
		raw = append(raw, v.RawValue())
	}
	o["value"] = raw
	return o.RawValue()
}
func (l List) ValueType() string { return "List" }

func NewList(val ...Value) List {
	if val != nil {
		return List(val)
	}
	l := make([]Value, 0)
	return List(l)
}

type ListofObjects List

func (l ListofObjects) discriminant() discriminant { return "particleui" }
func (l ListofObjects) RawValue() Object {
	o := NewObject().SetType("List")

	raw := make([]interface{}, 0)
	for _, v := range List(l) {
		raw = append(raw, v.RawValue())
	}
	o["value"] = raw
	return o.RawValue()
}
func (l ListofObjects) ValueType() string { return "List" }

func NewListofObjects() ListofObjects {
	l := make([]Value, 0)

	return ListofObjects(l)
}

func (l ListofObjects) Push(objs ...Object) ListofObjects {
	for _, v := range objs {
		l = append(l, v)
	}
	return l
}

func (l ListofObjects) Pop(index int) ListofObjects {
	i := len(l)
	if i == 0 {
		return l
	}
	if index < 0 || index >= i {
		return l
	}
	m := make([]Value, i-1)
	m = append(l[:index], l[index+1:]...)
	return m
}

func (l ListofObjects) Get(index int) Object {
	i := len(l)
	if i == 0 {
		return nil
	}
	if index < 0 || index >= i {
		return nil
	}
	v := l[index]
	o, ok := v.(Object)
	if !ok {
		panic("this should be a list of objects. it should contain objects only")
	}
	return o
}
