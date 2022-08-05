package service

import (
	"context"

	"github.com/pkg/errors"
	"isp-gate-service/conf"
	"isp-gate-service/domain"
)

type ThrottlingRepo interface {
	IsAllowRequestPerSecond(ctx context.Context, applicationId int, rate int) (*domain.RateLimitResult, error)
}

type Throttling struct {
	repo   ThrottlingRepo
	limits map[int]int
}

func NewThrottling(repo ThrottlingRepo, configs []conf.Throttling) Throttling {
	limits := make(map[int]int)
	for _, config := range configs {
		limits[config.ApplicationId] = config.RequestsPerSeconds
	}
	return Throttling{
		repo:   repo,
		limits: limits,
	}
}

func (s Throttling) AllowRateLimit(ctx context.Context, applicationId int) (*domain.RateLimitResult, error) {
	rate, ok := s.limits[applicationId]
	if !ok {
		return &domain.RateLimitResult{
			Allow:      true,
			Remaining:  -1,
			RetryAfter: -1,
		}, nil
	}

	result, err := s.repo.IsAllowRequestPerSecond(ctx, applicationId, rate)
	if err != nil {
		return nil, errors.WithMessage(err, "is allow request per second")
	}

	return result, nil
}
