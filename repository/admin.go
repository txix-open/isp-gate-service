// nolint:dupl
package repository

import (
	"context"

	"isp-gate-service/domain"

	"github.com/pkg/errors"
	"github.com/txix-open/isp-kit/grpc/client"
)

const (
	adminAuthenticate = "admin/secure/authenticate"
	adminAuthorize    = "admin/secure/authorize"
)

type Admin struct {
	cli *client.Client
}

func NewAdmin(cli *client.Client) Admin {
	return Admin{cli: cli}
}

func (r Admin) Authenticate(ctx context.Context, token string) (*domain.AdminAuthenticateResponse, error) {
	resp := domain.AdminAuthenticateResponse{}
	err := r.cli.Invoke(adminAuthenticate).
		JsonRequestBody(domain.AuthenticateRequest{Token: token}).
		JsonResponseBody(&resp).
		Do(ctx)
	if err != nil {
		return nil, errors.WithMessagef(err, "grpc client invoke %s", adminAuthenticate)
	}
	return &resp, nil
}

func (r Admin) Authorize(ctx context.Context, adminId int, permission string) (bool, error) {
	resp := domain.AdminAuthorizeResponse{}
	err := r.cli.Invoke(adminAuthorize).
		JsonRequestBody(domain.AdminAuthorizeRequest{AdminId: adminId, Permission: permission}).
		JsonResponseBody(&resp).
		Do(ctx)
	if err != nil {
		return false, errors.WithMessagef(err, "grpc client invoke: %s", adminAuthorize)
	}
	return resp.Authorized, nil
}
