package domain

import (
	"github.com/txix-open/isp-kit/errors"
)

var (
	ErrAuthenticationCacheMiss = errors.New("authentication not found in cache")
)
