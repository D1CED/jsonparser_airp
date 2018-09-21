package jsonparser_airp

import (
	"io/ioutil"
	"strings"
	"testing"
)

func BenchmarkLexer(b *testing.B) {
	input := `{{{{{[[[[]null,]]false]}:::::::::::::}},,,,,,,,,,}}true
	-54235.54324e22452566666"fasdhlsahglsahglahgahslöggfhal        "
	{{]]                                "fasfaf"::true:,,""{}[125421525426]
	0.53123[]{{{}null,,,,,,,,"hibas"::5::false[[{{}}       `
	for i := 0; i < b.N; i++ {
		lexc, _ := lex(strings.NewReader(input))
		for range lexc {
		}
	}
}

func BenchmarkParser(b *testing.B) {
	input := `{"a":{"ab":[]},"b":[0,true,{}],"c":null,"d":0,"e":"",
	"n":{"bool":true,"obj":{"v":null},"values":[{"a":5,"b":"hi","c":5.8,
	"d":null,"e":true},{"a":[5,6,7,8],"b":"hi2","c":5.9,"d":{
	"f":"Hello there!"},"e":false}]}}`
	lexc, _ := lex(strings.NewReader(input))
	var lexs []token
	for tk := range lexc {
		if tk.Type == errToken {
			b.Fatal("non-valid token stream")
		}
		lexs = append(lexs, tk)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		inpc := make(chan token, len(lexs))
		for _, tk := range lexs {
			inpc <- tk
		}
		close(inpc)
		_, err := parse(inpc, func() {})
		if err != nil {
			b.Fatalf("non-valid token stream: %v", err)
		}
	}
}

func BenchmarkFormat(b *testing.B) {
	input := `{"a":{"ab":[]},"b":[0,true,{}],"c":null,"d":0,"e":"",
	"n":{"bool":true,"obj":{"v":null},"values":[{"a":5,"b":"hi","c":5.8,
	"d":null,"e":true},{"a":[5,6,7,8],"b":"hi2","c":5.9,"d":{
	"f":"Hello there!"},"e":false}]}}`
	n, err := parse(lex(strings.NewReader(input)))
	if err != nil {
		b.Fatalf("benchmark setup failed: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = n.format(ioutil.Discard, "~~", "^", "__", "==")
		if err != nil {
			b.Fatal(err)
		}
	}
}
