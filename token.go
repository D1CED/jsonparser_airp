package airp

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
	value    string
	position [2]int
}

func newToken(b rune, r, c int) token {
	switch b {
	case '{':
		return token{Type: objectOToken, position: [2]int{r, c}}
	case '}':
		return token{Type: objectCToken, position: [2]int{r, c}}
	case '[':
		return token{Type: arrayOToken, position: [2]int{r, c}}
	case ']':
		return token{Type: arrayCToken, position: [2]int{r, c}}
	case ':':
		return token{Type: colonToken, position: [2]int{r, c}}
	case ',':
		return token{Type: commaToken, position: [2]int{r, c}}
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
		return "<num " + t.value + ">"
	case stringToken:
		return `<str "` + t.value + `">`
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
		return "<err " + t.value + ">"
	default:
		return "<unknown " + t.value + ">"
	}
}

// Error implements the error interface for token.
func (t token) Error() string {
	if t.Type == errToken {
		return fmt.Sprintf("%d:%d '%v'", t.position[0], t.position[1], t.value)
	}
	return fmt.Sprintf("%d:%d %v", t.position[0], t.position[1], t.String())
}
