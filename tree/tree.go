package tree

import (
	"fmt"
	"io"
	"os"
	"rips/rips/lex"
)

type Prog struct {
	Env       Env //global variables, kept for execution, see PushVars
	Levels    []*Sym
	Decls     []*Decl
	RuleSects []*RuleSect
}

type RuleSect struct {
	SectId *Sym
	Rules  []*Rule
}

func (envs *StkEnv) NewRuleSect(name string, pos lex.Position) (rs *RuleSect, err error) {
	id, err := envs.NewSym(name, SSect)
	if err != nil {
		return nil, err
	}
	id.Pos = pos
	return &RuleSect{SectId: id, Rules: nil}, nil
}

func NewProg() (prog *Prog) {
	return &Prog{}
}

func (s *Sym) assertype(stype int) {
	if s.SType != stype {
		panic("cannot happen, wrong stype")
	}
}

func (p *Prog) AddLevel(level *Sym) int {
	n := len(p.Levels)
	level.assertype(SLevel)
	p.Levels = append(p.Levels, level)
	return n
}

func (p *Prog) AddDecl(decl *Decl) {
	p.Decls = append(p.Decls, decl)
}

type Rule struct {
	Pos     lex.Position
	Expr    *Sym
	Actions []*Action
}

func (r *Rule) Errorf(errout io.Writer, nerr int, str string, v ...interface{}) {
	if nerr >= 1 {
		return
	}
	place := fmt.Sprintf("%s:%d: ", r.Pos.File, r.Pos.Line)
	out := errout
	if lex.DOut {
		out = os.Stderr
	}
	fmt.Fprintf(out, place+str+"\n", v...)
}

type Action struct {
	Con  lex.TokType //connector => and so on
	What *Sym        //can only be an fcall for now (maybe assign later)
}

func NewRule(p lex.Position, expr *Sym) (rule *Rule) {
	return &Rule{p, expr, nil}
}

func (r *RuleSect) AddRule(rule *Rule) {
	r.Rules = append(r.Rules, rule)
}
func (p *Prog) AddRuleSect(sect *RuleSect) {
	p.RuleSects = append(p.RuleSects, sect)
}

func NewAction(expr *Sym, con lex.TokType) (action *Action) {
	return &Action{con, expr}
}

func (r *Rule) AddAction(a *Action) {
	r.Actions = append(r.Actions, a)
}

type Asign struct {
	LVal *Sym
	RVal *Sym
}

func NewAsign(lval *Sym, rval *Sym) (decl *Asign) {
	return &Asign{LVal: lval, RVal: rval}
}

type Decl struct {
	Asign
}

func NewDecl(lval *Sym, rval *Sym) (decl *Decl) {
	return &Decl{Asign{LVal: lval, RVal: rval}}
}

type Expr struct {
	FCall  *Sym   //for fcall, when resolved
	Args   []*Sym //when fcall
	Op     int
	ELeft  *Sym
	ERight *Sym
}

func (expr *Sym) AddArg(arg *Sym) {
	e := expr.Expr
	if e == nil {
		return
	}
	e.Args = append(expr.Expr.Args, arg)
}

func (expr *Sym) AddLeft(left *Sym) {
	e := expr.Expr
	if left == nil || e == nil {
		return
	}
	e.ELeft = left
	expr.SType = SBinary
}

func (expr *Sym) AddRight(right *Sym) {
	if right == nil {
		return
	}
	e := expr.Expr
	if e == nil {
		return
	}
	e.ERight = right
	if expr.SType == SConst {
		expr.SType = SUnary
	}
}

func NewExpr(left *Sym, right *Sym) (expr *Sym) {
	e := &Expr{ELeft: left, ERight: right}
	expr = NewAnonSym(SBinary)
	if left == nil {
		if right == nil {
			expr = NewAnonSym(SConst)
		} else if right != nil {
			expr = NewAnonSym(SUnary)
		}
	}
	expr.Expr = e
	return expr
}

func (expr *Sym) AddFCall(f *Sym) {
	e := expr.Expr
	if e == nil {
		return
	}
	e.FCall = f
	expr.SType = SFCall
}
