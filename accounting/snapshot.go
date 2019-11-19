package accounting

import (
	log "github.com/integration-system/isp-log"
	"isp-gate-service/log_code"
	"isp-gate-service/model"
	"sync"
	"time"
)

var snapshot = &snapshotTask{
	process: false,
	timeout: make(chan time.Time),
	close:   make(chan bool),
}

type snapshotTask struct {
	mx      sync.Mutex
	process bool
	timeout <-chan time.Time
	close   chan bool
}

func (s *snapshotTask) Start(timeout time.Duration) {
	s.Stop()

	s.mx.Lock()
	s.process = true
	s.timeout = time.After(timeout)
	s.mx.Unlock()

	for {
		select {
		case <-s.close:
			return
		case <-s.timeout:
			if err := s.complete(); err != nil {
				log.Error(log_code.ErrorSnapshotAccounting, err)
			}
			s.timeout = time.After(timeout)
		}
	}
}

func (s *snapshotTask) Stop() {
	s.mx.Lock()
	if s.process {
		s.process = false
		s.close <- true
	}
	s.mx.Unlock()
}

func (s *snapshotTask) complete() error {
	snapshotList := takeSnapshot()
	return model.SnapshotRep.Update(snapshotList)
}
