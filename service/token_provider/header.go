package token_provider

import (
	"isp-gate-service/conf"
	"isp-gate-service/request"
	"strings"
)

type HeaderProvider struct {
	headerName string
}

func NewHeaderProvider(cfg conf.HeaderTokenProvider) HeaderProvider {
	return HeaderProvider{
		headerName: cfg.HeaderName,
	}
}

func (p HeaderProvider) ExtractToken(ctx *request.Context) (string, error) {
	value := ctx.Request().Header.Get(p.headerName)
	return strings.TrimSpace(value), nil
}
