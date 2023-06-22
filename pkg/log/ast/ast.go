package _ast

import (
	"go/ast"
	"leo/pkg/callgraph"
	"leo/util"
	"strings"
)

func GenerateLog(content string) ast.Stmt {
	var (
		expr   = new(ast.CallExpr)
		fun    = new(ast.SelectorExpr)
		pack   = new(ast.Ident)
		method = new(ast.Ident)
		args   = new(ast.BasicLit)
		stmt   = new(ast.ExprStmt)
	)
	pack.Name = "log"
	method.Name = "Print"
	fun.X = pack
	fun.Sel = method
	args.Value = content
	expr.Fun = fun
	expr.Args = append(expr.Args, args)
	stmt.X = expr
	return stmt
}

type File struct {
	File   *ast.File
	Logged bool
}

func InjureLog(filePath string, file *File, diffs []*callgraph.Diff) ([]byte, bool) {
	hasLogged := false
	for _, diff := range diffs {
		// 这种情况下，说明是原调用图存在该调用关系，可以进行注入
		if diff.NodeA != nil {
			if strings.HasPrefix(filePath, diff.NodeA.Caller.FilePath) {
				diffVisitor := &DiffVisitor{
					diff: diff,
				}
				ast.Walk(diffVisitor, file.File)
				hasLogged = *diffVisitor.HasLogged
			}
		}
	}
	return util.GetFileCode(file.File), hasLogged
}
