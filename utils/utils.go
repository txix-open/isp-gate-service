package utils

import (
	"encoding/json"
	"github.com/integration-system/isp-lib/http"
	"github.com/integration-system/isp-lib/structure"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc/codes"
)

const (
	JsonContentType = "application/json; charset=utf-8"
)

func SendError(errorMessage string, errorCode codes.Code, details []interface{}, ctx *fasthttp.RequestCtx) {
	grpcCode := errorCode.String()

	structureError := structure.GrpcError{
		ErrorMessage: errorMessage,
		ErrorCode:    grpcCode,
		Details:      details,
	}

	ctx.SetContentType(JsonContentType)
	ctx.SetStatusCode(http.CodeToHttpStatus(errorCode))
	msg, _ := json.Marshal(structureError)
	_, _ = ctx.Write(msg)
}
