package state

import (
	"isp-gate-service/conf"
	"sync"
	"time"
)

type (
	LimitState interface {
		check() (bool, int, time.Time)
		update(int, time.Time)
	}

	updateRequest struct {
		pointer  int
		datetime time.Time
	}
)

func InitLimitState(limits []conf.LimitSetting) (map[string]LimitState, []string, error) {
	limitStates := make(map[string]LimitState)
	patternArray := make([]string, len(limits))

	for i, limit := range limits {
		lifetime, err := time.ParseDuration(limit.Lifetime)
		if err != nil {
			return nil, nil, err
		}
		if lifetime == 0 && limit.MaxCount != 0 {
			limit.MaxCount = 1
		}

		patternArray[i] = limit.Pattern
		limitStates[limit.Pattern] = &limiter{
			lifetime: lifetime,
			datetime: make([]time.Time, limit.MaxCount),
			pattern:  limit.Pattern,
			point:    -1,
			mx:       sync.RWMutex{},
		}
	}
	return limitStates, patternArray, nil
}

func Update(states []LimitState) bool {
	update := make([]updateRequest, len(states))
	for i, st := range states {
		if ok, point, datetime := st.check(); ok {
			update[i] = updateRequest{pointer: point, datetime: datetime}
		} else {
			return false
		}
	}
	for i, st := range states {
		u := update[i]
		st.update(u.pointer, u.datetime)
	}
	return true
}
