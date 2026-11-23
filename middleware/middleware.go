package middleware

import (
	"slices"

	"isp-gate-service/request"
)

type Handler interface {
	Handle(ctx *request.Context) error
}

type HandlerFunc func(ctx *request.Context) error

func (f HandlerFunc) Handle(ctx *request.Context) error {
	return f(ctx)
}

type Middleware func(next Handler) Handler

// nolint:ireturn
func Chain(root Handler, middlewares ...Middleware) Handler {
	for _, middleware := range slices.Backward(middlewares) {
		root = middleware(root)
	}
	return root
}
