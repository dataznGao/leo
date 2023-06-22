package callgraph

import (
	"fmt"
	"os"
	"runtime/pprof"
	"testing"
)

func TestDraw(t *testing.T) {
	file := new(os.File)
	var err error
	file, err = os.Create("callgraph.txt")
	if err != nil {
		fmt.Println("无法创建文件:", err)
		return
	}
	defer file.Close()
	var x = [4]int{1, 2, 3, 4}
	y := x
	y[0] = 102
	f := &x
	for _, i2 := range f {
		println(i2)
	}
	println(&x)
	println(&y)
	// 使用 pprof.Lookup("goroutine") 获取 goroutine 信息
	prof := pprof.Lookup("goroutine")
	if prof == nil {
		fmt.Println("找不到 pprof goroutine 信息")
		return
	}

	// 收集函数调用信息
	err = prof.WriteTo(file, 2)
	if err != nil {
		fmt.Println("写入函数调用信息失败:", err)
		return
	}
	fmt.Println("函数调用信息已写入文件 callgraph.txt")
}

func TestAnal(t *testing.T) {

}

// 判断所给路径文件/文件夹是否存在
func Exists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}
