package accounting

import (
	log "github.com/integration-system/isp-log"
	"isp-gate-service/conf"
	"isp-gate-service/entity"
	"isp-gate-service/log_code"
	"isp-gate-service/model"
	"sync"
	"time"
)

var Unload = &unloadingTask{
	mx:          sync.Mutex{},
	counter:     0,
	process:     false,
	chanTimeout: make(chan time.Time),
	chanCounter: make(chan bool),
	chanClose:   make(chan bool),
}

type unloadingTask struct {
	cache   []entity.Unload
	counter int

	process     bool
	chanTimeout <-chan time.Time
	chanCounter chan bool
	chanClose   chan bool

	mx sync.Mutex
}

func (u *unloadingTask) Init(setting conf.UnloadSetting) error {
	u.mx.Lock()
	u.process = true
	u.cache = make([]entity.Unload, setting.Count)
	u.counter = 0
	u.mx.Unlock()

	timeout, err := time.ParseDuration(setting.Timeout)
	if err != nil {
		return err
	}
	go u.run(timeout)
	return nil
}

func (u *unloadingTask) TakeRequest(appId int32, method string, date time.Time) {
	if !u.process {
		return
	}

	u.mx.Lock()
	u.cache[u.counter] = entity.Unload{
		AppId:     appId,
		Method:    method,
		CreatedAt: date,
	}

	u.counter++
	if u.counter == len(u.cache) {
		u.chanCounter <- true
	}

	u.mx.Unlock()
}

func (u *unloadingTask) Stop() {
	u.mx.Lock()
	if u.process {
		defer u.unload()
		u.chanClose <- true
		u.process = false
	}
	u.mx.Unlock()
}

func (u *unloadingTask) run(timeout time.Duration) {
	u.chanTimeout = time.After(timeout)
	for {
		select {
		case <-u.chanClose:
			return
		case <-u.chanCounter:
			u.unload()
			u.chanTimeout = time.After(timeout)
		case <-u.chanTimeout:
			u.unload()
			u.chanTimeout = time.After(timeout)
		}
	}
}

func (u *unloadingTask) unload() {
	u.mx.Lock()
	if u.counter == 0 {
		u.mx.Unlock()
		return
	}

	oldLen := len(u.cache)
	if err := model.UnloadRep.Insert(u.cache[:u.counter]); err != nil {
		log.Error(log_code.ErrorUnloadAccounting, err)
	}
	u.counter = 0
	u.cache = make([]entity.Unload, oldLen)

	u.mx.Unlock()
}
