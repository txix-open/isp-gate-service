module isp-gate-service

go 1.16

require (
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/fasthttp/websocket v1.4.3-rc.7
	github.com/go-pg/pg/v9 v9.2.1
	github.com/go-redis/redis/v8 v8.8.2
	github.com/golang/protobuf v1.5.2
	github.com/integration-system/go-cmp v0.0.0-20190131081942-ac5582987a2f
	github.com/integration-system/isp-kit v1.6.0
	github.com/integration-system/isp-lib/v2 v2.8.7
	github.com/integration-system/isp-log v1.1.8
	github.com/json-iterator/go v1.1.12
	github.com/pkg/errors v0.9.1
	github.com/rcrowley/go-metrics v0.0.0-20200313005456-10cdbea86bc0
	github.com/spf13/cast v1.4.1
	github.com/stretchr/testify v1.7.1
	github.com/valyala/fasthttp v1.28.0
	golang.org/x/net v0.0.0-20220421235706-1d1ef9303861
	google.golang.org/grpc v1.46.0
	google.golang.org/protobuf v1.28.0
)

replace google.golang.org/grpc v1.46.0 => google.golang.org/grpc v1.38.0
