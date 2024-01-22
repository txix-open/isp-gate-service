package service

import (
	"context"

	"github.com/pkg/errors"
)

type AuthorizationCache interface {
	Get(ctx context.Context, applicationId int, endpoint string) (bool, error)
	SetAuthorized(ctx context.Context, applicationId int, endpoint string) error
}

type AuthorizationRepo interface {
	Authorize(ctx context.Context, applicationId int, endpoint string) (bool, error)
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

func (s Authorization) Authorize(ctx context.Context, applicationId int, endpoint string) (bool, error) {
	ok, err := s.cache.Get(ctx, applicationId, endpoint)
	if err != nil {
		return false, errors.WithMessage(err, "authz cache get")
	}
	if ok {
		return ok, nil
	}

	ok, err = s.repo.Authorize(ctx, applicationId, endpoint)
	if err != nil {
		return false, errors.WithMessagef(err, "authz repo authorize")
	}
	if ok {
		err = s.cache.SetAuthorized(ctx, applicationId, endpoint)
		if err != nil {
			return false, errors.WithMessagef(err, "authz cache set")
		}
	}

	return ok, nil
}
