package accounting

import (
	log "github.com/integration-system/isp-log"
	"github.com/integration-system/isp-log/stdcodes"
	"isp-gate-service/accounting/state"
	"isp-gate-service/conf"
	"isp-gate-service/log_code"
	"isp-gate-service/model"
	"isp-gate-service/service/matcher"
	"sync"
	"time"
)

var accountingByApplicationId = make(map[int32]Accounting)

type Accounting interface {
	Check(string) bool
	getLimitState() map[string]state.LimitState
}

func ReceiveConfiguration(conf conf.Accounting) {
	snapshot.Stop()

	newAccountingByApplicationId := make(map[int32]Accounting)
	if conf.Enable {
		for _, s := range conf.Setting {
			limitStates, patternArray, err := state.InitLimitState(s.Limits)
			if err != nil {
				log.Fatal(stdcodes.ModuleInvalidRemoteConfig, err)
			}

			if snapshot, err := model.SnapshotRep.GetByApplication(s.ApplicationId); err != nil {
				log.Warn(log_code.ErrorSnapshotAccounting, err)
			} else if snapshot != nil {
				for method, limitState := range limitStates {
					if oldLimitState, ok := snapshot.LimitState[method]; ok {
						if err := limitState.Import(oldLimitState); err != nil {
							log.Warn(log_code.ErrorSnapshotAccounting, err)
						}
					}
				}
			}

			newAccountingByApplicationId[s.ApplicationId] = &accountant{
				matcher:     matcher.NewCacheableMatcher(patternArray),
				limitStates: limitStates,
				mx:          sync.Mutex{},
			}
		}
		if snapshotTimeout, err := time.ParseDuration(conf.SnapshotTimeout); err != nil {
			log.Fatal(stdcodes.ModuleInvalidRemoteConfig, err)
		} else {
			go snapshot.Start(snapshotTimeout)
		}
	}
	accountingByApplicationId = newAccountingByApplicationId
}

func Close() {
	snapshot.Stop()
}

func GetAccounting(appId int32) Accounting {
	return accountingByApplicationId[appId]
}
