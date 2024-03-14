package parser_test

import (
	_ "embed"
	"fmt"
	"io/ioutil"
	"os"
	"rips/rips/extern"
	"rips/rips/lex"
	"rips/rips/parser"
	"rips/rips/tree"
	"rips/rips/types"
	"strings"
	"testing"
)

//go:embed examples/example.rul
var example string

func FakeBuiltin(context *extern.Ctx, args ...*tree.Sym) *tree.Sym {
	_ = context //does not use
	fmt.Fprintf(os.Stderr, "FakeBuiltin call, %s", args)
	val := tree.NewAnonSym(tree.SConst)
	val.DataType = types.BoolType
	val.BoolVal = true
	return val
}

var fakebuiltins = []*tree.Builtin{
	{
		Name:     "heremsg",
		RetType:  types.MsgType,
		Fn:       FakeBuiltin,
		ArgTypes: []types.Type{types.UnivType},
		IsAction: true,
	},
	{
		Name:     "heregraph",
		RetType:  types.BoolType,
		Fn:       FakeBuiltin,
		ArgTypes: []types.Type{types.UnivType},
		IsAction: true,
	},
	{
		Name:     "gohere",
		RetType:  types.BoolType,
		Fn:       FakeBuiltin,
		ArgTypes: []types.Type{types.UnivType},
		IsAction: true,
	},
}

func TestSimpleProg(t *testing.T) {
	rd := strings.NewReader(example)
	l, err := lex.NewLexerRd(rd, "examples/example.rul", os.Stderr) //ioutil.Discard)
	if err != nil {
		t.Fatal(err)
	}
	p := parser.NewParser(l)
	p.Envs.PushEnv()
	p.Builtins(fakebuiltins)
	prog, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	_ = prog
	if n := p.NErrors(); n > 0 {
		s := fmt.Sprintf("There were %d errors", n)
		t.Fatal(s)
	}
}

func TestTypesSimpleProg(t *testing.T) {
	rd := strings.NewReader(example)
	l, err := lex.NewLexerRd(rd, "examples/example.rul", ioutil.Discard)
	if err != nil {
		t.Fatal(err)
	}
	p := parser.NewParser(l)
	p.Envs.PushEnv()
	p.Builtins(fakebuiltins)
	prog, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	nerr := prog.TypeCheck(os.Stderr) //l.Errout())
	if n := nerr + p.NErrors(); n > 0 {
		s := fmt.Sprintf("There were %d errors", n)
		t.Fatal(s)
	}
}

//go:embed examples/example_typeerr.rul
var exampleerr string

func TestTypesErrs(t *testing.T) {
	rd := strings.NewReader(exampleerr)
	l, err := lex.NewLexerRd(rd, "examples/example_typeerr.rul", ioutil.Discard)
	if err != nil {
		t.Fatal(err)
	}
	p := parser.NewParser(l)
	p.Envs.PushEnv()
	p.Builtins(fakebuiltins)
	prog, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	nerr := prog.TypeCheck(l.Errout())
	if n := nerr + p.NErrors(); n < 2 {
		s := fmt.Sprintf("There were not enough errors")
		t.Fatal(s)
	}
}
