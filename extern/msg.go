package extern

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"strings"
)

// go get gopkg.in/yaml.v2

type RosService struct {
	Service string
	Params  []string
}

func tabs(n int) string {
	return strings.Repeat("\t", n)
}

func (rs *RosService) String() (s string) {
	ntabs := 4
	s += tabs(ntabs) + fmt.Sprintf("srv: \"%s\" params[\n", rs.Service)
	for _, p := range rs.Params {
		s += tabs(ntabs+1) + fmt.Sprintf("\"%s\",\n", p)
	}
	return s + tabs(ntabs) + "]\n"
}

type RosTopic struct {
	Topic       string
	Parameters  []string
	Publishers  []string
	Subscribers []string
}

func (rn *RosTopic) String() (s string) {
	ntabs := 4
	s += tabs(ntabs) + fmt.Sprintf("topic: %s \n", rn.Topic)
	s += tabs(ntabs) + fmt.Sprintf("parameters[\n")
	for _, p := range rn.Parameters {
		s += tabs(ntabs+1) + fmt.Sprintf("\"%s\",\n", p)
	}
	s += tabs(ntabs) + "]\n"
	s += tabs(ntabs) + "publishers: [\n"
	for _, pubr := range rn.Publishers {
		s += tabs(ntabs+1) + fmt.Sprintf("\"%s\",\n", pubr)
	}
	s += tabs(ntabs) + "]\n"
	s += tabs(ntabs) + "subscribers: [\n"
	for _, sr := range rn.Subscribers {
		s += tabs(ntabs+1) + fmt.Sprintf("\"%s\",\n", sr)
	}
	s += tabs(ntabs) + "]\n"
	return s
}

type RosNode struct {
	Node     string
	Gids     []string
	Services []RosService //translate to map for faster access?
}

func (rn *RosNode) String() (s string) {
	ntabs := 2
	s += tabs(ntabs) + fmt.Sprintf("node: %s", rn.Node)
	ntabs += 2
	s += tabs(ntabs) + fmt.Sprintf("gids[\n")
	for _, n := range rn.Gids {
		s += tabs(ntabs+1) + fmt.Sprintf("\"%s\",\n", n)
	}
	s += tabs(ntabs) + "]\n"
	s += tabs(ntabs) + "services: [\n"
	for _, sr := range rn.Services {
		s += tabs(ntabs+1) + fmt.Sprintf("\"%s\",\n", sr)
	}
	s += tabs(ntabs) + "]\n"
	return s
}

type RosContext struct {
	Nodes  []RosNode  //translate to map for faster access?
	Topics []RosTopic //translate to map for faster access?
}

func (rc *RosContext) String() (s string) {
	ntabs := 2
	s += "\n" + tabs(ntabs) + "Nodes: [\n"
	for _, rn := range rc.Nodes {
		s += fmt.Sprintf("\t%s\n", &rn)
	}
	s += "\n" + tabs(ntabs) + "]\n"
	s += "\n" + tabs(ntabs) + "Nodes: [\n"
	for _, rt := range rc.Topics {
		s += tabs(ntabs+1) + fmt.Sprintf("%s\n", &rt)
	}
	s += "\n" + tabs(ntabs) + "]\n"
	return s
}

type RosMsg struct {
	Event     string
	FromTopic string
	RawMsg    string
	Msg       any //it is a map of maps of maps... with strings.
	Context   RosContext
}

func (rm *RosMsg) String() (s string) {
	s += fmt.Sprintf("event \"%s\"\n", rm.Event)
	s += fmt.Sprintf("\tfromtopic \"%s\"\n", rm.FromTopic)
	s += fmt.Sprintf("\trawmsg: \"%s\",\n", rm.RawMsg)
	s += fmt.Sprintf("\tmsg: \"%v\",\n", rm.Msg)
	s += fmt.Sprintf("\tcontext {%s\t},\n", &rm.Context)
	return s
}

type RosDecoder struct {
	d *yaml.Decoder
}

func (dec *RosDecoder) Decode(v any) (err error) {
	return dec.d.Decode(v)
}

const bufDecoderSz = 128 * 1024

func NewRosDecoder(rd io.Reader) (rosd *RosDecoder) {
	brd := bufio.NewReaderSize(rd, bufDecoderSz)
	dec := yaml.NewDecoder(brd)
	dec.SetStrict(false)
	return &RosDecoder{d: dec}
}

type Msg struct {
	rosm *RosMsg
}

var typemsgs = map[string]string{
	"graph":   "Graph",
	"message": "Msg",
}

func (m *Msg) Type() (t string) {
	t, isok := typemsgs[m.rosm.Event]
	if !isok {
		return "Unknown" //will not run any rule, should there be a default rule?
	}
	return t
}

func (m *Msg) Topic() (t string) {
	return m.rosm.FromTopic
}

func (m *Msg) RawMsg() (b []byte, err error) {
	rawmsg := m.rosm.RawMsg
	dst := make([]byte, base64.StdEncoding.DecodedLen(len(rawmsg)))
	n, err := base64.StdEncoding.Decode(dst, []byte(rawmsg))
	if err != nil {
		return nil, err
	}
	return dst[:n], nil
}

func NewMsg(rosm *RosMsg) (m *Msg) {
	return &Msg{rosm}
}
