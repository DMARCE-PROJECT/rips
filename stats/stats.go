package stats

import (
	"fmt"
	"time"
)

const (
	PrintStats = false //only when benchmarking.
)

const (
	Compiling = iota
	Decoding
	Executing
	Total
	NStats
)

var stname = [NStats]string{
	"Compiling",
	"Decoding",
	"Executing",
	"Total",
}

func StatName(st int) string {
	if st < 0 || st > NStats {
		return "Unknown"
	}
	return stname[st]
}

type Stats struct {
	startTime   [NStats]time.Time
	TimeElapsed [NStats]time.Duration
}

func (s *Stats) Start(st int) {
	if s == nil {
		return
	}
	s.startTime[st] = time.Now()
}

func (s *Stats) End(st int) {
	if s == nil {
		return
	}
	n := time.Now()
	s.TimeElapsed[st] += n.Sub(s.startTime[st])
}

func (s *Stats) String() (str string) {
	if s == nil {
		return "empty"
	}
	for st, n := range stname {
		str += fmt.Sprintf("%s: %v, ", n, s.TimeElapsed[st])
	}
	return str
}
