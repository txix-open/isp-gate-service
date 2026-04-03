package domain

import (
	"context"
)

type EndpointMeta struct {
	Inner            bool
	UserAuthRequired bool

	ModuleName              string
	RequiredAdminPermission string
	// Объявляемый сервисом метод
	PathSchema string
	// Вызываемый метод
	Endpoint string
	// Нормализованный путь, для известных путей берётся объявляемый метод, для неизвестных - вызываемый
	// Удаляет '/' из начала пути
	NormalizedEndpoint string
}

type endpointMetaKey struct{}

func (m EndpointMeta) ToContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, endpointMetaKey{}, m)
}

func EndpointMetaFromContext(ctx context.Context) EndpointMeta {
	meta, _ := ctx.Value(endpointMetaKey{}).(EndpointMeta)
	return meta
}
