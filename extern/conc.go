package extern

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"rips/rips/stats"
	"syscall"
	"time"
)

// Runs in the main thread, when it returns all is over.
// starts main dispatcher by sending a response
func MsgDecoder(context *Ctx, mc chan<- *Msg, mcr <-chan *Msg) (err error) {
	context.Stats.Start(stats.Total)
	mc <- nil //start main dispatcher
	rd := NewRosDecoder(context.Conn)
	for {
		var rosmsg RosMsg
		context.Stats.Start(stats.Decoding)
		err = rd.Decode(&rosmsg)
		context.Stats.End(stats.Decoding)
		if err != nil {
			mc <- nil
			close(mc)
			break
		}
		msg := NewMsg(&rosmsg)
		mc <- msg
		<-mcr
	}
	if err != io.EOF {
		fmt.Fprintf(os.Stderr, "Rips: msg error\n")
		return err
	}
	context.Stats.End(stats.Total)
	fmt.Fprintf(os.Stderr, "Stats: %s\n", context.Stats)
	fmt.Fprintf(os.Stderr, "Rips: msg decoder exiting\n")
	return nil
}

func SockRemove(path string) (err error) {
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if (fi.Mode() & fs.ModeType) == fs.ModeSocket {
		return os.Remove(path)
	}
	return nil
}

type Dispatch struct {
	Coremain func(context *Ctx)
	Sockpath string
	Sigc     <-chan os.Signal
	Mc       <-chan *Msg
	Mcr      chan<- *Msg
	Pathsc   <-chan string
}

const PollInterval = 200 * time.Millisecond

func Dispatcher(context *Ctx, d *Dispatch) {
	var iserr int
	<-d.Mc //receive for kick-off from msg decoder

	ct := make(chan int, 0)
	go func() {
		for range ct {
			time.Sleep(PollInterval)
		}
	}()
	defer close(ct)
OutFor:
	for {
		select {
		case path := <-d.Pathsc:
			context.Paths[path] = true
		case ct <- 0:
			//here we will poll whatever needs to be polled
			//call program without msg
			context.Update(nil)
			context.Stats.Start(stats.Executing)
			d.Coremain(context)
			context.Stats.End(stats.Executing)
			context.Paths = make(map[string]bool)
		case msg := <-d.Mc:
			context.Update(msg)
			if msg == nil {
				//fmt.Fprintf(os.Stderr, "final message\n")
				break OutFor
			}
			context.Stats.Start(stats.Executing)
			d.Coremain(context)
			context.Stats.End(stats.Executing)
			d.Mcr <- nil
		case sig := <-d.Sigc:
			switch sig {
			case syscall.SIGUSR1:
				context.Nusr1++
			case syscall.SIGUSR2:
				context.Nusr2++
			default:
				fmt.Fprintf(os.Stderr, "Rips: signal received %s, exiting\n", sig)
				iserr = 1
				break OutFor
			}
		}
	}
	//fmt.Fprintf(os.Stderr, "Rips: exiting\n")
	SockRemove(d.Sockpath)
	os.Exit(iserr)
}
