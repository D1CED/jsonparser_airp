package jsonparser_airp_test

import (
	"fmt"

	airp "github.com/d1ced/jsonparser-airp"
)

func ExampleNode_MarshalJSON() {
	n := airp.StandaloneNode("{}")
	m := airp.StandaloneNode("3.125e-4")
	o := airp.StandaloneNode(`"Hello, World!"`)
	n.AddChildren(*m, *o)
	data, _ := n.MarshalJSON()
	fmt.Printf("%s", data)
	// Output: {"Num": 3.125e-4, "Str": "Hello, World!"}
}

func ExampleNode_UnmarshalJSON() {
	data := []byte(`{"a": 20, "b": [true, null]}`)
	root := airp.Node{}
	err := root.UnmarshalJSON(data)
	if err != nil {
		return
	}
	// root now holds the top of the JSON ast.
	fmt.Println(root.String())
	// Output: {"a":20,"b":[true,null]}
}

func ExampleNode_Value() {
	data := []byte(`[{"a": null}, true]`)
	root := airp.Node{}
	_ = root.UnmarshalJSON(data)
	v, _ := root.Value()
	fmt.Println(v)
	// Output: [map[a:<nil>] true]
}
