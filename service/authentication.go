package service

import (
	"context"
	"isp-gate-service/domain"
	"isp-gate-service/entity"

	"github.com/pkg/errors"
)

type AuthenticationCache interface {
	Get(ctx context.Context, token string) (*entity.AppAuthData, error)
	Set(ctx context.Context, token string, data entity.AppAuthData) error
}

type AuthenticationRepo interface {
	Authenticate(ctx context.Context, token string) (*entity.AuthenticateResponse, error)
}

type Authentication struct {
	cache AuthenticationCache
	repo  AuthenticationRepo
}

func NewAuthentication(
	cache AuthenticationCache,
	repo AuthenticationRepo,
) Authentication {
	return Authentication{
		cache: cache,
		repo:  repo,
	}
}

func (s Authentication) Authenticate(ctx context.Context, token string) (*domain.AuthenticateAppResponse, error) {
	authData, err := s.cache.Get(ctx, token)
	switch {
	case errors.Is(err, domain.ErrAuthenticationCacheMiss):
		resp, err := s.repo.Authenticate(ctx, token)
		if err != nil {
			return nil, errors.WithMessage(err, "auth repo authenticate")
		}
		if !resp.Authenticated {
			return s.convertAuthReponse(resp), nil
		}
		err = s.cache.Set(ctx, token, *resp.AuthData)
		if err != nil {
			return nil, errors.WithMessage(err, "auth cache set")
		}
		return s.convertAuthReponse(resp), nil
	case err != nil:
		return nil, errors.WithMessage(err, "auth cache get")
	default:
		return &domain.AuthenticateAppResponse{
			Authenticated: true,
			ErrorReason:   "",
			AuthData:      s.convertAuthData(authData),
		}, nil
	}
}

func (s Authentication) convertAuthReponse(resp *entity.AuthenticateResponse) *domain.AuthenticateAppResponse {
	return &domain.AuthenticateAppResponse{
		Authenticated: resp.Authenticated,
		ErrorReason:   resp.ErrorReason,
		AuthData:      s.convertAuthData(resp.AuthData),
	}
}

func (Authentication) convertAuthData(authData *entity.AppAuthData) *domain.AppAuthData {
	if authData == nil {
		return nil
	}

	return &domain.AppAuthData{
		AppName:       authData.AppName,
		SystemId:      authData.SystemId,
		DomainId:      authData.DomainId,
		ServiceId:     authData.ServiceId,
		ApplicationId: authData.ApplicationId,
	}
}
