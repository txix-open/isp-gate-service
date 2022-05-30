package middleware

import (
	"fmt"
	"strings"
)

func Path(pathPrefix string) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *Context) error {
			prefix := fmt.Sprintf("%s/", pathPrefix)
			ctx.Path = strings.TrimPrefix(ctx.Request.URL.String(), prefix)

			return next.Handle(ctx)
		})
	}
}
