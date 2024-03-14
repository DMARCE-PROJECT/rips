package xrips

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"rips/rips/extern"
	"rips/rips/lex"
	"rips/rips/parser"
	"rips/rips/stats"
	"rips/rips/tree"
)

type Rips struct {
	ProgramFile *bufio.Reader
	Parser      *parser.Parser
	Lexer       *lex.Lexer
	Context     *extern.Ctx
	ExecEnv     *tree.StkEnv
	Program     *tree.Prog
	DebLevel    int
}

func NewRips(progfname string, progfile io.Reader, deblevel int, out io.Writer) (r *Rips) {
	r = &Rips{}
	bufrd := bufio.NewReader(progfile)
	l, err := lex.NewLexerRd(bufrd, progfname, out)
	if err != nil {
		log.Fatal(err)
	}
	r.Lexer = l
	r.ProgramFile = bufrd
	r.Parser = parser.NewParser(l)
	r.DebLevel = deblevel
	return r
}

func (r *Rips) BuildAst(xstats *stats.Stats) (nerr int, err error) {
	xstats.Start(stats.Compiling)
	defer xstats.End(stats.Compiling)
	prog, err := r.Parser.Parse()
	if err != nil {
		nerr = r.Parser.NErrors()
		return nerr, err
	}
	r.Program = prog
	if r.DebLevel > 2 {
		fmt.Fprintf(os.Stderr, "############Before typing###########\n%s", prog)
		fmt.Fprintf(os.Stderr, "##################################\n")
	}
	if nerr = r.Parser.NErrors(); nerr > 0 {
		s := fmt.Sprintf("There were parsing/lexing errors")
		return nerr, errors.New(s)
	}
	nerr += r.Program.TypeCheck(r.Lexer.Errout())
	if nerr > 0 {
		s := fmt.Sprintf("There were type errors")
		return nerr, errors.New(s)
	}
	if r.DebLevel > 1 {
		fmt.Fprintf(os.Stderr, "############Before folding###########\n%s", prog)
		fmt.Fprintf(os.Stderr, "##################################\n")
	}
	nerr += r.Program.Fold(r.Lexer.Errout())
	if nerr > 0 {
		s := fmt.Sprintf("There were optimization/constant evaluation errors")
		return nerr, errors.New(s)
	}
	nerr += r.Program.StatesCheck(r.Lexer.Errout())
	if nerr > 0 {
		s := fmt.Sprintf("There were state machine errors")
		return nerr, errors.New(s)
	}
	if r.DebLevel > 0 {
		fmt.Fprintf(os.Stderr, "%s", r.Program)
	}
	return nerr, nil
}

func CheckLevelScripts(p *tree.Prog, context *extern.Ctx) (nerr int) {
	for _, ls := range p.Levels {
		fromname := fmt.Sprintf(extern.LevelFromFmt, ls.Name)
		toname := fmt.Sprintf(extern.LevelToFmt, ls.Name)
		spathfrom := context.ScriptsPath + "/" + fromname
		spathto := context.ScriptsPath + "/" + toname
		if !extern.IsExecutable(spathfrom) {
			cdir := extern.CurrDir()
			fmtstr := "trigger: program %s is not executable from %s\n"
			context.Printf(fmtstr, spathfrom, cdir)
			nerr++
		}
		if !extern.IsExecutable(spathto) {
			cdir := extern.CurrDir()
			fmtstr := "trigger: program %s is not executable from %s\n"
			context.Printf(fmtstr, spathto, cdir)
			nerr++
		}
	}
	return nerr
}
