package token_provider

import (
	"isp-gate-service/conf"
	"isp-gate-service/request"
	"strings"
)

type HeaderProvider struct {
	name       string
	headerName string
}

func NewHeaderProvider(name string, cfg conf.HeaderTokenProvider) HeaderProvider {
	return HeaderProvider{
		name:       name,
		headerName: cfg.HeaderName,
	}
}

func (p HeaderProvider) GetName() string {
	return p.name
}

func (p HeaderProvider) ExtractToken(ctx *request.Context) (string, error) {
	value := ctx.Request().Header.Get(p.headerName)
	return strings.TrimSpace(value), nil
}
