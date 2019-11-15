package entity

type Snapshot struct {
	TableName  string `sql:"gate_service.snapshot" json:"-"`
	AppId      int32  `sql:",pk"`
	LimitState map[string]interface{}
}
