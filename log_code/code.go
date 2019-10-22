package log_code

const (
	InfoOnLocalConfigLoad = 601

	WarnRequestHandler                  = 604 //metadata: {"typeData":"", "method":""}
	WarnConvertErrorDataMarshalResponse = 605
	ErrorGrpcClientDialing              = 606
	WarnJournalCouldNotWriteToFile      = 607
	WarnJournalClientDialing            = 608

	ErrorClientJournal = 903

	WarnCreateRestServerHttpSrvShutdown        = 602
	ErrorCreateRestServerHttpSrvListenAndServe = 603

	ErrorLocalConfig = 904
	ErrorClientGrpc  = 905
	ErrorClientHttp  = 906
)
