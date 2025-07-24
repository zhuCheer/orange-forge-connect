package forge_connect

import (
	"fmt"
	"os"
	"time"
)

var appTimeZone = ""

func init() {
	// get TZ from environment
	tz := os.Getenv("TZ")
	if tz != "" {
		// validate the TZ
		if _, err := time.LoadLocation(tz); err == nil {
			appTimeZone = tz
			return
		}
	}
	appTimeZone = time.Now().Location().String()
}

// DateToTm make string type date to go time.Time
func DateToTm(date string) time.Time {
	var cstSh, _ = time.LoadLocation(appTimeZone)
	if appTimeZone == "Local" {
		cstSh = time.Local
	}
	tm, _ := time.ParseInLocation("2006-01-02 15:04:05", date, cstSh)
	return tm
}

// TimeFormat format the time to string
func TimeFormat(tm time.Time) string {
	if tm.IsZero() {
		return ""
	}

	return tm.Format("2006-01-02 15:04:05")
}

// consoleRouter show http handler router
func consoleRouter(method, patten string) {
	spaceLen := 7 - len(method)
	space := "       "
	space = space[:spaceLen]
	fmt.Println(fmt.Sprintf("[FORGE-CONNECT] \033[0;33m %s\033[0m%s %s", method, space, patten))
}
