package _log

import (
	"go/parser"
	"go/token"
	_ast "leo/pkg/log/ast"
	"leo/util"
)

// LoadPackage 加载需要添加日志的文件夹，返回文件名对应的文件
func LoadPackage(path string) (map[string]*_ast.File, []string, error) {
	m, n, err := util.LoadAllFile(path)
	if err != nil {
		return nil, nil, err
	}
	files := make(map[string]*_ast.File, 0)
	fset := token.NewFileSet() // positions are relative to fset
	for _, file := range m {
		f, err1 := parser.ParseFile(fset, file, nil, 0)
		if err1 != nil {
			return nil, nil, err1
		}
		lf := &_ast.File{
			File:   f,
			Logged: false,
		}
		files[file] = lf
	}

	return files, n, nil
}
