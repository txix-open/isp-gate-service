package domain

import (
	"github.com/pkg/errors"
)

const ServiceIsNotAvailableErrorMessage = "Service is not available now, please try later"

type Error struct {
	ErrorMessage string
	ErrorCode    string
	Details      []interface{}
}

var (
	ErrAuthorize    = errors.New("authorize failed")
	ErrAuthenticate = errors.New("authenticate failed")
)
