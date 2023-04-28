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
	o := NewObject()
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
	o := NewObject()
	o["zui_object_typ"] = "String"
	o["zui_object_value"] = string(s)
	return o.RawValue()
}
func (s String) ValueType() string { return "String" }

func(s String) String() string{return string(s)}

type Number float64

func (n Number) discriminant() discriminant { return "zui" }
func (n Number) RawValue() Object {
	o := NewObject()
	o["zui_object_typ"] = "Number"
	o["zui_object_value"] = float64(n)
	return o.RawValue()
}
func (n Number) ValueType() string { return "Number" }

func(n Number) Float64() float64{return float64(n)}
func(n Number) Int() int {return int(n)}
func(n Number) Int64() int64{return int64(n)}

type Object map[string]interface{}

func (o Object) discriminant() discriminant { return "zui" }

func (o Object) RawValue() Object {
	p := NewObject()
	for k, val := range o {
		v, ok := val.(Value)
		if ok {
			p[k] = map[string]interface{}(v.RawValue())
			continue
		}
		p[k] = val 
		// zui_object_typ should still be a plain string, calling RawValue twice in a row should be idempotent
		// zui_object_value is also not tranformed allowing for idempotence of successive calls to RawValue.
		continue
	}
	p["zui_object_raw"] = true
	return p
}

func (o Object) ValueType() string {
	t, ok := o.Get("zui_object_typ")
	if !ok {
		panic("zui error: object does not have a type")
	}
	return string(t.(String))
}

func (o Object) Get(key string) (Value, bool) {
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
			for i, val := range t {
				/*if lv,ok:= val.(Value);ok{
					m = append(m,lv)
					continue
				}*/

				or, ok := val.(Object)
				if ok {
					m = append(m, or.Value())
					continue
				}
				
				r, ok := val.(map[string]interface{})
				if ok {
					v := Object(r).Value()
					m = append(m, v)
					continue
				}
				DEBUG(i,val)
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
	/*if v,ok:= value.(Object);ok{
		if v["zui_object_raw"] == true{
			o["zui_object_raw"] = true
		}
	}*/ // could be needed to distinguish objects storing raw encoded ones. Although on removal 
	// would not be updated. In any case, we consider for now that a raw encoded object is fully raw and 
	// vice cersa, a non raw encoded object does not store raw object values
	return o
}
func (o Object) setType(typ string) Object {
	o["zui_object_typ"] = typ
	return o
}

func (o Object) MarkedRaw() Object {
	o["zui_object_raw"] = true
	return o
}


func (o Object) Value() Value {
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
		if r,ok:= o["zui_object_raw"]; !ok || !r.(bool){
			return o
		}

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
				if k != "zui_object_raw" {
					p[k] = val
				}
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
	default:
		return o
	}
}

func NewObject() Object {
	o := Object(make(map[string]interface{}))
	o["zui_object_typ"] = "Object"
	return o
}

type List []Value

func (l List) discriminant() discriminant { return "zui" }
func (l List) RawValue() Object {
	o := NewObject().setType("List")

	raw := make([]interface{}, 0,len(l))
	for _, v := range l {
		raw = append(raw, v.RawValue())
	}
	o["zui_object_value"] = raw
	return o.RawValue()
}
func (l List) ValueType() string { return "List" }

func(l List) Filter(validator func(Value)bool) List{
	var insertIndex int
	nl:= Copy(l).(List)
	for _, e := range nl {
		if validator(e) {
			nl[insertIndex] = e
			insertIndex++
		}
	}
	nl = nl[:insertIndex] // TODO clear potential remaining trailing elements?
	return nl
}

func NewList(val ...Value) List {
	if val != nil {
		return List(val)
	}
	l := make([]Value, 0)
	return List(l)
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
		r:= List(make([]Value,len(t),cap(t)))
		for i,v:= range t{
			r[i] = Copy(v)
		}
		return r
	case Object:
		o:= NewObject()
		for k,v:= range t{
			vv,ok:= v.(Value)
			if !ok{
				o[k]=v
				continue
			}
			o[k]=Copy(vv)
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
		// Let's not compare rawvalues as an otpimization
		// RawValues should be strictly used for serialization requirements
		vo := v.(Object)
		wo := w.(Object)
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

