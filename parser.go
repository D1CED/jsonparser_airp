package airp

import (
	"fmt"
	"strconv"
)

// Parse reads tokens from a channel and generates a ast.
// The returned node is the root node of the ast.
func parse(ch <-chan token) (*Node, error) {
	p := &parser{
		in:   ch,
		init: expektValue,
		ast:  new(Node),
	}
	var err error
	for f := p.init; f != nil && err == nil; f, err = f(p) {
	}
	return p.ast, err
}

type parser struct {
	in   <-chan token
	init parseFunc
	ast  *Node
}

type parseFunc func(p *parser) (parseFunc, error)

func expektKey(p *parser) (parseFunc, error) {
	t := <-p.in
	if t.Type != stringToken {
		return nil, fmt.Errorf("expected string, got %v", t)
	}
	p.ast.key = t.Value
	t = <-p.in
	if t.Type != colonToken {
		return nil, fmt.Errorf("expected colon, got %v", t)
	}
	return expektValue, nil
}

func expektValue(p *parser) (parseFunc, error) {
	t := <-p.in
	switch t.Type {
	case numberToken:
		p.ast.jsonType = Number
		// number check
		if _, err := strconv.ParseFloat(t.Value, 64); err != nil {
			return nil, err
		}
		p.ast.value = t.Value
		return expektDelim, nil
	case stringToken:
		p.ast.jsonType = String
		p.ast.value = t.Value
		return expektDelim, nil
	case nullToken:
		p.ast.jsonType = Null
		return expektDelim, nil
	case trueToken:
		p.ast.jsonType = Bool
		p.ast.value = "true"
		return expektDelim, nil
	case falseToken:
		p.ast.jsonType = Bool
		p.ast.value = "false"
		return expektDelim, nil
	case arrayOToken:
		p.ast.jsonType = Array
		p.ast.Children = make([]Node, 1)
		p.ast.Children[0].parent = p.ast
		p.ast = &p.ast.Children[0]
		return expektValue, nil
	case objectOToken:
		p.ast.jsonType = Object
		p.ast.Children = make([]Node, 1)
		p.ast.Children[0].parent = p.ast
		p.ast = &p.ast.Children[0]
		return expektKey, nil
	default:
		return nil, fmt.Errorf("expected value, got %v", t)
	}
}

func expektDelim(p *parser) (parseFunc, error) {
	t, ok := <-p.in
	if !ok {
		// all ok
		return nil, nil
	}
	switch t.Type {
	case commaToken:
		if p.ast.parent == nil {
			return nil, fmt.Errorf("not in array or object. got ','")
		}
		if p.ast.parent.jsonType == Array {
			p.ast.parent.Children = append(p.ast.parent.Children, Node{parent: p.ast.parent})
			p.ast = &p.ast.parent.Children[len(p.ast.parent.Children)-1]
			return expektValue, nil
		}
		if p.ast.parent.jsonType == Object {
			p.ast.parent.Children = append(p.ast.parent.Children, Node{parent: p.ast.parent})
			p.ast = &p.ast.parent.Children[len(p.ast.parent.Children)-1]
			return expektKey, nil
		}
		return nil, fmt.Errorf("not in array or object. got ','")
	case arrayCToken, objectCToken:
		if p.ast.parent != nil {
			p.ast = p.ast.parent
		}
		return expektDelim, nil
	default:
		return nil, fmt.Errorf("expected delimiter, got %v", t)
	}
}
