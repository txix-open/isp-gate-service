package middleware

import (
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
	for i := len(middlewares) - 1; i >= 0; i-- {
		root = middlewares[i](root)
	}
	return root
}
