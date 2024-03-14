package tree

// #######################
const GoPrelude = `
//automatically generated by rips
//run go fmt before reading it
package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"regexp"
	"rips/rips/extern"
	"rips/rips/stats"
	"strings"
	"syscall"
	"time"
	godebug "runtime/debug"
	"github.com/kgwinnup/go-yara/yara"
)

const DefPathScripts = "/etc/rips/scripts"
const DefSockPath = "/tmp/sock.rips"
const HasStats = true

func usage() {
	fmt.Fprintf(os.Stderr, "usage: rips [-D] [-r rootpath] [-s sockpath] [scriptspath]\n")
	os.Exit(1)
}
var yaraRules = map[string] *yara.Yara {
} 

var regexRules = map[string] *regexp.Regexp {
}

func lookupYara(pathrule string) *yara.Yara{
	if yr, ok := yaraRules[pathrule]; ok {
		return yr
	}
	yr, err := extern.YaraRule(pathrule)
	if err != nil {
		return nil
	}
	yaraRules[pathrule] = yr
	return yr
	
}

func lookupRegexp(res string) *regexp.Regexp{
	if re, ok := regexRules[res]; ok {
		return re
	}
	re, err := regexp.Compile(res)
	if err != nil {
		return nil
	}
	regexRules[res] = re
	return re
}
`

// #######################
const GoMiddle = `
func initPredefVars(context *extern.Ctx) {
	now :=time.Now()
	Time = now.UnixNano()
	context.TimeStarted = now
	d := now.Sub(context.TimeStarted)
	Uptime =  d.Nanoseconds()
}
func updatePredefVars(context *extern.Ctx) {
	now := time.Now()
	Time = now.UnixNano()
	d := now.Sub(context.TimeStarted)
	Uptime =  d.Nanoseconds()
}
func main() {
	var stats stats.Stats
	log.SetPrefix("RipsG:")
	args := os.Args
	errout := os.Stderr
	deblevel := 0
	levelNames = levelNames
	defer func() {
		if e := recover(); e != nil {
			errs := fmt.Sprint(e)
			if strings.HasPrefix(errs, "runtime error:") {
				errs = strings.Replace(errs, "runtime error:", "rips internal error:", 1)
			}
			if deblevel > 0 {
				fmt.Fprintf(os.Stderr, "%s", godebug.Stack())
			}
			log.Fatal(errs)
		}
	}()
	args = args[1:]

	sockpath := DefSockPath
	rootpath := "."
	doneargs := false
	for len(args) > 0 && len(args[0]) >= 2 && args[0][0] == '-' {
		switch args[0][:2] {
		case "-D":
			deblevel = len(args[0]) - 1
			args = args[1:]
		case "-s":
			if len(args) < 2 {
				usage()
			}
			sockpath = args[1]
			args = args[2:]
		case "-r":
			if len(args) < 2 {
				usage()
			}
			rootpath = args[1]
			args = args[2:]
			err := os.Chdir(rootpath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "cannot chdir to %s: %s\n", rootpath, err)
				usage()
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
		fmt.Fprintf(os.Stderr, "cannot access path for scripts: %s\n", pathscripts)
		usage()
	}
	if len(args) != 0 {
		usage()
	}

	context := extern.NewContext(nil, pathscripts, len(levelNames), errout, &stats)
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

	pathsc := make(chan string, 1)
	exitc := make(chan int, 1)
	extern.Watcher(extern.DirSnort, pathsc, exitc)
	defer func() { exitc <- 1 }()

	mc := make(chan *extern.Msg, 1)
	mcr := make(chan *extern.Msg, 1)
	runprog := func(internal_Rips_context *extern.Ctx) { return }

	d := &extern.Dispatch{
		Coremain: runprog,
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
	initPredefVars(context)
	context.CurrLevel = CurrLevel
	Uptime = Uptime //make them used
	Time = Time
	currid := int(CurrLevel)
	currname := levelNames[CurrLevel]
	extern.Trigger(context, currname, currid, currname, currid, false)

	//######### start of function GENERATED
	runprog = func(internal_Rips_context *extern.Ctx) {
		defer func() {
			if e := recover(); e != nil {
				errs := fmt.Sprint(e)
				if strings.HasPrefix(errs, "runtime error:") {
					errs = strings.Replace(errs, "runtime error:", "rips internal error:", 1)
				}
				if deblevel > 0 {
					fmt.Fprintf(os.Stderr, "%s", godebug.Stack())
				}
				log.Fatal(errs)
			}
		}()
		updatePredefVars(context)
		Uptime = Uptime //make them used
		Time = Time
		CurrLevel = context.CurrLevel
		CurrLevel = CurrLevel
`

// #######################
const GoEpilogue = `
	}
	//######### end of function GENERATED
	d.Coremain = runprog
	err = extern.MsgDecoder(context, mc, mcr)
	if err != nil {
		log.Fatal(err)
	}
}
`