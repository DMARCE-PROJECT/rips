package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"rips/rips/extern"
	"rips/rips/stats"
	"rips/rips/xrips"
	godebug "runtime/debug"
	"strings"
	"syscall"
)

const DefPathScripts = "/etc/rips/scripts"
const DefSockPath = "/tmp/sock.rips"
const DebugSpy = true
const HasStats = true

func usage() {
	fmt.Fprintf(os.Stderr, "usage: rips [-s sockpath|-c] [-r rootpath] [-D] [pathscripts] file.rul\n")
	os.Exit(1)
}

func spliceConn(context *extern.Ctx, conn *net.Conn) (err error) {
	il, err := os.OpenFile("/tmp/inputlog.rips", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	ol, err := os.OpenFile("/tmp/outputlog.rips", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	context.Conn = io.TeeReader(*conn, il)
	context.RConn = io.MultiWriter(*conn, ol)
	return nil
}

func main() {
	var stats stats.Stats
	var r *xrips.Rips
	log.SetPrefix("Rips:")
	args := os.Args
	defer func() {
		if e := recover(); e != nil {
			errs := fmt.Sprint(e)
			if strings.HasPrefix(errs, "runtime error:") {
				errs = strings.Replace(errs, "runtime error:", "rips internal error:", 1)
			}
			if r.DebLevel > 0 {
				fmt.Fprintf(os.Stderr, "%s", godebug.Stack())
			}
			log.Fatal(errs)
		}
	}()
	args = args[1:]
	//HACK for args in hashbang
	if len(args) == 2 && strings.ContainsRune(args[0], ' ') && !extern.IsReadable(args[0]) {
		xargs := strings.Split(args[0], " ")
		args = append(xargs, args[1])
	}
	deblevel := 0
	if len(args) < 1 {
		usage()
	}
	fname := args[len(args)-1]
	args = args[:len(args)-1]

	sockpath := DefSockPath
	iscompile := false
	issock := false
	rootpath := "."
	doneargs := false
	for len(args) > 0 && len(args[0]) >= 2 && args[0][0] == '-' {
		switch args[0][:2] {
		case "-D":
			deblevel = len(args[0]) - 1
			args = args[1:]
		case "-c":
			if issock {
				usage()
			}
			iscompile = true
			args = args[1:]
		case "-s":
			if iscompile {
				usage()
			}
			issock = true
			if len(args) > 2 {
				sockpath = args[1]
				args = args[2:]
			}
		case "-r":
			if len(args) > 2 {
				rootpath = args[1]
				args = args[2:]
				err := os.Chdir(rootpath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "cannot chdir to %s: %s\n", rootpath, err)
					usage()
				}
			}
		case "--":
			doneargs = true
			args = args[1:]
			break
		default:
			usage()
		}
	}
	if !doneargs && len(args) > 0 && len(args[0]) > 0 && args[0][0] == '-' {
		usage()
	}
	pathscripts := DefPathScripts
	if len(args) > 0 {
		pathscripts = args[len(args)-1]
		args = args[0 : len(args)-1]
	}
	if !extern.IsExecutable(pathscripts) || !extern.IsReadable(pathscripts) {
		fmt.Fprintf(os.Stderr, "cannot access path for scripts: %s from %s\n", pathscripts, extern.CurrDir())
		usage()
	}

	if len(args) != 0 {
		usage()
	}
	pfile, err := os.Open(fname)
	if err != nil {
		log.Fatal(err)
	}
	r = xrips.NewRips(fname, pfile, deblevel, os.Stderr)
	defer pfile.Close()

	_, err = r.BuildAst(&stats)
	if err != nil {
		log.Fatal(err)
	}

	context := extern.NewContext(nil, pathscripts, len(r.Program.Levels), r.Lexer.Errout(), &stats)
	for _, level := range r.Program.Levels {
		context.AddLevel(level.Name)
	}
	nerr := xrips.CheckLevelScripts(r.Program, context)
	if nerr > 0 {
		log.Fatal("level scripts errors")
	}
	if iscompile {
		fmt.Fprintf(os.Stdout, "%g", r.Program)
		fmt.Fprintf(os.Stderr, "Stats: %s\n", context.Stats)
		os.Exit(0)
	}

	if err := extern.SockRemove(sockpath); err != nil {
		log.Fatal(err)
	}

	sock, err := net.Listen("unix", sockpath)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	defer sock.Close()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	signal.Notify(c, os.Interrupt, syscall.SIGUSR1)
	signal.Notify(c, os.Interrupt, syscall.SIGUSR2)

	pathsc := make(chan string, 500)
	exitc := make(chan int, 1)
	extern.Watcher(extern.DirSnort, pathsc, exitc)
	defer func() { exitc <- 1 }()

	mc := make(chan *extern.Msg, 1)
	mcr := make(chan *extern.Msg, 1)
	execEnv := r.Program.NewExecEnv(context)
	coremain := func(context *extern.Ctx) { r.Program.Interp(context, execEnv) }
	d := &extern.Dispatch{
		Coremain: coremain,
		Sockpath: sockpath,
		Sigc:     c,
		Mc:       mc,
		Mcr:      mcr,
		Pathsc:   pathsc,
	}
	go extern.Dispatcher(context, d)
	// Accept an incoming connection.
	conn, err := sock.Accept()
	if err != nil {
		extern.SockRemove(sockpath)
		log.Fatal(err)
	}
	extern.SockRemove(sockpath)
	defer conn.Close()

	context.Conn = conn
	context.RConn = conn
	if DebugSpy {
		err = spliceConn(context, &conn)
		if err != nil {
			log.Fatal(err)
		}
	}
	err = execEnv.SetPredefVars(r.Program, context)
	if err != nil {
		panic(err)
	}

	level := execEnv.GetSym("CurrLevel")
	if level == nil {
		panic("Level var disappeared")
	}
	extern.Trigger(context, level.Val.Name, level.Val.SLevel, level.Val.Name, level.Val.SLevel, false)
	err = extern.MsgDecoder(context, mc, mcr)
	if err != nil {
		log.Fatal(err)
	}
	r.Program.Done(execEnv)
}
