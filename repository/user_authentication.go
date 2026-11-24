package repository

import (
	"context"
	"fmt"
	"isp-gate-service/entity"

	"github.com/txix-open/isp-kit/errors"
	"github.com/txix-open/isp-kit/http/httpcli"
	"github.com/txix-open/isp-kit/http/httpclix"
	"github.com/txix-open/isp-kit/lb"
	"github.com/txix-open/isp-kit/metrics/http_metrics"
)

type UserAuth struct {
	lb  *lb.RoundRobin
	cli *httpcli.Client
}

func NewUserAuth(lb *lb.RoundRobin) UserAuth {
	return UserAuth{
		lb:  lb,
		cli: httpclix.Default(),
	}
}

func (r UserAuth) Authenticate(
	ctx context.Context,
	method string,
	token string,
) (*entity.UserAuthenticateResponse, error) {
	ctx = http_metrics.ClientEndpointToContext(ctx, method)

	host, err := r.lb.Next()
	if err != nil {
		return nil, errors.WithMessage(err, "lb next")
	}

	endpoint := fmt.Sprintf("http://%s/%s", host, method)
	resp := entity.UserAuthenticateResponse{}
	err = r.cli.Post(endpoint).
		JsonRequestBody(entity.UserAuthenticateRequest{
			Token: token,
		}).JsonResponseBody(&resp).
		StatusCodeToError().
		DoWithoutResponse(ctx)
	if err != nil {
		return nil, errors.WithMessagef(err, "http client invoke: %s", endpoint)
	}
	return &resp, nil
}

func (r UserAuth) Authorize(
	ctx context.Context,
	method string,
	identity string,
	permission string,
) (bool, error) {
	ctx = http_metrics.ClientEndpointToContext(ctx, method)

	host, err := r.lb.Next()
	if err != nil {
		return false, errors.WithMessage(err, "lb next")
	}

	endpoint := fmt.Sprintf("http://%s/%s", host, method)
	resp := entity.UserAuthorizeResponse{}
	err = r.cli.Post(endpoint).
		JsonRequestBody(entity.UserAuthorizeRequest{
			Identity:   identity,
			Permission: permission,
		}).JsonResponseBody(&resp).
		StatusCodeToError().
		DoWithoutResponse(ctx)
	if err != nil {
		return false, errors.WithMessagef(err, "http client invoke: %s", endpoint)
	}
	return resp.Authorized, nil
}
