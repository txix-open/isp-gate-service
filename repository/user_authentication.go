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
	userAuthenticateFmt = "http://%s/%s/authenticate"
)

type UserAuth struct {
	lb  *lb.RoundRobin
	cli *httpcli.Client
}

func NewUserAuth(lb *lb.RoundRobin) UserAuth {
	return UserAuth{
		lb:  lb,
		cli: httpcli.New(),
	}
}

func (r UserAuth) Authenticate(
	ctx context.Context,
	authModuleName string,
	token string,
) (*entity.UserAuthenticateResponse, error) {
	ctx = http_metrics.ClientEndpointToContext(ctx, userAuthenticateFmt)

	host, err := r.lb.Next()
	if err != nil {
		return nil, errors.WithMessage(err, "lb next")
	}

	endpoint := fmt.Sprintf(userAuthenticateFmt, host, authModuleName)

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
