package ab_test

import (
	"fmt"
	"os"
	"os/exec"
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

func TestAB(t *testing.T) {
	for _, f := range files {
		if testing.Verbose() {
			fmt.Fprintf(os.Stderr, "testing ./ab_test.sh %s\n", f)
		}
		cmd := exec.Command("./ab_test.sh", f)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("./ab_test.sh %s\n\tfailed: %s", f, err)
		}
	}
}
