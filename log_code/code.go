package log_code

const (
	//grpc
	WarnProxyGrpcHandler                = 604 //metadata: {"typeData":"", "method":""}
	WarnConvertErrorDataMarshalResponse = 605
	ErrorGrpcClientDialing              = 606
	WarnJournalCouldNotWriteToFile      = 607
	WarnJournalClientDialing            = 608

	ErrorClientJournal = 903

	WarnHttpServerShutdown = 602
	ErrorHttpServerListen  = 603

	ErrorLocalConfig = 904
	ErrorClientGrpc  = 905
	ErrorClientHttp  = 906

	ErrorClientRedis = 907

	ErrorAuthenticate = 908
)
