package log_code

const (
	WarnHttpServerShutdown = 602
	ErrorHttpServerListen  = 603

	WarnProxyGrpcHandler                = 604 //metadata: {"typeData":"", "method":""}
	WarnConvertErrorDataMarshalResponse = 605

	ErrorClientGrpc = 606

	WarnJournalCouldNotWriteToFile = 607

	ErrorClientJournal = 608
	ErrorClientRedis   = 609
	ErrorClientHttp    = 611

	ErrorAuthenticate = 612

	ErrorClientDatabase     = 615
	ErrorSnapshotAccounting = 616
	ErrorUnloadAccounting   = 617

	ErrorWebsocketProxy = 618
)
