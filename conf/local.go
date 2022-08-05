package conf

const (
	HttpProtocol = "http"
	GrpcProtocol = "grpc"
)

type Local struct {
	Locations []Location
}

type Location struct {
	SkipAuth     bool
	PathPrefix   string `valid:"required"`
	Protocol     string `valid:"required,in(http|grpc)"`
	TargetModule string `valid:"required"`
}
