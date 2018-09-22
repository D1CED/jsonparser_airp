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
		if ast, err := parse(lex(strings.NewReader(test.have))); err != nil || !EqNode(ast, &test.want) {
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
			token:      token{Value: "nul", Position: [2]int{0, 6}},
			before:     token{Type: colonToken, Position: [2]int{0, 4}},
			parentType: Object,
			key:        "a",
		}},
		{`{"a": null`, ParseError{
			msg:        "delimiter",
			token:      token{Type: nullToken, Position: [2]int{0, 6}},
			before:     token{Type: nullToken, Position: [2]int{0, 6}},
			parentType: Object,
			key:        "a",
		}},
		{`{"b": "\"}`, ParseError{
			msg:        "value",
			token:      token{Value: `"\"}`, Position: [2]int{0, 6}},
			before:     token{Type: colonToken, Position: [2]int{0, 4}},
			parentType: Object,
			key:        "b",
		}},
		{`{"a":[],"b":{"a". false}}`, ParseError{
			msg:        "colon",
			token:      token{Value: ".", Position: [2]int{0, 16}},
			before:     token{Type: stringToken, Value: "a", Position: [2]int{0, 13}},
			parentType: Object,
			key:        "b.a",
		}},
		{"{\"very_long\"\n <garbage>}", ParseError{
			msg:        "colon",
			token:      token{Value: "<garbage>", Position: [2]int{1, 1}},
			before:     token{Type: stringToken, Value: "very_long", Position: [2]int{0, 1}},
			parentType: Object,
			key:        "very_long",
		}},
		{"{", ParseError{
			msg:        "key",
			before:     token{Type: objectOToken, Position: [2]int{0, 0}},
			parentType: Object,
		}},
		{`[{"b":}]`, ParseError{
			msg:        "value",
			token:      token{Type: objectCToken, Position: [2]int{0, 6}},
			before:     token{Type: colonToken, Position: [2]int{0, 5}},
			parentType: Object,
			key:        "0.b",
		}},
		{`[{"b":true},false,5.2,]`, ParseError{
			msg:        "value",
			token:      token{Type: arrayCToken, Position: [2]int{0, 22}},
			before:     token{Type: commaToken, Position: [2]int{0, 21}},
			parentType: Array,
			key:        "3",
		}},
		{`abcdefghij`, ParseError{
			msg:   "value",
			token: token{Value: "abcdefghij", Position: [2]int{0, 0}},
		}},
	}
	for _, test := range tests {
		_, err := parse(lex(strings.NewReader(test.have)))
		pErr, ok := err.(*ParseError)
		if !ok {
			t.Fatal("error is not of type parse error in test")
		}
		if *pErr != test.want {
			t.Errorf("got %v, want %v, for %v", pErr, test.want, test.have)
		}
	}
}

func TestFile(t *testing.T) {
	want := &Node{jsonType: Object, value: []Node{
		{key: "bool", jsonType: Bool, value: true},
		{key: "obj", jsonType: Object, value: []Node{
			{key: "v", jsonType: Null, value: nil}}},
		{key: "values", jsonType: Array, value: []Node{
			{key: "", jsonType: Object, value: []Node{
				{key: "a", jsonType: Number, value: 5.},
				{key: "b", jsonType: String, value: "hi"},
				{key: "c", jsonType: Number, value: 5.8},
				{key: "d", jsonType: Null, value: nil},
				{key: "e", jsonType: Bool, value: true}}},
			{key: "", jsonType: Object, value: []Node{
				{key: "a", jsonType: Array, value: []Node{
					{key: "", jsonType: Number, value: 5.},
					{key: "", jsonType: Number, value: 6.},
					{key: "", jsonType: Number, value: 7.},
					{key: "", jsonType: Number, value: 8.}}},
				{key: "b", jsonType: String, value: "hi2"},
				{key: "c", jsonType: Number, value: 5.9},
				{key: "d", jsonType: Object, value: []Node{
					{key: "f", jsonType: String, value: "Hello there!"}}},
				{key: "e", jsonType: Bool, value: false}}}}}}}
	data, err := ioutil.ReadFile("testfiles/test.json")
	if err != nil {
		t.Fatalf("failed reading golden file 'testfiles/test.json': %v", err)
	}
	n, err := parse(lex(bytes.NewReader(data)))
	if err != nil || !EqNode(want, n) {
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

func TestASTStringerDebug(t *testing.T) {
	tests := []struct {
		want string
		have Node
	}{
		{`{~!"a":^null~}`, Node{
			jsonType: Object,
			value: []Node{
				{key: "a", jsonType: Null},
			},
		}},
		{`[~!false,-~!-31.2,-~!5,-~!"ab\"cd"~]`, Node{
			jsonType: Array,
			value: []Node{
				{jsonType: Bool, value: false},
				{jsonType: Number, value: -31.2},
				{jsonType: Number, value: float64(5)},
				{jsonType: String, value: "ab\\\"cd"},
			},
		}},
		{`{~!"a":^20,-~!"b":^[~!!true,-~!!null~!]~}`, Node{
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
		got := test.have.stringDebug()
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

func TestNewJSONGo(t *testing.T) {
	type myType int
	var intPtr = new(int)
	*intPtr = 50
	tests := []struct {
		have interface{}
		want string
	}{
		{nil, "null"},
		{true, "true"},
		{5, "5"},
		{myType(550022), "550022"},
		{5., "5"},
		{"Hello, World!", `"Hello, World!"`},
		{[...]int{1, 2, 3, 4}, "[1,2,3,4]"},
		{[]interface{}{nil, true, 3, "hi"}, `[null,true,3,"hi"]`},
		{map[string]interface{}{"bb": false}, `{"bb":false}`},
		{struct {
			Integer int
			a       string
		}{20, "aa"}, `{"Integer":20}`},
		{struct {
			Integer int `json:"int"`
			a       string
		}{20, "aa"}, `{"int":20}`},
		{&struct {
			Integer *int `json:"intptr"`
			a       string
		}{intPtr, "aa"}, `{"intptr":50}`},
		{&[...]uint64{6}, "[6]"},
		{[]byte("bytes"), `"bytes"`},
	}
	for _, test := range tests {
		n, err := NewJSONGo(test.have)
		if err != nil {
			t.Error(err)
		}
		if n.String() != test.want {
			t.Errorf("got %s, want %s", n, test.want)
		}
	}
}

func TestGetChild(t *testing.T) {
	tests := []struct {
		json  string
		key   string
		want  bool
		value string
	}{
		{`[null,5,"hello there"]`, "2", true, `"hello there"`},
		{`{"a":null,"b":5,"json":"hello there"}`, "json", true, `"hello there"`},
		{`{"index":{"inner":[true]}}`, "index.inner.0", true, "true"},
		// BUG(JMH): wrong closing order
		{`{"index":[{"inner":[null,true]}}]`, "index.inner.0", false, ""},
		{`{"index":[{"inner":[null,true]}]}`, "index.0.inner.1", true, "true"},
		{`{"index":{"inner":[true]}}`, "index.iner.0", false, ""},
	}
	for _, test := range tests {
		n, err := parse(lex(strings.NewReader(test.json)))
		if err != nil {
			t.Fatal(err)
		}
		if m, ok := n.GetChild(test.key); ok != test.want {
			t.Errorf("%s %s", test.json, test.key)
		} else if ok && m.String() != test.value {
			t.Errorf("%s %s", m, test.value)
		}
	}
}

func TestEqNode(t *testing.T) {
	tests := []struct {
		goval interface{}
		json  string
	}{
		{5, "5"},
		{nil, "null"},
		{"hello", `"hello"`},
		{[]bool{false, true}, "[false, true]"},
		{map[string]interface{}{
			"a":    true,
			"long": 100000,
		}, `{"long":100000,"a":true}`},
	}
	for _, test := range tests {
		n, err := NewJSONGo(test.goval)
		if err != nil {
			t.Fatal(err)
		}
		m, err := NewJSON(strings.NewReader(test.json))
		if err != nil {
			t.Fatal(err)
		}
		if !EqNode(n, m) {
			t.Error(err)
		}
	}
}

/*
func TestJSON2Go(t *testing.T) {
	tests := []struct {
		have string
		want interface{}
	}{
		{"true", true},
		{"52", 52},
		{"3452.1", 3452.1},
		{"3452.1", 3452.1},
	}
	for _, test := range tests {
		n, err := parse(lex(test.have))
		if err != nil {
			t.Fatal("test setup fail")
		}
		n
	}
}
*/
