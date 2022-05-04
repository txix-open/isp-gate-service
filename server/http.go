package server

import (
	"sync"
	"time"

	"github.com/integration-system/isp-kit/log"
	"github.com/integration-system/isp-lib/v2/config"
	logrus "github.com/integration-system/isp-log"
	"github.com/valyala/fasthttp"
	"isp-gate-service/conf"
	"isp-gate-service/handler"
	"isp-gate-service/log_code"
)

const defaultTimeout = 60 * time.Second

var Http = &httpSrv{mx: sync.Mutex{}}

type httpSrv struct {
	working bool
	srv     *fasthttp.Server
	mx      sync.Mutex
	logger  log.Logger
}

func (s *httpSrv) Init(isDifferentSetting bool, bodySize int64, logger log.Logger) {
	s.logger = logger
	if s.working {
		if isDifferentSetting {
			s.run(bodySize)
		}
	} else {
		s.run(bodySize)
	}
}

func (s *httpSrv) run(maxRequestBodySize int64) {
	s.mx.Lock()
	s.working = true
	if s.srv != nil {
		if err := s.srv.Shutdown(); err != nil {
			logrus.Warn(log_code.WarnHttpServerShutdown, err)
		}
	}
	localConfig := config.Get().(*conf.Configuration)
	restAddress := localConfig.HttpInnerAddress.GetAddress()
	s.srv = &fasthttp.Server{
		Handler:            handler.New(s.logger).CompleteRequest,
		WriteTimeout:       defaultTimeout,
		ReadTimeout:        defaultTimeout,
		MaxRequestBodySize: int(maxRequestBodySize),
	}
	go func() {
		if err := s.srv.ListenAndServe(restAddress); err != nil {
			logrus.Error(log_code.ErrorHttpServerListen, err)
		}
	}()
	s.mx.Unlock()
}

func (s *httpSrv) Close() {
	if s.srv != nil {
		s.working = false
		if err := s.srv.Shutdown(); err != nil {
			logrus.Warn(log_code.WarnHttpServerShutdown, err)
		}
	}
}
