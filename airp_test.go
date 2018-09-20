package jsonparser_airp

import (
	"bytes"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestLexer(t *testing.T) {
	tests := []struct {
		have string
		want []token
	}{
		{`{"a": null}`, []token{
			{Type: objectOToken},
			{Type: stringToken, Value: "a"},
			{Type: colonToken},
			{Type: nullToken},
			{Type: objectCToken},
		}},
		{`[false, -31.2, 5, "ab\"cd"]`, []token{
			{Type: arrayOToken},
			{Type: falseToken},
			{Type: commaToken},
			{Type: numberToken, Value: "-31.2"},
			{Type: commaToken},
			{Type: numberToken, Value: "5"},
			{Type: commaToken},
			{Type: stringToken, Value: "ab\\\"cd"},
			{Type: arrayCToken},
		}},
		{`{"a": 20, "b": [true, null]}`, []token{
			{Type: objectOToken},
			{Type: stringToken, Value: "a"},
			{Type: colonToken},
			{Type: numberToken, Value: "20"},
			{Type: commaToken},
			{Type: stringToken, Value: "b"},
			{Type: colonToken},
			{Type: arrayOToken},
			{Type: trueToken},
			{Type: commaToken},
			{Type: nullToken},
			{Type: arrayCToken},
			{Type: objectCToken},
		}},
		{`[0]`, []token{
			{Type: arrayOToken},
			{Type: numberToken, Value: "0"},
			{Type: arrayCToken},
		}},
		{`{"a":{},"b":[],"c":null,"d":0,"e":""}`, []token{
			{Type: objectOToken},
			{Type: stringToken, Value: "a"},
			{Type: colonToken},
			{Type: objectOToken},
			{Type: objectCToken},
			{Type: commaToken},
			{Type: stringToken, Value: "b"},
			{Type: colonToken},
			{Type: arrayOToken},
			{Type: arrayCToken},
			{Type: commaToken},
			{Type: stringToken, Value: "c"},
			{Type: colonToken},
			{Type: nullToken},
			{Type: commaToken},
			{Type: stringToken, Value: "d"},
			{Type: colonToken},
			{Type: numberToken, Value: "0"},
			{Type: commaToken},
			{Type: stringToken, Value: "e"},
			{Type: colonToken},
			{Type: stringToken},
			{Type: objectCToken},
		}},
	}
	for _, test := range tests {
		lexc, q := lex(strings.NewReader(test.have))
		first := true
		for _, w := range test.want {
			tk := <-lexc
			if tk.Type != w.Type || tk.Value != w.Value {
				t.Errorf("have %v, got %s, want %s", test.have, tk.String(), w)
				q()
				return
			}
			if !first && tk.Position == [2]int{} {
				t.Errorf("token %s is missing position", tk.String())
				q()
				return
			}
			first = false
		}
		if tk, ok := <-lexc; ok {
			t.Errorf("expected nothing, got %s", tk.String())
		}
	}
}

func TestLexeErr(t *testing.T) {
	tests := []struct {
		have string
		want token
	}{
		{`{"a": nul}`, token{
			Value:    "nul",
			Position: [2]int{0, 6},
		}},
		{`{"a": "\"}`, token{
			Value:    `"\"}`,
			Position: [2]int{0, 6},
		}},
		{`{"a". false}`, token{
			Value:    ".",
			Position: [2]int{0, 4},
		}},
		{"{\"a\"\n <garbage>}", token{
			Value:    "<garbage>",
			Position: [2]int{1, 1},
		}},
	}
	for _, test := range tests {
		var have token
		lexc, _ := lex(strings.NewReader(test.have))
		for tk := range lexc {
			have = tk
		}
		if have != test.want {
			t.Errorf("got %v, want %v, for %v", have.Error(), test.want, test.have)
		}
	}
}

func TestParser(t *testing.T) {
	tests := []struct {
		have string
		want Node
	}{
		{`{"a": null}`, Node{
			jsonType: Object,
			value: []Node{
				{key: "a", jsonType: Null},
			},
		}},
		{`[false, -31.2, 5, "ab\"cd"]`, Node{
			jsonType: Array,
			value: []Node{
				{jsonType: Bool, value: false},
				{jsonType: Number, value: -31.2},
				{jsonType: Number, value: 5.},
				{jsonType: String, value: "ab\\\"cd"},
			},
		}},
		{`{"a": 20, "b": [true, null]}`, Node{
			jsonType: Object,
			value: []Node{
				{key: "a", jsonType: Number, value: 20.},
				{key: "b", jsonType: Array, value: []Node{
					{jsonType: Bool, value: true},
					{jsonType: Null},
				}},
			},
		}},
		{`[0]`, Node{
			jsonType: Array,
			value: []Node{
				{jsonType: Number, value: 0.},
			},
		}},
		{`{"a":{},"b":[],"c":null,"d":0,"e":""}`, Node{
			jsonType: Object,
			value: []Node{
				{key: "a", jsonType: Object, value: []Node(nil)},
				{key: "b", jsonType: Array, value: []Node(nil)},
				{key: "c", jsonType: Null},
				{key: "d", jsonType: Number, value: 0.},
				{key: "e", jsonType: String, value: ""},
			},
		}},
	}
	for i, test := range tests {
		if ast, err := parse(lex(strings.NewReader(test.have))); err != nil || !eqNode(ast, &test.want) {
			t.Errorf("for %v, got %v, with err: %v; %d", &test.want, ast, err, i)
		}
	}
}

func TestParseErr(t *testing.T) {
	tests := []struct {
		have string
		want ParseError
	}{
		{`{"a": nul}`, ParseError{
			msg:        "value",
			Token:      token{Value: "nul", Position: [2]int{0, 6}},
			before:     token{Type: colonToken, Position: [2]int{0, 4}},
			ParentType: Object,
			Key:        "a",
		}},
		{`{"b": "\"}`, ParseError{
			msg:        "value",
			Token:      token{Value: `"\"}`, Position: [2]int{0, 6}},
			before:     token{Type: colonToken, Position: [2]int{0, 4}},
			ParentType: Object,
			Key:        "b",
		}},
		{`{"a":[],"b":{"a". false}}`, ParseError{
			msg:        "colon",
			Token:      token{Value: ".", Position: [2]int{0, 16}},
			before:     token{Type: stringToken, Value: "a", Position: [2]int{0, 13}},
			ParentType: Object,
			Key:        "b.a",
		}},
		{"{\"very_long\"\n <garbage>}", ParseError{
			msg:        "colon",
			Token:      token{Value: "<garbage>", Position: [2]int{1, 1}},
			before:     token{Type: stringToken, Value: "very_long", Position: [2]int{0, 1}},
			ParentType: Object,
			Key:        "very_long",
		}},
		{"{", ParseError{
			msg:        "key",
			before:     token{Type: objectOToken, Position: [2]int{0, 0}},
			ParentType: Object,
		}},
		{`[{"b":}]`, ParseError{
			msg:        "value",
			Token:      token{Type: objectCToken, Position: [2]int{0, 6}},
			before:     token{Type: colonToken, Position: [2]int{0, 5}},
			ParentType: Object,
			Key:        "0.b",
		}},
	}
	for _, test := range tests {
		_, err := parse(lex(strings.NewReader(test.have)))
		pErr, ok := err.(*ParseError)
		if !ok {
			t.Fatal("error is not of type parse error in test")
		}
		if *pErr != test.want {
			t.Errorf("got %#v, want %v, for %v", pErr, test.want, test.have)
		}
	}
}

func TestFile(t *testing.T) {
	want := &Node{jsonType: 6, value: []Node{
		{key: "bool", jsonType: 2, value: true},
		{key: "obj", jsonType: 6, value: []Node{
			{key: "v", jsonType: 1, value: nil}}},
		{key: "values", jsonType: 5, value: []Node{
			{key: "", jsonType: 6, value: []Node{
				{key: "a", jsonType: 3, value: 5.},
				{key: "b", jsonType: 4, value: "hi"},
				{key: "c", jsonType: 3, value: 5.8},
				{key: "d", jsonType: 1, value: nil},
				{key: "e", jsonType: 2, value: true}}},
			{key: "", jsonType: 6, value: []Node{
				{key: "a", jsonType: 5, value: []Node{
					{key: "", jsonType: 3, value: 5.},
					{key: "", jsonType: 3, value: 6.},
					{key: "", jsonType: 3, value: 7.},
					{key: "", jsonType: 3, value: 8.}}},
				{key: "b", jsonType: 4, value: "hi2"},
				{key: "c", jsonType: 3, value: 5.9},
				{key: "d", jsonType: 6, value: []Node{
					{key: "f", jsonType: 4, value: "Hello there!"}}},
				{key: "e", jsonType: 2, value: false}}}}}}}
	data, err := ioutil.ReadFile("testfiles/test.json")
	if err != nil {
		t.Fatalf("failed reading golden file 'testfiles/test.json': %v", err)
	}
	n, err := parse(lex(bytes.NewReader(data)))
	if err != nil || !eqNode(want, n) {
		t.Errorf("test failed with error: %v", err)
	}
}

func TestValue(t *testing.T) {
	tests := []struct {
		have string
		want interface{}
	}{
		{`{"a": null}`, map[string]interface{}{"a": nil}},
		{`[false, -31.2, 5, "ab\"cd"]`, []interface{}{
			false, -31.2, 5., "ab\\\"cd",
		}},
		{`{"a": 20, "b": [true, null]}`, map[string]interface{}{
			"a": 20., "b": []interface{}{true, nil},
		}},
	}
	for _, test := range tests {
		ast, _ := parse(lex(strings.NewReader(test.have)))
		itf, err := ast.Value()
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(itf, test.want) {
			t.Errorf("want %v, got %v", test.want, itf)
		}
	}
}

func TestASTStringer(t *testing.T) {
	tests := []struct {
		want string
		have Node
	}{
		{`{"a":null}`, Node{
			jsonType: Object,
			value: []Node{
				{key: "a", jsonType: Null},
			},
		}},
		{`[false,-31.2,5,"ab\"cd"]`, Node{
			jsonType: Array,
			value: []Node{
				{jsonType: Bool, value: false},
				{jsonType: Number, value: -31.2},
				{jsonType: Number, value: float64(5)},
				{jsonType: String, value: "ab\\\"cd"},
			},
		}},
		{`{"a":20,"b":[true,null]}`, Node{
			jsonType: Object,
			value: []Node{
				{key: "a", jsonType: Number, value: float64(20)},
				{key: "b", jsonType: Array, value: []Node{
					{jsonType: Bool, value: true},
					{jsonType: Null},
				}},
			},
		}},
	}
	for _, test := range tests {
		got := test.have.String()
		if got != test.want {
			t.Errorf("want: %s, got: %s", test.want, got)
		}
	}
}

func TestLexQuit(t *testing.T) {
	lexc, q := lex(strings.NewReader(`["Hello, World!", 0, true]`))
	if cap(lexc) != 1 {
		t.Fatal("lex-channel must have capacity of 1")
	}
	time.Sleep(time.Millisecond) // fill channel
	q()                          // quit lexer
	time.Sleep(time.Millisecond) // wait for quit
	if len(lexc) != 1 {
		t.Fatal("lex-channel must have length of 1")
	}
	<-lexc // empty channel (length 1)
	if _, ok := <-lexc; ok {
		t.Error("lexer not stopped after recieving quit")
	}
}
