package jsonparser_airp

import (
	"fmt"
	"reflect"
)

// Valid reports whether data is a valid JSON encoding.
func Valid(data []byte) bool {
	_, err := parse(lex(string(data)))
	return err == nil
}

// TODO(JMH): Create an AST from Go values.
func NewNode(v interface{}) *Node {
	return nil
}
func Marshal(v interface{}) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}
func MarshalIndent(v interface{}, prefix, indent string) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}
func Unmarshal(data []byte, v interface{}) (err error) {
	r := reflect.ValueOf(v)
	if !r.CanAddr() {
		return fmt.Errorf("v %v not addressable", v)
	}
	if s, ok := v.(*interface{}); ok {
		n, err := parse(lex(string(data)))
		if err != nil {
			return err
		}
		i, err := n.Value()
		if err != nil {
			return err
		}
		*s = i
		return nil
	}
	// struct
	n, err := parse(lex(string(data)))
	if err != nil {
		return err
	}
	i, err := n.Value()
	if err != nil {
		return err
	}
	// null case?
	defer func() {
		recover()
		err = fmt.Errorf("bad type or nil derefernce")
	}()
	switch j := i.(type) {
	case bool:
		*v.(*bool) = j
		return nil
	case float64:
		*v.(*float64) = j
		return nil
	case string:
		*v.(*string) = j
		return nil
	case []interface{}:
		*v.(*[]interface{}) = j
		return nil
	case map[string]interface{}:
		*v.(*map[string]interface{}) = j
		return nil
	}
	return fmt.Errorf("not implemented")
}
