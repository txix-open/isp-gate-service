package matcher

import "path"

type runtimeMatcher struct {
	patterns []string
}

func (mm runtimeMatcher) Match(method string) []string {
	resp := make([]string, 0)
	for _, pattern := range mm.patterns {
		if matched, _ := path.Match(pattern, method); matched {
			resp = append(resp, pattern)
		}
	}
	return resp
}
