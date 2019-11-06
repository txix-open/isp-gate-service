package approve

import (
	"github.com/stretchr/testify/assert"
	"isp-gate-service/conf"
	"testing"
	"time"
)

var (
	approveSetting = []conf.ApproveSetting{
		{ApplicationId: 1, Limits: []conf.LimitSetting{
			{Pattern: "mdm-master/group/method", MaxCount: 1, Lifetime: "290ms"},
			{Pattern: "mdm-master/group3/method", MaxCount: 10, Lifetime: "10s"},
			{Pattern: "mdm-master/group3/*", MaxCount: 5, Lifetime: "10s"},
			{Pattern: "mdm-master/group4/method", MaxCount: 2, Lifetime: "0s"},
		}},
		{ApplicationId: 2, Limits: []conf.LimitSetting{
			{Pattern: "mdm-master/group/method", MaxCount: 0, Lifetime: "10s"},
		}},
	}

	reqExample = []struct {
		appId int64
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

	expected := true
	req := reqExample[0]
	for _, path := range req.path {
		a.Equal(expected, GetApprove(req.appId).Complete(path))
		expected = !expected
		time.Sleep(150 * time.Millisecond)
	}

	//expected == true
	req = reqExample[1]
	for _, path := range req.path {
		a.Equal(expected, GetApprove(req.appId).Complete(path))
		expected = !expected
	}

	req = reqExample[2]
	expectedArray := []bool{true, true, true, true, true, false}
	for key, path := range req.path {
		a.Equal(expectedArray[key], GetApprove(req.appId).Complete(path))
	}

	//expected == true
	req = reqExample[3]
	for _, path := range req.path {
		a.Equal(expected, GetApprove(req.appId).Complete(path))
	}
}
