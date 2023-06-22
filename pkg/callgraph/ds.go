package callgraph

import "strings"

type Node struct {
	Caller      *Func
	Callee      *Func
	Description string
}

func String2Func(caller string) *Func {
	fun := &Func{
		FilePath:   "",
		StructName: "",
		FuncName:   "",
		IsPointer:  false,
	}
	split := strings.Split(caller, ".")
	// 有括号，是结构体，会有2个.
	if strings.HasPrefix(caller, "(") {
		firstSplit := ""
		realSplit := make([]string, 0)
		if len(split) > 3 {
			for i := 0; i < len(split)-2; i++ {
				firstSplit += split[i] + "."
			}
			firstSplit = firstSplit[:len(firstSplit)-1]
			realSplit = append(realSplit, firstSplit)
			realSplit = append(realSplit, split[len(split)-2:]...)
		} else {
			realSplit = split
		}
		// 取中间的，去掉)
		str := realSplit[1][:len(realSplit[1])-1]
		fun.StructName = str
		// 去掉(
		if realSplit[0][1] == '*' {
			fun.FilePath = realSplit[0][2:]
			fun.IsPointer = true
		} else {
			fun.FilePath = realSplit[0][1:]
			fun.IsPointer = false
		}
		fun.FuncName = realSplit[2]
	} else {
		firstSplit := ""
		realSplit := make([]string, 0)
		if len(split) > 2 {
			for i := 0; i < len(split)-1; i++ {
				firstSplit += split[i] + "."
			}
			firstSplit = firstSplit[:len(firstSplit)-1]
			realSplit = append(realSplit, firstSplit)
			realSplit = append(realSplit, split[len(split)-1:]...)
		} else {
			realSplit = split
		}
		fun.FilePath = realSplit[0]
		fun.FuncName = realSplit[1]
	}
	return fun
}

// Func (*/Users/misery/GolandProjects/jupiter/pkg/core/sentinel.etcdv3DataSource).Initialize
type Func struct {
	FilePath   string
	StructName string
	FuncName   string
	IsPointer  bool
}

func (n *Node) ToString() string {
	return n.Caller.ToString() + "." + n.Callee.ToString() + "." + n.Description
}

func (f *Func) ToString() string {
	res := ""
	if f.StructName != "" {
		if f.IsPointer {
			res += "(*" + f.FilePath + "." + f.StructName + ")." + f.FuncName
		}
		res += "(" + f.FilePath + "." + f.StructName + ")." + f.FuncName
	} else {
		res += f.FilePath + "." + f.FuncName
	}
	return res
}

type Diff struct {
	NodeA  *Node
	NodeB  *Node
	Detail *Detail
}

func (d *Diff) ToString() string {
	var de Detail
	if d.Detail == nil {
		de = 0
	} else {
		de = *d.Detail
	}
	if d.NodeA == nil && d.NodeB != nil {
		return "|" + d.NodeB.ToString() + "|" + string(rune(de))
	} else if d.NodeA != nil && d.NodeB == nil {
		return d.NodeA.ToString() + "|" + "|" + string(rune(de))
	} else if d.NodeA == nil && d.NodeB == nil {
		return ""
	}
	return d.NodeA.ToString() + "|" + d.NodeB.ToString() + "|" + string(rune(de))
}

func (d *Diff) PrintTrace() []string {
	return []string{d.NodeA.Caller.FilePath, d.NodeA.Caller.FuncName, d.NodeA.Callee.FuncName}
}

type Detail int

// Diff 定义图的不同
var (
	DescDiff     Detail = 1
	SideLackDiff Detail = 2
)
