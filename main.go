package main

import (
	_log "leo/pkg/log"
)

func main() {

	err := _log.Log("/Users/misery/GolandProjects/jupiter",
		"/Users/misery/GolandProjects/jupiter2")
	if err != nil {
		return
	}

	//err := _log.generateDiff("/Users/misery/GolandProjects/tidb", "/Users/misery/GolandProjects/tidb/bindinfo",
	//	"/Users/misery/GolandProjects/tidb2")
	//if err != nil {
	//	return
	//}
}
