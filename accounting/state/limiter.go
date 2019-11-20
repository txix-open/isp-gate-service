package state

import (
	"time"
)

type limiter struct {
	timeout  time.Duration
	datetime []time.Time
	pattern  string
	pointer  int
}

func (lim *limiter) check() (bool, int, time.Time) {
	if len(lim.datetime) == 0 {
		return false, 0, time.Time{}
	}
	if lim.timeout == 0 {
		return true, 0, time.Now()
	}

	pointer := lim.pointer + 1
	if pointer >= len(lim.datetime) {
		pointer = 0
	}
	date := lim.datetime[pointer]
	requestTime := time.Now()
	account := requestTime.Sub(date) > lim.timeout

	return account, pointer, requestTime
}

func (lim *limiter) update(point int, requestTime time.Time) {
	lim.pointer = point
	lim.datetime[point] = requestTime
}
