package proxy

import (
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/integration-system/isp-kit/grpc"
	"github.com/integration-system/isp-kit/grpc/client"
	"github.com/integration-system/isp-kit/grpc/isp"
	"github.com/integration-system/isp-kit/json"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"isp-gate-service/domain"
	"isp-gate-service/middleware"
)

const (
	adminIdHeader = "x-auth-admin"
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
	cli *client.Client
}

func NewGrpc(cli *client.Client) Grpc {
	return Grpc{
		cli: cli,
	}
}

func (p Grpc) Handle(ctx *middleware.Context) error {
	body, err := ioutil.ReadAll(ctx.Request.Body)
	if err != nil {
		return errors.WithMessage(err, "read body")
	}

	md := make(metadata.MD)
	md[grpc.ProxyMethodNameHeader] = []string{ctx.Path}
	md[grpc.ApplicationIdHeader] = []string{strconv.Itoa(ctx.AppId)}
	md[adminIdHeader] = []string{strconv.Itoa(ctx.AdminId)}
	requestContext := metadata.NewOutgoingContext(ctx.Request.Context(), md)

	cli := p.cli.BackendClient()

	var resultBody []byte
	var resultStatus int
	result, err := cli.Request(requestContext, &isp.Message{
		Body: &isp.Message_BytesBody{BytesBody: body},
	})
	if err != nil {
		resultBody, resultStatus = p.convertRequestErrorToBodyWithStatus(err)
	} else {
		resultBody = result.GetBytesBody()
		resultStatus = http.StatusOK
	}

	ctx.ResponseWriter.Header().Set("Content-Type", "application/json; charset=utf-8")
	ctx.ResponseWriter.WriteHeader(resultStatus)
	_, err = ctx.ResponseWriter.Write(resultBody)
	if err != nil {
		return errors.WithMessage(err, "response write")
	}

	return nil
}

func (p Grpc) convertRequestErrorToBodyWithStatus(err error) ([]byte, int) {
	s, ok := status.FromError(err)
	if !ok {
		return []byte(domain.ServiceIsNotAvailableErrorMessage), http.StatusServiceUnavailable
	}

	details := s.Details()
	response := make([]byte, 0)
	for _, detail := range details {
		switch typeOfDetail := detail.(type) {
		case *structpb.Struct:
			result, err := typeOfDetail.MarshalJSON()
			if err != nil {
				return []byte(domain.ServiceIsNotAvailableErrorMessage), http.StatusServiceUnavailable
			}
			response = result
		case *isp.Message:
			listBody := typeOfDetail.GetListBody()
			if listBody != nil {
				result, err := listBody.MarshalJSON()
				if err != nil {
					return []byte(domain.ServiceIsNotAvailableErrorMessage), http.StatusServiceUnavailable
				}
				response = result

				break
			}

			result := typeOfDetail.GetBytesBody()
			response = result
		default:
			result, err := json.Marshal(typeOfDetail)
			if err != nil {
				return []byte(domain.ServiceIsNotAvailableErrorMessage), http.StatusServiceUnavailable
			}
			response = result
		}

		break
	}

	if len(response) == 0 {
		detail := domain.Error{
			ErrorMessage: s.Message(),
			ErrorCode:    s.Code().String(),
		}

		result, err := json.Marshal(detail)
		if err != nil {
			return []byte(domain.ServiceIsNotAvailableErrorMessage), http.StatusServiceUnavailable
		}
		response = result
	}

	return response, p.codeToHttpStatus(s.Code())
}

func (p Grpc) codeToHttpStatus(code codes.Code) int {
	s, ok := inverseCodeMap[code]
	if !ok {
		return http.StatusInternalServerError
	}

	return s
}
