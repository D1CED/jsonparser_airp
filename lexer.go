package jsonparser_airp

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"unicode/utf8"
)

// lexer gnereates tokens from json
// after sending an error token the lexer has to quit
type lexer struct {
	mode     lexFunc
	reader   *bufio.Reader
	buf      *bytes.Buffer
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
func lex(data io.Reader) (stream <-chan token, quit func()) {
	ch, q := make(chan token, 1), make(chan struct{})
	l := &lexer{
		mode:   noneMode,
		reader: bufio.NewReader(data),
		buf:    new(bytes.Buffer),
		out:    ch,
		quit:   q,
	}
	go func() {
		for f := l.mode; f != nil; f = f(l) {
		}
		close(l.out)
	}()
	return ch, func() { close(q) }
}

func noneMode(l *lexer) lexFunc {
	r, _, err := l.reader.ReadRune()
	if err != nil {
		return nil
	}
	switch r {
	case '\n':
		l.row++
		l.col = 0
		return noneMode
	case '\r':
		l.col = 0
		return noneMode
	case ' ', '\t':
		l.col++
		return noneMode
	case '{', '}', '[', ']', ',', ':':
		m := lexSend(l, noneMode, newToken(r, l.row, l.col))
		l.col++
		return m
	case '"':
		return stringMode
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		l.reader.UnreadRune()
		return numberMode
	default:
		l.reader.UnreadRune()
		return otherMode
	}
}

func stringMode(l *lexer) lexFunc {
	r, _, err := l.reader.ReadRune()
	if err != nil {
		lexSend(l, nil, token{
			value:    `"` + l.buf.String(),
			position: [2]int{l.row, l.col},
		})
		return nil
	}
	if r == '\\' {
		err := escape(l)
		if err != nil {
			lexSend(l, nil, token{
				value: l.buf.String(),
				position: [2]int{
					l.row,
					l.col - utf8.RuneCount(l.buf.Bytes()) - 1,
				},
			})
			return nil
		}
		return stringMode
	}
	if r == '"' {
		m := lexSend(l, noneMode, token{
			Type:     stringToken,
			value:    l.buf.String(),
			position: [2]int{l.row, l.col},
		})
		l.col += utf8.RuneCount(l.buf.Bytes()) + 2
		l.buf.Reset()
		return m
	}
	l.buf.WriteRune(r)
	return stringMode
}

func otherMode(l *lexer) lexFunc {
	var err error
	var r rune
	for i := 0; i < 4; i++ {
		r, _, err = l.reader.ReadRune()
		if err != nil {
			goto errL
		}
		l.buf.WriteRune(r)
	}
	if l.buf.String() == "null" {
		m := lexSend(l, noneMode, token{
			Type:     nullToken,
			position: [2]int{l.row, l.col},
		})
		l.col += len("null")
		l.buf.Reset()
		return m
	}
	if l.buf.String() == "true" {
		m := lexSend(l, noneMode, token{
			Type:     trueToken,
			position: [2]int{l.row, l.col},
		})
		l.col += len("true")
		l.buf.Reset()
		return m
	}
	r, _, err = l.reader.ReadRune()
	if err != nil {
		goto errL
	}
	l.buf.WriteRune(r)
	if l.buf.String() == "false" {
		m := lexSend(l, noneMode, token{
			Type:     falseToken,
			position: [2]int{l.row, l.col},
		})
		l.col += len("false")
		l.buf.Reset()
		return m
	}
errL:
	for i, r := range l.buf.String() {
		switch r {
		case ' ', '\t', '\r', '\n', '{', '[', ']', '}', ',', ':':
			lexSend(l, nil, token{
				value:    l.buf.String()[:i],
				position: [2]int{l.row, l.col},
			})
			return nil
		}
	}
	for {
		r, _, err := l.reader.ReadRune()
		if err != nil {
			lexSend(l, nil, token{
				value:    l.buf.String(),
				position: [2]int{l.row, l.col},
			})
			return nil
		}
		switch r {
		case ' ', '\t', '\r', '\n', '{', '}', '[', ']', ',', ':':
			lexSend(l, nil, token{
				value:    l.buf.String(),
				position: [2]int{l.row, l.col},
			})
			return nil
		}
		l.buf.WriteRune(r)
	}
}

func numberMode(l *lexer) lexFunc {
	r, _, err := l.reader.ReadRune()
	if err != nil {
		lexSend(l, nil, token{
			Type:     numberToken,
			value:    l.buf.String(),
			position: [2]int{l.row, l.col},
		})
		return nil
	}
	switch r {
	case '-', '+', 'e', 'E', '.', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		l.buf.WriteRune(r)
		return numberMode
	default:
		l.reader.UnreadRune()
		lexSend(l, noneMode, token{
			Type:     numberToken,
			value:    l.buf.String(),
			position: [2]int{l.row, l.col},
		})
		l.col += utf8.RuneCount(l.buf.Bytes())
		l.buf.Reset()
		return noneMode
	}
}

func escape(l *lexer) error {
	r, _, err := l.reader.ReadRune()
	if err != nil {
		return err
	}
	switch r {
	case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
		l.buf.WriteRune('\\')
		l.buf.WriteRune(r)
		return nil
	case 'u':
		b := make([]byte, 4)
		n, err := l.reader.Read(b)
		if err != nil || n != 4 {
			return fmt.Errorf("malformed escape sequence '\\%s'", string(b))
		}
		i, err := strconv.ParseInt(string(b), 16, 32)
		if err != nil {
			return fmt.Errorf("malformed escape sequence '\\%s'", string(b))
		}
		l.buf.WriteRune(rune(i))
		return nil
	default:
		return fmt.Errorf("malformed escape sequence")
	}
}
