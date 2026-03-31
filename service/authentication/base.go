package authentication

import (
	"context"
	"isp-gate-service/domain"
	"isp-gate-service/entity"

	"github.com/pkg/errors"
)

type AuthenticationCache interface {
	Get(ctx context.Context, token string) (*entity.AuthData, error)
	Set(ctx context.Context, token string, data entity.AuthData) error
}

type AuthenticationRepo interface {
	Authenticate(ctx context.Context, token string) (*entity.AuthenticateResponse, error)
}

type BaseAuthentication struct {
	cache AuthenticationCache
	repo  AuthenticationRepo
}

func NewBaseAuthentication(
	cache AuthenticationCache,
	repo AuthenticationRepo,
) BaseAuthentication {
	return BaseAuthentication{
		cache: cache,
		repo:  repo,
	}
}

func (s BaseAuthentication) Authenticate(ctx context.Context, token string) (*domain.AuthenticateResponse, error) {
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
		return &domain.AuthenticateResponse{
			Authenticated: true,
			ErrorReason:   "",
			AuthData:      s.convertAuthData(authData),
		}, nil
	}
}

func (s BaseAuthentication) convertAuthReponse(resp *entity.AuthenticateResponse) *domain.AuthenticateResponse {
	return &domain.AuthenticateResponse{
		Authenticated: resp.Authenticated,
		ErrorReason:   resp.ErrorReason,
		AuthData:      s.convertAuthData(resp.AuthData),
	}
}

func (BaseAuthentication) convertAuthData(authData *entity.AuthData) *domain.AuthData {
	return &domain.AuthData{
		AppName:       authData.AppName,
		SystemId:      authData.SystemId,
		DomainId:      authData.DomainId,
		ServiceId:     authData.ServiceId,
		ApplicationId: authData.ApplicationId,
	}
}
