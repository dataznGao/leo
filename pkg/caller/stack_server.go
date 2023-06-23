package caller

import (
	"github.com/dataznGao/leo/constant"
	"net"
	"net/http"
	"net/rpc"
)

type StackUtil struct {
}

type SendStackReq struct {
	Chain *CallChain
	Num   int
}

// CallChain 调用链
type CallChain struct {
	Data map[string]string //记录函数调用关系
}

func NewCallStack() *CallChain {
	return &CallChain{Data: make(map[string]string)}
}

func (mu *StackUtil) SendStack(req *SendStackReq, resq *bool) error {
	constant.CallGraph[req.Num] = add(constant.CallGraph[req.Num], req.Chain.Data)
	*resq = false
	return nil
}

func add(mother map[string]map[string]string, son map[string]string) map[string]map[string]string {
	if mother == nil {
		mother = make(map[string]map[string]string)
	}
	for k, v := range son {
		if _, ok := mother[k]; ok {
			if _, ok := mother[k][v]; !ok {
				mother[k][v] = "common call"
			}
		} else {
			mother[k] = map[string]string{v: "common call"}
		}
	}
	return mother
}

func StartServe() {
	//初始化结构体
	stackService := StackUtil{}
	// 调用net/rpc的功能进行注册
	//err := rpc.Register(&mathUtil)
	//这里可以使用取别名的方式
	err := rpc.RegisterName("stack", &stackService)
	//判断结果是否正确
	if err != nil {
		panic(err.Error())
	}
	//通过HandleHTTP()把mathUtil提供的服务注册到HTTP协议上，方便调用者利用http的方式进行数据传递
	rpc.HandleHTTP()
	//指定端口监听
	address := ":" + string(constant.CommonPort)
	listen, err := net.Listen("tcp", address)
	if err != nil {
		panic(err.Error())
	}
	//开启服务
	http.Serve(listen, nil)

}
