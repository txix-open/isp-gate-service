package service

import (
	"context"

	"isp-gate-service/domain"

	"github.com/pkg/errors"
)

type AdminAuth interface {
	Authenticate(ctx context.Context, token string) (*domain.AdminAuthenticateResponse, error)
	Authorize(ctx context.Context, adminId int, permission string) (bool, error)
}

type Admin struct {
	cache AuthorizationCache
	repo  AdminAuth
}

func NewAdmin(cache AuthorizationCache, adminAuth AdminAuth) Admin {
	return Admin{
		cache: cache,
		repo:  adminAuth,
	}
}

func (s Admin) AdminAuthenticate(ctx context.Context, token string) (*domain.AdminAuthenticateResponse, error) {
	resp, err := s.repo.Authenticate(ctx, token)
	if err != nil {
		return nil, errors.WithMessage(err, "get admin token data from admin service")
	}
	return resp, nil
}

func (s Admin) AdminAuthorize(ctx context.Context, adminId int, permission string) (bool, error) {
	ok, err := s.cache.Get(ctx, adminId, permission)
	if err != nil {
		return false, errors.WithMessage(err, "authz cache get")
	}
	if ok {
		return ok, nil
	}

	ok, err = s.repo.Authorize(ctx, adminId, permission)
	if err != nil {
		return false, errors.WithMessagef(err, "authz repo authorize")
	}
	if ok {
		err = s.cache.SetAuthorized(ctx, adminId, permission)
		if err != nil {
			return false, errors.WithMessagef(err, "authz cache set")
		}
	}

	return ok, nil
}
