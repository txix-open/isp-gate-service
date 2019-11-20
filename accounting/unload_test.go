package accounting

import (
	"github.com/stretchr/testify/assert"
	"isp-gate-service/conf"
	"isp-gate-service/entity"
	"isp-gate-service/model"
	"testing"
	"time"
)

var unloadingSetting = conf.UnloadSetting{
	Count:   3,
	Timeout: "50ms",
}

func TestUnloadingTask(t *testing.T) {
	a := assert.New(t)
	repository := &unloadRepository{cache: make([]entity.Unload, 0)}
	model.UnloadRep = repository

	err := unload.Init(unloadingSetting)
	a.NoError(err)

	for i, j := range []string{"1", "2", "3", "4", "5"} {
		unload.TakeRequest(int32(i), j, time.Now())
	}
	a.Len(repository.cache, 3)
	time.Sleep(time.Millisecond * 60)
	a.Len(repository.cache, 5)

	for i, j := range []string{"1", "2", "3", "4", "5"} {
		unload.TakeRequest(int32(i), j, time.Now())
	}
	unload.Stop()
	a.Len(repository.cache, 10)
}

type unloadRepository struct {
	cache []entity.Unload
}

func (r *unloadRepository) Insert(model []entity.Unload) error {
	r.cache = append(r.cache, model...)
	return nil
}
