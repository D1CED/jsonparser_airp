// Jannis Hoffmann, 20. 6. 2018

/*
Package airp encodes and decodes JSON.
In contrast to encoding/json airp is centered around an ast (Abstract Syntax
Tree) model. An ast can be manipulated and new nodes can be created.
Every non error-node is valid JSON.

airp is partly comartible with encoding/json.
Node fulfills the json.Marshaler/Unmarshaler interface.

    TODO
    -   Improve error handling.
    -   Children []Node => []*Node
*/
package airp // import "github.com/D1CED/jsonparser/airp"
