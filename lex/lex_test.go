package lex_test

import (
	_ "embed"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"rips/rips/lex"
	"runtime/debug"
	"strings"
	"testing"
)

type tokExamp struct {
	input string
	Types []lex.TokType //tokBad has to be the last
	//must be a prefix of the input tokenization
}

var genToks = []tokExamp{
	{"(", []lex.TokType{lex.TokLPar}},
	{")", []lex.TokType{lex.TokRPar}},
	{"+", []lex.TokType{lex.TokAdd}},
	{"*", []lex.TokType{lex.TokMul}},
	{"-", []lex.TokType{lex.TokMin}},
	{"/", []lex.TokType{lex.TokDiv}},
	//	{"\n", []lex.TokType{lex.TokEol}},
	{";", []lex.TokType{lex.TokSemi}},
	{"", []lex.TokType{lex.TokEof}},
	{"3.8e-12", []lex.TokType{lex.TokFloatVal}},
	{"69", []lex.TokType{lex.TokIntVal}},
	{"0x16", []lex.TokType{lex.TokIntVal}},
	{"0xff", []lex.TokType{lex.TokIntVal}},
	{`"hola\nperola"`, []lex.TokType{lex.TokStrVal}},
	{`"hola\tperola"`, []lex.TokType{lex.TokStrVal}},
	{`"hola\t"`, []lex.TokType{lex.TokStrVal}},
	{`""`, []lex.TokType{lex.TokStrVal}},
	{`"`, []lex.TokType{lex.TokBad}},
	{`"hola`, []lex.TokType{lex.TokBad}},
	{`"hola\"`, []lex.TokType{lex.TokBad}},
	{`"hola\"otro"`, []lex.TokType{lex.TokStrVal}},
	{"true", []lex.TokType{lex.TokBoolVal}},
	{"==", []lex.TokType{lex.TokEq}},
	{"=", []lex.TokType{lex.TokAsig}},
	{"=,", []lex.TokType{lex.TokAsig}},
	{">=", []lex.TokType{lex.TokGEq}},
	{">", []lex.TokType{lex.TokG}},
	{"=>", []lex.TokType{lex.TokThen}},
	{"!>", []lex.TokType{lex.TokNThen}},
	{"↛", []lex.TokType{lex.TokNThen}},
	{"→", []lex.TokType{lex.TokThen}},
	{"#hola hola\n>", []lex.TokType{lex.TokG}},
	{"~", []lex.TokType{lex.TokCompl}},
	{"\xff\n>", []lex.TokType{lex.TokBad}},      //bad rune token
	{`"\xff\n>"`, []lex.TokType{lex.TokStrVal}}, //bad rune inside string (valid)
}

func recovCrashFail(t *testing.T, l *lex.Lexer) {
	if r := recover(); r != nil {
		errs := fmt.Sprintf("%s %s", l, r)
		if strings.HasPrefix(errs, "runtime error:") {
			errs = strings.Replace(errs, "runtime error:", "compiler error:", 1)
		}
		err := errors.New(errs)
		if lex.DFlex {
			fmt.Fprintf(os.Stderr, "%s\n%s", err, debug.Stack())
		}
		t.Fatal("something bad happened")
	}
}

func exampleTest(t *testing.T, examples []tokExamp) {
	var l *lex.Lexer
	defer recovCrashFail(t, l)
	for _, ex := range examples {
		l, err := lex.NewFakeLexer(ex.input, ioutil.Discard)
		if err != nil {
			t.Fatal(err)
		}
		for i, tokT := range ex.Types {
			tok, err := l.Lex()
			if err != nil && tokT != lex.TokBad {
				errs := fmt.Sprintf("%s %s -> %s [%d] %s", l, err, ex.input, i, tokT)
				t.Fatal(errs)
			}
			if tok.Type != tokT {
				errs := fmt.Sprintf("input: %s [%d] %s != %s", ex.input, i, tok.Type, tokT)
				t.Fatal(errs)
			}
			if tok.Type == lex.TokEof {
				break
			}

		}

	}
}

func TestGenToks(t *testing.T) {
	exampleTest(t, genToks)
}

type tokExampFloat struct {
	input string
	Types []lex.TokType //tokBad has to be the last
	f     float64
}

var floatToks = []tokExampFloat{
	{"1.0", []lex.TokType{lex.TokFloatVal}, 1.0},
	{"1.", []lex.TokType{lex.TokFloatVal}, 1.0},
	{"0.1", []lex.TokType{lex.TokFloatVal}, 0.1},
	{".0", []lex.TokType{lex.TokFloatVal}, .0},
	{".12", []lex.TokType{lex.TokFloatVal}, .12},
	{"1.0e", []lex.TokType{lex.TokBad}, -1},
	{"1.0e2", []lex.TokType{lex.TokFloatVal}, 100.0},
	{"1.0e-1", []lex.TokType{lex.TokFloatVal}, 0.1},
	{"1.02+3", []lex.TokType{lex.TokFloatVal}, 1.02},
	{".0e-2", []lex.TokType{lex.TokFloatVal}, .0},
	{".0e+3", []lex.TokType{lex.TokFloatVal}, .0},
	{"1E3", []lex.TokType{lex.TokFloatVal}, 1e3},
	{"1e3", []lex.TokType{lex.TokFloatVal}, 1e3},
	{"1e+2", []lex.TokType{lex.TokFloatVal}, 1e2},
	{"13e-4", []lex.TokType{lex.TokFloatVal}, 13e-4},
	{"1..", []lex.TokType{lex.TokFloatVal}, 1},
	{".1.1.0", []lex.TokType{lex.TokBad}, -1},
	{"..", []lex.TokType{lex.TokBad}, -1},
	{"1e+a2", []lex.TokType{lex.TokBad}, -1},
	{"1ea+2", []lex.TokType{lex.TokBad}, -1},
	{"1e2+", []lex.TokType{lex.TokFloatVal, lex.TokAdd}, 1e2},
}

func TestFloatToks(t *testing.T) {
	exampleTestFloat(t, floatToks)
}

const Eps = 1e-9

func almostEqual(f, g float64) bool {
	return math.Abs(f-g) <= Eps
}

func exampleTestFloat(t *testing.T, examples []tokExampFloat) {
	var l *lex.Lexer
	defer recovCrashFail(t, l)
	for _, ex := range examples {
		l, err := lex.NewFakeLexer(ex.input, ioutil.Discard)
		if err != nil {
			t.Fatal(err)
		}
		for i, tokT := range ex.Types {
			tok, err := l.Lex()
			if err != nil && tokT != lex.TokBad {
				errs := fmt.Sprintf("%s %s -> %s [%d] %s", l, err, ex.input, i, tokT)
				t.Fatal(errs)
			}
			if err == nil && tokT != lex.TokBad && tokT == lex.TokFloatVal {
				if !almostEqual(tok.TokFloatVal, ex.f) {
					t.Fatalf("%s should be %f and is %f", ex.input, ex.f, tok.TokFloatVal)
				}
			}
			if tok.Type != tokT {
				errs := fmt.Sprintf("input: %s [%d] %s != %s", ex.input, i, tok.Type, tokT)
				t.Fatal(errs)
			}
			if tok.Type == lex.TokEof {
				break
			}

		}

	}
}

type tokExampInt struct {
	input string
	Types []lex.TokType //tokBad has to be the last
	i     int64
}

var intToks = []tokExampInt{
	{"12", []lex.TokType{lex.TokIntVal}, 12},
	{"0xff", []lex.TokType{lex.TokIntVal}, 0xff},
	{"0xff a", []lex.TokType{lex.TokIntVal}, 0xff},
	{"01a", []lex.TokType{lex.TokIntVal}, 1},
	{"0ba", []lex.TokType{lex.TokBad}, 1},
	{"0oa", []lex.TokType{lex.TokBad}, 1},
	{"000", []lex.TokType{lex.TokIntVal}, 0},
	{"017", []lex.TokType{lex.TokIntVal}, 17},
	{"0b11", []lex.TokType{lex.TokIntVal}, 3},
	{"0b", []lex.TokType{lex.TokBad}, 3},
}

func TestIntToks(t *testing.T) {
	exampleTestInt(t, intToks)
}

func exampleTestInt(t *testing.T, examples []tokExampInt) {
	var l *lex.Lexer
	defer recovCrashFail(t, l)
	for _, ex := range examples {
		l, err := lex.NewFakeLexer(ex.input, ioutil.Discard)
		if err != nil {
			t.Fatal(err)
		}
		for i, tokT := range ex.Types {
			tok, err := l.Lex()
			if err != nil && tokT != lex.TokBad {
				errs := fmt.Sprintf("%s %s -> %s [%d] %s", l, err, ex.input, i, tokT)
				t.Fatal(errs)
			}
			if err == nil && tokT != lex.TokBad && tokT == lex.TokIntVal {
				if tok.TokIntVal != ex.i {
					t.Fatalf("%s should be %d and is %d", ex.input, ex.i, tok.TokIntVal)
				}
			}
			if tok.Type != tokT {
				errs := fmt.Sprintf("input: %s [%d] %s != %s", ex.input, i, tok.Type, tokT)
				t.Fatal(errs)
			}
			if tok.Type == lex.TokEof {
				break
			}

		}

	}
}

func TestLexWhileNot(t *testing.T) {
	var l *lex.Lexer
	defer recovCrashFail(t, l)
	l, err := lex.NewFakeLexer(" 3, 4, hola", ioutil.Discard)
	if err != nil {
		t.Fatal(err)
	}
	tok, err := l.LexWhileNot(lex.TokId, lex.TokEof)
	if err != nil && tok.Type != lex.TokBad {
		errs := fmt.Sprintf("%s %s -> %s", l, err, tok.Type)
		t.Fatal(errs)
	}
	if tok.Type != lex.TokId {
		errs := fmt.Sprintf("input:  %s != %s", tok.Type, lex.TokId)
		t.Fatal(errs)
	}
	if tok.Type == lex.TokEof {
		t.Fatal("should not fine Eof")
	}

}

//go:embed examples/example.rul
var fuzzfile string

//go:embed examples/anotherexample.rul
var fuzzfile2 string

// testing of various false conditions
func FuzzRipsLex(f *testing.F) {
	nn := []int{1, 300, 25, 9999, 1, 660, 25, 30, 1, 3, 2555, 675, 12, 45, 25}
	lx, err := lex.NewFakeLexer(fuzzfile, ioutil.Discard)
	if err != nil {
		f.Fatal(err)
	}
	for i := 0; ; i++ {
		tok, err := lx.Lex()
		if err != nil {
			f.Fatalf("lexer should not fail in this case: %s\n", err)
		}
		if tok.Type == lex.TokEof {
			break
		}
		f.Add(tok.Lexema, nn[i%len(nn)]+i)
	}
	f.Fuzz(func(t *testing.T, in string, ncut int) {
		if ncut < 0 {
			ncut = -ncut
		}
		ncut = ncut % len(fuzzfile2)
		sr := []rune(fuzzfile2)
		cc := string(sr[0:ncut]) + in + string(sr[ncut:])
		out := ioutil.Discard
		if testing.Verbose() {
			out = os.Stderr
		}
		l, err := lex.NewFakeLexer(cc, out)
		if err != nil {
			t.Fatal(err)
		}
		defer recovCrashFail(t, l)
		for i := 0; ; i++ {
			tok, err := l.Lex()
			if err != nil {
				break
			}
			if tok.Type == lex.TokEof {
				break
			}
		}
	})
}

//TODO PEEK test
//TODO, lexId, Couples
