package repository

import (
	"context"
	"fmt"
	"isp-gate-service/entity"

	"github.com/pkg/errors"
	"github.com/txix-open/isp-kit/http/httpcli"
	"github.com/txix-open/isp-kit/lb"
	"github.com/txix-open/isp-kit/metrics/http_metrics"
)

const (
	thirdPartyAuthenticateFmt = "http://%s/%s/authenticate"
)

type ThirdPartyAuth struct {
	lb  *lb.RoundRobin
	cli *httpcli.Client
}

func NewCustomAuth(lb *lb.RoundRobin) ThirdPartyAuth {
	return ThirdPartyAuth{
		lb:  lb,
		cli: httpcli.New(),
	}
}

func (r ThirdPartyAuth) Authenticate(
	ctx context.Context,
	moduleName string,
	token string,
) (*entity.CustomAuthenticateResponse, error) {
	ctx = http_metrics.ClientEndpointToContext(ctx, thirdPartyAuthenticateFmt)

	host, err := r.lb.Next()
	if err != nil {
		return nil, errors.WithMessage(err, "lb next")
	}

	endpoint := fmt.Sprintf(thirdPartyAuthenticateFmt, host, moduleName)

	resp := entity.CustomAuthenticateResponse{}
	err = r.cli.Post(endpoint).
		JsonRequestBody(entity.CustomAuthenticateRequest{
			Token: token,
		}).JsonResponseBody(&resp).
		StatusCodeToError().
		DoWithoutResponse(ctx)
	if err != nil {
		return nil, errors.WithMessagef(err, "http client invoke: %s", endpoint)
	}
	return &resp, nil
}
