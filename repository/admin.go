// nolint:dupl
package repository

import (
	"context"

	"isp-gate-service/entity"

	"github.com/txix-open/isp-kit/errors"
	"github.com/txix-open/isp-kit/http/httpclix"
	"github.com/txix-open/isp-kit/metrics/http_metrics"
)

const (
	adminAuthenticate = "admin/secure/authenticate"
	adminAuthorize    = "admin/secure/authorize"
)

type Admin struct {
	cli *httpclix.ClientBalancer
}

func NewAdmin(cli *httpclix.ClientBalancer) Admin {
	return Admin{cli: cli}
}

func (r Admin) Authenticate(ctx context.Context, token string) (*entity.AdminAuthenticateResponse, error) {
	ctx = http_metrics.ClientEndpointToContext(ctx, adminAuthenticate)

	resp := entity.AdminAuthenticateResponse{}
	err := r.cli.Post(adminAuthenticate).
		JsonRequestBody(entity.AuthenticateRequest{Token: token}).
		JsonResponseBody(&resp).
		StatusCodeToError().
		DoWithoutResponse(ctx)
	if err != nil {
		return nil, errors.WithMessagef(err, "http client invoke %s", adminAuthenticate)
	}
	return &resp, nil
}

func (r Admin) Authorize(ctx context.Context, adminId int, permission string) (bool, error) {
	ctx = http_metrics.ClientEndpointToContext(ctx, adminAuthorize)

	resp := entity.AdminAuthorizeResponse{}
	err := r.cli.Post(adminAuthorize).
		JsonRequestBody(entity.AdminAuthorizeRequest{AdminId: adminId, Permission: permission}).
		JsonResponseBody(&resp).
		StatusCodeToError().
		DoWithoutResponse(ctx)
	if err != nil {
		return false, errors.WithMessagef(err, "http client invoke %s", adminAuthorize)
	}
	return resp.Authorized, nil
}
