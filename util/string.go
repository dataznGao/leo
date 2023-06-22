package util

import (
	"strings"
)

func CompareAndExchange(oldPath, newPath, inputPath string) string {
	if strings.HasPrefix(oldPath, inputPath) {
		replace := strings.Replace(oldPath, inputPath, newPath, 1)
		return replace
	}
	return ""
}

func CompareAndExchangeTestPath(oldPath, newPath, inputPath string) string {
	if strings.HasPrefix(oldPath, inputPath) {
		replace := strings.Replace(oldPath, inputPath, newPath, 1)
		return replace
	}
	return ""
}

func Contains(elem string, elems []string) bool {
	if elem == "*" {
		return true
	}
	for _, s := range elems {
		if s == elem {
			return true
		}
	}
	return false
}
