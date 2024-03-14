package ab_test

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

var files = []string{
	"./xrips/examples/count.rul",
	"./xrips/examples/string.rul",
	"./xrips/examples/countdivzero.rul",
	"./xrips/examples/dead.rul",
	"./xrips/examples/dead2.rul",
	"./xrips/examples/ids.rul",
	//"./xrips/examples/plug.rul",
	"./xrips/examples/regexp.rul",
	"./xrips/examples/rulename.rul",
	"./xrips/examples/signal.rul",
	"./xrips/examples/simple.rul",
	"./xrips/examples/stringlev.rul",
	"./xrips/examples/stringlong.rul",
	"./xrips/examples/scenariox.rul",
}

func BenchmarkGen(b *testing.B) {
	pid := os.Getpid()
	pids := fmt.Sprintf("%d", pid)
	b.StopTimer()
	for _, f := range files {
		if testing.Verbose() {
			fmt.Fprintf(os.Stderr, "testing ./bench.sh -g %s %s\n", pids, f)
		}
		fpart := strings.Split(f, "/")
		testname, _ := strings.CutSuffix(fpart[len(fpart)-1], ".rul")
		//run once to prepare
		cmd := exec.Command("./bench.sh", "-g", pids, f)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			b.Fatalf("./bench.sh %s\n\tfailed: %s", f, err)
		}
		cmd = exec.Command("./bench.sh", "-r", "-g", pids, f)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			b.Fatalf("./bench.sh %s\n\tfailed: %s", f, err)
		}
		testpath := fmt.Sprintf("/tmp/%s%d", testname, pid)
		b.StartTimer()
		cmd = exec.Command("./nc.sh", testpath)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			b.Fatalf("./nc.sh %s\n\tfailed: %s", testpath, err)
		}
		b.StopTimer()
	}
}

func BenchmarkReg(b *testing.B) {
	pid := os.Getpid()
	pids := fmt.Sprintf("%d", pid)
	b.StopTimer()
	for _, f := range files {
		if testing.Verbose() {
			fmt.Fprintf(os.Stderr, "testing ./bench.sh %s %s\n", pids, f)
		}
		fpart := strings.Split(f, "/")
		testname, _ := strings.CutSuffix(fpart[len(fpart)-1], ".rul")
		cmd := exec.Command("./bench.sh", pids, f)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			b.Fatalf("./bench.sh %s\n\tfailed: %s", f, err)
		}
		cmd = exec.Command("./bench.sh", "-r", pids, f)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			b.Fatalf("./bench.sh %s\n\tfailed: %s", f, err)
		}
		testpath := fmt.Sprintf("/tmp/%s%d", testname, pid)
		b.StartTimer()
		cmd = exec.Command("./nc.sh", testpath)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			b.Fatalf("./nc.sh %s\n\tfailed: %s", testpath, err)
		}
		b.StopTimer()
	}
}

func BenchmarkRegStats(b *testing.B) {
	pid := os.Getpid()
	pids := fmt.Sprintf("%d", pid)
	b.StopTimer()
	for _, f := range files {
		if testing.Verbose() {
			fmt.Fprintf(os.Stderr, "testing ./bench.sh %s %s\n", pids, f)
		}
		fpart := strings.Split(f, "/")
		testname, _ := strings.CutSuffix(fpart[len(fpart)-1], ".rul")
		cmd := exec.Command("./bench.sh", pids, f)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			b.Fatalf("./bench.sh %s\n\tfailed: %s", f, err)
		}
		cmd = exec.Command("./bench.sh", "-r", pids, f)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			b.Fatalf("./bench.sh %s\n\tfailed: %s", f, err)
		}
		testpath := fmt.Sprintf("/tmp/%s%d", testname, pid)
		b.StartTimer()
		cmd = exec.Command("./nc.sh", "-s", testpath)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			b.Fatalf("./nc.sh %s\n\tfailed: %s", testpath, err)
		}
		b.StopTimer()
	}
}

func BenchmarkGenStats(b *testing.B) {
	pid := os.Getpid()
	pids := fmt.Sprintf("%d", pid)
	b.StopTimer()
	for _, f := range files {
		if testing.Verbose() {
			fmt.Fprintf(os.Stderr, "testing ./bench.sh -g %s %s\n", pids, f)
		}
		fpart := strings.Split(f, "/")
		testname, _ := strings.CutSuffix(fpart[len(fpart)-1], ".rul")
		//run once to prepare
		cmd := exec.Command("./bench.sh", "-g", "-s", pids, f)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			b.Fatalf("./bench.sh %s\n\tfailed: %s", f, err)
		}
		cmd = exec.Command("./bench.sh", "-r", "-g", pids, f)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			b.Fatalf("./bench.sh %s\n\tfailed: %s", f, err)
		}
		testpath := fmt.Sprintf("/tmp/%s%d", testname, pid)
		b.StartTimer()
		cmd = exec.Command("./nc.sh", "-s", testpath)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			b.Fatalf("./nc.sh %s\n\tfailed: %s", testpath, err)
		}
		b.StopTimer()
	}
}
