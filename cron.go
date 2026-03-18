package main

import (
	"fmt"
	"time"
)

// RelativeToCron returns a cron expression for an exact time `minutes` from now.
func RelativeToCron(minutes int) string {
	t := time.Now().UTC().Add(time.Duration(minutes) * time.Minute)
	return fmt.Sprintf("%d %d %d %d *", t.Minute(), t.Hour(), t.Day(), int(t.Month()))
}

// AbsoluteToCron returns a cron expression for a specific weekday/hour/minute.
// weekday: 0=Sunday .. 6=Saturday
func AbsoluteToCron(weekday, hour, minute int) string {
	return fmt.Sprintf("%d %d * * %d", minute, hour, weekday)
}
