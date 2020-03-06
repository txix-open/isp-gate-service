//nolint
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
	accountingSetting = conf.Accounting{
		Enable:          true,
		SnapshotTimeout: "290ms",
		Setting: []conf.AccountingSetting{
			{ApplicationId: 1, Limits: []conf.LimitSetting{
				{Pattern: "mdm-master/group/method", MaxCount: 1, Timeout: "90ms"},
				{Pattern: "mdm-master/group3/method", MaxCount: 10, Timeout: "10s"},
				{Pattern: "mdm-master/group3/*", MaxCount: 5, Timeout: "10s"},
				{Pattern: "mdm-master/group4/method", MaxCount: 2, Timeout: "0s"},
				{Pattern: "mdm-master/group5/method", MaxCount: 3, Timeout: "10s"},
				{Pattern: "mdm-master/group6/method", MaxCount: 5, Timeout: "10s"},
			}},
			{ApplicationId: 2, Limits: []conf.LimitSetting{
				{Pattern: "mdm-master/group/method", MaxCount: 0, Timeout: "10s"},
			}},
		},
		Storing: conf.StoringSetting{
			Size:    100,
			Timeout: "1h",
		},
	}

	reqExample = []struct {
		appId int32
		path  []string
	}{
		0: {appId: 1, path: []string{
			"mdm-master/group/method", "mdm-master/group/method", "mdm-master/group/method",
			"mdm-master/group/method", "mdm-master/group/method", "mdm-master/group/method",
		}},

		1: {appId: 2, path: []string{
			"mdm-master/group3/method", "mdm-master/group/method", "mdm-master/group3/method",
			"mdm-master/group/method", "mdm-master/group3/method", "mdm-master/group/method",
		}},

		2: {appId: 1, path: []string{
			"mdm-master/group3/method", "mdm-master/group3/method", "mdm-master/group3/method",
			"mdm-master/group3/method", "mdm-master/group3/method", "mdm-master/group3/method",
		}},
		3: {appId: 1, path: []string{
			"mdm-master/group4/method", "mdm-master/group4/method", "mdm-master/group4/method",
			"mdm-master/group4/method", "mdm-master/group4/method", "mdm-master/group4/method",
		}},
		4: {appId: 1, path: []string{
			"mdm-master/group5/method", "mdm-master/group5/method", "mdm-master/group5/method",
		}},
	}
)

func TestAccounting(t *testing.T) {
	a := assert.New(t)
	snapshotRep := newSnapshotRepository(false)
	model.SnapshotRep = snapshotRep
	worker = &accountingWorker{
		accountingStorage: make(map[int32]Accounting),
		requestsStoring:   make(map[int32]bool),
	}
	worker.init(accountingSetting)

	req := reqExample[4]
	for _, path := range req.path {
		a.True(AcceptRequest(req.appId, path))
	}

	expected := true
	req = reqExample[0]
	for _, path := range req.path {
		a.Equal(expected, AcceptRequest(req.appId, path))
		expected = !expected
		time.Sleep(50 * time.Millisecond)
	}

	//expected == true
	req = reqExample[1]
	for _, path := range req.path {
		a.Equal(expected, AcceptRequest(req.appId, path))
		expected = !expected
	}

	req = reqExample[2]
	expectedArray := []bool{true, true, true, true, true, false}
	for key, path := range req.path {
		a.Equal(expectedArray[key], AcceptRequest(req.appId, path))
	}

	//expected == true
	req = reqExample[3]
	for _, path := range req.path {
		a.Equal(expected, AcceptRequest(req.appId, path))
	}

	worker.init(accountingSetting)

	req = reqExample[4]
	for _, path := range req.path {
		a.False(AcceptRequest(req.appId, path))
	}
}

func TestWorker_recoveryAccounting(t *testing.T) {
	a := assert.New(t)
	snapshotRep := newSnapshotRepository(true)
	model.SnapshotRep = snapshotRep

	worker = &accountingWorker{
		accountingStorage: make(map[int32]Accounting),
		requestsStoring:   make(map[int32]bool),
	}
	secondWorker := &accountingWorker{
		accountingStorage: make(map[int32]Accounting),
		requestsStoring:   make(map[int32]bool),
	}

	worker.init(accountingSetting)
	secondWorker.init(accountingSetting)

	appId := int32(1)
	request := "1_3_5"
	for range request {
		a.True(AcceptRequest(appId, "mdm-master/group6/method"))
	}
	a.False(AcceptRequest(appId, "mdm-master/group6/method"))

	time.Sleep(time.Millisecond * 300)
	a.True(snapshotRep.Wait())
	snapshot, err := model.SnapshotRep.GetByApplication(appId)
	a.NoError(err)
	a.NotNil(snapshot)
	a.Equal(len(request), int(snapshot.Version))

	Close()
	worker = secondWorker
	worker.init(accountingSetting)
	a.False(AcceptRequest(appId, "mdm-master/group6/method"))
}

type snapshotRepository struct {
	enableWaitGroup bool
	wg              sync.WaitGroup
	mx              sync.Mutex
	cache           map[int32]entity.Snapshot
}

func newSnapshotRepository(enableWaitGroup bool) *snapshotRepository {
	return &snapshotRepository{
		enableWaitGroup: enableWaitGroup,
		wg:              sync.WaitGroup{},
		cache:           make(map[int32]entity.Snapshot),
	}
}

func (r *snapshotRepository) Wait() bool {
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-time.After(time.Second):
		return false
	}
}

func (r *snapshotRepository) GetByApplication(appId int32) (*entity.Snapshot, error) {
	r.mx.Lock()
	defer r.mx.Unlock()

	if snapshot, ok := r.cache[appId]; ok {
		return &snapshot, nil
	} else {
		return nil, nil
	}
}

func (r *snapshotRepository) Update(list []entity.Snapshot) error {
	r.mx.Lock()
	defer r.mx.Unlock()

	if r.enableWaitGroup {
		r.wg.Add(1)
		defer r.wg.Done()
	}

	for _, s := range list {
		r.cache[s.AppId] = s
	}
	return nil
}
