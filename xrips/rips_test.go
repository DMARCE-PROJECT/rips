package xrips_test

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"rips/rips/extern"
	"rips/rips/lex"
	"rips/rips/xrips"
	"runtime/debug"
	"sort"
	"strings"
	"testing"
)

//go:embed examples/onemsg1
var msg string

//go:embed examples/simple.rul
var simple string

type RegexpMatch struct {
	re        string
	ismatch   bool
	iscorrect bool
}

func Nop() {
	return
}

// testing of regexp
func TestMatch(t *testing.T) {
	rms := []RegexpMatch{
		{".*[^/]/pose", true, true},
		{".*[^/]pose", false, true},
		{"*", false, false},
	}
	for _, rm := range rms {
		pstr := strings.Replace(simple, "RULE", rm.re, 1)
		pfile := strings.NewReader(pstr)
		deblevel := 0
		out := ioutil.Discard
		if testing.Verbose() {
			out = os.Stderr
		}
		r := xrips.NewRips("examples/simple.rul", pfile, deblevel, out)
		_, err := r.BuildAst(nil)
		if err != nil && rm.iscorrect {
			t.Fatal(err)
		}
		if err == nil && !rm.iscorrect {
			t.Fatal(err)
		}
		if !rm.iscorrect {
			continue
		}
		conn := strings.NewReader(msg)

		context := extern.NewContext(nil, "", len(r.Program.Levels), out, nil)
		context.Fatal = Nop
		context.Conn = conn
		context.RConn = bytes.NewBufferString("")
		var rosmsg extern.RosMsg
		rd := extern.NewRosDecoder(context.Conn)

		execEnv := r.Program.NewExecEnv(context)
		err = rd.Decode(&rosmsg)
		if err != nil {
			t.Fatal("decoding ../extern/examples/onemsg1 message")
		}
		msg := extern.NewMsg(&rosmsg)
		context.Update(msg)
		r.Program.Interp(context, execEnv)
		svar := execEnv.GetSym("ismatch")
		if svar == nil || svar.Val == nil || svar.Val.BoolVal != rm.ismatch {
			t.Fatal("should match")
		}
		r.Program.Done(execEnv)
	}
}

//go:embed examples/dead.rul
var dead string

// testing of various false conditions
func TestFalseCond(t *testing.T) {
	pfile := strings.NewReader(dead)
	deblevel := 0
	out := ioutil.Discard
	if testing.Verbose() {
		out = os.Stderr
	}
	r := xrips.NewRips("examples/dead.rul", pfile, deblevel, out)
	_, err := r.BuildAst(nil)
	if err != nil {
		t.Fatal(err)
	}
	conn := strings.NewReader(msg)
	context := extern.NewContext(nil, "", len(r.Program.Levels), out, nil)
	context.Conn = conn
	context.Fatal = Nop
	context.RConn = bytes.NewBufferString("")

	var rosmsg extern.RosMsg
	rd := extern.NewRosDecoder(context.Conn)

	execEnv := r.Program.NewExecEnv(context)
	err = rd.Decode(&rosmsg)
	if err != nil {
		t.Fatal("decoding ../extern/examples/onemsg1 message")
	}
	msg := extern.NewMsg(&rosmsg)
	context.Update(msg)
	r.Program.Interp(context, execEnv)
	svar := execEnv.GetSym("isseen")
	if svar == nil || svar.Val == nil || svar.Val.BoolVal {
		t.Fatal("should not be seen")
	}
	svar = execEnv.GetSym("num")
	// the rule is not triggered, should keep original value
	if svar == nil || svar.Val == nil || (svar.Val.IntVal != 16) {
		t.Fatalf("num is not correct: %d\n", svar.Val.IntVal)
	}
	// see the calculation in examples/dead.rul
	a := r.Program.RuleSects[0].Rules[0].Actions[0]
	v := a.What.Expr.Args[1]
	if v == nil || (v.IntVal != 10) {
		t.Fatalf("constant val is not correct: %d\n", v.IntVal)
	}
	r.Program.Done(execEnv)
}

// testing of dead code
func TestDead(t *testing.T) {
	pfile := strings.NewReader(dead)
	deblevel := 0
	out := ioutil.Discard
	if testing.Verbose() {
		out = os.Stderr
	}
	r := xrips.NewRips("examples/dead.rul", pfile, deblevel, out)
	_, err := r.BuildAst(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Program.RuleSects[0].Rules) != 3 {
		t.Fatal("dead code appearing")
	}
}

// testing of error files
func ErrFiles(t *testing.T, out io.Writer) {
	fes, err := errfs.ReadDir("examples")
	if err != nil {
		t.Fatalf("read dir should not fail")
	}
	for _, fe := range fes {
		fname := "examples/" + fe.Name()
		if testing.Verbose() {
			fmt.Fprintf(os.Stderr, "File %s\n", fname)
		}
		data, err := errfs.ReadFile(fname)
		if err != nil {
			t.Fatalf("read %s should not fail", fe.Name())
		}
		fcont := string(data)
		pfile := strings.NewReader(fcont)
		deblevel := 0
		r := xrips.NewRips(fname, pfile, deblevel, out)
		n, err := r.BuildAst(nil)
		if testing.Verbose() {
			fmt.Fprintf(os.Stderr, "File %s %d errors: %s\n", fname, n, err)
		}
		//should err
		if err == nil {
			t.Fatalf("%s should fail\n", fe.Name())
		}
		if n <= 0 {
			t.Fatalf("should only be one error: %d", n)
		}
	}
}

//go:embed examples/*err.rul
var errfs embed.FS

// testing of error files
func TestAllErr(t *testing.T) {
	out := ioutil.Discard
	if testing.Verbose() {
		out = os.Stderr
	}
	ErrFiles(t, out)
}

const bufSz = 4 * 1024

func areEqualReaders(r1 io.Reader, r2 io.Reader) bool {
	b1 := make([]byte, bufSz)
	b2 := make([]byte, bufSz)
	for {
		_, err1 := r1.Read(b1)
		_, err2 := r2.Read(b2)
		if err1 != nil || err2 != nil {
			if err1 == io.EOF && err2 == io.EOF {
				return true
			} else {
				return false
			}
		}
		if !bytes.Equal(b1, b2) {
			return false
		}
	}
	//return false
}

const OutName = "errouts/okout.out"

// compare the byte slices lexicographically
// it does not matter, we just want them to be always the same
func (l lexicalorder) Less(i, j int) bool { return bytes.Compare(l[i], l[j]) < 0 }
func (l lexicalorder) Len() int           { return len(l) }
func (l lexicalorder) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }

type lexicalorder [][]byte

// testing of error files
func TestErrOut(t *testing.T) {
	r, w := io.Pipe()

	go func() { ErrFiles(t, w); w.Close() }()

	content, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	r.Close()
	lines := bytes.Split(content, []byte{'\n'})

	//Order so they appear in order regardless of filesystem, etc.
	sort.Sort(lexicalorder(lines))

	content = bytes.Join(lines, []byte{'\n'})

	out, err := os.CreateTemp("/tmp", "rips_test")
	if err != nil {
		t.Fatal(err)
	}
	_, err = out.Write(content)
	if err != nil {
		t.Fatal(err)
	}

	_, err = out.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	okout, err := os.OpenFile(OutName, os.O_RDWR, 0666)
	if err != nil {
		if !os.IsNotExist(err) {
			t.Fatal(err)
		}
		okout.Close()
		okout, err := os.OpenFile(OutName, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
		if err != nil {
			t.Fatal(err)
		}
		defer okout.Close()
		_, err = io.Copy(okout, out)
		if err != nil {
			t.Fatal(err)
		}
		return
	}
	defer okout.Close()
	if areEqualReaders(out, okout) {
		os.Remove(out.Name())
		return
	}
	t.Fatalf("Diff output: run:\n\t9 diff -n %s %s\n", out.Name(), okout.Name())
}

//go:embed examples/initvarerr.rul
var initvarerr string

// testing of constant init var
func TestInitVar(t *testing.T) {
	pfile := strings.NewReader(initvarerr)
	deblevel := 0
	out := ioutil.Discard
	if testing.Verbose() {
		out = os.Stderr
	}
	r := xrips.NewRips("examples/initvarerr.rul", pfile, deblevel, out)
	n, err := r.BuildAst(nil)
	//should err
	if err == nil {
		t.Fatal(err)
	}
	if n <= 0 || n > 1 {
		t.Fatal("should only be one error")
	}
}

//go:embed examples/count.rul
var count string

//go:embed examples/msg1
var msgs string

func TestMsgCount(t *testing.T) {
	pfile := strings.NewReader(count)
	deblevel := 0
	out := ioutil.Discard
	if testing.Verbose() {
		out = os.Stderr
	}
	r := xrips.NewRips("examples/count.rul", pfile, deblevel, out)
	_, err := r.BuildAst(nil)
	if err != nil {
		t.Fatal(err)
	}
	conn := strings.NewReader(msgs)
	context := extern.NewContext(nil, "", len(r.Program.Levels), out, nil)
	context.Conn = conn
	context.Fatal = Nop
	context.RConn = bytes.NewBufferString("")

	rd := extern.NewRosDecoder(context.Conn)

	execEnv := r.Program.NewExecEnv(context)
	for {
		var rosmsg extern.RosMsg
		err = rd.Decode(&rosmsg)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal("decoding ../extern/examples/onemsg1 message")
		}
		msgr := extern.NewMsg(&rosmsg)
		context.Update(msgr)
		r.Program.Interp(context, execEnv)
	}
	if err != io.EOF {
		t.Fatal(err)
	}
	svar := execEnv.GetSym("nmsg")
	// egrep '^event:' examples/msg1|grep message|wc -l
	if svar == nil || svar.Val == nil || svar.Val.IntVal != 532 {
		t.Fatalf("wrong number of msgs: %d", svar.Val.IntVal)
	}
	r.Program.Done(execEnv)
}

// testing of various false conditions
func TestRun(t *testing.T) {
	pfile := strings.NewReader(count)
	deblevel := 0
	out := ioutil.Discard
	if testing.Verbose() {
		out = os.Stderr
	}
	r := xrips.NewRips("examples/count.rul", pfile, deblevel, out)
	_, err := r.BuildAst(nil)
	if err != nil {
		t.Fatal(err)
	}
	conn := strings.NewReader(msg)
	context := extern.NewContext(nil, "", len(r.Program.Levels), out, nil)
	context.Conn = conn
	context.Fatal = Nop
	context.RConn = bytes.NewBufferString("")

	rd := extern.NewRosDecoder(context.Conn)

	var rosmsg extern.RosMsg

	execEnv := r.Program.NewExecEnv(context)
	err = rd.Decode(&rosmsg)
	if err != nil {
		t.Fatal("decoding ../extern/examples/msg1 message")
	}
	msg := extern.NewMsg(&rosmsg)
	context.Update(msg)
	r.Program.Interp(context, execEnv)
	r.Program.Done(execEnv)
}

func recovCrashFail(f *testing.F) {
	if r := recover(); r != nil {
		fmt.Fprintf(os.Stderr, "%s\n%s", r, debug.Stack())
		f.Fatal("something bad happened")
	}
}

// testing of various false conditions
func FuzzRipsSmall(f *testing.F) {
	defer recovCrashFail(f)
	nn := []int{1, 300, 25, 9999, 1, 660, 25, 30, 1, 3, 2555, 675, 12, 45, 25}
	for i, seed := range []string{"const", "vars", "8", "<", "rules", "true", "?", "=", "levels", ";", "+", "int", " ", "\n", "#", "Msg", "CurrTime", "CurrRule", "Uptime", "0xaa", "set", "topicmatches", ":", "soft", "!", "-", "(", ")", `"`, " "} {
		f.Add(seed, nn[i%len(nn)]+i)
	}
	f.Fuzz(func(t *testing.T, in string, ncut int) {
		if ncut < 0 {
			ncut = -ncut
		}
		ncut = ncut % len(count)
		sr := []rune(count)
		cc := string(sr[0:ncut]) + in + string(sr[ncut:])
		pfile := strings.NewReader(cc)
		deblevel := 0
		out := ioutil.Discard
		if testing.Verbose() {
			out = os.Stderr
		}
		r := xrips.NewRips("examples/count.rul", pfile, deblevel, out)
		_, err := r.BuildAst(nil)
		if err != nil {
			return
		}
		conn := strings.NewReader(msg)
		context := extern.NewContext(nil, "", len(r.Program.Levels), out, nil)
		context.Fatal = Nop
		context.Conn = conn
		context.RConn = bytes.NewBufferString("")

		rd := extern.NewRosDecoder(context.Conn)

		var rosmsg extern.RosMsg

		execEnv := r.Program.NewExecEnv(context)
		err = rd.Decode(&rosmsg)
		if err != nil {
			t.Fatal("decoding ../extern/examples/onemsg1 message")
		}
		msg := extern.NewMsg(&rosmsg)
		context.Update(msg)
		r.Program.Interp(context, execEnv)
		r.Program.Done(execEnv)
	})
}

func fuzztarget(f *testing.F, fname string, fcontents string) func(t *testing.T, in string, ncut int) {
	nn := []int{1, 300, 25, 9999, 1, 660, 25, 30, 1, 3, 2555, 675, 12, 45, 25, 7777, 4754}
	l, err := lex.NewFakeLexer(fcontents, ioutil.Discard)
	if err != nil {
		f.Fatal(err)
	}
	for i := 0; ; i++ {
		tok, err := l.Lex()
		if err != nil {
			f.Fatalf("lexer should not fail in this case: %s\n", err)
		}
		if tok.Type == lex.TokEof {
			break
		}
		f.Add(tok.Lexema, nn[i%len(nn)]+i)
		f.Add(" "+tok.Lexema, nn[i%len(nn)]+i)
		f.Add(tok.Lexema+" ", nn[(2*i)%len(nn)]+i)
		f.Add(" "+tok.Lexema+" ", nn[(3*i)%len(nn)]+i)
	}
	return func(t *testing.T, in string, ncut int) {
		if ncut < 0 {
			ncut = -ncut
		}
		ncut = ncut % len(fcontents)
		sr := []rune(fcontents)
		cc := string(sr[0:ncut]) + in + string(sr[ncut:])
		pfile := strings.NewReader(cc)
		deblevel := 0
		out := ioutil.Discard
		if testing.Verbose() {
			out = os.Stderr
		}
		r := xrips.NewRips(fname, pfile, deblevel, out)
		_, err := r.BuildAst(nil)
		if err != nil {
			return
		}
		conn := strings.NewReader(msg)

		context := extern.NewContext(nil, "", len(r.Program.Levels), out, nil)
		context.Conn = conn
		context.Fatal = Nop
		context.RConn = bytes.NewBufferString("")
		var rosmsg extern.RosMsg
		rd := extern.NewRosDecoder(context.Conn)

		execEnv := r.Program.NewExecEnv(context)
		err = rd.Decode(&rosmsg)
		if err != nil {
			t.Fatal("decoding ../extern/examples/onemsg1 message")
		}
		msg := extern.NewMsg(&rosmsg)
		context.Update(msg)
		r.Program.Interp(context, execEnv)
		r.Program.Done(execEnv)
	}

}

//go:embed examples/regexp.rul
var fuzzfile0 string

// testing of various false conditions
func FuzzRipsBigger0(f *testing.F) {
	defer recovCrashFail(f)
	ft := fuzztarget(f, "examples/regexp.rul", fuzzfile0)
	f.Fuzz(ft)
}

//go:embed examples/initvarerr.rul
var fuzzfile1 string

func FuzzRipsBigger1(f *testing.F) {
	defer recovCrashFail(f)
	ft := fuzztarget(f, "examples/initvarerr.rul", fuzzfile1)
	f.Fuzz(ft)
}

//go:embed examples/countstr.rul
var fuzzfile2 string

func FuzzRipsBigger2(f *testing.F) {
	defer recovCrashFail(f)
	ft := fuzztarget(f, "examples/countstr.rul", fuzzfile2)
	f.Fuzz(ft)
}

//go:embed examples/count.rul
var fuzzfile3 string

func FuzzRipsBigger3(f *testing.F) {
	defer recovCrashFail(f)
	ft := fuzztarget(f, "examples/count.rul", fuzzfile3)
	f.Fuzz(ft)
}

//go:embed examples/scenario1.rul
var fuzzfile4 string

func FuzzRipsBigger4(f *testing.F) {
	defer recovCrashFail(f)
	ft := fuzztarget(f, "examples/scenario1.rul", fuzzfile4)
	f.Fuzz(ft)
}

// this is just a concatenation of all of them
//
//go:embed examples/badfullerr.rul
var fuzzfile5 string

func FuzzRipsBigger5(f *testing.F) {
	defer recovCrashFail(f)
	ft := fuzztarget(f, "examples/badfullerr.rul", fuzzfile5)
	f.Fuzz(ft)
}

//go:embed examples/scenariox.rul
var fuzzfile6 string

func FuzzRipsBigger6(f *testing.F) {
	defer recovCrashFail(f)
	ft := fuzztarget(f, "examples/scenariox.rul", fuzzfile6)
	f.Fuzz(ft)
}

//go:embed examples/dead2.rul
var fuzzdead string

func FuzzRipsDead(f *testing.F) {
	defer recovCrashFail(f)
	ft := fuzztarget(f, "examples/dead2.rul", fuzzdead)
	f.Fuzz(ft)
}

//TODO error testing, make sure errors are reported...
