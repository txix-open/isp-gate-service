package model

import (
	"github.com/go-pg/pg/orm"
	"github.com/integration-system/isp-lib/database"
	"isp-gate-service/entity"
)

type (
	RequestsRepository interface {
		Insert([]entity.Request) error
	}

	requestsRepository struct {
		DB       orm.DB
		rxClient *database.RxDbClient
	}
)

func (r requestsRepository) Insert(model []entity.Request) error {
	_, err := r.getDb().Model(&model).Insert()
	return err
}

func (r requestsRepository) getDb() orm.DB {
	if r.DB != nil {
		return r.DB
	}
	return r.rxClient.Unsafe()
}
