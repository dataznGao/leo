package caller

import "testing"

func TestSendStack(t *testing.T) {
	level3()
}

func level() {
	print(1)
	SendStack()

}

func level2() {
	SendStack()
	level()
}

func level3() {
	SendStack()
	level2()
}
