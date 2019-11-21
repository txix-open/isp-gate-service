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

var storage = &storingTask{
	mx:          sync.Mutex{},
	counter:     0,
	process:     false,
	chanTimeout: make(chan time.Time),
	chanCounter: make(chan []entity.Request),
	chanClose:   make(chan bool),
}

type storingTask struct {
	buffer  []entity.Request
	counter int

	process     bool
	chanTimeout <-chan time.Time
	chanCounter chan []entity.Request
	chanClose   chan bool

	mx sync.Mutex
}

func (u *storingTask) Init(setting conf.StoringSetting) error {
	u.mx.Lock()
	u.process = true
	u.buffer = make([]entity.Request, setting.Size)
	u.counter = 0
	u.mx.Unlock()

	timeout, err := time.ParseDuration(setting.Timeout)
	if err != nil {
		return err
	}
	go u.run(timeout)
	return nil
}

func (u *storingTask) TakeRequest(appId int32, method string, date time.Time) {
	u.mx.Lock()
	if !u.process {
		u.mx.Unlock()
		return
	}

	u.buffer[u.counter] = entity.Request{
		AppId:     appId,
		Method:    method,
		CreatedAt: date,
	}

	u.counter++
	if u.counter == len(u.buffer) {
		u.chanCounter <- u.buffer

		oldLen := len(u.buffer)
		u.counter = 0
		u.buffer = make([]entity.Request, oldLen)
	}

	u.mx.Unlock()
}

func (u *storingTask) Stop() {
	u.mx.Lock()
	if u.process {
		if u.counter != 0 {
			defer u.unload(u.buffer[:u.counter])
		}
		u.chanClose <- true
		u.process = false
	}
	u.mx.Unlock()
}

func (u *storingTask) run(timeout time.Duration) {
	u.chanTimeout = time.After(timeout)
	for {
		select {
		case <-u.chanClose:
			return
		case cache := <-u.chanCounter:
			go u.unload(cache)
			u.chanTimeout = time.After(timeout)
		case <-u.chanTimeout:
			u.mx.Lock()
			if u.counter != 0 {
				cache := u.buffer[:u.counter]
				oldLen := len(u.buffer)
				u.counter = 0
				u.buffer = make([]entity.Request, oldLen)
				u.mx.Unlock()

				go u.unload(cache)
			} else {
				u.mx.Unlock()
			}
			u.chanTimeout = time.After(timeout)
		}
	}
}

func (u *storingTask) unload(cache []entity.Request) {
	if err := model.RequestsRep.Insert(cache); err != nil {
		log.Error(log_code.ErrorUnloadAccounting, err)
	}
}
