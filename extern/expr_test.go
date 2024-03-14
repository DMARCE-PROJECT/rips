package extern_test

import (
	"bytes"
	_ "embed"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"rips/rips/extern"
	"strings"
	"testing"
)

func rosMsg(m string) (rosmsg extern.RosMsg, err error) {
	rd := strings.NewReader(m)
	rosd := extern.NewRosDecoder(rd)
	err = rosd.Decode(&rosmsg)
	if err == io.EOF {
		return rosmsg, errors.New("empty msg")
	}
	return rosmsg, err
}

//go:embed examples/onemsg1
var onemsg1 string

//go:embed examples/onemsg2
var onemsg2 string

func TestMsgSubtype(t *testing.T) {
	context := extern.NewContext(nil, "", 0, os.Stderr, nil)
	rosmsg, err := rosMsg(onemsg1)
	if err != nil {
		t.Fatalf("decoding: %s\n", err)
	}
	msg := extern.NewMsg(&rosmsg)
	//fmt.Fprintf(os.Stderr, "%#v\n", rosmsg)
	context.Update(msg)
	if !extern.MsgSubtype(context, "turtlesim", "Pose") {
		t.Fatalf("MsgSubtype should give true for %#v\n", rosmsg)
	}
	if extern.MsgSubtype(context, "tuasdfsim", "Pose") {
		t.Fatalf("MsgSubtype should give false for: \n\t%#v\n", rosmsg)
	}
	if extern.MsgSubtype(context, "turtlesim", "adfasdf") {
		t.Fatalf("MsgSubtype should give false for: \n\t%#v\n", rosmsg)
	}
}

func TestMsgTypeIn(t *testing.T) {
	context := extern.NewContext(nil, "", 0, os.Stderr, nil)
	rosmsg, err := rosMsg(onemsg1)
	if err != nil {
		t.Fatalf("decoding: %s\n", err)
	}
	msg := extern.NewMsg(&rosmsg)
	context.Update(msg)
	if !extern.MsgTypeIn(context, "gambas", "turtlesim", "gaitas") {
		t.Fatalf("MsgTypeIn should give true for %#v\n", rosmsg)
	}
	if extern.MsgTypeIn(context, "gambas", "gaitas") {
		t.Fatalf("MsgTypeIn should give true for %#v\n", rosmsg)
	}
	if !extern.MsgTypeIn(context) {
		t.Fatalf("MsgTypeIn should give true for %#v\n", rosmsg)
	}
}

func TestPublisherCount(t *testing.T) {
	context := extern.NewContext(nil, "", 0, os.Stderr, nil)
	rosmsg, err := rosMsg(onemsg1)
	if err != nil {
		t.Fatalf("decoding: %s\n", err)
	}
	msg := extern.NewMsg(&rosmsg)
	context.Update(msg)
	if !extern.PublisherCount(context, 0, 10) {
		t.Fatalf("PublisherCount should give true for %#v\n", rosmsg)
	}
	if !extern.PublisherCount(context, 0, 1) {
		t.Fatalf("PublisherCount should give true for %#v\n", rosmsg)
	}
	if extern.PublisherCount(context, 2, 10) {
		t.Fatalf("PublisherCount should give false for %#v\n", rosmsg)
	}
}

func TestSubscribers(t *testing.T) {
	context := extern.NewContext(nil, "", 0, os.Stderr, nil)
	rosmsg, err := rosMsg(onemsg1)
	if err != nil {
		t.Fatalf("decoding: %s\n", err)
	}
	msg := extern.NewMsg(&rosmsg)
	context.Update(msg)
	if !extern.Subscribers(context) {
		t.Fatalf("Subscribers should give true for %#v\n", rosmsg)
	}
}

func TestPublishersMore(t *testing.T) {
	context := extern.NewContext(nil, "", 0, os.Stderr, nil)
	rosmsg, err := rosMsg(onemsg2)
	if err != nil {
		t.Fatalf("decoding: %s\n", err)
	}
	msg := extern.NewMsg(&rosmsg)
	context.Update(msg)
	if !extern.Publishers(context, "rips", "teleop_turtle", "turtlesim") {
		t.Fatalf("Publishers should give true for %#v\n", rosmsg)
	}
	if !extern.Publishers(context, "rips", "teleop_turtle", "rips", "turtlesim") {
		t.Fatalf("Publishers should give true for %#v\n", rosmsg)
	}
	if extern.Publishers(context, "rips", "turtlesim") {
		t.Fatalf("Publishers should give true for %#v\n", rosmsg)
	}
	if extern.Publishers(context) {
		t.Fatalf("Publishers should give true for %#v\n", rosmsg)
	}
}

func TestPublishersIncludeMore(t *testing.T) {
	context := extern.NewContext(nil, "", 0, os.Stderr, nil)
	rosmsg, err := rosMsg(onemsg2)
	if err != nil {
		t.Fatalf("decoding: %s\n", err)
	}
	msg := extern.NewMsg(&rosmsg)
	context.Update(msg)
	if !extern.PublishersInclude(context, "rips", "teleop_turtle", "turtlesim") {
		t.Fatalf("PublishersInclude should give true for %#v\n", rosmsg)
	}
	if !extern.PublishersInclude(context, "rips", "teleop_turtle", "rips", "turtlesim") {
		t.Fatalf("Publishers should give true for %#v\n", rosmsg)
	}
	if !extern.PublishersInclude(context, "rips", "turtlesim") {
		t.Fatalf("Publishers should give true for %#v\n", rosmsg)
	}
	if !extern.PublishersInclude(context) {
		t.Fatalf("Publishers should give true for %#v\n", rosmsg)
	}
	if extern.PublishersInclude(context, "rips", "teleop_turtle", "potato", "turtlesim") {
		t.Fatalf("Publishers should give true for %#v\n", rosmsg)
	}
}

func TestPlugin(t *testing.T) {
	context := extern.NewContext(nil, "", 0, os.Stderr, nil)
	rosmsg, err := rosMsg(onemsg1)
	if err != nil {
		t.Fatalf("decoding: %s\n", err)
	}
	msg := extern.NewMsg(&rosmsg)
	context.Update(msg)
	if extern.Plugin(context, "/usr/bin/false") {
		t.Fatalf("executing false should give false")
	}
	if !extern.Plugin(context, "/usr/bin/true") {
		t.Fatalf("executing true should give true")
	}
	script := "./examples/scripts/plug.sh"
	out := "/tmp/plug.io"
	os.Remove(out)
	if !extern.Plugin(context, "./examples/scripts/plug.sh") {
		t.Fatalf("executing %s should give true", script)
	}
	f, err := os.Open(out)
	if err != nil {
		t.Fatalf("%s opening problem %s", out, err)
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatalf("%s reading problem %s", out, err)
	}
	m, err := context.CurrentMsg.RawMsg()
	if err != nil {
		t.Fatalf("cannot decode onemsg1: %s", err)
	}
	if !bytes.Equal(b, m) {
		t.Fatalf("should be equal\n\t%x\n\t%x\n", b, m)
	}
}

func TestTopicMatches(t *testing.T) {
	context := extern.NewContext(nil, "", 0, os.Stderr, nil)
	rosmsg, err := rosMsg(onemsg1)
	if err != nil {
		t.Fatalf("decoding: %s\n", err)
	}
	msg := extern.NewMsg(&rosmsg)
	context.Update(msg)
	if !extern.TopicMatches(context, "pose", nil) {
		t.Fatalf("TestTopicMatches should give true for %#v\n", rosmsg)
	}
	if !extern.TopicMatches(context, "/turtle1/pose", nil) {
		t.Fatalf("TestTopicMatches should give true for %#v\n", rosmsg)
	}
	if !extern.TopicMatches(context, ".*pose.*", nil) {
		t.Fatalf("TestTopicMatches should give true for %#v\n", rosmsg)
	}
	if !extern.TopicMatches(context, "[/].*pose.*", nil) {
		t.Fatalf("TestTopicMatches should give true for %#v\n", rosmsg)
	}
	if extern.TopicMatches(context, ".*patata.*", nil) {
		t.Fatalf("TestTopicMatches should give false for %#v\n", rosmsg)
	}
	if extern.TopicMatches(context, "^[^/].*pose.*", nil) {
		t.Fatalf("TestTopicMatches should give false for %#v\n", rosmsg)
	}
}
