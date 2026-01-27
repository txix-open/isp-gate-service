package service

import (
	"context"
	"fmt"
	"isp-gate-service/domain"

	"github.com/pkg/errors"
)

type AuthorizationCache interface {
	Get(ctx context.Context, applicationId int, endpoint string) (bool, error)
	SetAuthorized(ctx context.Context, applicationId int, endpoint string) error
}

type AuthorizationRepo interface {
	Authorize(ctx context.Context, req domain.AuthorizeRequest) (bool, error)
}

type Authorization struct {
	cache AuthorizationCache
	repo  AuthorizationRepo
}

func NewAuthorization(cache AuthorizationCache, repo AuthorizationRepo) Authorization {
	return Authorization{
		cache: cache,
		repo:  repo,
	}
}

func (s Authorization) Authorize(ctx context.Context, applicationId int, httpMethod string, endpoint string) (bool, error) {
	cacheKey := fmt.Sprintf("%s %s", httpMethod, endpoint)
	ok, err := s.cache.Get(ctx, applicationId, cacheKey)
	if err != nil {
		return false, errors.WithMessage(err, "authz cache get")
	}
	if ok {
		return true, nil
	}

	ok, err = s.repo.Authorize(ctx, domain.AuthorizeRequest{
		ApplicationId: applicationId,
		HttpMethod:    httpMethod,
		Endpoint:      endpoint,
	})
	if err != nil {
		return false, errors.WithMessagef(err, "authz repo authorize")
	}
	if ok {
		err = s.cache.SetAuthorized(ctx, applicationId, cacheKey)
		if err != nil {
			return false, errors.WithMessagef(err, "authz cache set")
		}
	}

	return ok, nil
}
