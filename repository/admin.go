package repository

import (
	"context"
	"isp-gate-service/domain"

	"github.com/integration-system/isp-kit/grpc/client"
	"github.com/pkg/errors"
)

const (
	adminAuthenticate = "admin/secure/authenticate"
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
		ReadJsonResponse(&resp).
		Do(ctx)
	if err != nil {
		return nil, errors.WithMessagef(err, "grpc client invoke %s", adminAuthenticate)
	}
	return &resp, nil
}
