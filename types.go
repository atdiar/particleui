// Package ui is a library of functions for simple, generic gui development.
package ui

import (
//"strings"
)

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

	constructorname, ok := e.Get("internals", "constructor")
	if !ok {
		DEBUG("no constructorname for ", e.ID)
		return nil
	}
	cname, ok := constructorname.(String)
	if !ok {
		DEBUG("bad constructorname")
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
	o["pui_object_typ"] = "Bool"
	o["pui_object_value"] = bool(b)
	return o.RawValue()
}
func (b Bool) ValueType() string { return "Bool" }

type String string

func (s String) discriminant() discriminant { return "particleui" }
func (s String) RawValue() Object {
	o := NewObject()
	o["pui_object_typ"] = "String"
	o["pui_object_value"] = string(s)
	return o.RawValue()
}
func (s String) ValueType() string { return "String" }

type Number float64

func (n Number) discriminant() discriminant { return "particleui" }
func (n Number) RawValue() Object {
	o := NewObject()
	o["pui_object_typ"] = "Number"
	o["pui_object_value"] = float64(n)
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
		p[k] = val 
		// pui_object_typ should still be a plain string, calling RawValue twice in a row should be idempotent
		// pui_object_value is also not tranformed allowing for idempotence of successive calls to RawValue.
		continue
	}
	return p
}

func (o Object) ValueType() string {
	t, ok := o.Get("pui_object_typ")
	if !ok {
		return "undefined"
	}
	return string(t.(String))
}

func (o Object) Get(key string) (Value, bool) {
	if key == "pui_object_typ"{
		return String(o[key].(string)),true
	}
	if key == "pui_object_value"{
		val,ok:= o[key]
		if !ok{
			return nil, ok
		}
		switch t:= val.(type){
		case bool:
			return Bool(t),ok
		case string:
			return String(t),ok
		case float64:
			return Number(t),ok
		case []interface{}:
			m := NewList()
			for _, val := range t {
				r, ok := val.(map[string]interface{})
				if ok {
					v := Object(r).Value()
					m = append(m, v)
					continue
				}
				panic("pui error: bad list rawencoding. Unable to decode.")
			}
			return m,ok
		default:
			panic("pui error: unknown raw value type")
		}
	}
	v, ok := o[key]
	if !ok{
		return nil,ok
	}
	return v.(Value), ok
}

func(o Object) MustGetString(key string) String{
	v,ok:= o.Get(key)
	if !ok{
		panic("Expected value to be present in object but it was not found")
	}
	return v.(String)
}

func(o Object) MustGetNumber(key string) Number{
	v,ok:= o.Get(key)
	if !ok{
		panic("Expected value to be present in object but it was not found")
	}
	return v.(Number)
}

func(o Object) MustGetBool(key string) Bool{
	v,ok:= o.Get(key)
	if !ok{
		panic("Expected value to be present in object but it was not found")
	}
	return v.(Bool)
}

func(o Object) MustGetList(key string) List{
	v,ok:= o.Get(key)
	if !ok{
		panic("Expected value to be present in object but it was not found")
	}
	return v.(List)
}

func(o Object) MustGetObject(key string) Object{
	v,ok:= o.Get(key)
	if !ok{
		panic("Expected value to be present in object but it was not found")
	}
	return v.(Object)
}

func (o Object) Set(key string, value Value) Object {
	o[key] = value
	return o
}
func (o Object) SetType(typ string) Object {
	o["pui_object_typ"] = typ
	return o
}
func (o Object) Value() Value {
	switch o.ValueType() {
	case "Bool":
		v, ok := o.Get("pui_object_value")
		if !ok {
			panic("pui error: raw bool value can't be found.")
		}
		return v
	case "String":
		v, ok := o.Get("pui_object_value")
		if !ok {
			panic("pui error: raw string value can't be found.")
		}
		return v
	case "Number":
		v, ok := o.Get("pui_object_value")
		if !ok {
			panic("pui error: raw number value can't be found.")
		}
		return v
	case "List":
		v, ok := o.Get("pui_object_value")
		if !ok {
			panic("pui error: raw List value can't be found.")
		}
		return v
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
			p.Set(k, u.Value())
		}
		return p
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
			DEBUG("no id")
			return nil
		}
		name, ok := p.Get("name")
		if !ok {
			DEBUG("no name")
			return nil
		}
		elementstoreid, ok := p.Get("elementstoreid")
		if !ok {
			DEBUG("no elementstore id")
			return nil
		}
		constructorname, ok := p.Get("constructorname")
		if !ok {
			DEBUG("no constructor name")
			return nil
		}
		elstoreid, ok := elementstoreid.(String)
		if !ok {
			DEBUG("Wrong type for ElementStore ID")
			return nil
		}
		// Let's get the elementstore
		elstore, ok := Stores.Get(string(elstoreid))
		if !ok {
			DEBUG("no elementstore")
			return nil
		}
		// Let's try to see if the element is in the ElementStore already
		elid, ok := id.(String)
		if !ok {
			DEBUG("Wrong type for Element ID stored in ui.Value")
			return nil
		}
		element := elstore.GetByID(string(elid))
		if element != nil {
			return element
		}
		// Otherwise we construct it. (TODO: make sure that element constructors try to get the data in store)
		cname, ok := constructorname.(String)
		if !ok {
			DEBUG("Wrong type for constructor name.")
			return nil
		}
		constructor, ok := elstore.Constructors[string(cname)]
		if !ok {
			DEBUG("constructor not found at the recorded name from Element store. Cannot create Element " + elid + " from Value")
			return nil
		}
		ename, ok := name.(String)
		if !ok {
			DEBUG("Element name in Value of wrong type.")
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
							DEBUG("bad option")
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
	o["pui_object_typ"] = "Object"
	return o
}

type List []Value

func (l List) discriminant() discriminant { return "particleui" }
func (l List) RawValue() Object {
	o := NewObject().SetType("List")

	raw := make([]interface{}, 0,len(l))
	for _, v := range l {
		raw = append(raw, v.RawValue())
	}
	o["pui_object_value"] = raw
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

/* type ListofObjects List

func (l ListofObjects) discriminant() discriminant { return "particleui" }
func (l ListofObjects) RawValue() Object {
	o := NewObject().SetType("List")

	raw := make([]interface{}, 0)
	for _, v := range List(l) {
		raw = append(raw, v.RawValue())
	}
	o["pui_object_value"] = raw
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

*/

func Copy(v Value) Value {
	o := NewObject()
	w := v.RawValue()
	for k, mv := range w {
		o[k] = mv
	}
	return o.Value()
}

func Equal(v Value, w Value) bool {
	if v == nil || w == nil {
		return false
	}
	if v.ValueType() != w.ValueType() {
		return false
	}
	if vo,ok:= v.(Object);ok{
		v= vo.Value()
	}

	if wo,ok:= w.(Object);ok{
		w= wo.Value()
	}

	switch v.ValueType() {
	case "Bool":
		return v == w
	case "String":
		return v == w
	case "Number":
		return v == w
	case "List":
		vl := v.(List)
		wl := w.(List)
		if len(vl) != len(wl) {
			return false
		}
		for i, item := range vl {
			if !Equal(item, wl[i]) {
				return false
			}
		}
		return true
	case "Object":
		vo := v.(Object).Value().(Object)
		wo := w.(Object).Value().(Object)
		if len(vo) != len(wo) {
			return false
		}
		for k, rval := range vo {
			if k == "pui_object_typ" {
				continue
			}
			val, ok := rval.(Value)
			if !ok {
				return false
			}
			rwal, ok := wo[k]
			if !ok {
				return false
			}
			wal, ok := rwal.(Value)
			if !ok {
				return false
			}
			if !Equal(val, wal) {
				return false
			}
		}
		return true
	case "Element":
		ve,ok:= v.(*Element)
		if !ok{
			panic("Element was astonishingly not marshalled back")
		}
		we,ok:= w.(*Element)
		if !ok{
			panic("Element was astonishingly not marshalled back")
		}
		if ve.ID != we.ID{
			return false
		}
	}
	return true
}
