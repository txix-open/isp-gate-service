package invoker

import (
	"github.com/integration-system/isp-journal/rx"
	"github.com/integration-system/isp-lib/v2/backend"
	"github.com/integration-system/isp-lib/v2/structure"
	"google.golang.org/grpc"
)

type journalService struct {
	*rx.RxJournal
}

var (
	journalClient = backend.NewRxGrpcClient(
		backend.WithDialOptions(grpc.WithInsecure()),
	)
	Journal = &journalService{RxJournal: rx.NewDefaultRxJournal(journalClient)}
)

func (*journalService) ReceiveServiceAddressList(list []structure.AddressConfiguration) bool {
	if ok := journalClient.ReceiveAddressList(list); !ok {
		return false
	}
	go Journal.CollectAndTransferExistedLogs()
	return true
}
