package caller

import (
	_ast "github.com/dataznGao/leo/pkg/log/ast"
	"github.com/dataznGao/leo/util"
	"go/ast"
	"go/token"
	"strconv"
)

// 生成插桩的代码 leo.SendStack()
func generateCollect(num int) ast.Stmt {
	var (
		expr   = new(ast.CallExpr)
		fun    = new(ast.SelectorExpr)
		pack   = new(ast.Ident)
		method = new(ast.Ident)
		args   = new(ast.BasicLit)
		stmt   = new(ast.ExprStmt)
	)
	pack.Name = "leo"
	method.Name = "SendStack"
	fun.X = pack
	fun.Sel = method
	expr.Fun = fun
	args.Kind = token.INT
	args.Value = strconv.Itoa(num)
	expr.Args = append(expr.Args, args)
	stmt.X = expr
	return stmt
}

func StartCollect(file *ast.File, num int) []byte {
	// 设置log
	funs := _ast.GetFuns(file)
	stmt := generateCollect(num)
	// 获取匿名函数map
	anonyFuncMap := _ast.GetAnonyFuns(funs)
	for _, fun := range funs {
		// 每个函数都得插桩
		// 进行注入，当有一个故障日志被成功注入时，就应该import log
		// 函数粒度注入
		if setCollectInFun(fun, stmt) {
			setImportLeo(file)
		}
	}

	// 对匿名函数进行注入
	for name, lit := range anonyFuncMap {
		// 匿名函数不会有结构体
		// 函数粒度注入
		if setCollectInFun(&ast.FuncDecl{
			Doc:  nil,
			Recv: nil,
			Name: &ast.Ident{Name: name},
			Type: lit.Type,
			Body: lit.Body,
		}, stmt) {
			setImportLeo(file)
		}

	}
	return util.GetFileCode(file)
}

func setCollectInFun(fun *ast.FuncDecl, stmt ast.Stmt) bool {
	hasCollect := false
	vis := &collectVis{stmt, &hasCollect}
	ast.Walk(vis, fun)
	return *vis.hasCollect
}

type collectVis struct {
	stmt       ast.Stmt
	hasCollect *bool
}

func (v *collectVis) Visit(node ast.Node) ast.Visitor {
	if fun, ok := node.(*ast.FuncDecl); ok {
		hasCollect := true
		v.hasCollect = &hasCollect
		if fun.Body == nil || fun.Body.List == nil {
			fun.Body = new(ast.BlockStmt)
			fun.Body.List = make([]ast.Stmt, 0)
		}
		if len(fun.Body.List) == 0 {
			fun.Body.List = append(fun.Body.List, v.stmt)
			// 注入过无需注入
		} else if expr, ok := fun.Body.List[0].(*ast.ExprStmt); ok {
			if call, ok := expr.X.(*ast.CallExpr); ok {
				if ex, ok := call.Fun.(*ast.SelectorExpr); ok {
					if pack, ok := ex.X.(*ast.Ident); ok {
						if pack.Name == "leo" {
							if ex.Sel.Name == "SendStack" {
								return nil
							}
						}
					}
				}
			}
		}
		fun.Body.List = append([]ast.Stmt{v.stmt}, fun.Body.List...)
	}
	return v
}

func setImportLeo(file *ast.File) {
	logPath := new(ast.BasicLit)
	logPath.Value = "\"github.com/dataznGao/leo\""
	logPath.Kind = token.STRING
	fiHasLeo := false
	isHasLeo := false
	if file.Imports == nil {
		file.Imports = make([]*ast.ImportSpec, 0)
	}
	for _, decl := range file.Imports {
		if decl.Path.Value == "\"github.com/dataznGao/leo\"" {
			fiHasLeo = true
			break
		}
	}
	if !fiHasLeo {
		file.Imports = append(file.Imports, &ast.ImportSpec{Path: logPath})
	}
	hasImport := false
	if file.Decls == nil {
		file.Decls = make([]ast.Decl, 0)
	}
	for _, decl := range file.Decls {
		if imports, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range imports.Specs {
				if ispec, ok := spec.(*ast.ImportSpec); ok {
					hasImport = true
					if ispec.Path.Value == "\"github.com/dataznGao/leo\"" {
						isHasLeo = true
						break
					}
				}
			}
		}
	}
	if !hasImport {
		tmp := new(ast.GenDecl)
		tmp.Tok = token.IMPORT
		tmp.Specs = make([]ast.Spec, 0)
		tmp.Specs = append(tmp.Specs, &ast.ImportSpec{Path: logPath})
		if len(file.Decls) == 0 {
			file.Decls = append(file.Decls, tmp)
		} else {
			file.Decls = append([]ast.Decl{tmp}, file.Decls...)
		}
		isHasLeo = true
	}
	if !isHasLeo {
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
