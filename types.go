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


type Bool bool

func (b Bool) discriminant() discriminant { return "zui" }
func (b Bool) RawValue() Object {
	o := newobject()
	o["zui_object_typ"] = "Bool"
	o["zui_object_value"] = bool(b)
	return o.RawValue()
}
func (b Bool) ValueType() string { return "Bool" }

func(b Bool) Bool() bool{
	return bool(b)
}

type String string

func (s String) discriminant() discriminant { return "zui" }
func (s String) RawValue() Object {
	o := newobject()
	o["zui_object_typ"] = "String"
	o["zui_object_value"] = string(s)
	return o.RawValue()
}
func (s String) ValueType() string { return "String" }

func(s String) String() string{return string(s)}

type Number float64

func (n Number) discriminant() discriminant { return "zui" }
func (n Number) RawValue() Object {
	o := newobject()
	o["zui_object_typ"] = "Number"
	o["zui_object_value"] = float64(n)
	return o.RawValue()
}
func (n Number) ValueType() string { return "Number" }

func(n Number) Float64() float64{return float64(n)}
func(n Number) Int() int {return int(n)}
func(n Number) Int64() int64{return int64(n)}


// Object

func NewObject() Object {
	return Object{newobject(), false,2}
	//return objectsPool.Get()
}

type Object struct{
	o object
	copied bool
	offset int
}

func (o Object) discriminant() discriminant { return "zui" }
func(o Object) RawValue() Object{
	return o.o.RawValue()
}

func (o Object) ValueType() string {
	t, ok := o.o.Get("zui_object_typ")
	if !ok {
		panic("zui error: object does not have a type")
	}
	return string(t.(String))
}

func(o Object) MustGetString(key string) String{
	v,ok:= o.o.Get(key)
	if !ok{
		panic("Expected value to be present in object but it was not found")
	}
	return v.(String)
}

func(o Object) MustGetNumber(key string) Number{
	v,ok:= o.o.Get(key)
	if !ok{
		panic("Expected value to be present in object but it was not found")
	}
	return v.(Number)
}

func(o Object) MustGetBool(key string) Bool{
	v,ok:= o.o.Get(key)
	if !ok{
		panic("Expected value to be present in object but it was not found")
	}
	return v.(Bool)
}

func(o Object) MustGetList(key string) List{
	v,ok:= o.o.Get(key)
	if !ok{
		panic("Expected value to be present in object but it was not found")
	}
	return v.(List)
}

func(o *Object) MustGetObject(key string) Object{
	v,ok:= o.o.Get(key)
	if !ok{
		panic("Expected value to be present in object but it was not found")
	}
	return v.(Object)
}

func(o Object) Get(key string) (Value,bool){
	return o.o.Get(key)
}

func(o *Object) Set(key string, val Value) Object{ // TODO if value is an object and a list, check copied. Only insert copies with copied field set to false then (new copies)
	if o.copied{
		o.o.Set(key,Copy(val))
		return *o
	}
	o.o = Copy(o.o).(object)
	o.o.Set(key,val)
	o.copied = true
	if obj,ok:= val.(Object);ok{
		if obj.offset == 3{
			// means it's a raw object
			o.offset = 3
		}	
	}

	if obj,ok:= val.(object); ok && obj["zui_object_raw"] == true{
		o.offset = 3
	}

	return *o
}

func(o Object) DeepCopy() Object{
	return Object{Copy(o.o).(object),false,o.offset}
}

func(o Object)setType(typ string) Object{
	o.o.setType(typ)
	return o
}

func(o Object) Value() Value{
	return Object{o.o.Value().(object),o.copied, 2}
}

func (o Object) Size() int{
	s:= len(o.o)-o.offset
	if s < 0{
		panic("zui error: object does not have a valid size")
	}
	return s
}

func(o Object) Delete(key string) Object{
	_,ok:= o.o.Get(key)
	if !ok{
		return o
	}
	if o.copied{
		delete(o.o,key)
		return o
	}
	o.o = Copy(o.o).(object)
	delete(o.o,key)
	o.copied = true
	return o
}

func (o Object) Range(f func(key string, val Value) (done bool)){
	for k,v := range o.o{
		if k == "zui_object_typ" || k == "zui_object_value" || k == "zui_object_raw"{continue}
		if f(k,v.(Value)){break}
	}
}
// Unwrap returnis the underlying map that is used to store the object values.
// It can be used furing object creation to store values wihtout trigegering a copy.
// It can be also used after having called RawValue to get a serializable type ffor the object.
func(o Object) Unwrap() map[string]any{
	return o.o
}

func NewObjectFrom(m map[string]any) Object{
	m["zui_object_raw"] = true
	return Object{object(m),false,3}
}

func (o *Object) clear() {
	o.copied = false
	o.offset = 2
	for k,_:= range o.o{
		delete(o.o,k)
	}
}

// raw object
// in general it is the underlying format that is also used before serialization

func newobject() object{
	o := object(make(map[string]interface{}))
	o["zui_object_typ"] = "Object"
	return o
}

type object map[string]interface{}

func (o object) discriminant() discriminant { return "zui" }

func (o object) RawValue() Object {
	p := newobject()
	for k, val := range o {
		v, ok := val.(Value)
		if ok {
			p[k] = map[string]interface{}(v.RawValue().o)
			continue
		}
		p[k] = val 
		// zui_object_typ should still be a plain string, calling RawValue twice in a row should be idempotent
		// zui_object_value is also not tranformed allowing for idempotence of successive calls to RawValue.
		continue
	}
	p["zui_object_raw"] = true
	return p.AsObject()
}

func (o object) ValueType() string {
	t, ok := o.Get("zui_object_typ")
	if !ok {
		panic("zui error: object does not have a type")
	}
	return string(t.(String))
}

func (o object) Get(key string) (Value, bool) {
	if key == "zui_object_typ"{
		return String(o[key].(string)),true
	}
	if key == "zui_object_value"{
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
				/*if lv,ok:= val.(Value);ok{
					m = append(m,lv)
					continue
				}*/

				or, ok := val.(object)
				if ok {
					m.l = append(m.l, or.Value())
					continue
				}
				
				r, ok := val.(map[string]interface{})
				if ok {
					v := object(r).Value()
					m.l = append(m.l, v)
					continue
				}
				panic("zui error: bad list rawencoding. Unable to decode.")
			}
			return m,ok
		default:
			panic("zui error: unknown raw value type")
		}
	}
	v, ok := o[key]
	if !ok{
		return nil,ok
	}
	return v.(Value), ok
}

func(o object) MustGetString(key string) String{
	v,ok:= o.Get(key)
	if !ok{
		panic("Expected value to be present in object but it was not found")
	}
	return v.(String)
}

func(o object) MustGetNumber(key string) Number{
	v,ok:= o.Get(key)
	if !ok{
		panic("Expected value to be present in object but it was not found")
	}
	return v.(Number)
}

func(o object) MustGetBool(key string) Bool{
	v,ok:= o.Get(key)
	if !ok{
		panic("Expected value to be present in object but it was not found")
	}
	return v.(Bool)
}

func(o object) MustGetList(key string) List{
	v,ok:= o.Get(key)
	if !ok{
		panic("Expected value to be present in object but it was not found")
	}
	return v.(List)
}

func(o object) MustGetObject(key string) object{
	v,ok:= o.Get(key)
	if !ok{
		panic("Expected value to be present in object but it was not found")
	}
	return v.(object)
}

func (o object) Set(key string, value Value) object {
	if v,ok:= value.(Object);ok{
		if v.o["zui_object_raw"] == true{
			o["zui_object_raw"] = true
		}
		if v.copied{
			v = Object{Copy(v.o).(object),false,v.offset}
		}
		o[key] = v
	}

	if v,ok:= value.(List);ok{
		if v.copied{
			v = List{Copy(v.l).(list),false}
		}
		o[key] = v
	}


	
	if v,ok:= value.(object);ok{
		if v["zui_object_raw"] == true{
			o["zui_object_raw"] = true
		}
		o[key] = v
	} 

	return o
}
func (o object) setType(typ string) object {
	o["zui_object_typ"] = typ
	return o
}


func (o object) Value() Value {
	switch o.ValueType() {
	case "Bool":
		v, ok := o.Get("zui_object_value")
		if !ok {
			panic("zui error: raw bool value can't be found.")
		}
		return v
	case "String":
		v, ok := o.Get("zui_object_value")
		if !ok {
			panic("zui error: raw string value can't be found.")
		}
		return v
	case "Number":
		v, ok := o.Get("zui_object_value")
		if !ok {
			panic("zui error: raw number value can't be found.")
		}
		return v
	case "List":
		v, ok := o.Get("zui_object_value")
		if !ok {
			DEBUG(o)
			panic("zui error: raw List value can't be found.")
		}
		return v
	case "Object":
		delete(o, "zui_object_raw")

		p := newobject()
		for k, val := range o {
			v, ok := val.(Value)
			if !ok {
				m, ok := val.(map[string]interface{})
				if ok {
					obj := object(m)
					p.Set(k, obj.Value())
					continue
				}
				if k != "zui_object_raw" {
					p[k] = val
				}
				continue
			}
			u, ok := v.(object)
			if !ok {
				p.Set(k, v)
				continue
			}
			p.Set(k, u.Value())
		}
		
		return p		
	default:
		return o
	}
}

func (o object) AsObject() Object{
	if _,ok:= o["zui_object_raw"]; ok{
		return Object{o,false,3}
	}
	return Object{o,false,2}
}



// List

type List struct{
	l list
	copied bool
}

func NewList(val ...Value) List {
	return List{newlist(val...),false}
}

func (l List) discriminant() discriminant { return "zui" }
func (l List) RawValue() Object {
	o := newobject().setType("List")

	raw := make([]interface{}, 0,len(l.l))
	for _, v := range l.l {
		raw = append(raw, v.RawValue())
	}
	o["zui_object_value"] = raw
	return o.RawValue().o.AsObject()
}
func (l List) ValueType() string { return "List" }

func(l List) Filter(validator func(Value)bool) List{
	return List{l.l.Filter(validator),false}
}

func(l *List) Append(val ...Value) List{
	if !l.copied{
		l.l = Copy(l.l).(list)
		l.copied = true
	}
	l.l = append(l.l, val...)
	return *l
}

func (l List) Get(index int) Value {
	return l.l[index]
}

func(l *List) Set(index int, val Value) List{
	if !l.copied{
		l.l = Copy(l.l).(list)
		l.copied = true
	}
	l.l[index] = Copy(val)
	return *l
}

// Unwrap returns the raw list. Useful for iterating over it.
func (l List) Unwrap() []Value{
	return l.l
}


// rawlist

func newlist(val ...Value) list {
	if val != nil {
		return list(val)
	}
	l := make([]Value, 0)
	return list(l)
}

type list []Value

func (l list) discriminant() discriminant { return "zui" }
func (l list) RawValue() Object {
	o := newobject().setType("List")

	raw := make([]interface{}, 0,len(l))
	for _, v := range l {
		raw = append(raw, v.RawValue())
	}
	o["zui_object_value"] = raw
	return o.RawValue()
}
func (l list) ValueType() string { return "List" }

func(l list) Filter(validator func(Value)bool) list{
	var insertIndex int
	nl:= Copy(l).(list)
	for _, e := range nl {
		if validator(e) {
			nl[insertIndex] = e
			insertIndex++
		}
	}
	nl = nl[:insertIndex] // TODO clear potential remaining trailing elements?
	return nl
}


// Copy creates a deep-copy of a Value unless it is an *Element in which case it returns the
// *Element as an objecvt of type Value.
func Copy(v Value) Value {
	if v == nil{
		return v
	}
	switch t:= v.(type){
	case Bool:
		return t
	case String:
		return t
	case Number:
		return t
	case List:
		r:= List{make([]Value,0,cap(t.l)),false}
		for i,v:= range t.l{
			r.l[i] = v
		}
		return r
	case list:
		r:= list(make([]Value,0,cap(t)))
		for i,v:= range t{
			r[i] = v
		}
		return r
	case Object: 
		o:= newobject()
		for k,v:= range t.o{
			vv,ok:= v.(Value)
			if !ok{
				o[k]=v
				continue
			}
			o[k]=vv
		}
		return Object{o,false, t.offset}
	case object:
		o:= newobject()
		for k,v:= range t{
			vv,ok:= v.(Value)
			if !ok{
				o[k]=v
				continue
			}
			o[k]=vv
		}
		return o
	default:
		panic("unsupported Value type")
	}
}

func Equal(v Value, w Value) bool {
	// first, let's deal with nil
	nilv := v == nil
	nilw := w == nil

	if nilv != nilw{
		return false
	}

	// should be same value types
	if v.ValueType() != w.ValueType() {
		return false
	}

	switch v.(type) {
	case Bool:
		return v == w
	case String:
		return v == w
	case Number:
		return v == w
	case List: // TODO
		vl := v.(List)
		wl,ok := w.(List)
		if !ok{
			wl.l = w.(list)
		}
		if len(vl.l) != len(wl.l) {
			return false
		}
		for i, item := range vl.l {
			if !Equal(item, wl.l[i]) {
				return false
			}
		}
		return true
	case list: 
		vl := v.(list)
		wl,ok := w.(list)
		if !ok{
			wl = w.(List).l
		}
		if len(vl) != len(wl) {
			return false
		}
		for i, item := range vl {
			if !Equal(item, wl[i]) {
				return false
			}
		}
		return true
	case object: // TODO
		// Let's not compare rawvalued objects as an otpimization
		// RawValued objects should be strictly used for serialization requirements
		vo := v.(object)
		wo,ok := w.(object)
		if !ok{
			wo = w.(Object).o
		}
		if len(vo) != len(wo) {
			return false
		}
		for k, rval := range vo {
			if k == "zui_object_typ"  {
				continue
			}
			val := rval.(Value)
			rwal, ok := wo[k]
			if !ok {
				return false
			}
			wal := rwal.(Value)

			if !Equal(val, wal) {
				return false
			}
		}
		return true
	case Object:
		vo := v.(Object).o
		wo,ok := w.(object)
		if !ok{
			wo = w.(Object).o
		} 
		if len(vo) != len(wo) {
			return false
		}
		for k, rval := range vo {
			if k == "zui_object_typ"  {
				continue
			}
			val := rval.(Value)
			rwal, ok := wo[k]
			if !ok {
				return false
			}
			wal := rwal.(Value)

			if !Equal(val, wal) {
				return false
			}
		}
		return true

	}
	

	panic("Equality is not specified for this Value type")
}

func  CopyIfWritten(value Value) Value{
	v,ok:= value.(Object)
	if ok{
		if v.copied{
			return v.DeepCopy()
		}
		return value
	}

	vv,ok:= value.(List)
	if ok{
		if vv.copied{
			return Copy(vv)
		}
		return value
	}
	return value
}