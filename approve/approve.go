package approve

import (
	log "github.com/integration-system/isp-log"
	"isp-gate-service/approve/matcher"
	"isp-gate-service/approve/state"
	"isp-gate-service/conf"
	"isp-gate-service/log_code"
)

var approvingByAppId = make(map[int64]approve)

type approve struct {
	matcher     matcher.Matcher
	limitStates map[string]state.LimitState
}

func ReceiveConfiguration(setting []conf.ApproveSetting) {
	for _, s := range setting {
		limitState, patternArray, err := state.InitLimitState(s.Limits)
		if err != nil {
			log.Fatal(log_code.FatalConfigApproveSetting, err)
		}

		approvingByAppId[s.ApplicationId] = approve{
			matcher:     matcher.NewCacheableMatcher(patternArray),
			limitStates: limitState,
		}
	}
}

func Complete(appId int64, method string) bool {
	approve, ok := approvingByAppId[appId]
	if !ok {
		return true
	} else {
		stateStorage := make([]state.LimitState, 0)
		patternArray := approve.matcher.Match(method)
		for _, pattern := range patternArray {
			stateStorage = append(stateStorage, approve.limitStates[pattern])
		}
		return state.Update(stateStorage)
	}
}
