package log_code

const (
	WarnProxyGrpcHandler                = 901 //metadata: {"typeData":"", "method":""}
	WarnConvertErrorDataMarshalResponse = 902
	WarnJournalCouldNotWriteToFile      = 903

	WarnHttpServerShutdown = 904
	ErrorHttpServerListen  = 905

	ErrorLocalConfig = 906
	ErrorClientGrpc  = 907
	ErrorClientHttp  = 908

	ErrorClientJournal = 909
	ErrorClientRedis   = 910

	ErrorAuthenticate = 911
)
