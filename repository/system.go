// nolint:dupl
package repository

import (
	"context"

	"isp-gate-service/entity"

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

func (r System) Authenticate(ctx context.Context, token string) (*entity.AuthenticateResponse, error) {
	resp := entity.AuthenticateResponse{}
	err := r.cli.Invoke(authenticate).
		JsonRequestBody(entity.AuthenticateRequest{Token: token}).
		JsonResponseBody(&resp).
		Do(ctx)
	if err != nil {
		return nil, errors.WithMessagef(err, "grpc client invoke: %s", authenticate)
	}
	return &resp, nil
}

func (r System) Authorize(ctx context.Context, req entity.AuthorizeRequest) (bool, error) {
	resp := entity.AuthorizeResponse{}
	err := r.cli.Invoke(authorize).
		JsonRequestBody(req).
		JsonResponseBody(&resp).
		Do(ctx)
	if err != nil {
		return false, errors.WithMessagef(err, "grpc client invoke: %s", authorize)
	}
	return resp.Authorized, nil
}
