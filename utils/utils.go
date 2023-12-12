package utils

import (
	"os"
	"time"
)

const DAY_START_TIME_ENV string = "TICKTOCK_DAY_START"
const DAY_END_TIME_ENV string = "TICKTOCK_DAY_END"
const HM_ONLY string = "15:04"

func DayStartEnd(day time.Time) (time.Time, time.Time) {
	day = day.Local()
	y, m, d := day.Year(), day.Month(), day.Day()

	startHour, startMinute := timeFromEnv(DAY_START_TIME_ENV, "08:30")
	endHour, endMinute := timeFromEnv(DAY_END_TIME_ENV, "21:00")

	dayStart := time.Date(y, m, d, startHour, startMinute, 0, 0, time.Local)
	dayEnd := time.Date(y, m, d, endHour, endMinute, 0, 0, time.Local)

	return dayStart, dayEnd
}

func timeFromEnv(envName, defaultValue string) (int, int) {
	timeStr := os.Getenv(envName)
	if timeStr == "" {
		timeStr = defaultValue
	}

	t, err := time.ParseInLocation(HM_ONLY, timeStr, time.Local)
	if err != nil {
		t, _ = time.ParseInLocation(HM_ONLY, defaultValue, time.Local)
	}

	return t.Hour(), t.Minute()
}
