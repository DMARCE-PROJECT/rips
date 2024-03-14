package tree

import (
	"io"
)

func (p *Prog) StatesCheck(errout io.Writer) (nerr int) {
	if len(p.Levels) == 0 {
		return //should we err?
	}
	for _, ls := range p.Levels {
		if !ls.IsReach {
			ls.Errorf(errout, 0, "level %s not reachable", ls.Name)
			nerr++
		}
	}
	return nerr
}
