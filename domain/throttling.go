package domain

import (
	"time"
)

type RateLimitResult struct {
	Allow      bool
	Remaining  int
	RetryAfter time.Duration
}
