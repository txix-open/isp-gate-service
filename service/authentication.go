package service

import (
	"context"

	"github.com/pkg/errors"
	"isp-gate-service/domain"
)

type AuthenticationCache interface {
	Get(ctx context.Context, token string) (*domain.AuthData, error)
	Set(ctx context.Context, token string, data domain.AuthData) error
}

type AuthenticationRepo interface {
	Authenticate(ctx context.Context, token string) (*domain.AuthenticateResponse, error)
}

type Auth struct {
	cache AuthenticationCache
	repo  AuthenticationRepo
}

func NewAuthentication(cache AuthenticationCache, repo AuthenticationRepo) Auth {
	return Auth{
		cache: cache,
		repo:  repo,
	}
}

func (s Auth) Authenticate(ctx context.Context, token string) (*domain.AuthenticateResponse, error) {
	authData, err := s.cache.Get(ctx, token)
	if errors.Is(err, domain.ErrAuthenticationCacheMiss) {
		resp, err := s.repo.Authenticate(ctx, token)
		if err != nil {
			return nil, errors.WithMessage(err, "auth repo authenticate")
		}
		if !resp.Authenticated {
			return resp, nil
		}
		err = s.cache.Set(ctx, token, *resp.AuthData)
		if err != nil {
			return nil, errors.WithMessage(err, "auth cache set")
		}
		return resp, nil
	}
	if err != nil {
		return nil, errors.WithMessage(err, "auth cache get")
	}
	return &domain.AuthenticateResponse{
		Authenticated: true,
		ErrorReason:   "",
		AuthData:      authData,
	}, nil
}
