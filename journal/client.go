package journal

import (
	"github.com/integration-system/isp-journal/rx"
	"github.com/integration-system/isp-lib/backend"
	"github.com/integration-system/isp-lib/structure"
	log "github.com/integration-system/isp-log"
	"google.golang.org/grpc"
	"isp-gate-service/log_code"
)

var (
	journalServiceClient = backend.NewRxGrpcClient(
		backend.WithDialOptions(grpc.WithInsecure(), grpc.WithBlock()),
		backend.WithDialingErrorHandler(func(err error) {
			log.Warnf(log_code.ErrorClientJournal, "journal client dialing err: %v", err)
		}),
	)
	Client = rx.NewDefaultRxJournal(journalServiceClient)
)

func RequiredModule() (string, func([]structure.AddressConfiguration) bool, bool) {
	return "journal", receiveJournalServiceAddressList, false
}

func receiveJournalServiceAddressList(list []structure.AddressConfiguration) bool {
	if ok := journalServiceClient.ReceiveAddressList(list); !ok {
		return false
	} else {
		go Client.CollectAndTransferExistedLogs()
		return true
	}
}
