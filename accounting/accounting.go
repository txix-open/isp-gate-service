package accounting

import (
	"github.com/integration-system/isp-lib/config"
	"github.com/integration-system/isp-lib/structure"
	log "github.com/integration-system/isp-log"
	"isp-gate-service/accounting/state"
	"isp-gate-service/conf"
	"isp-gate-service/entity"
	"isp-gate-service/log_code"
	"isp-gate-service/model"
	"isp-gate-service/service/matcher"
	"sync"
	"time"
)

var worker = &accountingWorker{
	accountingStorage: make(map[int32]Accounting),
	requestsStoring:   make(map[int32]bool),
}

type (
	Accounting interface {
		Accept(string) bool
		Snapshot() (map[string]state.Snapshot, int64)
	}

	accountingWorker struct {
		accountingStorage map[int32]Accounting
		requestsStoring   map[int32]bool
	}
)

func NewConnectionConsumer(list []structure.AddressConfiguration) bool {
	accountingSetting := config.GetRemote().(*conf.RemoteConfig).AccountingSetting
	worker.init(accountingSetting)
	return true
}

func ReceiveConfiguration(accountingSetting conf.Accounting) {
	worker.init(accountingSetting)
}

func AcceptRequest(appId int32, path string) bool {
	if accouter, ok := worker.accountingStorage[appId]; ok {
		ok := accouter.Accept(path)
		if ok && worker.requestsStoring[appId] {
			storage.TakeRequest(appId, path, time.Now())
		}
		return ok
	} else {
		if worker.requestsStoring[appId] {
			storage.TakeRequest(appId, path, time.Now())
		}
		return true
	}
}

func Close() {
	if snapshot != nil {
		snapshot.Stop()
	}
	if storage != nil {
		storage.Stop()
	}
}

func (w *accountingWorker) init(accountingSetting conf.Accounting) {
	Close()

	newAccountingStorage := make(map[int32]Accounting)
	newRequestsStoringStorage := make(map[int32]bool)

	if accountingSetting.Enable {
		InitStoringTask(accountingSetting.Storing)

		for _, s := range accountingSetting.Setting {
			limitStates, patternArray := state.InitLimitState(s.Limits)
			accounting := w.recoveryAccounting(s.ApplicationId, limitStates, patternArray)

			newAccountingStorage[s.ApplicationId] = accounting
			newRequestsStoringStorage[s.ApplicationId] = s.EnableStoring
		}

		InitSnapshotTask(accountingSetting.SnapshotTimeout)
	}

	w.requestsStoring = newRequestsStoringStorage
	w.accountingStorage = newAccountingStorage
}

func (w *accountingWorker) takeSnapshot() []entity.Snapshot {
	lastStorage := w.accountingStorage

	response := make([]entity.Snapshot, 0, len(lastStorage))
	for appId, account := range lastStorage {

		limitState, version := account.Snapshot()

		response = append(response, entity.Snapshot{
			AppId:      appId,
			LimitState: limitState,
			Version:    version,
		})
	}
	return response
}

func (w *accountingWorker) recoveryAccounting(appId int32, limitStates map[string]state.LimitState, patternArray []string) *accountant {
	var (
		version           = int64(0)
		importNotComplete = true
	)

	dbSnapshot, err := model.SnapshotRep.GetByApplication(appId)
	if err != nil {
		log.Warn(log_code.ErrorSnapshotAccounting, err)
	}

	if cash, ok := w.accountingStorage[appId]; ok {
		cashLimitState, cashVersion := cash.Snapshot()

		if dbSnapshot == nil || dbSnapshot.Version < cashVersion {
			version = cashVersion
			w.importLimitState(limitStates, cashLimitState)
			importNotComplete = false
		}
	}

	if dbSnapshot != nil && importNotComplete {
		version = dbSnapshot.Version
		w.importLimitState(limitStates, dbSnapshot.LimitState)
	}

	return &accountant{
		mx:          sync.Mutex{},
		matcher:     matcher.NewCacheableMatcher(patternArray),
		limitStates: limitStates,
		version:     version,
	}
}

func (w *accountingWorker) importLimitState(limitStates map[string]state.LimitState, snapshot map[string]state.Snapshot) {
	for method, limitState := range limitStates {
		if oldState, ok := snapshot[method]; ok {
			limitState.Import(oldState)
		}
	}
}
