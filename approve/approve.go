package approve

import (
	log "github.com/integration-system/isp-log"
	"isp-gate-service/approve/state"
	"isp-gate-service/conf"
	"isp-gate-service/log_code"
	"isp-gate-service/service/matcher"
	"sync"
)

var approvingByAppId map[int64]*approve

type approve struct {
	mx          sync.Mutex
	matcher     matcher.Matcher
	limitStates map[string]state.LimitState
}

func ReceiveConfiguration(setting []conf.ApproveSetting) {
	approvingByAppId = make(map[int64]*approve)
	for _, s := range setting {
		limitState, patternArray, err := state.InitLimitState(s.Limits)
		if err != nil {
			log.Fatal(log_code.FatalConfigApproveSetting, err)
		}

		approvingByAppId[s.ApplicationId] = &approve{
			matcher:     matcher.NewCacheableMatcher(patternArray),
			limitStates: limitState,
			mx:          sync.Mutex{},
		}
	}
}

func GetApprove(appId int64) *approve {
	return approvingByAppId[appId]
}

func (app *approve) ApproveMethod(method string) bool {
	app.mx.Lock()

	stateStorage := make([]state.LimitState, 0)
	patternArray := app.matcher.Match(method)
	for _, pattern := range patternArray {
		stateStorage = append(stateStorage, app.limitStates[pattern])
	}

	resp := state.Update(stateStorage)
	app.mx.Unlock()
	return resp
}
