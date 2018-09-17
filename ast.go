package jsonparser_airp

import (
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

func (n *Node) format(prefix, postfix, commaSep, colonSep string, level int) (string, error) {
	if n == nil {
		return "", nil
	}
	if !assertNodeType(n) {
		return "", fmt.Errorf("format; assertion failure")
	}
	switch n.jsonType {
	case Null:
		return "null", nil
	case Bool:
		if n.value.(bool) {
			return "true", nil
		}
		return "false", nil
	case Number:
		return fmt.Sprint(n.value.(float64)), nil
	case String:
		return `"` + n.value.(string) + `"`, nil
	case Array:
		cc := n.value.([]Node)
		if len(cc) == 0 {
			return strings.Repeat(prefix, level) + "[]", nil
		}
		builder := strings.Builder{}
		builder.Grow(64)
		builder.WriteString(strings.Repeat(prefix, level) +
			"[" + postfix)
		for _, c := range cc[:len(cc)-1] {
			s, err := c.format(prefix, postfix, commaSep,
				colonSep, level+1)
			if err != nil {
				return "", err
			}
			builder.WriteString(strings.Repeat(prefix, level+1) +
				s + "," + commaSep + postfix,
			)
		}
		s, err := cc[len(cc)-1].format(prefix,
			postfix, commaSep, colonSep, level+1)
		if err != nil {
			return "", err
		}
		builder.WriteString(strings.Repeat(prefix, level+1) +
			s + postfix + strings.Repeat(prefix, level) + "]",
		)
		return builder.String(), nil
	case Object:
		cc := n.value.([]Node)
		if len(cc) == 0 {
			return strings.Repeat(prefix, level) + "{}", nil
		}
		builder := strings.Builder{}
		builder.Grow(64)
		builder.WriteString(strings.Repeat(prefix, level) + "{" +
			postfix)
		for _, c := range cc[:len(cc)-1] {
			s, err := c.format(prefix, postfix, commaSep,
				colonSep, level+1)
			if err != nil {
				return "", err
			}
			builder.WriteString(strings.Repeat(prefix, level+1) +
				"\"" + c.key + "\":" + colonSep +
				s + "," + commaSep + postfix,
			)
		}
		s, err := cc[len(cc)-1].format(prefix,
			postfix, commaSep, colonSep, level+1)
		if err != nil {
			return "", err
		}
		builder.WriteString(strings.Repeat(prefix, level+1) +
			"\"" + cc[len(cc)-1].key + "\":" +
			colonSep +
			s + postfix + strings.Repeat(prefix, level) + "}",
		)
		return builder.String(), nil
	case Error:
		return "<error>", nil
	default:
		return "", fmt.Errorf("node of unkown type: &airp.Node{key: %v jsonType: %v value: %v} ",
			n.key, n.jsonType, n.value)
	}
}

// String formats an ast as valid JSON with few whitspace.
func (n *Node) String() string {
	s, err := n.format("", "", "", "", 0)
	if err != nil {
		return ""
	}
	return s
}

// stringDebug formats an ast for inspecting the internals.
func (n *Node) stringDebug() string {
	s, _ := n.format("!", "~", "_", "^", 0)
	return s
}

// MarshalJSON implements the json.Mashaler interface for Node
func (n *Node) MarshalJSON() ([]byte, error) {
	s, err := n.format("", "", " ", " ", 0)
	if err != nil {
		return nil, err
	}
	return []byte(s), nil
}

// UnmarshalJSON implements the json.Unmashaler interface for Node
func (n *Node) UnmarshalJSON(data []byte) error {
	m, err := parse(lex(string(data)))
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

func StandaloneNode(str string) *Node {
	n, err := parse(lex(str))
	if err != nil {
		panic(err)
	}
	if cc, ok := n.value.([]Node); ok && len(cc) > 0 {
		panic("given value must be single!")
	}
	return n
}

func (n *Node) AddChildren(nn ...Node) {
	n.value = append(n.value.([]Node), nn...)
}
