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

var unload = &unloadingTask{
	mx:          sync.Mutex{},
	counter:     0,
	process:     false,
	chanTimeout: make(chan time.Time),
	chanCounter: make(chan []entity.Unload),
	chanClose:   make(chan bool),
}

type unloadingTask struct {
	cache   []entity.Unload
	counter int

	process     bool
	chanTimeout <-chan time.Time
	chanCounter chan []entity.Unload
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
	u.mx.Lock()
	if !u.process {
		u.mx.Unlock()
		return
	}

	u.cache[u.counter] = entity.Unload{
		AppId:     appId,
		Method:    method,
		CreatedAt: date,
	}

	u.counter++
	if u.counter == len(u.cache) {
		u.chanCounter <- u.cache

		oldLen := len(u.cache)
		u.counter = 0
		u.cache = make([]entity.Unload, oldLen)
	}

	u.mx.Unlock()
}

func (u *unloadingTask) Stop() {
	u.mx.Lock()
	if u.process {
		if u.counter != 0 {
			defer u.unload(u.cache[:u.counter])
		}
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
		case cache := <-u.chanCounter:
			u.unload(cache)
			u.chanTimeout = time.After(timeout)
		case <-u.chanTimeout:
			u.mx.Lock()
			if u.counter != 0 {
				cache := u.cache[:u.counter]
				oldLen := len(u.cache)
				u.counter = 0
				u.cache = make([]entity.Unload, oldLen)
				u.mx.Unlock()

				u.unload(cache)
			} else {
				u.mx.Unlock()
			}
			u.chanTimeout = time.After(timeout)
		}
	}
}

func (u *unloadingTask) unload(cache []entity.Unload) {
	if err := model.UnloadRep.Insert(cache); err != nil {
		log.Error(log_code.ErrorUnloadAccounting, err)
	}
}
