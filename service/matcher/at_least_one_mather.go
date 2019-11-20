package matcher

import "sync"

type atLeastOneMatcher struct {
	cache map[string]bool
	lock  sync.RWMutex
	rm    runtimeMatcher
}

func (mm *atLeastOneMatcher) Match(method string) bool {
	if len(mm.rm.patterns) == 0 {
		return false
	}

	mm.lock.RLock()
	matched, ok := mm.cache[method]
	mm.lock.RUnlock()
	if ok {
		return matched
	}

	mm.lock.Lock()
	defer mm.lock.Unlock()
	if matched, ok = mm.cache[method]; ok {
		return matched
	} else {
		arrayPattern := mm.rm.Match(method)
		matched = len(arrayPattern) > 0
	}
	mm.cache[method] = matched
	return matched
}
