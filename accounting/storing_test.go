package accounting

import (
	"github.com/stretchr/testify/assert"
	"isp-gate-service/conf"
	"isp-gate-service/entity"
	"isp-gate-service/model"
	"sync"
	"testing"
	"time"
)

var (
	timeoutSetting = conf.StoringSetting{
		Size:    3,
		Timeout: "30ms",
	}

	bufferSetting = conf.StoringSetting{
		Size:    3,
		Timeout: "2s",
	}

	unloadSetting = conf.StoringSetting{
		Size:    3,
		Timeout: "100ms",
	}
)

func initStoring(setting conf.StoringSetting) *requestsRepository {
	repository := &requestsRepository{cache: make([]entity.Request, 0), wg: &sync.WaitGroup{}}
	model.RequestsRep = repository
	InitStoringTask(setting)
	return repository
}

func wait(wg *sync.WaitGroup) bool {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-time.After(time.Second):
		return false
	}
}

func TestStoringTask_Stop(t *testing.T) {
	a := assert.New(t)
	rep := initStoring(bufferSetting)

	rep.wg.Add(2)
	for i, j := range []string{"Stop_1", "Stop_2", "Stop_3", "Stop_4", "Stop_5"} {
		storage.TakeRequest(int32(i), j, time.Now())
	}
	storage.Stop()
	a.Len(rep.cache, 5)
}

func TestStoringTask_Buffer(t *testing.T) {
	a := assert.New(t)
	rep := initStoring(bufferSetting)

	rep.wg.Add(1)
	for i, j := range []string{"Buffer_1", "Buffer_2", "Buffer_3", "Buffer_4", "Buffer_5"} {
		storage.TakeRequest(int32(i), j, time.Now())
	}
	a.True(wait(rep.wg))
	a.Len(rep.cache, 3)
}

func TestStoringTask_Timeout(t *testing.T) {
	a := assert.New(t)
	rep := initStoring(timeoutSetting)

	rep.wg.Add(1)
	for i, j := range []string{"Timeout_1", "Timeout_2"} {
		storage.TakeRequest(int32(i), j, time.Now())
	}
	a.True(wait(rep.wg))
	a.Len(rep.cache, 2)
}

func TestStoringTask_Unload(t *testing.T) {
	a := assert.New(t)
	rep := initStoring(unloadSetting)

	rep.wg.Add(2)
	for i, j := range []string{"Unload_1", "Unload_2", "Unload_3", "Unload_4", "Unload_5"} {
		storage.TakeRequest(int32(i), j, time.Now())
	}
	a.True(wait(rep.wg))
	a.Len(rep.cache, 5)
}

type requestsRepository struct {
	cache []entity.Request
	mx    sync.Mutex
	wg    *sync.WaitGroup
}

func (r *requestsRepository) Insert(model []entity.Request) error {
	r.mx.Lock()
	r.cache = append(r.cache, model...)
	r.mx.Unlock()
	r.wg.Done()
	return nil
}
