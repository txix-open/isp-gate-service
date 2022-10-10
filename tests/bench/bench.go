package main

import (
	"context"
	"net"
	"net/http"

	"isp-gate-service/assembly"
	"isp-gate-service/conf"
	"isp-gate-service/domain"
	"isp-gate-service/routes"

	"github.com/go-redis/redis/v8"
	"github.com/integration-system/isp-kit/grpc"
	"github.com/integration-system/isp-kit/grpc/client"
	endpoint2 "github.com/integration-system/isp-kit/grpc/endpoint"
	"github.com/integration-system/isp-kit/grpc/isp"
	"github.com/integration-system/isp-kit/log"
)

type request struct {
	Id string
}

type response struct {
	Id string
}

//nolint
func main() {
	redisCli := redis.NewClient(&redis.Options{})
	ctx := context.Background()
	logger, err := log.New()
	if err != nil {
		panic(err)
	}
	defer func() {
		_, _ = redisCli.Pipelined(ctx, func(p redis.Pipeliner) error {
			p.Select(ctx, 0)
			p.FlushDB(ctx)
			p.Select(ctx, 1)
			p.FlushDB(ctx)
			p.Select(ctx, 2)
			p.FlushDB(ctx)
			return nil
		})
	}()
	config := conf.Remote{
		Redis:       &conf.Redis{Address: "localhost:6379"},
		Http:        conf.Http{MaxRequestBodySizeInMb: 1, ProxyTimeoutInSec: 15},
		Logging:     conf.Logging{LogLevel: log.DebugLevel, RequestLogEnable: true, BodyLogEnable: true},
		Caching:     conf.Caching{AuthorizationDataInSec: 15, AuthenticationDataInSec: 15},
		DailyLimits: []conf.DailyLimit{{ApplicationId: 1, RequestsPerDay: 100000000}},
		Throttling:  []conf.Throttling{{ApplicationId: 1, RequestsPerSeconds: 5000}},
	}

	systemService, systemCli, _ := NewMock(logger)
	systemService.Mock("system/secure/authenticate", func() domain.AuthenticateResponse {
		return domain.AuthenticateResponse{
			Authenticated: true,
			ErrorReason:   "",
			AuthData: &domain.AuthData{
				SystemId:      1,
				DomainId:      1,
				ServiceId:     1,
				ApplicationId: 1,
			},
		}
	}).Mock("system/secure/authorize", func() domain.AuthorizeResponse {
		return domain.AuthorizeResponse{Authorized: true}
	})

	adminService, _, adminCli := NewMock(logger)
	adminService.Mock("admin/secure/authenticate", func() domain.AdminAuthenticateResponse {
		return domain.AdminAuthenticateResponse{
			Authenticated: true,
			ErrorReason:   "",
			AdminId:       1,
		}
	})

	targetService, targetCli, _ := NewMock(logger)
	targetService.Mock("endpoint", func(req request) response {
		return response{Id: req.Id}
	})
	targetClients := map[string]*client.Client{"target": targetCli}
	logger, _ = log.New(log.WithLevel(log.DebugLevel))
	locator := assembly.NewLocator(logger, targetClients, nil, routes.NewRoutes(), systemCli, adminCli)
	locations := []conf.Location{{
		SkipAuth:     false,
		PathPrefix:   "/api",
		Protocol:     "grpc",
		TargetModule: "target",
	}}
	handler, _ := locator.Handler(config, locations, redisCli)

	srv := &http.Server{
		Addr:    "localhost:8000",
		Handler: handler,
	}
	srv.ListenAndServe()
}

func TestServer(service isp.BackendServiceServer) (*grpc.Server, *client.Client, *client.Client) {
	listener, _ := net.Listen("tcp", "127.0.0.1:")
	srv := grpc.NewServer()
	sysCli, _ := client.Default()
	adminCli, _ := client.Default()
	srv.Upgrade(service)
	go func() {
		_ = srv.Serve(listener)
	}()

	sysCli.Upgrade([]string{listener.Addr().String()})
	adminCli.Upgrade([]string{listener.Addr().String()})
	return srv, sysCli, adminCli
}

type MockServer struct {
	srv           *grpc.Server
	logger        log.Logger
	mockEndpoints map[string]interface{}
}

func NewMock(logger log.Logger) (*MockServer, *client.Client, *client.Client) {
	srv, syscli, admincli := TestServer(grpc.NewMux())
	return &MockServer{
		srv:           srv,
		logger:        logger,
		mockEndpoints: make(map[string]interface{}),
	}, syscli, admincli
}

func (m *MockServer) Mock(endpoint string, handler interface{}) *MockServer {
	m.mockEndpoints[endpoint] = handler
	wrapper := endpoint2.DefaultWrapper(m.logger)
	muxer := grpc.NewMux()
	for e, handler := range m.mockEndpoints {
		muxer.Handle(e, wrapper.Endpoint(handler))
	}
	m.srv.Upgrade(muxer)
	return m
}
