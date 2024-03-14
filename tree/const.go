package tree

import (
	"fmt"
	"io"
	"os"
	"rips/rips/lex"
	"rips/rips/types"
	"strings"
)

const DFold = false

type vardesc struct {
	whereset  *Action //first time set
	set       bool
	whereused *Action //firts time used
	used      bool
}

func (vd *vardesc) String() (str string) {
	wu := (*USym)(vd.whereused.What)
	ws := (*USym)(vd.whereset.What)
	if wu == nil {
		wu = &USym{}
	}
	if ws == nil {
		ws = &USym{}
	}
	str = fmt.Sprintf("whereused:%s whereset:%s u:%v s:%v\n", wu, ws, vd.used, vd.set)
	return str
}

func setused(a *Action, s *Sym, vds map[string]*vardesc) {
	expr := s.Expr
	if expr == nil {
		//should not happen
		return
	}
	switch s.SType {
	case SFCall:
		if s.Name == "set" {
			v := expr.Args[0].Name
			if vd, ok := vds[v]; ok {
				vd.set = true
			} else {
				vds[v] = &vardesc{set: true, whereset: a}
			}
			for i := 0; i < len(expr.Args); i++ {
				setused(a, expr.Args[i], vds)
			}
		}
	case SBinary:
		if s.Expr.ERight == nil {
			return
		}
		setused(a, s.Expr.ERight, vds)
		if s.Expr.ELeft == nil {
			return
		}
		setused(a, s.Expr.ELeft, vds)
	case SUnary:
		if s.Expr.ERight == nil {
			return
		}
		setused(a, s.Expr.ERight, vds)
	case SVar:
		v := s.Name
		if vd, ok := vds[v]; ok {
			vd.used = true
		} else {
			vds[v] = &vardesc{used: true, whereused: a}
		}
	}
}

//constant folding and expression compilation (yara, regex)

func (p *Prog) Fold(errout io.Writer) (nerr int) {
	fakeenv := (*StkEnv)(&[]Env{p.Env})
	for _, decl := range p.Decls {
		n := 0
		n, decl.RVal = decl.RVal.Fold(fakeenv, errout)
		if !decl.RVal.IsConstant() {
			decl.RVal.Errorf(errout, nerr, "%t is not a constant expression\n", (*USym)(decl.RVal))
			nerr++
		}
		if decl.LVal.SType == SVar {
			decl.LVal.Val = decl.RVal
		} else {
			decl.LVal.CopyValFrom(decl.RVal)
		}
		nerr += n
		nerr += decl.Fold(errout)
	}
	for _, rs := range p.RuleSects {
		rules := rs.Rules
		rs.Rules = nil
		for _, r := range rules {
			n := 0
			n, r.Expr = r.Expr.Fold(fakeenv, errout)
			nerr += n
			//delete dead code
			if n == 0 && !r.Expr.IsTrue() {
				continue
			}
			nd := true
			for nd {
				acts := r.Actions
				r.Actions = nil
				vds := make(map[string]*vardesc)
				nd = false
			ActionLoop:
				for _, a := range acts {
					// Local setused dead code elimination
					// only deletes "root" calls to set, not calls
					// to set as arguments because they
					// are not needed.
					// This is forbidden (will not compile): set(wasgood, set(another, true)):
					// Note that sets and actions as arguments are forbidden in general.
					//	and will not compile (type check):
					// This is also forbidden: set(wasgood, exec("xx")):
					// The same can be programmed with a flag and => thus:
					// set(wasgood, false) => exec("xx") => set(wasgood, true)
					// semantically equivalent and equally readable

					setused(a, a.What, vds)
					if a.What.Name == "set" {
						v := a.What.Expr.Args[0].Name
						vd, ok := vds[v]
						if ok && DFold {
							fmt.Fprintf(os.Stderr, "local dead code? %s, %v\n", vd, vd.whereset != a)
						}
						if ok && !vd.used && vd.whereset != a && vd.whereset.What.Name == "set" {
							if DFold {
								fmt.Fprintf(os.Stderr, "local dead deleting %s\n", (*USym)(vd.whereset.What))
							}
							// Note that lateral effects in arguments are
							// forbidden anyway (this check could be deleted)
							vd.whereset.What = NewBool(true)
							vd.whereset = a
							nd = true
						}
					}
					n, a.What = a.What.Fold(fakeenv, errout)
					nerr += n
					//Delete dead code, rest of chain
					if a.What.IsConstant() && a.What.BoolVal && a.Con == lex.TokNThen {
						nd = true
						break ActionLoop
					}
					//Delete dead code, rest of chain
					if a.What.IsConstant() && !a.What.BoolVal && a.Con == lex.TokThen {
						nd = true
						break ActionLoop
					}
					//Delete dead code, only this one
					if a.What.IsConstant() {
						nd = true
						continue
					}
					r.Actions = append(r.Actions, a)
				}
			}
			if len(r.Actions) == 0 {
				continue
			}
			rs.Rules = append(rs.Rules, r)
		}
	}
	return nerr
}

func (decl *Decl) Fold(errout io.Writer) (nerr int) {
	return nerr
}

func (s *Sym) IsConstant() bool {
	if s.SType == SConst || s.SType == SLevel {
		return true
	}
	return false
}

func (s *Sym) FoldInt(e1 *Sym, e2 *Sym, val int64) (news *Sym) {
	news = s
	if e1.IsConstant() && val == e1.IntVal {
		news = e2
	}
	if e2.IsConstant() && val == e2.IntVal {
		news = e1
	}
	return news
}

func (s *Sym) FoldFloat(e1 *Sym, e2 *Sym, val float64) (news *Sym) {
	news = s
	if e1.IsConstant() && val == e1.FloatVal {
		news = e2
	}
	if e2.IsConstant() && val == e2.FloatVal {
		news = e1
	}
	return news
}

func (s *Sym) FoldBinary(envs *StkEnv, errout io.Writer) (news *Sym) {
	news = s
	isint := s.DataType.TVal == types.TypeVals[types.TVInt]
	isfloat := s.DataType.TVal == types.TypeVals[types.TVFloat]
	if s.Expr.ELeft.IsConstant() && s.Expr.ERight.IsConstant() {
		if DFold {
			fmt.Fprintf(os.Stderr, "Fold: %s %s\n", s.Expr.ELeft, s.Expr.ERight)
		}
		news = s.EvalExpr(envs, nil)
		return news
	}
	if lex.TokType(s.Expr.Op) == lex.TokLogOr && s.Expr.ELeft.IsConstant() {
		news = s.Expr.ERight
		if s.Expr.ELeft.BoolVal {
			news = NewBool(true)
		}
		return news
	}
	if lex.TokType(s.Expr.Op) == lex.TokLogAnd && s.Expr.ELeft.IsConstant() {
		news = s.Expr.ERight
		if !s.Expr.ELeft.BoolVal {
			news = NewBool(false)
		}
		return news
	}
	if s.Expr.Op == '+' {
		if isint {
			news = s.FoldInt(s.Expr.ELeft, s.Expr.ERight, 0)
		}
		if isfloat {
			news = s.FoldFloat(s.Expr.ELeft, s.Expr.ERight, 0)
		}
		return news
	}
	if s.Expr.Op == '|' {
		news = s.FoldInt(s.Expr.ELeft, s.Expr.ERight, 0)
		return news
	}
	if s.Expr.Op == '-' {
		rightzero := isint && (s.Expr.ERight.IntVal == 0)
		rightzero = rightzero || (isfloat && (s.Expr.ERight.FloatVal == 0))
		if s.Expr.ERight.IsConstant() && rightzero {
			news = s.Expr.ELeft
		}
		return news
	}
	if s.Expr.Op == '&' {
		news = s.FoldInt(s.Expr.ELeft, s.Expr.ERight, -1)
		return news
	}
	if s.Expr.ERight.SType == SVar {
		//floats can be different to themselves
		if !isfloat && s.Expr.ELeft.Name == s.Expr.ERight.Name {
			if lex.TokType(s.Expr.Op) == lex.TokEq {
				news = NewBool(true)
			} else if lex.TokType(s.Expr.Op) == lex.TokNEq {
				news = NewBool(false)
			}
			return news
		}
	}
	return news
}

func (s *Sym) iszerodiv() bool {
	if s.Expr.Op == '/' {
		er := s.Expr.ERight
		if !er.IsConstant() {
			return false
		}
		isint := s.DataType.TVal == types.TypeVals[types.TVInt]
		isfloat := s.DataType.TVal == types.TypeVals[types.TVFloat]
		if isint && er.IntVal == 0 {
			return true
		}
		if isfloat && er.FloatVal == 0 {
			return true
		}
	}
	return false
}

func (s *Sym) Fold(envs *StkEnv, errout io.Writer) (nerr int, news *Sym) {
	news = s
	expr := s.Expr
	switch s.SType {
	case SRegexp, SYara, SFunc, SConst, SLevel, SVar, SSect:
	case SFCall:
		if expr == nil {
			return 1, news
		}
		if s.Name == "set" {
			if expr != nil && len(expr.Args) >= 2 && expr.Args[0] == expr.Args[1] {
				return 0, NewBool(true)
			}
		}
		//dead code for variadic set functions.
		isinclude := strings.HasSuffix(s.Name, "in") || strings.HasSuffix(s.Name, "include")
		if s.Name != "plugin" && isinclude && len(expr.Args) == 0 {
			return 0, NewBool(true)
		}
		//this check should never happen, it is defensive (should be caught in type.go)
		if (s.Name == "payload" || s.Name == "topicmatches") && len(expr.Args) == 0 {
			s.Errorf(errout, nerr, "%t not enough arguments\n",
				(*USym)(s))
			nerr++
			return 1, news
		}
		//special check for set
		if s.Name == "payload" && len(expr.Args) != 0 {
			a0 := expr.Args[0]
			if a0.SType != SConst {
				s.Errorf(errout, nerr, "%t is not a constant string for yara rule\n",
					(*USym)(a0))
				nerr++
			}
			err := a0.Yarify()
			if err != nil || expr.Args[0].Yr == nil {
				s.Errorf(errout, nerr, "%t is not a valid yara rule yara error: [%s]\n",
					(*USym)(a0), err)
				nerr++
			}
		}
		if s.Name == "topicmatches" && len(expr.Args) != 0 {
			a0 := expr.Args[0]
			if a0.SType != SConst {
				s.Errorf(errout, nerr, "%t is not a constant string for regexp\n",
					(*USym)(a0))
				nerr++
			}
			err := a0.Regexify()
			if err != nil || a0.Re == nil {
				s.Errorf(errout, nerr, "%t is not a valid regexp %s\n",
					(*USym)(a0), err)
				nerr++
			}
		}
		n := 0
		for i := 0; i < len(expr.Args); i++ {
			n, expr.Args[i] = expr.Args[i].Fold(envs, errout)
			nerr += n
		}
		if len(expr.Args) == 0 {
			return nerr, news
		}
		if s.Name == "string" && expr.Args[0].IsConstant() {
			if DFold {
				fmt.Fprintf(os.Stderr, "Fold: %s\n", s)
			}
			news = String(nil, expr.Args[0])
		}
		if s.Name == "levelname" && expr.Args[0].SType == SLevel {
			if DFold {
				fmt.Fprintf(os.Stderr, "Fold: %s\n", s)
			}
			news = NewString(expr.Args[0].Name)
		}
	case SBinary:
		n := 0
		if s.Expr.ELeft != nil {
			n, s.Expr.ELeft = s.Expr.ELeft.Fold(envs, errout)
			nerr += n
		}
		if s.Expr.ERight != nil {
			n, s.Expr.ERight = s.Expr.ERight.Fold(envs, errout)
			nerr += n
		}
		if s.Expr.ELeft != nil && s.Expr.ERight != nil {
			if DFold {
				fmt.Fprintf(os.Stderr, "Fold? %s %s\n", s.Expr.ELeft, s.Expr.ERight)
			}
			news = s.FoldBinary(envs, errout)
			if s.iszerodiv() {
				s.Errorf(errout, nerr, "division by zero %s\n", (*USym)(s))
				nerr++
			}
		}
	case SUnary:
		n := 0
		if s.Expr.ERight == nil {
			break
		}
		n, s.Expr.ERight = s.Expr.ERight.Fold(envs, errout)
		nerr += n
		if DFold {
			fmt.Fprintf(os.Stderr, "Fold? %s\n", s.Expr.ERight)
		}
		if s.Expr.ERight.IsConstant() {
			if DFold {
				fmt.Fprintf(os.Stderr, "Fold: %s\n", s.Expr.ERight)
			}
			news = s.EvalExpr(envs, nil)
		}
	case SNone:
		nerr++
	default:
		panic("should not happen")
	}
	return nerr, news
}
