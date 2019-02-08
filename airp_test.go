package jsonparser_airp

import (
	"bytes"
	"io/ioutil"
	"strings"
	"testing"
	"time"
)

func TestLexer(t *testing.T) {
	tests := []struct {
		have string
		want []token
	}{{
		`{"a": null}`,
		[]token{
			{Type: objectOToken, position: [2]int{0, 0}},
			{Type: stringToken, value: "a", position: [2]int{0, 1}},
			{Type: colonToken, position: [2]int{0, 4}},
			{Type: nullToken, position: [2]int{0, 6}},
			{Type: objectCToken, position: [2]int{0, 10}},
		},
	}, {
		`[false, -31.2, 5, "ab\"cd"]`,
		[]token{
			{Type: arrayOToken, position: [2]int{0, 0}},
			{Type: falseToken, position: [2]int{0, 1}},
			{Type: commaToken, position: [2]int{0, 6}},
			{Type: numberToken, value: "-31.2", position: [2]int{0, 8}},
			{Type: commaToken, position: [2]int{0, 13}},
			{Type: numberToken, value: "5", position: [2]int{0, 15}},
			{Type: commaToken, position: [2]int{0, 16}},
			{Type: stringToken, value: "ab\\\"cd", position: [2]int{0, 18}},
			{Type: arrayCToken, position: [2]int{0, 26}},
		},
	}, {
		`{"a": 20, "b": [true, null]}`,
		[]token{
			{Type: objectOToken, position: [2]int{0, 0}},
			{Type: stringToken, value: "a", position: [2]int{0, 1}},
			{Type: colonToken, position: [2]int{0, 4}},
			{Type: numberToken, value: "20", position: [2]int{0, 6}},
			{Type: commaToken, position: [2]int{0, 8}},
			{Type: stringToken, value: "b", position: [2]int{0, 10}},
			{Type: colonToken, position: [2]int{0, 13}},
			{Type: arrayOToken, position: [2]int{0, 15}},
			{Type: trueToken, position: [2]int{0, 16}},
			{Type: commaToken, position: [2]int{0, 20}},
			{Type: nullToken, position: [2]int{0, 22}},
			{Type: arrayCToken, position: [2]int{0, 26}},
			{Type: objectCToken, position: [2]int{0, 27}},
		},
	}, {
		`[0]`,
		[]token{
			{Type: arrayOToken, position: [2]int{0, 0}},
			{Type: numberToken, value: "0", position: [2]int{0, 1}},
			{Type: arrayCToken, position: [2]int{0, 2}},
		},
	}, {
		`{"a":{},"b":[],"c":null,"d":0,"e":""}`,
		[]token{
			{Type: objectOToken, position: [2]int{0, 0}},
			{Type: stringToken, value: "a", position: [2]int{0, 1}},
			{Type: colonToken, position: [2]int{0, 4}},
			{Type: objectOToken, position: [2]int{0, 5}},
			{Type: objectCToken, position: [2]int{0, 6}},
			{Type: commaToken, position: [2]int{0, 7}},
			{Type: stringToken, value: "b", position: [2]int{0, 8}},
			{Type: colonToken, position: [2]int{0, 11}},
			{Type: arrayOToken, position: [2]int{0, 12}},
			{Type: arrayCToken, position: [2]int{0, 13}},
			{Type: commaToken, position: [2]int{0, 14}},
			{Type: stringToken, value: "c", position: [2]int{0, 15}},
			{Type: colonToken, position: [2]int{0, 18}},
			{Type: nullToken, position: [2]int{0, 19}},
			{Type: commaToken, position: [2]int{0, 23}},
			{Type: stringToken, value: "d", position: [2]int{0, 24}},
			{Type: colonToken, position: [2]int{0, 27}},
			{Type: numberToken, value: "0", position: [2]int{0, 28}},
			{Type: commaToken, position: [2]int{0, 29}},
			{Type: stringToken, value: "e", position: [2]int{0, 30}},
			{Type: colonToken, position: [2]int{0, 33}},
			{Type: stringToken, position: [2]int{0, 34}},
			{Type: objectCToken, position: [2]int{0, 36}},
		},
	}, {
		`{"index":[{"inner":[null,true]}}]`,
		[]token{
			{Type: objectOToken, position: [2]int{0, 0}},
			{Type: stringToken, value: "index", position: [2]int{0, 1}},
			{Type: colonToken, position: [2]int{0, 8}},
			{Type: arrayOToken, position: [2]int{0, 9}},
			{Type: objectOToken, position: [2]int{0, 10}},
			{Type: stringToken, value: "inner", position: [2]int{0, 11}},
			{Type: colonToken, position: [2]int{0, 18}},
			{Type: arrayOToken, position: [2]int{0, 19}},
			{Type: nullToken, position: [2]int{0, 20}},
			{Type: commaToken, position: [2]int{0, 24}},
			{Type: trueToken, position: [2]int{0, 25}},
			{Type: arrayCToken, position: [2]int{0, 29}},
			{Type: objectCToken, position: [2]int{0, 30}},
			{Type: objectCToken, position: [2]int{0, 31}},
			{Type: arrayCToken, position: [2]int{0, 32}},
		},
	}}
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
	}{{
		`{"a": nul}`,
		token{
			value:    "nul",
			position: [2]int{0, 6},
		},
	}, {
		`{"a": "\"}`,
		token{
			value:    `"\"}`,
			position: [2]int{0, 6},
		},
	}, {
		`{"a". false}`,
		token{
			value:    ".",
			position: [2]int{0, 4},
		},
	}, {
		"{\"a\"\n <garbage>}",
		token{
			value:    "<garbage>",
			position: [2]int{1, 1},
		},
	}}
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
	}{{
		`{"a": null}`,
		Node{
			jsonType: Object,
			value: []KeyNode{
				{"a", &Node{jsonType: Null}},
			},
		},
	}, {
		`[false, -31.2, 5, "ab\"cd"]`,
		Node{
			jsonType: Array,
			value: []*Node{
				{jsonType: Bool, value: false},
				{jsonType: Number, value: -31.2},
				{jsonType: Number, value: 5.},
				{jsonType: String, value: "ab\\\"cd"},
			},
		},
	}, {
		`{"a": 20, "b": [true, null]}`,
		Node{
			jsonType: Object,
			value: []KeyNode{
				{"a", &Node{jsonType: Number, value: 20.}},
				{"b", &Node{jsonType: Array, value: []*Node{
					{jsonType: Bool, value: true},
					{jsonType: Null},
				}}},
			},
		},
	}, {
		`[0]`,
		Node{
			jsonType: Array,
			value: []*Node{
				{jsonType: Number, value: 0.},
			},
		},
	}, {
		`{"a":{},"b":[],"c":null,"d":0,"e":""}`,
		Node{
			jsonType: Object,
			value: []KeyNode{
				{"a", &Node{jsonType: Object, value: []KeyNode(nil)}},
				{"b", &Node{jsonType: Array, value: []*Node(nil)}},
				{"c", &Node{jsonType: Null}},
				{"d", &Node{jsonType: Number, value: 0.}},
				{"e", &Node{jsonType: String, value: ""}},
			},
		},
	}}
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
	}{{
		"",
		ParseError{msg: "value"},
	}, {
		"null 5",
		ParseError{
			msg:    "delimiter",
			token:  token{Type: numberToken, value: "5", position: [2]int{0, 5}},
			before: token{Type: nullToken, position: [2]int{0, 0}},
		},
	}, {
		`{"a": nul}`,
		ParseError{
			msg:        "value",
			token:      token{value: "nul", position: [2]int{0, 6}},
			before:     token{Type: colonToken, position: [2]int{0, 4}},
			parentType: Object,
			key:        "a",
		},
	}, {
		`{"a": null`,
		ParseError{
			msg:        "delimiter",
			token:      token{Type: nullToken, position: [2]int{0, 6}},
			before:     token{Type: nullToken, position: [2]int{0, 6}},
			parentType: Object,
			key:        "a",
		},
	}, {
		`{"b": "\"}`,
		ParseError{
			msg:        "value",
			token:      token{value: `"\"}`, position: [2]int{0, 6}},
			before:     token{Type: colonToken, position: [2]int{0, 4}},
			parentType: Object,
			key:        "b",
		},
	}, {
		`{"a":[],"b":{"a". false}}`,
		ParseError{
			msg:        "colon",
			token:      token{value: ".", position: [2]int{0, 16}},
			before:     token{Type: stringToken, value: "a", position: [2]int{0, 13}},
			parentType: Object,
			key:        "b.a",
		},
	}, {
		"{\"very_long\"\n <garbage>}",
		ParseError{
			msg:        "colon",
			token:      token{value: "<garbage>", position: [2]int{1, 1}},
			before:     token{Type: stringToken, value: "very_long", position: [2]int{0, 1}},
			parentType: Object,
			key:        "very_long",
		},
	}, {
		"{",
		ParseError{
			msg:        "key",
			before:     token{Type: objectOToken, position: [2]int{0, 0}},
			parentType: Object,
		},
	}, {
		`[{"b":}]`,
		ParseError{
			msg:        "value",
			token:      token{Type: objectCToken, position: [2]int{0, 6}},
			before:     token{Type: colonToken, position: [2]int{0, 5}},
			parentType: Object,
			key:        "0.b",
		},
	}, {
		`[{"b":true},false,5.2,]`,
		ParseError{
			msg:        "value",
			token:      token{Type: arrayCToken, position: [2]int{0, 22}},
			before:     token{Type: commaToken, position: [2]int{0, 21}},
			parentType: Array,
			key:        "3",
		},
	}, {
		`abcdefghij`,
		ParseError{
			msg:   "value",
			token: token{value: "abcdefghij", position: [2]int{0, 0}},
		},
	}, {
		`{"index":[{"inner":[null,true]}}]`,
		ParseError{
			msg:        "array closing",
			token:      token{Type: objectCToken, position: [2]int{0, 31}},
			before:     token{Type: objectCToken, position: [2]int{0, 30}},
			parentType: Array,
			key:        "index.0",
		},
	}, {
		`{"a":null,"a":true}`,
		ParseError{
			msg:        "unique key",
			parentType: Object,
			token:      token{Type: stringToken, value: "a", position: [2]int{0, 10}},
			before:     token{Type: commaToken, position: [2]int{0, 9}},
		},
	}}
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

func TestGetKey(t *testing.T) {
	tests := []struct {
		json  string
		key   string
		value interface{}
	}{{
		`[true]`, "0", true,
	}, {
		`[true, 25]`, "1", 25.,
	}, {
		`{"a":true}`, "a", true,
	}, {
		`{"long":5,"a":true}`, "a", true,
	}, {
		`{"long":5,"a":true}`, "long", 5.,
	}, {
		`[{"long":5,"a":true},{"hi":"yes"}]`, "1.hi", "yes",
	}, {
		`{"long":[5,null],"a":true}`, "a", true,
	}, {
		`[[[["inner"]]]]`, "0.0.0.0", "inner",
	}, {
		`[[null,[null,null,["inner"]]]]`, "0.1.2.0", "inner",
	}, {
		`{"a":{"a":{"a":{"a":"inner"}}}}`, "a.a.a.a", "inner",
	}, {
		`{"a":{"a":null,"b":{"a":null,"b":null,"c":{"a":"inner"}}}}`, "a.b.c.a", "inner",
	}, {
		`{"a":[5]}`, "a.0", 5.,
	}, {
		`[{"a":5}]`, "0.a", 5.,
	}, {
		`{"a":[5,"hi"]}`, "a.1", "hi",
	}, {
		`[[1],null]`, "0.0", 1.,
	}, {
		`[null,[1],null]`, "1.0", 1.,
	}, {
		`{"a":{"c":5},"b":"hi"}`, "a.c", 5.,
	}, {
		`{"a":[5],"b":"hi"}`, "a.0", 5.,
	}, {
		`{"long":[5,null],"a":true}`, "long.1", nil,
	}, {
		`{"go":[{"long":[5,null],"a":true}]}`, "go.0.long.1", nil,
	}}
	for i, test := range tests {
		n, err := NewJSONString(test.json)
		if err != nil {
			t.Fatal(err)
		}
		m, ok := n.GetChild(test.key)
		if !ok {
			t.Errorf("key %s not found in %s", test.key, test.json)
		}
		if v, err := m.Value(); err != nil || v != test.value {
			t.Errorf("got %v want %v; with err: %v", v, test.value, err)
		}
		if k := m.Key(); k != test.key {
			t.Errorf("got %s want %s", k, test.key)
		}
		t.Run("memloc", func(t *testing.T) {
			if testing.Short() {
				t.Skip("skip memloc in short test")
			}
			if o, _ := n.GetChild(test.key); m != o {
				t.Errorf("expected same memmory location: second time")
			}
			if keys := strings.Split(test.key, "."); len(keys) > 1 {
				parKey := strings.Join(keys[:len(keys)-1], ".")
				par, _ := n.GetChild(parKey)
				if !EqNode(par, m.parent) {
					t.Fatalf("expected same values: %s %s", par, m.parent)
				}
				if par != m.parent {
					t.Errorf("expected same memmory location: parent")
				}
			}
			if maxParent(m) != n {
				t.Errorf("xx1xx")
			}
		})
		t.Run("debug", func(t *testing.T) {
			if i != 14 {
				t.Skip("only test testcase 14")
			}

			if !cyclicTest(m.parent) {
				t.Error("non-cyclic upper")
			}
			if !cyclicTest(m) {
				t.Error("non-cyclic lowe")
			}

			if !EqNode(n, maxParent(m)) {
				t.Errorf("upper %s == %s\n", n, maxParent(m))
			}
			if n != maxParent(m) {
				t.Errorf("upper %p == %p\n", n, maxParent(m))
			}
			if !EqNode(m, n.value.([]*Node)[0].value.([]*Node)[0]) {
				t.Errorf("lower %s == %s\n", m, n.value.([]*Node)[0].value.([]*Node)[0])
			}
			if m != n.value.([]*Node)[0].value.([]*Node)[0] {
				t.Errorf("lower %p == %p\n", m, n.value.([]*Node)[0].value.([]*Node)[0])
			}

			if testing.Verbose() {
				n.SetChild(StandaloneNode("0.0", "2"))
				t.Logf("%s %s %s\n", n, maxParent(m), m)
				m.value = 3.
				t.Logf("%s, %s %s\n", n, maxParent(m), m)
				t.Log(n, n.value.([]*Node)[0].parent, m.parent.parent)
				t.Log("--")
				t.Logf("n-addr %p; n-child %p\n", n, n.value.([]*Node)[0])
				t.Logf("m-addr %p; m-parent %p\n", m, m.parent)
				t.Logf("1-n %p; 1-m %p\n", n.value.([]*Node)[0].parent, n.value.([]*Node)[0].value.([]*Node)[0])
				t.Logf("2-n %p; 2-m %p\n", m.parent.parent, m.parent.value.([]*Node)[0])
				t.Log("--")
				x, y, z := n, n.value.([]*Node)[0].parent, m.parent.parent
				t.Logf("%p %p %p\n", x.value.([]*Node)[0], y.value.([]*Node)[0], z.value.([]*Node)[0])

				m.value = 1.
			}
		})
	}
}

func TestFile(t *testing.T) {
	want := &Node{jsonType: Object, value: []KeyNode{
		{"bool", &Node{jsonType: Bool, value: true}},
		{"obj", &Node{jsonType: Object, value: []KeyNode{
			{"v", &Node{jsonType: Null, value: nil}},
		}}},
		{"values", &Node{jsonType: Array, value: []*Node{
			{jsonType: Object, value: []KeyNode{
				{"a", &Node{jsonType: Number, value: 5.}},
				{"b", &Node{jsonType: String, value: "hi"}},
				{"c", &Node{jsonType: Number, value: 5.8}},
				{"d", &Node{jsonType: Null, value: nil}},
				{"e", &Node{jsonType: Bool, value: true}},
			}},
			{jsonType: Object, value: []KeyNode{
				{"a", &Node{jsonType: Array, value: []*Node{
					{jsonType: Number, value: 5.},
					{jsonType: Number, value: 6.},
					{jsonType: Number, value: 7.},
					{jsonType: Number, value: 8.},
				}}},
				{"b", &Node{jsonType: String, value: "hi2"}},
				{"c", &Node{jsonType: Number, value: 5.9}},
				{"d", &Node{jsonType: Object, value: []KeyNode{
					{"f", &Node{jsonType: String, value: "Hello there!"}},
				}}},
				{"e", &Node{jsonType: Bool, value: false}},
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

func TestASTStringer(t *testing.T) {
	tests := []struct {
		want string
		have Node
	}{{
		`{"a":null}`,
		Node{
			jsonType: Object,
			value: []KeyNode{
				{"a", &Node{jsonType: Null}},
			},
		},
	}, {
		`[false,-31.2,5,"ab\"cd"]`,
		Node{
			jsonType: Array,
			value: []*Node{
				{jsonType: Bool, value: false},
				{jsonType: Number, value: -31.2},
				{jsonType: Number, value: float64(5)},
				{jsonType: String, value: "ab\\\"cd"},
			},
		},
	}, {
		`{"a":20,"b":[true,null]}`,
		Node{
			jsonType: Object,
			value: []KeyNode{
				{"a", &Node{jsonType: Number, value: float64(20)}},
				{"b", &Node{jsonType: Array, value: []*Node{
					{jsonType: Bool, value: true},
					{jsonType: Null},
				}}},
			},
		},
	}}
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
	}{{
		`{~!"a":^null~}`,
		Node{
			jsonType: Object,
			value: []KeyNode{
				{"a", &Node{jsonType: Null}},
			},
		},
	}, {
		`[~!false,-~!-31.2,-~!5,-~!"ab\"cd"~]`,
		Node{
			jsonType: Array,
			value: []*Node{
				{jsonType: Bool, value: false},
				{jsonType: Number, value: -31.2},
				{jsonType: Number, value: float64(5)},
				{jsonType: String, value: "ab\\\"cd"},
			},
		},
	}, {
		`{~!"a":^20,-~!"b":^[~!!true,-~!!null~!]~}`,
		Node{
			jsonType: Object,
			value: []KeyNode{
				{"a", &Node{jsonType: Number, value: float64(20)}},
				{"b", &Node{jsonType: Array, value: []*Node{
					{jsonType: Bool, value: true},
					{jsonType: Null},
				}}},
			},
		},
	}}
	for _, test := range tests {
		b := &strings.Builder{}
		test.have.format(b, "!", "~", "-", "^")
		got := b.String()
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
		t.Error("lexer not stopped after receiving quit")
	}
}
