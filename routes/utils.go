package routes

import "strings"

// nolint:mnd
func normalizeDescriptorPath(path string) string {
	pathParts := strings.Split(path, " ")
	// хак для endpoint'ов с GET /path
	if len(pathParts) == 2 {
		path = pathParts[1]
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}
