package letterbox

type scheduleMode string

const (
	modeCron     scheduleMode = "cron"
	modeInterval scheduleMode = "interval"
)
