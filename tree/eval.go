package tree

import (
	"errors"
	"fmt"
	"os"
	"rips/rips/extern"
	"rips/rips/lex"
	"rips/rips/types"
	"strings"
)

const (
	DEval = false
)

func (envs *StkEnv) dprintf(s string, v ...interface{}) {
	lvl := len(*envs)
	indent := strings.Repeat("\t", lvl)
	if DEval {
		prefix := fmt.Sprintf("%sEVAL: ", indent)
		fmt.Fprintf(os.Stderr, prefix+s, v...)
	}
}

func (s *Sym) SetVal(s2 *Sym) {
	s.Val = s2
	s.DataType = s2.DataType
}

func isRuntime(s string) bool {
	return strings.Contains(s, "runtime error:")
}

func isOpPanic(s string) bool {
	isshift := strings.Contains(s, "negative shift")
	isdiv := strings.Contains(s, "divide by zero")
	return isdiv || isshift
}

// This method fixes panics from operations:
//
//	panic: runtime error: negative shift amount
//	panic: runtime error: integer divide by zero
//	floating point is spec dependant
func (s *Sym) OpVal(s2 *Sym, op int) (err error) {
	defer func() {
		if r := recover(); r != nil {
			errs := fmt.Sprintf("%s", r)
			if isRuntime(errs) && isOpPanic(errs) {
				str, _ := strings.CutPrefix(errs, "panic: runtime error: ")
				err = errors.New(str)
				return
			}
			panic(errs)
		}
	}()
	if s2 != nil && s2.DataType.IsTypeUndef() {
		s.DataType.TVal = types.TypeVals[types.TVUndef]
	}
	switch s.DataType.TVal {
	case types.TypeVals[types.TVInt]:
		s.IntOp(s2, op)
	case types.TypeVals[types.TVFloat]:
		s.FloatOp(s2, op)
	case types.TypeVals[types.TVString]:
		s.StrOp(s2, op)
	case types.TypeVals[types.TVBool]:
		s.BoolOp(s2, op)
	case types.TypeVals[types.TVUndef]:
		return nil
	default:
		return errors.New("bad op")
	}
	return nil
}

type CompType interface {
	int64 | float64 | ~string
}

func CompOp[T CompType](v1 T, v2 T, op int) (res bool, err error) {
	opt := lex.TokType(op)
	switch opt {
	case lex.TokL:
		return v1 < v2, nil
	case lex.TokG:
		return v1 > v2, nil
	case lex.TokEq:
		return v1 == v2, nil
	case lex.TokNEq:
		return v1 != v2, nil
	case lex.TokGEq:
		return v1 >= v2, nil
	case lex.TokLEq:
		return v1 <= v2, nil
	}
	err = fmt.Errorf("unsuported boolean operation: %s\n", opt)
	return false, err
}

/*
	***WARNING. Do not add >> or << operator
		without adding an unsigned
		integer type.
*/

func (s *Sym) IntOp(s2 *Sym, op int) (err error) {
	v := int64(0)
	if s2 == nil {
		v = s.IntVal
		s.IntVal = 0 //hack for -3 unary op
	} else {
		v = s2.IntVal
	}
	if isbool, _ := isCompOp[lex.TokType(op)]; isbool {
		s.BoolVal, err = CompOp(s.IntVal, v, op)
	}
	switch op {
	case '^':
		s.IntVal ^= v
	case '~':
		s.IntVal = ^v
	case '+':
		s.IntVal += v
	case '-':
		s.IntVal -= v
	case '*':
		s.IntVal *= v
	case '/':
		s.IntVal /= v
	case '%':
		s.IntVal %= v
	case '&':
		s.IntVal &= v
	case '|':
		s.IntVal |= v
	default:
		return errors.New("undef int op")
	}
	return nil
}

func (s *Sym) FloatOp(s2 *Sym, op int) (err error) {
	v := 0.0
	if s2 == nil {
		v = s.FloatVal
		s.FloatVal = 0.0 //hack for -3.0
	} else {
		v = s2.FloatVal
	}
	if isbool, _ := isCompOp[lex.TokType(op)]; isbool {
		s.BoolVal, err = CompOp(s.FloatVal, v, op)
		return err
	}
	switch op {
	case '+':
		s.FloatVal += v
	case '-':
		s.FloatVal -= v
	case '*':
		s.FloatVal *= v
	case '/':
		s.FloatVal /= v
	default:
		return errors.New("undef float op")
	}
	return nil
}

func intFromBool(v bool) int64 {
	if v {
		return 1
	}
	return 0
}

// Take great care, s2 can be nil for unary
func (s *Sym) BoolOp(s2 *Sym, op int) (err error) {
	if isbool, _ := isCompOp[lex.TokType(op)]; isbool {
		//bool is not comparable in go but it is in our language
		// we convert to int to fix this
		s.BoolVal, err = CompOp(intFromBool(s.BoolVal), intFromBool(s2.BoolVal), op)
		return err
	}
	opt := lex.TokType(op)
	switch opt {
	case lex.TokLogNeg:
		s.BoolVal = !s.BoolVal
	case lex.TokLogAnd:
		s.BoolVal = s.BoolVal && s2.BoolVal
	case lex.TokLogOr:
		s.BoolVal = s.BoolVal || s2.BoolVal
	default:
		return errors.New("undef bool op")
	}
	return nil
}

const (
	MaxStr = 4 * 1024
)

// take great care, s2 can be nil for unary
func (s *Sym) StrOp(s2 *Sym, op int) (err error) {
	switch op {
	case '+':
		if s2 == nil {
			return errors.New("undef unary str op")
		}
		s.StrVal += s2.StrVal
		r := []rune(s.StrVal)
		if len(r) > MaxStr {
			fmt.Fprintf(os.Stderr, "warning: string %s too long, truncating to len %d\n", s.StrVal, MaxStr)
			s.StrVal = string(r[0:MaxStr])
		}
	}
	if isbool, _ := isCompOp[lex.TokType(op)]; isbool {
		s.BoolVal, err = CompOp(s.StrVal, s2.StrVal, op)
		return err
	}
	return errors.New("undef str op")
}

func (s *Sym) BinExpr(s2 *Sym, op int) (err error) {
	if !s.DataType.IsCompat(&s2.DataType, op) {
		errs := fmt.Sprintf("\tUncompat Types %s %s for op %c", s.DataType, s2.DataType, rune(op))
		err = errors.New(errs)
		s.DataType.TVal = types.TypeVals[types.TVUndef]
	} else {
		err = s.OpVal(s2, op)
	}
	return err
}

func (s *Sym) UnaryExpr(op int) (err error) {
	if !s.DataType.IsCompat(nil, op) {
		errs := fmt.Sprintf("\tUncompat Type %s for op %c", s.DataType, rune(op))
		err = errors.New(errs)
		s.DataType.TVal = types.TypeVals[types.TVUndef]
	} else {
		err = s.OpVal(nil, op)
	}
	return err
}

func (s *Sym) CopyValFrom(s2 *Sym) {
	if s2.SType == SVar && s2.Val != nil {
		s.DataType = s2.Val.DataType
		s2 = s2.Val //take val to s2 to copy to s
	} else {
		s.DataType = s2.DataType
	}
	switch s.DataType.TVal {
	case types.TypeVals[types.TVBool]:
		s.BoolVal = s2.BoolVal
	case types.TypeVals[types.TVInt]:
		s.IntVal = s2.IntVal
	case types.TypeVals[types.TVFloat]:
		s.FloatVal = s2.FloatVal
	case types.TypeVals[types.TVString]:
		s.StrVal = s2.StrVal
	default:
		fmt.Fprintf(os.Stderr, "Copying unknown type %s to %s\n", s2, s)
	}
}

func (s *Sym) BoolExpr(op int) (err error) {
	s.DataType.TVal = types.TypeVals[types.TVBool]
	return nil
}

func (s *Sym) EvalExpr(envs *StkEnv, context *extern.Ctx) (val *Sym) {
	envs.dprintf(" ---> %s\n", s)
	val = NewAnonSym(SConst)
	val.DataType.TExpr = types.TypeExprs[types.TEExpr]
	switch s.SType {
	case SConst:
		envs.dprintf("SConst\n")
		val.CopyValFrom(s)
	case SVar:
		envs.dprintf("SVar\n")
		v := envs.GetSym(s.Name)
		if v == nil {
			panic("bad var, cannot happen")
		}
		*val = *(v.Val)
	case SFCall:
		envs.dprintf("SFcall\n")
		fn := s.Expr.FCall
		if fn == nil {
			break
		}
		args := make([]*Sym, len(s.Expr.Args))
		for i, p := range s.Expr.Args {
			if s.Name == "set" && i == 0 {
				args[i] = envs.GetSym(p.Name)
				continue //first arg to set is an LValue, do not evaluate
			}
			args[i] = p.EvalExpr(envs, context)
		}
		if s.Name == "trigger" {
			//prepend the symbol with the current level
			level := envs.GetSym("CurrLevel")
			if level == nil {
				panic("Level var disappeared")
			}
			args = append([]*Sym{level}, args...)
		}
		val = fn.Fn(context, args...)
	case SBinary:
		envs.dprintf("SBinary\n")
		left := s.Expr.ELeft.EvalExpr(envs, context)
		val.CopyValFrom(left)
		isshort := s.IsOrShort(val) || s.IsAndShort(val)
		if !isshort {
			right := s.Expr.ERight.EvalExpr(envs, context)
			err := val.BinExpr(right, s.Expr.Op)
			if err != nil && context != nil {
				context.Printf("%s:%d error evaluating, undefined behaviour: %s\n", s.Pos.File, s.Pos.Line, err)
				context.Fatal()
			}
		}
		if isbool, _ := isCompOp[lex.TokType(s.Expr.Op)]; isbool {
			val.BoolExpr(s.Expr.Op)
		}
	case SUnary:
		envs.dprintf("SUnary\n")
		right := s.Expr.ERight.EvalExpr(envs, context)
		val.CopyValFrom(right)
		val.UnaryExpr(s.Expr.Op)
		if isbool, _ := isCompOp[lex.TokType(s.Expr.Op)]; isbool {
			val.BoolExpr(s.Expr.Op)
		}
	case SLevel:
		return s
	case SYara, SRegexp, SNone:
		return s
	default:
		panic("not a value: " + s.Name)
	}

	envs.dprintf("\n\t--->val:  %v\n", val)
	return val
}

var isCompOp = map[lex.TokType]bool{
	lex.TokG:   true,
	lex.TokL:   true,
	lex.TokEq:  true,
	lex.TokNEq: true,
	lex.TokGEq: true,
	lex.TokLEq: true,
}
