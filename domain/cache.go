package domain

import (
	"github.com/pkg/errors"
)

var (
	ErrAuthenticationCacheMiss = errors.New("authentication not found in cache")
)
