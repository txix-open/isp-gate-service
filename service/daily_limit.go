package service

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"isp-gate-service/conf"
)

type DailyLimitRepo interface {
	Increment(ctx context.Context, applicationId int, today time.Time) (int64, error)
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
	max, ok := s.limits[applicationId]
	if !ok {
		return true, nil
	}

	newValue, err := s.repo.Increment(ctx, applicationId, time.Now())
	if err != nil {
		return false, errors.WithMessage(err, "increment")
	}

	return newValue <= max, nil
}
