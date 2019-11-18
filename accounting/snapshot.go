package accounting

import (
	log "github.com/integration-system/isp-log"
	"isp-gate-service/accounting/state"
	"isp-gate-service/entity"
	"isp-gate-service/log_code"
	"isp-gate-service/model"
	"sync"
	"time"
)

var snapshot = &snapshotStruct{
	process: false,
	timeout: make(chan time.Time),
	close:   make(chan bool),
}

type snapshotStruct struct {
	wg      sync.WaitGroup
	process bool
	timeout <-chan time.Time
	close   chan bool
}

func (s *snapshotStruct) Start(timeout time.Duration) {
	s.Stop()
	s.process = true
	s.timeout = time.After(timeout)

	for {
		select {
		case <-s.close:
			s.wg.Wait()
			return
		case <-s.timeout:
			s.wg.Add(1)
			applicationAccounting := accountingByApplicationId
			if err := s.complete(applicationAccounting); err != nil {
				log.Error(log_code.ErrorSnapshotAccounting, err)
			}
			s.timeout = time.After(timeout)
			s.wg.Done()
		}
	}
}

func (s *snapshotStruct) Stop() {
	if s.process {
		s.process = false
		s.close <- true
	}
	s.wg.Wait()
}

func (s *snapshotStruct) complete(applicationAccounting map[int32]Accounting) error {
	snapshot := make([]entity.Snapshot, 0)
	for application, account := range applicationAccounting {
		snapshotLimitState := make(map[string]state.Snapshot)
		limitStates := account.getLimitState()
		for method, limitState := range limitStates {
			snapshotLimitState[method] = limitState.Export()
		}
		snapshot = append(snapshot, entity.Snapshot{
			AppId:      application,
			LimitState: snapshotLimitState,
		})
	}
	return model.SnapshotRep.Update(snapshot)
}
