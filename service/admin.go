package service

import (
	"context"
	"isp-gate-service/domain"

	"github.com/pkg/errors"
)

type AdminAuth interface {
	Authenticate(ctx context.Context, token string) (*domain.AdminAuthenticateResponse, error)
}

type Admin struct {
	admin AdminAuth
}

func NewAdmin(adminAuth AdminAuth) Admin {
	return Admin{
		admin: adminAuth,
	}
}

func (s Admin) AdminAuthenticate(ctx context.Context, token string) (*domain.AdminAuthenticateResponse, error) {
	resp, err := s.admin.Authenticate(ctx, token)
	if err != nil {
		return nil, errors.WithMessage(err, "get admin token data from admin service")
	}
	return resp, nil
}
