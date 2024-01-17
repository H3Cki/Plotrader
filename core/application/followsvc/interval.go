package followsvc

import (
	"math"
	"sync"
	"time"

	"go.uber.org/zap"
)

var predefinedIntervals = map[string]time.Duration{
	"1d": 24 * time.Hour,
	"2d": 2 * 24 * time.Hour,
	"3d": 3 * 24 * time.Hour,
	"4d": 4 * 24 * time.Hour,
	"5d": 5 * 24 * time.Hour,
	"6d": 6 * 24 * time.Hour,
	"1w": 7 * 24 * time.Hour,
	"2w": 14 * 24 * time.Hour,
	"1M": 30 * 24 * time.Hour,
}

var avgExecTimeRatio = 0.5

type intervalLoop struct {
	logger    *zap.SugaredLogger
	interval  time.Duration
	execTimes []time.Duration
	f         func(time.Time) error
	stopC     chan struct{}
	mu        *sync.Mutex
}

func newIntervalLoop(logger *zap.SugaredLogger, interval time.Duration, f func(time.Time) error) *intervalLoop {
	return &intervalLoop{
		logger:    logger,
		interval:  interval,
		f:         f,
		stopC:     make(chan struct{}),
		mu:        &sync.Mutex{},
		execTimes: []time.Duration{time.Second},
	}
}

func (l *intervalLoop) addExecTime(t time.Duration) {
	l.execTimes = append(l.execTimes, t)
	if len(l.execTimes) > 5 {
		l.execTimes = l.execTimes[1:]
	}
}

func (l *intervalLoop) headstart() time.Duration {
	n := int(math.Max(float64(len(l.execTimes)), 1))
	var total time.Duration
	for _, et := range l.execTimes {
		total += et
	}
	avg := time.Duration(total / time.Duration(n))
	return time.Duration(float64(avg) * avgExecTimeRatio)
}

func (l *intervalLoop) loop() error {
	for {
		if ok, err := l.call(); !ok || err != nil {
			return err
		}
	}
}

func (l *intervalLoop) call() (bool, error) {
	headstart := l.headstart()
	nextStart := nextIntervalStart(time.Now().Add(time.Duration(0.5*float64(l.interval))), l.interval)
	l.logger.Debugf("next interval: %s (-%s)", nextStart.String(), headstart)

	select {
	case t := <-time.After(time.Until(nextStart.Add(-headstart))):
		t = t.Add(headstart)
		start := time.Now()
		err := l.f(t)
		execTime := time.Since(start)
		l.logger.Debugf("exec time: %s", execTime)
		if err != nil {
			return false, err
		}
		l.addExecTime(execTime)
	case <-l.stopC:
		return false, nil
	}

	return true, nil
}

// parseInterval adds more units on top of time.Parse():
// 1d, 2d, 3d, 4d, 5d, 6d, 1w, 2w, 1M.
// These units are added for convenience and cannot be combined e.g. Parse("1d12h") or Parse("1w1d") wont work.
func parseInterval(itv string) (time.Duration, error) {
	if d, ok := predefinedIntervals[itv]; ok {
		return d, nil
	}

	d, err := time.ParseDuration(itv)
	if err != nil {
		return 0, err
	}

	return d, nil
}

// IntervalStart returns the start time of the current interval,
// for example if it's 03:15 and the interval duration is 2h,
// the returned value will be 02:00
func intervalStart(now time.Time, every time.Duration) time.Time {
	now = now.In(time.UTC)

	itvSeconds := int64(every.Seconds())
	div := now.Unix() / itvSeconds

	prevStartSeconds := div * itvSeconds

	return time.Unix(prevStartSeconds, 0).In(time.UTC)
}

// IntervalStart returns the start time of the current interval,
// for example if it's 03:15 and the interval duration is 2h,
// the returned value will be 04:00
func nextIntervalStart(now time.Time, every time.Duration) time.Time {
	now = now.In(time.UTC)

	itvSeconds := int64(every.Seconds())
	div := now.Unix() / itvSeconds

	prevStartSeconds := div * itvSeconds
	nextStartSeconds := prevStartSeconds + itvSeconds

	return time.Unix(nextStartSeconds, 0).In(time.UTC)
}
