package accounting

import (
	log "github.com/integration-system/isp-log"
	"github.com/integration-system/isp-log/stdcodes"
	"isp-gate-service/conf"
	"isp-gate-service/entity"
	"isp-gate-service/log_code"
	"isp-gate-service/repository"
	"sync"
	"time"
)

var storage *storingTask

type storingTask struct {
	buffer  []entity.Request
	counter int

	chanTimeout <-chan time.Time
	chanCounter chan []entity.Request
	chanClose   chan bool

	mx sync.Mutex
	wg sync.WaitGroup

	process bool
}

func InitStoringTask(setting conf.StoringSetting) {
	timeout, err := time.ParseDuration(setting.Timeout)
	if err != nil {
		log.Fatal(stdcodes.ModuleInvalidRemoteConfig, err)
	}

	storage = newStoringTask(timeout, setting.Size)
}

func (t *storingTask) TakeRequest(appId int32, method string, date time.Time) {
	t.mx.Lock()
	if !t.process {
		t.mx.Unlock()
		return
	}

	t.buffer[t.counter] = entity.Request{
		AppId:     appId,
		Method:    method,
		CreatedAt: date,
	}

	t.counter++
	if t.counter == len(t.buffer) {
		t.chanCounter <- t.buffer
		t.clearBuffer()
	}

	t.mx.Unlock()
}

func (t *storingTask) Stop() {
	t.mx.Lock()
	if t.process {
		if t.counter != 0 {
			buffer := t.clearBuffer()
			defer func() {
				t.wg.Add(1)
				t.unload(buffer)
				t.wg.Wait()
			}()
		}
		t.chanClose <- true
		t.process = false
	}
	t.mx.Unlock()
}

func (t *storingTask) run(timeout time.Duration) {
	defer t.wg.Done()
	t.chanTimeout = time.After(timeout)
	for {
		select {
		case <-t.chanClose:
			return
		case cache := <-t.chanCounter:
			t.wg.Add(1)
			go t.unload(cache)
			t.chanTimeout = time.After(timeout)
		case <-t.chanTimeout:
			t.mx.Lock()
			if t.counter != 0 {
				buffer := t.clearBuffer()
				t.wg.Add(1)
				go t.unload(buffer)
			}
			t.mx.Unlock()
			t.chanTimeout = time.After(timeout)
		}
	}
}

func (t *storingTask) unload(cache []entity.Request) {
	if err := repository.RequestsRep.Insert(cache); err != nil {
		log.Error(log_code.ErrorUnloadAccounting, err)
	}
	t.wg.Done()
}

func (t *storingTask) clearBuffer() []entity.Request {
	oldBuffer := t.buffer[:t.counter]
	oldLen := len(t.buffer)
	t.counter = 0
	t.buffer = make([]entity.Request, oldLen)
	return oldBuffer
}

func newStoringTask(timeout time.Duration, bufSize int) *storingTask {
	task := &storingTask{
		mx:          sync.Mutex{},
		counter:     0,
		buffer:      make([]entity.Request, bufSize),
		process:     true,
		chanTimeout: make(chan time.Time),
		chanCounter: make(chan []entity.Request),
		chanClose:   make(chan bool),
	}
	task.wg.Add(1)
	go task.run(timeout)

	return task
}
