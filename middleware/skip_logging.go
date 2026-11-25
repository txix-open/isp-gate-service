package middleware

import "strings"

// SkipLoggingPaths — префиксы сырого request path, для которых
// полностью отключено логирование (строка лога, не только тела).
type SkipLoggingPaths struct {
	prefixes []string
}

// NewSkipLoggingPaths нормализует значения конфы: обрезает ведущий '/',
// отбрасывает пустые.
func NewSkipLoggingPaths(rawPrefixes []string) SkipLoggingPaths {
	prefixes := make([]string, 0, len(rawPrefixes))
	for _, p := range rawPrefixes {
		p = strings.TrimPrefix(p, "/")
		if p != "" {
			prefixes = append(prefixes, p)
		}
	}
	return SkipLoggingPaths{prefixes: prefixes}
}

// ShouldSkip — case-sensitive prefix-match. path — сырой r.URL.Path.
func (s SkipLoggingPaths) ShouldSkip(path string) bool {
	path = strings.TrimPrefix(path, "/")
	for _, p := range s.prefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}
