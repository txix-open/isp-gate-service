package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
)

type DailyLimit struct {
	cli redis.UniversalClient
}

func NewDailyLimit(cli redis.UniversalClient) DailyLimit {
	return DailyLimit{
		cli: cli,
	}
}

func (r DailyLimit) Increment(ctx context.Context, applicationId int, today time.Time) (int64, error) {
	key := r.key(applicationId, today)
	value, err := r.cli.Incr(ctx, key).Result()
	if err != nil {
		return 0, errors.WithMessage(err, "incr")
	}

	if value == 1 {
		err := r.cli.ExpireNX(ctx, key, 24*time.Hour).Err() //nolint:mnd
		if err != nil {
			return 0, errors.WithMessage(err, "expire nx")
		}
	}

	return value, nil
}

func (r DailyLimit) key(applicationId int, today time.Time) string {
	y, m, d := today.Date()
	return fmt.Sprintf("daily_limit:%d:%d-%d-%d", applicationId, y, m, d)
}
