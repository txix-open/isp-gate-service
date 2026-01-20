package domain

import "context"

type EndpointMeta struct {
	Inner                   bool
	RequiredAdminPermission string
	// Объявляемый сервисом метод
	PathSchema string
	// Вызываемый метод
	Endpoint string
}

type endpointMetaKey struct{}

func (m EndpointMeta) ToContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, endpointMetaKey{}, m)
}

func EndpointMetaFromContext(ctx context.Context) EndpointMeta {
	meta, _ := ctx.Value(endpointMetaKey{}).(EndpointMeta)
	return meta
}
