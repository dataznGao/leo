package caller

import "testing"

func TestSendStack(t *testing.T) {
	level3()
}

func level() {
	print(1)
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
