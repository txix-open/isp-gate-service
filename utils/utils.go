package utils

import (
	"encoding/json"
	"github.com/integration-system/isp-lib/v2/http"
	"github.com/integration-system/isp-lib/v2/structure"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc/codes"
)

const JsonContentType = "application/json; charset=utf-8"

func WriteError(ctx *fasthttp.RequestCtx, message string, code codes.Code, details []interface{}) {
	grpcCode := code.String()

	structureError := structure.GrpcError{
		ErrorMessage: message,
		ErrorCode:    grpcCode,
		Details:      details,
	}

	ctx.SetContentType(JsonContentType)
	ctx.SetStatusCode(http.CodeToHttpStatus(code))
	msg, _ := json.Marshal(structureError)
	_, _ = ctx.Write(msg)
}
