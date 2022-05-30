package middleware

import (
	"net/http"
)

type Handler interface {
	Handle(ctx *Context) error
}

type HandlerFunc func(ctx *Context) error

func (f HandlerFunc) Handle(ctx *Context) error {
	return f(ctx)
}

type Middleware func(next Handler) Handler

type Context struct {
	Id             string
	Request        *http.Request
	ResponseWriter http.ResponseWriter
	Path           string
	AppId          int
	AdminId        int
	authenticated  bool
}
