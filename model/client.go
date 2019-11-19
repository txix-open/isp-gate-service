package model

import (
	"github.com/integration-system/isp-lib/database"
	log "github.com/integration-system/isp-log"
	"isp-gate-service/log_code"
)

var (
	DbClient = database.NewRxDbClient(
		database.WithSchemaEnsuring(),
		database.WithSchemaAutoInjecting(),
		database.WithMigrationsEnsuring(),
		database.WithInitializingErrorHandler(func(err *database.ErrorEvent) {
			log.Error(log_code.ErrorClientDatabase, err)
		}))

	SnapshotRep SnapshotRepository = snapshotRepository{rxClient: DbClient}
	UnloadRep   UnloadRepository   = unloadRepository{rxClient: DbClient}
)
