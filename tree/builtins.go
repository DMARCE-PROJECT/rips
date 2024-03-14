package tree

import (
	"fmt"
	"os"
	"rips/rips/extern"
	"rips/rips/types"
)

// When enabled True and False will print an internal representation with
const DebTrue = false
const DebSet = false

// Take care, if IsVariadic and there is a mandatory argument,
//
//	add two arguments because Variadic may mean zero args.
type Builtin struct {
	Name       string
	Fn         func(context *extern.Ctx, args ...*Sym) *Sym
	RetType    types.Type
	ArgTypes   []types.Type
	IsVariadic bool //last argtype is repeated, may be zero
	IsAction   bool
}

func (envs *StkEnv) Builtins(bs []*Builtin) {
	for _, b := range bs {
		_, err := envs.NewFunc(b.Name, b.ArgTypes, b.RetType, b.Fn, b.IsVariadic, b.IsAction)
		if err != nil {
			s := fmt.Sprintf("problem pushing builtin \"%s\": %s\n", b.Name, err)
			panic(s)
		}
	}
}

// Normal Expressions
func LevelName(context *extern.Ctx, args ...*Sym) *Sym {
	level := args[0]
	if args[0] == nil {
		panic("levelname: cannot happen")
	}
	str := ""
	if level.IntVal >= 0 && int(level.IntVal) < context.NLevels {
		str = context.Levels[level.IntVal]
	}
	return NewString(str)
}
func String(context *extern.Ctx, args ...*Sym) *Sym {
	s := args[0]
	str := ""
	switch s.DataType.TVal {
	case types.TypeVals[types.TVBool]:
		str = extern.String(context, s.BoolVal)
	case types.TypeVals[types.TVInt]:
		str = extern.String(context, s.IntVal)
	case types.TypeVals[types.TVFloat]:
		str = extern.String(context, s.FloatVal)
	case types.TypeVals[types.TVString]:
		str = extern.String(context, s.StrVal)
	default:
		str = "Unk_Type"
	}
	return NewString(str)
}

// Actions
// Set is special, the implementation is here
func Set(context *extern.Ctx, args ...*Sym) *Sym {
	_ = context //does not use
	args[0].SetVal(args[1])
	if DebSet {
		fmt.Fprintf(os.Stderr, "set call, %s\n", args)
	}
	return NewBool(true)
}

// False and True are for debugging, they are also implementations
// They return (false, true) and they print their syms
func True(context *extern.Ctx, args ...*Sym) *Sym {
	_ = context //does not use
	if DebTrue {
		fmt.Fprintf(os.Stderr, "true call, %s\n", args)
	} else {
		s := "true call, ["
		for _, a := range args {
			if a.SType == SLevel {
				s += fmt.Sprintf("%d ", a.SLevel)
				continue
			}
			s += fmt.Sprintf("%s ", (*USym)(a))
		}
		fmt.Fprintf(os.Stderr, "%s\n", s[0:len(s)-1]+"]")
	}
	return NewBool(true)
}
func False(context *extern.Ctx, args ...*Sym) *Sym {
	_ = context //does not use
	if DebTrue {
		fmt.Fprintf(os.Stderr, "false call, %s\n", args)
	} else {
		s := "true call, "
		for _, a := range args {
			if a.SType == SLevel {
				s += fmt.Sprintf("%d ", a.SLevel)
				continue
			}
			s += fmt.Sprintf("%s ", (*USym)(a))
		}
		fmt.Fprintf(os.Stderr, "%s\n", s[0:len(s)-1])
	}
	return NewBool(false)
}

//From now on, the functions delegate to the extern package

// action for crashing the program
func Crash(context *extern.Ctx, args ...*Sym) *Sym {
	extern.Crash(context, args[0].StrVal) //should not return
	return nil
}

func Alert(context *extern.Ctx, args ...*Sym) *Sym {
	v := extern.Alert(context, args[0].StrVal)
	return NewBool(v)
}

func Exec(context *extern.Ctx, args ...*Sym) *Sym {
	v := extern.Exec(context, args[0].StrVal, VarArgs(args[1:]...)...)
	return NewBool(v)
}

// In trigger the arg is an SLevel symbol, slevel.SLevel is the int identifying it
func Trigger(context *extern.Ctx, args ...*Sym) *Sym {
	currlevel := args[0]
	args = args[1:]
	if currlevel == nil || currlevel.Val == nil {
		panic("no current level, cannot happen")
	}
	slevelto := args[0]
	// same is nop
	v := extern.Trigger(context, slevelto.Name, slevelto.SLevel, currlevel.Val.Name, currlevel.Val.SLevel, currlevel.Val.IsSoft)
	if v {
		currlevel.Val = slevelto
	} else {
		context.Printf("trigger: could not change level %s[%d] -> %s[%d]\n", currlevel.Val.Name, currlevel.Val.SLevel, slevelto.Name, slevelto.SLevel)
	}
	return NewBool(v)
}

//Expressions

// Msg expressions
func VarArgs(args ...*Sym) (strargs []string) {
	for _, arg := range args {
		strargs = append(strargs, arg.StrVal)
	}
	return strargs
}
func MsgSubtype(context *extern.Ctx, args ...*Sym) *Sym {
	v := extern.MsgSubtype(context, args[0].StrVal, args[1].StrVal)
	return NewBool(v)
}

func MsgTypeIn(context *extern.Ctx, args ...*Sym) *Sym {
	v := extern.MsgTypeIn(context, VarArgs(args...)...)
	return NewBool(v)
}
func Payload(context *extern.Ctx, args ...*Sym) *Sym {
	v := extern.Payload(context, args[0].StrVal, args[0].Yr)
	return NewBool(v)
}
func Plugin(context *extern.Ctx, args ...*Sym) *Sym {
	v := extern.Plugin(context, args[0].StrVal)
	return NewBool(v)

}
func PublisherCount(context *extern.Ctx, args ...*Sym) *Sym {
	v := extern.PublisherCount(context, args[0].IntVal, args[1].IntVal)
	return NewBool(v)
}
func Publishers(context *extern.Ctx, args ...*Sym) *Sym {
	v := extern.Publishers(context, VarArgs(args...)...)
	return NewBool(v)

}
func PublishersInclude(context *extern.Ctx, args ...*Sym) *Sym {
	v := extern.PublishersInclude(context, VarArgs(args...)...)
	return NewBool(v)
}

//Unimplementable
//func SenderIn(context *extern.Ctx, args ...*Sym) *Sym {}

func SubscriberCount(context *extern.Ctx, args ...*Sym) *Sym {
	v := extern.SubscriberCount(context, args[0].IntVal, args[1].IntVal)
	return NewBool(v)
}
func Subscribers(context *extern.Ctx, args ...*Sym) *Sym {
	v := extern.Subscribers(context, VarArgs(args...)...)
	return NewBool(v)
}
func SubscribersInclude(context *extern.Ctx, args ...*Sym) *Sym {
	v := extern.SubscribersInclude(context, VarArgs(args...)...)
	return NewBool(v)
}
func TopicIn(context *extern.Ctx, args ...*Sym) *Sym {
	v := extern.TopicIn(context, VarArgs(args...)...)
	return NewBool(v)
}
func TopicMatches(context *extern.Ctx, args ...*Sym) *Sym {
	v := extern.TopicMatches(context, args[0].StrVal, args[0].Re)
	return NewBool(v)
}

// Graph expressions
func NodeCount(context *extern.Ctx, args ...*Sym) *Sym {
	v := extern.NodeCount(context, args[0].IntVal, args[1].IntVal)
	return NewBool(v)
}
func Nodes(context *extern.Ctx, args ...*Sym) *Sym {
	v := extern.Nodes(context, VarArgs(args...)...)
	return NewBool(v)
}
func NodesInclude(context *extern.Ctx, args ...*Sym) *Sym {
	v := extern.NodesInclude(context, VarArgs(args...)...)
	return NewBool(v)
}
func Service(context *extern.Ctx, args ...*Sym) *Sym {
	v := extern.Service(context, args[0].StrVal, args[1].StrVal)
	return NewBool(v)
}
func ServiceCount(context *extern.Ctx, args ...*Sym) *Sym {
	v := extern.ServiceCount(context, args[0].StrVal, args[1].IntVal, args[2].IntVal)
	return NewBool(v)
}
func Services(context *extern.Ctx, args ...*Sym) *Sym {
	v := extern.Services(context, args[0].StrVal, VarArgs(args[1:]...)...)
	return NewBool(v)
}
func ServicesInclude(context *extern.Ctx, args ...*Sym) *Sym {
	v := extern.ServicesInclude(context, args[0].StrVal, VarArgs(args[1:]...)...)
	return NewBool(v)
}
func TopicCount(context *extern.Ctx, args ...*Sym) *Sym {
	v := extern.TopicCount(context, args[0].IntVal, args[1].IntVal)
	return NewBool(v)
}
func TopicPublisherCount(context *extern.Ctx, args ...*Sym) *Sym {
	v := extern.TopicPublisherCount(context, args[0].StrVal, args[1].IntVal, args[2].IntVal)
	return NewBool(v)
}
func TopicPublishers(context *extern.Ctx, args ...*Sym) *Sym {
	v := extern.TopicPublishers(context, args[0].StrVal, VarArgs(args[1:]...)...)
	return NewBool(v)
}
func TopicPublishersInclude(context *extern.Ctx, args ...*Sym) *Sym {
	v := extern.TopicPublishersInclude(context, args[0].StrVal, VarArgs(args[1:]...)...)
	return NewBool(v)
}
func Topics(context *extern.Ctx, args ...*Sym) *Sym {
	v := extern.Topics(context, VarArgs(args...)...)
	return NewBool(v)
}
func TopicsInclude(context *extern.Ctx, args ...*Sym) *Sym {
	v := extern.TopicsInclude(context, VarArgs(args...)...)
	return NewBool(v)
}
func TopicSubscriberCount(context *extern.Ctx, args ...*Sym) *Sym {
	v := extern.TopicSubscriberCount(context, args[0].StrVal, args[1].IntVal, args[2].IntVal)
	return NewBool(v)
}
func TopicSubscribers(context *extern.Ctx, args ...*Sym) *Sym {
	v := extern.TopicSubscribers(context, args[0].StrVal, VarArgs(args[1:]...)...)
	return NewBool(v)
}
func TopicSubscribersInclude(context *extern.Ctx, args ...*Sym) *Sym {
	v := extern.TopicSubscribersInclude(context, args[0].StrVal, VarArgs(args[1:]...)...)
	return NewBool(v)
}

// External expression
func Signal(context *extern.Ctx, args ...*Sym) *Sym {
	v := extern.Signal(context, args[0].StrVal)
	return NewBool(v)
}
func IdsAlert(context *extern.Ctx, args ...*Sym) *Sym {
	v := extern.IdsAlert(context, args[0].StrVal)
	return NewBool(v)
}

var Builtins = []*Builtin{
	//actions
	//set is special, the first argument is not evaluated, etc.
	{Name: "set",
		RetType:    types.BoolType,
		Fn:         Set,
		ArgTypes:   []types.Type{types.UnivType, types.UnivType},
		IsVariadic: false,
		IsAction:   true,
	},
	//trigger is special, the first argument is an SLevel symbol
	{Name: "trigger",
		RetType:    types.BoolType,
		Fn:         Trigger,
		ArgTypes:   []types.Type{types.IntType},
		IsVariadic: false,
		IsAction:   true,
	},
	{Name: "alert",
		RetType:    types.BoolType,
		Fn:         Alert,
		ArgTypes:   []types.Type{types.StringType},
		IsVariadic: false,
		IsAction:   true,
	},
	{Name: "exec",
		RetType:    types.BoolType,
		Fn:         Exec,
		ArgTypes:   []types.Type{types.StringType, types.StringType},
		IsVariadic: true,
		IsAction:   true,
	},
	//for debugging
	{Name: "True",
		RetType:    types.BoolType,
		Fn:         True,
		ArgTypes:   []types.Type{types.UnivType},
		IsVariadic: true,
		IsAction:   true,
	},
	{Name: "False",
		RetType:    types.BoolType,
		Fn:         False,
		ArgTypes:   []types.Type{types.UnivType},
		IsVariadic: true,
		IsAction:   true,
	},
	//internal action for crashing
	{Name: "crash",
		RetType:    types.BoolType,
		Fn:         Crash,
		ArgTypes:   []types.Type{types.StringType},
		IsVariadic: false,
		IsAction:   true,
	},
	//Messages expressions
	{Name: "msgsubtype",
		RetType:    types.BoolType,
		Fn:         MsgSubtype,
		ArgTypes:   []types.Type{types.MsgStrType, types.MsgStrType},
		IsVariadic: false,
		IsAction:   false,
	},
	{Name: "msgtypein",
		RetType:    types.BoolType,
		Fn:         MsgTypeIn,
		ArgTypes:   []types.Type{types.MsgStrType},
		IsVariadic: true,
		IsAction:   false,
	},
	{
		Name:       "payload",
		RetType:    types.BoolType,
		Fn:         Payload,
		ArgTypes:   []types.Type{types.MsgStrType},
		IsVariadic: false,
		IsAction:   false,
	},
	{
		Name:       "plugin",
		RetType:    types.BoolType,
		Fn:         Plugin,
		ArgTypes:   []types.Type{types.MsgStrType},
		IsVariadic: false,
		IsAction:   false,
	},
	{
		Name:       "publishercount",
		RetType:    types.BoolType,
		Fn:         PublisherCount,
		ArgTypes:   []types.Type{types.MsgIntType, types.MsgIntType},
		IsVariadic: false,
		IsAction:   false,
	},
	{
		Name:       "publishers",
		RetType:    types.BoolType,
		Fn:         Publishers,
		ArgTypes:   []types.Type{types.MsgStrType},
		IsVariadic: true,
		IsAction:   false,
	},
	{
		Name:       "publishersinclude",
		RetType:    types.BoolType,
		Fn:         PublishersInclude,
		ArgTypes:   []types.Type{types.MsgStrType},
		IsVariadic: true,
		IsAction:   false,
	},
	{
		Name:       "subscribercount",
		RetType:    types.BoolType,
		Fn:         SubscriberCount,
		ArgTypes:   []types.Type{types.MsgIntType, types.MsgIntType},
		IsVariadic: false,
		IsAction:   false,
	},
	{
		Name:       "subscribers",
		RetType:    types.BoolType,
		Fn:         Subscribers,
		ArgTypes:   []types.Type{types.MsgStrType},
		IsVariadic: true,
		IsAction:   false,
	},
	{
		Name:       "subscribersinclude",
		RetType:    types.BoolType,
		Fn:         SubscribersInclude,
		ArgTypes:   []types.Type{types.MsgStrType},
		IsVariadic: true,
		IsAction:   false,
	},
	{
		Name:       "topicin",
		RetType:    types.BoolType,
		Fn:         TopicIn,
		ArgTypes:   []types.Type{types.MsgStrType},
		IsVariadic: true,
		IsAction:   false,
	},
	{
		Name:       "topicmatches",
		RetType:    types.BoolType,
		Fn:         TopicMatches,
		ArgTypes:   []types.Type{types.MsgStrType},
		IsVariadic: false,
		IsAction:   false,
	},
	//Graph expressions
	{
		Name:       "nodecount",
		RetType:    types.BoolType,
		Fn:         NodeCount,
		ArgTypes:   []types.Type{types.GraphIntType, types.GraphIntType},
		IsVariadic: false,
		IsAction:   false,
	},
	{
		Name:       "nodes",
		RetType:    types.BoolType,
		Fn:         Nodes,
		ArgTypes:   []types.Type{types.GraphStrType},
		IsVariadic: true,
		IsAction:   false,
	},
	{
		Name:       "nodesinclude",
		RetType:    types.BoolType,
		Fn:         NodesInclude,
		ArgTypes:   []types.Type{types.GraphStrType},
		IsVariadic: true,
		IsAction:   false,
	},
	{
		Name:       "service",
		RetType:    types.BoolType,
		Fn:         Service,
		ArgTypes:   []types.Type{types.GraphStrType},
		IsVariadic: false,
		IsAction:   false,
	},
	{
		Name:       "servicecount",
		RetType:    types.BoolType,
		Fn:         ServiceCount,
		ArgTypes:   []types.Type{types.GraphIntType, types.GraphIntType},
		IsVariadic: false,
		IsAction:   false,
	},
	{
		Name:       "services",
		RetType:    types.BoolType,
		Fn:         Services,
		ArgTypes:   []types.Type{types.GraphStrType},
		IsVariadic: true,
		IsAction:   false,
	},
	{
		Name:       "servicesinclude",
		RetType:    types.BoolType,
		Fn:         ServicesInclude,
		ArgTypes:   []types.Type{types.GraphStrType},
		IsVariadic: true,
		IsAction:   false,
	},
	{
		Name:       "topiccount",
		RetType:    types.BoolType,
		Fn:         TopicCount,
		ArgTypes:   []types.Type{types.GraphIntType, types.GraphIntType},
		IsVariadic: false,
		IsAction:   false,
	},
	{
		Name:       "topicpublishercount",
		RetType:    types.BoolType,
		Fn:         TopicPublisherCount,
		ArgTypes:   []types.Type{types.GraphStrType, types.GraphIntType, types.GraphIntType},
		IsVariadic: false,
		IsAction:   false,
	},
	{
		Name:       "topicpublishers",
		RetType:    types.BoolType,
		Fn:         TopicPublishers,
		ArgTypes:   []types.Type{types.GraphStrType, types.GraphStrType},
		IsVariadic: true,
		IsAction:   false,
	},
	{
		Name:       "topicpublishersinclude",
		RetType:    types.BoolType,
		Fn:         TopicPublishersInclude,
		ArgTypes:   []types.Type{types.GraphStrType, types.GraphStrType},
		IsVariadic: true,
		IsAction:   false,
	},
	{
		Name:       "topics",
		RetType:    types.BoolType,
		Fn:         Topics,
		ArgTypes:   []types.Type{types.GraphStrType},
		IsVariadic: true,
		IsAction:   false,
	},
	{
		Name:       "topicsinclude",
		RetType:    types.BoolType,
		Fn:         TopicsInclude,
		ArgTypes:   []types.Type{types.GraphStrType},
		IsVariadic: true,
		IsAction:   false,
	},
	{
		Name:       "topicsubscribercount",
		RetType:    types.BoolType,
		Fn:         TopicSubscriberCount,
		ArgTypes:   []types.Type{types.GraphStrType, types.GraphIntType, types.GraphIntType},
		IsVariadic: false,
		IsAction:   false,
	},
	{
		Name:       "topicsubscribers",
		RetType:    types.BoolType,
		Fn:         TopicSubscribers,
		ArgTypes:   []types.Type{types.GraphStrType, types.GraphStrType},
		IsVariadic: false,
		IsAction:   false,
	},
	{
		Name:       "topicsubscribersinclude",
		RetType:    types.BoolType,
		Fn:         TopicSubscribersInclude,
		ArgTypes:   []types.Type{types.GraphStrType, types.GraphStrType},
		IsVariadic: true,
		IsAction:   false,
	},
	//External,
	{
		Name:       "signal",
		RetType:    types.BoolType,
		Fn:         Signal,
		ArgTypes:   []types.Type{types.ExternalStrType},
		IsVariadic: false,
		IsAction:   false,
	},
	{
		Name:       "idsalert",
		RetType:    types.BoolType,
		Fn:         IdsAlert,
		ArgTypes:   []types.Type{types.ExternalStrType},
		IsVariadic: false,
		IsAction:   false,
	},
	//non-message normal expressions
	{Name: "levelname",
		RetType:    types.StringType,
		Fn:         LevelName,
		ArgTypes:   []types.Type{types.IntType},
		IsVariadic: false,
		IsAction:   false,
	},
	{Name: "string",
		RetType:    types.StringType,
		Fn:         String,
		ArgTypes:   []types.Type{types.UnivType},
		IsVariadic: false,
		IsAction:   false,
	},
}
