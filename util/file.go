package util

import (
	"bufio"
	"go/ast"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
)

func LoadAllGoFile(path string) ([]string, error) {
	dir, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	res := make([]string, 0)
	err = getAllGoFile(path, dir, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func LoadAllFile(path string) ([]string, []string, error) {
	dir, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, nil, err
	}
	res := make([]string, 0)
	notGoFile := make([]string, 0)
	err = getAllFile(path, dir, &res, &notGoFile)
	if err != nil {
		return nil, nil, err
	}
	return res, notGoFile, nil
}

func getAllFile(parent string, dir []fs.FileInfo, res, notGoFile *[]string) error {
	for _, file := range dir {
		absolutePath := path.Join(parent, file.Name())
		if file.IsDir() {
			readDir, err := ioutil.ReadDir(absolutePath)
			if err != nil {
				return err
			}
			err = getAllFile(absolutePath, readDir, res, notGoFile)
			if err != nil {
				return err
			}
		} else {
			if strings.HasSuffix(file.Name(), ".go") {
				*res = append(*res, absolutePath)
			} else {
				*notGoFile = append(*notGoFile, absolutePath)
			}
		}
	}
	return nil
}

func getAllGoFile(parent string, dir []fs.FileInfo, res *[]string) error {
	for _, file := range dir {
		absolutePath := path.Join(parent, file.Name())
		if file.IsDir() {
			readDir, err := ioutil.ReadDir(absolutePath)
			if err != nil {
				return err
			}
			err = getAllGoFile(absolutePath, readDir, res)
			if err != nil {
				return err
			}
		} else {
			if strings.HasSuffix(file.Name(), ".go") {
				*res = append(*res, absolutePath)
			}
		}
	}
	return nil
}

func LoadTestPath(path string) ([]string, error) {
	dir, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	res := make([]string, 0)
	err = getAllTestFile(path, dir, &res)
	if err != nil {
		return nil, err
	}
	return BatchGetFather(res), nil
}

func getAllTestFile(parent string, dir []fs.FileInfo, res *[]string) error {
	for _, file := range dir {
		absolutePath := path.Join(parent, file.Name())
		if file.IsDir() {
			readDir, err := ioutil.ReadDir(absolutePath)
			if err != nil {
				return err
			}
			err = getAllTestFile(absolutePath, readDir, res)
			if err != nil {
				return err
			}
		} else {
			if strings.HasSuffix(file.Name(), "_test.go") {
				*res = append(*res, absolutePath)
			}
		}
	}
	return nil
}

func ConvertConfigMap(i interface{}) interface{} {
	switch x := i.(type) {
	case map[interface{}]interface{}:
		m2 := map[string]interface{}{}
		for k, v := range x {
			m2[k.(string)] = ConvertConfigMap(v)
		}
		return m2
	case []interface{}:
		for i, v := range x {
			x[i] = ConvertConfigMap(v)
		}
	}
	return i
}

func GetFilePackage(file *ast.File) string {
	return file.Name.Name
}
func isExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		if os.IsNotExist(err) {
			return false
		}
		return false
	}
	return true
}

func CreateFile(path string, code []byte) error {
	length := len(path)
	pos := length
	for pos = length - 1; pos >= 0; pos-- {
		if path[pos] == '/' {
			break
		}
	}
	if path == "" {
		return nil
	}
	prefix := path[:pos]
	if !isExist(prefix) {
		err := os.MkdirAll(prefix, os.ModePerm)
		if err != nil {
			return err
		}
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	writer := bufio.NewWriter(file)
	if err != nil {
		return err
	}
	_, err = writer.Write(code)
	writer.Flush()
	return err
}

func GetPackageName(inputPath string) string {
	file, err := ioutil.ReadFile(inputPath + "/go.mod")
	if err != nil {
		panic(err)
	}
	res := ""
	fileStr := string(file)
	for i := range fileStr {
		if i+6 == len(fileStr) {
			log.Fatalf("[GetPackageName] can`t get package name, please check your inputPath: %v", inputPath)
		}
		if fileStr[i:i+6] == "module" {
			start := i + 6
			for start < len(fileStr) {
				if fileStr[start] == ' ' {
					start++
				} else {
					break
				}
			}
			end := start
			for end < len(fileStr) {
				if fileStr[end] != '\n' {
					end++
				} else {
					break
				}
			}
			res = fileStr[start:end]
			break
		}
	}

	return strings.TrimSpace(res)
}

func Dedup(strs []string) []string {
	dedupMap := make(map[string]bool)
	res := make([]string, 0)
	for _, str := range strs {
		if _, ok := dedupMap[str]; !ok {
			res = append(res, str)
			dedupMap[str] = true
		}
	}
	return res
}
func BatchGetFather(fileNames []string) []string {
	res := make([]string, 0)
	for _, name := range fileNames {
		res = append(res, GetFather(name))
	}
	return Dedup(res)
}

func GetFather(fileName string) string {
	n := len(fileName)
	end := n
	for i := n - 1; i >= 0; i-- {
		if fileName[i] == '/' {
			end = i
			break
		}
	}
	return fileName[:end]
}
