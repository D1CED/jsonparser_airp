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
			{Type: objectOToken, Position: [2]int{0, 0}},
			{Type: stringToken, Value: "a", Position: [2]int{0, 1}},
			{Type: colonToken, Position: [2]int{0, 4}},
			{Type: nullToken, Position: [2]int{0, 6}},
			{Type: objectCToken, Position: [2]int{0, 10}},
		}},
		{`[false, -31.2, 5, "ab\"cd"]`, []token{
			{Type: arrayOToken, Position: [2]int{0, 0}},
			{Type: falseToken, Position: [2]int{0, 1}},
			{Type: commaToken, Position: [2]int{0, 6}},
			{Type: numberToken, Value: "-31.2", Position: [2]int{0, 8}},
			{Type: commaToken, Position: [2]int{0, 13}},
			{Type: numberToken, Value: "5", Position: [2]int{0, 15}},
			{Type: commaToken, Position: [2]int{0, 16}},
			{Type: stringToken, Value: "ab\\\"cd", Position: [2]int{0, 18}},
			{Type: arrayCToken, Position: [2]int{0, 26}},
		}},
		{`{"a": 20, "b": [true, null]}`, []token{
			{Type: objectOToken, Position: [2]int{0, 0}},
			{Type: stringToken, Value: "a", Position: [2]int{0, 1}},
			{Type: colonToken, Position: [2]int{0, 4}},
			{Type: numberToken, Value: "20", Position: [2]int{0, 6}},
			{Type: commaToken, Position: [2]int{0, 8}},
			{Type: stringToken, Value: "b", Position: [2]int{0, 10}},
			{Type: colonToken, Position: [2]int{0, 13}},
			{Type: arrayOToken, Position: [2]int{0, 15}},
			{Type: trueToken, Position: [2]int{0, 16}},
			{Type: commaToken, Position: [2]int{0, 20}},
			{Type: nullToken, Position: [2]int{0, 22}},
			{Type: arrayCToken, Position: [2]int{0, 26}},
			{Type: objectCToken, Position: [2]int{0, 27}},
		}},
		{`[0]`, []token{
			{Type: arrayOToken, Position: [2]int{0, 0}},
			{Type: numberToken, Value: "0", Position: [2]int{0, 1}},
			{Type: arrayCToken, Position: [2]int{0, 2}},
		}},
		{`{"a":{},"b":[],"c":null,"d":0,"e":""}`, []token{
			{Type: objectOToken, Position: [2]int{0, 0}},
			{Type: stringToken, Value: "a", Position: [2]int{0, 1}},
			{Type: colonToken, Position: [2]int{0, 4}},
			{Type: objectOToken, Position: [2]int{0, 5}},
			{Type: objectCToken, Position: [2]int{0, 6}},
			{Type: commaToken, Position: [2]int{0, 7}},
			{Type: stringToken, Value: "b", Position: [2]int{0, 8}},
			{Type: colonToken, Position: [2]int{0, 11}},
			{Type: arrayOToken, Position: [2]int{0, 12}},
			{Type: arrayCToken, Position: [2]int{0, 13}},
			{Type: commaToken, Position: [2]int{0, 14}},
			{Type: stringToken, Value: "c", Position: [2]int{0, 15}},
			{Type: colonToken, Position: [2]int{0, 18}},
			{Type: nullToken, Position: [2]int{0, 19}},
			{Type: commaToken, Position: [2]int{0, 23}},
			{Type: stringToken, Value: "d", Position: [2]int{0, 24}},
			{Type: colonToken, Position: [2]int{0, 27}},
			{Type: numberToken, Value: "0", Position: [2]int{0, 28}},
			{Type: commaToken, Position: [2]int{0, 29}},
			{Type: stringToken, Value: "e", Position: [2]int{0, 30}},
			{Type: colonToken, Position: [2]int{0, 33}},
			{Type: stringToken, Position: [2]int{0, 34}},
			{Type: objectCToken, Position: [2]int{0, 36}},
		}},
		{`{"index":[{"inner":[null,true]}}]`, []token{
			{Type: objectOToken, Position: [2]int{0, 0}},
			{Type: stringToken, Value: "index", Position: [2]int{0, 1}},
			{Type: colonToken, Position: [2]int{0, 8}},
			{Type: arrayOToken, Position: [2]int{0, 9}},
			{Type: objectOToken, Position: [2]int{0, 10}},
			{Type: stringToken, Value: "inner", Position: [2]int{0, 11}},
			{Type: colonToken, Position: [2]int{0, 18}},
			{Type: arrayOToken, Position: [2]int{0, 19}},
			{Type: nullToken, Position: [2]int{0, 20}},
			{Type: commaToken, Position: [2]int{0, 24}},
			{Type: trueToken, Position: [2]int{0, 25}},
			{Type: arrayCToken, Position: [2]int{0, 29}},
			{Type: objectCToken, Position: [2]int{0, 30}},
			{Type: objectCToken, Position: [2]int{0, 31}},
			{Type: arrayCToken, Position: [2]int{0, 32}},
		}},
	}
outer:
	for _, test := range tests {
		lexc, q := lex(strings.NewReader(test.have))
		for _, w := range test.want {
			tk := <-lexc
			if tk != w {
				t.Errorf("have %v, got %s, want %s", test.have, tk, w)
				q()
				continue outer
			}
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
			value: []KeyNode{
				{"a", Node{jsonType: Null}},
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
			value: []KeyNode{
				{"a", Node{jsonType: Number, value: 20.}},
				{"b", Node{jsonType: Array, value: []Node{
					{jsonType: Bool, value: true},
					{jsonType: Null},
				}}},
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
			value: []KeyNode{
				{"a", Node{jsonType: Object, value: []Node(nil)}},
				{"b", Node{jsonType: Array, value: []Node(nil)}},
				{"c", Node{jsonType: Null}},
				{"d", Node{jsonType: Number, value: 0.}},
				{"e", Node{jsonType: String, value: ""}},
			},
		}},
	}
	for i, test := range tests {
		ast, err := parse(lex(strings.NewReader(test.have)))
		if err != nil || !EqNode(ast, &test.want) {
			t.Errorf("for %v, got %v, with err: %v; %d", &test.want, ast, err, i)
		}
	}
}

func TestParseErr(t *testing.T) {
	tests := []struct {
		have string
		want ParseError
	}{
		{"", ParseError{msg: "value"}},
		{"null 5", ParseError{
			msg:    "delimiter",
			token:  token{Type: numberToken, Value: "5", Position: [2]int{0, 5}},
			before: token{Type: nullToken, Position: [2]int{0, 0}},
		}},
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
		{`{"index":[{"inner":[null,true]}}]`, ParseError{
			msg:        "array closing",
			token:      token{Type: objectCToken, Position: [2]int{0, 31}},
			before:     token{Type: objectCToken, Position: [2]int{0, 30}},
			parentType: Array,
			key:        "index.0",
		}},
	}
	for _, test := range tests {
		_, err := parse(lex(strings.NewReader(test.have)))
		pErr, ok := err.(*ParseError)
		if !ok {
			t.Fatalf("error is not of type parse error in test: %T", err)
		}
		if *pErr != test.want {
			t.Errorf("got %v, want %s, for %v", pErr, test.want.Error(), test.have)
		}
	}
}

func TestFile(t *testing.T) {
	want := &Node{jsonType: Object, value: []KeyNode{
		{"bool", Node{jsonType: Bool, value: true}},
		{"obj", Node{jsonType: Object, value: []KeyNode{
			{"v", Node{jsonType: Null, value: nil}},
		}}},
		{"values", Node{jsonType: Array, value: []Node{
			{jsonType: Object, value: []KeyNode{
				{"a", Node{jsonType: Number, value: 5.}},
				{"b", Node{jsonType: String, value: "hi"}},
				{"c", Node{jsonType: Number, value: 5.8}},
				{"d", Node{jsonType: Null, value: nil}},
				{"e", Node{jsonType: Bool, value: true}},
			}},
			{jsonType: Object, value: []KeyNode{
				{"a", Node{jsonType: Array, value: []Node{
					{jsonType: Number, value: 5.},
					{jsonType: Number, value: 6.},
					{jsonType: Number, value: 7.},
					{jsonType: Number, value: 8.},
				}}},
				{"b", Node{jsonType: String, value: "hi2"}},
				{"c", Node{jsonType: Number, value: 5.9}},
				{"d", Node{jsonType: Object, value: []KeyNode{
					{"f", Node{jsonType: String, value: "Hello there!"}},
				}}},
				{"e", Node{jsonType: Bool, value: false}},
			}},
		}}},
	}}
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
			value: []KeyNode{
				{"a", Node{jsonType: Null}},
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
			value: []KeyNode{
				{"a", Node{jsonType: Number, value: float64(20)}},
				{"b", Node{jsonType: Array, value: []Node{
					{jsonType: Bool, value: true},
					{jsonType: Null},
				}}},
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
			value: []KeyNode{
				{"a", Node{jsonType: Null}},
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
			value: []KeyNode{
				{"a", Node{jsonType: Number, value: float64(20)}},
				{"b", Node{jsonType: Array, value: []Node{
					{jsonType: Bool, value: true},
					{jsonType: Null},
				}}},
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
		{`{"index":[{"inner":[null,true]}]}`, "index.inner.0", false, ""},
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

func TestLen(t *testing.T) {
	tests := []struct {
		json string
		len  int
	}{
		{"true", 1},
		{"{}", 0},
		{`{"a":5,"b":null}`, 2},
		{"[1,2,3,4,5,6,7,8,9]", 9},
	}
	for _, test := range tests {
		n, err := NewJSON(strings.NewReader(test.json))
		if err != nil {
			t.Fatal(err)
		}
		if n.Len() != test.len {
			t.Errorf("want %v got %v for %v", test.len, n.Len(), test.json)
		}
	}
}

func TestJSON2Go(t *testing.T) {
	tests := []struct {
		have  string
		store interface{}
		want  interface{}
	}{
		{"true", new(bool), true},
		{"52", new(int), 52},
		{"3452.1", new(float64), 3452.1},
		{"3452.1", new(float32), float32(3452.1)},
		{`"Hello, World!"`, new(string), "Hello, World!"},
		{`[true, "hi"]`, &[]interface{}{}, []interface{}{true, "hi"}},
		{`[52, 420]`, &[]float64{}, []float64{52, 420}},
		{`[52, 420]`, &[]int{}, []int{52, 420}},
		{`{"a":52,"b":420}`, &map[string]int{}, map[string]int{"a": 52, "b": 420}},
		{`{"a":52,"b":true}`, &struct {
			A int  `json:"a"`
			B bool `json:"b"`
		}{}, struct {
			A int  `json:"a"`
			B bool `json:"b"`
		}{52, true}},
		{`{"Str":true,"bool":false,"This":5}`, &struct {
			Str  string `json:",string"`
			Bool bool   `json:"bool"`
			This int    `json:"-"`
		}{}, struct {
			Str  string `json:",string"`
			Bool bool   `json:"bool"`
			This int    `json:"-"`
		}{Str: "true", Bool: false}},
		{`{"a":true,"bool":false,"This":5}`, &struct {
			Str  string `json:"bool,string"`
			Bool bool   `json:"a,"`
			This int    `json:",omitempty"`
		}{}, struct {
			Str  string `json:"bool,string"`
			Bool bool   `json:"a,"`
			This int    `json:",omitempty"`
		}{Str: "false", Bool: true, This: 5}},
	}
	for _, test := range tests {
		n, err := parse(lex(strings.NewReader(test.have)))
		if err != nil {
			t.Fatalf("test setup fail: %v", err)
		}
		err = n.JSON2Go(test.store)
		if err != nil {
			t.Error(err)
			continue
		}
		got := reflect.ValueOf(test.store).Elem().Interface()
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("want %v got %v", test.want, got)
		}
	}
}
