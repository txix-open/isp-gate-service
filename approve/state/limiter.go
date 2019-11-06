package state

import (
	"time"
)

type limiter struct {
	lifetime time.Duration
	datetime []time.Time
	pattern  string
	point    int
}

func (lim *limiter) check() (bool, int, time.Time) {
	if len(lim.datetime) == 0 {
		return false, 0, time.Time{}
	}
	if lim.lifetime == 0 {
		return true, 0, time.Now()
	}

	point := lim.point + 1
	if point >= len(lim.datetime) {
		point = 0
	}
	date := lim.datetime[point]
	now := time.Now()
	approve := now.Sub(date) > lim.lifetime

	return approve, point, now
}

func (lim *limiter) update(point int, time time.Time) {
	lim.point = point
	lim.datetime[point] = time
}
