package accounting

import (
	"github.com/stretchr/testify/assert"
	"isp-gate-service/conf"
	"isp-gate-service/entity"
	"isp-gate-service/model"
	"testing"
	"time"
)

var (
	approveSetting = conf.Accounting{
		Enable:          true,
		SnapshotTimeout: "2s",
		Setting: []conf.AccountingSetting{
			{ApplicationId: 1, Limits: []conf.LimitSetting{
				{Pattern: "mdm-master/group/method", MaxCount: 1, Timeout: "290ms"},
				{Pattern: "mdm-master/group3/method", MaxCount: 10, Timeout: "10s"},
				{Pattern: "mdm-master/group3/*", MaxCount: 5, Timeout: "10s"},
				{Pattern: "mdm-master/group4/method", MaxCount: 2, Timeout: "0s"},
			}},
			{ApplicationId: 2, Limits: []conf.LimitSetting{
				{Pattern: "mdm-master/group/method", MaxCount: 0, Timeout: "10s"},
			}},
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
	}
)

func TestApprove(t *testing.T) {
	a := assert.New(t)
	ReceiveConfiguration(approveSetting)
	model.SnapshotRep = &snapshotRepository{cache: make(map[int32]map[string]interface{})}

	expected := true
	req := reqExample[0]
	for _, path := range req.path {
		a.Equal(expected, GetAccounting(req.appId).Check(path))
		expected = !expected
		time.Sleep(150 * time.Millisecond)
	}

	//expected == true
	req = reqExample[1]
	for _, path := range req.path {
		a.Equal(expected, GetAccounting(req.appId).Check(path))
		expected = !expected
	}

	req = reqExample[2]
	expectedArray := []bool{true, true, true, true, true, false}
	for key, path := range req.path {
		a.Equal(expectedArray[key], GetAccounting(req.appId).Check(path))
	}

	//expected == true
	req = reqExample[3]
	for _, path := range req.path {
		a.Equal(expected, GetAccounting(req.appId).Check(path))
	}
}

type snapshotRepository struct {
	cache map[int32]map[string]interface{}
}

func (r *snapshotRepository) GetByApplication(appId int32) (*entity.Snapshot, error) {
	return &entity.Snapshot{
		AppId:      appId,
		LimitState: r.cache[appId],
	}, nil
}

func (r *snapshotRepository) Update(list []entity.Snapshot) error {
	for _, s := range list {
		r.cache[s.AppId] = s.LimitState
	}
	return nil
}

func (r *snapshotRepository) GetAll() ([]entity.Snapshot, error) {
	resp := make([]entity.Snapshot, 0)
	for appId, limitState := range r.cache {
		resp = append(resp, entity.Snapshot{
			AppId:      appId,
			LimitState: limitState,
		})
	}
	return resp, nil
}
