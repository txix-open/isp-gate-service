package accounting

import (
	"isp-gate-service/accounting/state"
	"isp-gate-service/service/matcher"
	"sync"
	"sync/atomic"
)

type accountant struct {
	mx          sync.Mutex
	matcher     matcher.Matcher
	limitStates map[string]state.LimitState
	version     int64
}

func (app *accountant) Accept(method string) bool {
	patternArray := app.matcher.Match(method)
	stateStorage := make([]state.LimitState, len(patternArray))
	for i, pattern := range patternArray {
		stateStorage[i] = app.limitStates[pattern]
	}

	app.mx.Lock()
	resp := state.Update(stateStorage)
	//only true or every request todo
	atomic.AddInt64(&app.version, 1)
	app.mx.Unlock()
	return resp
}

func (app *accountant) Snapshot() map[string]state.Snapshot {
	app.mx.Lock()
	snapshotLimitState := make(map[string]state.Snapshot, len(app.limitStates))
	for method, limitState := range app.limitStates {
		snapshotLimitState[method] = limitState.Export()
	}
	app.mx.Unlock()
	return snapshotLimitState
}

func (app *accountant) GetVersion() int64 {
	return atomic.LoadInt64(&app.version)
}