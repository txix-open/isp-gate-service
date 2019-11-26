package accounting

import (
	"github.com/integration-system/isp-lib/config"
	"github.com/integration-system/isp-lib/structure"
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
	requestsStoring   = make(map[int32]bool)
)

type Accounting interface {
	Accept(string) bool
	Snapshot() map[string]state.Snapshot
	GetVersion() int64
}

func Consumer(list []structure.AddressConfiguration) bool {
	accountingSetting := config.GetRemote().(*conf.RemoteConfig).AccountingSetting
	receiveConfiguration(accountingSetting)
	return true
}

func receiveConfiguration(conf conf.Accounting) {
	Close()

	newAccountingStorage := make(map[int32]Accounting)
	newRequestsStoring := make(map[int32]bool)

	if conf.Enable {
		if err := InitStoringTask(conf.Storing); err != nil {
			log.Fatal(stdcodes.ModuleInvalidRemoteConfig, err)
		}

		for _, s := range conf.Setting {
			newRequestsStoring[s.ApplicationId] = s.EnableStoring

			limitStates, patternArray, err := state.InitLimitState(s.Limits)
			if err != nil {
				log.Fatal(stdcodes.ModuleInvalidRemoteConfig, err)
			}

			version := recoveryAccounting(s.ApplicationId, limitStates)

			newAccountingStorage[s.ApplicationId] = &accountant{
				version:     version,
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

	requestsStoring = newRequestsStoring
	accountingStorage = newAccountingStorage
}

func Accept(appId int32, path string) bool {
	if accouter, ok := accountingStorage[appId]; !ok {
		if requestsStoring[appId] {
			storage.TakeRequest(appId, path, time.Now())
		}
		return true
	} else {
		ok := accouter.Accept(path)
		if ok && requestsStoring[appId] {
			storage.TakeRequest(appId, path, time.Now())
		}
		return ok
	}
}

func Close() {
	snapshot.Stop()
	if storage != nil {
		storage.Stop()
	}
}

func takeSnapshot() []entity.Snapshot {
	response := make([]entity.Snapshot, 0, len(accountingStorage))
	for appId, account := range accountingStorage {
		response = append(response, entity.Snapshot{
			AppId:      appId,
			LimitState: account.Snapshot(),
			Version:    account.GetVersion(),
		})
	}
	return response
}

func recoveryAccounting(appId int32, limitStates map[string]state.LimitState) int64 {
	version := int64(0)
	if accountingWorking {
		if acc, ok := accountingStorage[appId]; ok {
			version = acc.GetVersion()
			snapshot := acc.Snapshot()
			importLimitState(limitStates, snapshot)
		}
	} else {
		if snapshot, err := model.SnapshotRep.GetByApplication(appId); err != nil {
			log.Warn(log_code.ErrorSnapshotAccounting, err)
		} else if snapshot != nil {
			version = snapshot.Version
			importLimitState(limitStates, snapshot.LimitState)
		}
	}
	return version
}

func importLimitState(limitStates map[string]state.LimitState, snapshot map[string]state.Snapshot) {
	for method, limitState := range limitStates {
		if oldState, ok := snapshot[method]; ok {
			limitState.Import(oldState)
		}
	}
}
