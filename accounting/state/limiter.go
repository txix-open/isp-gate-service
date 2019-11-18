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

func (lim *limiter) Import(data Snapshot) {
	importLim := limiter{
		timeout:  data.Timeout,
		datetime: data.Datetime,
		pattern:  data.Pattern,
		pointer:  data.Pointer,
	}

	lenImportDatetime := len(importLim.datetime)
	lenDatetime := len(lim.datetime)
	switch true {
	case lenImportDatetime == lenDatetime:
		lim.datetime = importLim.datetime
		lim.pointer = importLim.pointer

	case lenImportDatetime < lenDatetime:
		oldPointer := importLim.pointer
		for i := range lim.datetime {
			importLim.pointer++
			if importLim.pointer >= len(importLim.datetime) {
				importLim.pointer = 0
			}

			lim.datetime[i] = importLim.datetime[importLim.pointer]

			if oldPointer == importLim.pointer {
				lim.pointer = i
				break
			}
		}

	case lenImportDatetime > lenDatetime:
		for i := range lim.datetime {
			importLim.pointer++
			if importLim.pointer >= len(importLim.datetime) {
				importLim.pointer = 0
			}
			lim.datetime[i] = importLim.datetime[importLim.pointer]
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
