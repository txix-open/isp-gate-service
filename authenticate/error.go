package authenticate

import (
	"fmt"
	"google.golang.org/grpc/codes"
)

func createError(status codes.Code, details ...interface{}) ErrorDescription {
	return ErrorDescription{grpcStatus: status, details: details}
}

type ErrorDescription struct {
	grpcStatus codes.Code
	details    []interface{}
}

func (e ErrorDescription) Error() string {
	return fmt.Sprintf("%v", e)
}

func (e ErrorDescription) ConvertToGrpcStatus() codes.Code {
	return e.grpcStatus
}

func (e ErrorDescription) Details() []interface{} {
	return e.details
}
