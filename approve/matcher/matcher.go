package matcher

type Matcher interface {
	Match(method string) []string
}

func NewRuntimeMatcher(patterns []string) Matcher {
	return &runtimeMatcher{
		patterns: patterns,
	}
}

func NewCacheableMatcher(patterns []string) Matcher {
	return &cacheableMatcher{
		cache: make(map[string][]string, 0),
		rm:    runtimeMatcher{patterns: patterns},
	}
}
