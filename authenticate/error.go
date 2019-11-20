package authenticate

import (
	"fmt"
	"google.golang.org/grpc/codes"
)

func createError(msg string, status codes.Code, details ...interface{}) error {
	return ErrorDescription{message: msg, grpcStatus: status, details: details}
}

type ErrorDescription struct {
	message    string
	grpcStatus codes.Code
	details    []interface{}
}

func (e ErrorDescription) Error() string {
	return fmt.Sprintf("%v", e)
}

func (e ErrorDescription) Message() string {
	return e.message
}

func (e ErrorDescription) ConvertToGrpcStatus() codes.Code {
	return e.grpcStatus
}

func (e ErrorDescription) Details() []interface{} {
	return e.details
}
