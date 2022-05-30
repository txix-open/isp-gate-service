package conf

import (
	"github.com/integration-system/isp-kit/bootstrap"
)

const (
	HttpProtocol      = "http"
	GrpcProtocol      = "grpc"
	WebsocketProtocol = "websocket"
)

type Local struct {
	*bootstrap.LocalConfig
	Locations []Location
}

type Location struct {
	SkipAuth       bool
	SkipExistCheck bool
	PathPrefix     string `valid:"required"`
	Protocol       string `valid:"required,in(http|grpc|websocket)"`
	TargetModule   string `valid:"required"`
}
