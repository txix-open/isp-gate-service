package model

import (
	"github.com/go-pg/pg/orm"
	"github.com/integration-system/isp-lib/database"
	"isp-gate-service/entity"
)

type (
	UnloadRepository interface {
		Insert([]entity.Unload) error
	}

	unloadRepository struct {
		DB       orm.DB
		rxClient *database.RxDbClient
	}
)

func (r unloadRepository) Insert(model []entity.Unload) error {
	_, err := r.getDb().Model(&model).Insert()
	return err
}

func (r unloadRepository) getDb() orm.DB {
	if r.DB != nil {
		return r.DB
	}
	return r.rxClient.Unsafe()
}
