package tree_test

import (
	"fmt"
	"rips/rips/tree"
	"rips/rips/types"
	"testing"
)

// TODO more testing of sym table
func TestGen(t *testing.T) {
	var s tree.StkEnv
	s.PushEnv()
	sym, _ := s.NewSym("aaaa", tree.SVar)
	ud := types.TypeExprs[types.TVUndef]
	sym.DataType = types.Type{TExpr: ud, TVal: types.TypeVals[types.TVFloat]}
	s.PushEnv()

	s.NewSym("bbbb", tree.SVar)
	s.NewSym("aaaa", tree.SVar)

	sym = s.GetSym("aaaa")
	fmt.Println(sym)
	s.PopEnv()
	sym = s.GetSym("aaaa")
	fmt.Println(sym)
	fmt.Println(s)
}
