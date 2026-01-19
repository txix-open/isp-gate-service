package service

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
)

type AuthorizationCache interface {
	Get(ctx context.Context, applicationId int, endpoint string) (bool, error)
	SetAuthorized(ctx context.Context, applicationId int, endpoint string) error
}

type AuthorizationRepo interface {
	AuthorizeOneOf(ctx context.Context, applicationId int, endpoints []string) (bool, error)
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
	ok, err := s.cache.Get(ctx, applicationId, endpoint)
	if err != nil {
		return false, errors.WithMessage(err, "authz cache get")
	}
	if ok {
		return true, nil
	}

	endpoints := []string{
		fmt.Sprintf("%s %s", httpMethod, endpoint),
		endpoint,
	}

	ok, err = s.repo.AuthorizeOneOf(ctx, applicationId, endpoints)
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
