package main

import "github.com/dataznGao/leo/pkg/caller"

// SendStack 将栈推到服务端，根据数字不同，会推到不同的调用母图中组合
func SendStack(num int) {
	caller.SendStack(num)
}
