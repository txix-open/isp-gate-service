package accounting

import (
	"isp-gate-service/accounting/state"
	"isp-gate-service/service/matcher"
	"sync"
)

type accountant struct {
	mx          sync.Mutex
	matcher     matcher.Matcher
	limitStates map[string]state.LimitState
}

func (app *accountant) Check(method string) bool {
	patternArray := app.matcher.Match(method)
	stateStorage := make([]state.LimitState, len(patternArray))
	for i, pattern := range patternArray {
		stateStorage[i] = app.limitStates[pattern]
	}

	app.mx.Lock()
	resp := state.Update(stateStorage)
	app.mx.Unlock()
	return resp
}

func (app *accountant) getLimitState() map[string]state.LimitState {
	limitStates := app.limitStates
	return limitStates
}
