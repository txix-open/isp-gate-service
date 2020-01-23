package server

import (
	"github.com/integration-system/go-cmp/cmp"
	"github.com/integration-system/isp-lib/config"
	log "github.com/integration-system/isp-log"
	"github.com/valyala/fasthttp"
	"isp-gate-service/conf"
	"isp-gate-service/handler"
	"isp-gate-service/log_code"
	"sync"
	"time"
)

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

func (s *httpSrv) run(MaxRequestBodySize int64) {
	s.mx.Lock()
	s.working = true
	if s.srv != nil {
		if err := s.srv.Shutdown(); err != nil {
			log.Warn(log_code.WarnHttpServerShutdown, err)
		}
	}
	maxRequestBodySize := MaxRequestBodySize
	localConfig := config.Get().(*conf.Configuration)
	restAddress := localConfig.HttpInnerAddress.GetAddress()
	s.srv = &fasthttp.Server{
		Handler:            handler.CompleteRequest,
		WriteTimeout:       time.Second * 60,
		ReadTimeout:        time.Second * 60,
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
		if err := s.srv.Shutdown(); err != nil {
			log.Warn(log_code.WarnHttpServerShutdown, err)
		}
	}
}
