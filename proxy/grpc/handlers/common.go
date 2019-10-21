package handlers

import (
	"github.com/integration-system/isp-lib/backend"
	isp "github.com/integration-system/isp-lib/proto/stubs"
	s "github.com/integration-system/isp-lib/streaming"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"io"
	"isp-gate-service/log_code"
	"isp-gate-service/proxy/grpc/utils"
	"mime/multipart"
	"time"
)

const (
	headerKeyContentDisposition = "Content-Disposition"
	headerKeyContentType        = "Content-Type"
	headerKeyContentLength      = "Content-Length"
	headerKeyTransferEncoding   = "Transfer-Encoding"

	errorMsgInternal   = "Internal server error"
	errorMsgInvalidArg = "Not able to read request body"
)

func openStream(headers *fasthttp.RequestHeader, method string, timeout time.Duration, client *backend.RxGrpcClient) (
	isp.BackendService_RequestStreamClient, context.CancelFunc, error) {

	cli, err := client.Conn()
	if err != nil {
		return nil, nil, err
	}
	md, _ := utils.MakeMetadata(headers, method)
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	stream, err := cli.RequestStream(ctx)
	if err != nil {
		return nil, nil, err
	}
	return stream, cancel, nil
}

func transferFile(f multipart.File, stream isp.BackendService_RequestStreamClient,
	buffer []byte, ctx *fasthttp.RequestCtx) (bool, bool) {

	ok := true
	eof := false
	for {
		n, err := f.Read(buffer)
		if n > 0 {
			err = stream.Send(&isp.Message{Body: &isp.Message_BytesBody{buffer[:n]}})
			if ok, eof = checkError(err, ctx); !ok || eof {
				break
			}
		}
		if err != nil {
			if ok, eof = checkError(err, ctx); ok && eof {
				err = stream.Send(s.FileEnd())
				ok, eof = checkError(err, ctx)
			}
			break
		}
	}
	return ok, eof
}

func checkError(err error, ctx *fasthttp.RequestCtx) (bool, bool) {
	if err != nil {
		if err != io.EOF {
			utils.LogRequestHandlerError(log_code.TypeData.GetFile, "", err)
			utils.SendError(errorMsgInternal, codes.Internal, []interface{}{err.Error()}, ctx)
			return false, false
		}
		return true, true
	}
	return true, false
}
