package log_code

const (
	WarnProxyGrpcHandler                = 604 //metadata: {"typeData":"", "method":""}
	WarnConvertErrorDataMarshalResponse = 605
	WarnJournalCouldNotWriteToFile      = 607

	WarnHttpServerShutdown = 602
	ErrorHttpServerListen  = 603

	FatalLocalConfig = 610
	ErrorClientGrpc  = 606
	ErrorClientHttp  = 611

	ErrorClientJournal = 608
	ErrorClientRedis   = 609

	ErrorAuthenticate         = 612
	FatalConfigApproveSetting = 613
)
