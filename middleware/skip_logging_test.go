package middleware_test

import (
	"testing"

	"isp-gate-service/middleware"

	"github.com/stretchr/testify/require"
)

func TestSkipLoggingPaths_ShouldSkip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		prefixes []string
		path     string
		wantSkip bool
	}{
		{name: "prefix without leading slash", prefixes: []string{"health"}, path: "/health", wantSkip: true},
		{name: "prefix with leading slash is equivalent", prefixes: []string{"/health"}, path: "/health", wantSkip: true},
		{name: "no match", prefixes: []string{"health"}, path: "/other", wantSkip: false},
		{name: "empty value is discarded", prefixes: []string{""}, path: "/health", wantSkip: false},
		{name: "root slash value is discarded", prefixes: []string{"/"}, path: "/health", wantSkip: false},
		{name: "case sensitive", prefixes: []string{"Health"}, path: "/health/x", wantSkip: false},
		{name: "plain prefix matches longer name", prefixes: []string{"health"}, path: "/healthz", wantSkip: true},
		{name: "plain prefix matches nested path", prefixes: []string{"health"}, path: "/health/x", wantSkip: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			matcher := middleware.NewSkipLoggingPaths(test.prefixes)
			require.Equal(t, test.wantSkip, matcher.ShouldSkip(test.path))
		})
	}
}

func TestSkipLoggingPaths_ShouldSkip_EmptyList(t *testing.T) {
	t.Parallel()

	matcher := middleware.NewSkipLoggingPaths(nil)
	for _, path := range []string{"/", "/health", "/health/x", "/unknown"} {
		require.False(t, matcher.ShouldSkip(path))
	}
}
