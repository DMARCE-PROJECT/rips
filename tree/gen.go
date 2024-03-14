package tree

import (
	"fmt"
	"rips/rips/lex"
	"rips/rips/types"
)

const varPrefix = "rul_rips_user_var_"
const levelPrefix = "rul_rips_user_level_"

func (prog *Prog) Format(f fmt.State, verb rune) {
	str := ""
	switch verb {
	case 's':
		str = prog.String()
	case 'g':
		str = prog.Gen() //gen in go
	default:
		str = fmt.Sprintf("!%%%c: %s", verb, str)
	}
	f.Write([]byte(str))
}

func (prog *Prog) Gen() (s string) {
	s += GoPrelude
	for _, v := range prog.Env {
		switch v.SType {
		case SVar:
			s += fmt.Sprintf("var %g = %g\n", (*USym)(v), (*USym)(v.Val))
		}
	}
	s += "//Levels:\n"
	for _, ls := range prog.Levels {
		s += fmt.Sprintf("const %g = int64(%d)\n", (*USym)(ls), ls.SLevel)
	}
	//use map so we cannot get out of bounds
	// TODO could generate a function to translate (FASTER)
	s += fmt.Sprintf("var levelNames = map[int64]string{\n")
	for _, ls := range prog.Levels {
		s += fmt.Sprintf("\t%d: \"%s\",\n", ls.SLevel, ls.Name)
	}
	s += fmt.Sprintf("}\n")
	//use map so we cannot get out of bounds
	// TODO could generate a function to translate (FASTER)
	s += fmt.Sprintf("var isSoftLevel = map[int64]bool{\n")
	for _, ls := range prog.Levels {
		s += fmt.Sprintf("\t%d: %v,\n", ls.SLevel, ls.IsSoft)
	}
	s += fmt.Sprintf("}\n")
	s += fmt.Sprintf("//Predefvars:\n")
	s += "var Uptime = int64(0)\n"
	s += "var Time = int64(0)\n"
	s += fmt.Sprintf("var CurrLevel = int64(%d)\n", prog.Levels[0].SLevel)

	s += GoMiddle
	s += fmt.Sprintf("levelNames = levelNames\n")
	s += fmt.Sprintf("//Rules:\n")
	i := 0
	s += fmt.Sprintf("\ttm := \"External\"\n")
	s += fmt.Sprintf("\tif context.CurrentMsg != nil {\n")
	s += fmt.Sprintf("\t\ttm = context.CurrentMsg.Type()\n\t}\n")
	for _, rs := range prog.RuleSects {
		s2, j := rs.Gen(i)
		s += s2
		i += j + 1
	}
	s += GoEpilogue
	return s
}
func (ruledecl *RuleSect) Gen(i int) (s string, j int) {
	s += fmt.Sprintf("//\tSection %s:\n", ruledecl.SectId)
	stag := fmt.Sprintf("DoneSect%d", i)
	s += fmt.Sprintf("\tif \"%s\" != tm {goto %s}\n", ruledecl.SectId.Name, stag)

	i++
	for _, r := range ruledecl.Rules {
		s2 := r.Gen(i)
		s += s2
		i++
	}
	s += fmt.Sprintf("\n%s:\n\n", stag)
	i++
	return s, i
}
func (rule *Rule) Gen(i int) (s string) {
	s += fmt.Sprintf("\tif %g {\n", (*USym)(rule.Expr))
	tt := "\t\t"
	tt += "\t"
	s += tt + "iscomma := true; iscomma = iscomma\n"
	s += tt + "tokthen := false; tokthen = tokthen\n"
	s += tt + "issuccess := true; issuccess = issuccess\n"
	s += tt + "donext := true; donext = donext\n"
	isfst := true
	for _, a := range rule.Actions {
		s += a.Gen(isfst, i)
		isfst = false
	}
	s += fmt.Sprintf("\nDone%d:\n\n", i)
	return s + "\t}\n"
}
func (action *Action) Gen(isfst bool, i int) (s string) {
	tt := "\t\t\t"
	if !isfst {
		switch (lex.TokType)(action.Con) {
		case lex.TokThen:
			s += tt + "tokthen = true\n"
		case lex.TokNThen:
		case lex.TokComma:
			s += tt + "iscomma = true\n"
		}
	}
	s += tt + fmt.Sprintf("issuccess = %g\n", (*USym)(action.What))
	condstr := "issuccess && tokthen || iscomma || !issuccess && !tokthen"
	s += tt + fmt.Sprintf("if !(%s) { goto Done%d }\n", condstr, i)
	return s
}

var builtinNames = map[string]string{
	"TopicMatches":            "TopicMatches",
	"set":                     "Set",
	"trigger":                 "Trigger",
	"alert":                   "Alert",
	"exec":                    "Exec",
	"True":                    "True",
	"False":                   "False",
	"crash":                   "Crash",
	"msgsubtype":              "MsgSubtype",
	"msgtypein":               "MsgTypeIn",
	"payload":                 "Payload",
	"plugin":                  "Plugin",
	"publishercount":          "PublisherCount",
	"publishers":              "Publishers",
	"publishersinclude":       "PublishersInclude",
	"subscribercount":         "SubscriberCount",
	"subscribers":             "Subscribers",
	"subscribersinclude":      "SubscribersInclude",
	"topicin":                 "Topicin",
	"topicmatches":            "TopicMatches",
	"nodecount":               "NodeCount",
	"nodes":                   "Nodes",
	"nodesinclude":            "NodesInclude",
	"service":                 "Service",
	"servicecount":            "ServiceCount",
	"services":                "Services",
	"servicesinclude":         "ServicesInclude",
	"topiccount":              "TopicCount",
	"topicpublishercount":     "TopicPublisherCount",
	"topicpublishers":         "TopicPublishers",
	"topicpublishersinclude":  "TopicPublishersInclude",
	"topics":                  "Topics",
	"topicsinclude":           "TopicsInclude",
	"topicsubscribercount":    "TopicSubscriberCount",
	"topicsubscribers":        "TopicSubscribers",
	"topicsubscribersinclude": "TopicSubscribersInclude",
	"signal":                  "Signal",
	"idsalert":                "IdsAlert",
	"string":                  "String",
}

func prvars(args []*Sym) (str string) {
	for i, a := range args {
		as := fmt.Sprintf("%g", (*USym)(a))
		str += as
		if i < len(args)-1 {
			str += ", "
		}
	}
	return str
}

func fmtvars(args []*Sym) (str string) {
	str = "["
	for i, a := range args {
		switch a.DataType.TVal {
		case types.TypeVals[types.TVInt]:
			str += "%%d"
		case types.TypeVals[types.TVFloat]:
			str += "%%f"
		case types.TypeVals[types.TVBool]:
			str += "%%v"
		case types.TypeVals[types.TVString]:
			str += `\"%%s\"`
		default:
			str += "%%v"
		}
		if i < len(args)-1 {
			str += " "
		}
	}
	str += "]"
	return str
}

func (s *USym) GoString() (str string) {
	if s == nil {
		return "nil"
	}
	switch s.SType {
	case SConst:
		switch s.DataType.TVal {
		case types.TypeVals[types.TVInt]:
			str = fmt.Sprintf("int64(%d)", s.IntVal)
		case types.TypeVals[types.TVFloat]:
			str = fmt.Sprintf("float64(%f)", s.FloatVal)
		case types.TypeVals[types.TVBool]:
			str = fmt.Sprintf("%v", s.BoolVal)
		case types.TypeVals[types.TVString]:
			str = fmt.Sprintf("%q", s.StrVal)
		default:
			str = "?"
		}
	case SVar:
		if s.Name != "CurrLevel" && s.Name != "Time" && s.Name != "Uptime" {
			str = varPrefix
		}
		str += s.Name
	case SFCall:
		if s.Name == "levelname" {
			//HACK, should probably use an extern and context.Levels
			str += fmt.Sprintf(`levelNames[%g]`, (*USym)(s.Expr.Args[0]))
			return
		}
		fname := "false"
		switch s.Name {
		case "True":
			fname = "true"
			fallthrough
		case "False":
			const funcfmthead = `func()bool{fmt.Fprintf(os.Stderr, "%s call, `
			const funcfmttail = `\n", %s);return %s}()`
			fmtstr := funcfmthead + fmtvars(s.Expr.Args) + funcfmttail
			str += fmt.Sprintf(fmtstr, fname, prvars(s.Expr.Args), fname)
			return
		}
		if s.Name == "set" {
			lval := s.Expr.Args[0]
			rval := s.Expr.Args[1]
			str += fmt.Sprintf("func()bool{%g = %g; return true}()", (*USym)(lval), (*USym)(rval))
			return
		}
		if s.Name == "trigger" {
			levelname := s.Expr.Args[0].Name
			str += fmt.Sprintf(`extern.Trigger(context, "%s", int(%g),`, levelname, (*USym)(s.Expr.Args[0]))
			str += fmt.Sprintf("levelNames[int64(CurrLevel)], int(CurrLevel), isSoftLevel[int64(CurrLevel)]);")

			str += fmt.Sprintf("CurrLevel = context.CurrLevel\n")
			return
		}
		bn, ok := builtinNames[s.Name]
		if !ok {
			panic("bad name " + s.Name)
		}
		str = fmt.Sprintf("extern.%s(context, ", bn)
		str += prvars(s.Expr.Args)
		str += ")"
	case SFunc:
		str += fmt.Sprintf("%s(%d args)", s.Name, len(s.ArgDataTypes))
	case SLevel:
		str = levelPrefix + s.Name
	case SAsign:
		str = (*USym)(s.Asign.LVal).GoString() + " = " + (*USym)(s.Asign.RVal).GoString()
	case SBinary:
		str = "(" + (*USym)(s.Expr.ELeft).GoString() + " "
		str += lex.UTokType(s.Expr.Op).String() + " "
		str += (*USym)(s.Expr.ERight).GoString() + ")"
	case SUnary:
		str = "(" + lex.UTokType(s.Expr.Op).String() + (*USym)(s.Expr.ERight).GoString() + ")"
	case SRegexp:
		str += fmt.Sprintf("\"%s\", lookupRegexp(\"%s\")", s.StrVal, s.StrVal)
	case SYara:
		str += fmt.Sprintf("\"%s\", lookupYara(\"%s\")", s.StrVal, s.StrVal)
	}
	return str
}
