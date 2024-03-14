package types

import (
	"fmt"
	"rips/rips/lex"
	"strings"
)

const (
	TVUndef = iota
	TVUniv
	TVInt
	TVFloat
	TVBool
	TVString
	NTypesVal
)

type TypeVal struct {
	Id int
}

var typeValNames = []string{
	TVUndef:  "undef",
	TVUniv:   "univ",
	TVInt:    "int",
	TVFloat:  "float",
	TVBool:   "bool",
	TVString: "string",
}

func (tvp *TypeVal) String() string {
	if tvp == nil || tvp.Id < TVUndef || tvp.Id >= NTypesVal {
		return "unktype"
	}
	return typeValNames[tvp.Id]
}

var TypeVals = []*TypeVal{
	TVUndef:  {TVUndef},
	TVUniv:   {TVUniv},
	TVInt:    {TVInt},
	TVFloat:  {TVFloat},
	TVBool:   {TVBool},
	TVString: {TVString},
}
var TypeValsFromNames = map[string]*TypeVal{
	"int":    TypeVals[TVInt],
	"float":  TypeVals[TVFloat],
	"bool":   TypeVals[TVBool],
	"string": TypeVals[TVString],
}

type TypeExpr struct {
	Id int
}

const (
	TEUndef = iota
	TEExpr
	TEExternal
	TEMsg
	TEGraph
	TEMsgGraph
	NTypesExpr
)

var typeExprNames = []string{
	TEUndef:    "eundef",
	TEExpr:     "eexpr",
	TEMsg:      "emsg",
	TEGraph:    "egraph",
	TEMsgGraph: "emsggraph",
	TEExternal: "eexternal",
}

func (tep *TypeExpr) String() string {
	if tep == nil || tep.Id < TVUndef || tep.Id >= NTypesExpr {
		return "unktype"
	}
	return typeExprNames[tep.Id]
}

var TypeExprs = []*TypeExpr{
	TEUndef:    {TEUndef},
	TEExpr:     {TEExpr}, //universal, only constants
	TEMsg:      {TEMsg},
	TEGraph:    {TEGraph},
	TEMsgGraph: {TEMsgGraph},
	TEExternal: {TEExternal},
}

var TypeExprFromNames = map[string]*TypeExpr{
	"Msg":      TypeExprs[TEMsg],
	"Graph":    TypeExprs[TEGraph],
	"MsgGraph": TypeExprs[TEMsgGraph],
	"External": TypeExprs[TEExternal],
}

type Type struct {
	TVal  *TypeVal
	TExpr *TypeExpr
}

// Types are compatible if they are of a compatible type
//
//	compatible type of TExpr.
func (tvp *TypeVal) IsTypeValCompat(tvp2 *TypeVal) bool {
	if tvp == TypeVals[TVUndef] || tvp2 == TypeVals[TVUndef] {
		return true
	}
	if tvp == TypeVals[TVUniv] || tvp2 == TypeVals[TVUniv] {
		return true
	}
	return tvp.Id == tvp2.Id
}

func (tep *TypeExpr) IsTypeExprCompatSpecial(tep2 *TypeExpr) bool {
	if tep == TypeExprs[TEMsgGraph] && tep2 == TypeExprs[TEMsg] {
		return true
	}
	if tep == TypeExprs[TEMsgGraph] && tep2 == TypeExprs[TEGraph] {
		return true
	}
	return false
}
func (tep *TypeExpr) IsTypeExprCompat(tep2 *TypeExpr) bool {
	if tep == TypeExprs[TEUndef] || tep2 == TypeExprs[TEUndef] {
		return true
	}
	//Expr is compatible with anything
	if tep == TypeExprs[TEExpr] || tep2 == TypeExprs[TEExpr] {
		return true
	}
	if tep.IsTypeExprCompatSpecial(tep2) || tep2.IsTypeExprCompatSpecial(tep) {
		return true
	}
	return tep.Id == tep2.Id
}

func (tp Type) IsTypeCompat(tp2 Type) bool {
	tvp, tep := tp.TVal, tp.TExpr
	tvp2, tep2 := tp2.TVal, tp2.TExpr

	return tvp.IsTypeValCompat(tvp2) && tep.IsTypeExprCompat(tep2)
}

func (tp Type) String() string {
	return fmt.Sprintf("(%s, %s)", tp.TVal, tp.TExpr)
}

func isCompareOp(op int) bool {
	switch lex.TokType(op) {
	case lex.TokEq, lex.TokG, lex.TokL, lex.TokNEq, lex.TokGEq, lex.TokLEq:
		return true
	}
	return false
}

func (tvp *TypeVal) IsCompatOp(op int) bool {
	switch tvp {
	case TypeVals[TVInt]:
		if strings.ContainsRune("+-*/%&|~^", rune(op)) {
			return true
		}
		return isCompareOp(op) //two chars...
	case TypeVals[TVFloat]:
		if strings.ContainsRune("+-*/", rune(op)) {
			return true
		}
		return isCompareOp(op) //two chars...
	case TypeVals[TVBool]:
		//should bools be ordered?
		if isCompareOp(op) { //two chars..
			return true
		}
		switch lex.TokType(op) {
		case lex.TokLogAnd, lex.TokLogOr, lex.TokLogNeg:
			return true
		default:
			return false
		}
	case TypeVals[TVString]: // > (lexicographically)
		if strings.ContainsRune("+", rune(op)) { //concatenation
			return true
		}
		return isCompareOp(op) //two chars...
	case TypeVals[TVUniv]:
		return true
	case TypeVals[TVUndef]:
		return true
	default:
		return false
	}
}

func (tp Type) IsCompat(tp2 *Type, op int) bool {
	tvp := tp.TVal
	opC := tvp.IsCompatOp(op)
	if tp2 == nil { //unary operations
		return opC
	}
	return tp.IsTypeCompat(*tp2) && opC
}

func (tp Type) IsTypeUndef() bool {
	return tp.TVal == TypeVals[TVUndef] || tp.TExpr == TypeExprs[TEUndef]
}

// would be constants if it was possible in go
var FloatType = Type{TypeVals[TVFloat], TypeExprs[TEExpr]}
var IntType = Type{TypeVals[TVInt], TypeExprs[TEExpr]}
var StringType = Type{TypeVals[TVString], TypeExprs[TEExpr]}
var BoolType = Type{TypeVals[TVBool], TypeExprs[TEExpr]}
var UnivType = Type{TypeVals[TVUniv], TypeExprs[TEExpr]}
var UndefType = Type{TypeVals[TVUndef], TypeExprs[TEUndef]}
var UndefExprType = Type{TypeVals[TVUndef], TypeExprs[TEExpr]}
var MsgGraphType = Type{TypeVals[TVUniv], TypeExprs[TEMsgGraph]}
var MsgType = Type{TypeVals[TVUniv], TypeExprs[TEMsg]}
var GraphType = Type{TypeVals[TVUniv], TypeExprs[TEGraph]}

var MsgStrType = Type{TypeVals[TVString], TypeExprs[TEMsg]}
var MsgIntType = Type{TypeVals[TVInt], TypeExprs[TEMsg]}

var GraphStrType = Type{TypeVals[TVString], TypeExprs[TEGraph]}
var GraphIntType = Type{TypeVals[TVInt], TypeExprs[TEGraph]}

var ExternalStrType = Type{TypeVals[TVString], TypeExprs[TEExternal]}
