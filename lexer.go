package jsonparser_airp

// Lex reads in a json string and generate tokens for the parser.
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

type lexer struct {
	mode  lexFunc
	data  string
	start int
	pos   int
	out   chan<- token
}

type lexFunc func(*lexer) lexFunc

func noneMode(l *lexer) lexFunc {
	if l.start >= len(l.data) {
		return nil
	}
	switch l.data[l.pos] {
	case ' ', '\t', '\n', '\r':
		l.pos++
		l.start = l.pos
		return noneMode
	case '{', '}', '[', ']', ',', ':':
		l.out <- newToken(l.data[l.pos])
		l.pos++
		l.start = l.pos
		return noneMode
	case '"':
		l.pos++
		l.start = l.pos
		return stringMode
	case '-', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return numberMode
	case '0':
		l.out <- token{Type: numberToken, Value: "0"}
		l.pos++
		l.start = l.pos
		return noneMode
	default:
		return otherMode
	}
}

func stringMode(l *lexer) lexFunc {
	if l.start >= len(l.data) {
		l.out <- token{stringToken, l.data[l.start:l.pos]}
		return nil
	}
	if l.data[l.pos] == '"' && l.data[l.pos-1] != '\\' {
		l.out <- token{stringToken, l.data[l.start:l.pos]}
		l.pos++
		l.start = l.pos
		return noneMode
	}
	l.pos++
	return stringMode
}

func otherMode(l *lexer) lexFunc {
	if l.start+len("false") > len(l.data) {
		l.out <- token{Type: errToken, Value: l.data[l.start : l.start+1]}
		l.pos++
		l.start = l.pos
		return noneMode
	}
	if l.data[l.start:l.start+len("null")] == "null" {
		l.out <- token{Type: nullToken}
		l.pos += len("null")
		l.start = l.pos
		return noneMode
	}
	if l.data[l.start:l.start+len("true")] == "true" {
		l.out <- token{Type: trueToken}
		l.pos += len("true")
		l.start = l.pos
		return noneMode
	}
	if l.data[l.start:l.start+len("false")] == "false" {
		l.out <- token{Type: falseToken}
		l.pos += len("false")
		l.start = l.pos
		return noneMode
	}
	l.out <- token{Type: errToken, Value: l.data[l.start : l.start+1]}
	l.pos++
	l.start = l.pos
	return noneMode
}

func numberMode(l *lexer) lexFunc {
	if l.start >= len(l.data) {
		l.out <- token{numberToken, l.data[l.start:l.pos]}
		return nil
	}
	switch b := l.data[l.pos]; b {
	case '-', '+', 'e', 'E', '.', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		l.pos++
		return numberMode
	default:
		l.out <- token{numberToken, l.data[l.start:l.pos]}
		l.start = l.pos
		return noneMode
	}
}
