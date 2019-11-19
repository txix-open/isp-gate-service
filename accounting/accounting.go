package accounting

import (
	log "github.com/integration-system/isp-log"
	"github.com/integration-system/isp-log/stdcodes"
	"isp-gate-service/accounting/state"
	"isp-gate-service/conf"
	"isp-gate-service/entity"
	"isp-gate-service/log_code"
	"isp-gate-service/model"
	"isp-gate-service/service/matcher"
	"sync"
	"time"
)

var (
	accountingStorage = make(map[int32]Accounting)
	accountingWorking = false
)

type Accounting interface {
	Accept(string) bool
	Snapshot() map[string]state.Snapshot
}

func ReceiveConfiguration(conf conf.Accounting) {
	snapshot.Stop()
	Unload.Stop()
	newAccountingByApplicationId := make(map[int32]Accounting)

	if conf.Enable {
		if err := Unload.Init(conf.Unload); err != nil {
			log.Fatal(stdcodes.ModuleInvalidRemoteConfig, err)
		}

		for _, s := range conf.Setting {
			limitStates, patternArray, err := state.InitLimitState(s.Limits)
			if err != nil {
				log.Fatal(stdcodes.ModuleInvalidRemoteConfig, err)
			}

			recoveryLimitState(s.ApplicationId, limitStates)

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

		accountingWorking = true
	} else {
		accountingWorking = false
	}

	accountingStorage = newAccountingByApplicationId
}

func Close() {
	snapshot.Stop()
	Unload.Stop()
}

func GetAccounting(appId int32) Accounting {
	return accountingStorage[appId]
}

func takeSnapshot() []entity.Snapshot {
	response := make([]entity.Snapshot, 0)
	for appId, account := range accountingStorage {
		response = append(response, entity.Snapshot{
			AppId:      appId,
			LimitState: account.Snapshot(),
		})
	}
	return response
}

func recoveryLimitState(appId int32, limitStates map[string]state.LimitState) {
	if accountingWorking {
		if acc, ok := accountingStorage[appId]; ok {
			snapshot := acc.Snapshot()
			importLimitState(limitStates, snapshot)
		}
	} else {
		if snapshot, err := model.SnapshotRep.GetByApplication(appId); err != nil {
			log.Warn(log_code.ErrorSnapshotAccounting, err)
		} else if snapshot != nil {
			importLimitState(limitStates, snapshot.LimitState)
		}
	}
}

func importLimitState(limitStates map[string]state.LimitState, snapshot map[string]state.Snapshot) {
	for method, limitState := range limitStates {
		if oldState, ok := snapshot[method]; ok {
			limitState.Import(oldState)
		}
	}
}
