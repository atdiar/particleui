// Package ui is a library of functions for simple, generic gui development.
package ui

import (
	//"strings"
	"encoding/json"
	"fmt"
	"reflect"
)

type discriminant string // just here to pin the definition of the Value interface to this package

// Value is the type for Element property values.
type Value interface {
	discriminant() discriminant
	RawValue() object
	ValueType() string
}

// Conv is a conversion function that converts any acceptable Go value in to a ui.Value type.
func Conv[T interface {
	~int | ~string | ~float64 | ~bool | []Value | map[string]Value
}](v T) Value {
	switch w := any(v).(type) {
	case int:
		return Number(w)
	case string:
		return String(w)
	case float64:
		return Number(w)
	case bool:
		return Bool(w)
	case []Value:
		return NewList(w...).Commit()
	case map[string]Value:
		o := NewObject()
		for k, v := range w {
			o.Set(k, v)
		}
		return o.Commit()
	default:
		panic("zui error: unsupported type")
	}
}

type Bool bool

func (b Bool) discriminant() discriminant { return "zui" }
func (b Bool) RawValue() object {
	o := newobject()
	o["zui_object_typ"] = "Bool"
	o["zui_object_raw"] = true
	o["zui_object_value"] = bool(b)
	return o
}
func (b Bool) ValueType() string { return "Bool" }

func (b Bool) Bool() bool {
	return bool(b)
}

type String string

func (s String) discriminant() discriminant { return "zui" }
func (s String) RawValue() object {
	o := newobject()
	o["zui_object_typ"] = "String"
	o["zui_object_raw"] = true
	o["zui_object_value"] = string(s)
	return o
}
func (s String) ValueType() string { return "String" }

func (s String) String() string { return string(s) }

type Number float64

func (n Number) discriminant() discriminant { return "zui" }
func (n Number) RawValue() object {
	o := newobject()
	o["zui_object_typ"] = "Number"
	o["zui_object_raw"] = true
	o["zui_object_value"] = float64(n)
	return o
}

func (n Number) ValueType() string { return "Number" }

func (n Number) Float64() float64 { return float64(n) }
func (n Number) Int() int         { return int(n) }
func (n Number) Int64() int64     { return int64(n) }

// Object

// NewObject returns a *TempObject which is a wrapper around an Object with uncommited changes
// Once values have been inserted if needed, a call to Commit returns the new Object value.
func NewObject() *TempObject {
	o := Object{newobject(), false, 2}
	return &TempObject{o}
	//return objectsPool.Get()
}

type Object struct {
	o      object
	copied bool
	offset int
}

// TempObject is a wrapper around an Object that defines a copy that has a Set method.
// This Set method mutates the copy in place.
// Once done with the modifications, ta full-fledged Object can be created by "commiting" the changes.
type TempObject struct {
	Object
}

func (o TempObject) discriminant() {} //TempObject should not be usable as a Value

// Commit commits the changes made to an object copy.
func (o *TempObject) Commit() Object {
	if !o.copied {
		return o.Object
	}
	o.copied = false
	return o.Object
}

func (o *TempObject) Set(key string, val Value) *TempObject {
	if o.copied {
		o.Object.o.Set(key, val)
		if obj, ok := val.(Object); ok {
			if obj.offset == 3 {
				// means it's a raw object
				o.Object.offset = 3
			}
		}

		if obj, ok := val.(object); ok && obj["zui_object_raw"] == true {
			o.Object.offset = 3
		}
		return o
	}
	o.Object = Copy(o.Object).(Object)
	o.Object.copied = true
	o.Object.o.Set(key, val)

	if obj, ok := val.(Object); ok {
		if obj.offset == 3 {
			// means it's a raw object
			o.Object.offset = 3
		}
	}

	if obj, ok := val.(object); ok && obj["zui_object_raw"] == true {
		o.Object.offset = 3
	}

	return o
}

func (o *TempObject) Delete(key string) *TempObject {
	if o.copied {
		delete(o.Object.o, key)
		return o
	}
	o.Object = Copy(o.Object).(Object)
	o.Object.copied = true
	delete(o.Object.o, key)
	return o
}

func (o Object) discriminant() discriminant { return "zui" }
func (o Object) RawValue() object {
	return o.o.RawValue()
}

func (o Object) ValueType() string {
	t, ok := o.o.Get("zui_object_typ")
	if !ok {
		panic("zui error: object does not have a type")
	}
	return string(t.(String))
}

func (o Object) MustGetString(key string) String {
	v, ok := o.o.Get(key)
	if !ok {
		panic("Expected value to be present in object but it was not found")
	}
	return v.(String)
}

func (o Object) MustGetNumber(key string) Number {
	v, ok := o.o.Get(key)
	if !ok {
		panic("Expected value to be present in object but it was not found")
	}
	return v.(Number)
}

func (o Object) MustGetBool(key string) Bool {
	v, ok := o.o.Get(key)
	if !ok {
		panic("Expected value to be present in object but it was not found")
	}
	return v.(Bool)
}

func (o Object) MustGetList(key string) List {
	v, ok := o.o.Get(key)
	if !ok {
		panic("Expected value to be present in object but it was not found")
	}
	return v.(List)
}

func (o Object) MustGetObject(key string) Object {
	v, ok := o.o.Get(key)
	if !ok {
		panic("Expected value to be present in object but it was not found")
	}
	return v.(Object)
}

func (o Object) Get(key string) (Value, bool) {
	return o.o.Get(key)
}

func (o Object) MakeCopy() *TempObject { // TODO if value is an object and a list, check copied. Only insert copies with copied field set to false then (new copies)
	t := &TempObject{Copy(o).(Object)}
	return t
}

func (o Object) setType(typ string) Object {
	o.o.setType(typ)
	return o
}

func (o Object) Value() Value {
	return o.o.Value()
}

func (o Object) Length() int {
	s := len(o.o) - o.offset
	if s < 0 {
		panic("zui error: object does not have a valid size")
	}
	return s
}

func (o Object) Range(f func(key string, val Value) (done bool)) {
	for k, v := range o.o {
		if k == "zui_object_typ" || k == "zui_object_value" || k == "zui_object_raw" {
			continue
		}
		done := f(k, v.(Value))
		if done {
			break
		}
	}
}

// Unwrap returnis the underlying map that is used to store the object values.
// It can be used furing object creation to store values wihtout trigegering a copy.
// It can be also used after having called RawValue to get a serializable type ffor the object.
func (o Object) Unwrap() map[string]any {
	return o.MakeCopy().o
}

// UnsafelyUnwrap returns the underlying map that is used to store the object values.
// It can be used when iteraing over an object keys without having to mutatie it (avoids a copy)
// It is deemd unsafe. This is somethign to resort to in case it shows up in performance profiling.
func (o Object) UnsafelyUnwrap() map[string]any {
	return o.o
}

func ValueFrom(m map[string]any) Value {
	return object(m).Value()
}

// raw object
// in general it is the underlying format that is also used before serialization

func newobject() object {
	o := object(make(map[string]interface{}))
	o["zui_object_typ"] = "Object"
	return o
}

type object map[string]interface{}

func (o object) discriminant() discriminant { return "zui" }

func (o object) RawValue() object {
	p := newobject()
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

func (o object) ValueType() string {
	t, ok := o.Get("zui_object_typ")
	if !ok {
		panic("zui error: object does not have a type")
	}
	return string(t.(String))
}

func (o object) Get(key string) (Value, bool) {
	if key == "zui_object_typ" {
		return String(o[key].(string)), true
	}
	if key == "zui_object_value" {
		val, ok := o[key]
		if !ok {
			return nil, ok
		}
		switch t := val.(type) {
		case bool:
			return Bool(t), ok
		case string:
			return String(t), ok
		case float64:
			return Number(t), ok
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
			return m.Commit(), ok
		default:
			panic("zui error: unknown raw value type")
		}
	}
	v, ok := o[key]
	if !ok {
		return nil, ok
	}
	return v.(Value), ok
}

func (o object) MustGetString(key string) String {
	v, ok := o.Get(key)
	if !ok {
		panic("Expected value to be present in object but it was not found")
	}
	return v.(String)
}

func (o object) MustGetNumber(key string) Number {
	v, ok := o.Get(key)
	if !ok {
		panic("Expected value to be present in object but it was not found")
	}
	return v.(Number)
}

func (o object) MustGetBool(key string) Bool {
	v, ok := o.Get(key)
	if !ok {
		panic("Expected value to be present in object but it was not found")
	}
	return v.(Bool)
}

func (o object) MustGetList(key string) List {
	v, ok := o.Get(key)
	if !ok {
		panic("Expected value to be present in object but it was not found")
	}
	return v.(List)
}

func (o object) MustGetObject(key string) object {
	v, ok := o.Get(key)
	if !ok {
		panic("Expected value to be present in object but it was not found")
	}
	return v.(object)
}

func (o object) Set(key string, value Value) object {
	if v, ok := value.(Object); ok {
		if v.o["zui_object_raw"] == true {
			o["zui_object_raw"] = true
		}
		if v.copied {
			v = Object{Copy(v.o).(object), false, v.offset}
		}
		value = v

	}

	if v, ok := value.(List); ok {
		if v.copied {
			v = List{Copy(v.l).(list), false}
		}
		value = v
	}

	if v, ok := value.(object); ok {
		if v["zui_object_raw"] == true {
			o["zui_object_raw"] = true
		}
		value = v
	}

	o[key] = value

	return o
}
func (o object) setType(typ string) object {
	o["zui_object_typ"] = typ
	return o
}

func (o object) Value() Value {
	switch o.ValueType() {
	case "Bool":
		v, ok := o["zui_object_value"]
		if !ok {
			panic("zui error: raw bool value can't be found.")
		}
		return Bool(v.(bool))
	case "String":
		v, ok := o["zui_object_value"]
		if !ok {
			panic("zui error: raw string value can't be found.")
		}
		return String(v.(string))
	case "Number":
		v, ok := o["zui_object_value"]
		if !ok {
			panic("zui error: raw number value can't be found.")
		}
		return Number(v.(float64))
	case "List":
		v, ok := o["zui_object_value"]
		if !ok {
			panic("zui error: raw List value can't be found.")
		}
		l, ok := v.([]any)
		if !ok {
			panic("zui error: raw List value is not a []any.")
		}
		nl := newlist()

		for _, val := range l {
			nl = append(nl, ValueFrom(val.(map[string]any)))
		}
		return NewListFrom(nl)

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
				if u, ok := v.(Object); ok {
					p.Set(k, u.Value())
					continue
				}
				p.Set(k, v)
				continue
			}
			p.Set(k, u.Value())
		}

		return p.AsObject()
	default:
		return o
	}
}

func (o object) AsObject() Object {
	if _, ok := o["zui_object_raw"]; ok {
		return Object{o, false, 3}
	}
	return Object{o, false, 2}
}

// List

type List struct {
	l      list
	copied bool
}

type TempList struct {
	List
}

func (t TempList) discriminant() {}

func NewList(val ...Value) *TempList {
	return &TempList{List{newlist(val...), false}}
}

func (l List) discriminant() discriminant { return "zui" }
func (l List) RawValue() object {
	o := newobject().setType("List")

	raw := make([]interface{}, 0, len(l.l))
	for _, v := range l.l {
		raw = append(raw, v.RawValue())
	}
	o["zui_object_value"] = raw
	o["zui_object_raw"] = true
	return o
}
func (l List) ValueType() string { return "List" }

func (l List) Filter(validator func(Value) bool) List {
	return List{l.l.Filter(validator), false}
}

func (l List) Range(f func(index int, val Value) (done bool)) {
	for i, v := range l.l {
		done := f(i, v)
		if done {
			break
		}
	}
}

func (l List) Length() int {
	return len(l.UnsafelyUnwrap())
}

func (l *TempList) Append(val ...Value) *TempList {
	if !l.copied {
		l.l = Copy(l.l).(list)
		l.copied = true
	}
	l.l = append(l.l, val...)
	return l
}

func (l List) Get(index int) Value {
	return l.l[index]
}

func (l *TempList) Set(index int, val Value) *TempList {
	if !l.copied {
		l.l = Copy(l.l).(list)
		l.copied = true
	}
	n := len(l.l)

	switch {
	case index < n:
		l.l[index] = Copy(val)
	case index == n:
		l.l = append(l.l, Copy(val))
	case index < cap(l.l):
		l.l = l.l[:index+1]
		for i := n; i < index; i++ {
			l.l[i] = nil
		}
		l.l[index] = Copy(val)
	default:
		l.l = append(make([]Value, 0, index+128), l.l...)
		l.l[index] = Copy(val)
	}

	return l
}

func (l List) MakeCopy() *TempList {
	return &TempList{List{Copy(l.l).(list), true}}
}

func (l *TempList) Commit() List {
	if !l.copied {
		return l.List
	}
	l.copied = false
	return l.List
}

// Unwrap returns the raw list. Useful for iterating over it.
// If Unwrap is passed anything, it will return the raw list without copying it whcih is UNSAFE.
// In such cases, the List object could be modified if the rawlist is mutated
func (l List) Unwrap(unsafelyfast ...bool) []Value {
	return l.MakeCopy().l
}

// UnsafelyUnwrap returns the raw list. Useful for iterating over it, but without copying it beforehand
// It can be used to avoid a copy if the list is not going to be modified.
// But this is unsafe, mostly for use in range statements.
func (l List) UnsafelyUnwrap() []Value {
	return l.MakeCopy().l
}

func (l List) Contains(val Value) bool {
	for _, v := range l.l {
		if Equal(v, val) {
			return true
		}
	}
	return false
}

func NewListFrom(s []Value) List {
	return List{s, false}
}

// rawlist

func newlist(val ...Value) list {
	if val != nil {
		return list(val)
	}
	l := make([]Value, 0, 128)
	return list(l)
}

type list []Value

func (l list) discriminant() discriminant { return "zui" }
func (l list) RawValue() object {
	o := newobject().setType("List")

	raw := make([]interface{}, 0, len(l))
	for _, v := range l {
		raw = append(raw, v.RawValue())
	}
	o["zui_object_value"] = raw
	o["zui_object_raw"] = true
	return o
}

func (l list) ValueType() string { return "List" }

func (l list) Filter(validator func(Value) bool) list {
	var insertIndex int
	nl := Copy(l).(list)
	for _, e := range nl {
		if validator(e) {
			nl[insertIndex] = e
			insertIndex++
		}
	}
	nl = nl[:insertIndex] // TODO clear potential remaining trailing elements?
	return nl
}

// Copy creates a shallow/deep-copy of a Value unless it is an *Element in which case it returns the
// *Element as an objecvt of type Value.
// A shallow-deep copy simply relies on Copy-on-Write behavior to avoid copying the underlying data.
func Copy(v Value) Value {
	if v == nil {
		return v
	}
	switch t := v.(type) {
	case Bool:
		return t
	case String:
		return t
	case Number:
		return t
	case List:
		r := List{make([]Value, 0, cap(t.l)), false}
		for _, v := range t.l {
			r.l = append(r.l, v)
		}
		return r
	case list:
		r := list(make([]Value, 0, cap(t)))
		for _, v := range t {
			r = append(r, v)
		}
		return r
	case Object:
		o := newobject()
		for k, v := range t.o {
			vv, ok := v.(Value)
			if !ok {
				o[k] = v
				continue
			}
			o[k] = vv
		}
		return Object{o, false, t.offset}
	case object:
		o := newobject()
		for k, v := range t {
			vv, ok := v.(Value)
			if !ok {
				o[k] = v
				continue
			}
			o[k] = vv
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

	if nilv != nilw {
		return false
	}

	// should be same value types
	/*if v.ValueType() != w.ValueType() {
		return false
	}
	*/

	switch v.(type) {
	case Bool:
		return v == w
	case String:
		return v == w
	case Number:
		return v == w // NaN might need some special handling here
	case Object:
		vo := v.(Object).o
		wo, ok := w.(Object)
		if !ok {
			return false
		}
		if len(vo) != len(wo.o) {
			return false
		}
		for k, rval := range vo {
			if k == "zui_object_typ" {
				continue
			}
			val := rval.(Value)
			rwal, ok := wo.o[k]
			if !ok {
				return false
			}
			wal := rwal.(Value)

			if !Equal(val, wal) {
				return false
			}
		}
		return true
	case List:
		vl := v.(List)
		wl, ok := w.(List)
		if !ok {
			return false
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
		wl, ok := w.(list)
		if !ok {
			return false
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
		wo, ok := w.(object)
		if !ok {
			return false
		}
		if len(vo) != len(wo) {
			return false
		}
		for k, rval := range vo {
			if k == "zui_object_typ" {
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

// Polymorphic Serialization/Deserialization functions:
// These functions are used to serialize and deserialize values of arbitrary go types.
// Should enable easy interfacing with the Value type for storage or retrieval of data to/from the UI tree.

// Serialize converts a Go value to a Value.
// The value must be of a type that doesn't contain unserializable types like functions or channels.
// Not even as map values or slice or array elements.
// The function returns an error if the value is not serializable.
func Serialize(v interface{}) (Value, error) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return nil, fmt.Errorf("invalid value")
	}

	switch rv.Kind() {
	case reflect.Ptr:
		if rv.IsNil() {
			// Explicitly handle nil pointers by returning a nil Value without error.
			return nil, nil
		}
		// Dereference non-nil pointers to serialize the actual value they point to.
		rv = rv.Elem()
		return Serialize(rv.Interface())
	case reflect.Bool:
		return Bool(rv.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return Number(rv.Convert(reflect.TypeOf(float64(0))).Float()), nil
	case reflect.String:
		return String(rv.String()), nil
	case reflect.Slice, reflect.Array:
		list := NewList()
		for i := 0; i < rv.Len(); i++ {
			elem, err := Serialize(rv.Index(i).Interface())
			if err != nil {
				return nil, err
			}
			list.Append(elem)
		}
		return list.Commit(), nil
	case reflect.Map, reflect.Struct:
		obj := NewObject()
		if rv.Kind() == reflect.Map {
			for _, key := range rv.MapKeys() {
				keyStr, err := mapKeyToString(key)
				if err != nil {
					return nil, err
				}
				value, err := Serialize(rv.MapIndex(key).Interface())
				if err != nil {
					return nil, err
				}
				obj.Set(keyStr, value)
			}
		} else if rv.Kind() == reflect.Struct {
			for i := 0; i < rv.NumField(); i++ {
				field := rv.Field(i)
				fieldName := rv.Type().Field(i).Name
				fieldValue, err := Serialize(field.Interface())
				if err != nil {
					return nil, err
				}
				obj.Set(fieldName, fieldValue)
			}
		}
		return obj.Commit(), nil
	default:
		return nil, fmt.Errorf("unsupported type: %s", rv.Kind())
	}
}

// mapKeyToString converts complex map keys to a string using JSON encoding.
func mapKeyToString(key reflect.Value) (string, error) {
	if key.Kind() == reflect.String {
		return key.String(), nil
	}
	keyData, err := json.Marshal(key.Interface())
	if err != nil {
		return "", err
	}
	return string(keyData), nil
}

// Deserialize converts a serialized Value back to a Go value.
func Deserialize[T any](v Value, ptr *T) error {
	if ptr == nil {
		return fmt.Errorf("nil pointer passed to Deserialize")
	}
	return deserialize(v, ptr)
}

// deserialize is an intermediary function that handles the actual deserialization logic.
func deserialize(v Value, target any) error {
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer")
	}
	targetValue = targetValue.Elem()

	switch val := v.(type) {
	case Bool:
		if targetValue.Kind() == reflect.Bool {
			targetValue.SetBool(bool(val))
		} else {
			return fmt.Errorf("type mismatch: expected bool, got %T", val)
		}
	case Number:
		// Example for handling ui.Number conversion to various Go numeric types.
		if !setNumber(targetValue, float64(val)) {
			return fmt.Errorf("type mismatch: expected numeric type, got %T", val)
		}
	case String:
		if targetValue.Kind() == reflect.String {
			targetValue.SetString(string(val))
		} else {
			return fmt.Errorf("type mismatch: expected string, got %T", val)
		}
	case List:
		if targetValue.Kind() != reflect.Slice {
			return fmt.Errorf("expected a slice for List deserialization, got %s", targetValue.Kind())
		}
		return deserializeList(val, targetValue)
	case Object:
		if targetValue.Kind() == reflect.Struct {
			return deserializeObject(val, targetValue)
		} else if targetValue.Kind() == reflect.Map {
			return deserializeMap(val, targetValue)
		} else {
			return fmt.Errorf("expected a struct or map for Object deserialization, got %s", targetValue.Kind())
		}
	default:
		if targetValue.Kind() == reflect.Interface {
			// if the interface is Value, we can just set it
			if targetValue.Type().Implements(reflect.TypeOf((*Value)(nil)).Elem()) {
				targetValue.Set(reflect.ValueOf(v))
				return nil
			}

			// If the interface is the empty interface (aka any), we can just set it
			if targetValue.Type().Implements(reflect.TypeOf((*any)(nil)).Elem()) {
				targetValue.Set(reflect.ValueOf(v))
				return nil
			}
		} else {
			return fmt.Errorf("unsupported Value type: %T", v)
		}

	}

	return nil
}

// setNumber attempts to set a target reflect.Value with a numeric value, adjusting for the target's kind.
func setNumber(target reflect.Value, num float64) bool {
	switch target.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		target.SetInt(int64(num))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		target.SetUint(uint64(num))
	case reflect.Float32, reflect.Float64:
		target.SetFloat(num)
	default:
		return false
	}
	return true
}

// deserializeList handles the deserialization of a ui.List into a Go slice.
func deserializeList(list List, target reflect.Value) error {
	elemType := target.Type().Elem()
	var Err error
	list.Range(func(i int, elem Value) bool {
		newElemPtr := reflect.New(elemType).Interface()
		if err := deserialize(elem, newElemPtr); err != nil {
			Err = err
			return true
		}
		target.Set(reflect.Append(target, reflect.ValueOf(newElemPtr).Elem()))
		return false
	})
	return Err
}

// deserializeObject handles the deserialization of a ui.Object into a Go struct.
func deserializeObject(obj Object, target reflect.Value) error {
	for i := 0; i < target.NumField(); i++ {
		field := target.Type().Field(i)
		if val, ok := obj.Get(field.Name); ok {
			fieldValue := target.Field(i)
			if err := deserialize(val, fieldValue.Addr().Interface()); err != nil {
				return err
			}
		}
	}
	return nil
}

// deserializeMap handles the deserialization of a ui.Object into a Go map.
func deserializeMap(obj Object, target reflect.Value) error {
	// Example implementation assumes map[string]interface{} for simplicity.
	// Adjust as necessary for other map types.
	var Err error
	target.Set(reflect.MakeMap(target.Type()))
	obj.Range(func(key string, val Value) bool {
		keyVal := reflect.ValueOf(key)
		elemPtr := reflect.New(target.Type().Elem()).Interface()

		if err := deserialize(val, elemPtr); err != nil {
			Err = err
			return true
		}
		target.SetMapIndex(keyVal, reflect.ValueOf(elemPtr).Elem())
		return false
	})
	return Err
}
