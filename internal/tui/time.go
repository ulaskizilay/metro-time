package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func remainingTime(clock string) (string, bool) {
	return remainingTimeAt(clock, time.Now())
}

func remainingTimeAt(clock string, now time.Time) (string, bool) {
	hour, min, ok := parseClock(clock)
	if !ok {
		return clock, false
	}

	nowTotal := now.Hour()*60 + now.Minute()
	targetTotal := hour*60 + min
	// Trips after midnight belong to the previous evening's service.
	if targetTotal < nowTotal && now.Hour() > 20 && hour < 4 {
		targetTotal += 24 * 60
	}
	if diff := targetTotal - nowTotal; diff > 0 {
		return fmt.Sprintf("%d Dk", diff), false
	}
	return "Peronda", true
}

func parseClock(clock string) (hour, min int, ok bool) {
	parts := strings.Split(clock, ":")
	if len(parts) != 2 {
		return 0, 0, false
	}
	h, errH := strconv.Atoi(strings.TrimSpace(parts[0]))
	m, errM := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errH != nil || errM != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, false
	}
	return h, m, true
}
