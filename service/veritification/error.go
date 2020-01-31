package veritification

import "errors"

var (
	ErrorInvalidUserId              = errors.New("received unexpected user identity")
	ErrorPermittedToCallApplication = errors.New("application has no rights to call this method")
	ErrorPermittedToCallUser        = errors.New("user has no rights to call this method")
)
