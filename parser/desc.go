package parser

import (
	"errors"
	"fmt"
	"log"
	"os"
	"rips/rips/lex"
	"rips/rips/tree"
	"rips/rips/types"
	"strings"
)

var DebugDesc = false

type Parser struct {
	l     lex.LexPeeker
	depth int
	tag   []string
	nErr  int
	Envs  tree.StkEnv
}

func NewParser(l lex.LexPeeker) *Parser {
	return &Parser{l, 0, nil, 0, nil}
}

func (p *Parser) match(tt lex.TokType) (tok lex.Token, err error, ismatch bool) {
	//p.dPrintf("match: trying to find: %s\n", tt)
	tok, err = p.l.Peek()
	if err != nil {
		return lex.Token{}, err, false
	}
	//p.dPrintf("match: peek %s\n", tok)
	if tok.Type != tt {
		p.dPrintf("no match for %s,  found: %s\n", tt, tok)
		return tok, nil, false
	}
	p.l.Lex() //already peeked
	p.dPrintf("match %s\n", tok)
	return tok, nil, true
}

func (p *Parser) matchErr(tt lex.TokType) (tok lex.Token, ismatch bool) {
	var err error
	if tok, err, ismatch = p.match(tt); err != nil {
		p.Errorf("expected %s but err: %s", lex.UTokType(tt), err)
		//try to resynchronize
		_, _ = p.NextSync()
		return tok, false
	} else if !ismatch {
		p.Errorf("expected %s found %s", lex.UTokType(tt), tok.Lexema)
		//try to resynchronize
		_, _ = p.NextSync()
		return tok, false
	}
	return tok, true
}

// LEVELDECLS := 	ID ATTROPT ';' LEVELDECLS
//
//	ε
//
// ATTROPT :=	'soft'
//
//	ε
func (p *Parser) LevelDecls(prog *tree.Prog) (err error) {
	var tokid lex.Token
	p.pushTrace("LevelDecls")
	defer p.popTrace(&err)
	istokid := false
	if tokid, err, istokid = p.match(lex.TokId); !istokid && err == nil {
		//ε
		return err
	}
	if err != nil {
		p.Errorf("expected level id, error: %s", err)
		p.NextSync()
		return p.LevelDecls(prog)
	}
	level, err := p.Envs.NewSym(tokid.Lexema, tree.SLevel)
	if err != nil {
		p.Errorf("declaring %s: %s", tokid.Lexema, err)
		_, err = p.NextSync()
		if err != nil {
			return err
		}
		return p.LevelDecls(prog)
	}
	level.SLevel = prog.AddLevel(level)
	level.DataType = types.IntType     //in case they are compared
	level.IntVal = int64(level.SLevel) //in case they are compared
	level.Pos = p.l.Pos()
	t, err := p.l.Peek()
	if t.Type == lex.TokSoft {
		level.IsSoft = true
		p.l.Lex() //already peeked
	}
	//instantiate new level
	p.matchErr(lex.TokSemi) //continue in case it recovered
	return p.LevelDecls(prog)
}

var tokEndSect = []lex.TokType{
	lex.TokLevels,
	lex.TokVars,
	lex.TokConsts,
	lex.TokRules,
	lex.TokEof,
}

func (p *Parser) IsEndSection() (isend bool, err error) {
	tok, err := p.l.Peek() //for recovery, try to see if it is end of section
	switch tok.Type {
	//same as tokEndSect, just for efficiency a switch
	case lex.TokLevels, lex.TokVars, lex.TokConsts, lex.TokRules, lex.TokEof:
		return true, err
	}
	return false, err
}

// CONSTDECLS :=	ID ID '=' EXPR ';' CONSTDECLS
//
//	ε
func (p *Parser) ConstDecls(prog *tree.Prog) (err error) {
	p.pushTrace("ConstDecls")
	defer p.popTrace(&err)
	var (
		constname lex.Token
		tokid     lex.Token
		typename  lex.Token
	)
	istokid := false
	if tokid, err, istokid = p.match(lex.TokId); !istokid {
		//ε
		if isend, err := p.IsEndSection(); isend || err != nil {
			//in case of recover, no endless recurring
			return err
		}
		t, _ := p.NextSync()
		if err != nil || t.Type == lex.TokEof {
			return errors.New("unexpected eof or error")
		}
		return p.ConstDecls(prog)
	}
	constname = tokid
	if tokid, istokid = p.matchErr(lex.TokId); !istokid {
		//only one error, add the fake constant
		p.Envs.NewConst(constname.Lexema, types.UndefType.TVal)
		p.NextSync() //try to recover
		return p.ConstDecls(prog)
	}
	typename = tokid
	consttype, ok := types.TypeValsFromNames[typename.Lexema]
	if !ok {
		p.Errorf("%s is not a type", typename.Lexema)
		consttype = types.TypeVals[types.TVUndef] //for later...
	}
	sconst, err := p.Envs.NewConst(constname.Lexema, consttype)
	if err != nil {
		p.Errorf("declaring %s: %s", typename.Lexema, err)
		sconst = tree.NewAnonSym(tree.SConst) //inject fake var
		sconst.DataType.TVal = consttype
	}
	sconst.Pos = p.l.Pos()
	if _, istokas := p.matchErr(lex.TokAsig); !istokas {
		return p.ConstDecls(prog) //in case it recovered
	}

	expr, err := p.Expr(-1)
	if expr == nil || err != nil {
		err = fmt.Errorf("incorrect expression %s...: %s", (*tree.USym)(expr), err)
		p.Errorf("%s", err)
		//try to resynchronize
		if t, errrecov := p.NextSync(); errrecov != nil || t.Type != lex.TokSemi {
			return errrecov
		}
		return p.ConstDecls(prog)
	}
	p.dPrintf("value: %s\n", expr)
	sconst.DataType = types.UnivType
	sconst.DataType.TVal = consttype
	//Decl repeats info in sconst, important for debug, reformatting, etc
	decl := tree.NewDecl(sconst, expr) //delayed until assignment
	prog.AddDecl(decl)
	p.matchErr(lex.TokSemi) //on error continue (recovered)
	return p.ConstDecls(prog)
}

// VARDECLS :=	ID ID '=' EXPR ';' VARDECLS
//
//	ε
func (p *Parser) VarDecls(prog *tree.Prog) (err error) {
	p.pushTrace("VarDecls")
	defer p.popTrace(&err)
	var (
		varname  lex.Token
		tokid    lex.Token
		typename lex.Token
	)
	istokid := false
	if tokid, err, istokid = p.match(lex.TokId); !istokid {
		//ε
		if isend, err := p.IsEndSection(); isend || err != nil {
			//in case of recover, no endless recurring
			return err
		}
		t, _ := p.NextSync()
		if err != nil || t.Type == lex.TokEof {
			return errors.New("unexpected eof or error")
		}
		return p.VarDecls(prog)
	}
	varname = tokid
	if tokid, istokid = p.matchErr(lex.TokId); !istokid {
		//only one error, add the fake var
		p.Envs.NewVar(varname.Lexema, types.UndefType.TVal)
		p.NextSync() //try to recover
		return p.VarDecls(prog)
	}
	typename = tokid
	vartype, ok := types.TypeValsFromNames[typename.Lexema]
	if !ok {
		p.Errorf("%s is not a type", typename.Lexema)
		vartype = types.TypeVals[types.TVUndef] //for later...
	}
	svar, err := p.Envs.NewVar(varname.Lexema, vartype)
	if err != nil {
		p.Errorf("declaring %s: %s", typename.Lexema, err)
		svar = tree.NewAnonSym(tree.SVar) //inject fake var
		svar.DataType.TVal = vartype
	}
	svar.Pos = p.l.Pos()
	if _, istokas := p.matchErr(lex.TokAsig); !istokas {
		return p.VarDecls(prog) //in case it recovered
	}

	expr, err := p.Expr(-1)
	if expr == nil || err != nil {
		svar.Val = tree.NewAnonSym(tree.SConst) //make up, but there is no decl
		err = fmt.Errorf("incorrect expression %s...: %s", (*tree.USym)(expr), err)
		p.Errorf("%s", err)
		//try to resynchronize
		if t, errrecov := p.NextSync(); errrecov != nil || t.Type != lex.TokSemi {
			return errrecov
		}
		return p.VarDecls(prog)
	}
	p.dPrintf("value: %s\n", expr)
	svar.Val = expr //HACK, calculated from RVal in constant folding, this is for debugging
	svar.Val.Pos = p.l.Pos()
	svar.DataType = types.UnivType
	svar.DataType.TVal = vartype
	//Decl repeats info in svar.Val, important for debug, reformatting, etc
	decl := tree.NewDecl(svar, svar.Val)
	prog.AddDecl(decl)
	p.matchErr(lex.TokSemi) //on error continue (recovered)
	return p.VarDecls(prog)
}

func IsConnector(con lex.TokType) bool {
	switch con {
	case lex.TokThen, lex.TokNThen, lex.TokComma:
		return true
	}
	return false
}

// RULER :=	ACTION '=>' RULER
//
//	ACTION '!>' RULER
//	ACTION ',' RULER
//	ACTION
func (p *Parser) Ruler(r *tree.Rule, con lex.TokType) (err error) {
	p.pushTrace("Ruler")
	defer p.popTrace(&err)
	tok, err := p.l.Peek()
	if err != nil {
		return err
	}
	if !isExprTok(tok) {
		p.Errorf("no action for rule")
		return errors.New("no action for rule")
	}
	expr, err := p.Expr(-1)
	if err != nil {
		errs := fmt.Sprintf("incorrect expression for action %s...: %s", (*tree.USym)(expr), err)
		//is no operator clear enough?
		p.Errorf(errs)
		//cannot resynch with comma, arg separator, rule connector...
		_, err := p.NextSyncNoSemi(lex.TokThen, lex.TokNThen)
		return err
	}
	action := tree.NewAction(expr, con)
	r.AddAction(action)
	ct, err := p.l.Peek()
	if err != nil {
		return err
	}
	if !IsConnector(ct.Type) {
		ct, err = p.NextSyncNoSemi(lex.TokThen, lex.TokNThen, lex.TokComma)
		if err != nil || !IsConnector(ct.Type) {
			return err
		}
	}
	p.l.Lex()
	return p.Ruler(r, ct.Type)
}

func (p *Parser) declareCurrRule(nrulesects int, nrules int) {
	ruleconstname := "CurrRule"
	rulename := fmt.Sprintf("sect_%5.5d:rule_%5.5d", nrulesects-1, nrules)
	sconst, err := p.Envs.NewConst(ruleconstname, types.StringType.TVal)
	if err != nil {
		p.Errorf("declaring %s: %s", rulename, err)
		sconst = tree.NewAnonSym(tree.SConst) //inject fake const
	}
	sconst.StrVal = rulename
}

// ACTIONDECLS :=	EXPR '?' RULER ';' ACTIONDECLS
//
//	ε
func (p *Parser) ActionDecls(rs *tree.RuleSect, prog *tree.Prog) (err error) {
	p.pushTrace("ActionDecls")
	p.Envs.PushEnv() //for the rule, be careful, pop cannot be deferred (it is recursive)
	defer p.popTrace(&err)

	p.declareCurrRule(len(prog.RuleSects), len(rs.Rules))
	tok, err := p.l.Peek()
	if err != nil {
		p.Envs.PopEnv()
		_, err = p.NextSync()
		return err
	}
	if !isExprTok(tok) {
		p.Envs.PopEnv()
		//ε
		return nil
	}
	expr, err := p.Expr(-1)
	if err != nil {
		p.Envs.PopEnv()
		p.Errorf("incorrect expression %s...: %s", (*tree.USym)(expr), err)
		_, err = p.NextSync(lex.TokQuest)
		return p.ActionDecls(rs, prog) //on error continue (recovered)
	}
	rule := tree.NewRule(p.l.Pos(), expr)
	rs.AddRule(rule)
	if _, istokq := p.matchErr(lex.TokQuest); !istokq {
		p.Envs.PopEnv()
		return p.ActionDecls(rs, prog) //on error continue (recovered)
	}
	err = p.Ruler(rule, lex.TokComma)
	if err != nil {
		p.Errorf(err.Error())
		p.Envs.PopEnv()
		_, err = p.NextSync(lex.TokQuest)
		return err
	}
	p.matchErr(lex.TokSemi) //on error continue (recovered)
	p.Envs.PopEnv()
	return p.ActionDecls(rs, prog)
}

func (p *Parser) NextSection() (t lex.Token, err error) {
	return p.l.LexWhileNot(tokEndSect...)
}

// Exactly like NextSync but does not lex the semicolon
func (p *Parser) NextSyncNoSemi(ts ...lex.TokType) (t lex.Token, err error) {
	ts = append(ts, tokEndSect...)
	ts = append(ts, lex.TokSemi) //add toksemi so it is not lexed (but used to stop)
	t, err = p.l.LexWhileNot(ts...)
	if DebugDesc {
		p.dPrintf("NextSyncNoSemi: resynch on %s\n", t)
	}
	return t, err
}

func (p *Parser) NextSync(ts ...lex.TokType) (t lex.Token, err error) {
	t, err = p.NextSyncNoSemi(ts...)
	if t.Type == lex.TokSemi {
		p.l.Lex()
	}
	if DebugDesc {
		p.dPrintf("NextSync: resynch on %s\n", t)
	}
	return t, err
}

func (p *Parser) Builtins(bs []*tree.Builtin) {
	p.Envs.Builtins(bs)
}

func (p *Parser) PredefVars() {
	p.Envs.PredefVars()
}

func (p *Parser) RuleSect(prog *tree.Prog, tokid lex.Token) (err error) {
	rs, err := p.Envs.NewRuleSect(tokid.Lexema, p.l.Pos())
	if err != nil {
		p.Errorf("%s", err)
	}
	if rs == nil {
		p.Errorf("bad rule")
		_, errrecov := p.NextSection()
		return errrecov
	}
	prog.AddRuleSect(rs)
	err = p.ActionDecls(rs, prog)
	return err
}

// PROG :=	'levels' ':' LEVELDECLS'  PROG
//
//	'vars' ':' VARDECLS PROG
//	RULES PROG
//	ε
func (p *Parser) Prog(prog *tree.Prog) (err error) {
	p.pushTrace("Prog")
	defer p.popTrace(&err)

	tok, err := p.l.Peek()
	if err != nil {
		return err
	}
	istokid := false
	var tokid lex.Token
	switch tok.Type {
	case lex.TokLevels, lex.TokVars, lex.TokConsts:
		p.l.Lex()
	case lex.TokRules:
		p.l.Lex()
		if tokid, _, istokid = p.match(lex.TokId); !istokid {
			p.NextSection() //the whole section is compromised
			return p.Prog(prog)
		}
	case lex.TokEof:
		return nil
	default:
		p.Errorf("found %s at start of section", lex.UTokType(tok.Type))
		t, errrecov := p.NextSection()
		if errrecov != nil {
			return errrecov
		}
		if t.Type == lex.TokEof {
			return nil
		}
		return nil
	}
	if _, istokcolon := p.matchErr(lex.TokColon); !istokcolon {
		return nil
	}
	switch tok.Type {
	case lex.TokLevels:
		err = p.LevelDecls(prog)
		if len(prog.Levels) == 0 {
			p.Errorf("no levels declared")
		}
	case lex.TokConsts:
		err = p.ConstDecls(prog)
	case lex.TokVars:
		err = p.VarDecls(prog)
	case lex.TokRules:
		err = p.RuleSect(prog, tokid)
	case lex.TokEof:
		return nil
	default:
		log.Fatalf("cannot happen, prog without decls")
	}
	if err != nil {
		p.Errorf("error in %s section ", tok.Type)
		t, errrecov := p.NextSection()
		if errrecov != nil {
			return errrecov
		}
		if t.Type == lex.TokEof {
			return nil
		}
		return nil
	}
	return p.Prog(prog)
}

const (
	ErrTooMany = "too many errors"
)

func (p *Parser) Parse() (prog *tree.Prog, err error) {
	defer func() {
		if r := recover(); r != nil {
			errs := fmt.Sprintf("%s", r)
			err = errors.New(errs)
			if errs == ErrTooMany {
				prog = nil
				return
			}
			panic(r)
		}
	}()
	p.pushTrace("Parse")
	p.Envs.PushEnv() //general protection, not popable

	p.Envs.PushEnv() //builtins
	defer p.Envs.PopEnv()
	p.Builtins(tree.Builtins)
	p.PredefVars()
	p.Envs.PushEnv() //user vars
	defer p.Envs.PopEnv()
	prog = tree.NewProg()
	if err = p.Prog(prog); err != nil {
		return nil, err
	}
	prog.Env = p.Envs.CurrEnv() //user vars
	return prog, nil
}

// Potential optimization, if !DebugDesc return and not
//
//	do anything. But this has to be done for any function
//	using depth. Be careful. TODO
func (p *Parser) pushTrace(tag string) {
	if DebugDesc {
		tabs := strings.Repeat("\t", p.depth)
		fmt.Fprintf(os.Stderr, "%s->{%s\n", tabs, tag)
	}
	p.tag = append(p.tag, tag)
	p.depth++
}

func (p *Parser) dPrintf(format string, a ...interface{}) {
	if DebugDesc {
		tabs := strings.Repeat("\t", p.depth)
		format = fmt.Sprintf("%s%s", tabs, format)
		fmt.Fprintf(os.Stderr, format, a...)
	}
}

func (p *Parser) popTrace(e *error) {
	var err error
	err = errors.New("ok")
	if e != nil && *e != nil {
		err = *e
	}
	p.depth--
	if DebugDesc {
		tabs := strings.Repeat("\t", p.depth)
		fmt.Fprintf(os.Stderr, "%s<-}%s:%s\n", tabs, p.tag[len(p.tag)-1], err)
	}
	p.tag = p.tag[0 : len(p.tag)-1]
}

const maxErrors = 10

func (p *Parser) Errorf(s string, v ...interface{}) {
	place := fmt.Sprintf("%s: ", p.l.Pos())
	out := p.l.Errout()
	if lex.DOut {
		out = os.Stderr
	}
	fmt.Fprintf(out, place+s+"\n", v...)
	p.nErr++
	if p.nErr >= maxErrors {
		panic(ErrTooMany)
	}
}

func (p *Parser) Error(e error) {
	p.Errorf("%s", e)
}

func (p *Parser) NErrors() int {
	return p.nErr
}
