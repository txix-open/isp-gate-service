package authenticate

import (
	"fmt"
	"google.golang.org/grpc/codes"
)

var Error errorHelper

type errorHelper struct{}

func (errorHelper) create(status codes.Code) ErrorDescription {
	return ErrorDescription{grpcStatus: status}
}

type ErrorDescription struct {
	grpcStatus codes.Code
}

func (e ErrorDescription) Error() string {
	return fmt.Sprintf("%v", e)
}

func (e ErrorDescription) ConvertToGrpcStatus() codes.Code {
	return e.grpcStatus
}
