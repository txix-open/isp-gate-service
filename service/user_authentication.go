// nolint:ireturn
package service

import (
	"context"
	"isp-gate-service/conf"
	"isp-gate-service/domain"
	"isp-gate-service/entity"
	"isp-gate-service/request"
	"isp-gate-service/service/token_provider"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type UserAuthenticationCache interface {
	Get(ctx context.Context, authEndpoint string, token string) (*entity.UserAuthData, error)
	Set(ctx context.Context, authEndpoint string, token string, data entity.UserAuthData, duration time.Duration) error
}

type UserAuthenticationRepo interface {
	Authenticate(ctx context.Context, authEndpoint string, token string) (*entity.UserAuthenticateResponse, error)
}

type TokenProvider interface {
	ExtractToken(ctx *request.Context) (string, error)
}

type userAuthSetting struct {
	tokenProvider     string
	authEndpoint      string
	authCacheDuration time.Duration
	skipAppAuth       bool
}

type UserAuthentication struct {
	cache                UserAuthenticationCache
	repo                 UserAuthenticationRepo
	settingsByModuleName map[string]userAuthSetting
	tokenProviders       map[string]TokenProvider
}

func NewUserAuthentication(
	cfg conf.CustomAuth,
	cache UserAuthenticationCache,
	repo UserAuthenticationRepo,
) (UserAuthentication, error) {
	tokenProviders := make(map[string]TokenProvider, len(cfg.TokenProviders))
	for i, provider := range cfg.TokenProviders {
		_, ok := tokenProviders[provider.Name]
		if ok {
			return UserAuthentication{}, errors.Errorf("token provider name must have unique name, found duplicate at [%d] with name '%s'", i, provider.Name)
		}

		tokenProvider, err := tokenProviderFromConfig(provider)
		if err != nil {
			return UserAuthentication{}, errors.WithMessagef(err, "init token provider with name '%s'", provider.Name)
		}
		tokenProviders[provider.Name] = tokenProvider
	}

	settingsByModuleName := make(map[string]userAuthSetting, len(cfg.UserAuthSettings))
	for _, setting := range cfg.UserAuthSettings {
		_, ok := tokenProviders[setting.TokenProvider]
		if !ok {
			return UserAuthentication{},
				errors.Errorf("modules with names'[%s]' has unknown token provider '%s'",
					strings.Join(setting.ModuleNameList, ","),
					setting.TokenProvider,
				)
		}

		cacheDuration := time.Duration(setting.CacheDataInSec) * time.Second
		for _, moduleName := range setting.ModuleNameList {
			_, ok := settingsByModuleName[moduleName]
			if ok {
				return UserAuthentication{},
					errors.Errorf("setting unique violation, setting for module with name '%s' already present",
						moduleName,
					)
			}
			settingsByModuleName[moduleName] = userAuthSetting{
				tokenProvider:     setting.TokenProvider,
				authEndpoint:      setting.AuthenticateEndpoint,
				authCacheDuration: cacheDuration,
				skipAppAuth:       setting.SkipAppAuth,
			}
		}
	}

	return UserAuthentication{
		cache:                cache,
		repo:                 repo,
		settingsByModuleName: settingsByModuleName,
		tokenProviders:       tokenProviders,
	}, nil
}

func (s UserAuthentication) Authenticate(ctx *request.Context) (*domain.AuthenticateUserResponse, error) {
	meta := ctx.EndpointMeta()
	if !meta.UserAuthRequired {
		return &domain.AuthenticateUserResponse{
			SkipUserAuth: true,
		}, nil
	}

	setting, ok := s.settingsByModuleName[meta.ModuleName]
	if !ok {
		return nil, errors.WithMessagef(domain.ErrUserAuthSettingNotFound,
			"setting for module '%s' and endpoint '%s' not found",
			meta.ModuleName, meta.Endpoint,
		)
	}

	provider := s.tokenProviders[setting.tokenProvider]

	token, err := provider.ExtractToken(ctx)
	if err != nil {
		return nil, errors.WithMessagef(domain.ErrInvalidUserToken,
			"extract token by '%s' error: %s", setting.tokenProvider, err.Error())
	}
	if token == "" {
		return nil, domain.ErrEmptyUserToken
	}

	resp, err := s.authenticate(
		ctx.Context(),
		setting,
		token,
	)
	if err != nil {
		return nil, errors.WithMessage(err, "auth")
	}
	return resp, nil
}

func (s UserAuthentication) authenticate(
	ctx context.Context,
	setting userAuthSetting,
	token string,
) (*domain.AuthenticateUserResponse, error) {
	if setting.authCacheDuration <= 0 {
		resp, err := s.repo.Authenticate(ctx, setting.authEndpoint, token)
		if err != nil {
			return nil, errors.WithMessage(err, "auth repo authenticate")
		}
		return s.convertAuthResponse(resp, setting.skipAppAuth), nil
	}

	authData, err := s.cache.Get(ctx, setting.authEndpoint, token)
	switch {
	case errors.Is(err, domain.ErrAuthenticationCacheMiss):
		resp, err := s.repo.Authenticate(ctx, setting.authEndpoint, token)
		if err != nil {
			return nil, errors.WithMessage(err, "auth repo authenticate")
		}
		if !resp.Authenticated {
			return s.convertAuthResponse(resp, setting.skipAppAuth), nil
		}
		err = s.cache.Set(
			ctx,
			setting.authEndpoint,
			token,
			*resp.AuthData,
			setting.authCacheDuration,
		)
		if err != nil {
			return nil, errors.WithMessage(err, "auth cache set")
		}
		return s.convertAuthResponse(resp, setting.skipAppAuth), nil
	case err != nil:
		return nil, errors.WithMessage(err, "auth cache get")
	default:
		return &domain.AuthenticateUserResponse{
			Authenticated: true,
			ErrorReason:   "",
			AuthData:      s.convertAuthData(authData, setting.skipAppAuth),
		}, nil
	}
}

func (s UserAuthentication) convertAuthResponse(resp *entity.UserAuthenticateResponse, skipAppAuth bool) *domain.AuthenticateUserResponse {
	return &domain.AuthenticateUserResponse{
		Authenticated: resp.Authenticated,
		ErrorReason:   resp.ErrorReason,
		AuthData:      s.convertAuthData(resp.AuthData, skipAppAuth),
	}
}

func (UserAuthentication) convertAuthData(authData *entity.UserAuthData, skipAppAuth bool) *domain.UserAuthData {
	if authData == nil {
		return nil
	}

	return &domain.UserAuthData{
		Identity:       authData.Identity,
		IdentityHeader: authData.IdentityHeader,
		ExtraHeaders:   authData.ExtraHeaders,
		SkipAppAuth:    skipAppAuth,
	}
}

func tokenProviderFromConfig(cfg conf.TokenProvider) (TokenProvider, error) {
	switch cfg.Type {
	case conf.HeaderTokenProviderType:
		if cfg.HeaderProvider == nil {
			return nil, errors.Errorf("token method '%s' has empty header provider", cfg.Name)
		}
		return token_provider.NewHeaderProvider(*cfg.HeaderProvider), nil
	case conf.CookieTokenProviderType:
		if cfg.CookieProvider == nil {
			return nil, errors.Errorf("token method '%s' has empty cookie provider", cfg.Name)
		}
		return token_provider.NewCookieProvider(*cfg.CookieProvider), nil
	default:
		return nil, errors.Errorf("unknown token provider with type '%s'", cfg.Type)
	}
}
