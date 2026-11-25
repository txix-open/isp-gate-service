package middleware

import (
	"net/http"

	"github.com/txix-open/isp-kit/log"
)

// UnknownLocation — catch-all для запросов, не попавших ни в один location.
func UnknownLocation(logger log.Logger, skipLogging SkipLoggingPaths) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !skipLogging.ShouldSkip(r.URL.Path) {
			logger.Warn(r.Context(), "request to unknown location",
				log.String("httpMethod", r.Method),
				log.String("path", r.URL.Path),
				log.String("remoteAddr", r.RemoteAddr),
				log.String("xForwardedFor", r.Header.Get("X-Forwarded-For")),
				log.String("userAgent", r.Header.Get("User-Agent")),
				log.Int("statusCode", http.StatusNotFound),
			)
		}

		// Ответ идентичен текущему дефолтному NotFoundHandler gorilla/mux
		http.Error(w, "404 page not found", http.StatusNotFound)
	})
}
