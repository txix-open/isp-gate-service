// nolint:gochecknoinits
package proxy

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"time"

	"isp-gate-service/httperrors"
	"isp-gate-service/request"

	"github.com/pkg/errors"
	"github.com/txix-open/isp-kit/grpc"
	"github.com/txix-open/isp-kit/grpc/client"
	"github.com/txix-open/isp-kit/grpc/isp"
	"github.com/txix-open/isp-kit/json"
	"github.com/txix-open/isp-kit/requestid"
	_ "google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func init() {
	for httpCode, grpcCode := range codeMap {
		inverseCodeMap[grpcCode] = httpCode
	}
}

var (
	codeMap = map[int]codes.Code{
		http.StatusOK:                  codes.OK,
		http.StatusRequestTimeout:      codes.Canceled,
		http.StatusBadRequest:          codes.InvalidArgument,
		http.StatusGatewayTimeout:      codes.DeadlineExceeded,
		http.StatusNotFound:            codes.NotFound,
		http.StatusConflict:            codes.AlreadyExists,
		http.StatusForbidden:           codes.PermissionDenied,
		http.StatusUnauthorized:        codes.Unauthenticated,
		http.StatusTooManyRequests:     codes.ResourceExhausted,
		http.StatusPreconditionFailed:  codes.FailedPrecondition,
		http.StatusNotImplemented:      codes.Unimplemented,
		http.StatusInternalServerError: codes.Internal,
		http.StatusServiceUnavailable:  codes.Unavailable,
	}
	inverseCodeMap = map[codes.Code]int{}
)

type Grpc struct {
	cli      *client.Client
	skipAuth bool
	timeout  time.Duration
}

func NewGrpc(cli *client.Client, skipAuth bool, timeout time.Duration) Grpc {
	return Grpc{
		cli:      cli,
		skipAuth: skipAuth,
		timeout:  timeout,
	}
}

func (p Grpc) Handle(ctx *request.Context) error {
	body, err := io.ReadAll(ctx.Request().Body)
	if err != nil {
		return errors.WithMessage(err, "grpc: read body")
	}

	requestId := requestid.FromContext(ctx.Context())
	md := metadata.MD{
		grpc.ProxyMethodNameHeader: {ctx.Endpoint()},
		requestid.Header:           {requestId},
	}

	if !p.skipAuth {
		authData, err := ctx.GetAuthData()
		if err != nil {
			return errors.WithMessage(err, "get auth data")
		}
		md[grpc.SystemIdHeader] = []string{strconv.Itoa(authData.SystemId)}
		md[grpc.DomainIdHeader] = []string{strconv.Itoa(authData.DomainId)}
		md[grpc.ServiceIdHeader] = []string{strconv.Itoa(authData.ServiceId)}
		md[grpc.ApplicationIdHeader] = []string{strconv.Itoa(authData.ApplicationId)}
		if ctx.IsAdminAuthenticated() {
			md[xAdminIdHeader] = []string{strconv.Itoa(ctx.AdminId())}
		}
	}
	requestContext := metadata.NewOutgoingContext(ctx.Context(), md)

	requestContext, cancel := context.WithTimeout(requestContext, p.timeout)
	defer cancel()
	result, err := p.cli.BackendClient().Request(requestContext, &isp.Message{
		Body: &isp.Message_BytesBody{BytesBody: body},
	})
	if err != nil {
		return p.handleError(err, requestId, ctx.ResponseWriter())
	}

	return p.writeResponse(http.StatusOK, result.GetBytesBody(), requestid.FromContext(ctx.Context()), ctx.ResponseWriter())
}

func (p Grpc) handleError(err error, requestId string, w http.ResponseWriter) error {
	status, ok := status.FromError(err)
	if !ok {
		return httperrors.New(
			http.StatusServiceUnavailable,
			"upstream is not available",
			errors.WithMessage(err, "grpc proxy"),
		)
	}

	statusCode := p.codeToHttpStatus(status.Code())
	for _, detail := range status.Details() {
		switch typeOfDetail := detail.(type) {
		case *isp.Message:
			switch {
			case typeOfDetail.GetBytesBody() != nil:
			case typeOfDetail.GetListBody() != nil:
				return p.writeProto(statusCode, typeOfDetail.GetListBody(), requestId, w)
			case typeOfDetail.GetStructBody() != nil:
				return p.writeProto(statusCode, typeOfDetail.GetStructBody(), requestId, w)
			}
		default:
			return p.writeProto(statusCode, typeOfDetail, requestId, w)
		}
	}

	return httperrors.New(
		statusCode,
		status.Message(),
		status.Err(),
	)
}

func (p Grpc) writeProto(statusCode int, proto interface{}, requestId string, w http.ResponseWriter) error {
	data, err := json.Marshal(proto)
	if err != nil {
		return errors.WithMessage(err, "marshal grpc details to json")
	}
	return p.writeResponse(statusCode, data, requestId, w)
}

func (p Grpc) writeResponse(statusCode int, data []byte, requestId string, w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set(requestid.Header, requestId)
	w.WriteHeader(statusCode)
	_, err := w.Write(data)
	if err != nil {
		return errors.WithMessage(err, "response write")
	}
	return nil
}

func (p Grpc) codeToHttpStatus(code codes.Code) int {
	s, ok := inverseCodeMap[code]
	if !ok {
		return http.StatusInternalServerError
	}

	return s
}
