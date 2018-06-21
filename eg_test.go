package airp_test

import (
	"fmt"

	"." // "github.com/D1CED/jsonparser/airp"
)

func ExampleNode_MarshalJSON() {
	n := airp.StandaloneNode("", airp.Object, "")
	m := airp.StandaloneNode("Num", airp.Number, "3.125e-4")
	o := airp.StandaloneNode("Str", airp.String, "Hello, World!")
	n.Children = append(n.Children, *m, *o)
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
