package jsonparser_airp

type lexer struct {
	mode     lexFunc
	data     string
	start    int
	pos      int
	out      chan<- token
	row, col int
}

type lexFunc func(*lexer) lexFunc

// lex reads in a json string and generate tokens for the parser.
func lex(data string) <-chan token {
	ch := make(chan token, 1)
	l := &lexer{
		mode: noneMode,
		data: data,
		out:  ch,
	}
	go func() {
		for f := l.mode; f != nil; f = f(l) {
		}
		close(l.out)
	}()
	return ch
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
		l.out <- newToken(l.data[l.pos], l.row, l.col)
		fwd()
		return noneMode
	case '"':
		fwd()
		return stringMode
	case '-', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return numberMode
	case '0':
		l.out <- token{Type: numberToken, Value: "0", Position: [2]int{l.row, l.col}}
		fwd()
		return noneMode
	default:
		return otherMode
	}
}

func stringMode(l *lexer) lexFunc {
	if l.start >= len(l.data) {
		l.out <- token{
			Type:     errToken,
			Value:    l.data[l.start:l.pos],
			Position: [2]int{l.row, l.col},
		}
		return nil
	}
	if l.data[l.pos] == '"' && l.data[l.pos-1] != '\\' {
		l.out <- token{
			Type:     stringToken,
			Value:    l.data[l.start:l.pos],
			Position: [2]int{l.row, l.col},
		}
		l.pos++
		l.start = l.pos
		l.col += l.pos - l.start
		return noneMode
	}
	l.pos++
	return stringMode
}

func otherMode(l *lexer) lexFunc {
	if l.start+len("false") > len(l.data) {
		l.out <- token{
			Type:     errToken,
			Value:    l.data[l.start:],
			Position: [2]int{l.row, l.col},
		}
		l.pos++
		l.start = l.pos
		return nil
	}
	if l.data[l.start:l.start+len("null")] == "null" {
		l.out <- token{Type: nullToken, Position: [2]int{l.row, l.col}}
		l.pos += len("null")
		l.start = l.pos
		l.col += len("null")
		return noneMode
	}
	if l.data[l.start:l.start+len("true")] == "true" {
		l.out <- token{Type: trueToken, Position: [2]int{l.row, l.col}}
		l.pos += len("true")
		l.start = l.pos
		l.col += len("true")
		return noneMode
	}
	if l.data[l.start:l.start+len("false")] == "false" {
		l.out <- token{Type: falseToken, Position: [2]int{l.row, l.col}}
		l.pos += len("false")
		l.start = l.pos
		l.col += len("false")
		return noneMode
	}
	l.out <- token{
		Type:     errToken,
		Value:    l.data[l.start : l.start+1],
		Position: [2]int{l.row, l.col},
	}
	l.pos++
	l.start = l.pos
	l.col++
	return nil
}

func numberMode(l *lexer) lexFunc {
	if l.start >= len(l.data) {
		l.out <- token{
			Type:     numberToken,
			Value:    l.data[l.start:l.pos],
			Position: [2]int{l.row, l.col},
		}
		return nil
	}
	switch b := l.data[l.pos]; b {
	case '-', '+', 'e', 'E', '.', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		l.pos++
		return numberMode
	default:
		l.out <- token{
			Type:     numberToken,
			Value:    l.data[l.start:l.pos],
			Position: [2]int{l.row, l.col},
		}
		l.start = l.pos
		l.col += l.pos - l.start
		return noneMode
	}
}
