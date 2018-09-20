package jsonparser_airp

import "fmt"

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
	Type     tokenType
	Value    string
	Position [2]int
}

func newToken(b rune, r, c int) token {
	switch b {
	case '{':
		return token{Type: objectOToken, Position: [2]int{r, c}}
	case '}':
		return token{Type: objectCToken, Position: [2]int{r, c}}
	case '[':
		return token{Type: arrayOToken, Position: [2]int{r, c}}
	case ']':
		return token{Type: arrayCToken, Position: [2]int{r, c}}
	case ':':
		return token{Type: colonToken, Position: [2]int{r, c}}
	case ',':
		return token{Type: commaToken, Position: [2]int{r, c}}
	default:
		panic("only single byte tokens allowed")
	}
}

// String generates a readable form of a token meant for debuging.
func (t token) String() string {
	switch t.Type {
	case nullToken:
		return "<null>"
	case trueToken:
		return "<true>"
	case falseToken:
		return "<false>"
	case numberToken:
		return "<num " + t.Value + ">"
	case stringToken:
		return `<str "` + t.Value + `">`
	case commaToken:
		return "<,>"
	case colonToken:
		return "<:>"
	case arrayOToken:
		return "<[>"
	case arrayCToken:
		return "<]>"
	case objectOToken:
		return "<{>"
	case objectCToken:
		return "<}>"
	case errToken:
		return "<err " + t.Value + ">"
	default:
		return "<unkown " + t.Value + ">"
	}
}

// Error implements the error interface for token.
func (t token) Error() string {
	if t.Type == errToken {
		return fmt.Sprintf("%d:%d '%v'", t.Position[0], t.Position[1], t.Value)
	}
	return fmt.Sprintf("%d:%d %v", t.Position[0], t.Position[1], t.String())
}
