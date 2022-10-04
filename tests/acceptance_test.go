package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"isp-gate-service/assembly"
	"isp-gate-service/conf"
	"isp-gate-service/domain"
	"isp-gate-service/routes"

	"github.com/go-redis/redis/v8"
	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"github.com/integration-system/isp-kit/grpc"
	"github.com/integration-system/isp-kit/grpc/client"
	"github.com/integration-system/isp-kit/lb"
	"github.com/integration-system/isp-kit/log"
	"github.com/integration-system/isp-kit/requestid"
	"github.com/integration-system/isp-kit/test"
	"github.com/integration-system/isp-kit/test/grpct"
	"github.com/integration-system/isp-kit/test/httpt"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/metadata"
)

type request struct {
	Id string
}

type response struct {
	Id string
}

type HappyPathTestSuite struct {
	suite.Suite
}

func (s *HappyPathTestSuite) TestGrpcProxy() {
	test, require := test.New(s.T())
	config, redisCli, systemCli, adminCli := s.commonDependencies(test)

	requestId := requestid.Next()
	targetService, targetCli := grpct.NewMock(test)
	targetService.Mock("endpoint", func(ctx context.Context, authData grpc.AuthData, req request) response {
		systemId, err := authData.SystemId()
		require.NoError(err)
		domainId, err := authData.DomainId()
		require.NoError(err)
		serviceId, err := authData.ServiceId()
		require.NoError(err)
		applicationId, err := authData.ApplicationId()
		require.NoError(err)
		require.EqualValues(requestId, requestid.FromContext(ctx))
		require.EqualValues(1, systemId)
		require.EqualValues(2, domainId)
		require.EqualValues(3, serviceId)
		require.EqualValues(4, applicationId)

		adminId, err := grpc.StringFromMd("x-admin-id", metadata.MD(authData))
		require.NoError(err)
		require.EqualValues("1", adminId)

		return response{Id: req.Id} //nolint:gosimple
	})
	targetClients := map[string]*client.Client{"target": targetCli}
	logger, err := log.New(log.WithLevel(log.DebugLevel))
	require.NoError(err)
	locator := assembly.NewLocator(logger, targetClients, nil, routes.NewRoutes(), systemCli, adminCli)

	locations := []conf.Location{{
		SkipAuth:     false,
		PathPrefix:   "/api",
		Protocol:     "grpc",
		TargetModule: "target",
	}}
	handler, err := locator.Handler(config, locations, redisCli)
	require.NoError(err)

	srv := httptest.NewServer(handler)
	cli := resty.NewWithClient(srv.Client()).SetBaseURL(srv.URL)
	req := request{Id: uuid.New().String()}
	resp := response{}
	_, err = cli.R().
		SetHeader("x-application-token", "token").
		SetHeader("x-request-id", requestId).
		SetHeader("x-auth-admin", "mock-token").
		SetBody(req).
		SetResult(&resp).
		Post("/api/endpoint")
	require.NoError(err)
	require.EqualValues(req.Id, resp.Id)
}

func (s *HappyPathTestSuite) TestHttpProxy() {
	test, require := test.New(s.T())
	config, redisCli, systemCli, adminCli := s.commonDependencies(test)

	requestId := requestid.Next()
	targetService := httpt.NewMock(test)
	targetService.POST("/endpoint", func(ctx context.Context, httpReq *http.Request, req request) response {
		systemId, err := strconv.Atoi(httpReq.Header.Get("x-system-identity"))
		require.NoError(err)
		domainId, err := strconv.Atoi(httpReq.Header.Get("x-domain-identity"))
		require.NoError(err)
		serviceId, err := strconv.Atoi(httpReq.Header.Get("x-service-identity"))
		require.NoError(err)
		applicationId, err := strconv.Atoi(httpReq.Header.Get("x-application-identity"))
		require.NoError(err)
		require.EqualValues(requestId, requestid.FromContext(ctx))
		require.EqualValues(1, systemId)
		require.EqualValues(2, domainId)
		require.EqualValues(3, serviceId)
		require.EqualValues(4, applicationId)

		adminId, err := strconv.Atoi(httpReq.Header.Get("x-admin-id"))
		require.NoError(err)
		require.EqualValues(1, adminId)

		return response{Id: req.Id} //nolint:gosimple
	})
	targetUrl, err := url.Parse(targetService.BaseURL())
	require.NoError(err)
	rr := lb.NewRoundRobin([]string{targetUrl.Host})
	targetClients := map[string]*lb.RoundRobin{"target": rr}
	locator := assembly.NewLocator(test.Logger(), nil, targetClients, routes.NewRoutes(), systemCli, adminCli)
	locations := []conf.Location{{
		SkipAuth:     false,
		PathPrefix:   "/api",
		Protocol:     "http",
		TargetModule: "target",
	}}
	handler, err := locator.Handler(config, locations, redisCli)
	require.NoError(err)

	srv := httptest.NewServer(handler)
	cli := resty.NewWithClient(srv.Client()).SetBaseURL(srv.URL)
	req := request{Id: uuid.New().String()}
	resp := response{}
	_, err = cli.R().
		SetHeader("x-application-token", "token").
		SetHeader("x-request-id", requestId).
		SetHeader("x-auth-admin", "mock-token").
		SetBody(req).
		SetResult(&resp).
		Post("/api/endpoint")
	require.NoError(err)
	require.EqualValues(req.Id, resp.Id)
}

func (s *HappyPathTestSuite) commonDependencies(test *test.Test) (conf.Remote, redis.UniversalClient, *client.Client, *client.Client) {
	require := test.Assert()
	redisCli := NewRedis(test)
	ctx := context.Background()

	s.T().Cleanup(func() {
		_, err := redisCli.Pipelined(ctx, func(p redis.Pipeliner) error {
			p.Select(ctx, 0)
			p.FlushDB(ctx)
			p.Select(ctx, 1)
			p.FlushDB(ctx)
			p.Select(ctx, 2)
			p.FlushDB(ctx)
			return nil
		})
		require.NoError(err)
	})

	config := conf.Remote{
		Redis:       &conf.Redis{Address: redisCli.address},
		Http:        conf.Http{MaxRequestBodySizeInMb: 1, ProxyTimeoutInSec: 15},
		Logging:     conf.Logging{LogLevel: log.DebugLevel, RequestLogEnable: true, BodyLogEnable: true},
		Caching:     conf.Caching{AuthorizationDataInSec: 1, AuthenticationDataInSec: 1},
		DailyLimits: []conf.DailyLimit{{ApplicationId: 1, RequestsPerDay: 100}},
		Throttling:  []conf.Throttling{{ApplicationId: 1, RequestsPerSeconds: 100}},
	}

	systemService, systemCli := grpct.NewMock(test)
	systemService.Mock("system/secure/authenticate", func() domain.AuthenticateResponse {
		return domain.AuthenticateResponse{
			Authenticated: true,
			ErrorReason:   "",
			AuthData: &domain.AuthData{
				SystemId:      1,
				DomainId:      2,
				ServiceId:     3,
				ApplicationId: 4,
			},
		}
	}).Mock("system/secure/authorize", func() domain.AuthorizeResponse {
		return domain.AuthorizeResponse{Authorized: true}
	})

	adminService, adminCli := grpct.NewMock(test)
	adminService.Mock("admin/secure/authenticate", func() domain.AdminAuthenticateResponse {
		return domain.AdminAuthenticateResponse{
			Authenticated: true,
			ErrorReason:   "",
			AdminId:       1,
		}
	})

	return config, redisCli, systemCli, adminCli
}

func TestHappyPathTestSuite(t *testing.T) {
	suite.Run(t, new(HappyPathTestSuite))
}
