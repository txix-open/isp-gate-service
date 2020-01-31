package veritification

const (
	ErrorCodeInvalidUserId   code = 0
	ErrorCodePermittedToCall code = 1
)

type (
	code int

	Error struct {
		Code    code
		message string
	}
)

func (e Error) Error() string {
	return e.message
}

func newError(code code, message string) Error {
	return Error{
		Code:    code,
		message: message,
	}
}
