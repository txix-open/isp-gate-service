// nolint:canonicalheader
package tests

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"isp-gate-service/assembly"
	"isp-gate-service/cache"
	"isp-gate-service/conf"
	"isp-gate-service/entity"
	"isp-gate-service/routes"

	"github.com/stretchr/testify/require"
	"github.com/txix-open/etp/v3"
	"github.com/txix-open/etp/v3/msg"
	"github.com/txix-open/isp-kit/cluster"
	"github.com/txix-open/isp-kit/http/endpoint"
	"github.com/txix-open/isp-kit/http/endpoint/httplog"
	"github.com/txix-open/isp-kit/http/httpcli"
	"github.com/txix-open/isp-kit/http/router"
	"github.com/txix-open/isp-kit/test/fake"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"github.com/txix-open/isp-kit/grpc"
	"github.com/txix-open/isp-kit/grpc/client"
	"github.com/txix-open/isp-kit/lb"
	"github.com/txix-open/isp-kit/log"
	"github.com/txix-open/isp-kit/requestid"
	"github.com/txix-open/isp-kit/test"
	"github.com/txix-open/isp-kit/test/grpct"
	"github.com/txix-open/isp-kit/test/httpt"
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
	config, systemCli, adminCli := s.commonDependencies(test)

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

		return response{Id: req.Id}
	})
	targetClients := map[string]*client.Client{"target": targetCli}
	logger, err := log.New(log.WithLevel(log.DebugLevel))
	require.NoError(err)

	routes := routes.NewRoutes(logger)
	err = routes.ReceiveRoutes(s.T().Context(), cluster.RoutingConfig{{
		Endpoints: []cluster.EndpointDescriptor{{
			Path: "endpoint",
		}},
	}})
	require.NoError(err)

	locator := assembly.NewLocator(logger, targetClients, nil, routes, systemCli, adminCli, nil, nil, nil)

	locations := []conf.Location{{
		SkipAuth:     false,
		PathPrefix:   "/api",
		Protocol:     "grpc",
		TargetModule: "target",
	}}
	handler, err := locator.Handler(config, locations)
	require.NoError(err)

	srv := httptest.NewServer(handler)
	cli := httpcli.New()
	req := request{Id: uuid.New().String()}
	resp := response{}
	err = cli.Post(srv.URL+"/api/endpoint").
		Header("x-application-token", "token").
		Header("x-request-id", requestId).
		Header("x-auth-admin", "mock-token").
		JsonRequestBody(req).
		JsonResponseBody(&resp).
		StatusCodeToError().
		DoWithoutResponse(s.T().Context())
	require.NoError(err)
	require.EqualValues(req.Id, resp.Id)
}

func (s *HappyPathTestSuite) TestHttpProxy() {
	test, require := test.New(s.T())
	config, systemCli, adminCli := s.commonDependencies(test)

	requestId := requestid.Next()
	targetService := httpt.NewMock(test)
	targetService.POST("/endpoint", func(ctx context.Context, httpReq *http.Request, req request) response {
		assertHeaders(require, requestId, ctx, httpReq.Header)
		return response{Id: req.Id}
	})
	targetUrl, err := url.Parse(targetService.BaseURL())
	require.NoError(err)
	rr := lb.NewRoundRobin([]string{targetUrl.Host})
	targetClients := map[string]*lb.RoundRobin{"target": rr}

	routes := routes.NewRoutes(test.Logger())
	err = routes.ReceiveRoutes(s.T().Context(), cluster.RoutingConfig{{
		Endpoints: []cluster.EndpointDescriptor{{
			Path: "/endpoint",
		}},
	}})
	require.NoError(err)

	locator := assembly.NewLocator(test.Logger(), nil, targetClients, routes, systemCli, adminCli, nil, nil, nil)
	locations := []conf.Location{{
		SkipAuth:     false,
		PathPrefix:   "/api",
		Protocol:     "http",
		TargetModule: "target",
	}}
	handler, err := locator.Handler(config, locations)
	require.NoError(err)

	srv := httptest.NewServer(handler)
	cli := httpcli.New()
	req := request{Id: uuid.New().String()}
	resp := response{}
	err = cli.Post(srv.URL+"/api/endpoint").
		Header("x-application-token", "token").
		Header("x-request-id", requestId).
		Header("x-auth-admin", "mock-token").
		JsonRequestBody(req).
		JsonResponseBody(&resp).
		StatusCodeToError().
		DoWithoutResponse(s.T().Context())
	require.NoError(err)
	require.EqualValues(req.Id, resp.Id)
}

func (s *HappyPathTestSuite) TestWsProxy() { // nolint:funlen
	test, require := test.New(s.T())
	config, systemCli, adminCli := s.commonDependencies(test)

	requestId := requestid.Next()
	wsServer := etp.NewServer(etp.WithServerReadLimit(2048))
	wsServer.On("hello", wsEventHandlerMock{
		requestId: requestId,
		require:   require,
		t:         test.T(),
	})

	wsMux := http.NewServeMux()
	wsMux.Handle("/service", http.HandlerFunc(func(writer http.ResponseWriter, r *http.Request) {
		wsServer.ServeHTTP(writer, r)
	}))
	targetService := httptest.NewServer(wsMux)
	targetUrl, err := url.Parse(targetService.URL)
	require.NoError(err)
	rr := lb.NewRoundRobin([]string{targetUrl.Host})
	targetClients := map[string]*lb.RoundRobin{"target": rr}

	routes := routes.NewRoutes(test.Logger())
	err = routes.ReceiveRoutes(s.T().Context(), cluster.RoutingConfig{{
		Endpoints: []cluster.EndpointDescriptor{{
			Path: "/service",
		}},
	}})
	require.NoError(err)

	locator := assembly.NewLocator(test.Logger(), nil, targetClients, routes, systemCli, adminCli, nil, nil, nil)
	locations := []conf.Location{{
		SkipAuth:     false,
		PathPrefix:   "/ws",
		Protocol:     "ws",
		TargetModule: "target",
	}}
	handler, err := locator.Handler(config, locations)
	require.NoError(err)
	srv := httptest.NewServer(handler)

	cli := etp.NewClient(etp.WithClientDialOptions(&etp.DialOptions{
		HTTPHeader: map[string][]string{
			"x-request-id": {requestId},
		},
	}))
	requestUrl, err := url.Parse(srv.URL)
	require.NoError(err)
	requestUrl.Path = "ws/service"
	requestUrl.RawQuery = url.Values{
		"x-application-token": []string{"token"},
		"x-auth-admin":        []string{"mock-token"},
	}.Encode()
	err = cli.Dial(context.Background(), requestUrl.String())
	require.NoError(err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	resp, err := cli.EmitWithAck(ctx, "hello", []byte("data"))
	require.NoError(err)
	require.EqualValues("world", string(resp))
	err = cli.Close()
	require.NoError(err)
}

type wsEventHandlerMock struct {
	requestId string
	require   *require.Assertions
	t         require.TestingT
}

func (s wsEventHandlerMock) Handle(_ context.Context, conn *etp.Conn, event msg.Event) []byte {
	headers := conn.HttpRequest().Header
	ctx := requestid.ToContext(context.Background(), headers.Get("x-request-id"))
	assertHeaders(s.require, s.requestId, ctx, headers)
	require.EqualValues(s.t, "data", event.Data)
	return []byte("world")
}

func (s *HappyPathTestSuite) TestAdminAuthorization() {
	test, require := test.New(s.T())
	config, systemCli, adminCli := s.commonDependencies(test)

	targetService, targetCli := grpct.NewMock(test)
	targetService.Mock("endpoint", func(ctx context.Context, authData grpc.AuthData, req request) response {
		return response{Id: req.Id}
	})
	targetClients := map[string]*client.Client{"target": targetCli}
	logger, err := log.New(log.WithLevel(log.DebugLevel))
	require.NoError(err)
	routes := routes.NewRoutes(logger)
	locator := assembly.NewLocator(logger, targetClients, nil, routes, systemCli, adminCli, nil, nil, nil)

	locations := []conf.Location{{
		SkipAuth:     false,
		PathPrefix:   "/api",
		Protocol:     "grpc",
		TargetModule: "target",
	}}
	handler, err := locator.Handler(config, locations)
	require.NoError(err)

	err = routes.ReceiveRoutes(context.Background(), cluster.RoutingConfig{{
		ModuleName: "target",
		Endpoints: []cluster.EndpointDescriptor{{
			Path:  "endpoint",
			Inner: true,
			Extra: cluster.RequireAdminPermission("ok_permission"),
		}, {
			Path:  "failed_endpoint",
			Inner: true,
			Extra: cluster.RequireAdminPermission("failed_permission"),
		}},
	}})
	require.NoError(err)

	srv := httptest.NewServer(handler)
	cli := httpcli.New()
	req := request{Id: uuid.New().String()}
	resp := response{}
	err = cli.Post(srv.URL+"/api/endpoint").
		Header("x-application-token", "token").
		Header("x-auth-admin", "mock-token").
		JsonRequestBody(req).
		JsonResponseBody(&resp).
		StatusCodeToError().
		DoWithoutResponse(s.T().Context())
	require.NoError(err)
	require.EqualValues(req.Id, resp.Id)

	err = cli.Post(srv.URL+"/api/failed_endpoint").
		Header("x-application-token", "token").
		Header("x-auth-admin", "mock-token").
		JsonRequestBody(req).
		JsonResponseBody(&resp).
		StatusCodeToError().
		DoWithoutResponse(s.T().Context())
	errResp := httpcli.ErrorResponse{}
	require.ErrorAs(err, &errResp)
	require.EqualValues(http.StatusForbidden, errResp.StatusCode)
}

func (s *HappyPathTestSuite) TestUserAuthorization() { // nolint:funlen
	test, require := test.New(s.T())
	config, systemCli, adminCli := s.commonDependencies(test)
	userAuthData := fake.It[entity.UserAuthData]()
	userAuthData.IdentityHeader = "x-" + userAuthData.IdentityHeader

	targetService, targetCli := grpct.NewMock(test)
	targetService.Mock("endpoint", func(ctx context.Context, authData grpc.AuthData, req request) response {
		md, ok := metadata.FromIncomingContext(ctx)
		require.True(ok)
		require.ElementsMatch([]string{userAuthData.Identity}, md.Get(userAuthData.IdentityHeader))

		return response{Id: req.Id}
	})
	targetService.Mock("failed_auth_endpoint", func(ctx context.Context, authData grpc.AuthData, req request) response {
		return response{Id: req.Id}
	})

	routerMux := router.New()
	defaultWrapper := endpoint.DefaultWrapper(
		test.Logger(),
		httplog.Noop(),
	)

	userAuthToken := fake.It[string]()

	routerMux.POST("/test-user-auth/authenticate",
		defaultWrapper.Endpoint(func(req entity.UserAuthenticateRequest) entity.UserAuthenticateResponse {
			require.Equal(userAuthToken, req.Token)
			return entity.UserAuthenticateResponse{
				Authenticated: true,
				AuthData:      &userAuthData,
			}
		}))
	routerMux.POST("/failed-user-auth/authenticate",
		defaultWrapper.Endpoint(func(req entity.UserAuthenticateRequest) entity.UserAuthenticateResponse {
			require.Equal(userAuthToken, req.Token)
			return entity.UserAuthenticateResponse{
				Authenticated: false,
				ErrorReason:   "test fail",
			}
		}))
	routerMock := httptest.NewServer(routerMux)
	targetUrl, err := url.Parse(routerMock.URL)
	require.NoError(err)
	rr := lb.NewRoundRobin([]string{targetUrl.Host})

	targetClients := map[string]*client.Client{"target": targetCli}
	logger, err := log.New(log.WithLevel(log.DebugLevel))
	require.NoError(err)
	routes := routes.NewRoutes(logger)
	locator := assembly.NewLocator(logger, targetClients, nil, routes, systemCli, adminCli, nil, rr, cache.New())

	locations := []conf.Location{{
		SkipAuth:     false,
		PathPrefix:   "/api",
		Protocol:     "grpc",
		TargetModule: "target",
	}}
	config.CustomAuth = conf.CustomAuth{
		TokenProviders: []conf.TokenProvider{
			{
				Name: "test_provider",
				Type: conf.HeaderTokenProviderType,
				HeaderProvider: &conf.HeaderTokenProvider{
					HeaderName: "test-token-header",
				},
			},
		},
		UserAuthSettings: []conf.UserAuthSetting{
			{
				ModuleNameList:         []string{"target"},
				TokenProvider:          "test_provider",
				AuthenticateMethodPath: "test-user-auth/authenticate",
			},
			{
				ModuleNameList:         []string{"target2"},
				TokenProvider:          "test_provider",
				AuthenticateMethodPath: "failed-user-auth/authenticate",
			},
		},
	}
	handler, err := locator.Handler(config, locations)
	require.NoError(err)

	err = routes.ReceiveRoutes(context.Background(), cluster.RoutingConfig{
		{
			ModuleName: "target",
			Endpoints: []cluster.EndpointDescriptor{{
				Path:             "endpoint",
				Inner:            true,
				UserAuthRequired: true,
			},
			},
		},
		{
			ModuleName: "target2",
			Endpoints: []cluster.EndpointDescriptor{
				{
					Path:             "failed_auth_endpoint",
					Inner:            true,
					UserAuthRequired: true,
				}, {
					Path:             "failed_endpoint",
					Inner:            true,
					UserAuthRequired: true,
				},
			},
		},
	})
	require.NoError(err)

	srv := httptest.NewServer(handler)
	cli := httpcli.New()
	req := request{Id: uuid.New().String()}
	resp := response{}
	err = cli.Post(srv.URL+"/api/endpoint").
		Header("x-application-token", "token").
		Header("test-token-header", userAuthToken).
		Header("x-auth-admin", "mock-token").
		JsonRequestBody(req).
		JsonResponseBody(&resp).
		StatusCodeToError().
		DoWithoutResponse(s.T().Context())
	require.NoError(err)
	require.EqualValues(req.Id, resp.Id)

	err = cli.Post(srv.URL+"/api/failed_auth_endpoint").
		Header("x-application-token", "token").
		Header("test-token-header", userAuthToken).
		Header("x-auth-admin", "mock-token").
		JsonRequestBody(req).
		JsonResponseBody(&resp).
		StatusCodeToError().
		DoWithoutResponse(s.T().Context())
	errResp := httpcli.ErrorResponse{}
	require.ErrorAs(err, &errResp)
	require.EqualValues(http.StatusUnauthorized, errResp.StatusCode)
}

func (s *HappyPathTestSuite) TestUserAuthorization_SkipAppAuth() { // nolint:funlen
	test, require := test.New(s.T())
	config, _, adminCli := s.commonDependencies(test)
	systemService, systemCli := grpct.NewMock(test)
	systemService.Mock("system/secure/authenticate", func() entity.AuthenticateResponse {
		return entity.AuthenticateResponse{
			Authenticated: false,
			ErrorReason:   "",
		}
	}).Mock("system/secure/authorize", func() entity.AuthorizeResponse {
		return entity.AuthorizeResponse{Authorized: false}
	})

	userAuthData := fake.It[entity.UserAuthData]()
	userAuthData.IdentityHeader = "x-" + userAuthData.IdentityHeader

	targetService, targetCli := grpct.NewMock(test)
	targetService.Mock("endpoint", func(ctx context.Context, authData grpc.AuthData, req request) response {
		md, ok := metadata.FromIncomingContext(ctx)
		require.True(ok)
		require.ElementsMatch([]string{userAuthData.Identity}, md.Get(userAuthData.IdentityHeader))

		return response{Id: req.Id}
	})

	routerMux := router.New()
	defaultWrapper := endpoint.DefaultWrapper(
		test.Logger(),
		httplog.Noop(),
	)

	userAuthToken := fake.It[string]()

	routerMux.POST("/test-user-auth/authenticate",
		defaultWrapper.Endpoint(func(req entity.UserAuthenticateRequest) entity.UserAuthenticateResponse {
			require.Equal(userAuthToken, req.Token)
			return entity.UserAuthenticateResponse{
				Authenticated: true,
				AuthData:      &userAuthData,
			}
		}))
	routerMock := httptest.NewServer(routerMux)
	targetUrl, err := url.Parse(routerMock.URL)
	require.NoError(err)
	rr := lb.NewRoundRobin([]string{targetUrl.Host})

	targetClients := map[string]*client.Client{"target": targetCli}
	logger, err := log.New(log.WithLevel(log.DebugLevel))
	require.NoError(err)
	routes := routes.NewRoutes(logger)
	locator := assembly.NewLocator(logger, targetClients, nil, routes, systemCli, adminCli, nil, rr, cache.New())

	locations := []conf.Location{{
		SkipAuth:     false,
		PathPrefix:   "/api",
		Protocol:     "grpc",
		TargetModule: "target",
	}}
	config.CustomAuth = conf.CustomAuth{
		TokenProviders: []conf.TokenProvider{
			{
				Name: "test_provider",
				Type: conf.HeaderTokenProviderType,
				HeaderProvider: &conf.HeaderTokenProvider{
					HeaderName: "test-token-header",
				},
			},
		},
		UserAuthSettings: []conf.UserAuthSetting{
			{
				ModuleNameList:         []string{"target"},
				TokenProvider:          "test_provider",
				AuthenticateMethodPath: "test-user-auth/authenticate",
				SkipAppAuth:            true,
			},
			{
				ModuleNameList:         []string{"target2"},
				TokenProvider:          "test_provider",
				AuthenticateMethodPath: "test-user-auth/authenticate",
			},
		},
	}
	handler, err := locator.Handler(config, locations)
	require.NoError(err)

	err = routes.ReceiveRoutes(context.Background(), cluster.RoutingConfig{{
		ModuleName: "target",
		Endpoints: []cluster.EndpointDescriptor{{
			Path:             "endpoint",
			Inner:            true,
			UserAuthRequired: true,
		}}},
		{
			ModuleName: "target2",
			Endpoints: []cluster.EndpointDescriptor{{
				Path:             "failed_endpoint",
				Inner:            true,
				UserAuthRequired: true,
			}},
		},
	})
	require.NoError(err)

	srv := httptest.NewServer(handler)
	cli := httpcli.New()
	req := request{Id: uuid.New().String()}
	resp := response{}
	err = cli.Post(srv.URL+"/api/endpoint").
		Header("x-application-token", "token").
		Header("test-token-header", userAuthToken).
		Header("x-auth-admin", "mock-token").
		JsonRequestBody(req).
		JsonResponseBody(&resp).
		StatusCodeToError().
		DoWithoutResponse(s.T().Context())
	require.NoError(err)
	require.EqualValues(req.Id, resp.Id)

	err = cli.Post(srv.URL+"/api/failed_endpoint").
		Header("x-application-token", "token").
		Header("test-token-header", userAuthToken).
		Header("x-auth-admin", "mock-token").
		JsonRequestBody(req).
		JsonResponseBody(&resp).
		StatusCodeToError().
		DoWithoutResponse(s.T().Context())
	errResp := httpcli.ErrorResponse{}
	require.ErrorAs(err, &errResp)
	require.EqualValues(http.StatusUnauthorized, errResp.StatusCode)
}

// nolint:ireturn
func (s *HappyPathTestSuite) commonDependencies(test *test.Test) (conf.Remote, *client.Client, *client.Client) {
	config := conf.Remote{
		Http: conf.Http{MaxRequestBodySizeInMb: 1, ProxyTimeoutInSec: 15},
		Logging: conf.Logging{LogLevel: log.DebugLevel, RequestLogEnable: true, BodyLogEnable: true,
			SkipBodyLoggingEndpointPrefixes: []string{"endpoint"}},
		Caching:                         conf.Caching{AuthorizationDataInSec: 1, AuthenticationDataInSec: 1},
		DailyLimits:                     []conf.DailyLimit{{ApplicationId: 1, RequestsPerDay: 100}},
		Throttling:                      []conf.Throttling{{ApplicationId: 1, RequestsPerSeconds: 100}},
		EnableClientRequestIdForwarding: true,
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
		if req.Permission == "ok_permission" {
			return entity.AdminAuthorizeResponse{Authorized: true}
		}
		return entity.AdminAuthorizeResponse{Authorized: false}
	})

	return config, systemCli, adminCli
}

func TestHappyPathTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(HappyPathTestSuite))
}

func assertHeaders(require *require.Assertions, requestId string, ctx context.Context, headers http.Header) {
	systemId, err := strconv.Atoi(headers.Get("x-system-identity"))
	require.NoError(err)
	domainId, err := strconv.Atoi(headers.Get("x-domain-identity"))
	require.NoError(err)
	serviceId, err := strconv.Atoi(headers.Get("x-service-identity"))
	require.NoError(err)
	applicationId, err := strconv.Atoi(headers.Get("x-application-identity"))
	require.NoError(err)
	require.EqualValues(requestId, requestid.FromContext(ctx))
	require.EqualValues(1, systemId)
	require.EqualValues(2, domainId)
	require.EqualValues(3, serviceId)
	require.EqualValues(4, applicationId)
	expectedAppName := base64.StdEncoding.EncodeToString([]byte("test"))
	require.EqualValues(expectedAppName, headers.Get("x-application-name"))

	adminId, err := strconv.Atoi(headers.Get("x-admin-id"))
	require.NoError(err)
	require.EqualValues(1, adminId)
}
