package entity

import (
	"isp-gate-service/accounting/state"
	"time"
)

type Snapshot struct {
	TableName  string `sql:"gate_service.snapshot" json:"-"`
	AppId      int32  `sql:",pk"`
	LimitState map[string]state.Snapshot
}

type Unload struct {
	TableName string `sql:"gate_service.unload" json:"-"`
	Id        int    `sql:",pk"`
	AppId     int32
	Method    string
	CreatedAt time.Time
}
