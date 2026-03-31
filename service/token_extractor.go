package service

import (
	"isp-gate-service/conf"
	"isp-gate-service/httperrors"
	"isp-gate-service/request"
	"isp-gate-service/service/token_provider"
	"net/http"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

const (
	applicationTokenHeader = "x-application-token"
)

type TokenProvider interface {
	ExtractToken(ctx *request.Context) (string, error)
}

type prefixTokenProviders struct {
	prefix    string
	providers []string
}

type TokenExtractor struct {
	providersByPrefixes []prefixTokenProviders
	tokenProviders      map[string]TokenProvider
}

func NewTokenExtractor(endpointsCfg []conf.AuthEndpointSetting, tokenProviders []conf.TokenProvider) (TokenExtractor, error) {
	providers := make(map[string]TokenProvider, len(tokenProviders))
	for i, provider := range tokenProviders {
		_, ok := providers[provider.Name]
		if ok {
			return TokenExtractor{}, errors.Errorf("token provider name must have unique name, found duplicate at [%d] with name '%s'", i, provider.Name)
		}

		tokenProvider, err := tokenProviderFromConfig(provider)
		if err != nil {
			return TokenExtractor{}, errors.WithMessagef(err, "init token provider with name '%s'", provider.Name)
		}
		providers[provider.Name] = tokenProvider
	}

	providersByPrefixes := make([]prefixTokenProviders, 0, len(endpointsCfg))
	for _, endpoint := range endpointsCfg {
		prefix := strings.TrimPrefix(endpoint.EndpointPrefix, "/")
		for _, providerName := range endpoint.TokenProviders {
			_, ok := providers[providerName]
			if !ok {
				return TokenExtractor{}, errors.Errorf("endpoint with prefix '%s' has unknown token provider '%s'", prefix, providerName)
			}
		}

		providersByPrefixes = append(providersByPrefixes, prefixTokenProviders{
			prefix:    prefix,
			providers: endpoint.TokenProviders,
		})
	}
	sort.Slice(providersByPrefixes, func(i, j int) bool {
		return len(providersByPrefixes[i].prefix) > len(providersByPrefixes[j].prefix)
	})

	return TokenExtractor{
		providersByPrefixes: providersByPrefixes,
		tokenProviders:      providers,
	}, nil
}

func (s TokenExtractor) ExtractToken(ctx *request.Context) (string, string, error) {
	endpoint := ctx.EndpointMeta().NormalizedEndpoint
	for _, providers := range s.providersByPrefixes {
		if !strings.HasPrefix(endpoint, providers.prefix) {
			continue
		}

		return s.customExtractToken(ctx, providers.providers)
	}

	return s.defaultExtractToken(ctx)
}

func (s TokenExtractor) customExtractToken(ctx *request.Context, providers []string) (string, string, error) {
	for _, providerName := range providers {
		provider := s.tokenProviders[providerName]

		token, err := provider.ExtractToken(ctx)
		if err != nil {
			return "", "", errors.WithMessagef(err, "extract token by '%s'", providerName)
		}
		if token != "" {
			return token, "", nil
		}
	}
	return "", "", nil
}

func (s TokenExtractor) defaultExtractToken(ctx *request.Context) (string, string, error) {
	token := ctx.Param(applicationTokenHeader)
	if token != "" {
		return token, "", nil
	}

	appName, token, ok := ctx.Request().BasicAuth()
	if ok && appName == "" {
		return "", "", httperrors.New(
			http.StatusUnauthorized,
			"application name required",
			errors.New("authenticate: application name required on basic auth"),
		)
	}

	return token, appName, nil
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
