package parser

import (
	"errors"
	"fmt"
	"rips/rips/lex"
	"rips/rips/tree"
	"rips/rips/types"
)

/*
 * 	An interpretation of Pratt parsing
 *	https://web.archive.org/web/20151223215421/http://hall.org.ua/halls/wizzard/pdf/Vaughan.Pratt.TDOP.pdf
 *	Proceedings of the 1st Annual ACM SIGACT-SIGPLAN
 *	Symposium on Principles of Programming Languages (1973)
 *	The big difference with the original Pratt parser is that it is table driven instead of
 *	using a function per nud. This makes it shorter and (hopefully) more understandable.
 */

const MaxNTerms = 1000

var precTab = map[lex.TokType]int{
	')':           1,
	lex.TokLogOr:  4 * MaxNTerms,
	lex.TokLogAnd: 5 * MaxNTerms,
	'&':           6 * MaxNTerms,
	'^':           7 * MaxNTerms,
	'|':           8 * MaxNTerms,
	lex.TokEq:     9 * MaxNTerms,
	lex.TokNEq:    9 * MaxNTerms,
	lex.TokGEq:    10 * MaxNTerms,
	lex.TokLEq:    10 * MaxNTerms,
	lex.TokG:      10 * MaxNTerms,
	lex.TokL:      10 * MaxNTerms,
	'+':           13 * MaxNTerms,
	'-':           13 * MaxNTerms,
	'*':           14 * MaxNTerms,
	'/':           14 * MaxNTerms,
	'%':           14 * MaxNTerms,
	'(':           18 * MaxNTerms,
}

// operators which appear here (are Unary) and be binary
var precTabUnary = map[lex.TokType]int{
	'+': 16 * MaxNTerms,
	'-': 16 * MaxNTerms,
	'~': 17 * MaxNTerms,
	'!': 17 * MaxNTerms,
	'(': 18 * MaxNTerms,
}

// There is no right associative is there?? TODO
var rightTab = map[rune]bool{
	// '???': true,
}
var unaryTab = map[rune]bool{
	'~': true,
	'+': true,
	'-': true,
	'!': true,
	'(': true,
}

var exprTokTab = map[lex.TokType]bool{
	lex.TokLPar: true,
	lex.TokRPar: true,
	//op
	lex.TokMod:    true,
	lex.TokAdd:    true,
	lex.TokMin:    true,
	lex.TokMul:    true,
	lex.TokDiv:    true,
	lex.TokXor:    true,
	lex.TokBitAnd: true,
	lex.TokBitOr:  true,
	lex.TokAsig:   true,
	lex.TokGEq:    true,
	lex.TokLEq:    true,
	lex.TokEq:     true,
	lex.TokNEq:    true,
	lex.TokG:      true,
	lex.TokL:      true,
	lex.TokLogNeg: true,
	lex.TokLogAnd: true,
	lex.TokLogOr:  true,
	lex.TokCompl:  true,
	//vals
	lex.TokFloatVal: true,
	lex.TokIntVal:   true,
	lex.TokStrVal:   true,
	lex.TokBoolVal:  true,
	lex.TokId:       true,
}

func isExprTok(tok lex.Token) (isexpr bool) {
	isexpr, _ = exprTokTab[tok.Type]
	return isexpr
}

// helper for Nud
// ParenExpr := '(' Expr ')' | empty
func (p *Parser) parenExpr(tok lex.Token) (expr *tree.Sym, err error) {
	var rbp int
	if tok.Type != lex.TokLPar { //empty
		return expr, nil
	}
	//'('Expr ')'
	expr, err = p.Expr(rbp)
	if err != nil {
		return expr, err
	}
	if _, err, isclosed := p.match(lex.TokRPar); err != nil {
		return expr, err
	} else if !isclosed {
		return expr, errors.New("unmatched parenthesis")
	}
	return expr, nil
}

// helper for Nud
// Args :== Arg, Args	| empty
func (p *Parser) funcArgs(expr *tree.Sym) (err error) {
	p.pushTrace("Args")
	defer p.popTrace(&err)
	tok, err := p.l.Peek()
	if err != nil {
		return err
	}
	if tok.Type == lex.TokRPar || !isExprTok(tok) {
		return nil
	}
	param, err := p.Expr(defRbp - 1)
	if err != nil {
		return fmt.Errorf("%s at parameter %s", err, (*tree.USym)(param))
	}
	if param != nil {
		expr.AddArg(param)
	}
	_, err, iscomma := p.match(lex.TokComma)
	if err != nil {
		return fmt.Errorf("%s in parameter list", err)
	}
	if !iscomma {
		return nil
	}
	if param == nil {
		return errors.New("empty parameter")
	}
	p.dPrintf("params: next parameter\n")

	err = p.funcArgs(expr)
	return err
}

func (p *Parser) SetExprVal(expr *tree.Sym, tok lex.Token, pos lex.Position) (svar *tree.Sym) {
	svar = expr
	expr.DataType.TExpr = types.TypeExprs[types.TEExpr]
	expr.DataType.TVal = types.TypeVals[types.TVUndef]
	switch tok.Type {
	case lex.TokFloatVal:
		expr.DataType.TVal = types.TypeVals[types.TVFloat]
		expr.FloatVal = tok.TokFloatVal
	case lex.TokIntVal:
		expr.DataType.TVal = types.TypeVals[types.TVInt]
		expr.IntVal = tok.TokIntVal
	case lex.TokStrVal:
		expr.DataType.TVal = types.TypeVals[types.TVString]
		expr.StrVal = tok.TokStrVal
	case lex.TokBoolVal:
		expr.DataType.TVal = types.TypeVals[types.TVBool]
		expr.BoolVal = tok.TokBoolVal
	case lex.TokId:
		expr.SType = tree.SVar
		expr.Name = tok.Lexema
		svar = p.Envs.GetSym(tok.Lexema)
		if svar == nil {
			p.Errorf("undeclared symbol '%s'", expr.Name)
			svar = expr
			//declare to prevent cascade of errors
			p.Envs.NewSym(tok.Lexema, tree.SNone)
			break
		}
		if svar.SType == tree.SFunc {
			expr.IsAction = svar.IsAction
			expr.SType = tree.SFCall //params will come later
			expr.Expr.FCall = svar   // the function we found
			svar = expr
		}
	default:
		//if not, is a binary/unary operation
		expr.Expr.Op = int(tok.Type)
	}
	expr.Pos = pos
	return svar
}

// No left context, null-denotation: nud
func (p *Parser) Nud(tok lex.Token) (expr *tree.Sym, err error) {
	var right *tree.Sym
	var rbp int
	p.pushTrace("Nud:")
	defer p.popTrace(&err)
	p.dPrintf("nud:  %d, %s \n", bindPow(tok, false), tok)
	expr, err = p.parenExpr(tok)
	if expr != nil || err != nil {
		p.dPrintf("nud:  parenthesis\n")
		return expr, err
	}
	expr = tree.NewExpr(nil, nil)
	expr = p.SetExprVal(expr, tok, p.l.Pos())
	rbp = bindPow(tok, false)
	rtok := rune(tok.Type)
	if rbp != defRbp { //regular unary operators
		if !unaryTab[rtok] {
			errs := fmt.Sprintf("%s  is not unary", tok.Type)
			return expr, errors.New(errs)
		}
		rbp = bindPow(tok, true)
		p.dPrintf("nud: unary %d, %s \n", bindPow(tok, false), tok)
		right, err = p.Expr(rbp)
		if right == nil || err != nil {
			return expr, errors.New("unary operator without operand")
		}
		expr.AddRight(right)
	}
	_, err, islpar := p.match(lex.TokLPar)
	if tok.Type != lex.TokId && islpar {
		return expr, fmt.Errorf("cannot call a %s", lex.UTokType(tok.Type))
	}
	if tok.Type != lex.TokId || !islpar || err != nil {
		//is svar
		return expr, err
	}
	//is fcall
	expr.SType = tree.SFCall
	err = p.funcArgs(expr)
	if err != nil {
		return expr, err
	}
	_, err, isclosed := p.match(lex.TokRPar)
	if err != nil {
		return expr, err
	}
	if !isclosed {
		return expr, errors.New("missing ')' at end of params")
	}
	return expr, nil
}

// left context, left-denotation: led
func (p *Parser) Led(left *tree.Sym, tok lex.Token) (expr *tree.Sym, err error) {
	var rbp int
	p.pushTrace("Led:")
	defer p.popTrace(&err)
	expr = tree.NewExpr(nil, nil)
	expr = p.SetExprVal(expr, tok, p.l.Pos())
	expr.AddLeft(left)
	rbp = bindPow(tok, false)
	if isright := rightTab[rune(tok.Type)]; isright {
		rbp -= 1
		if rbp > MaxNTerms && (rbp%MaxNTerms) == 0 {
			ef := "expression too big, greater than %d terms without parenthesis\n"
			errs := fmt.Sprintf(ef, MaxNTerms)
			return expr, errors.New(errs)
		}
	}
	p.dPrintf("led: %d, {{%s}} %s \n", rbp, left, tok)
	right, err := p.Expr(rbp)
	if err != nil {
		return expr, err
	}
	if right == nil {
		errs := fmt.Sprintf("missing operand for %s\n", lex.UTokType(tok.Type))
		return expr, errors.New(errs)
	}
	expr.AddRight(right)
	return expr, nil
}

const defRbp = 0

func bindPow(tok lex.Token, isunary bool) int {
	if rbp, ok := precTabUnary[tok.Type]; isunary && ok {
		return rbp
	}
	if rbp, ok := precTab[tok.Type]; ok {
		return rbp
	}
	//if it is unary and there is no eq binary, return unary
	if rbp, ok := precTabUnary[tok.Type]; ok {
		return rbp
	}
	return defRbp
}

func (p *Parser) Expr(rbp int) (expr *tree.Sym, err error) {
	var left *tree.Sym

	defer func() {
		if expr == nil {
			expr = tree.NewAnonSym(tree.SNone)
			if err == nil {
				err = errors.New("bad expression")
			}
		}
	}()

	s := fmt.Sprintf("Expr: %d", rbp)
	p.pushTrace(s)
	defer p.popTrace(&err)

	tok, err := p.l.Peek()
	if err != nil {
		return expr, err
	}
	p.dPrintf("expr: nud Lex: %s\n", tok)

	if !isExprTok(tok) {
		return expr, nil
	}
	p.l.Lex() //already peeked
	if left, err = p.Nud(tok); err != nil {
		return left, err
	}
	if left == nil {
		return expr, errors.New("no left operator in expression")
	}
	expr = left
	for {
		tok, err := p.l.Peek()
		if err != nil {
			return expr, err
		}

		if !isExprTok(tok) || tok.Type == lex.TokRPar {
			return expr, nil
		}
		newrbp := bindPow(tok, false)
		if newrbp == defRbp {
			return expr, errors.New("no operator")
		}
		if newrbp <= rbp {
			p.dPrintf("Not enough binding: %d <= %d, %s\n",
				bindPow(tok, false), rbp, tok)
			return expr, nil
		}
		p.l.Lex() //already peeked
		p.dPrintf("expr: led Lex: %s", tok)
		if left, err = p.Led(left, tok); err != nil {
			return expr, err
		}
		expr = left
	}
}
