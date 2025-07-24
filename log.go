package forge_connect

import (
	"fmt"
	"time"
)

// consoleLog show service logs
func consoleLog(logType, fmtMessage string, args ...interface{}) {
	showMessage := fmt.Sprintf(fmtMessage, args...)
	logTime := time.Now().Format(time.RFC3339)
	fmt.Println(fmt.Sprintf("%s[%s] \033[0;33m \033[0m %s", logTime, logType, showMessage))
}
