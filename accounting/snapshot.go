package accounting

import (
	log "github.com/integration-system/isp-log"
	"github.com/integration-system/isp-log/stdcodes"
	"isp-gate-service/entity"
	"isp-gate-service/log_code"
	"isp-gate-service/model"
	"sync"
	"time"
)

var snapshot *snapshotTask

type snapshotTask struct {
	mx sync.Mutex
	wg sync.WaitGroup

	process bool
	timeout <-chan time.Time
	close   chan bool
}

func InitSnapshotTask(snapshotTimeout string) {
	timeout, err := time.ParseDuration(snapshotTimeout)
	if err != nil {
		log.Fatal(stdcodes.ModuleInvalidRemoteConfig, err)
	}

	snapshot = newSnapshotTask(timeout)
}

func (t *snapshotTask) Stop() {
	t.mx.Lock()
	if t.process {
		snapshotList := worker.takeSnapshot()
		defer func() {
			t.unload(snapshotList)
			t.wg.Wait()
		}()
		t.process = false
		t.close <- true
	}
	t.mx.Unlock()
}

func (t *snapshotTask) run(timeout time.Duration) {
	defer t.wg.Done()
	t.timeout = time.After(timeout)
	for {
		select {
		case <-t.close:
			return
		case <-t.timeout:
			list := worker.takeSnapshot()
			go t.unload(list)
			t.timeout = time.After(timeout)
		}
	}
}

func (t *snapshotTask) unload(list []entity.Snapshot) {
	t.wg.Add(1)
	if err := model.SnapshotRep.Update(list); err != nil {
		log.Error(log_code.ErrorSnapshotAccounting, err)
	}
	t.wg.Done()
}

func newSnapshotTask(timeout time.Duration) *snapshotTask {
	task := &snapshotTask{
		process: true,
		timeout: make(chan time.Time),
		close:   make(chan bool),
	}

	task.wg.Add(1)
	go task.run(timeout)
	return task
}
