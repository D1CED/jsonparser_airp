package jsonparser_airp

// lexer gnereates tokens from json
// after sending an error token the lexer has to quit
type lexer struct {
	mode     lexFunc
	data     string
	start    int
	pos      int
	out      chan<- token
	quit     <-chan struct{}
	row, col int
}

type lexFunc func(*lexer) lexFunc

func lexSend(l *lexer, f lexFunc, t token) lexFunc {
	select {
	case <-l.quit:
		return nil
	case l.out <- t:
		return f
	}
}

// lex reads in a json string and generate tokens for the parser.
func lex(data string) (stream <-chan token, quit func()) {
	ch := make(chan token, 1)
	q := make(chan struct{})
	l := &lexer{
		mode: noneMode,
		data: data,
		out:  ch,
		quit: q,
	}
	go func() {
		for f := l.mode; f != nil; f = f(l) {
		}
		close(l.out)
	}()
	return ch, func() { close(q) }
}

func noneMode(l *lexer) lexFunc {
	fwd := func() {
		l.pos++
		l.start = l.pos
		l.col++
	}
	if l.start >= len(l.data) {
		return nil
	}
	switch l.data[l.pos] {
	case ' ', '\t', '\r':
		fwd()
		return noneMode
	case '\n':
		l.pos++
		l.start = l.pos
		l.col = 0
		l.row++
		return noneMode
	case '{', '}', '[', ']', ',', ':':
		m := lexSend(l, noneMode, newToken(l.data[l.pos], l.row, l.col))
		fwd()
		return m
	case '"':
		fwd()
		return stringMode
	case '-', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return numberMode
	case '0':
		m := lexSend(l, noneMode, token{
			Type:     numberToken,
			Value:    "0",
			Position: [2]int{l.row, l.col},
		})
		fwd()
		return m
	default:
		return otherMode
	}
}

func stringMode(l *lexer) lexFunc {
	if l.pos >= len(l.data) {
		lexSend(l, nil, token{
			Type:     errToken,
			Value:    l.data[l.start-1:],
			Position: [2]int{l.row, l.col - 1},
		})
		return nil
	}
	if l.data[l.pos] == '"' && l.data[l.pos-1] != '\\' {
		m := lexSend(l, noneMode, token{
			Type:     stringToken,
			Value:    l.data[l.start:l.pos],
			Position: [2]int{l.row, l.col - 1},
		})
		l.pos++
		l.start = l.pos
		l.col += l.pos - l.start + 2
		return m
	}
	l.pos++
	return stringMode
}

func otherMode(l *lexer) lexFunc {
	switch {
	case l.start+len("false") > len(l.data):
		break
	case l.data[l.start:l.start+len("null")] == "null":
		m := lexSend(l, noneMode, token{
			Type:     nullToken,
			Position: [2]int{l.row, l.col},
		})
		l.pos += len("null")
		l.start = l.pos
		l.col += len("null")
		return m
	case l.data[l.start:l.start+len("true")] == "true":
		m := lexSend(l, noneMode, token{
			Type:     trueToken,
			Position: [2]int{l.row, l.col},
		})
		l.pos += len("true")
		l.start = l.pos
		l.col += len("true")
		return m
	case l.data[l.start:l.start+len("false")] == "false":
		m := lexSend(l, noneMode, token{
			Type:     falseToken,
			Position: [2]int{l.row, l.col},
		})
		l.pos += len("false")
		l.start = l.pos
		l.col += len("false")
		return m
	}
	var length int
outer:
	for _, rune_ := range l.data[l.start:] {
		switch rune_ {
		case ' ', '\t', '\r', '\n', '{', '}', '[', ']', ',', ':':
			break outer
		}
		length++
	}
	lexSend(l, nil, token{
		Type:     errToken,
		Value:    l.data[l.start : l.start+length],
		Position: [2]int{l.row, l.col},
	})
	l.pos++
	l.start = l.pos
	l.col++
	return nil
}

func numberMode(l *lexer) lexFunc {
	if l.start >= len(l.data) {
		lexSend(l, nil, token{
			Type:     numberToken,
			Value:    l.data[l.start:l.pos],
			Position: [2]int{l.row, l.col},
		})
		return nil
	}
	switch b := l.data[l.pos]; b {
	case '-', '+', 'e', 'E', '.', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		l.pos++
		return numberMode
	default:
		lexSend(l, noneMode, token{
			Type:     numberToken,
			Value:    l.data[l.start:l.pos],
			Position: [2]int{l.row, l.col},
		})
		l.start = l.pos
		l.col += l.pos - l.start
		return noneMode
	}
}
