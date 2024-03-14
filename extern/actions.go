package extern

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

const DebActions = true

type yameler struct {
	w io.Writer
}

func (y yameler) Write(data []byte) (n int, err error) {
	s := "---\n"
	s += string(data)
	s += "\n...\n"
	n, err = y.w.Write([]byte(s))
	return n, err
}

func dprintfActions(s string, v ...interface{}) {
	if !DebActions {
		return
	}
	fmt.Fprintf(os.Stderr, "Extern action: "+s, v...)
}

func Alert(context *Ctx, msg string) bool {
	dprintfActions("alert: \"%s\"\n", msg)
	y := yameler{w: context.RConn}
	fmt.Fprintf(y, "alert: '%s'", msg) //TODO think about timeouts, etc.
	return true
}

func Exec(context *Ctx, path string, args ...string) bool {
	dprintfActions("exec: %s\n", path)
	cmd := exec.Command(path, args...)
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		context.Printf("exec: action error: %s\n", err)
		return false
	}
	return true
}

func Crash(context *Ctx, msg string) bool {
	dprintfActions("crash: \"%s\"\n", msg)
	ch := make(chan int, 1)
	go func() {
		y := yameler{w: context.RConn}
		fmt.Fprintf(y, "crashing: '%s'", msg)
		ch <- 0
	}()
	//shout everywhere
	fmt.Fprintf(os.Stderr, "crashing[stderr]: %s\n", msg)
	context.Printf("crashing: %s\n", msg)

	//alert the system, but if in a second it does not come back, crash anyway
	select {
	case <-ch:
	case <-time.After(time.Second):
	}
	context.Printf("exiting\n")
	os.Exit(2)
	return true
}

const (
	LevelFromFmt = "%s.from"
	LevelToFmt   = "%s.to"
)

const (
	X_OK = 1
	R_OK = 4
)

func Trigger(context *Ctx, levelto string, levelidto int, levelfrom string, levelidfrom int, fromsoft bool) bool {
	isfirst := !context.Init
	if levelto == levelfrom && !isfirst {
		return true
	}
	if levelidto > context.NLevels || levelidto < 0 {
		context.Printf("trigger: bad level %s[%d]soft:%v -> %s[%d]\n", levelfrom, levelidfrom, fromsoft, levelto, levelidto)
		return false
	}
	context.Init = true
	tohigher := levelidto > levelidfrom || isfirst
	tolower := fromsoft && (levelidfrom-1 == levelidto)
	if !tohigher && !tolower {
		context.Printf("trigger: error, trying to descalate too much %s[%d]soft:%v -> %s[%d]\n", levelfrom, levelidfrom, fromsoft, levelto, levelidto)
		return false
	}
	dprintfActions("trigger: to:%s[%d], from:%s[%d]\n", levelto, levelidto, levelfrom, levelidfrom)
	if context.ScriptsPath == "" {
		context.Printf("empty ScriptsPath")
		return false
	}
	spathfrom := context.ScriptsPath + "/" + fmt.Sprintf(LevelFromFmt, levelfrom)
	spathto := context.ScriptsPath + "/" + fmt.Sprintf(LevelToFmt, levelto)
	if !IsExecutable(spathfrom) {
		context.Printf("trigger: program %s is not executable from %s\n", spathfrom, CurrDir())
		return false
	}
	if !IsExecutable(spathto) {
		context.Printf("trigger: program %s is not executable from %s\n", spathto, CurrDir())
		return false
	}
	var cmd *exec.Cmd
	if !isfirst {
		cmd := exec.Command(spathfrom, levelto, levelfrom)
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			context.Printf("trigger:  error running program %s: %s\n", spathfrom, err)
			return false
		}
	}
	cmd = exec.Command(spathto, levelto, levelfrom)
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		context.Printf("trigger:  error running program %s: %s\n", spathto, err)
		return false
	}
	context.CurrLevel = int64(levelidto)
	y := yameler{w: context.RConn}
	max := context.NLevels - 1
	if max == 0 {
		max = 1
	}
	grav := float64(levelidto)
	grav = grav / float64(max)
	fmt.Fprintf(y, "level: '%s'\ngravity: %f", levelto, grav) //TODO think about timeouts, etc.
	return true
}
