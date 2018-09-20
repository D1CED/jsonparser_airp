package jsonparser_airp

import (
	"bytes"
	"fmt"
	"strings"
)

// JSONType is an enum for any JSON-types
type JSONType uint8

//go:generate stringer -type JSONType
const (
	Error JSONType = iota
	Null
	Bool
	Number
	String
	Array
	Object
)

// Node is one node of tree building a JSON-ast.
// Depending on its internal type it holds different values:
//     JSONType	ValueType
//     Error	nil
//     Bool	bool
//     Number	float64
//     String	string
//     Array	[]Node
//     Object	[]Node
type Node struct {
	key      string
	jsonType JSONType
	value    interface{}
	parent   *Node
}

// Key returns the name of a Node.
func (n *Node) Key() string {
	if n == nil {
		return ""
	}
	return n.key
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
		for _, f := range n.value.([]Node) {
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

func (n *Node) format(prefix, postfix, commaSep, colonSep string) (string, error) {
	if n == nil {
		return "", nil
	}
	var inner func(int) error
	var m, o = *n, Node{}
	buf := make([]byte, 0, 64)
	inner = func(level int) error {
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
			cc := n.value.([]Node)
			if len(cc) == 0 {
				buf = append(buf, (strings.Repeat(prefix, level) + "{}")...)
				return nil
			}
			buf = append(buf, ("{" + postfix)...)
			for _, c := range cc[:len(cc)-1] {
				buf = append(buf, (strings.Repeat(prefix, level+1) +
					"\"" + c.key + "\":" + colonSep)...)
				m, o = c, m
				err := inner(level + 1)
				if err != nil {
					return err
				}
				buf = append(buf, ("," + commaSep + postfix)...)
				m = o
			}
			buf = append(buf, (strings.Repeat(prefix, level+1) + "\"" +
				cc[len(cc)-1].key + "\":" + colonSep)...)
			m, o = cc[len(cc)-1], m
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
		return "", err
	}
	return string(buf), nil
}

// String formats an ast as valid JSON with few whitspace.
func (n *Node) String() string {
	s, err := n.format("", "", "", "")
	if err != nil {
		return ""
	}
	return s
}

// stringDebug formats an ast for inspecting the internals.
func (n *Node) stringDebug() string {
	s, _ := n.format("!", "~", "-", "^")
	return s
}

// MarshalJSON implements the json.Mashaler interface for Node
func (n *Node) MarshalJSON() ([]byte, error) {
	s, err := n.format("", "", " ", " ")
	if err != nil {
		return nil, err
	}
	return []byte(s), nil
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

func eqNode(a, b *Node) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if !assertNodeType(a) || !assertNodeType(b) {
		fmt.Printf("assertion failure, %v %T\n", a.jsonType, a.value)
		return false
	}
	an, aok := a.value.([]Node)
	bn, bok := b.value.([]Node)
	if aok && bok && len(an) == len(bn) {
		for i := range an {
			if !eqNode(&an[i], &bn[i]) {
				return false
			}
		}
		return true
	}
	if a.key == b.key && a.jsonType == b.jsonType && a.value == b.value {
		return true
	}
	return false
}

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
		return n.jsonType == Array || n.jsonType == Object
	default:
		return false
	}
}

// StandaloneNode generates a single json value of str.
// It panics if str is a compund json expression.
func StandaloneNode(key, str string) *Node {
	n, err := parse(lex(strings.NewReader(str)))
	if err != nil {
		panic(err)
	}
	if cc, ok := n.value.([]Node); ok && len(cc) > 0 {
		panic("given value must be single!")
	}
	n.key = key
	return n
}

// AddChildren appends nn nodes to the Array or Object n.
// It panics if n is not of the two mentioned types or if appended values
// in an object don't have keys.
func (n *Node) AddChildren(nn ...Node) {
	if n.jsonType == Object {
		for _, n := range nn {
			if n.key == "" {
				panic("empty key for object value")
			}
		}
		n.value = append(n.value.([]Node), nn...)
	} else if n.jsonType == Array {
		n.value = append(n.value.([]Node), nn...)
	} else {
		panic("n is not array or object")
	}
}
