package entity

import (
	"time"
)

type IncrementRequest struct {
	Key   string
	Today time.Time
}

type IncrementResponse struct {
	Value uint64
}

type RateLimiterRequest struct {
	Key    string
	MaxRps int
}

type RateLimiterResponse struct {
	Allow      bool
	Remaining  int
	RetryAfter time.Duration
}
