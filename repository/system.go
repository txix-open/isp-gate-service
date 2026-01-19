// nolint:dupl
package repository

import (
	"context"

	"isp-gate-service/domain"

	"github.com/pkg/errors"
	"github.com/txix-open/isp-kit/grpc/client"
)

const (
	authenticate = "system/secure/authenticate"
	authorize    = "system/secure/authorize"
)

type System struct {
	cli *client.Client
}

func NewSystem(cli *client.Client) System {
	return System{
		cli: cli,
	}
}

func (r System) Authenticate(ctx context.Context, token string) (*domain.AuthenticateResponse, error) {
	resp := domain.AuthenticateResponse{}
	err := r.cli.Invoke(authenticate).
		JsonRequestBody(domain.AuthenticateRequest{Token: token}).
		JsonResponseBody(&resp).
		Do(ctx)
	if err != nil {
		return nil, errors.WithMessagef(err, "grpc client invoke: %s", authenticate)
	}
	return &resp, nil
}

func (r System) Authorize(ctx context.Context, req domain.AuthorizeRequest) (bool, error) {
	resp := domain.AuthorizeResponse{}
	err := r.cli.Invoke(authorize).
		JsonRequestBody(req).
		JsonResponseBody(&resp).
		Do(ctx)
	if err != nil {
		return false, errors.WithMessagef(err, "grpc client invoke: %s", authorize)
	}
	return resp.Authorized, nil
}
