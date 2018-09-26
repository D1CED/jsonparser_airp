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
//     JSONType	ValueType
//     Error	nil
//     Null     nil
//     Bool	bool
//     Number	float64
//     String	string
//     Array	[]Node
//     Object	[]keyNode
type Node struct {
	jsonType JSONType
	value    interface{}
	parent   *Node
}

type KeyNode struct {
	key string
	Node
}

// Key returns the name of a Node.
func (n *Node) Key() string {
	ss := make([]string, 0, 4)
	for o, p := n, n.parent; o != nil && p != nil; o, p = p, p.parent {
		switch p.jsonType {
		case Object:
			kn := p.value.([]KeyNode)
			for i := range kn {
				if o == &kn[i].Node {
					ss = append(ss, kn[i].key)
				}
			}
		case Array:
			nn := p.value.([]Node)
			for i := range nn {
				if o == &nn[i] {
					ss = append(ss, strconv.Itoa(i))
				}
			}
		default:
			break
		}
	}
	rr := make([]string, len(ss))
	for i, s := range ss {
		rr[len(ss)-i-1] = s
	}
	return strings.Join(rr, ".")
}

// Type returns the JSONType of a node.
func (n *Node) Type() JSONType {
	if n == nil {
		return Error
	}
	return n.jsonType
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
	if !assertNodeType(n) {
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
			m[f.key] = itf
		}
		return m, nil
	case Array:
		s := make([]interface{}, 0, 2)
		for _, f := range n.value.([]Node) {
			itf, err := f.Value()
			if err != nil {
				return nil, err
			}
			s = append(s, itf)
		}
		return s, nil
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
	var m, o = *n, Node{}
	buf := make([]byte, 0, 64)
	inner = func(level int) error { // closure with single buffer
		if !assertNodeType(&m) {
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
			cc := m.value.([]Node)
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
					"\"" + c.key + "\":" + colonSep)...)
				m, o = c.Node, m
				err := inner(level + 1)
				if err != nil {
					return err
				}
				buf = append(buf, ("," + commaSep + postfix)...)
				m = o
			}
			buf = append(buf, (strings.Repeat(prefix, level+1) + "\"" +
				cc[len(cc)-1].key + "\":" + colonSep)...)
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
			return fmt.Errorf("node of unkown type: %#v", m)
		}
	}
	err := inner(0)
	if err != nil {
		return 0, err
	}
	return w.Write(buf)
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

// stringDebug formats an ast for inspecting the internals.
func (n *Node) stringDebug() string {
	b := &strings.Builder{}
	n.format(b, "!", "~", "-", "^")
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
	m, err := parse(lex(bytes.NewReader(data)))
	if err != nil {
		return err
	}
	*n = *m
	return nil
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
		an, bn := a.value.([]Node), b.value.([]Node)
		if len(an) != len(bn) {
			return false
		}
		for i := range an {
			if !EqNode(&an[i], &bn[i]) {
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
			if m, ok := b.GetChild(an[i].key); !ok && !EqNode(&an[i].Node, m) {
				return false
			}
		}
		return true
	} else if a.value == b.value {
		return true
	}
	return false
}

// -> IsValid
func assertNodeType(n *Node) bool {
	switch n.value.(type) {
	case nil:
		return n.jsonType == Null || n.jsonType == Error
	case bool:
		return n.jsonType == Bool
	case float64:
		return n.jsonType == Number
	case string:
		return n.jsonType == String
	case []Node:
		return n.jsonType == Array
	case []KeyNode:
		return n.jsonType == Object
	default:
		return false
	}
}

// StandaloneNode generates a single json value of str.
// It panics if str is a compund json expression.
func StandaloneNode(key, str string) *KeyNode {
	n, err := parse(lex(strings.NewReader(str)))
	if err != nil {
		panic(err)
	}
	if cc, ok := n.value.([]Node); ok && len(cc) > 0 {
		panic("given value must be single!")
	}
	if cc, ok := n.value.([]KeyNode); ok && len(cc) > 0 {
		panic("given value must be single!")
	}
	return &KeyNode{key, *n}
}

// AddChildren appends nn nodes to the Array or Object n.
// It panics if n is not of the two mentioned types or if appended values
// in an object don't have keys.
func (n *Node) AddChildren(nn ...KeyNode) {
	if n.jsonType == Object {
		for _, n := range nn {
			if n.key == "" {
				panic("empty key for object value")
			}
		}
		n.value = append(n.value.([]KeyNode), nn...)
	} else if n.jsonType == Array {
		for _, m := range nn {
			n.value = append(n.value.([]Node), m.Node)
		}
	} else {
		panic(errors.Wrapf(ErrNotArrayOrObject, "n is %s", n.jsonType))
	}
}

// NewJSON reads from r and generates an AST
func NewJSON(r io.Reader) (*Node, error) {
	return parse(lex(r))
}

// WriteJSON writes the AST hold by n to w with the same representation as
// n.String() and no whitspace.
func (n *Node) WriteJSON(w io.Writer) (int, error) {
	return n.format(w, "", "", "", "")
}

// WriteIndent writes the AST hold by n to w with the given indent
// (preferably spaces or a tab).
func (n *Node) WriteIndent(w io.Writer, indent string) (int, error) {
	return n.format(w, indent, "\n", " ", " ")
}

// NewJSONGo reads in a Go-value and generates a json ast that can be
// manipulated easily.
// TODO(JMH): add full support for struct-tag options
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
		nn := []Node(nil)
		for i := 0; i < v.Len(); i++ {
			n, err := NewJSONGo(v.Index(i).Interface())
			if err != nil {
				return nil, err
			}
			nn = append(nn, *n)
		}
		return &Node{jsonType: Array, value: nn}, nil
	case reflect.Map:
		nn := []KeyNode(nil)
		for _, key := range v.MapKeys() {
			elem := v.MapIndex(key)
			n, err := NewJSONGo(elem.Interface())
			if err != nil {
				return nil, err
			}
			nn = append(nn, KeyNode{key.String(), *n})
		}
		return &Node{jsonType: Object, value: nn}, nil
	case reflect.Struct:
		nn := []KeyNode(nil)
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			if r, _ := utf8.DecodeRuneInString(t.Field(i).Name); !unicode.IsUpper(r) {
				continue
			}
			n, err := NewJSONGo(v.Field(i).Interface())
			if err != nil {
				return nil, err
			}
			elemT := t.Field(i)
			tags := strings.Split(elemT.Tag.Get("json"), ",")
			key := tags[0]
			if key == "" {
				key = elemT.Name
			}
			nn = append(nn, KeyNode{key, *n})
		}
		return &Node{jsonType: Object, value: nn}, nil
	case reflect.Ptr:
		return NewJSONGo(v.Elem().Interface())
	default:
		return nil, fmt.Errorf("invalid type %s", v.Kind())
	}
}

// GetChild returns the node specifiend by name.
// GetChild panics if n is not of type array or object, but
// the key "" always returns the node itself.
func (n *Node) GetChild(name string) (*Node, bool) {
	keys := strings.Split(name, ".")
	if len(keys) == 1 && keys[0] == "" {
		return n, true
	}
	switch n.jsonType {
	case Object:
		for _, c := range n.value.([]KeyNode) {
			if c.key == keys[0] {
				return c.GetChild(strings.Join(keys[1:], "."))
			}
		}
		return nil, false
	case Array:
		i, err := strconv.Atoi(keys[0])
		if err != nil {
			return nil, false
		}
		nn := n.value.([]Node)
		if len(nn) < i {
			return nil, false
		}
		return nn[i].GetChild(strings.Join(keys[1:], "."))
	default:
		panic(errors.Wrapf(ErrNotArrayOrObject, "is %s", n.jsonType))
	}
}

// SetChild adds or replaces the child key of n with val.
// SetChild does not add multiple level of objects.
// SetChild panics a to extended object is not array or object.
func (n *Node) SetChild(key string, val *Node) error {
	m, ok := n.GetChild(key)
	keys := strings.Split(key, ".")
	if ok {
		*m = *val
		return nil
	}
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
	return ErrNotArrayOrObject
}

// Len gives the length of an array or items in an object
func (n *Node) Len() int {
	switch n.Type() {
	case Array:
		return len(n.value.([]Node))
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
	case Array, Object:
		i := 0
		for _, eml := range n.value.([]Node) {
			i += eml.Total()
		}
		return i + 1
	default:
		return n.Len()
	}
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
			return fmt.Errorf("node n does not have child %s", key)
		}
	}
	if n.jsonType == Object {
		nn := n.value.([]KeyNode)
		for i, m := range nn {
			if keys[0] == m.key {
				nn = append(nn[:i], nn[i+1:]...)
			}
		}
		return nil
	} else if n.jsonType == Array {
		i, err := strconv.Atoi(keys[0])
		if err != nil {
			return fmt.Errorf("not-a-number key in array")
		}
		nn := n.value.([]Node)
		nn = append(nn[:i], nn[i+1:]...)
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
		nn := n.value.([]Node)
		ss := make([]string, len(nn))
		for i, m := range nn {
			ss[i] = m.Key()
		}
		return ss
	case Array:
		nn := n.value.([]Node)
		ss := make([]string, len(nn))
		for i := range nn {
			ss[i] = strconv.Itoa(i)
		}
		return ss
	default:
		return nil
	}
}

// JSON2Go reads contents from n and writes them into val.
// val has to be a pointer value and may panic if types don't match.
func (n *Node) JSON2Go(val interface{}) (err error) {
	return json2Go(n, val, false)
}

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
		nn := n.value.([]Node)
		defer func() {
			if e := recover(); e != nil {
				switch val := e.(type) {
				case error:
					err = val
				case string:
					err = fmt.Errorf(val)
				default:
					err = fmt.Errorf("incomparible types in array")
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
			if !ok {
				omitempty := false
				for _, tag := range tags[1:] {
					if tag == "omitempty" {
						omitempty = true
						break
					}
				}
				if omitempty {
					continue
				}
				return fmt.Errorf("key in json missing")
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
					err = fmt.Errorf("incomparible types in array")
				}
			}
		}()
		for _, nn := range n.value.([]KeyNode) {
			inner.SetMapIndex(reflect.ValueOf(nn.key), reflect.
				ValueOf(nn.value).
				Convert(t.Elem()))
		}
		return nil
	default:
		return fmt.Errorf("invalid type supplied")
	}
}
