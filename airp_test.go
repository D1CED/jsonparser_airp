package jsonparser_airp

import (
	"io/ioutil"
	"reflect"
	"testing"
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
	}
	for _, test := range tests {
		ch := lex(test.have)
		first := true
		for _, w := range test.want {
			tk := <-ch
			if tk.Type != w.Type || tk.Value != w.Value {
				t.Errorf("have %v, got %s, want %s", test.have, tk.String(), w)
			}
			if !first && tk.Position == [2]int{} {
				t.Errorf("token %s is missing position", tk.String())
			}
			first = false
		}
		if tk, ok := <-ch; ok {
			t.Errorf("expected nothing, got %s", tk)
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
		for tk := range lex(test.have) {
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
			Children: []Node{
				{key: "a", jsonType: Null},
			},
		}},
		{`[false, -31.2, 5, "ab\"cd"]`, Node{
			jsonType: Array,
			Children: []Node{
				{jsonType: Bool, value: "false"},
				{jsonType: Number, value: "-31.2"},
				{jsonType: Number, value: "5"},
				{jsonType: String, value: "ab\\\"cd"},
			},
		}},
		{`{"a": 20, "b": [true, null]}`, Node{
			jsonType: Object,
			Children: []Node{
				{key: "a", jsonType: Number, value: "20"},
				{key: "b", jsonType: Array, Children: []Node{
					{jsonType: Bool, value: "true"},
					{jsonType: Null},
				}},
			},
		}},
		{`[0]`, Node{
			jsonType: Array,
			Children: []Node{
				{jsonType: Number, value: "0"},
			},
		}},
	}
	for _, test := range tests {
		ast, err := parse(lex(test.have))
		if err != nil {
			t.Error(err)
		}
		if !eqNode(ast, &test.want) {
			t.Errorf("for %v got %v", test.have, ast)
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
			Token:      token{Value: ".", Position: [2]int{0, 17}},
			before:     token{Type: stringToken, Value: "a", Position: [2]int{0, 14}},
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
		_, err := parse(lex(test.have))
		if *(err.(*ParseError)) != test.want {
			t.Errorf("got %v, want %v, for %v", (err.(*ParseError)), test.want, test.have)
		}
	}
}

func TestFile(t *testing.T) {
	want := &Node{key: "", jsonType: 6, value: "", Children: []Node{
		{key: "bool", jsonType: 2, value: "true"},
		{key: "obj", jsonType: 6, value: "", Children: []Node{
			{key: "v", jsonType: 1, value: ""}}},
		{key: "values", jsonType: 5, value: "", Children: []Node{
			{key: "", jsonType: 6, value: "", Children: []Node{
				{key: "a", jsonType: 3, value: "5"},
				{key: "b", jsonType: 4, value: "hi"},
				{key: "c", jsonType: 3, value: "5.8"},
				{key: "d", jsonType: 1, value: ""},
				{key: "e", jsonType: 2, value: "true"}}},
			{key: "", jsonType: 6, value: "", Children: []Node{
				{key: "a", jsonType: 5, value: "", Children: []Node{
					{key: "", jsonType: 3, value: "5"},
					{key: "", jsonType: 3, value: "6"},
					{key: "", jsonType: 3, value: "7"},
					{key: "", jsonType: 3, value: "8"}}},
				{key: "b", jsonType: 4, value: "hi2"},
				{key: "c", jsonType: 3, value: "5.9"},
				{key: "d", jsonType: 6, value: "", Children: []Node{
					{key: "f", jsonType: 4, value: "Hello there!"}}},
				{key: "e", jsonType: 2, value: "false"}}}}}}}
	data, err := ioutil.ReadFile("testfiles/test.json")
	if err != nil {
		t.Error(err)
	}
	n, err := parse(lex(string(data)))
	if err != nil {
		t.Error(err)
	}
	if !eqNode(want, n) {
		t.Error("WRONG!")
	}
}

func TestValue(t *testing.T) {
	tests := []struct {
		have string
		want interface{}
	}{
		{`{"a": null}`, map[string]interface{}{"a": nil}},
		{`[false, -31.2, 5, "ab\"cd"]`, []interface{}{
			false, -31.2, float64(5), "ab\\\"cd",
		}},
		{`{"a": 20, "b": [true, null]}`, map[string]interface{}{
			"a": float64(20), "b": []interface{}{true, nil},
		}},
	}
	for _, test := range tests {
		ast, _ := parse(lex(test.have))
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
			Children: []Node{
				{key: "a", jsonType: Null},
			},
		}},
		{`[false,-31.2,5,"ab\"cd"]`, Node{
			jsonType: Array,
			Children: []Node{
				{jsonType: Bool, value: "false"},
				{jsonType: Number, value: "-31.2"},
				{jsonType: Number, value: "5"},
				{jsonType: String, value: "ab\\\"cd"},
			},
		}},
		{`{"a":20,"b":[true,null]}`, Node{
			jsonType: Object,
			Children: []Node{
				{key: "a", jsonType: Number, value: "20"},
				{key: "b", jsonType: Array, Children: []Node{
					{jsonType: Bool, value: "true"},
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
