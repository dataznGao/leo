package util

import (
	"strings"
	"testing"
)

func TestCommand(t *testing.T) {
	caller := "github.com/douyu/jupiter/pkg/util/xgo.try2"
	packageName := "github.com/douyu/jupiter"
	caller = strings.Replace(caller, packageName, "", 1)
	println(caller)
}
