package jsonparser_airp

import "strconv"

// parser is a state machine creating an ast from lex tokens
// the parser is only allowed to cancel it if recieves an error from the lexer
type parser struct {
	in     <-chan token
	quitIn func()
	init   parseFunc
	ast    *Node
	prev   token
}

type parseFunc func(p *parser) (parseFunc, error)

// Parse reads tokens from a channel and generates a ast.
// The returned node is the root node of the ast.
func parse(ch <-chan token, quit func()) (*Node, error) {
	p := &parser{
		in:     ch,
		quitIn: quit,
		init:   expektValue,
		ast:    new(Node),
	}
	var err error
	for f := p.init; f != nil && err == nil; f, err = f(p) {
	}
	return p.ast, err
}

// parseFunc's

func expektKey(p *parser) (parseFunc, error) {
	t := <-p.in
	if t.Type != stringToken {
		p.quitIn()
		return nil, newParseError("key", p.prev, t, p.ast)
	}
	p.ast.key = t.Value
	p.prev, t = t, <-p.in
	defer func() { p.prev = t }()
	if t.Type != colonToken {
		p.quitIn()
		return nil, newParseError("colon", p.prev, t, p.ast)
	}
	return expektValue, nil
}

func expektValue(p *parser) (parseFunc, error) {
	t := <-p.in
	defer func() { p.prev = t }()
	switch t.Type {
	case numberToken:
		p.ast.jsonType = Number
		// number check
		if _, err := strconv.ParseFloat(t.Value, 64); err != nil {
			p.quitIn()
			return nil, newParseError("number", p.prev, t, p.ast)
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
		p.quitIn()
		return nil, newParseError("value", p.prev, t, p.ast)
	}
}

func expektDelim(p *parser) (parseFunc, error) {
	t, ok := <-p.in
	defer func() { p.prev = t }()
	if !ok {
		return nil, nil // all OK!
	}
	switch t.Type {
	case commaToken:
		if p.ast.parent == nil {
			p.quitIn()
			return nil, newParseError("no comma", p.prev, t, p.ast)
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
		p.quitIn()
		return nil, newParseError("no comma", p.prev, t, p.ast)
	case arrayCToken, objectCToken:
		if p.ast.parent != nil {
			p.ast = p.ast.parent
		}
		return expektDelim, nil
	default:
		p.quitIn()
		return nil, newParseError("delimiter", p.prev, t, p.ast)
	}
}
