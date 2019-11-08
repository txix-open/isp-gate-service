package matcher

import "sync"

type cacheableMatcher struct {
	cache map[string][]string
	lock  sync.RWMutex
	rm    runtimeMatcher
}

func (mm *cacheableMatcher) Match(method string) []string {
	if len(mm.rm.patterns) == 0 {
		return []string{}
	}

	mm.lock.RLock()
	arrayPattern, ok := mm.cache[method]
	mm.lock.RUnlock()
	if ok {
		return arrayPattern
	}

	mm.lock.Lock()
	defer mm.lock.Unlock()
	if arrayPattern, ok = mm.cache[method]; ok {
		return arrayPattern
	} else if arrayPattern = mm.rm.Match(method); len(arrayPattern) > 0 {
		mm.cache[method] = arrayPattern
	}

	return arrayPattern
}
