package server

import (
	"github.com/integration-system/go-cmp/cmp"
	"github.com/integration-system/isp-lib/v2/config"
	log "github.com/integration-system/isp-log"
	"github.com/valyala/fasthttp"
	"isp-gate-service/conf"
	"isp-gate-service/handler"
	"isp-gate-service/log_code"
	"sync"
	"time"
)

const defaultTimeout = 60 * time.Second

var Http = &httpSrv{mx: sync.Mutex{}}

type httpSrv struct {
	working bool
	srv     *fasthttp.Server
	mx      sync.Mutex
}

func (s *httpSrv) Init(new, old conf.HttpSetting) {
	if s.working {
		if !cmp.Equal(new, old) {
			s.run(new.GetMaxRequestBodySize())
		}
	} else {
		s.run(new.GetMaxRequestBodySize())
	}
}

func (s *httpSrv) run(maxRequestBodySize int64) {
	s.mx.Lock()
	s.working = true
	if s.srv != nil {
		if err := s.srv.Shutdown(); err != nil {
			log.Warn(log_code.WarnHttpServerShutdown, err)
		}
	}
	localConfig := config.Get().(*conf.Configuration)
	restAddress := localConfig.HttpInnerAddress.GetAddress()
	s.srv = &fasthttp.Server{
		Handler:            handler.New().CompleteRequest,
		WriteTimeout:       defaultTimeout,
		ReadTimeout:        defaultTimeout,
		MaxRequestBodySize: int(maxRequestBodySize),
	}
	go func() {
		if err := s.srv.ListenAndServe(restAddress); err != nil {
			log.Error(log_code.ErrorHttpServerListen, err)
		}
	}()
	s.mx.Unlock()
}

func (s *httpSrv) Close() {
	if s.srv != nil {
		s.working = false
		if err := s.srv.Shutdown(); err != nil {
			log.Warn(log_code.WarnHttpServerShutdown, err)
		}
	}
}
