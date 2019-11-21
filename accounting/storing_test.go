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

func initStoring(setting conf.StoringSetting) (*requestsRepository, error) {
	repository := &requestsRepository{cache: make([]entity.Request, 0), wg: &sync.WaitGroup{}}
	model.RequestsRep = repository

	err := storage.Init(setting)
	return repository, err
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
	rep, err := initStoring(bufferSetting)
	a.NoError(err)

	rep.wg.Add(2)
	for i, j := range []string{"1", "2", "3", "4", "5"} {
		storage.TakeRequest(int32(i), j, time.Now())
	}
	storage.Stop()
	a.Len(rep.cache, 5)
}

func TestStoringTask_Buffer(t *testing.T) {
	a := assert.New(t)
	rep, err := initStoring(bufferSetting)
	a.NoError(err)

	rep.wg.Add(1)
	for i, j := range []string{"1", "2", "3", "4", "5"} {
		storage.TakeRequest(int32(i), j, time.Now())
	}
	a.True(wait(rep.wg))
	a.Len(rep.cache, 3)
}

func TestStoringTask_Timeout(t *testing.T) {
	a := assert.New(t)
	rep, err := initStoring(timeoutSetting)
	a.NoError(err)

	rep.wg.Add(1)
	for i, j := range []string{"1", "2"} {
		storage.TakeRequest(int32(i), j, time.Now())
	}
	a.True(wait(rep.wg))
	a.Len(rep.cache, 2)
}

func TestStoringTask_Unload(t *testing.T) {
	a := assert.New(t)
	rep, err := initStoring(unloadSetting)
	a.NoError(err)

	rep.wg.Add(2)
	for i, j := range []string{"1", "2", "3", "4", "5"} {
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
	r.cache = append(r.cache, model...)
	r.wg.Done()
	return nil
}
