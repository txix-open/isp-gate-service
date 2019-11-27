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

var Worker = &worker{
	process:           false,
	accountingStorage: make(map[int32]Accounting),
	requestsStoring:   make(map[int32]bool),
}

type (
	Accounting interface {
		Accept(string) bool
		Snapshot() (map[string]state.Snapshot, int64)
	}

	worker struct {
		mx                sync.RWMutex
		process           bool
		accountingStorage map[int32]Accounting
		requestsStoring   map[int32]bool
	}
)

func (w *worker) Consumer(list []structure.AddressConfiguration) bool {
	accountingSetting := config.GetRemote().(*conf.RemoteConfig).AccountingSetting
	w.ReceiveConfiguration(accountingSetting)
	return true
}

func (w *worker) ReceiveConfiguration(conf conf.Accounting) {
	w.Close()

	newAccountingStorage := make(map[int32]Accounting)
	newRequestsStoring := make(map[int32]bool)
	process := false

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

			version := w.recoveryAccounting(s.ApplicationId, limitStates)

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

		process = true
	}

	w.mx.Lock()
	w.requestsStoring = newRequestsStoring
	w.accountingStorage = newAccountingStorage
	w.process = process
	w.mx.Unlock()
}

func (w *worker) AcceptRequest(appId int32, path string) bool {
	w.mx.RLock()
	defer w.mx.RUnlock()
	if accouter, ok := w.accountingStorage[appId]; !ok {
		if w.requestsStoring[appId] {
			storage.TakeRequest(appId, path, time.Now())
		}
		return true
	} else {
		ok := accouter.Accept(path)
		if ok && w.requestsStoring[appId] {
			storage.TakeRequest(appId, path, time.Now())
		}
		return ok
	}
}

func (w *worker) Close() {
	snapshot.Stop()
	if storage != nil {
		storage.Stop()
	}
}

func (w *worker) takeSnapshot() []entity.Snapshot {
	w.mx.RLock()
	response := make([]entity.Snapshot, 0, len(w.accountingStorage))
	for appId, account := range w.accountingStorage {

		limitState, version := account.Snapshot()

		response = append(response, entity.Snapshot{
			AppId:      appId,
			LimitState: limitState,
			Version:    version,
		})
	}
	w.mx.RUnlock()
	return response
}

func (w *worker) recoveryAccounting(appId int32, limitStates map[string]state.LimitState) int64 {
	version := int64(0)
	w.mx.RLock()
	if w.process {
		if acc, ok := w.accountingStorage[appId]; ok {
			cashLimitState, cashVersion := acc.Snapshot()
			dbSnapshot, err := model.SnapshotRep.GetByApplication(appId)
			if err != nil {
				log.Warn(log_code.ErrorSnapshotAccounting, err)
			}

			if dbSnapshot != nil && dbSnapshot.Version > cashVersion {
				w.importLimitState(limitStates, dbSnapshot.LimitState)
			} else {
				w.importLimitState(limitStates, cashLimitState)
			}
		}
	} else {
		if snapshot, err := model.SnapshotRep.GetByApplication(appId); err != nil {
			log.Warn(log_code.ErrorSnapshotAccounting, err)
		} else if snapshot != nil {
			version = snapshot.Version
			w.importLimitState(limitStates, snapshot.LimitState)
		}
	}
	w.mx.RUnlock()
	return version
}

func (w *worker) importLimitState(limitStates map[string]state.LimitState, snapshot map[string]state.Snapshot) {
	for method, limitState := range limitStates {
		if oldState, ok := snapshot[method]; ok {
			limitState.Import(oldState)
		}
	}
}
