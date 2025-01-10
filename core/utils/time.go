package utils

import (
	"time"
	_ "time/tzdata"
)

// Now now
func Now() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), 0, time.Local)
}

// TimeZone time zone
func TimeZone() string {
	zone, _ := Now().Zone()
	return zone
}
