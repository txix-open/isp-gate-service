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

func (lim *limiter) Export() Snapshot {
	ls := *lim
	return Snapshot{
		Timeout:  ls.timeout,
		Datetime: ls.datetime,
		Pattern:  ls.pattern,
		Pointer:  ls.pointer,
	}
}

func (lim *limiter) Import(oldState Snapshot) {
	oldLim := limiter{
		timeout:  oldState.Timeout,
		datetime: oldState.Datetime,
		pattern:  oldState.Pattern,
		pointer:  oldState.Pointer,
	}

	lenOldDatetime := len(oldLim.datetime)
	lenNewDatetime := len(lim.datetime)

	switch true {
	case lenOldDatetime == lenNewDatetime:
		lim.datetime = oldLim.datetime
		lim.pointer = oldLim.pointer

	case lenOldDatetime < lenNewDatetime:
		oldPointer := oldLim.pointer
		for i := range lim.datetime {
			oldLim.pointer++
			if oldLim.pointer >= len(oldLim.datetime) {
				oldLim.pointer = 0
			}

			lim.datetime[i] = oldLim.datetime[oldLim.pointer]

			if oldPointer == oldLim.pointer {
				lim.pointer = i
				break
			}
		}

	case lenOldDatetime > lenNewDatetime:
		for i := range lim.datetime {
			oldLim.pointer++
			if oldLim.pointer >= len(oldLim.datetime) {
				oldLim.pointer = 0
			}
			lim.datetime[i] = oldLim.datetime[oldLim.pointer]
		}
	}
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
