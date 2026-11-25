package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"isp-gate-service/assembly"
	"isp-gate-service/conf"
	"isp-gate-service/routes"

	"github.com/stretchr/testify/suite"
	"github.com/txix-open/isp-kit/cluster"
	"github.com/txix-open/isp-kit/http/httpcli"
	"github.com/txix-open/isp-kit/lb"
	"github.com/txix-open/isp-kit/test"
	"github.com/txix-open/isp-kit/test/httpt"
	"go.uber.org/zap/zapcore"
)

type UnknownLocationsLoggingTestSuite struct {
	suite.Suite
}

func (s *UnknownLocationsLoggingTestSuite) Test_UnknownPath_LogsWarn() {
	test, require := test.New(s.T())
	capture := &logCapture{}

	locator := assembly.NewLocator(capture, nil, nil, routes.NewRoutes(test.Logger()), nil, nil, nil, nil, nil)
	handler, err := locator.Handler(conf.Remote{Http: conf.Http{MaxRequestBodySizeInMb: 1}}, nil)
	require.NoError(err)

	srv := httptest.NewServer(handler)
	defer srv.Close()

	resp, err := httpcli.New().Get(srv.URL + "/unknown/path").Do(s.T().Context())
	require.NoError(err)
	require.EqualValues(http.StatusNotFound, resp.StatusCode())
	body, err := resp.BodyCopy()
	require.NoError(err)
	require.EqualValues("404 page not found\n", string(body))

	// ровно одна лог-строка на запрос, и это warn о незнакомой location
	require.Len(capture.entries, 1)
	entries := capture.entriesWithMessage("request to unknown location")
	require.Len(entries, 1)
	require.EqualValues(zapcore.WarnLevel, entries[0].level)
	require.EqualValues(http.MethodGet, entries[0].fields["httpMethod"])
	require.EqualValues("/unknown/path", entries[0].fields["path"])
	require.EqualValues(http.StatusNotFound, entries[0].fields["statusCode"])
	require.NotEmpty(entries[0].fields["remoteAddr"])
}

func (s *UnknownLocationsLoggingTestSuite) Test_UnknownPath_SuppressionSkipsWarn() {
	test, require := test.New(s.T())
	capture := &logCapture{}

	config := conf.Remote{
		Http:    conf.Http{MaxRequestBodySizeInMb: 1},
		Logging: conf.Logging{SkipLoggingPathPrefixes: []string{"/unknown"}},
	}
	locator := assembly.NewLocator(capture, nil, nil, routes.NewRoutes(test.Logger()), nil, nil, nil, nil, nil)
	handler, err := locator.Handler(config, nil)
	require.NoError(err)

	srv := httptest.NewServer(handler)
	defer srv.Close()

	resp, err := httpcli.New().Get(srv.URL + "/unknown/path").Do(s.T().Context())
	require.NoError(err)
	require.EqualValues(http.StatusNotFound, resp.StatusCode())

	// запрос полностью не логируется
	require.Empty(capture.entries)
}

func (s *UnknownLocationsLoggingTestSuite) Test_UnknownEndpoint_LogsWarn() {
	test, require := test.New(s.T())
	capture := &logCapture{}

	routes := routes.NewRoutes(test.Logger())
	err := routes.ReceiveRoutes(s.T().Context(), cluster.RoutingConfig{{
		Endpoints: []cluster.EndpointDescriptor{{Path: "endpoint"}},
	}})
	require.NoError(err)

	config := conf.Remote{Http: conf.Http{MaxRequestBodySizeInMb: 1}}
	locator := assembly.NewLocator(capture, nil, nil, routes, nil, nil, nil, nil, nil)
	locations := []conf.Location{{
		SkipAuth:     false,
		PathPrefix:   "/api",
		Protocol:     "grpc",
		TargetModule: "target",
	}}
	handler, err := locator.Handler(config, locations)
	require.NoError(err)

	srv := httptest.NewServer(handler)
	defer srv.Close()

	resp, err := httpcli.New().Post(srv.URL + "/api/nope").Do(s.T().Context())
	require.NoError(err)
	require.EqualValues(http.StatusNotImplemented, resp.StatusCode())

	// ровно одна лог-строка на запрос — warn о неизвестном эндпоинте
	require.Len(capture.entries, 1)
	entries := capture.entriesWithMessage("call unknown method")
	require.Len(entries, 1)
	require.EqualValues(zapcore.WarnLevel, entries[0].level)
}

func (s *UnknownLocationsLoggingTestSuite) Test_UnknownEndpoint_SuppressionSkipsWarn() {
	test, require := test.New(s.T())
	capture := &logCapture{}

	routes := routes.NewRoutes(test.Logger())
	err := routes.ReceiveRoutes(s.T().Context(), cluster.RoutingConfig{{
		Endpoints: []cluster.EndpointDescriptor{{Path: "endpoint"}},
	}})
	require.NoError(err)

	config := conf.Remote{
		Http:    conf.Http{MaxRequestBodySizeInMb: 1},
		Logging: conf.Logging{SkipLoggingPathPrefixes: []string{"/api/nope"}},
	}
	locator := assembly.NewLocator(capture, nil, nil, routes, nil, nil, nil, nil, nil)
	locations := []conf.Location{{
		SkipAuth:     false,
		PathPrefix:   "/api",
		Protocol:     "grpc",
		TargetModule: "target",
	}}
	handler, err := locator.Handler(config, locations)
	require.NoError(err)

	srv := httptest.NewServer(handler)
	defer srv.Close()

	resp, err := httpcli.New().Post(srv.URL + "/api/nope").Do(s.T().Context())
	require.NoError(err)
	require.EqualValues(http.StatusNotImplemented, resp.StatusCode())

	require.Empty(capture.entries)
}

func (s *UnknownLocationsLoggingTestSuite) Test_KnownEndpoint_LogsRequest() {
	test, require := test.New(s.T())
	capture := &logCapture{}

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

	config := conf.Remote{
		Http:    conf.Http{MaxRequestBodySizeInMb: 1, ProxyTimeoutInSec: 15},
		Logging: conf.Logging{RequestLogEnable: true},
	}
	locator := assembly.NewLocator(capture, nil, targetHosts, routes, nil, nil, nil, nil, nil)
	locations := []conf.Location{{
		SkipAuth:     true,
		PathPrefix:   "/api",
		Protocol:     "http",
		TargetModule: "target",
	}}
	handler, err := locator.Handler(config, locations)
	require.NoError(err)

	srv := httptest.NewServer(handler)
	defer srv.Close()

	resp, err := httpcli.New().Post(srv.URL + "/api/endpoint").
		JsonRequestBody(request{Id: "test"}).
		Do(s.T().Context())
	require.NoError(err)
	require.EqualValues(http.StatusOK, resp.StatusCode())

	// ровно одна лог-строка на запрос — log request
	require.Len(capture.entries, 1)
	entries := capture.entriesWithMessage("log request")
	require.Len(entries, 1)
	require.EqualValues("/api/endpoint", entries[0].fields["path"])
}

func (s *UnknownLocationsLoggingTestSuite) Test_KnownEndpoint_SuppressionSkipsRequestLog() {
	test, require := test.New(s.T())
	capture := &logCapture{}

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

	config := conf.Remote{
		Http: conf.Http{MaxRequestBodySizeInMb: 1, ProxyTimeoutInSec: 15},
		Logging: conf.Logging{
			RequestLogEnable:        true,
			SkipLoggingPathPrefixes: []string{"/api/endpoint"},
		},
	}
	locator := assembly.NewLocator(capture, nil, targetHosts, routes, nil, nil, nil, nil, nil)
	locations := []conf.Location{{
		SkipAuth:     true,
		PathPrefix:   "/api",
		Protocol:     "http",
		TargetModule: "target",
	}}
	handler, err := locator.Handler(config, locations)
	require.NoError(err)

	srv := httptest.NewServer(handler)
	defer srv.Close()

	resp, err := httpcli.New().Post(srv.URL + "/api/endpoint").
		JsonRequestBody(request{Id: "test"}).
		Do(s.T().Context())
	require.NoError(err)
	require.EqualValues(http.StatusOK, resp.StatusCode())

	require.Empty(capture.entries)
}

func TestUnknownLocationsLoggingTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UnknownLocationsLoggingTestSuite))
}
