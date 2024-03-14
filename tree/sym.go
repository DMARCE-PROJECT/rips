package tree

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"rips/rips/extern"
	"rips/rips/lex"
	"rips/rips/types"

	"github.com/kgwinnup/go-yara/yara"
)

const DebSym = false

const (
	SNone   = iota
	SSect   // Section
	SLevel  // Level
	SFCall  // procedure or function call
	SVar    // object definition
	SFunc   // procedure or function definition
	SConst  // constant or literal
	SUnary  // unary expression
	SBinary // binary expression
	SAsign  //assignment
	SRegexp //for compiled regexps
	SYara   //for yara rules
)

type BuiltinFunc func(context *extern.Ctx, args ...*Sym) *Sym

type Func struct {
	ArgDataTypes []types.Type
	Fn           BuiltinFunc //the function itself
	IsVariadic   bool        //last argtype is repeated
}

type Sym struct {
	Name  string //name of the var, literal, etc.
	SType int    //SVar, SConst...
	Pos   lex.Position

	DataType types.Type //in func, return type
	//other stuff for the tree

	/* one of */

	FloatVal float64
	IntVal   int64
	StrVal   string
	BoolVal  bool
	Expr     *Expr
	Asign    *Asign

	/* Slevel two */
	SLevel int
	IsSoft bool

	Func

	IsAction bool //HACK for function and fcall, turned off after checked by the parser

	Val       *Sym /* for var when init (an later for eval)*/
	IsSet     bool
	IsUsed    bool
	IsBuiltin bool

	IsReach bool /*for levels */

	/* for builtin parameters, they are string vals with extras */
	Yr *yara.Yara
	Re *regexp.Regexp
}

func (s *Sym) Errorf(errout io.Writer, nerr int, str string, v ...interface{}) {
	if nerr >= 1 {
		return
	}
	place := fmt.Sprintf("%s:%d: ", s.Pos.File, s.Pos.Line)
	out := errout
	if lex.DOut {
		out = os.Stderr
	}
	fmt.Fprintf(out, place+str+"\n", v...)
}

type Env map[string]*Sym

type StkEnv []Env

func (envs *StkEnv) PushEnv() {
	env := Env{}
	*envs = append(*envs, env)
}
func (envs *StkEnv) PopEnv() {
	eS := *envs
	if len(eS) == 1 {
		panic("cannot pop bultin")
	}
	*envs = eS[:len(eS)-1]
}

func (envs *StkEnv) CurrEnv() (e Env) {
	return (*envs)[len(*envs)-1]
}

func (envs *StkEnv) CurrLevel() int {
	return len(*envs) - 1
}

func NewAnonSym(sType int) (s *Sym) {
	s = &Sym{Name: "lit", SType: sType}
	s.DataType = types.UndefExprType
	s.Pos.File = "Builtin"
	s.Pos.Line = 0
	return s
}

func (envs *StkEnv) NewSym(name string, sType int) (s *Sym, err error) {
	eS := *envs
	s = &Sym{Name: name, SType: sType}
	if DebSym {
		fmt.Fprintf(os.Stderr, "NewSym: %s\n", s)
	}
	e := eS[len(eS)-1]
	if _, ok := e[name]; ok {
		return nil, fmt.Errorf("already declared sym '%s'", name)
	}
	e[name] = s
	s.Pos.File = "Builtin"
	s.Pos.Line = 0
	return s, nil
}

func (envs *StkEnv) GetSym(name string) (sym *Sym) {
	eS := *envs
	for i := len(eS) - 1; i >= 0; i-- {
		if s, ok := eS[i][name]; ok {
			sym = s
			break
		}
	}
	return sym
}

func (envs *StkEnv) NewVar(name string, typeval *types.TypeVal) (sym *Sym, err error) {
	//to forbid shadowing, look up
	s := envs.GetSym(name)
	if s != nil {
		return nil, fmt.Errorf("already declared sym: no shadowing '%s'", name)
	}
	s, err = envs.NewSym(name, SVar)
	if err != nil {
		return nil, err
	}
	s.DataType = types.UndefExprType
	s.DataType.TVal = typeval

	return s, nil
}

func (envs *StkEnv) NewConst(name string, typeval *types.TypeVal) (sym *Sym, err error) {
	//to forbid shadowing, look up
	s := envs.GetSym(name)
	if s != nil {
		return nil, fmt.Errorf("already declared sym: no shadowing '%s'", name)
	}
	s, err = envs.NewSym(name, SConst)
	if err != nil {
		return nil, err
	}
	s.Name = name
	s.DataType = types.UndefExprType
	s.DataType.TVal = typeval

	return s, nil
}

func NewBool(v bool) (s *Sym) {
	val := NewAnonSym(SConst)
	val.DataType = types.BoolType
	val.BoolVal = v
	return val
}

func NewString(v string) (s *Sym) {
	val := NewAnonSym(SConst)
	val.DataType = types.StringType
	val.StrVal = v
	return val
}

func (s *Sym) IsOrShort(left *Sym) bool {
	isor := lex.TokType(s.Expr.Op) == lex.TokLogOr
	return isor && left.BoolVal
}

func (s *Sym) IsAndShort(left *Sym) bool {
	isand := lex.TokType(s.Expr.Op) == lex.TokLogAnd
	return isand && !left.BoolVal
}

func (envs *StkEnv) NewFunc(name string, typeargs []types.Type, typeret types.Type,
	fn BuiltinFunc,
	isvariadic bool, isaction bool) (sym *Sym, err error) {

	//to forbid shadowing, look up
	s := envs.GetSym(name)
	if s != nil {
		return nil, fmt.Errorf("already declared sym: no shadowing %s", name)
	}
	s, err = envs.NewSym(name, SFunc)
	if err != nil {
		return nil, err
	}
	s.DataType = typeret
	s.ArgDataTypes = typeargs
	s.IsVariadic = isvariadic
	s.IsAction = isaction
	s.Fn = fn
	return s, nil
}

func (s *Sym) String() string {
	if s == nil {
		return "nil"
	}
	str := "[!"
	if s.Name != "" {
		str += s.Name + " "
	}
	str += s.DataType.String() + " "
	switch s.SType {
	case SConst:
		str += "CONST: "
		switch s.DataType.TVal {
		case types.TypeVals[types.TVInt]:
			str += fmt.Sprintf("%d", s.IntVal)
		case types.TypeVals[types.TVFloat]:
			str += fmt.Sprintf("%f", s.FloatVal)
		case types.TypeVals[types.TVBool]:
			str += fmt.Sprintf("%v", s.BoolVal)
		case types.TypeVals[types.TVString]:
			str += fmt.Sprintf("\"%s\"", s.StrVal)
		default:
			str += "?"
		}
	case SVar:
		str += "Var"
		str += fmt.Sprintf("[used %v, set %v]", s.IsUsed, s.IsSet)
		if s.Val != nil {
			str += fmt.Sprintf("-> val: %s", s.Val)
		}
	case SFCall:
		str += fmt.Sprintf("FCall(%s)", s.Expr.Args)
	case SFunc:
		str += fmt.Sprintf("Func(%d args)", len(s.ArgDataTypes))
	case SLevel:
		str += "Level"
		//if s.Val != nil {
		//	str += s.Val.EvalString()
		//}
	case SAsign:
		str += "Asig:" + s.Asign.LVal.String() + " = " + s.Asign.RVal.String()
	case SBinary:
		op := lex.TokType(s.Expr.Op)
		left := s.Expr.ELeft
		right := s.Expr.ERight
		str += fmt.Sprintf("EBin: %s, {%s, %s}", op, left.String(), right.String())
	case SUnary:
		op := lex.TokType(s.Expr.Op)
		right := s.Expr.ERight
		str += fmt.Sprintf("EUn: %s, {%s}", op, right.String())
	case SRegexp:
		str += fmt.Sprintf("Regexp: \"%s\", %v", s.StrVal, s.Re)
	case SYara:
		str += fmt.Sprintf("Yara: \"%s\", %v", s.StrVal, s.Yr)
	case SSect:
		str += fmt.Sprintf("Section")
	}
	str += "!]"
	return str
}

// To format for user messages
type USym Sym

func (s *USym) SimpleString() (str string) {
	if s == nil {
		return "nil"
	}

	switch s.SType {
	case SConst:
		switch s.DataType.TVal {
		case types.TypeVals[types.TVInt]:
			str = fmt.Sprintf("%d", s.IntVal)
		case types.TypeVals[types.TVFloat]:
			str = fmt.Sprintf("%f", s.FloatVal)
		case types.TypeVals[types.TVBool]:
			str = fmt.Sprintf("%v", s.BoolVal)
		case types.TypeVals[types.TVString]:
			str = fmt.Sprintf("\"%s\"", s.StrVal)
		default:
			str = "?"
		}
	case SVar:
		str = s.Name
	case SFCall:
		str = fmt.Sprintf("%s(", s.Name)
		for i, a := range s.Expr.Args {
			str += fmt.Sprintf("%s", (*USym)(a))
			if i < len(s.Expr.Args)-1 {
				str += ", "
			}
		}
		str += ")"
	case SFunc:
		str += fmt.Sprintf("%s(%d args)", s.Name, len(s.ArgDataTypes))
	case SLevel:
		str = s.Name
	case SAsign:
		lvalstr := (*USym)(s.Asign.LVal).SimpleString()
		rvalstr := (*USym)(s.Asign.RVal).SimpleString()
		str = lvalstr + " = " + rvalstr
	case SBinary:
		leftstr := (*USym)(s.Expr.ELeft).SimpleString()
		rightstr := (*USym)(s.Expr.ERight).SimpleString()
		opstr := lex.UTokType(s.Expr.Op).String()
		str = "(" + leftstr + " " + opstr + " " + rightstr + ")"
	case SUnary:
		rightstr := (*USym)(s.Expr.ERight).SimpleString()
		opstr := lex.UTokType(s.Expr.Op).String()
		str = opstr + rightstr
	case SRegexp:
		str += fmt.Sprintf("Regexp: \"%s\", %v", s.StrVal, s.Re)
	case SYara:
		str += fmt.Sprintf("Yara: \"%s\", %v", s.StrVal, s.Yr)
	case SSect:
		str += fmt.Sprintf("sectionid:%s", s.Name)
	}
	return str
}

func (s *USym) TypeString() (str string) {
	if s == nil {
		return "nil"
	}
	switch s.SType {
	case SConst:
		str = "constant"
	case SVar:
		str = "variable"
	case SFCall:
		str = "function call"
	case SFunc:
		str = "function"
	case SLevel:
		str = "level"
	case SAsign:
		str = "assignment"
	case SBinary:
		str = "binary expression"
	case SRegexp:
		str = "regular expression"
	case SYara:
		str = "yara rule"
	default:
		str = "unknown symbol"
	}
	str = fmt.Sprintf("%s %s of type %s", str, s.SimpleString(), s.DataType)
	return str
}

func (s *USym) Format(f fmt.State, verb rune) {
	str := ""
	switch verb {
	case 'g':
		str = s.GoString()
	case 's':
		str = s.SimpleString()
	case 't':
		str = s.TypeString()
	default:
		str = fmt.Sprintf("!%%%c: %s", verb, str)
	}
	f.Write([]byte(str))
}

func (s *Sym) Yarify() (err error) {
	s.SType = SYara
	s.Yr, err = extern.YaraRule(s.StrVal)
	if err != nil {
		return err
	}
	return nil
}

func (s *Sym) Regexify() (err error) {
	s.SType = SRegexp
	s.Re, err = regexp.Compile(s.StrVal)
	if err != nil {
		return err
	}
	return nil
}

// only for dead code, if the type is not what it should be, it is true
// so it is not dead
func (s *Sym) IsTrue() (cond bool) {
	if s == nil || s.SType != SConst {
		return true
	}
	return s.BoolVal
}
