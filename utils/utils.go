package utils

import (
	"time"
)

func DayStartEnd(day time.Time) (time.Time, time.Time) {
	day = day.Local()
	y, m, d := day.Year(), day.Month(), day.Day()
	dayStart := time.Date(y, m, d, 8, 30, 0, 0, time.Local)
	dayEnd := time.Date(y, m, d, 21, 0, 0, 0, time.Local)

	return dayStart, dayEnd
}
