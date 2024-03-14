package tree

import (
	"rips/rips/extern"
	"rips/rips/lex"
)

func prVars(envs *StkEnv) {
	currEnv := envs.CurrEnv()
	for _, v := range currEnv {
		if v.SType == SVar {
			envs.dprintf("VAR %s\n", v.Val)
		}
		if v.SType == SLevel {
			envs.dprintf("SLEVEL %s\n", v)
		}
		if v.SType == SConst {
			envs.dprintf("CONST %s\n", v.Val)
		}
	}
}

func (p *Prog) PushSyms(envs *StkEnv) {
	envs.PushEnv()
	for _, v := range p.Env {
		if v.SType != SVar && v.SType != SLevel && v.SType != SConst {
			continue
		}
		envs.dprintf("PUSHING Sym %s\n", v.Name)
		if v.Val != nil {
			envs.dprintf("\t\tval %s\n", v.Val)
		}
		s, err := envs.NewSym(v.Name, v.SType)
		if err != nil {
			panic(err)
		}
		s.DataType = v.DataType
		if v.SType == SVar {
			val := NewAnonSym(SConst)
			val.CopyValFrom(v)
			s.SetVal(val)
		} else {
			s.CopyValFrom(v)
		}
	}
}

func (p *Prog) PopSyms(envs *StkEnv) {
	for _, v := range envs.CurrEnv() {
		if v.SType == SVar || v.SType == SLevel || v.SType == SConst {
			envs.dprintf("POPING Sym %s\n", v.Name)
		}
	}
	envs.PopEnv()
}

func (r *Rule) Interp(context *extern.Ctx, execEnv *StkEnv) {
	execEnv.dprintf("Rule Expr: %s\n", r.Expr)
	val := r.Expr.EvalExpr(execEnv, context)
	execEnv.dprintf("Rule ExprVal: %s\n", val)
	if val.BoolVal {
		execEnv.dprintf("Rule Interp: activated %s\n", r)
		donext := true
		issuccess := true
		for _, a := range r.Actions {
			switch {
			case issuccess && a.Con == lex.TokThen:
				fallthrough
			case !issuccess && a.Con == lex.TokNThen:
				fallthrough
			case a.Con == lex.TokComma:
				donext = true
			default:
				donext = false
			}
			execEnv.dprintf("do next %v\n", donext)
			if !donext {
				break
			}
			actVal := a.What.EvalExpr(execEnv, context)
			issuccess = actVal.BoolVal
			execEnv.dprintf("is successful %v %s\n", issuccess, lex.TokType(a.Con))
		}
	}
}
func (p *Prog) NewExecEnv(context *extern.Ctx) (execEnv *StkEnv) {
	execEnv = new(StkEnv) //execution stack
	execEnv.dprintf("Prog\n")
	execEnv.PushEnv() //general protection, not popable

	execEnv.PushEnv() //builtins
	execEnv.Builtins(Builtins)
	execEnv.PredefVars()
	p.PushSyms(execEnv)
	return execEnv
}

func (p *Prog) Interp(context *extern.Ctx, execEnv *StkEnv) {

	err := execEnv.SetPredefVars(p, context)
	if err != nil {
		panic(err)
	}
	tm := "External"
	if context.CurrentMsg != nil {
		tm = context.CurrentMsg.Type()
	}
	for _, rs := range p.RuleSects {
		if rs.SectId.Name == tm {
			execEnv.dprintf("Section Interp: for msg type %s: %s\n", tm, rs)
			for _, r := range rs.Rules {
				r.Interp(context, execEnv)
			}
			break
		}
	}
}
func (p *Prog) Done(execEnv *StkEnv) {
	p.PopSyms(execEnv)
}
