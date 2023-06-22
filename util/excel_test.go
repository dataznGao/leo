package util

import "testing"

func TestDataToExcel(t *testing.T) {
	DataToExcel("/Users/misery/GolandProjects/leo/test.xlsx", [][]string{{"aa"}, {"bb"}})
}
