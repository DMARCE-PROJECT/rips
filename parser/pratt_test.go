package parser_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"rips/rips/extern"
	"rips/rips/lex"
	"rips/rips/parser"
	"rips/rips/tree"
	"rips/rips/types"
	"testing"
)

type exprExampFloat struct {
	input   string
	isBad   bool
	val     float64
	isDebug bool
}

var genExprsFloat = []exprExampFloat{
	{"1.0 *2.0+3.0", false, 5.0, false},
	{"1.0 +2.0*3.0", false, 7.0, false},
	{"3.0 *4.7+5.2", false, 19.3, false},
	{"3.0 *(4.7+5.2)", false, 29.7, false},
	{"-(2.0)", false, -2.0, false},
	{"--(2.0)", false, 2.0, false},
	{"3.0 *+5.2", false, 15.60, false},
	//bad expr
	{"", true, -1.0, false},
	{"3.0 **5.2", true, -1.0, false},
	{"3.0 *", true, -1.0, false},
	{"* 3.0", true, -1.0, false},
	{"3.0 *4.7 5.2", true, -1.0, false},
	//{"3.0 * 4.7+5.2)", true, -1.0, false},
	{"3.0 * (4.7+5.2", true, -1.0, false},
	{"3.0 * (4.7+5.2", true, -1.0, false},
	{"3.0 * 4.7+)5.2", true, -1.0, false},
	{"2.0 ^ *2.0 ^ (2.0 ^ 2.0", true, -1.0, false},
	{"unosiete * 2.0", false, 34.0, false},
	{"12.0 * funky(1.2)", false, 26.4, false},
	{"12.0 * funky(1.2, 0xa(3+5))", true, 26.4, false},
	{"12.0 * funky(1.2, 0xa 3+5))", true, 26.4, false},
	{"12.0 * funky(1.2, 0xa(3+5)", true, 26.4, false},
	{"*", true, -1.0, false},
	{"()", true, -1.0, false},
	{"(", true, -1.0, false},
	{"-", true, -1.0, false},
	{"1.0 *2.0?", false, 2.0, false},
	{"-3.0 * 2.0", false, -6.0, false},
	{"-3.0  2.0", true, -6.0, false},
	{"3.0 /-2.0/ 5.0", false, -0.3, false},
	{"3.0 /(-2.0/ 5.0)", false, -7.5, false},
}

const Eps = 1e-9

func almostEqual(f, g float64) bool {
	return math.Abs(f-g) <= Eps
}

func floatVar(t *testing.T, p *parser.Parser, name string, val float64) (s *tree.Sym) {
	svar, err := p.Envs.NewVar(name, types.TypeVals[types.TVFloat])
	if err != nil {
		t.Fatal(err)
	}
	svar.Val = tree.NewAnonSym(tree.SConst)
	svar.Val.DataType = types.FloatType
	svar.Val.FloatVal = val
	return svar
}

func declFunc(t *testing.T, p *parser.Parser, name string, fn func(context *extern.Ctx, args ...*tree.Sym) *tree.Sym) (s *tree.Sym) {
	argtypes := []types.Type{types.FloatType}
	f, err := p.Envs.NewFunc(name, argtypes, argtypes[0], fn, false, false)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestGenFloat(t *testing.T) {
	var expr *tree.Sym
	var olddebug bool
	for _, v := range genExprsFloat {
		if testing.Verbose() {
			fmt.Fprintf(os.Stderr, "--> %s\n", v.input)
		}
		if v.isDebug {
			olddebug = parser.DebugDesc
			parser.DebugDesc = true
			defer func() { parser.DebugDesc = olddebug }()
		}
		l, err := lex.NewFakeLexer(v.input, ioutil.Discard)
		if err != nil {
			t.Fatal(err)
		}
		//fmt.Printf("------- to do: %s\n", v.input)
		p := parser.NewParser(l)
		p.Envs.PushEnv()
		floatVar(t, p, "unosiete", 17.0)
		declFunc(t, p, "funky", func(context *extern.Ctx, args ...*tree.Sym) *tree.Sym {
			//fmt.Printf("executing funky(args %s)\n", args)
			//for i, a := range(args) {
			//	fmt.Printf("\t func arg [%d] is %s\n", i, a)
			//}
			val := tree.NewAnonSym(tree.SConst)
			val.CopyValFrom(args[0])
			val.FloatVal += 1.0
			return val
		})
		if expr, err = p.Expr(-1); err != nil && !v.isBad {
			errs := fmt.Sprintf("%s: %s", err, v.input)
			t.Fatal(errs)
		}
		if !v.isBad && expr == nil {
			errs := fmt.Sprintf("%s should not fail evals to nil: %s", v.input, err)
			t.Fatal(errors.New(errs))
		}
		if expr == nil || err != nil {
			continue
		}
		expr.Annotate()
		if v.isDebug {
			fmt.Fprintf(os.Stderr, "expr to eval: %s\n", expr)
		}
		//hack for testing p.Envs will be stack...
		valsym := expr.EvalExpr(&p.Envs, nil)
		if v.isBad && (err == nil && valsym.DataType.TVal == types.TypeVals[types.TVFloat]) {
			errs := fmt.Sprintf("%s should fail evals to %s", v.input, valsym)
			t.Fatal(errors.New(errs))
		}
		if !v.isBad && valsym.DataType.TVal != types.TypeVals[types.TVFloat] {
			errs := fmt.Sprintf("%s  is %s should be float", v.input, valsym)
			t.Fatal(errs)
		}
		val := valsym.FloatVal
		if !v.isBad && !almostEqual(val, v.val) {
			errs := fmt.Sprintf("%s  is %f should be %f", v.input, val, v.val)
			t.Fatal(errs)
		}
		if v.isDebug {
			parser.DebugDesc = olddebug
		}
	}
}

type exprExampInt struct {
	input   string
	isBad   bool
	val     int64
	isDebug bool
}

var genExprsInt = []exprExampInt{
	{"1 *2+3", false, 5, false},
	{"1 +2*3", false, 7, false},
	{"3 *4+5", false, 17, false},
	{"3 *(4+5)", false, 27, false},
	//(2^2)^2 = 8 and 2^(2^2) = 16, correct, right associative
	{"-(2)", false, -2, false},
	{"--(2)", false, 2, false},
	{"3 *+5", false, 15, false},
	//bad expr
	{"", true, -1, false},
	{"3 **5", true, -1, false},
	{"3 *", true, -1, false},
	{"* 3", true, -1, false},
	{"3 *4 5", true, -1, false},
	//{"3 * 4+5)", true, -1, false},
	{"3 * (4+5", true, -1, false},
	{"3 * (4+5", true, -1, false},
	{"3 * 4+)5", true, -1, false},
	{"2 ^ *2 ^ (2 ^ 2", true, -1, false},
	{"*", true, -1, false},
	{"()", true, -1, false},
	{"(", true, -1, false},
	{"-", true, -1, false},
	{"3 *4+5.0", true, 17, false},
	{"5 % 2+3", false, 4, false},
	{"5 | 4+3", false, 7, false},
	{"4 | 4+3", false, 7, false},
	{"4 | 2+5", false, 7, false},
	{"4 & 2+7", false, 0, false},
	{"9 / 2", false, 4, false},
	{"(-1) ^ (-1)", false, 0, false},
	{"~-1", false, 0, false},
	{"-~1", false, 2, false},
	{"2-3+4", false, (2 - 3) + 4, false},
}

func TestGenInt(t *testing.T) {
	var expr *tree.Sym
	var olddebug bool
	for _, v := range genExprsInt {
		if testing.Verbose() {
			fmt.Fprintf(os.Stderr, "--> %s\n", v.input)
		}
		if v.isDebug {
			olddebug = parser.DebugDesc
			parser.DebugDesc = true
			defer func() { parser.DebugDesc = olddebug }()
		}
		l, err := lex.NewFakeLexer(v.input, ioutil.Discard)
		if err != nil {
			t.Fatal(err)
		}
		//fmt.Printf("------- to do: %s\n", v.input)
		p := parser.NewParser(l)
		if expr, err = p.Expr(-1); err != nil && !v.isBad {
			errs := fmt.Sprintf("%s: %s", err, v.input)
			t.Fatal(errs)
		}
		if !v.isBad && expr == nil {
			errs := fmt.Sprintf("%s should not fail evals to nil: %s", v.input, err)
			t.Fatal(errors.New(errs))
		}
		if expr == nil || err != nil {
			continue
		}
		expr.Annotate()
		if v.isDebug {
			fmt.Fprintf(os.Stderr, "expr to eval: %s\n", expr)
		}
		valsym := expr.EvalExpr(&p.Envs, nil)
		if v.isBad && (err == nil && valsym.DataType.TVal == types.TypeVals[types.TVInt]) {
			errs := fmt.Sprintf("%s should fail evals to %s", v.input, valsym)
			t.Fatal(errors.New(errs))
		}
		if !v.isBad && valsym.DataType.TVal != types.TypeVals[types.TVInt] {
			errs := fmt.Sprintf("%s  is %s should be int", v.input, valsym)
			t.Fatal(errs)
		}
		val := valsym.IntVal
		if !v.isBad && val != v.val {
			errs := fmt.Sprintf("%s  is %d should be %d", v.input, val, v.val)
			t.Fatal(errs)
		}
		if v.isDebug {
			parser.DebugDesc = olddebug
		}
	}
}

func stringVar(t *testing.T, p *parser.Parser, name string, val string) (s *tree.Sym) {
	svar, err := p.Envs.NewVar(name, types.TypeVals[types.TVString])
	if err != nil {
		t.Fatal(err)
	}
	svar.Val = tree.NewAnonSym(tree.SConst)
	svar.Val.DataType = types.StringType
	svar.Val.StrVal = val
	return svar
}

type exprExampString struct {
	input   string
	isBad   bool
	val     bool
	isDebug bool
}

// strings (and some boolean)
var genExprsStrings = []exprExampString{
	{`"hola" > "aaa"`, false, true, false},
	{`"" < "aaa"`, false, true, false},
	{`"hola soy una prueba\n" == "hola soy una prueba\n"`, false, true, false},
	{`"hola soy una prueba\n" != "hola soy una prueba\n"`, false, false, false},
	{`"xxx" > "aaa"`, false, true, false},
	{`"xxx" >= "aaa"`, false, true, false},
	{`"xxx" < "aaa"`, false, false, false},
	{`"xxx" <= "aaa"`, false, false, false},
	{`!("zola" > "hola")`, false, false, false},
	{`!("zola" > "hola") && true`, false, false, false}, //mixed
	{`"aaa" < "xxx"`, false, true, false},
	{`"z"+"aaa" > "xxx"`, false, true, false},
	{`"aaa" <= "xxx"`, false, true, false},
	{`true > false`, false, true, false}, //also boolean
	{`"aaa" < varstr`, false, true, false},
	{`"xxx" < strfunky("xxx")`, false, true, false},
	{`!true && false`, false, false, false},   //also boolean
	{`!(true && false)`, false, true, false},  //also boolean
	{`!(true || false)`, false, false, false}, //also boolean
	{`"%" == "\x25"`, false, true, false},
	{`"%%" == "\x25%"`, false, true, false},
	{`"%%" == "\x25\x25"`, false, true, false},
	{`"$" < "\x25"`, false, true, false},
	{`"&" > "\x25"`, false, true, false},
	{`"\"" == "\x22"`, false, true, false},
	{`"\\" == "\x5c"`, false, true, false},
	{`"\xe2\x86\x92" == "→"`, false, true, false},
	{`"\u2192" == "→"`, false, true, false},
	{`"
" == "\n"`, false, true, false}, //multiline strings
}

func declFuncStr(t *testing.T, p *parser.Parser, name string, fn func(context *extern.Ctx, args ...*tree.Sym) *tree.Sym) (s *tree.Sym) {
	argtypes := []types.Type{types.StringType}
	f, err := p.Envs.NewFunc(name, argtypes, argtypes[0], fn, false, false)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestGenString(t *testing.T) {
	var expr *tree.Sym
	var olddebug bool
	for _, v := range genExprsStrings {
		if testing.Verbose() {
			fmt.Fprintf(os.Stderr, "--> %s\n", v.input)
		}
		if v.isDebug {
			olddebug = parser.DebugDesc
			parser.DebugDesc = true
			defer func() { parser.DebugDesc = olddebug }()
		}
		l, err := lex.NewFakeLexer(v.input, ioutil.Discard)
		if err != nil {
			t.Fatal(err)
		}
		//fmt.Printf("------- to do: %s\n", v.input)
		p := parser.NewParser(l)
		p.Envs.PushEnv()
		stringVar(t, p, "varstr", "content")
		declFuncStr(t, p, "strfunky", func(context *extern.Ctx, args ...*tree.Sym) *tree.Sym {
			//fmt.Printf("executing strfunky(args %s)\n", args)
			//for i, a := range(args) {
			//	fmt.Printf("\t func arg [%d] is %s\n", i, a)
			//}
			val := tree.NewAnonSym(tree.SConst)
			val.CopyValFrom(args[0])
			val.StrVal += args[0].StrVal + "result"
			return val
		})
		if expr, err = p.Expr(-1); err != nil && !v.isBad {
			errs := fmt.Sprintf("%s: %s", err, v.input)
			t.Fatal(errs)
		}
		if !v.isBad && expr == nil {
			errs := fmt.Sprintf("%s should not fail evals to nil: %s", v.input, err)
			t.Fatal(errors.New(errs))
		}
		if expr == nil || err != nil {
			continue
		}
		expr.Annotate()
		if v.isDebug {
			fmt.Fprintf(os.Stderr, "expr to eval: %s\n", expr)
		}
		//hack for testing p.Envs will be stack...
		valsym := expr.EvalExpr(&p.Envs, nil)
		if v.isBad && (err == nil && valsym.DataType.TVal == types.TypeVals[types.TVBool]) {
			errs := fmt.Sprintf("%s should fail evals to %s", v.input, valsym)
			t.Fatal(errors.New(errs))
		}
		if !v.isBad && valsym.DataType.TVal != types.TypeVals[types.TVBool] {
			errs := fmt.Sprintf("%s  is %s should be bool", v.input, valsym)
			t.Fatal(errs)
		}
		val := valsym.BoolVal
		if !v.isBad && val != v.val {
			errs := fmt.Sprintf("%s  is %v should be %v", v.input, val, v.val)
			t.Fatal(errs)
		}
		if v.isDebug {
			parser.DebugDesc = olddebug
		}
	}
}
