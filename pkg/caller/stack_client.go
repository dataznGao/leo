package caller

import (
	"github.com/dataznGao/leo/constant"
	"net/rpc"
	"runtime"
	"strings"
)

func SendStack(num int) {
	//创建连接
	address := ":" + string(constant.CommonPort)
	client, err := rpc.DialHTTP("tcp", address)
	if err != nil {
		panic(err.Error())
	}
	//请求值
	var traceOutput = make([]uintptr, 10)
	callDepth := runtime.Callers(0, traceOutput)
	traceOutput = traceOutput[:callDepth]

	stack := traceToCallStack(traceOutput)
	req := new(SendStackReq)
	req.Chain = stack
	req.Num = num
	//返回值
	var resp *bool

	//使用别名进行调用方法
	err = client.Call("stack.SendStack", req, &resp)
	if err != nil {
		panic(err.Error())
	}

}

func traceToCallStack(trace []uintptr) *CallChain {
	stack := NewCallStack()
	pre := ""
	first := true
	for i := 2; i < len(trace); i++ {
		pc := trace[i]
		funcInfo := runtime.FuncForPC(pc)
		funcName := funcInfo.Name()
		file, _ := funcInfo.FileLine(pc)
		println(file)
		if first {
			first = false
		} else {
			stack.Data[funcName] = pre
		}
		pre = funcName

	}
	return stack
}

//func

// funcConvert 返回函数包，结构体，函数名, 是否是指针
func funcConvert(funcInfo string) (string, string, string, bool) {
	split := strings.Split(funcInfo, ".")
	if len(split) == 2 {
		return split[0], "", split[1], false
	} else if len(split) == 3 {
		if split[1][0] == '(' {
			if split[1][1] == '*' {
				return split[0], split[1][2 : len(split[1])-1], split[2], true
			}
			return split[0], split[1][1 : len(split[1])-1], split[2], false
		} else {
			return split[0], split[1], split[2], false
		}
	} else {
		return "", "", split[len(split)-1], false
	}
}
