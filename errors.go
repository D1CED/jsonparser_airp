package jsonparser_airp

import (
	"fmt"
	"strings"
)

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
	key := currentKey(ast)
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
			e.token.Error(), e.msg, e.before.String(), e.parentType)
	}
	return fmt.Sprintf("%s; expected %s token after %s (at %s in %s)",
		e.token.Error(), e.msg, e.before.String(), e.key, e.parentType)
}

// Where returns the row and column where the syntax error in json occured.
func (e *ParseError) Where() (row, col int) {
	return e.token.Position[0], e.token.Position[1]
}

// helper functions

func parentType(n *Node) JSONType {
	if n == nil || n.parent == nil {
		return Error
	}
	return n.parent.jsonType
}

func currentKey(n *Node) string {
	ss := make([]string, 0, 4)
	for o := n; o != nil; o = o.parent {
		if o.key != "" {
			ss = append(ss, o.key)
		} else if o.jsonType == Array {
			ss = append(ss, fmt.Sprint(len(o.value.([]Node))-1))
		}
	}
	rr := make([]string, len(ss))
	for i, s := range ss {
		rr[len(ss)-i-1] = s
	}
	return strings.Join(rr, ".")
}
