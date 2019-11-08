package accounting

import (
	log "github.com/integration-system/isp-log"
	"isp-gate-service/accounting/state"
	"isp-gate-service/conf"
	"isp-gate-service/log_code"
	"isp-gate-service/service/matcher"
	"sync"
)

var accountingByApplicationId = make(map[int64]*accounting)

type accounting struct {
	mx          sync.Mutex
	matcher     matcher.Matcher
	limitStates map[string]state.LimitState
}

func ReceiveConfiguration(conf conf.Accounting) {
	newAccountingByApplicationId := make(map[int64]*accounting)
	if conf.Enable {
		for _, s := range conf.Setting {
			limitState, patternArray, err := state.InitLimitState(s.Limits)
			if err != nil {
				log.Fatal(log_code.FatalConfigApproveSetting, err)
			}

			newAccountingByApplicationId[s.ApplicationId] = &accounting{
				matcher:     matcher.NewCacheableMatcher(patternArray),
				limitStates: limitState,
				mx:          sync.Mutex{},
			}
		}
	}
	accountingByApplicationId = newAccountingByApplicationId
}

func GetAccounting(appId int64) *accounting {
	return accountingByApplicationId[appId]
}

func (app *accounting) TakeAccount(method string) bool {
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
