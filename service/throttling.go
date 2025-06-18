package service

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"isp-gate-service/conf"
	"isp-gate-service/domain"
	"isp-gate-service/entity"
)

type LockRepo interface {
	IsAllowRequestPerSecond(ctx context.Context, key string, rate int) (*entity.RateLimiterResponse, error)
}

type Throttling struct {
	repo   LockRepo
	limits map[int]int
}

func NewThrottling(repo LockRepo, configs []conf.Throttling) Throttling {
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

	key := s.key(applicationId)
	result, err := s.repo.IsAllowRequestPerSecond(ctx, key, rate)
	if err != nil {
		return nil, errors.WithMessage(err, "is allow request per second")
	}

	return &domain.RateLimitResult{
		Allow:      result.Allow,
		Remaining:  result.Remaining,
		RetryAfter: result.RetryAfter,
	}, nil
}

func (s Throttling) key(applicationId int) string {
	return fmt.Sprintf("isp-gate-service::rate-limit::%d", applicationId)
}
