package repository

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/txix-open/isp-kit/grpc/client"
	"isp-gate-service/entity"
)

const (
	incrementEndpoint = "isp-lock-service/daily_limit/increment"
	rateLimitEndpoint = "isp-lock-service/rate_limit"
)

type Locker struct {
	cli *client.Client
}

func NewLocker(cli *client.Client) Locker {
	return Locker{
		cli: cli,
	}
}

func (r Locker) Increment(ctx context.Context, key string, today time.Time) (int64, error) {
	resp := new(entity.IncrementResponse)
	err := r.cli.Invoke(incrementEndpoint).
		JsonRequestBody(entity.IncrementRequest{
			Key:   key,
			Today: today,
		}).
		JsonResponseBody(resp).
		Do(ctx)
	switch {
	case err != nil:
		return 0, errors.WithMessagef(err, "invoke isp-lock-service: '%s'", incrementEndpoint)
	default:
		return int64(resp.Value), nil //nolint:gosec
	}
}

func (r Locker) IsAllowRequestPerSecond(ctx context.Context, key string, rate int) (*entity.RateLimiterResponse, error) {
	resp := new(entity.RateLimiterResponse)
	err := r.cli.Invoke(rateLimitEndpoint).
		JsonRequestBody(entity.RateLimiterRequest{
			Key:    key,
			MaxRps: rate,
		}).
		JsonResponseBody(resp).
		Do(ctx)
	if err != nil {
		return nil, errors.WithMessagef(err, "invoke isp-lock-service: '%s'", rateLimitEndpoint)
	}

	return resp, nil
}
