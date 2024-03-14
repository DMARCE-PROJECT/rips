package extern_test

import (
	"os"
	"rips/rips/extern"
	"testing"
)

func TestExec(t *testing.T) {
	context := extern.NewContext(nil, "", 0, os.Stderr, nil)
	rosmsg, err := rosMsg(onemsg1)
	if err != nil {
		t.Fatalf("decoding: %s\n", err)
	}
	msg := extern.NewMsg(&rosmsg)
	//fmt.Fprintf(os.Stderr, "%#v\n", rosmsg)
	context.Update(msg)
	if !extern.Exec(context, "/usr/bin/true") {
		t.Fatalf("Exec'ing true should give true for %#v\n", rosmsg)
	}
	if extern.Exec(context, "/usr/bin/false") {
		t.Fatalf("Exec'ing false should give false for %#v\n", rosmsg)
	}
}
