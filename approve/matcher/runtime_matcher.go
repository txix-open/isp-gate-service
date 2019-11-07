package matcher

import (
	"strings"
)

type runtimeMatcher struct {
	patterns []string
}

func (mm runtimeMatcher) Match(method string) []string {
	resp := make([]string, 0)
	for _, pattern := range mm.patterns {
		if strings.HasPrefix(method, pattern) {
			resp = append(resp, pattern)
		}
	}
	return resp
}
