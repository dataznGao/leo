package _log

import (
	"github.com/dataznGao/leo/pkg/callgraph"
	_ast "github.com/dataznGao/leo/pkg/log/ast"
	"github.com/dataznGao/leo/util"
	"testing"
)

func TestInsertCollector(t *testing.T) {
	inputPath := "/Users/misery/GolandProjects/jupiter/pkg/conf"
	outputPath := "/Users/misery/GolandProjects/rpc_demo_3"
	InsertCollector(inputPath, outputPath, 1)
}

func TestFixCallGraph(t *testing.T) {

}

func TestInjureLog(t *testing.T) {
	inputPath := "/Users/misery/GolandProjects/rpc_demo"
	outputPath := "/Users/misery/GolandProjects/rpc_demo_2"
	files, _, err := LoadPackage(inputPath)
	if err != nil {
		return
	}
	diffs := make([]*callgraph.Diff, 0)
	diffs = append(diffs, &callgraph.Diff{
		NodeA: &callgraph.Node{
			Caller: &callgraph.Func{
				FilePath:   "/Users/misery/GolandProjects/rpc_demo/tttt/aaas",
				StructName: "MyT",
				FuncName:   "RunClient1$1$1$1",
				IsPointer:  false,
			},
			Callee: &callgraph.Func{
				FilePath:   "/Users/misery/GolandProjects/rpc_demo/tttt/aaas",
				StructName: "X",
				FuncName:   "Call",
				IsPointer:  false,
			},
			Description: "1",
		},
	})
	diffs = append(diffs, &callgraph.Diff{
		NodeA: &callgraph.Node{
			Caller: &callgraph.Func{
				FilePath:   "/Users/misery/GolandProjects/rpc_demo/tttt/aaas",
				StructName: "MyT",
				FuncName:   "RunClient1$1$1$1",
				IsPointer:  false,
			},
			Callee: &callgraph.Func{
				FilePath:   "/Users/misery/GolandProjects/rpc_demo/tttt/aaas",
				StructName: "X",
				FuncName:   "Call",
				IsPointer:  false,
			},
			Description: "2",
		},
	})
	// /Users/misery/GolandProjects/jupiter/pkg/executor	Stop$1$1	Info
	diffs = dedupDiff(diffs)
	// 注入error, 并产生import log
	for k, file := range files {
		code, hasLogged := _ast.InjureLog(k, file, diffs)
		err := util.CreateFile(util.CompareAndExchange(k, outputPath, inputPath), code)
		if err != nil {
			return
		}
		file.Logged = hasLogged
	}
}
func dedupDiff(diff []*callgraph.Diff) []*callgraph.Diff {
	res := make([]*callgraph.Diff, 0)
	deMap := make(map[string]*callgraph.Diff, 0)
	for _, di := range diff {
		deMap[di.ToString()] = di
	}
	for _, d := range deMap {
		res = append(res, d)
	}
	return res
}
