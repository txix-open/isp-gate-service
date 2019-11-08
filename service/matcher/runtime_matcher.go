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
		patternLen := len(pattern)
		if patternLen > 0 && pattern[patternLen-1] == '*' {
			pattern = pattern[:patternLen-1]
			if strings.HasPrefix(method, pattern) {
				resp = append(resp, pattern)
			}
		} else if strings.EqualFold(method, pattern) {
			resp = append(resp, pattern)
		}
	}
	return resp
}
