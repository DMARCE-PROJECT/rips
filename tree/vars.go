package tree

import (
	"errors"
	"fmt"
	"rips/rips/extern"
	"rips/rips/types"
	"time"
)

func (envs *StkEnv) NewIntVar(name string, intval int64) (s *Sym, err error) {
	s, err = envs.NewVar(name, types.TypeVals[types.TVInt])
	if err != nil {
		return nil, fmt.Errorf("cannot declare predefined %s", name)
	}
	val := NewAnonSym(SConst)
	val.DataType = types.IntType
	val.IntVal = intval
	s.Val = val
	return s, nil
}

// Create predefined variables for parser
func (envs *StkEnv) PredefVars() {
	s, err := envs.NewIntVar("CurrLevel", -1)
	if err != nil {
		panic(err)
	}
	s.Val.SType = SLevel
	s.IsBuiltin = true
	s.IsSet = true
	s.IsUsed = true
	s, err = envs.NewIntVar("Time", 0)
	if err != nil {
		panic(err)
	}
	s.IsBuiltin = true
	s.IsSet = true
	s.IsUsed = true
	s, err = envs.NewIntVar("Uptime", 0)
	if err != nil {
		panic(err)
	}
	s.IsBuiltin = true
	s.IsSet = true
	s.IsUsed = true
}

// Create vars. CurrLevel < 0 means first time initialization
func (execEnvs *StkEnv) SetPredefVars(p *Prog, context *extern.Ctx) (err error) {
	isinit := false
	if context.CurrLevel < 0 {
		isinit = true
	}
	s := execEnvs.GetSym("CurrLevel")
	if s == nil {
		return errors.New("cannot find CurrLevel")
	}
	s.Val.IntVal = context.CurrLevel
	if isinit {
		s.Val = p.Levels[0]
		if s.Val == nil {
			panic("no levels")
		}
		context.CurrLevel = p.Levels[0].IntVal
	}
	now := time.Now()
	s = execEnvs.GetSym("Time")
	if s == nil {
		return errors.New("cannot find Time")
	}
	s.Val.IntVal = now.UnixNano()
	s = execEnvs.GetSym("Uptime")
	if s == nil {
		return errors.New("cannot find Uptime")
	}
	if isinit {
		context.TimeStarted = now
	}
	d := now.Sub(context.TimeStarted)
	s.Val.IntVal = d.Nanoseconds()

	return nil
}
