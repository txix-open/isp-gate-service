package service

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"isp-gate-service/conf"
)

type DailyLimitRepo interface {
	Increment(ctx context.Context, key string, today time.Time) (int64, error)
}

type DailyLimit struct {
	repo   DailyLimitRepo
	limits map[int]int64
}

func NewDailyLimit(repo DailyLimitRepo, configs []conf.DailyLimit) DailyLimit {
	limits := make(map[int]int64)
	for _, config := range configs {
		limits[config.ApplicationId] = config.RequestsPerDay
	}
	return DailyLimit{
		repo:   repo,
		limits: limits,
	}
}

func (s DailyLimit) IncrementAndCheck(ctx context.Context, applicationId int) (bool, error) {
	limit, ok := s.limits[applicationId]
	if !ok {
		return true, nil
	}

	key := s.key(applicationId, time.Now())
	newValue, err := s.repo.Increment(ctx, key, time.Now())
	if err != nil {
		return false, errors.WithMessage(err, "increment")
	}

	return newValue <= limit, nil
}

func (s DailyLimit) key(applicationId int, today time.Time) string {
	y, m, d := today.Date()
	return fmt.Sprintf("isp-gate-service::daily-limit::%d:%d-%d-%d", applicationId, y, m, d)
}
