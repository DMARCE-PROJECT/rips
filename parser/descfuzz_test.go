package parser_test

import (
	_ "embed"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"rips/rips/lex"
	"rips/rips/parser"
	"runtime/debug"
	"strings"
	"testing"
)

//go:embed examples/example_fuzz.rul
var examplefuzz string

func TestSimpleProgDrop(t *testing.T) {
	var dli *DropLexer
	i := 0
	defer func() {
		if r := recover(); r != nil {
			p := dli.Pos()
			fmt.Fprintf(os.Stderr, "%s", debug.Stack())
			t.Fatalf("%s:%d when dropping the %d token %s and processing token %d", p.File, p.Line, i, dli.tokDropped, dli.n)
		}
	}()
	for i = -1; ; i++ {
		rd := strings.NewReader(examplefuzz)
		fname := "examples/example_fuzz.rul"
		l, err := lex.NewLexerRd(rd, fname, ioutil.Discard)
		if err != nil {
			t.Fatal(err)
		}
		dli = &DropLexer{l: l, idxDrop: i, n: 0, didDrop: false}
		p := parser.NewParser(dli)
		p.Envs.PushEnv()
		p.Builtins(fakebuiltins)
		prog, err := p.Parse()
		if err != nil {
			t.Fatal(err)
		}
		_ = prog
		if !dli.didDrop {
			break
		}
	}
}

func TestSimpleProgDropPair(t *testing.T) {
	var (
		dli *DropLexer
		dlj *DropLexer
	)
	i := 0
	j := 0
	defer func() {
		if r := recover(); r != nil {
			p := dli.Pos()
			fmt.Fprintf(os.Stderr, "%s", debug.Stack())
			t.Fatalf("%s:%d when dropping the %d, %d token %s, %s and processing token %d", p.File, p.Line, i, j, dlj.tokDropped, dlj.tokDropped, dlj.n)
		}
	}()
	for i = -1; ; i++ {
		rd := strings.NewReader(examplefuzz)
		fname := "examples/example_fuzz.rul"
		l, err := lex.NewLexerRd(rd, fname, ioutil.Discard)
		if err != nil {
			t.Fatal(err)
		}
		dli = &DropLexer{l: l, idxDrop: i, n: 0, didDrop: false}
		for j = -1; ; j++ {
			dlj = &DropLexer{l: dli, idxDrop: j, n: 0, didDrop: false}
			p := parser.NewParser(dli)
			p.Envs.PushEnv()
			p.Builtins(fakebuiltins)
			prog, err := p.Parse()
			if err != nil {
				t.Fatal(err)
			}
			_ = prog
			if !dlj.didDrop {
				break
			}
		}
		if !dli.didDrop {
			break
		}
	}
}

type DropLexer struct {
	l          lex.LexPeeker
	n          int
	idxDrop    int
	didDrop    bool
	tokDropped lex.Token
}

func (f *DropLexer) Peek() (t lex.Token, err error) {
	if f.n == f.idxDrop {
		f.didDrop = true
		f.tokDropped, _ = f.l.Lex()
		f.n++
	}
	return f.l.Peek()
}
func (f *DropLexer) Lex() (t lex.Token, err error) {
	if f.n == f.idxDrop {
		f.didDrop = true
		f.tokDropped, _ = f.l.Lex()
		f.n++
	}
	t, err = f.l.Lex()
	f.n++
	return t, err
}

func (f *DropLexer) LexWhileNot(tps ...lex.TokType) (t lex.Token, err error) {
	return f.l.LexWhileNot()
}

func (f *DropLexer) Pos() lex.Position {
	return f.l.Pos()
}

func (f *DropLexer) Errout() io.Writer {
	return f.l.Errout()
}

type InjectLexer struct {
	l           lex.LexPeeker
	n           int
	idxInjected int
	didInject   bool
	tokInjected lex.Token
}

func (f *InjectLexer) Peek() (t lex.Token, err error) {
	if f.n == f.idxInjected {
		f.didInject = true //in case inject breaks something
		return f.tokInjected, nil
	}
	return f.l.Peek()
}

func (f *InjectLexer) Lex() (t lex.Token, err error) {
	if f.n == f.idxInjected {
		f.didInject = true
		f.n++
		return f.tokInjected, nil
	}
	t, err = f.l.Lex()
	f.n++
	return t, err
}

func (f *InjectLexer) LexWhileNot(tps ...lex.TokType) (t lex.Token, err error) {
	return f.l.LexWhileNot()
}

func (f *InjectLexer) Pos() lex.Position {
	return f.l.Pos()
}

func (f *InjectLexer) Errout() io.Writer {
	return f.l.Errout()
}

func LexTokenN(fname string, fcontent string, n int) (t lex.Token, err error) {
	var l lex.LexPeeker
	rd := strings.NewReader(examplefuzz)
	fname = "examples/example_fuzz.rul"
	l, err = lex.NewLexerRd(rd, fname, ioutil.Discard)
	if err != nil {
		return t, err
	}
	for i := 0; i < n; i++ {
		t, err = l.Lex()
		if err != nil || t.Type == lex.TokEof {
			return t, err
		}
	}
	return t, err
}

func LexTokenCount(fname string, fcontent string) (n int, err error) {
	var l lex.LexPeeker
	rd := strings.NewReader(examplefuzz)
	fname = "examples/example_fuzz.rul"
	l, err = lex.NewLexerRd(rd, fname, ioutil.Discard)
	if err != nil {
		return n, err
	}
	for {
		t, err := l.Lex()
		if err != nil || t.Type == lex.TokEof {
			return n, err
		}
		n++
	}
}

func TestSimpleProgInject(t *testing.T) {
	var li *InjectLexer
	i := 0
	defer func() {
		if r := recover(); r != nil {
			p := li.Pos()
			fmt.Fprintf(os.Stderr, "%s", debug.Stack())
			ef := "%s:%d when injecting the %d token %s and processing token %d"
			t.Fatalf(ef, p.File, p.Line, i, li.tokInjected, li.n)
		}
	}()
	n, err := LexTokenCount("examples/example_fuzz.rul", examplefuzz)
	if err != nil {
		t.Fatal(err)
	}
	for i = 1; i < n+1; i++ {
		tok, err := LexTokenN("examples/example_fuzz.rul", examplefuzz, i)
		if err != nil {
			t.Fatal(err)
		}
		if tok.Type == lex.TokEof {
			break
		}
		for j := 0; ; j++ {
			rd := strings.NewReader(examplefuzz)
			fname := "examples/example_fuzz.rul"
			l, err := lex.NewLexerRd(rd, fname, ioutil.Discard)
			if err != nil {
				t.Fatal(err)
			}
			li = &InjectLexer{l: l, n: 0, idxInjected: j, didInject: false, tokInjected: tok}
			p := parser.NewParser(li)
			p.Envs.PushEnv()
			p.Builtins(fakebuiltins)
			prog, err := p.Parse()
			if err != nil {
				t.Fatal(err)
			}
			_ = prog
			if !li.didInject {
				break
			}
		}
	}
}

type NopLexer struct {
	l       lex.LexPeeker
	nSwitch int
	n       int
	lb      lex.LexPeeker
}

func (f *NopLexer) Peek() (t lex.Token, err error) {
	return f.l.Peek()
}

func (f *NopLexer) Lex() (t lex.Token, err error) {
	if f.n == f.nSwitch {
		f.l = f.lb
	}
	f.n++
	t, err = f.l.Lex()
	return t, err
}

func (f *NopLexer) LexWhileNot(tps ...lex.TokType) (t lex.Token, err error) {
	return f.l.LexWhileNot()
}

func (f *NopLexer) Pos() lex.Position {
	return f.l.Pos()
}

func (f *NopLexer) Errout() io.Writer {
	return f.l.Errout()
}

//go:embed descfuzz_test.go
var descfuzz string

func TestSimpleProgWhatever(t *testing.T) {
	var li *InjectLexer
	i := 0
	defer func() {
		if r := recover(); r != nil {
			p := li.Pos()
			fmt.Fprintf(os.Stderr, "%s", debug.Stack())
			mf := "%s:%d when injecting the %d token %s and processing token %d"
			t.Fatalf(mf, p.File, p.Line, i, li.tokInjected, li.n)
		}
	}()
	n, err := LexTokenCount("examples/example_fuzz.rul", examplefuzz)
	if err != nil {
		t.Fatal(err)
	}
	for i = 0; i < n+1; i++ {
		rd := strings.NewReader(examplefuzz)
		fname := "examples/example_fuzz.rul"
		la, err := lex.NewLexerRd(rd, fname, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		rd = strings.NewReader(examplefuzz)
		lb, err := lex.NewLexerRd(rd, fname, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		p := parser.NewParser(&NopLexer{l: la, lb: lb, nSwitch: i})
		p.Envs.PushEnv()
		p.Builtins(fakebuiltins)
		prog, err := p.Parse()
		if err != nil {
			t.Fatal(err)
		}
		_ = prog
	}
}
