package state

import (
	"isp-gate-service/conf"
	"time"
)

type (
	LimitState interface {
		Import(Snapshot)
		Export() Snapshot
		check() (bool, int, time.Time)
		update(int, time.Time)
	}

	Snapshot struct {
		Timeout  time.Duration
		Datetime []time.Time
		Pattern  string
		Pointer  int
	}

	updateRequest struct {
		pointer     int
		requestTime time.Time
	}
)

func InitLimitState(limits []conf.LimitSetting) (map[string]LimitState, []string, error) {
	limitStates := make(map[string]LimitState)
	patternArray := make([]string, len(limits))

	for i, limit := range limits {
		timeout, err := time.ParseDuration(limit.Timeout)
		if err != nil {
			return nil, nil, err
		}
		if timeout == 0 && limit.MaxCount != 0 {
			limit.MaxCount = 1
		}

		patternArray[i] = limit.Pattern
		limitStates[limit.Pattern] = &limiter{
			timeout:  timeout,
			datetime: make([]time.Time, limit.MaxCount),
			pattern:  limit.Pattern,
			pointer:  -1,
		}
	}
	return limitStates, patternArray, nil
}

func Update(states []LimitState) bool {
	update := make([]updateRequest, len(states))
	for i, st := range states {
		if ok, point, requestTime := st.check(); ok {
			update[i] = updateRequest{pointer: point, requestTime: requestTime}
		} else {
			return false
		}
	}
	for i, st := range states {
		u := update[i]
		st.update(u.pointer, u.requestTime)
	}
	return true
}
