package jsonparser_airp_test

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/andreyvit/diff"
	airp "github.com/d1ced/jsonparser_airp"
)

func TestFile2(t *testing.T) {
	f, err := os.Open("testfiles/json.org_example4.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	n, err := airp.NewJSONReader(f)
	if err != nil {
		t.Error(err)
	}
	if n.Total() != 87 {
		t.Errorf("want 87, got %d", n.Total())
	}

	m, ok := n.GetChild("web-app.servlet.1.init-param.mailHost")
	if v, _ := m.Value(); !ok || v != "mail1" {
		t.Errorf("%v, %v", ok, m)
	}
	if m.Type() != airp.String {
		t.Errorf("want String, got %s", m.Type())
	}
	if m.Key() != "web-app.servlet.1.init-param.mailHost" {
		t.Errorf(`key mismatch: want "web-app.servlet.1.init-param.mailHost", got %s`,
			m.Key())
	}

	m, _ = n.GetChild("web-app.servlet.4.init-param")
	m.AddChildren(airp.StandaloneNode("new", `"indeed!"`))
	err = m.RemoveChild("betaServer")
	if err != nil {
		t.Error(err)
	}
	err = m.SetChild(airp.StandaloneNode("log", "5"))
	if err != nil {
		t.Error(err)
	}
	err = m.SetChild(airp.StandaloneNode("dataLogMaxSize", "null"))
	if err != nil {
		t.Error(err)
	}
	want := `
{
  "templatePath": "toolstemplates/",
  "log": 5,
  "logLocation": "/usr/local/tomcat/logs/CofaxTools.log",
  "logMaxSize": "",
  "dataLog": 1,
  "dataLogLocation": "/usr/local/tomcat/logs/dataLog.log",
  "dataLogMaxSize": null,
  "removePageCache": "/content/admin/remove?cache=pages&id=",
  "removeTemplateCache": "/content/admin/remove?cache=templates&id=",
  "fileTransferFolder": "/usr/local/tomcat/webapps/content/fileTransferFolder",
  "lookInContext": 1,
  "adminGroupID": 4,
  "new": "indeed!"
}`
	if m.Len() != 13 {
		t.Errorf("want 13, got %d", m.Len())
	}
	b := &bytes.Buffer{}
	m.WriteIndent(b, "  ")
	if b.String() != strings.TrimSpace(want) {
		t.Errorf("string representation mismatch: \n%s",
			diff.LineDiff(b.String(), strings.TrimSpace(want)))
	}
}

func TestNewJSONGo(t *testing.T) {
	type myType int
	var intPtr = new(int)
	*intPtr = 50

	tests := []struct {
		have interface{}
		want string
	}{{
		nil, "null",
	}, {
		true, "true",
	}, {
		5, "5",
	}, {
		myType(550022), "550022",
	}, {
		5., "5",
	}, {
		"Hello, World!", `"Hello, World!"`,
	}, {
		[...]int{1, 2, 3, 4}, "[1,2,3,4]",
	}, {
		[]interface{}{nil, true, 3, "hi"}, `[null,true,3,"hi"]`,
	}, {
		map[string]interface{}{"bb": false}, `{"bb":false}`,
	}, {
		struct {
			Integer int
			a       string
		}{20, "aa"},
		`{"Integer":20}`,
	}, {
		struct {
			Integer uint `json:"int"`
			a       string
		}{20, "aa"},
		`{"int":20}`,
	}, {
		struct {
			Integer int `json:"-"`
			A       string
		}{20, "aa"},
		`{"A":"aa"}`,
	}, {
		struct {
			Integer int    `json:",omitempty"`
			A       string `json:"omitempty"`
		}{0, "aa"},
		`{"omitempty":"aa"}`,
	}, {
		struct {
			Integer int    `json:",omitempty"`
			A       string `json:"omitempty"`
		}{1, "aa"},
		`{"Integer":1,"omitempty":"aa"}`,
	}, {
		struct {
			Integer int    `json:",omitempty,string"`
			A       string `json:"a-b,"`
		}{1, "aa"},
		`{"Integer":"1","a-b":"aa"}`,
	}, {
		struct {
			Integer int64  `json:",string"`
			A       string `json:"string"`
		}{0, "aa"},
		`{"Integer":"0","string":"aa"}`,
	}, {
		&struct {
			Integer *int `json:"intptr"`
			a       string
		}{intPtr, "aa"},
		`{"intptr":50}`,
	}, {
		&[...]uint64{6}, "[6]",
	}, {
		[]byte("bytes"), `"bytes"`,
	}}
	for _, test := range tests {
		n, err := airp.NewJSONGo(test.have)
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
	}{{
		`[null,5,"hello there"]`, "2", true, `"hello there"`,
	}, {
		`{"a":null,"b":5,"json":"hello there"}`, "json", true, `"hello there"`,
	}, {
		`{"index":{"inner":[true]}}`, "index.inner.0", true, "true",
	}, {
		`{"index":[{"inner":[null,true]}]}`, "index.inner.0", false, "",
	}, {
		`{"index":[{"inner":[null,true]}]}`, "index.0.inner.1", true, "true",
	}, {
		`{"index":{"inner":[true]}}`, "index.iner.0", false, "",
	}}
	for _, test := range tests {
		n, err := airp.NewJSONString(test.json)
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
	}{{
		5, "5",
	}, {
		nil, "null",
	}, {
		"hello", `"hello"`,
	}, {
		[]bool{false, true}, "[false, true]",
	}, {
		map[string]interface{}{
			"a":    true,
			"long": 100000,
		},
		`{"long":100000,"a":true}`,
	}, {
		[]interface{}{"hello", false}, `["hello",false]`,
	}}
	for _, test := range tests {
		n, err := airp.NewJSONGo(test.goval)
		if err != nil {
			t.Fatal(err)
		}
		m, err := airp.NewJSONString(test.json)
		if err != nil {
			t.Fatal(err)
		}
		if !airp.EqNode(n, m) {
			t.Errorf("%s == %s", n, m)
		}
	}
}

func TestLen(t *testing.T) {
	tests := []struct {
		json string
		len  int
	}{{
		"true", 1,
	}, {
		"{}", 0,
	}, {
		`{"a":5,"b":null}`, 2,
	}, {
		"[1,2,3,4,5,6,7,8,9]", 9,
	}}
	for _, test := range tests {
		n, err := airp.NewJSONString(test.json)
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
	}{{
		"true", new(bool), true,
	}, {
		"52", new(int), 52,
	}, {
		"3452.1", new(float64), 3452.1,
	}, {
		"3452.1", new(float32), float32(3452.1),
	}, {
		`"Hello, World!"`, new(string), "Hello, World!",
	}, {
		`[true, "hi"]`, &[]interface{}{}, []interface{}{true, "hi"},
	}, {
		`[52, 420]`, &[]float64{}, []float64{52, 420},
	}, {
		`[52, 420]`, &[]int{}, []int{52, 420},
	}, {
		`{"a":52,"b":420}`,
		&map[string]int{},
		map[string]int{"a": 52, "b": 420},
	}, {
		`{"a":52,"b":true}`,
		&struct {
			A int  `json:"a"`
			B bool `json:"b"`
		}{},
		struct {
			A int  `json:"a"`
			B bool `json:"b"`
		}{52, true},
	}, {
		`{"Str":true,"bool":false,"This":5}`,
		&struct {
			Str  string `json:",string"`
			Bool bool   `json:"bool"`
			This int    `json:"-"`
		}{},
		struct {
			Str  string `json:",string"`
			Bool bool   `json:"bool"`
			This int    `json:"-"`
		}{Str: "true", Bool: false},
	}, {
		`{"a":true,"bool":false,"This":5}`,
		&struct {
			Str  string `json:"bool,string"`
			Bool bool   `json:"a,"`
			This int    `json:",omitempty"`
		}{},
		struct {
			Str  string `json:"bool,string"`
			Bool bool   `json:"a,"`
			This int    `json:",omitempty"`
		}{Str: "false", Bool: true, This: 5},
	}, {
		`{"a":true,"bool":false}`,
		&struct {
			Str  string `json:"bool,string"`
			Bool bool   `json:"a,"`
			This int    `json:",omitempty"`
		}{},
		struct {
			Str  string `json:"bool,string"`
			Bool bool   `json:"a,"`
			This int    `json:",omitempty"` // err
		}{Str: "false", Bool: true},
	}, {
		`{"str":true}`,
		&struct {
			Str string `json:",string"`
		}{},
		struct {
			Str string `json:",string"`
		}{Str: "true"},
	}, {
		`{"str":true}`,
		&struct {
			str string `json:",string"`
		}{},
		struct {
			str string `json:",string"`
		}{},
	}}
	for i, test := range tests {
		n, err := airp.NewJSONString(test.have)
		if err != nil {
			t.Fatalf("test setup fail: %v", err)
		}
		err = n.JSON2Go(test.store)
		if err != nil {
			t.Error(i, err)
			continue
		}
		got := reflect.ValueOf(test.store).Elem().Interface()
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("want %v got %v", test.want, got)
		}
	}
}

func TestValue(t *testing.T) {
	tests := []struct {
		have string
		want interface{}
	}{{
		`{"a": null}`,
		map[string]interface{}{"a": nil},
	}, {
		`[false, -31.2, 5, "ab\"cd"]`,
		[]interface{}{false, -31.2, 5., "ab\\\"cd"},
	}, {
		`{"a": 20, "b": [true, null]}`,
		map[string]interface{}{"a": 20., "b": []interface{}{true, nil}},
	}}
	for _, test := range tests {
		ast, _ := airp.NewJSONString(test.have)
		itf, err := ast.Value()
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(itf, test.want) {
			t.Errorf("want %v, got %v", test.want, itf)
		}
	}
}

func TestCopy(t *testing.T) {
	n, err := airp.NewJSONString(`{"a": ["hello", false], "b": "yes"}`)
	if err != nil {
		t.Fatal(err)
	}
	m := n.Copy()
	m.SetChild(airp.StandaloneNode("a.1", "true"))
	if airp.EqNode(n, m) {
		t.Errorf("%s != %s", n, m)
	}
}

func TestEscape(t *testing.T) {
	tests := []struct {
		name, have, want string
	}{{
		"c", `"ab\u0063"`, `"abc"`,
	}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			n, err := airp.NewJSONString(test.have)
			if err != nil {
				t.Fatalf("tests setup fail: %s", err)
			}
			if n.String() != test.want {
				t.Error(n.String() + " != " + test.want)
			}
		})
	}
}
