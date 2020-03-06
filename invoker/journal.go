package invoker

import (
	"github.com/integration-system/isp-journal/rx"
	"github.com/integration-system/isp-lib/v2/backend"
	"github.com/integration-system/isp-lib/v2/structure"
	log "github.com/integration-system/isp-log"
	"google.golang.org/grpc"
	"isp-gate-service/log_code"
)

type journalService struct {
	*rx.RxJournal
}

var (
	journalClient = backend.NewRxGrpcClient(
		backend.WithDialOptions(grpc.WithInsecure(), grpc.WithBlock()),
		backend.WithDialingErrorHandler(func(err error) {
			log.Warnf(log_code.ErrorClientJournal, "journal client dialing err: %v", err)
		}),
	)
	Journal = &journalService{RxJournal: rx.NewDefaultRxJournal(journalClient)}
)

func (*journalService) ReceiveServiceAddressList(list []structure.AddressConfiguration) bool {
	if ok := journalClient.ReceiveAddressList(list); !ok {
		return false
	} else {
		go Journal.CollectAndTransferExistedLogs()
		return true
	}
}
