package proxy

import (
	"github.com/integration-system/isp-kit/lb"
	"isp-gate-service/middleware"
)

type Websocket struct {
	roundRobin *lb.RoundRobin
}

func NewWebsocket(hostManager *lb.RoundRobin) (Websocket, error) {
	return Websocket{
		roundRobin: hostManager,
	}, nil
}

func (p Websocket) Handle(ctx middleware.Context) error {
	return nil
}
