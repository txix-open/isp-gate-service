package service

import (
	"context"
	"isp-gate-service/conf"
	"isp-gate-service/domain"
	"isp-gate-service/entity"
	"isp-gate-service/request"
	"isp-gate-service/service/token_provider"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type UserAuthenticationCache interface {
	Get(ctx context.Context, authModuleName string, token string) (*entity.UserAuthData, error)
	Set(ctx context.Context, authModuleName string, token string, data entity.UserAuthData, duration time.Duration) error
}

type UserAuthenticationRepo interface {
	Authenticate(ctx context.Context, authModuleName string, token string) (*entity.UserAuthenticateResponse, error)
}

type TokenProvider interface {
	ExtractToken(ctx *request.Context) (string, error)
}

type userAuthSetting struct {
	tokenProviders    []string
	endpointPrefix    string
	authModuleName    string
	authCacheDuration time.Duration
	skipAppAuth       bool
}

type UserAuthentication struct {
	cache             UserAuthenticationCache
	repo              UserAuthenticationRepo
	endpointsSettings []userAuthSetting
	tokenProviders    map[string]TokenProvider
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

	endpointsSettings := make([]userAuthSetting, 0, len(cfg.UserAuthSettings))
	for _, setting := range cfg.UserAuthSettings {
		for i := range setting.EndpointPrefixes {
			setting.EndpointPrefixes[i] = strings.TrimPrefix(setting.EndpointPrefixes[i], "/")
		}

		for _, providerName := range setting.TokenProviders {
			_, ok := tokenProviders[providerName]
			if !ok {
				return UserAuthentication{}, errors.Errorf("endpoint with prefixes '[%s]' has unknown token provider '%s'",
					strings.Join(setting.EndpointPrefixes, ","),
					providerName,
				)
			}
		}

		cacheDuration := time.Duration(setting.CacheDataInSec) * time.Second

		for _, prefix := range setting.EndpointPrefixes {
			endpointsSettings = append(endpointsSettings, userAuthSetting{
				endpointPrefix:    prefix,
				authModuleName:    setting.AuthModuleName,
				tokenProviders:    setting.TokenProviders,
				skipAppAuth:       setting.SkipAppAuth,
				authCacheDuration: cacheDuration,
			})
		}
	}
	sort.Slice(endpointsSettings, func(i, j int) bool {
		return len(endpointsSettings[i].endpointPrefix) > len(endpointsSettings[j].endpointPrefix)
	})

	return UserAuthentication{
		cache:             cache,
		repo:              repo,
		endpointsSettings: endpointsSettings,
		tokenProviders:    tokenProviders,
	}, nil
}

func (s UserAuthentication) Authenticate(ctx *request.Context) (*domain.AuthenticateUserResponse, error) {
	normalizedEndpoint := ctx.EndpointMeta().NormalizedEndpoint
	for _, setting := range s.endpointsSettings {
		if !strings.HasPrefix(normalizedEndpoint, setting.endpointPrefix) {
			continue
		}

		token, err := s.extractToken(ctx, setting.tokenProviders)
		if err != nil {
			return nil, errors.WithMessage(err, "extract user token")
		}
		if token == "" {
			return &domain.AuthenticateUserResponse{
				Authenticated: false,
				ErrorReason:   "failed extract user token",
			}, nil
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

	return &domain.AuthenticateUserResponse{
		SkipUserAuth: true,
	}, nil
}

func (s UserAuthentication) extractToken(ctx *request.Context, providers []string) (string, error) {
	for _, providerName := range providers {
		provider := s.tokenProviders[providerName]

		token, err := provider.ExtractToken(ctx)
		if err != nil {
			return "", errors.WithMessagef(err, "extract token by '%s'", providerName)
		}
		if token != "" {
			return token, nil
		}
	}
	return "", nil
}

func (s UserAuthentication) authenticate(
	ctx context.Context,
	setting userAuthSetting,
	token string,
) (*domain.AuthenticateUserResponse, error) {
	if setting.authCacheDuration <= 0 {
		resp, err := s.repo.Authenticate(ctx, setting.authModuleName, token)
		if err != nil {
			return nil, errors.WithMessage(err, "auth repo authenticate")
		}
		return s.convertAuthReponse(resp, setting.skipAppAuth), nil
	}

	authData, err := s.cache.Get(ctx, setting.authModuleName, token)
	switch {
	case errors.Is(err, domain.ErrAuthenticationCacheMiss):
		resp, err := s.repo.Authenticate(ctx, setting.authModuleName, token)
		if err != nil {
			return nil, errors.WithMessage(err, "auth repo authenticate")
		}
		if !resp.Authenticated {
			return s.convertAuthReponse(resp, setting.skipAppAuth), nil
		}
		err = s.cache.Set(
			ctx,
			setting.authModuleName,
			token,
			*resp.AuthData,
			setting.authCacheDuration,
		)
		if err != nil {
			return nil, errors.WithMessage(err, "auth cache set")
		}
		return s.convertAuthReponse(resp, setting.skipAppAuth), nil
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

func (s UserAuthentication) convertAuthReponse(resp *entity.UserAuthenticateResponse, skipAppAuth bool) *domain.AuthenticateUserResponse {
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

//nolint:ireturn
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
