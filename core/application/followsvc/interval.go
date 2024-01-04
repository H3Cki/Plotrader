package followsvc

import "time"

// IntervalStart returns the start time of the current interval,
// for example if it's 03:15 and the interval duration is 2h,
// the returned value will be 02:00
func StartTime(now time.Time, every time.Duration) time.Time {
	now = now.In(time.UTC)

	itvSeconds := int64(every.Seconds())
	div := now.Unix() / itvSeconds

	prevStartSeconds := div * itvSeconds

	return time.Unix(prevStartSeconds, 0).In(time.UTC)
}

// IntervalStart returns the start time of the current interval,
// for example if it's 03:15 and the interval duration is 2h,
// the returned value will be 04:00
func NextStartTime(now time.Time, every time.Duration) time.Time {
	now = now.In(time.UTC)

	itvSeconds := int64(every.Seconds())
	div := now.Unix() / itvSeconds

	prevStartSeconds := div * itvSeconds
	nextStartSeconds := prevStartSeconds + itvSeconds

	return time.Unix(nextStartSeconds, 0).In(time.UTC)

}
