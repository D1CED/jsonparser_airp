package airp

import (
	"regexp"
	"strconv"
)

var keyRegex = regexp.MustCompile(`[[:alpha:]][[:word:]:\-]*`)

// parser is a state machine creating an ast from lex tokens
// the parser is only allowed to cancel it if receives an error from the lexer
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
	defer quit()
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
	if p.ast.parent == nil || p.ast.parent.jsonType != Object {
		panic("invariant violation: expect key while not in object")
	}
	if t.Type == objectCToken {
		if kn, ok := p.ast.parent.value.([]KeyNode); ok && len(kn) == 1 {
			p.ast.parent.value = []KeyNode(nil)
			p.ast = p.ast.parent
			return expektDelim, nil
		}
	}
	if t.Type != stringToken {
		return nil, newParseError("key", p.prev, t, p.ast)
	}
	if t.value != keyRegex.FindString(t.value) {
		return nil, newParseError("valid key", p.prev, t, p.ast)
	}
	pp := p.ast.parent.value.([]KeyNode)
	for _, kn := range pp {
		if kn.Key == t.value {
			return nil, newParseError("unique key", p.prev, t, p.ast)
		}
	}
	if pp[len(pp)-1].Node != p.ast {
		panic("not 'this'")
	}
	pp[len(pp)-1].Key = t.value
	p.prev, t = t, <-p.in
	defer func() { p.prev = t }()
	if t.Type != colonToken {
		return nil, newParseError("colon", p.prev, t, p.ast)
	}
	return expektValue, nil
}

func expektValue(p *parser) (parseFunc, error) {
	t := <-p.in
	defer func() { p.prev = t }()
	if p.ast.parent != nil && t.Type == arrayCToken {
		if nn, ok := p.ast.parent.value.([]*Node); ok && len(nn) == 1 {
			p.ast.parent.value = []*Node(nil)
			p.ast = p.ast.parent
			return expektDelim, nil
		}
	}
	switch t.Type {
	case numberToken:
		p.ast.jsonType = Number
		// number check
		num, err := strconv.ParseFloat(t.value, 64)
		if err != nil {
			return nil, newParseError("number", p.prev, t, p.ast)
		}
		p.ast.value = num
		return expektDelim, nil
	case stringToken:
		p.ast.jsonType = String
		p.ast.value = t.value
		return expektDelim, nil
	case nullToken:
		p.ast.jsonType = Null
		return expektDelim, nil
	case trueToken:
		p.ast.jsonType = Bool
		p.ast.value = true
		return expektDelim, nil
	case falseToken:
		p.ast.jsonType = Bool
		p.ast.value = false
		return expektDelim, nil
	case arrayOToken:
		p.ast.jsonType = Array
		nn := make([]*Node, 1, 4)
		nn[0] = new(Node)
		nn[0].parent = p.ast
		p.ast.value = nn
		p.ast = nn[0]
		return expektValue, nil
	case objectOToken:
		p.ast.jsonType = Object
		kn := make([]KeyNode, 1, 4)
		kn[0].Node = new(Node)
		kn[0].parent = p.ast
		p.ast.value = kn
		p.ast = kn[0].Node
		return expektKey, nil
	default:
		return nil, newParseError("value", p.prev, t, p.ast)
	}
}

func expektDelim(p *parser) (parseFunc, error) {
	t, ok := <-p.in
	defer func() { p.prev = t }()
	if !ok {
		if p.ast.parent == nil {
			return nil, nil // all OK!
		}
		return nil, newParseError("delimiter", p.prev, p.prev, p.ast)
	}
	switch t.Type {
	case commaToken:
		if p.ast.parent == nil {
			return nil, newParseError("no comma", p.prev, t, p.ast)
		}
		if p.ast.parent.jsonType == Array {
			p.ast.parent.value = append(p.ast.parent.value.([]*Node), &Node{parent: p.ast.parent})
			p.ast = p.ast.parent.value.([]*Node)[len(p.ast.parent.value.([]*Node))-1]
			return expektValue, nil
		}
		if p.ast.parent.jsonType == Object {
			p.ast.parent.value = append(p.ast.parent.value.([]KeyNode), KeyNode{Node: &Node{parent: p.ast.parent}})
			p.ast = p.ast.parent.value.([]KeyNode)[len(p.ast.parent.value.([]KeyNode))-1].Node
			return expektKey, nil
		}
		return nil, newParseError("no comma", p.prev, t, p.ast)
	case arrayCToken, objectCToken:
		if p.ast.parent == nil {
			return nil, newParseError("to be in array or object", p.prev, t, p.ast)
		}
		switch p.ast.parent.jsonType {
		case Array:
			if t.Type != arrayCToken {
				return nil, newParseError("array closing", p.prev, t, p.ast)
			}
			p.ast = p.ast.parent
			return expektDelim, nil
		case Object:
			if t.Type != objectCToken {
				return nil, newParseError("object closing", p.prev, t, p.ast)
			}
			p.ast = p.ast.parent
			return expektDelim, nil
		default:
			return nil, newParseError("to be in array or object", p.prev, t, p.ast)
		}
	default:
		return nil, newParseError("delimiter", p.prev, t, p.ast)
	}
}
