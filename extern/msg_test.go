package extern_test

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"rips/rips/extern"
	"strings"
	"testing"
)

//go:embed examples/msg1
var examplemsg1 string

func TestDecoder(t *testing.T) {
	var rosmsg extern.RosMsg

	rd := strings.NewReader(examplemsg1)
	rosd := extern.NewRosDecoder(rd)
	for {
		err := rosd.Decode(&rosmsg)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("decoding: %s\n", err)
		}
		if testing.Verbose() {
			fmt.Fprintf(os.Stderr, "msg is %s\n", &rosmsg)
		}
	}
}

//go:embed examples/msg2
var examplemsg2 string

func TestDecoder2(t *testing.T) {
	var rosmsg extern.RosMsg

	rd := strings.NewReader(examplemsg2)
	rosd := extern.NewRosDecoder(rd)
	for i := 0; ; i++ {
		err := rosd.Decode(&rosmsg)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("decoding [%d]: %s\n", i, err)
		}
		if testing.Verbose() {
			fmt.Fprintf(os.Stderr, "[%d] msg is %s\n", i, &rosmsg)
		}
	}
}
