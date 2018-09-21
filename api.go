package jsonparser_airp

import (
	"bytes"
	"fmt"
)

// Valid reports whether data is a valid JSON encoding.
func Valid(data []byte) bool {
	_, err := parse(lex(bytes.NewReader(data)))
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
	return fmt.Errorf("not implemented")
}
