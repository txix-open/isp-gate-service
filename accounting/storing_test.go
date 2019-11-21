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

var unloadingSetting = conf.StoringSetting{
	Size:    3,
	Timeout: "50ms",
}

func TestStoringTask(t *testing.T) {
	a := assert.New(t)
	repository := &requestsRepository{cache: make([]entity.Request, 0)}
	model.RequestsRep = repository

	err := storage.Init(unloadingSetting)
	a.NoError(err)

	for i, j := range []string{"1", "2", "3", "4", "5"} {
		storage.TakeRequest(int32(i), j, time.Now())
	}
	a.Len(repository.cache, 3)
	time.Sleep(time.Millisecond * 60)
	a.Len(repository.cache, 5)

	for i, j := range []string{"1", "2", "3", "4", "5"} {
		storage.TakeRequest(int32(i), j, time.Now())
	}
	storage.Stop()
	a.Len(repository.cache, 10)
}

type requestsRepository struct {
	cache []entity.Request
	mx    sync.Mutex
}

func (r *requestsRepository) Insert(model []entity.Request) error {
	r.mx.Lock()
	r.cache = append(r.cache, model...)
	r.mx.Unlock()
	return nil
}
