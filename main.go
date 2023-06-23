package leo

import (
	_log "github.com/dataznGao/leo/pkg/log"
)

func Main() {

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
