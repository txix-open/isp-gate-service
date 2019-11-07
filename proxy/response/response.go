package response

import (
	"encoding/json"
	"github.com/integration-system/isp-lib/http"
	"github.com/integration-system/isp-lib/structure"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc/codes"
	"isp-gate-service/domain"
)

const JsonContentType = "application/json; charset=utf-8"

var Option optionStruct

type (
	optionStruct struct{}
	option       func(ctx *fasthttp.RequestCtx, response *domain.ProxyResponse)
)

func Create(ctx *fasthttp.RequestCtx, options ...option) domain.ProxyResponse {
	response := domain.ProxyResponse{
		RequestBody:  ctx.Request.Body(),
		ResponseBody: ctx.Response.Body(),
	}
	for _, option := range options {
		option(ctx, &response)
	}
	return response
}

func (optionStruct) EmptyRequest() option {
	return func(ctx *fasthttp.RequestCtx, response *domain.ProxyResponse) {
		response.RequestBody = make([]byte, 0)
	}
}

func (optionStruct) EmptyResponse() option {
	return func(ctx *fasthttp.RequestCtx, response *domain.ProxyResponse) {
		response.ResponseBody = make([]byte, 0)
	}
}

func (optionStruct) SetError(err error) option {
	return func(ctx *fasthttp.RequestCtx, response *domain.ProxyResponse) {
		response.Error = err
	}
}

func (optionStruct) SetAndSendError(msg string, code codes.Code, err error) option {
	return func(ctx *fasthttp.RequestCtx, response *domain.ProxyResponse) {
		response.Error = err

		grpcCode := code.String()

		structureError := structure.GrpcError{
			ErrorMessage: msg,
			ErrorCode:    grpcCode,
			Details:      []interface{}{err.Error()},
		}

		ctx.SetContentType(JsonContentType)
		ctx.SetStatusCode(http.CodeToHttpStatus(code))
		msg, _ := json.Marshal(structureError)
		_, _ = ctx.Write(msg)
	}
}
