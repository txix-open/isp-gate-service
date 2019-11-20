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
	accountingWorking = false
	accountingStorage = make(map[int32]Accounting)
	unloadingStorage  = make(map[int32]bool)
)

type Accounting interface {
	Accept(string) bool
	Snapshot() map[string]state.Snapshot
}

func ReceiveConfiguration(conf conf.Accounting) {
	Close()
	newAccountingStorage := make(map[int32]Accounting)
	newUnloadingStorage := make(map[int32]bool)

	if conf.Enable {
		if err := unload.Init(conf.Unload); err != nil {
			log.Fatal(stdcodes.ModuleInvalidRemoteConfig, err)
		}

		for _, s := range conf.Setting {
			newUnloadingStorage[s.ApplicationId] = s.EnableUnload

			limitStates, patternArray, err := state.InitLimitState(s.Limits)
			if err != nil {
				log.Fatal(stdcodes.ModuleInvalidRemoteConfig, err)
			}

			recoveryLimitState(s.ApplicationId, limitStates)

			newAccountingStorage[s.ApplicationId] = &accountant{
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

	unloadingStorage = newUnloadingStorage
	accountingStorage = newAccountingStorage
}

func Accept(appId int32, path string) bool {
	if unloadingStorage[appId] {
		unload.TakeRequest(appId, path, time.Now())
	}

	if accouter, ok := accountingStorage[appId]; !ok {
		return true
	} else {
		return accouter.Accept(path)
	}
}

func Close() {
	snapshot.Stop()
	unload.Stop()
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
