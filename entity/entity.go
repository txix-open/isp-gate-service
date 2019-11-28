package entity

import (
	"isp-gate-service/accounting/state"
	"time"
)

type Snapshot struct {
	TableName  string `sql:"?db_schema.snapshot" json:"-"`
	AppId      int32  `sql:",pk"`
	LimitState map[string]state.Snapshot
	Version    int64
}

type Request struct {
	TableName string `sql:"?db_schema.requests" json:"-"`
	Id        int    `sql:",pk"`
	AppId     int32
	Method    string
	CreatedAt time.Time
}
