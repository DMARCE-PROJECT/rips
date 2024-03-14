package extern

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"rips/rips/stats"
	"strings"
	"time"
)

type Ctx struct {
	CurrLevel   int64
	TimeStarted time.Time
	CurrentMsg  *Msg
	errout      io.Writer
	ScriptsPath string
	Conn        io.Reader
	RConn       io.Writer
	Nusr1       int
	Nusr2       int
	Paths       map[string]bool
	Levels      []string
	NLevels     int
	Init        bool
	Fatal       func()
	Stats       *stats.Stats
}

func DefFatal() {
	os.Exit(2)
}

func NewContext(msg *Msg, spath string, nlevels int, errout io.Writer, stats *stats.Stats) (context *Ctx) {
	paths := make(map[string]bool)
	return &Ctx{
		CurrLevel:   -1,
		CurrentMsg:  msg,
		errout:      errout,
		ScriptsPath: spath,
		Paths:       paths,
		NLevels:     nlevels,
		Fatal:       DefFatal,
		Stats:       stats,
	}
}

func (c *Ctx) Printf(s string, v ...interface{}) (int, error) {
	out := ioutil.Discard
	if c != nil {
		out = c.errout
	}
	return fmt.Fprintf(out, s, v...)
}

func (context *Ctx) Update(msg *Msg) {
	context.CurrentMsg = msg
}

func (context *Ctx) AddLevel(s string) {
	context.Levels = append(context.Levels, s)
}

func (context *Ctx) Topics() (tops []string) {
	roscontext := context.CurrentMsg.rosm.Context
	for _, rostop := range roscontext.Topics {
		tops = append(tops, rostop.Topic)
	}
	return tops
}

// CAREFUL: returns a reference to context
func (context *Ctx) rostopic(topic string) (rt *RosTopic) {
	roscontext := context.CurrentMsg.rosm.Context
	for _, rt := range roscontext.Topics {
		if rt.Topic == topic {
			return &rt
		}
	}
	return nil
}

// CAREFUL: returns a reference to context
func (context *Ctx) Publishers(topic string) (pubs []string) {
	rostopic := context.rostopic(topic)
	if rostopic == nil {
		return nil
	}
	return rostopic.Publishers
}

// CAREFUL: returns a reference to context
func (context *Ctx) Subscribers(topic string) (subs []string) {
	rostopic := context.rostopic(topic)
	if rostopic == nil {
		return nil
	}
	return rostopic.Subscribers
}

func (context *Ctx) RosType(topic string) (t string) {
	rostopic := context.rostopic(topic)
	if rostopic == nil || rostopic.Parameters == nil {
		return "UnknownRostype"
	}
	if rostopic.Parameters == nil {
		return "UnknownRosType"
	}
	param := rostopic.Parameters[0]
	fields := strings.Split(param, "/")
	return fields[0]
}

func (context *Ctx) RosSubtype(topic string) (t string) {
	rostopic := context.rostopic(topic)
	if rostopic == nil || rostopic.Parameters == nil {
		return "UnknownRosSubtype"
	}
	param := rostopic.Parameters[0]
	fields := strings.Split(param, "/")
	if len(fields) < 3 {
		return "UnknownRosSubtype"
	}
	return fields[2]
}

// CAREFUL: returns a reference to context
func (context *Ctx) RosNode(node string) (rn *RosNode) {
	roscontext := context.CurrentMsg.rosm.Context
	for _, rn := range roscontext.Nodes {
		if rn.Node == node {
			return &rn
		}
	}
	return nil
}

// CAREFUL: returns a reference to context
func (context *Ctx) Nodes() (rn []RosNode) {
	roscontext := context.CurrentMsg.rosm.Context
	return roscontext.Nodes
}

func (rn *RosNode) HasService(service string) bool {
	for _, s := range rn.Services {
		if s.Service == service {
			return true
		}
	}
	return false
}
