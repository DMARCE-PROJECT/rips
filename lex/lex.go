package lex

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode"
)

const (
	DFlex = false //low level lex debug
	DToks = false //print tokens
	//for parser and tree, to set errors here for imports...
	DOut = false
	DGet = false //underlying get unget calls
)

type RuneScanner interface {
	ReadRune() (r rune, size int, err error)
	UnreadRune() error
}

type Position struct {
	File string
	Line int
}

type LexPeeker interface {
	Peek() (t Token, err error)
	Lex() (t Token, err error)
	LexWhileNot(tps ...TokType) (t Token, err error)
	Pos() Position
	Errout() io.Writer
}

type Lexer struct {
	pos      Position
	r        RuneScanner
	lastrune rune
	errout   io.Writer //for errors (used mainly by the parser)

	accepted []rune
	tokSaved *Token
}

type TokType rune

const (
	TokBad = TokType(0) //this zero so uninitialized is bad
	//	TokEol  = TokType('\n')	//no EOL token!!
	TokQuest    = TokType('?')
	TokSemi     = TokType(';')
	TokColon    = TokType(':')
	TokLPar     = TokType('(')
	TokRPar     = TokType(')')
	TokMod      = TokType('%')
	TokAdd      = TokType('+')
	TokMin      = TokType('-')
	TokMul      = TokType('*')
	TokDiv      = TokType('/')
	TokXor      = TokType('^')
	TokBitAnd   = TokType('&')
	TokBitOr    = TokType('|')
	TokAsig     = TokType('=')
	TokG        = TokType('>')
	TokL        = TokType('<')
	TokComma    = TokType(',')
	TokLogNeg   = TokType('!')
	TokCompl    = TokType('~')
	TokFloatVal = TokType(unicode.MaxRune + 1 + iota)
	TokIntVal
	TokStrVal
	TokBoolVal
	TokId
	TokLogAnd // &&
	TokLogOr  // ||
	TokEq     // ==
	TokNEq    // !=
	TokGEq    // >=
	TokLEq    // <=
	TokThen   // =>
	TokNThen  // !>
	TokLevels
	TokSoft
	TokVars
	TokConsts
	TokRules
	TokEof
	RuneEof = -1
)

type Token struct {
	Lexema      string
	Type        TokType
	TokFloatVal float64
	TokIntVal   int64
	TokStrVal   string
	TokBoolVal  bool
}

func TypeName(tT TokType) string {
	switch tT {
	case TokMod:
		return "TokMod"
	case TokComma:
		return ","
	case TokBoolVal:
		return "BoolVal"
	case TokFloatVal:
		return "FloatVal"
	case TokIntVal:
		return "IntVal"
	case TokStrVal:
		return "StrVal"
	case TokId:
		return "TokId"
	case TokEof:
		return "TokEof"
		//	case TokEol:
		//		return "TokEol"
	case TokQuest:
		return "TokQuest"
	case TokSemi:
		return "TokSemi"
	case TokColon:
		return "TokColon"
	case TokLPar:
		return "TokLPar"
	case TokRPar:
		return "TokRPar"
	case TokAdd:
		return "TokAdd"
	case TokMin:
		return "TokMin"
	case TokMul:
		return "TokMul"
	case TokDiv:
		return "TokDiv"
	case TokCompl:
		return "TokCompl"
	case TokXor:
		return "TokXor"
	case TokBitAnd:
		return "TokBitAnd"
	case TokBitOr:
		return "TokBitOr"
	case TokLogAnd:
		return "TokLogAnd"
	case TokLogOr:
		return "TokLogOr"
	case TokEq:
		return "TokEq"
	case TokNEq:
		return "TokNEq"
	case TokAsig:
		return "TokAsig"
	case TokG:
		return "TokG"
	case TokL:
		return "TokL"
	case TokGEq:
		return "TokGEq"
	case TokThen:
		return "TokThen"
	case TokNThen:
		return "TokNThen"
	case TokLEq:
		return "TokLEq"
	case TokLogNeg:
		return "TokLogNeg"
	case TokBad:
		return "TokBad"
	case TokLevels:
		return "TokLevels"
	case TokSoft:
		return "TokSoft"
	case TokVars:
		return "TokVars"
	case TokRules:
		return "TokRules"
	default:
		return "TokUnk"
	}
}

var keywords = map[string]Token{
	"true":   {Type: TokBoolVal, TokBoolVal: true},
	"false":  {Type: TokBoolVal, TokBoolVal: false},
	"levels": {Type: TokLevels},
	"soft":   {Type: TokSoft},
	"vars":   {Type: TokVars},
	"consts": {Type: TokConsts},
	"rules":  {Type: TokRules},
}

func (l *Lexer) Pos() Position {
	return l.pos
}
func (l *Lexer) Errout() io.Writer {
	return l.errout
}

func (pos Position) String() string {
	return fmt.Sprintf("%s:%d", pos.File, pos.Line)
}
func (l *Lexer) String() string {
	return fmt.Sprintf("%s:%d", l.pos.File, l.pos.Line)
}

func (t TokType) String() string {
	return TypeName(t) + fmt.Sprintf("[%d]", t)
}

func (t Token) String() string {
	s := fmt.Sprintf("['%s': %s", t.Lexema, t.Type)
	switch t.Type {
	case TokBoolVal:
		s += fmt.Sprintf(" %v", t.TokBoolVal)
	case TokIntVal:
		s += fmt.Sprintf(" %d", t.TokIntVal)
	case TokFloatVal:
		s += fmt.Sprintf(" %f", t.TokFloatVal)
	case TokStrVal:
		s += fmt.Sprintf(" %s", t.TokStrVal)
	}

	s += "]"
	return s
}

func (l *Lexer) get() (r rune) {
	var err error
	r, _, err = l.r.ReadRune()
	if err == nil {
		if DGet {
			fmt.Fprintf(os.Stderr, "Lex: get: %c\n", r)
		}
		l.lastrune = r
		if r == '\n' {
			l.pos.Line++
		}
	}
	if err == io.EOF {
		l.lastrune = RuneEof
		return RuneEof
	}
	if err != nil {
		panic(err)
	}
	l.accepted = append(l.accepted, r)
	return r
}

func (l *Lexer) unget() {
	var err error
	if l.lastrune == RuneEof {
		return
	}
	if DGet {
		fmt.Fprintf(os.Stderr, "Lex: unget: %c\n", l.lastrune)
	}
	err = l.r.UnreadRune()
	if err == nil && l.lastrune == '\n' {
		l.pos.Line--
	}
	l.lastrune = unicode.ReplacementChar
	if len(l.accepted) != 0 {
		l.accepted = l.accepted[0 : len(l.accepted)-1]
	}
	if err != nil {
		panic(err)
	}
}

func NewLexerRd(r RuneScanner, fname string, errout io.Writer) (l *Lexer, err error) {
	var pos Position
	pos.Line = 1
	l = &Lexer{pos: pos, errout: errout}
	l.pos.File = fname
	l.r = r
	return l, nil
}

func NewLexer(fname string, errout io.Writer) (l *Lexer, err error) {
	if DToks {
		fmt.Fprintf(os.Stderr, "newLex\n")
	}
	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	r := bufio.NewReader(f)
	return NewLexerRd(r, fname, errout)
}

func NewFakeLexer(fcontent string, errout io.Writer) (l *Lexer, err error) {
	r := strings.NewReader(fcontent)
	return NewLexerRd(r, "/fakefile", errout)
}

func (l *Lexer) accept() (tok string) {
	tok = string(l.accepted)
	if tok == "" && l.lastrune != RuneEof {
		panic(errors.New("empty token"))
	}
	l.accepted = nil
	return tok
}

func (l *Lexer) lexId() (t Token, err error) {
	r := l.get()
	if !unicode.IsLetter(r) && r != '_' {
		return t, errors.New("bad Id, should not happen")
	}
	isAlpha := func(ar rune) bool {
		return unicode.IsDigit(ar) || unicode.IsLetter(ar) || r == '_'
	}
	for r = l.get(); isAlpha(r); r = l.get() {
	}
	l.unget()
	t.Type = TokId
	t.Lexema = l.accept()
	return t, nil
}

func (l *Lexer) quote(q rune, r rune) {
	l.accepted[len(l.accepted)-1] = q
	l.accepted = append(l.accepted, r)
}

func (l *Lexer) lexString() (t Token, err error) {
	r := l.get()
	if r != '"' {
		return t, errors.New("bad string, should not happen")
	}
	isStrDigit := func(ar rune) bool {
		return unicode.IsGraphic(ar) && ar != '"' || ar == '\n'
	}
	for r = l.get(); isStrDigit(r); r = l.get() {
		switch r {
		case '\n':
			l.quote('\\', 'n')
		case '\\':
			r = l.get()
			if r == '"' || isStrDigit(r) {
				continue
			}
			e := fmt.Errorf("bad \\%c [%s]", r, t.Lexema)
			return t, e
		}
	}
	l.unget()
	if r == '\n' {
		return t, errors.New("found \n looking for str [" + t.Lexema + "]")
	}
	if r != '"' {
		return t, errors.New("unterminated str [" + t.Lexema + "]")
	}
	l.get() //already seen it is "
	t.Type = TokStrVal
	runes := l.accepted
	t.Lexema = l.accept()
	s, err := strconv.Unquote(string(runes))
	if err != nil {
		return t, fmt.Errorf("badly quoted str [%s]: %s", t.Lexema, err)
	}
	sx := strings.Replace(s, "%", "%%", -1) //quote % for the next sprintf
	t.TokStrVal = fmt.Sprintf(sx)           //do \t and so on
	return t, nil
}

func isdigitBased(r rune, tnum rune) bool {
	isd := unicode.IsDigit(r)
	switch tnum {
	case '0':
		return isd
	case 'x':
		return isd || (r >= 'a' && r <= 'f')
	case 'o':
		return r >= '0' && r <= '7'
	case 'b':
		return r == '0' || r == '1'
	}
	return false
}

func (l *Lexer) lexNum() (t Token, err error) {
	const (
		Es    = "Ee"
		Signs = "+-"
	)
	hasDot := false
	isquote := false
	tnum := '0'
	r := l.get()
	if r == '.' {
		hasDot = true
		r = l.get()
	} else if r == '0' {
		r = l.get()
		if r == 'x' || r == 'b' || r == 'o' {
			tnum = r
			isquote = true
			r = l.get()
		} else {
			l.unget()
		}
	}
	for r = l.get(); isdigitBased(r, tnum); r = l.get() {
	}
	if r == '.' {
		if hasDot || isquote {
			return t, errors.New("bad float [" + l.accept() + "]")
		}
		hasDot = true
		for r = l.get(); unicode.IsDigit(r); r = l.get() {
		}
	}
	switch {
	case strings.ContainsRune(Es, r):
		r = l.get()
		if strings.ContainsRune(Signs, r) {
			r = l.get()
		}
	case hasDot:
		l.unget()
		break
	case !hasDot: //may be an int
		l.unget()
		if isquote {
			t.Lexema = l.accept()
			t.TokIntVal, err = strconv.ParseInt(t.Lexema, 0, 64)
		} else {
			t.Lexema = l.accept()
			t.TokIntVal, err = strconv.ParseInt(t.Lexema, 10, 64)
		}
		if err != nil {
			return t, errors.New("bad int [" + t.Lexema + "]")
		}
		t.Type = TokIntVal
		return t, nil
	default:
		return t, errors.New("bad float [" + l.accept() + "]")
	}
	if isquote {
		return t, errors.New("bad float with hex [" + l.accept() + "]")
	}
	for r = l.get(); unicode.IsDigit(r); r = l.get() {
	}
	l.unget()
	t.Lexema = l.accept()
	t.TokFloatVal, err = strconv.ParseFloat(t.Lexema, 64)
	if err != nil {
		return t, errors.New("bad float [" + t.Lexema + "]")
	}
	t.Type = TokFloatVal
	return t, nil
}

func (l *Lexer) Peek() (t Token, err error) {
	t, err = l.Lex()
	if err == nil {
		l.tokSaved = &t
	}
	if DToks {
		fmt.Fprintf(os.Stderr, "Peek: %s\n", t)
	}
	return t, err
}

func (l *Lexer) eatComment() {
	// comment, eat until eol included
	// do not eat eof
	for rx := l.get(); ; rx = l.get() {
		if rx == RuneEof {
			l.unget()
			break
		}
		l.accept()
		if rx == '\n' {
			break
		}
	}
}

func (l *Lexer) mayLexDouble(t *Token) (err error) {
	l.get()
	switch string(l.accepted) {
	case "!>":
		t.Type = TokNThen
	case "==":
		t.Type = TokEq
	case "!=":
		t.Type = TokNEq
	case ">=":
		t.Type = TokGEq
	case "<=":
		t.Type = TokLEq
	case "=>":
		t.Type = TokThen
	case "&&":
		t.Type = TokLogAnd
	case "||":
		t.Type = TokLogOr
	default:
		l.unget()
	}
	return nil
}

const (
	arrowThen  = '→'
	arrowNThen = '↛'
)

var synomTok = map[rune]Token{
	arrowThen:  {Lexema: "=>", Type: TokThen},
	arrowNThen: {Lexema: "!>", Type: TokNThen},
}

func (l *Lexer) takeSaved() (t Token, ok bool) {
	if l.tokSaved == nil {
		if DFlex {
			fmt.Fprintf(os.Stderr, "when Lex not saved\n")
		}
		return t, false
	}
	if DFlex {
		fmt.Fprintf(os.Stderr, "when Lex saved: %s\n", t)
	}
	t = *l.tokSaved
	l.tokSaved = nil
	return t, true
}

func keywordLookup(t Token) (nt Token) {
	nt = t
	if rt, isok := keywords[t.Lexema]; isok {
		rt.Lexema = t.Lexema
		nt = rt
	}
	return nt
}

func (l *Lexer) Lex() (t Token, err error) {
	defer func() {
		if t.Type == TokBad && err == nil {
			err = errors.New("bad token")
		}
		if r := recover(); r != nil {
			errs := fmt.Sprintf("%s", r)
			t.Type = TokBad
			err = errors.New(errs)
		}
		if DToks && err == nil {
			fmt.Fprintf(os.Stderr, "Lex: %s\n", t)
		}
	}()
	if t, ok := l.takeSaved(); ok {
		return t, nil
	}
LoopTok:
	for r := l.get(); ; r = l.get() {
		if DFlex {
			fmt.Fprintf(os.Stderr, "when Lex loop: %c\n", r)
		}
		if unicode.IsSpace(r) {
			l.accept()
			continue
		}

		switch r {
		case arrowThen, arrowNThen: //Synonyms
			l.accept()
			t = synomTok[r]
			return t, nil
		case '#':
			l.eatComment()
			continue
		case '(', ')', '*', '/', '+', '-', '^', ',', '%', ';', ':', '?', '~':
			t.Type = TokType(r)
			t.Lexema = l.accept()
			return t, nil
		case '=', '!', '>', '<', '&', '|':
			t.Type = TokType(r)
			err = l.mayLexDouble(&t)
			t.Lexema = l.accept()
			return t, err
		case RuneEof:
			t.Type = TokEof
			l.accept()
			return t, nil
		case '"':
			l.unget()
			t, err = l.lexString()
			return t, err
		}
		switch {
		case r == '.', unicode.IsDigit(r):
			l.unget()
			t, err = l.lexNum()
			return t, err
		case unicode.IsLetter(r) || r == '_':
			l.unget()
			t, err = l.lexId()
			break LoopTok //may need further processing
		default:
			errs := fmt.Sprintf("bad rune%c: %x", r, r)
			return t, errors.New(errs)
		}
	}
	//secondary processing
	if t.Type == TokId {
		t = keywordLookup(t)
	}
	return t, err
}

// to recover from errors
func (l *Lexer) LexWhileNot(tps ...TokType) (t Token, err error) {
LEXLOOP:
	for t, err = l.Peek(); err == nil; t, err = l.Peek() {
		if t.Type == TokEof {
			return t, nil
		}
		for _, tp := range tps {
			if tp == t.Type {
				break LEXLOOP
			}
		}
		t, err = l.Lex()
	}
	return t, err
}

// for reporting to the user
type UTokType TokType

func (t UTokType) String() string {
	switch TokType(t) {
	case TokMod:
		return "%"
	case TokComma:
		return ","
	case TokBoolVal:
		return "TokBoolVal"
	case TokFloatVal:
		return "TokFloatVal"
	case TokIntVal:
		return "TokIntVal"
	case TokStrVal:
		return "TokStrVal"
	case TokId:
		return "TokId"
	case TokEof:
		return "TokEof"
	case TokQuest:
		return "?"
	case TokSemi:
		return ";"
	case TokColon:
		return ":"
	case TokLPar:
		return "("
	case TokRPar:
		return ")"
	case TokAdd:
		return "+"
	case TokMin:
		return "-"
	case TokMul:
		return "*"
	case TokDiv:
		return "/"
	case TokCompl:
		return "~"
	case TokXor:
		return "^"
	case TokBitAnd:
		return "&"
	case TokBitOr:
		return "|"
	case TokLogAnd:
		return "&&"
	case TokLogOr:
		return "||"
	case TokEq:
		return "=="
	case TokNEq:
		return "!="
	case TokAsig:
		return "="
	case TokG:
		return ">"
	case TokL:
		return "<"
	case TokGEq:
		return ">="
	case TokThen:
		return "=>"
	case TokNThen:
		return "!>"
	case TokLEq:
		return "<="
	case TokLogNeg:
		return "!"
	case TokBad:
		return "TokBad"
	case TokLevels:
		return "Levels"
	case TokSoft:
		return "Soft"
	case TokVars:
		return "Vars"
	case TokConsts:
		return "Consts"
	case TokRules:
		return "Rules"
	default:
		return "TokUnk"
	}
}
