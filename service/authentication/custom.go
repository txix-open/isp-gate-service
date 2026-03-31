package authentication

import (
	"context"
	"isp-gate-service/conf"
	"isp-gate-service/domain"
	"isp-gate-service/entity"

	"github.com/pkg/errors"
)

type CustomAuthenticationCache interface {
	Get(ctx context.Context, authName string, token string) (*entity.CustomAuthenticateResponse, error)
	Set(ctx context.Context, authName string, token string, data entity.CustomAuthenticateResponse) error
}

type CustomAuthenticationRepo interface {
	Authenticate(ctx context.Context, moduleName string, token string) (*entity.CustomAuthenticateResponse, error)
}

type CustomAuthentication struct {
	cache                CustomAuthenticationCache
	repo                 CustomAuthenticationRepo
	moduleNameByAuthName map[string]string
}

func NewCustomAuthentication(
	cfg []conf.AuthProvider,
	cache CustomAuthenticationCache,
	repo CustomAuthenticationRepo,
) CustomAuthentication {
	moduleNameByAuthName := make(map[string]string, len(cfg))
	for _, provider := range cfg {
		moduleNameByAuthName[provider.Name] = provider.ModuleName
	}

	return CustomAuthentication{
		cache:                cache,
		repo:                 repo,
		moduleNameByAuthName: moduleNameByAuthName,
	}
}

func (s CustomAuthentication) Authenticate(ctx context.Context, authName string, token string) (*domain.AuthenticateResponse, error) {
	resp, err := s.cache.Get(ctx, authName, token)

	moduleName := s.moduleNameByAuthName[authName]
	switch {
	case errors.Is(err, domain.ErrAuthenticationCacheMiss):
		resp, err := s.repo.Authenticate(ctx, moduleName, token)
		if err != nil {
			return nil, errors.WithMessage(err, "auth repo authenticate")
		}
		if !resp.Authenticated {
			return s.convertAuthReponse(resp), nil
		}
		err = s.cache.Set(ctx, moduleName, token, *resp)
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
			AuthData:      s.convertAuthData(resp),
		}, nil
	}
}

func (s CustomAuthentication) convertAuthReponse(resp *entity.CustomAuthenticateResponse) *domain.AuthenticateResponse {
	return &domain.AuthenticateResponse{
		Authenticated: resp.Authenticated,
		ErrorReason:   resp.ErrorReason,

		AuthData: &domain.AuthData{
			CustomAuthData: &domain.ThirdPartyAuthData{
				Identity:       resp.Identity,
				IdentityHeader: resp.IdentityHeader,
				ExtraHeaders:   resp.ExtraHeaders,
			},
		},
	}
}

func (CustomAuthentication) convertAuthData(resp *entity.CustomAuthenticateResponse) *domain.AuthData {
	return &domain.AuthData{
		CustomAuthData: &domain.ThirdPartyAuthData{
			Identity:       resp.Identity,
			IdentityHeader: resp.IdentityHeader,
			ExtraHeaders:   resp.ExtraHeaders,
		},
	}
}
