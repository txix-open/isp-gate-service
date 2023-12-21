package conf

const (
	HttpProtocol = "http"
	GrpcProtocol = "grpc"
	WsProtocol   = "ws"
)

type Local struct {
	Locations []Location
}

type Location struct {
	SkipAuth     bool
	PathPrefix   string `validate:"required"`
	Protocol     string `validate:"required,oneof=http grpc ws"`
	TargetModule string `validate:"required"`
}
