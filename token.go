package jsonparser_airp

type tokenType uint8

const (
	errToken tokenType = iota
	nullToken
	trueToken
	falseToken
	numberToken
	stringToken
	commaToken
	colonToken
	arrayOToken
	arrayCToken
	objectOToken
	objectCToken
)

type token struct {
	Type  tokenType
	Value string
}

func newToken(b byte) token {
	switch b {
	case '{':
		return token{Type: objectOToken}
	case '}':
		return token{Type: objectCToken}
	case '[':
		return token{Type: arrayOToken}
	case ']':
		return token{Type: arrayCToken}
	case ':':
		return token{Type: colonToken}
	case ',':
		return token{Type: commaToken}
	default:
		return token{Value: string(b)}
	}
}

// String generates a readable form of a token meant for debuging.
func (t token) String() string {
	switch t.Type {
	case errToken:
		return "lex-err_" + string(t.Value)
	case nullToken:
		return "'null'"
	case trueToken:
		return "'true'"
	case falseToken:
		return "'false'"
	case numberToken:
		return "lex-num_" + string(t.Value)
	case stringToken:
		return "lex-str_" + string(t.Value)
	case commaToken:
		return "','"
	case colonToken:
		return "':'"
	case arrayOToken:
		return "'['"
	case arrayCToken:
		return "']'"
	case objectOToken:
		return "'{'"
	case objectCToken:
		return "'}'"
	default:
		return "lex-unkown"
	}
}
