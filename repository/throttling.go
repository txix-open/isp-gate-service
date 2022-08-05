package repository

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redis_rate/v9"
	"github.com/pkg/errors"
	"isp-gate-service/domain"
)

type Throttling struct {
	cli *redis_rate.Limiter
}

func NewThrottling(cli redis.UniversalClient) Throttling {
	return Throttling{
		cli: redis_rate.NewLimiter(cli),
	}
}

func (r Throttling) IsAllowRequestPerSecond(ctx context.Context, applicationId int, rate int) (*domain.RateLimitResult, error) {
	result, err := r.cli.Allow(ctx, r.key(applicationId), redis_rate.PerSecond(rate))
	if err != nil {
		return nil, errors.WithMessage(err, "redis_rate/Allow")
	}
	return &domain.RateLimitResult{
		Allow:      result.Allowed > 0,
		Remaining:  result.Remaining,
		RetryAfter: result.RetryAfter,
	}, nil
}

func (r Throttling) key(applicationId int) string {
	return fmt.Sprintf("throttling:%d", applicationId)
}
