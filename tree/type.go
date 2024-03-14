package tree

import (
	"fmt"
	"io"
	"os"
	"rips/rips/extern"
	"rips/rips/lex"
	"rips/rips/types"
)

const (
	DTypes   = false
	DAnnot   = false
	DTypeErr = false
)

// this one annotates and check types, uses Annotate for the Expr
func (p *Prog) TypeCheck(errout io.Writer) (nerr int) {
	for _, ls := range p.Levels {
		ls.Annotate()
	}
	for _, decl := range p.Decls {
		decl.LVal.Annotate()
		if decl.LVal != nil {
			decl.LVal.IsUsed = false
		}
		decl.RVal.Annotate()
		nerr += decl.TypeCheck(errout)
	}
	for _, rs := range p.RuleSects {
		//type from SectId
		s := rs.SectId
		if DTypes {
			fmt.Fprintf(os.Stderr, "Section: %s?\n", s.Name)
		}
		te, ok := types.TypeExprFromNames[s.Name]
		if s.Name == "Message" {
			te = types.MsgGraphType.TExpr
		}
		if !ok {
			s.Errorf(errout, nerr, "unknown secid type %s", s.Name)
			nerr++
		}
		//Compatible with any value
		//Compatible only with (univ, expr), (univ, te)  (xxx, undef) (undef, xxx)
		s.DataType = types.UnivType

		//ADD new type msg+graph compatible with both
		s.DataType.TExpr = te
		for _, r := range rs.Rules {
			r.Expr.Annotate()
			if r.Expr.DataType.TVal != types.TypeVals[types.TVBool] {
				r.Errorf(errout, 0, "incorrect trigger expression should be boolean %s", (*USym)(s))
				nerr++
			}
			if DTypes {
				fmt.Fprintf(os.Stderr, "\tExpr: %s?\n", r.Expr)
			}
			nerr += r.Expr.TypeCheck(errout, s.DataType)
			for _, a := range r.Actions {
				if DTypes {
					fmt.Fprintf(os.Stderr, "\t\tAction: %s\n", r.Expr)
				}
				a.What.Annotate()
				nn := 0
				if a.What.Expr == nil || a.What.SType != SFCall {
					a.What.Errorf(errout, nn,
						"incorrect action %s, can only be a function call", (*USym)(a.What))
					nn++
				}
				if !a.What.IsAction {
					a.What.Errorf(errout, nn,
						"incorrect action %s, an expression cannot be an action", (*USym)(a.What))
					nn++
				}
				nerr += nn
				a.What.IsAction = false //in typecheck it will be checked all are false
				nerr += a.What.TypeCheck(errout, s.DataType)
			}
		}
	}
	for _, v := range p.Env {
		if v.SType == SVar {
			switch {
			case v.IsUsed && !v.IsSet:
				v.Errorf(errout, 0, "var %s used but not set (should be constant)", v.Name)
				nerr++
			case !v.IsUsed && v.IsSet:
				v.Errorf(errout, 0, "var %s set and not used", v.Name)
				nerr++
			case !v.IsUsed && !v.IsSet:
				v.Errorf(errout, 0, "var %s unused and unset", v.Name)
				nerr++
			}
		}
	}
	return nerr
}

func (decl *Decl) TypeCheck(errout io.Writer) (nerr int) {
	lval := decl.LVal
	rval := decl.RVal
	rdt := rval.DataType
	ldt := lval.DataType
	if rdt.IsTypeUndef() {
		lval.Errorf(errout, nerr, "type error in initializer expression %s of type %s\n",
			(*USym)(rval), rval.DataType)
		nerr++
	}
	if ldt.IsTypeUndef() || !ldt.IsTypeCompat(decl.RVal.DataType) {
		lval.Errorf(errout, nerr, "%t incompatible initializer %s of type %s\n",
			(*USym)(lval), (*USym)(rval), rval.DataType)
		nerr++
	}
	return nerr
}

// only for a builtin variable
func (s *Sym) varDeref() (sv *Sym) {
	sv = s
	if s.SType == SVar && s.Val != nil {
		sv = s.Val
	}
	return sv
}

func (s *Sym) islevel() bool {
	s = s.varDeref()
	return s.SType != SLevel
}

func (s *Sym) TypeCheck(errout io.Writer, t types.Type) (nerr int) {
	if DTypes {
		fmt.Fprintf(os.Stderr, "TypeCheck ---> %s\n", s)
	}
	expr := s.Expr
	switch s.SType {
	case SFunc:
		panic("no typecheck for function")
	case SConst:
		if !t.IsTypeCompat(s.DataType) {
			s.Errorf(errout, nerr, "%t (incorrect) in section type  %s\n",
				(*USym)(s), s.DataType, t)
			nerr++
		}
		return
	case SFCall:
		if s.IsAction { //for expressions
			s.Errorf(errout, nerr, "%s expected expression, not an action\n", (*USym)(s))
			nerr++
		}
		if expr == nil || expr.FCall == nil {
			s.Errorf(errout, nerr, "%s bad function call\n", (*USym)(s))
			nerr++
			return
		}
		//CHECKED twice (to annotate and to give the error..., ???)
		argst := expr.FCall.ArgDataTypes
		if !expr.FCall.IsVariadic && len(argst) != len(expr.Args) {
			s.Errorf(errout, nerr, "bad number of args for function, %s expected %d, got %d\n",
				(*USym)(s), len(argst), len(expr.Args))
			nerr++
			break
		}
		//CHECKED twice (to annotate and to give the error..., ???)
		if expr.FCall.IsVariadic && len(argst)-1 > len(expr.Args) {
			s.Errorf(errout, nerr, "not enough args to variadic function %s, min %d\n",
				(*USym)(s), len(argst))
			nerr++
			break
		}

		for i := range expr.Args {
			argt := argst[len(argst)-1] //variadic
			if i < len(argst) {
				argt = argst[i]
			}
			nerr += expr.Args[i].TypeCheck(errout, t)
			if expr.Args[i].DataType.IsTypeUndef() || !t.IsTypeCompat(expr.Args[i].DataType) {
				s.Errorf(errout, nerr, "arg %t of %s of incorrect type %s in section type  %s\n",
					(*USym)(expr.Args[i]), (*USym)(s), argt, t)
				nerr++
				continue
			}
			if expr.Args[i].DataType.IsTypeUndef() || !t.IsTypeCompat(expr.Args[i].DataType) {
				s.Errorf(errout, nerr, "arg %d of %s of incorrect type %s in section type  %s\n",
					i, (*USym)(s), expr.Args[i].DataType, t)
				nerr++
			}
		}
		//special check for set
		if s.Name == "set" {
			if s.DataType.IsTypeUndef() || !t.IsTypeCompat(s.DataType) {
				if expr.Args[0].SType != SVar {
					s.Errorf(errout, nerr, "%s: lval %t is not a variable\n",
						(*USym)(s), (*USym)(expr.Args[0]))
				} else {
					s.Errorf(errout, nerr, "%s: lval %t rval %t in section type %s\n",
						(*USym)(s), (*USym)(expr.Args[0]), (*USym)(expr.Args[1]), t)
				}
				if expr.Args[0].IsBuiltin {
					s.Errorf(errout, nerr, "cannot set builtin var %s\n", expr.Args[0].Name)
				}
				nerr++
				break //skip next test
			}
		}
		//special check for trigger and levelname
		if s.Name == "trigger" || s.Name == "levelname" {
			if s.DataType.IsTypeUndef() || expr.Args[0].islevel() {
				s.Errorf(errout, nerr, "%t is not an SLevel\n", (*USym)(expr.Args[0]))
				nerr++
				break //skip next test
			}
		}
		// special check for plugin and exec (executable path)
		// needs to be an unnamed constant string, if not this is not going to work
		// for now, mandatory
		if s.Name == "exec" || s.Name == "plugin" {
			if !extern.IsExecutable(expr.Args[0].StrVal) {
				s.Errorf(errout, nerr, "%t: %t is not a valid unnamed constant string path for executable\n",
					(*USym)(s), (*USym)(expr.Args[0]))
				nerr++
			}
		}
		if s.Name == "payload" {
			if !extern.IsReadable(expr.Args[0].StrVal) {
				s.Errorf(errout, nerr, "%t: %t is not a valid unnamed constant string path for yara rule\n",
					(*USym)(s), (*USym)(expr.Args[0]))
				nerr++
			}
		}
		if s.DataType.IsTypeUndef() || !t.IsTypeCompat(s.DataType) {
			s.Errorf(errout, nerr, "%t (incorrect) in section type  %s\n", (*USym)(s), t)
			nerr++
		}
	case SVar:
		if s.DataType.IsTypeUndef() || !t.IsTypeCompat(s.DataType) {
			s.Errorf(errout, nerr, "%t (incorrect) in section type  %s\n", (*USym)(s), t)
			nerr++
		}
	case SBinary:
		le := expr.ELeft
		re := expr.ERight
		nerr += le.TypeCheck(errout, t)
		nerr += re.TypeCheck(errout, t)
		if s.DataType.IsTypeUndef() || !t.IsTypeCompat(s.DataType) {
			s.Errorf(errout, nerr, "%t in section type %s\n", (*USym)(s), t)
			nerr++
		}
	case SUnary:
		re := expr.ERight
		if s.DataType.TVal == types.TypeVals[types.TVString] {
			s.Errorf(errout, nerr, "%t no unary operators for strings %s\n", (*USym)(s))
			nerr++
		}
		nerr += re.TypeCheck(errout, t)
		if s.DataType.IsTypeUndef() || !t.IsTypeCompat(s.DataType) {
			s.Errorf(errout, nerr, "%t in section type %s\n", (*USym)(s), t)
			nerr++
		}
	case SLevel, SSect:
		//nothing
	case SNone:
		s.Errorf(errout, 0, "symbol should not be here %s\n", (*USym)(s))
		nerr++
	default:
		s := fmt.Sprintf("not a valid sym type: %s", s)
		panic(s)
	}
	return nerr
}

func dprintf(str string, v ...interface{}) {
	if !DTypeErr {
		return
	}
	fmt.Fprintf(os.Stderr, "annotate type err: "+str, v...)
}

func (s *Sym) Annotate() {
	if DAnnot {
		fmt.Fprintf(os.Stderr, "Annotate ---> %s\n", s)
	}
	expr := s.Expr
	switch s.SType {
	case SLevel:
		if s.SLevel == 0 {
			s.IsReach = true //the init level is always reachable
		}
		s.DataType = types.IntType
	case SFunc:
		panic("no annotation for function")
	case SConst:
		//nothing to do here
		return
	case SFCall:
		if expr == nil || expr.FCall == nil {
			dprintf("sfcall nil ptr\n")
			s.DataType.TExpr = types.TypeExprs[types.TVUndef]
			break
		}
		s.DataType = expr.FCall.DataType
		argst := expr.FCall.ArgDataTypes
		if !expr.FCall.IsVariadic && len(argst) != len(expr.Args) {
			dprintf("sfcall len args\n")
			s.DataType.TExpr = types.TypeExprs[types.TVUndef]
			break
		}
		if expr.FCall.IsVariadic && len(expr.Args) < len(argst)-1 {
			dprintf("sfcall variadic not enough args\n")
			s.DataType.TExpr = types.TypeExprs[types.TVUndef]
			break
		}
		var at types.Type
		wasused := false
		for i := range expr.Args {
			if i > len(argst)-1 && expr.FCall.IsVariadic {
				at = argst[len(argst)-1]
			} else {
				at = argst[i]
			}
			if s.Name == "set" {
				if i == 0 {
					wasused = expr.Args[i].IsUsed //save for later
				}
				if i != 0 && expr.Args[0] == expr.Args[i] {
					wasused = true //save for later
				}
			}
			expr.Args[i].Annotate()
			if !expr.Args[i].DataType.IsTypeCompat(at) {
				dprintf("sfcall is not type compat %s %s\n", expr.Args[i].DataType, at)
				expr.Args[i].DataType.TExpr = types.TypeExprs[types.TVUndef]
				s.DataType.TExpr = types.TypeExprs[types.TVUndef]
			}
		}
		//special check for set
		if s.Name == "set" {
			arg0 := expr.Args[0]
			if arg0.SType != SVar || !arg0.DataType.IsTypeCompat(expr.Args[1].DataType) {
				dprintf("sfcall set, lval != rval type\n")
				s.DataType.TExpr = types.TypeExprs[types.TVUndef]
			} else {
				arg0.IsSet = true
				arg0.IsUsed = wasused
			}
		}
		if s.Name == "trigger" && expr.Args[0].SType == SLevel {
			expr.Args[0].IsReach = true
		}
	case SVar:
		s.IsUsed = true
	case SBinary:
		if expr == nil {
			dprintf("sbinary nil ptr\n")
			s.DataType.TExpr = types.TypeExprs[types.TVUndef]
			break
		}
		le := expr.ELeft
		re := expr.ERight
		le.Annotate()
		re.Annotate()
		islecomp := le.DataType.IsCompat(&re.DataType, expr.Op)
		if !islecomp || le.DataType.IsTypeUndef() || re.DataType.IsTypeUndef() {
			dprintf("sbinary not compat\n")
			s.DataType.TExpr = types.TypeExprs[types.TVUndef]
			break
		}
		s.DataType = le.DataType
		if isbool, _ := isCompOp[lex.TokType(expr.Op)]; isbool {
			s.DataType.TVal = types.TypeVals[types.TVBool]
		}
	case SUnary:
		if expr == nil {
			dprintf("sunary nil ptr\n")
			s.DataType.TExpr = types.TypeExprs[types.TVUndef]
			break
		}
		re := expr.ERight
		if re.DataType.TVal == types.TypeVals[types.TVString] {
			re.DataType.TExpr = types.TypeExprs[types.TVUndef]
		}
		re.Annotate()
		if !re.DataType.IsCompat(nil, expr.Op) || re.DataType.IsTypeUndef() {
			dprintf("sunary not compat\n")
			s.DataType.TExpr = types.TypeExprs[types.TVUndef]
			break
		}
		s.DataType = re.DataType
	case SSect:
		//mark so it is not used as ID
		s.DataType = types.UndefType
	case SNone:
	default:
		errs := fmt.Sprintf("not a value %s", s)
		panic(errs)
	}
	if DAnnot {
		fmt.Fprintf(os.Stderr, "\t->Annotate ---> %s\n", s)
	}
}
