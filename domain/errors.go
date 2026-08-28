package domain

import "github.com/txix-open/isp-kit/errors"

var (
	ErrUserAuthSettingNotFound = errors.New("user auth setting not found")
	ErrEmptyUserToken          = errors.New("failed to extract user token")
	ErrInvalidUserToken        = errors.New("invalid user token")
)
