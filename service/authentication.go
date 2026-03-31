package service

import (
	"context"
	"isp-gate-service/conf"
	"isp-gate-service/domain"
	"strings"

	"github.com/pkg/errors"
)

type AuthenticationService interface {
	Authenticate(ctx context.Context, token string) (*domain.AuthenticateResponse, error)
}

type CustomAuthenticationService interface {
	Authenticate(ctx context.Context, authName string, token string) (*domain.AuthenticateResponse, error)
}

type prefixAuthenticationProviders struct {
	prefix   string
	provider string
}

type Auth struct {
	baseAuthentication   AuthenticationService
	customAuthentication CustomAuthenticationService
	providersByPrefixes  []prefixAuthenticationProviders
}

func NewAuth(
	endpointsCfg []conf.AuthEndpointSetting,
	baseAuthentication AuthenticationService,
	customAuthentication CustomAuthenticationService,
) Auth {
	providersByPrefixes := make([]prefixAuthenticationProviders, 0, len(endpointsCfg))
	for _, endpoint := range endpointsCfg {
		prefix := strings.TrimPrefix(endpoint.EndpointPrefix, "/")
		providersByPrefixes = append(providersByPrefixes, prefixAuthenticationProviders{
			prefix:   prefix,
			provider: endpoint.AuthProvider,
		})
	}

	return Auth{
		baseAuthentication:   baseAuthentication,
		customAuthentication: customAuthentication,
		providersByPrefixes:  providersByPrefixes,
	}
}

func (s Auth) Authenticate(ctx context.Context, normalizedEndpoint string, token string) (*domain.AuthenticateResponse, error) {
	for _, provider := range s.providersByPrefixes {
		if !strings.HasPrefix(normalizedEndpoint, provider.prefix) {
			continue
		}

		resp, err := s.customAuthentication.Authenticate(ctx, provider.provider, token)
		if err != nil {
			return nil, errors.WithMessage(err, "custom auth")
		}
		return resp, nil
	}

	resp, err := s.baseAuthentication.Authenticate(ctx, token)
	if err != nil {
		return nil, errors.WithMessage(err, "base auth")
	}
	return resp, nil
}
