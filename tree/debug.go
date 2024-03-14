package tree

import (
	"fmt"
	"rips/rips/lex"
)

//Helper functions to print the AST

func (env *Env) String() (s string) {
	for k, v := range *env {
		s += fmt.Sprintf("\t\tk: %s, v: %s\n", k, v)
	}
	return s
}

func (prog *Prog) String() (s string) {
	s += fmt.Sprintf("Env: %s\n", &prog.Env)
	s += "Levels:\n"
	for _, ls := range prog.Levels {
		s += fmt.Sprintf("\t%s\n", ls)
	}
	s += "Declarations:\n"
	for _, dec := range prog.Decls {
		s += fmt.Sprintf("\t%s", dec)
	}
	s += fmt.Sprintf("Rules:\n")
	for _, rs := range prog.RuleSects {
		s += fmt.Sprintf("%s", rs)
	}
	return s
}
func (decl *Decl) String() (s string) {
	if decl.LVal.SType == SVar {
		s += fmt.Sprintf("var %s = %s\n", (*USym)(decl.LVal), decl.RVal)
	} else {
		s += fmt.Sprintf("const %s = %s\n", decl.LVal.Name, decl.RVal)
	}
	return s
}
func (ruledecl *RuleSect) String() (s string) {
	s += fmt.Sprintf("\tSection %s:\n", ruledecl.SectId)
	for _, r := range ruledecl.Rules {
		s += fmt.Sprintf("\t%s", r)
	}
	return s
}
func (rule *Rule) String() (s string) {
	s += fmt.Sprintf("\t\t%s?\n", (*USym)(rule.Expr))
	s += "\t\t\t\t\t"
	for _, a := range rule.Actions {
		s += fmt.Sprintf("%s", a)
	}
	return s + "\n"
}
func (action *Action) String() (s string) {
	s += fmt.Sprintf(" %s ", (lex.UTokType)(action.Con))
	s += fmt.Sprintf("%s", (*USym)(action.What))
	return s
}
