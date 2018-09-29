package jsonparser_airp

import (
	"errors"
	"fmt"
)

// ErrNotArrayOrObject is a common error that multiple methods of Node
// or KeyNode return. This signals that the Node type is a standalone value.
var ErrNotArrayOrObject = errors.New("not array or object")

// ParseError captures information on errors when parsing.
type ParseError struct {
	msg        string
	token      token
	before     token
	parentType JSONType
	key        string
}

func newParseError(msg string, before, after token, ast *Node) *ParseError {
	parent := parentType(ast)
	key := ast.Key()
	return &ParseError{
		msg:        msg,
		before:     before,
		token:      after,
		parentType: parent,
		key:        key,
	}
}

func (e *ParseError) Error() string {
	if e.before == (token{}) {
		return fmt.Sprintf("%s; expected %s", e.token.Error(), e.msg)
	}
	if e.parentType == Error {
		return fmt.Sprintf("%s; expected %s token after %s",
			e.token.Error(), e.msg, e.before.String())
	}
	if e.key == "" {
		return fmt.Sprintf("%s; expected %s token after %s (in top-level %s)",
			e.token.Error(), e.msg, e.before.String(), e.parentType.String())
	}
	return fmt.Sprintf("%s; expected %s token after %s (at %s in %s)",
		e.token.Error(), e.msg, e.before.String(), e.key, e.parentType)
}

// Where returns the row and column where the syntax error in json occured.
func (e *ParseError) Where() (row, col int) {
	return e.token.position[0], e.token.position[1]
}

// helper functions

func parentType(n *Node) JSONType {
	if n == nil || n.parent == nil {
		return Error
	}
	return n.parent.jsonType
}
