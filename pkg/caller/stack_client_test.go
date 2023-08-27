package caller

import "testing"

//go:noinline

func TestSendStack(t *testing.T) {
	level()
}

func level() {
	SendStack(0)
}

func level2() {
	SendStack(0)
	level()
}

func level3() {
	SendStack(0)
	level2()
}
