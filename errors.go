package jsonparser_airp

import (
	"fmt"
	"strings"
)

type ParseError struct {
	msg           string
	Token, before token
	ParentType    JSONType
	Key           string
}

func newParseError(msg string, before, after token, ast *Node) *ParseError {
	parent := parentType(ast)
	key := currentKey(ast)
	return &ParseError{
		msg:        msg,
		before:     before,
		Token:      after,
		ParentType: parent,
		Key:        key,
	}
}

func (e *ParseError) Error() string {
	if e.before == (token{}) {
		return fmt.Sprintf("%s; expected %s", e.Token.Error(), e.msg)
	}
	return fmt.Sprintf("%s; expected %s token after %s (at %s in %s)",
		e.Token.Error(), e.msg, e.before.String(), e.Key, e.ParentType)
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
			ss = append(ss, fmt.Sprint(len(o.Children)-1))
		}
	}
	rr := make([]string, len(ss))
	for i, s := range ss {
		rr[len(ss)-i-1] = s
	}
	return strings.Join(rr, ".")
}
