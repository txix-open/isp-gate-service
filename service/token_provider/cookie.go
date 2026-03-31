package token_provider

import (
	"isp-gate-service/conf"
	"isp-gate-service/request"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

type CookieProvider struct {
	cookieName string
	validate   bool
}

func NewCookieProvider(cfg conf.CookieTokenProvider) CookieProvider {
	return CookieProvider{
		cookieName: cfg.CookieName,
		validate:   cfg.Validate,
	}
}

func (p CookieProvider) ExtractToken(ctx *request.Context) (string, error) {
	cookie, err := ctx.Request().Cookie(p.cookieName)
	switch {
	case errors.Is(err, http.ErrNoCookie):
		return "", nil
	case err != nil:
		return "", errors.WithMessagef(err, "get cookie with name '%s'", p.cookieName)
	}
	if !p.validate {
		return strings.TrimSpace(cookie.Value), nil
	}

	err = cookie.Valid()
	if err != nil {
		return "", errors.WithMessagef(err, "validate cookie with name '%s'", p.cookieName)
	}
	return strings.TrimSpace(cookie.Value), nil
}
