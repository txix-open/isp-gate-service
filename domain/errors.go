package domain

import "github.com/pkg/errors"

var (
	ErrEmptyUserToken = errors.New("failed to extract user token")
)
