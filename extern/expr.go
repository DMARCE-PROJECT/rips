package extern

import (
	"fmt"
	"github.com/kgwinnup/go-yara/yara"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const DebExpr = true

func dprintfExpr(s string, v ...interface{}) {
	if !DebExpr {
		return
	}
	fmt.Fprintf(os.Stderr, "Extern: "+s, v...)
}

//Message expressions

func MsgSubtype(context *Ctx, msgtype string, msgsubtype string) bool {
	dprintfExpr("MsgSubtype: %s %s\n", msgtype, msgsubtype)
	topic := context.CurrentMsg.rosm.FromTopic
	rostype := context.RosType(topic)
	rossubtype := context.RosSubtype(topic)
	dprintfExpr("MsgSubtype: ->  ros says topic[%s]: %s %s\n", topic, rostype, rossubtype)
	return rostype == msgtype && rossubtype == msgsubtype
}

func isEmpty(as []string) bool {
	if len(as) == 0 {
		return true
	}
	//decoding artifact, empty may be one empty string!!!
	return len(as) == 1 && as[0] == ""
}

// may be repeated, etc.
// this is the best aproximation
func matchSubset(as []string, subset []string) bool {
	//empty set
	if isEmpty(subset) {
		return true
	}
	amap := make(map[string]bool, len(as))
	for _, a := range as {
		amap[a] = true
	}
	for _, b := range subset {
		if !amap[b] {
			return false
		}
	}
	return true
}

// may be repeated, etc.
// What should be done if a=("a", "a") and b=("a")?
// they are also not ordered
func matchAll(as []string, bs []string) bool {
	return matchSubset(as, bs) && matchSubset(bs, as)
}

func MsgTypeIn(context *Ctx, msgtypes ...string) bool {
	dprintfExpr("expression MsgTypeIn: %s\n", msgtypes)
	topic := context.CurrentMsg.rosm.FromTopic
	rostype := context.RosType(topic)
	if len(msgtypes) == 0 {
		return true
	}
	for _, t := range msgtypes {
		if rostype == t {
			return true
		}
	}
	return false
}

func Plugin(context *Ctx, path string) bool {
	dprintfExpr("expression Plugin: %s\n", path)
	m, err := context.CurrentMsg.RawMsg()
	if err != nil {
		context.Printf("Plugin: decode error: %s", err)
		return false
	}
	cmd := exec.Command(path)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		context.Printf("Plugin: could not create command: %s", err)
		return false
	}

	err = cmd.Start()
	if err != nil {
		context.Printf("Plugin: error, cannot start command: %s\n", err)
		return false
	}
	go func() {
		defer stdin.Close()
		stdin.Write(m)
	}()

	sts := 0
	if err := cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			sts = exiterr.ExitCode()
		} else {
			context.Printf("Plugin: weird error: %s\n", err)
		}
	}
	if err != nil {
		context.Printf("Plugin: error: %s\n", err)
	}
	dprintfExpr("-> Plugin: status: %d -> %v\n", sts, sts == 0)
	return sts == 0
}

func Payload(context *Ctx, pathrule string, yr *yara.Yara) bool {
	dprintfExpr("Payload rule: %s\n", pathrule)
	m, err := context.CurrentMsg.RawMsg()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Payload: decode error: %s", err)
		return true
	}
	if yr == nil {
		rule, err := ioutil.ReadFile(pathrule)
		if err != nil {
			panic("corrupted yara rule " + pathrule)
		}
		yr, err = yara.New(string(rule))
		if err != nil {
			panic("corrupted yara rule " + pathrule)
		}
	}
	dprintfExpr("payload msg: %x\n", m)
	output, err := yr.Scan(m, 3, true) //second param is showstring
	if err != nil {
		fmt.Fprintf(os.Stderr, "Payload: scan rule %s: %v\n", pathrule, err)
		return true
	}
	if len(output) == 0 {
		dprintfExpr("payload rule: %s OK\n", pathrule)
		return false
	}
	fmt.Fprintf(os.Stderr, "Payload: positive in %s\n", pathrule)
	for _, obj := range output {
		fmt.Fprintf(os.Stderr, "\tRule: %s, %s\n", obj.Name, strings.Join(obj.Tags, ","))
	}
	dprintfExpr("payload rule: %s NOT OK\n", pathrule)
	return true
}
func PublisherCount(context *Ctx, min int64, max int64) bool {
	dprintfExpr("expression PublisherCount: %d:%d\n", min, max)
	topic := context.CurrentMsg.Topic()
	pubs := context.Publishers(topic)
	dprintfExpr("expression PublisherCount: ros says %d\n", len(pubs))
	dprintfExpr("-> PublisherCount: ros says %d\n", len(pubs))
	return int64(len(pubs)) >= min && int64(len(pubs)) <= max
}

// TODO PUBLISER (and similar) convert
//
//	some sets of strings into maps in compilation
//	look for matchALL
func Publishers(context *Ctx, pubs ...string) bool {
	dprintfExpr("expression Publishers: %s\n", pubs)
	topic := context.CurrentMsg.Topic()
	contextpubs := context.Publishers(topic)
	dprintfExpr("-> Publishers: ros says %s\n", contextpubs)
	return matchAll(pubs, contextpubs)
}

// TODO (PUBLISHERSINCLUDE and similar) convert
//
//	some sets of strings into maps in compilation
//	look for matchSubset
func PublishersInclude(context *Ctx, pubs ...string) bool {
	dprintfExpr("PublishersInclude: %s\n", pubs)
	//empty set, is included
	topic := context.CurrentMsg.Topic()
	contextpubs := context.Publishers(topic)
	dprintfExpr("-> PublishersInclude: ros says %s\n", contextpubs)
	return matchSubset(contextpubs, pubs)
}

//Unimplementable
//func SenderIn(context *Ctx, senders ...string) bool {}

func SubscriberCount(context *Ctx, min int64, max int64) bool {
	dprintfExpr("SubscriberCount: %d:%d\n", min, max)
	topic := context.CurrentMsg.Topic()
	subs := context.Subscribers(topic)
	dprintfExpr("-> SubscriberCount: ros says %s\n", len(subs))
	return int64(len(subs)) >= min && int64(len(subs)) <= max
}

func Subscribers(context *Ctx, subs ...string) bool {
	dprintfExpr("expression Subscribers: %s\n", subs)
	topic := context.CurrentMsg.Topic()
	contextsubs := context.Subscribers(topic)
	dprintfExpr("-> Subscribers: ros says %s\n", contextsubs)
	return matchAll(subs, contextsubs)
}
func SubscribersInclude(context *Ctx, subs ...string) bool {
	dprintfExpr("SubscribersInclude: %s\n", subs)
	//empty set, is included
	topic := context.CurrentMsg.Topic()
	contextsubs := context.Subscribers(topic)
	dprintfExpr("-> SubscribersInclude: ros says %s\n", contextsubs)
	return matchSubset(subs, contextsubs)
}
func TopicIn(context *Ctx, topics ...string) (ispres bool) {
	dprintfExpr("expression TopicIn: %s\n", topics)
	for _, t := range topics {
		if context.CurrentMsg.Topic() == t {
			return true
		}
	}
	return false
}

// https://github.com/google/re2/wiki/Syntax
func TopicMatches(context *Ctx, restr string, re *regexp.Regexp) (ism bool) {
	topic := context.CurrentMsg.Topic()
	dprintfExpr("expression TopicMatches: %s for %s\n", restr, topic)

	//in case it was not precompiled
	if re == nil {
		ism, _ = regexp.MatchString(restr, topic)
	} else {
		ism = re.MatchString(topic)
	}
	dprintfExpr("expression TopicMatches did it match?: %s for %s: %v\n", restr, topic, ism)
	return ism
}

// Graph Expressions

func NodeCount(context *Ctx, min int64, max int64) bool {
	dprintfExpr("expression NodeCount: %d:%d\n", min, max)
	ns := context.Nodes()
	dprintfExpr("-> NodeCount: ros says %d\n", len(ns))
	return int64(len(ns)) >= min && int64(len(ns)) <= max
}

func nodeNames(nodes []RosNode) (names []string) {
	for _, n := range nodes {
		names = append(names, n.Node)
	}
	return names
}

func Nodes(context *Ctx, nodes ...string) bool {
	dprintfExpr("expression Nodes: %s\n", nodes)
	contextns := context.Nodes()
	nodenames := nodeNames(contextns)
	return matchAll(nodes, nodenames)
}

func NodesInclude(context *Ctx, nodes ...string) bool {
	dprintfExpr("NodesInclude: %s\n", nodes)
	contextns := context.Nodes()
	nodenames := nodeNames(contextns)
	return matchSubset(nodenames, nodes)
}
func Service(context *Ctx, node string, service string) bool {
	dprintfExpr("expression Services: %s\n", service)
	n := context.RosNode(node)
	return n.HasService(service)
}

func ServiceCount(context *Ctx, node string, min int64, max int64) bool {
	dprintfExpr("expression ServiceCount: %d:%d\n", min, max)
	n := context.RosNode(node)
	return int64(len(n.Services)) >= min && int64(len(n.Services)) <= max
}

func serviceNames(ss []RosService) (names []string) {
	for _, s := range ss {
		names = append(names, s.Service)
	}
	return names
}

func Services(context *Ctx, node string, services ...string) bool {
	dprintfExpr("expression Services: %s\n", services)
	n := context.RosNode(node)
	servicesnames := serviceNames(n.Services)
	return matchAll(services, servicesnames)
}

func ServicesInclude(context *Ctx, node string, services ...string) bool {
	dprintfExpr("expression ServicesInclude: %s\n", services)
	n := context.RosNode(node)
	servicesnames := serviceNames(n.Services)
	return matchSubset(servicesnames, services)
}

func TopicCount(context *Ctx, min int64, max int64) bool {
	dprintfExpr("TopicCount: %d:%d\n", min, max)
	contexttopics := context.Topics()
	return int64(len(contexttopics)) >= min && int64(len(contexttopics)) <= max
}

func TopicPublisherCount(context *Ctx, topic string, min int64, max int64) bool {
	dprintfExpr("expression TopicPublisherCount[%s]: %d:%d\n", topic, min, max)
	pubs := context.Publishers(topic)
	return int64(len(pubs)) >= min && int64(len(pubs)) <= max
}

func TopicPublishers(context *Ctx, topic string, pubs ...string) bool {
	dprintfExpr("expression TopicPublishers[%s]: %s\n", topic, pubs)
	contextpubs := context.Publishers(topic)
	return matchAll(pubs, contextpubs)
}

func TopicPublishersInclude(context *Ctx, topic string, pubs ...string) bool {
	dprintfExpr("expression TopicPublishersInclude[%s]: %s\n", topic, pubs)
	contextpubs := context.Publishers(topic)
	return matchSubset(contextpubs, pubs)
}

func Topics(context *Ctx, topics ...string) bool {
	dprintfExpr("expression Topics: %s\n", topics)
	contexttopics := context.Topics()
	return matchAll(topics, contexttopics)
}

func TopicsInclude(context *Ctx, topics ...string) bool {
	dprintfExpr("expression TopicsInclude: %s\n", topics)
	contexttopics := context.Topics()
	return matchSubset(contexttopics, topics)
}

func TopicSubscriberCount(context *Ctx, topic string, min int64, max int64) bool {
	dprintfExpr("expression TopicSubscriberCount[%s]: %d:%d\n", topic, min, max)
	subs := context.Subscribers(topic)
	return int64(len(subs)) >= min && int64(len(subs)) <= max
}

func TopicSubscribers(context *Ctx, topic string, subs ...string) bool {
	dprintfExpr("expression TopicSubscribers[%s]: %s\n", topic, subs)
	contextsubs := context.Subscribers(topic)
	return matchAll(subs, contextsubs)
}

func TopicSubscribersInclude(context *Ctx, topic string, subs ...string) bool {
	dprintfExpr("expression TopicSubscribersInclude[%s]: %s\n", topic, subs)
	contextsubs := context.Subscribers(topic)
	return matchSubset(contextsubs, subs)
}

func Signal(context *Ctx, sig string) bool {
	dprintfExpr("expression Signal[%s]: (%d, %d)\n", sig, context.Nusr1, context.Nusr1)
	switch sig {
	case "SIGUSR1":
		if context.Nusr1 > 0 {
			context.Nusr1--
			return true
		}
	case "SIGUSR2":
		if context.Nusr2 > 0 {
			context.Nusr2--
			return true
		}
	}
	return false
}

func IdsAlert(context *Ctx, alert string) bool {
	dprintfExpr("expression IdsAlert[%s]\n", alert)
	isalert := FgrepAll(context.Paths, alert)
	return isalert
}

// may be evaluated statically
func String(context *Ctx, a any) string {
	return fmt.Sprintf("%v", a)
}
