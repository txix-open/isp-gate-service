//nolint
package matcher

import (
	"testing"
)

var (
	patterns = []string{
		"mdm-master/group/method",
		"mdm-master/group/*",
		"mdm-master/group2/method",
		"mdm-master/group3/*",
	}
	cases = []struct {
		method      string
		coincidence int
		atLeastOne  bool
	}{
		{method: "mdm-master/group/method", coincidence: 2, atLeastOne: true},
		{method: "mdm-master/group/method2", coincidence: 1, atLeastOne: true},
		{method: "mdm-master/group2/", coincidence: 0, atLeastOne: false},
		{method: "mdm-master/group2/method3", coincidence: 0, atLeastOne: false},
		{method: "mdm-master/group2/method", coincidence: 1, atLeastOne: true},
		{method: "mdm-master/group3/some", coincidence: 1, atLeastOne: true},
		{method: "mdm-master/group3/", coincidence: 1, atLeastOne: true},
	}
)

func TestCacheableMatcher_Match(t *testing.T) {
	matcher := NewCacheableMatcher(patterns)
	for _, c := range cases {
		res := matcher.Match(c.method)
		if c.coincidence != len(res) {
			t.Error(c)
		}
	}
}

func TestAtLeastOneMatcher_Match(t *testing.T) {
	matcher := NewAtLeastOneMatcher(patterns)
	for _, c := range cases {
		res := matcher.Match(c.method)
		if c.atLeastOne != res {
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
