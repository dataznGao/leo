package _ast

import (
	"go/ast"
	"go/token"
	"leo/constant"
	"leo/pkg/callgraph"
	"strconv"
	"strings"
)

type Item struct {
	Block  *ast.BlockStmt
	Callee string
}

// CanInjuredMap 函数块与被调用函数粒度的map，如果该map中存在，则不能注入
var CanInjuredMap = make(map[Item]bool)

var AnonyFuncMap = make(map[string]*ast.FuncLit)

type DiffVisitor struct {
	diff      *callgraph.Diff
	HasLogged *bool
}

func (v *DiffVisitor) Visit(node ast.Node) ast.Visitor {
	if f, ok := node.(*ast.File); ok {
		v.HasLogged = setLog(f, v.diff)
	}
	return nil
}

func setLog(file *ast.File, diff *callgraph.Diff) *bool {
	hasLog := false
	// 设置log
	caller := diff.NodeA.Caller
	funs := getFuns(file)
	// 获取匿名函数map
	AnonyFuncMap = getAnonyFuns(funs)
	// flag标识，标识真实函数是否被注入
	flag := false
	for _, fun := range funs {
		// 函数名和差异体中调用者的函数名相同，并且结构体也相同，就可以往函数里注入日志
		canLog := false
		if fun.Name.Name == caller.FuncName {
			if caller.StructName != "" {
				if fun.Recv == nil || fun.Recv.List == nil {
					continue
				}
				for _, field := range fun.Recv.List {
					if structName, ok := field.Type.(*ast.Ident); ok {
						if caller.StructName == structName.Name {
							canLog = true
							break
						}
					} else if star, ok := field.Type.(*ast.StarExpr); ok {
						if structName, ok := star.X.(*ast.Ident); ok {
							if caller.StructName == structName.Name {
								canLog = true
								break
							}
						}
					}
				}
			} else {
				canLog = true
			}
		}
		// 进行注入，当有一个故障日志被成功注入时，就应该import log
		if canLog {
			flag = true
			// 函数粒度注入
			if setLogInFun(fun, diff) {
				hasLog = true
				setImportLog(file)
			}
		}
	}
	if flag == false {
		// 对匿名函数进行注入
		for name, lit := range AnonyFuncMap {
			// 匿名函数不会有结构体
			if name == caller.FuncName {
				// 函数粒度注入
				if setLogInFun(&ast.FuncDecl{
					Doc:  nil,
					Recv: nil,
					Name: &ast.Ident{Name: name},
					Type: lit.Type,
					Body: lit.Body,
				}, diff) {
					hasLog = true
					setImportLog(file)
				}
			}
		}
	}
	return &hasLog
}

func getAnonyFuns(funs []*ast.FuncDecl) map[string]*ast.FuncLit {
	res := make(map[string]*ast.FuncLit)
	for _, fun := range funs {
		head := &ast.FuncLit{Body: fun.Body, Type: &ast.FuncType{}}
		node := &litNode{
			FuncName: fun.Name.Name,
			FuncLit:  head,
			Children: make([]*litNode, 0),
		}
		iterate(head, node)
		collect(node, &res)
	}
	return res
}

func iterate(head *ast.FuncLit, node *litNode) {
	// 收集一层
	lits := getLits(head)
	for i, lit := range lits {
		child := &litNode{
			FuncName: node.FuncName + "$" + strconv.Itoa(i+1),
			FuncLit:  lit,
			Children: make([]*litNode, 0),
		}
		node.Children = append(node.Children, child)
		iterate(lit, child)
	}
}

func collect(node *litNode, res *map[string]*ast.FuncLit) {
	ma := *res
	if strings.Contains(node.FuncName, "$") {
		ma[node.FuncName] = node.FuncLit
	}
	res = &ma
	for _, child := range node.Children {
		collect(child, res)
	}
}

type litNode struct {
	FuncName string
	FuncLit  *ast.FuncLit
	Children []*litNode
}

func getLits(lit *ast.FuncLit) []*ast.FuncLit {
	lits := make([]*ast.FuncLit, 0)
	for _, stmt := range lit.Body.List {
		counter := &litCounter{
			lits: &lits,
		}
		ast.Walk(counter, stmt)
	}
	return lits
}

type litCounter struct {
	lits *[]*ast.FuncLit
}

func (v *litCounter) Visit(node ast.Node) ast.Visitor {
	if fu, ok := node.(*ast.FuncLit); ok {
		*v.lits = append(*v.lits, fu)
		return nil
	}
	return v
}

func setImportLog(file *ast.File) {
	logPath := new(ast.BasicLit)
	logPath.Value = "\"log\""
	logPath.Kind = token.STRING
	fiHasLog := false
	isHasLog := false
	for _, decl := range file.Imports {
		if decl.Path.Value == "\"log\"" {
			fiHasLog = true
			break
		}
	}
	if !fiHasLog {
		file.Imports = append(file.Imports, &ast.ImportSpec{Path: logPath})
	}
	for _, decl := range file.Decls {
		if imports, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range imports.Specs {
				if ispec, ok := spec.(*ast.ImportSpec); ok {
					if ispec.Path.Value == "\"log\"" {
						isHasLog = true
						break
					}
				}
			}
		}
	}
	if !isHasLog {
		for _, decl := range file.Decls {
			if imports, ok := decl.(*ast.GenDecl); ok {
				for _, spec := range imports.Specs {
					if _, ok := spec.(*ast.ImportSpec); ok {
						imports.Specs = append(imports.Specs, &ast.ImportSpec{Path: logPath})
					}
				}
			}
		}
	}
}

func setLogInFun(fun *ast.FuncDecl, diff *callgraph.Diff) bool {
	hasLog := false
	vis := &calleeVis{diff, &hasLog}
	ast.Walk(vis, fun)
	return *vis.hasLog
}

func getFuns(file *ast.File) []*ast.FuncDecl {
	funs := make([]*ast.FuncDecl, 0)
	for _, decl := range file.Decls {
		if fun, ok := decl.(*ast.FuncDecl); ok {
			funs = append(funs, fun)
		}
	}
	return funs
}

func getAllCallee(stmt ast.Node, diff *callgraph.Diff) []*ast.CallExpr {
	res := make([]*ast.CallExpr, 0)
	v := &calleeStmtVis{
		diff:   diff,
		Callee: res,
	}
	ast.Walk(v, stmt)
	return v.Callee
}

type calleeVis struct {
	diff   *callgraph.Diff
	hasLog *bool
}

func (v *calleeVis) Visit(node ast.Node) ast.Visitor {
	// 函数中找到有有函数调用的block
	fun := node.(*ast.FuncDecl)
	block, index := FindHasCalleeBlock(fun, v.diff)
	if block != nil && len(block) > 0 {
		tr := true
		v.hasLog = &tr
		for j, stmt := range block {
			if _, ok := CanInjuredMap[Item{
				Block:  stmt,
				Callee: v.diff.NodeA.Callee.FuncName,
			}]; ok {
				continue
			} else {
				if len(stmt.List) == 0 {
					stmt.List = append(stmt.List, GenerateLog(constant.LogContent))
				} else {
					can := true
					if index[j] >= 0 {
						if _, ok := stmt.List[index[j]].(*ast.ReturnStmt); ok {
							can = false
						}
					}
					for _, s := range stmt.List {
						if stm, ok := s.(*ast.ExprStmt); ok {
							if ide, ok := stm.X.(*ast.CallExpr); ok {
								if len(ide.Args) > 0 {
									if lit, ok := ide.Args[0].(*ast.BasicLit); ok {
										if lit.Value == constant.LogContent {
											can = false
											break
										}
									}
								}
							}
						}
					}
					if can {
						stmt.List = append(stmt.List[:index[j]+1],
							append([]ast.Stmt{GenerateLog(constant.LogContent)}, stmt.List[index[j]+1:]...)...)
					}
				}
				CanInjuredMap[Item{
					Block:  stmt,
					Callee: v.diff.NodeA.Callee.FuncName,
				}] = true
			}
		}

	}
	return nil
}

type blockVisitor struct {
	diff  *callgraph.Diff
	block []*ast.BlockStmt
	index []int
}

func (v *blockVisitor) Visit(node ast.Node) ast.Visitor {
	if block, ok := node.(*ast.BlockStmt); ok {
		// 判断这个list里面有没有callee
		for i, stmt := range block.List {
			// 对这个stmt，判断其内部有没有callee
			if len(getAllCallee(stmt, v.diff)) > 0 {
				v.index = append(v.index, i)
				v.block = append(v.block, block)
			}
		}
		// 如果是If语句，条件内有CallExpr
	} else if ifStmt, ok := node.(*ast.IfStmt); ok {
		if ifStmt.Init != nil && len(getAllCallee(ifStmt.Init, v.diff)) > 0 {
			// 立刻添加日志，
			v.index = append(v.index, -1)
			v.block = append(v.block, ifStmt.Body)
		} else if ifStmt.Cond != nil && len(getAllCallee(ifStmt.Cond, v.diff)) > 0 {
			// 立刻添加日志，
			v.index = append(v.index, 0)
			v.block = append(v.block, ifStmt.Body)
		}
	}
	return v
}

func FindHasCalleeBlock(node *ast.FuncDecl, diff *callgraph.Diff) ([]*ast.BlockStmt, []int) {
	// block中有函数调用, 给他加日志
	v := &blockVisitor{
		diff:  diff,
		block: make([]*ast.BlockStmt, 0),
		index: make([]int, 0),
	}
	if node == nil {
		return nil, nil
	}
	ast.Walk(v, node)
	v.block, v.index = dedup(v.block, v.index)
	return v.block, v.index
}

func dedup(block []*ast.BlockStmt, index []int) ([]*ast.BlockStmt, []int) {
	ma := make(map[*ast.BlockStmt]string, 0)
	newBlock := make([]*ast.BlockStmt, 0)
	newIndex := make([]int, 0)
	for i, stmt := range block {
		if _, ok := ma[stmt]; !ok {
			newBlock = append(newBlock, stmt)
			newIndex = append(newIndex, index[i])
		}
	}
	return newBlock, newIndex
}

type calleeStmtVis struct {
	diff   *callgraph.Diff
	Callee []*ast.CallExpr
}

func (v *calleeStmtVis) Visit(node ast.Node) ast.Visitor {
	// 如果是block，就终止
	if _, ok := node.(*ast.BlockStmt); ok {
		return nil
	}
	if calleeStmt, ok := node.(*ast.CallExpr); ok {
		if callee, ok := calleeStmt.Fun.(*ast.Ident); ok {
			if callee.Name == v.diff.NodeA.Callee.FuncName && v.diff.NodeA.Callee.StructName == "" {
				v.Callee = append(v.Callee, calleeStmt)
			}
		} else if callee, ok := calleeStmt.Fun.(*ast.SelectorExpr); ok {
			if callee.Sel.Name == v.diff.NodeA.Callee.FuncName && v.diff.NodeA.Callee.StructName != "" {
				v.Callee = append(v.Callee, calleeStmt)
			}
		}
		// 对匿名函数，无法通过ast来遍历，直接用body判断
		if lit, ok := calleeStmt.Fun.(*ast.FuncLit); ok {
			for name, body := range AnonyFuncMap {
				if lit == body && name == v.diff.NodeA.Callee.FuncName {
					v.Callee = append(v.Callee, calleeStmt)
				}
			}
		}
	}
	return v
}
