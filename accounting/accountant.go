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

func (app *accountant) Accept(method string) bool {
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

func (app *accountant) Snapshot() map[string]state.Snapshot {
	app.mx.Lock()
	snapshotLimitState := make(map[string]state.Snapshot)
	for method, limitState := range app.limitStates {
		snapshotLimitState[method] = limitState.Export()
	}
	app.mx.Unlock()
	return snapshotLimitState
}
