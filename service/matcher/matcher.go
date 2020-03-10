package matcher

var JournalMethods AtLeastOneMatcher = &atLeastOneMatcher{}

type Matcher interface {
	Match(method string) []string
}

type AtLeastOneMatcher interface {
	Match(method string) bool
}

func NewRuntimeMatcher(patterns []string) Matcher {
	return &runtimeMatcher{
		patterns: patterns,
	}
}

func NewCacheableMatcher(patterns []string) Matcher {
	return &cacheableMatcher{
		cache: make(map[string][]string),
		rm:    runtimeMatcher{patterns: patterns},
	}
}

func NewAtLeastOneMatcher(patterns []string) AtLeastOneMatcher {
	return &atLeastOneMatcher{
		cache: make(map[string]bool),
		rm:    runtimeMatcher{patterns: patterns},
	}
}
