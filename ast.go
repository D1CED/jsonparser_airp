package jsonparser_airp

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/pkg/errors"
)

// JSONType is an enum for any JSON-types
type JSONType uint8

//go:generate stringer -type JSONType

// JSONTypes to caompare nodes of an ast with. The zero value signals invalid.
const (
	Error JSONType = iota
	Null
	Bool
	Number
	String
	Array
	Object
)

// Node is one node of a tree building a JSON-AST.
// Depending on its internal type it holds a different value:
//     JSONType ValueType
//     Error    nil
//     Null     nil
//     Bool     bool
//     Number   float64
//     String   string
//     Array    []*Node
//     Object   []KeyNode
type Node struct {
	jsonType JSONType
	value    interface{}
	parent   *Node
}

type KeyNode struct {
	Key string
	*Node
}

// EqNode compares the nodes and all their children. Object keys order is
// abitary.
func EqNode(a, b *Node) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil || a.jsonType != b.jsonType {
		return false
	}
	if a.jsonType == Array {
		an, bn := a.value.([]*Node), b.value.([]*Node)
		if len(an) != len(bn) {
			return false
		}
		for i := range an {
			if !EqNode(an[i], bn[i]) {
				return false
			}
		}
		return true
	} else if a.jsonType == Object {
		an, bn := a.value.([]KeyNode), b.value.([]KeyNode)
		if len(an) != len(bn) {
			return false
		}
		for i := range an {
			if m, ok := b.GetChild(an[i].Key); !ok || !EqNode(an[i].Node, m) {
				return false
			}
		}
		return true
	} else if a.value == b.value {
		return true
	}
	return false
}

// NewJSON reads from b and generates an AST
func NewJSON(b []byte) (*Node, error) {
	return parse(lex(bytes.NewReader(b)))
}

// NewJSONReader reads from r and generates an AST
func NewJSONReader(r io.Reader) (*Node, error) {
	return parse(lex(r))
}

// NewJSONString reads from s and generates an AST
func NewJSONString(s string) (*Node, error) {
	return parse(lex(strings.NewReader(s)))
}

// NewJSONGo reads in a Go-value and generates a json ast that can be
// manipulated easily.
func NewJSONGo(val interface{}) (*Node, error) {
	if val == nil {
		return &Node{jsonType: Null}, nil
	}
	v := reflect.ValueOf(val)
	switch v.Kind() {
	case reflect.Bool:
		return &Node{jsonType: Bool, value: v.Bool()}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &Node{jsonType: Number, value: float64(v.Int())}, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &Node{jsonType: Number, value: float64(v.Uint())}, nil
	case reflect.Float32, reflect.Float64:
		return &Node{jsonType: Number, value: v.Float()}, nil
	case reflect.String:
		return &Node{jsonType: String, value: v.String()}, nil
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return &Node{jsonType: String, value: string(v.Bytes())}, nil
		}
		fallthrough
	case reflect.Array:
		nn := []*Node(nil)
		for i := 0; i < v.Len(); i++ {
			n, err := NewJSONGo(v.Index(i).Interface())
			if err != nil {
				return nil, err
			}
			nn = append(nn, n)
		}
		o := &Node{jsonType: Array, value: nn}
		for _, m := range nn {
			m.parent = o
		}
		return o, nil
	case reflect.Map:
		nn := []KeyNode(nil)
		for _, key := range v.MapKeys() {
			elem := v.MapIndex(key)
			n, err := NewJSONGo(elem.Interface())
			if err != nil {
				return nil, err
			}
			nn = append(nn, KeyNode{key.String(), n})
		}
		o := &Node{jsonType: Object, value: nn}
		for _, m := range nn {
			m.parent = o
		}
		return o, nil
	case reflect.Struct:
		nn := []KeyNode(nil)
		t := v.Type()
	outer:
		for i := 0; i < v.NumField(); i++ {
			if r, _ := utf8.DecodeRuneInString(t.Field(i).Name); !unicode.IsUpper(r) {
				continue
			}
			elemT := t.Field(i)
			tags := strings.Split(elemT.Tag.Get("json"), ",")
			key := tags[0]
			if key == "-" {
				continue
			}
			if key == "" {
				key = elemT.Name
			}
			iVal := v.Field(i)
			for _, tag := range tags[1:] {
				if tag == "omitempty" &&
					iVal.Interface() == reflect.Zero(iVal.Type()).Interface() {
					continue outer
				}
				if tag == "string" {
					iVal = reflect.ValueOf(fmt.Sprint(iVal.Interface()))
				}
			}
			n, err := NewJSONGo(iVal.Interface())
			if err != nil {
				return nil, err
			}
			nn = append(nn, KeyNode{key, n})
		}
		o := &Node{jsonType: Object, value: nn}
		for _, m := range nn {
			m.parent = o
		}
		return o, nil
	case reflect.Ptr:
		return NewJSONGo(v.Elem().Interface())
	default:
		return nil, fmt.Errorf("invalid type %s", v.Kind())
	}
}

// StandaloneNode generates a single json value of str.
// It panics if str is a compund json expression.
func StandaloneNode(key, str string) KeyNode {
	n, err := NewJSONString(str)
	if err != nil {
		panic(err)
	}
	if cc, ok := n.value.([]*Node); ok && len(cc) > 0 {
		panic("given value must be single!")
	}
	if cc, ok := n.value.([]KeyNode); ok && len(cc) > 0 {
		panic("given value must be single!")
	}
	return KeyNode{key, n}
}

// Type returns the JSONType of a node.
func (n *Node) Type() JSONType {
	if n == nil {
		return Error
	}
	return n.jsonType
}

// Key returns the name of a Node.
func (n *Node) Key() string {
	if n == nil {
		return ""
	}
	ss := make([]string, 0, 8)
outer:
	for o, p := n, n.parent; p != nil; o, p = p, p.parent {
		switch p.jsonType {
		case Object:
			kn := p.value.([]KeyNode)
			for i := range kn {
				if o == kn[i].Node {
					ss = append(ss, kn[i].Key)
					continue outer
				}
			}
			if len(kn) != 0 {
				panic(fmt.Errorf("invariant violation: %s", maxParent(o)))
			}
		case Array:
			nn := p.value.([]*Node)
			for i := range nn {
				if o == nn[i] {
					ss = append(ss, strconv.Itoa(i))
					continue outer
				}
			}
			if len(nn) != 0 {
				panic(fmt.Errorf("invariant violation: %s", maxParent(o)))
			}
		default:
			break outer
		}
	}
	rr := make([]string, len(ss))
	for i, s := range ss {
		rr[len(ss)-i-1] = s
	}
	return strings.Join(rr, ".")
}

// Value creates the Go representation of a JSON-Node.
// Like encoding/json the possible underlying types of the first return
// parameter are:
//     Object    map[string]interface{}
//     Array     []interface{}
//     String    string
//     Number    float64
//     Bool      bool
//     Null      nil (with the error being nil too)
func (n *Node) Value() (interface{}, error) {
	if !isValid(n) {
		return nil, fmt.Errorf("internal type mismatch; want %s, got %T",
			n.jsonType, n.value)
	}
	switch n.jsonType {
	default:
		return n.value, nil
	case Object:
		m := make(map[string]interface{}, 2)
		for _, f := range n.value.([]KeyNode) {
			itf, err := f.Value()
			if err != nil {
				return nil, err
			}
			m[f.Key] = itf
		}
		return m, nil
	case Array:
		s := make([]interface{}, 0, 2)
		for _, f := range n.value.([]*Node) {
			itf, err := f.Value()
			if err != nil {
				return nil, err
			}
			s = append(s, itf)
		}
		return s, nil
	}
}

// String formats an ast as valid JSON with no whitspace.
func (n *Node) String() string {
	b := &strings.Builder{}
	_, err := n.format(b, "", "", "", "")
	if err != nil {
		return ""
	}
	return b.String()
}

// MarshalJSON implements the json.Mashaler interface for Node
func (n *Node) MarshalJSON() ([]byte, error) {
	b := &bytes.Buffer{}
	_, err := n.format(b, "", "", " ", " ")
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// UnmarshalJSON implements the json.Unmashaler interface for Node
func (n *Node) UnmarshalJSON(data []byte) error {
	m, err := NewJSON(data)
	if err != nil {
		return err
	}
	*n = *m
	return nil
}

// WriteJSON writes the AST hold by n to w with the same representation as
// n.String() and no whitspace.
func (n *Node) WriteJSON(w io.Writer) (int, error) {
	return n.format(w, "", "", "", "")
}

// WriteIndent writes the AST hold by n to w with the given indent
// (preferably spaces or a tab).
func (n *Node) WriteIndent(w io.Writer, indent string) (int, error) {
	return n.format(w, indent, "\n", "", " ")
}

// JSON2Go reads contents from n and writes them into val.
// val has to be a pointer value and may panic if types don't match.
func (n *Node) JSON2Go(val interface{}) (err error) {
	return json2Go(n, val, false)
}

// AddChildren appends nn nodes to the Array or Object n.
// It panics if n is not of the two mentioned types or if appended values
// in an object don't have keys.
func (n *Node) AddChildren(nn ...KeyNode) {
	if n.jsonType == Object {
		for _, n := range nn {
			if n.Key == "" {
				panic("empty key for object value")
			}
		}
		n.value = append(n.value.([]KeyNode), nn...)
	} else if n.jsonType == Array {
		for _, m := range nn {
			n.value = append(n.value.([]*Node), m.Node)
		}
	} else {
		panic(errors.Wrapf(ErrNotArrayOrObject, "n is %s", n.jsonType))
	}
}

// GetChild returns the node specifiend by name.
// The key "" always returns the node itself.
func (n *Node) GetChild(name string) (*Node, bool) {
	keys := strings.Split(name, ".")
	if len(keys) == 1 && keys[0] == "" {
		return n, true
	}
	switch n.jsonType {
	case Object:
		kn := n.value.([]KeyNode)
		for i := range kn {
			if kn[i].Key == keys[0] {
				return kn[i].GetChild(strings.Join(keys[1:], "."))
			}
		}
		return nil, false
	case Array:
		i, err := strconv.Atoi(keys[0])
		if err != nil {
			return nil, false
		}
		nn := n.value.([]*Node)
		if len(nn) < i {
			return nil, false
		}
		return nn[i].GetChild(strings.Join(keys[1:], "."))
	default:
		return nil, false
		//panic(errors.Wrapf(ErrNotArrayOrObject, "is %s", n.jsonType))
	}
}

// SetChild adds or replaces the child key of n with val.
// SetChild does not add multiple level of objects.
// SetChild panics a to extended object is not array or object.
func (n *Node) SetChild(kn KeyNode) error {
	m, ok := n.GetChild(kn.Key)
	_ = strings.Split(kn.Key, ".")
	if ok {
		m.jsonType = kn.Node.jsonType
		m.value = kn.Node.value
		return nil
	}
	/*
		if len(keys) > 1 {
			m, ok = n.GetChild(keys[len(keys)-2])
			if ok {
				if m.jsonType == Array {
					idx, err := strconv.Atoi(keys[len(keys)-1])
					if err != nil {
						return err
					}
					if m.Len() < idx+1 {
						return fmt.Errorf("too short")
					}
				} else if m.jsonType == Object {

				}
				return ErrNotArrayOrObject
			}
		}
	*/
	return ErrNotArrayOrObject
}

// RemoveChild removes key from the ast corrctly reducing arrays
func (n *Node) RemoveChild(key string) error {
	keys := strings.Split(key, ".")
	if keys[0] == "" {
		return fmt.Errorf("empty key supplied")
	}
	if len(keys) > 1 {
		var ok bool
		n, ok = n.GetChild(strings.Join(keys[:len(keys)-1], "."))
		if !ok {
			return fmt.Errorf("Node n does not have child %s", key)
		}
	}
	if n.jsonType == Object {
		nn := n.value.([]KeyNode)
		for i, m := range nn {
			if keys[0] == m.Key {
				m.parent = nil
				n.value = append(nn[:i], nn[i+1:]...)
				return nil
			}
		}
		return fmt.Errorf("Node n does not have child %s", key)
	} else if n.jsonType == Array {
		i, err := strconv.Atoi(keys[0])
		if err != nil {
			return fmt.Errorf("not-a-number key in array")
		}
		nn := n.value.([]*Node)
		if i >= len(nn) {
			return fmt.Errorf("index out of range")
		}
		nn[i].parent = nil
		n.value = append(nn[:i], nn[i+1:]...)
		return nil
	} else {
		return errors.Wrapf(ErrNotArrayOrObject, "in %s", n.jsonType)
	}
}

// GetChildrenKeys returns a slice of all keys an Object or array holds.
// It is nil if n is not array or object and is not nil but is non-nil with
// a lengh of 0 if n is an empty array or object.
func (n *Node) GetChildrenKeys() []string {
	switch n.Type() {
	case Object:
		nn := n.value.([]KeyNode)
		ss := make([]string, len(nn))
		for i, m := range nn {
			ss[i] = m.Key
		}
		return ss
	case Array:
		nn := n.value.([]*Node)
		ss := make([]string, len(nn))
		for i := range nn {
			ss[i] = strconv.Itoa(i)
		}
		return ss
	default:
		return nil
	}
}

// Copy creates a deep copy of a Node.
func (n *Node) Copy() *Node {
	switch n.jsonType {
	case Null, Bool, Number, String:
		return &Node{jsonType: n.jsonType, value: n.value}
	case Array:
		nn := n.value.([]*Node)
		mm := make([]*Node, len(nn))
		o := &Node{jsonType: Array, value: mm}
		for i, m := range nn {
			mm[i] = m.Copy()
			mm[i].parent = o
		}
		return o
	case Object:
		kn := n.value.([]KeyNode)
		mm := make([]KeyNode, len(kn))
		o := &Node{jsonType: Object, value: mm}
		for i, m := range kn {
			mm[i].Key = m.Key
			mm[i].Node = m.Copy()
			mm[i].parent = o
		}
		return o
	default:
		return nil
	}
}

// Len gives the length of an array or items in an object
func (n *Node) Len() int {
	switch n.Type() {
	case Array:
		return len(n.value.([]*Node))
	case Object:
		return len(n.value.([]KeyNode))
	case Error:
		return 0
	default:
		return 1
	}
}

// Total returns the number of total nodes hold by n
func (n *Node) Total() int {
	switch n.Type() {
	case Array:
		i := 0
		for _, eml := range n.value.([]*Node) {
			i += eml.Total()
		}
		return i + 1
	case Object:
		i := 0
		for _, eml := range n.value.([]KeyNode) {
			i += eml.Total()
		}
		return i + 1
	default:
		return n.Len()
	}
}

// format writes a valid json representation to w with prefix as indent,
// postfix after values or opening objects/arrays, colonSep after keys and
// commaSep after each comma.
func (n *Node) format(w io.Writer, prefix, postfix, commaSep, colonSep string) (int, error) {
	if n == nil {
		return 0, fmt.Errorf("<nil>")
	}
	var inner func(int) error
	var m, o = n, &Node{}
	buf := make([]byte, 0, 64)
	inner = func(level int) error { // closure with single buffer
		if !isValid(m) {
			return fmt.Errorf("format; assertion failure")
		}
		switch m.jsonType {
		case Null:
			buf = append(buf, "null"...)
			return nil
		case Bool:
			if m.value.(bool) {
				buf = append(buf, "true"...)
				return nil
			}
			buf = append(buf, "false"...)
			return nil
		case Number:
			buf = append(buf, fmt.Sprint(m.value.(float64))...)
			return nil
		case String:
			buf = append(buf, (`"` + m.value.(string) + `"`)...)
			return nil
		case Array:
			cc := m.value.([]*Node)
			if len(cc) == 0 {
				buf = append(buf, (strings.Repeat(prefix, level) + "[]")...)
				return nil
			}
			buf = append(buf, ("[" + postfix)...)
			for _, c := range cc[:len(cc)-1] {
				buf = append(buf, strings.Repeat(prefix, level+1)...)
				m, o = c, m
				err := inner(level + 1)
				if err != nil {
					return err
				}
				m = o
				buf = append(buf, ("," + commaSep + postfix)...)
			}
			buf = append(buf, strings.Repeat(prefix, level+1)...)
			m, o = cc[len(cc)-1], m
			err := inner(level + 1)
			if err != nil {
				return err
			}
			m = o
			buf = append(buf, (postfix + strings.Repeat(prefix, level) + "]")...)
			return nil
		case Object:
			cc := m.value.([]KeyNode)
			if len(cc) == 0 {
				buf = append(buf, (strings.Repeat(prefix, level) + "{}")...)
				return nil
			}
			buf = append(buf, ("{" + postfix)...)
			for _, c := range cc[:len(cc)-1] {
				buf = append(buf, (strings.Repeat(prefix, level+1) +
					"\"" + c.Key + "\":" + colonSep)...)
				m, o = c.Node, m
				err := inner(level + 1)
				if err != nil {
					return err
				}
				buf = append(buf, ("," + commaSep + postfix)...)
				m = o
			}
			buf = append(buf, (strings.Repeat(prefix, level+1) + "\"" +
				cc[len(cc)-1].Key + "\":" + colonSep)...)
			m, o = cc[len(cc)-1].Node, m
			err := inner(level + 1)
			if err != nil {
				return err
			}
			m = o
			buf = append(buf, (postfix + strings.Repeat(prefix, level) + "}")...)
			return nil
		case Error:
			buf = append(buf, "<error>"...)
			return nil
		default:
			return fmt.Errorf("node of unknown type: %#v", m)
		}
	}
	err := inner(0)
	if err != nil {
		return 0, err
	}
	return w.Write(buf)
}

// TODO(JMH): add case insensitive match on struct tags
func json2Go(n *Node, val interface{}, stringify bool) (err error) {
	v := reflect.ValueOf(val)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("v %v not pointer", v)
	}
	switch inner := v.Elem(); inner.Kind() {
	case reflect.Bool:
		if n.jsonType != Bool {
			return fmt.Errorf("mismatched type: want Bool got %s", n.jsonType)
		}
		inner.SetBool(n.value.(bool))
		return nil
	case reflect.Float64, reflect.Float32,
		reflect.Int, reflect.Int64, reflect.Int32,
		reflect.Uint, reflect.Uint64, reflect.Uint32:
		if n.jsonType != Number {
			return fmt.Errorf("mismatched type: want Number got %s", n.jsonType)
		}
		inner.Set(reflect.ValueOf(n.value).Convert(inner.Type()))
		return nil
	case reflect.String:
		if !stringify {
			if n.jsonType != String {
				return fmt.Errorf("mismatched type: want String got %s", n.jsonType)
			}
			inner.SetString(n.value.(string))
			return nil
		}
		switch n.jsonType {
		case Null:
			inner.SetString("null")
			return nil
		case Bool:
			if n.value.(bool) {
				inner.SetString("true")
				return nil
			}
			inner.SetString("false")
			return nil
		case Number:
			inner.SetString(strconv.FormatFloat(n.value.(float64), 'b', -1, 64))
			return nil
		case String:
			inner.SetString(n.value.(string))
			return nil
		default:
			return fmt.Errorf("mismatched type: can not convert %s to string", n.jsonType)
		}
	case reflect.Slice:
		if n.jsonType != Array {
			return fmt.Errorf("mismatched type: want Array got %s", n.jsonType)
		}
		t := inner.Type().Elem() // interface{}
		nn := n.value.([]*Node)
		defer func() {
			if e := recover(); e != nil {
				switch val := e.(type) {
				case error:
					err = val
				case string:
					err = fmt.Errorf(val)
				default:
					err = fmt.Errorf("incompatible types in array")
				}
			}
		}()
		for _, m := range nn {
			inner.Set(reflect.Append(inner, reflect.
				ValueOf(m.value).
				Convert(t)))
		}
		return nil
	case reflect.Struct:
		t := inner.Type()
		for i := 0; i < t.NumField(); i++ {
			elemT := t.Field(i)
			if r, _ := utf8.DecodeRuneInString(elemT.Name); !unicode.IsUpper(r) {
				continue
			}
			tags := strings.Split(elemT.Tag.Get("json"), ",")
			if len(tags) == 1 && tags[0] == "-" {
				continue
			}
			key := tags[0]
			if key == "" {
				key = elemT.Name
			}
			elm, ok := n.GetChild(key)
			// case insensitive match without struct tag-name
			if !ok && tags[0] == "" {
				keys := n.GetChildrenKeys()
				ss := make([]string, len(keys))
				for i, k := range keys {
					ss[i] = strings.ToUpper(k)
				}
				for i, s := range ss {
					if s == strings.ToUpper(key) {
						key = keys[i]
						break
					}
				}
				elm, ok = n.GetChild(key)
			}
			omitempty := false
			for _, tag := range tags[1:] {
				if tag == "omitempty" {
					omitempty = true
					break
				}
			}
			// return if not found and not omitempty
			if !ok && omitempty {
				continue
			} else if !ok {
				return fmt.Errorf("key \"%s\" in json missing %t", key, omitempty)
			}
			strfy := false
			for _, tag := range tags[1:] {
				if tag == "string" {
					strfy = true
				}
			}
			err = json2Go(elm, inner.Field(i).Addr().Interface(), strfy)
			if err != nil {
				return err
			}
		}
		return nil
	case reflect.Map:
		if n.jsonType != Object {
			return fmt.Errorf("mismatched type: want Object got %s", n.jsonType)
		}
		t := inner.Type()
		defer func() {
			if e := recover(); e != nil {
				switch val := e.(type) {
				case error:
					err = val
				case string:
					err = fmt.Errorf(val)
				default:
					err = fmt.Errorf("incompatible types in array")
				}
			}
		}()
		for _, nn := range n.value.([]KeyNode) {
			inner.SetMapIndex(reflect.ValueOf(nn.Key), reflect.
				ValueOf(nn.value).
				Convert(t.Elem()))
		}
		return nil
	default:
		return fmt.Errorf("invalid type supplied")
	}
}

func isValid(n *Node) bool {
	if n == nil {
		return false
	}
	switch n.value.(type) {
	case nil:
		return n.jsonType == Null || n.jsonType == Error
	case bool:
		return n.jsonType == Bool
	case float64:
		return n.jsonType == Number
	case string:
		return n.jsonType == String
	case []*Node:
		return n.jsonType == Array
	case []KeyNode:
		return n.jsonType == Object
	default:
		return false
	}
}

func cyclicTest(n *Node) bool {
	if n.parent == nil {
		return true
	}
	switch n.parent.jsonType {
	case Array:
		nn := n.parent.value.([]*Node)
		for i := range nn {
			if nn[i] == n {
				return true
			}
		}
		return false
	case Object:
		kn := n.parent.value.([]KeyNode)
		for i := range kn {
			if kn[i].Node == n {
				return true
			}
		}
		return false
	default:
		return true
	}
}

// -> Root
func maxParent(n *Node) *Node {
	if n == nil || n.parent == nil {
		return nil
	}
	m := n.parent
	for ; m.parent != nil; m = m.parent {
	}
	return m
}
