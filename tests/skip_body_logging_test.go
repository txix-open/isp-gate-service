package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"isp-gate-service/assembly"
	"isp-gate-service/conf"
	"isp-gate-service/entity"
	"isp-gate-service/routes"

	"github.com/stretchr/testify/suite"
	"github.com/txix-open/isp-kit/cluster"
	"github.com/txix-open/isp-kit/grpc/client"
	"github.com/txix-open/isp-kit/http/httpcli"
	"github.com/txix-open/isp-kit/lb"
	"github.com/txix-open/isp-kit/log"
	"github.com/txix-open/isp-kit/test"
	"github.com/txix-open/isp-kit/test/grpct"
	"github.com/txix-open/isp-kit/test/httpt"
)

type SkipBodyLoggingTestSuite struct {
	suite.Suite
}

// Regression: skipBodyLoggingEndpointPrefixes с ведущим '/' должны
// работать и в non-skipAuth цепочках — лог-строка пишется, тела нет.
func (s *SkipBodyLoggingTestSuite) Test_KnownEndpoint_BodySkip_StrippedPrefix() {
	test, require := test.New(s.T())
	capture := &logCapture{}

	config, systemCli, adminCli := s.authDependencies(test)

	targetService := httpt.NewMock(test)
	targetService.POST("/endpoint", func(ctx context.Context, httpReq *http.Request, req request) response {
		return response{Id: req.Id}
	})
	targetUrl, err := url.Parse(targetService.BaseURL())
	require.NoError(err)
	targetHosts := map[string]*lb.RoundRobin{"target": lb.NewRoundRobin([]string{targetUrl.Host})}

	routes := routes.NewRoutes(test.Logger())
	err = routes.ReceiveRoutes(s.T().Context(), cluster.RoutingConfig{{
		Endpoints: []cluster.EndpointDescriptor{{Path: "/endpoint"}},
	}})
	require.NoError(err)

	locator := assembly.NewLocator(capture, nil, targetHosts, routes, systemCli, adminCli, nil, nil, nil)
	locations := []conf.Location{{
		SkipAuth:     false,
		PathPrefix:   "/api",
		Protocol:     "http",
		TargetModule: "target",
	}}
	handler, err := locator.Handler(config, locations)
	require.NoError(err)

	srv := httptest.NewServer(handler)
	defer srv.Close()

	resp, err := httpcli.New().Post(srv.URL+"/api/endpoint").
		Header("x-application-token", "token").
		Header("x-auth-admin", "mock-token").
		JsonRequestBody(request{Id: "test"}).
		Do(s.T().Context())
	require.NoError(err)
	require.EqualValues(http.StatusOK, resp.StatusCode())

	// ровно одна лог-строка на запрос — без тел
	require.Len(capture.entries, 1)
	entries := capture.entriesWithMessage("log request")
	require.Len(entries, 1)
	require.NotContains(entries[0].fields, "request")
	require.NotContains(entries[0].fields, "response")
}

// nolint:ireturn
func (s *SkipBodyLoggingTestSuite) authDependencies(test *test.Test) (conf.Remote, *client.Client, *client.Client) {
	config := conf.Remote{
		Http: conf.Http{MaxRequestBodySizeInMb: 1, ProxyTimeoutInSec: 15},
		Logging: conf.Logging{
			LogLevel:                        log.DebugLevel,
			RequestLogEnable:                true,
			BodyLogEnable:                   true,
			SkipBodyLoggingEndpointPrefixes: []string{"/endpoint"},
		},
		Caching: conf.Caching{AuthorizationDataInSec: 1, AuthenticationDataInSec: 1},
	}

	systemService, systemCli := grpct.NewMock(test)
	systemService.Mock("system/secure/authenticate", func() entity.AuthenticateResponse {
		return entity.AuthenticateResponse{
			Authenticated: true,
			ErrorReason:   "",
			AuthData: &entity.AppAuthData{
				SystemId:      1,
				DomainId:      2,
				ServiceId:     3,
				ApplicationId: 4,
				AppName:       "test",
			},
		}
	}).Mock("system/secure/authorize", func() entity.AuthorizeResponse {
		return entity.AuthorizeResponse{Authorized: true}
	})

	adminService, adminCli := grpct.NewMock(test)
	adminService.Mock("admin/secure/authenticate", func(req entity.AdminAuthorizeRequest) entity.AdminAuthenticateResponse {
		return entity.AdminAuthenticateResponse{
			Authenticated: true,
			ErrorReason:   "",
			AdminId:       1,
		}
	}).Mock("admin/secure/authorize", func(req entity.AdminAuthorizeRequest) entity.AdminAuthorizeResponse {
		return entity.AdminAuthorizeResponse{Authorized: true}
	})

	return config, systemCli, adminCli
}

func TestSkipBodyLoggingTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SkipBodyLoggingTestSuite))
}
