package matcher

import "testing"

var (
	patterns = []string{
		"mdm-master/group/method",
		"mdm-master/group/*",
		"mdm-master/group2/method",
		"mdm-master/group3/*",
	}
	cases = []struct {
		method string
		length int
	}{
		{method: "mdm-master/group/method", length: 2},
		{method: "mdm-master/group/method2", length: 1},
		{method: "mdm-master/group2/method3", length: 0},
		{method: "mdm-master/group2/method", length: 1},
		{method: "mdm-master/group3/some", length: 1},
		{method: "mdm-master/group3/", length: 1},
	}
)

func TestCacheableMatcher_Match(t *testing.T) {
	matcher := NewCacheableMatcher(patterns)
	for _, c := range cases {
		res := matcher.Match(c.method)
		if c.length != len(res) {
			t.Error(c)
		}
	}
}

func BenchmarkCacheableMatcher_Match(b *testing.B) {
	matcher := NewCacheableMatcher(patterns)
	for i := 0; i < b.N; i++ {
		for _, c := range cases {
			_ = matcher.Match(c.method)
		}
	}
}

func BenchmarkRuntimeMatcher_Match(b *testing.B) {
	matcher := NewRuntimeMatcher(patterns)
	for i := 0; i < b.N; i++ {
		for _, c := range cases {
			_ = matcher.Match(c.method)
		}
	}
}
