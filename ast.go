package jsonparser_airp

import (
	"fmt"
	"strconv"
	"strings"
)

// JSONType is an enum for any JSON-types
type JSONType uint8

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
type Node struct {
	key      string
	jsonType JSONType
	value    string
	// Childen holds child Nodes if the type of the Node is either Object or
	// Array. This allows you to modify an ast.
	Children []Node
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
	switch n.jsonType {
	case String:
		return n.value, nil
	case Null:
		return nil, nil
	case Bool:
		if n.value == "true" {
			return true, nil
		}
		if n.value == "false" {
			return false, nil
		}
		return nil, fmt.Errorf("bool-value is not boolen?! '%s'", n.value)
	case Number:
		f, err := strconv.ParseFloat(string(n.value), 64)
		if err != nil {
			return 0, fmt.Errorf("js to Go number '%v' conversion failed: %v", n.value, err)
		}
		return f, nil
	case Object:
		m := make(map[string]interface{}, 2)
		for _, f := range n.Children {
			itf, err := f.Value()
			if err != nil {
				return nil, err
			}
			m[f.key] = itf
		}
		return m, nil
	case Array:
		s := make([]interface{}, 0, 2)
		for _, f := range n.Children {
			itf, err := f.Value()
			if err != nil {
				return nil, err
			}
			s = append(s, itf)
		}
		return s, nil
	default:
		return nil, fmt.Errorf("jsonToGo: %v", Error)
	}
}

func (n *Node) format(prefix, postfix, commaSep, colonSep string, level int) (string, error) {
	if n == nil {
		return "", nil
	}
	switch n.jsonType {
	case Null:
		return "null", nil
	case Bool, Number:
		return n.value, nil
	case String:
		return "\"" + n.value + "\"", nil
	case Array:
		if len(n.Children) == 0 {
			return strings.Repeat(prefix, level) + "[]", nil
		}
		builder := strings.Builder{}
		builder.Grow(64)
		builder.WriteString(strings.Repeat(prefix, level) +
			"[" + postfix)
		for _, c := range n.Children[:len(n.Children)-1] {
			s, err := c.format(prefix, postfix, commaSep,
				colonSep, level+1)
			if err != nil {
				return "", err
			}
			builder.WriteString(strings.Repeat(prefix, level+1) +
				s + "," + commaSep + postfix,
			)
		}
		s, err := n.Children[len(n.Children)-1].format(prefix,
			postfix, commaSep, colonSep, level+1)
		if err != nil {
			return "", err
		}
		builder.WriteString(strings.Repeat(prefix, level+1) +
			s + postfix + strings.Repeat(prefix, level) + "]",
		)
		return builder.String(), nil
	case Object:
		if len(n.Children) == 0 {
			return strings.Repeat(prefix, level) + "{}", nil
		}
		builder := strings.Builder{}
		builder.Grow(64)
		builder.WriteString(strings.Repeat(prefix, level) + "{" +
			postfix)
		for _, c := range n.Children[:len(n.Children)-1] {
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
		s, err := n.Children[len(n.Children)-1].format(prefix,
			postfix, commaSep, colonSep, level+1)
		if err != nil {
			return "", err
		}
		builder.WriteString(strings.Repeat(prefix, level+1) +
			"\"" + n.Children[len(n.Children)-1].key + "\":" +
			colonSep +
			s + postfix + strings.Repeat(prefix, level) + "}",
		)
		return builder.String(), nil
	default:
		return "", fmt.Errorf("node of unkown type: &airp.Node{key: %v jsonType: %v value: %v} ", n.key, n.jsonType, n.value)
	}
}

// String formats an ast as valid JSON with few whitspace.
func (n *Node) String() string {
	s, _ := n.format("", "", "", "", 0)
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
	defer func() { recover() }()
	if a.key == b.key && a.jsonType == b.jsonType && a.value == b.value {
		if a.Children != nil || b.Children != nil {
			for i := range a.Children {
				if !eqNode(&a.Children[i], &b.Children[i]) {
					return false
				}
			}
		}
		return true
	}
	return false
}

// StandaloneNode creates a new Node from given arguments meant for
// modification of an existing ast.
func StandaloneNode(k string, t JSONType, v string) *Node {
	return &Node{
		key:      k,
		jsonType: t,
		value:    v,
	}
}
