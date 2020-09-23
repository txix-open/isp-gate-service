package authenticate

import (
	"strconv"
	"time"

	"github.com/integration-system/isp-lib/v2/config"
	"github.com/integration-system/isp-lib/v2/utils"
	log "github.com/integration-system/isp-log"
	"github.com/integration-system/isp-log/stdcodes"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc/codes"
	"isp-gate-service/conf"
	"isp-gate-service/log_code"
	"isp-gate-service/routing"
	"isp-gate-service/service/veritification"
)

const (
	// TODO move to isp-lib
	AdminTokenHeader = "x-auth-admin" //nolint

	awaitLengthVerifiableHeaders = 4
)

var notExpectedHeaders = []string{
	utils.SystemIdHeader,
	utils.DomainIdHeader,
	utils.ServiceIdHeader,
	utils.ApplicationIdHeader,
	utils.UserIdHeader,
	utils.DeviceIdHeader,
}

var auth = authenticate{}

type (
	authenticate struct {
		verify veritification.Verify
	}

	verifiable struct {
		appId   int32
		ctx     *fasthttp.RequestCtx
		path    string
		headers map[string]string
	}
)

func ReceiveConfiguration(conf conf.Cache) {
	if conf.EnableCache {
		if timeout, err := time.ParseDuration(conf.EvictTimeout); err != nil {
			log.Fatalf(stdcodes.ModuleInvalidRemoteConfig, "invalid timeout '%s'", timeout)
		} else {
			auth.verify = veritification.NewCacheablesVerify(timeout)
		}
	} else {
		auth.verify = veritification.NewRuntimeVerify()
	}
}

func Do(ctx *fasthttp.RequestCtx, path string) (int32, error) {
	for _, notExpectedHeader := range notExpectedHeaders {
		ctx.Request.Header.Del(notExpectedHeader)
	}

	verifiable := newVerifiable(ctx, path)
	err := verifiable.verifyAppToken()
	if err != nil {
		return 0, err
	}

	err = verifiable.verifyUserToken()
	if err != nil {
		return 0, err
	}

	verifiable.setHeaders()

	err = verifiable.checkInnerMethods()
	if err != nil {
		return 0, err
	}

	err = verifiable.checkUserMethods()
	if err != nil {
		return 0, err
	}

	//TODO backward capability for isp-config-service 1.x.x
	uuid := config.Get().(*conf.Configuration).InstanceUuid
	ctx.Request.Header.Set(utils.InstanceIdHeader, uuid)

	return verifiable.appId, nil
}

func newVerifiable(ctx *fasthttp.RequestCtx, path string) *verifiable {
	return &verifiable{
		ctx:     ctx,
		path:    path,
		appId:   -1,
		headers: make(map[string]string),
	}
}

func (v *verifiable) verifyAppToken() error {
	appToken := v.getParam(utils.ApplicationTokenHeader)
	var err error

	if len(appToken) == 0 {
		return createError("unauthorized", codes.Unauthenticated, "application token required")
	} else if config.GetRemote().(*conf.RemoteConfig).TokensSetting.ApplicationVerify {
		if v.appId, err = verifyToken.Application(string(appToken)); err != nil || v.appId == 0 {
			return createError("unauthorized", codes.Unauthenticated, "received invalid application token")
		}
	}

	verifiableHeaders, err := auth.verify.ApplicationToken(string(appToken))
	if err != nil {
		log.Error(log_code.ErrorAuthenticate, err)
		return createError("internal Server error", codes.Internal)
	}
	if len(verifiableHeaders) != awaitLengthVerifiableHeaders {
		return createError("unauthorized", codes.Unauthenticated, "received unexpected identities")
	}

	applicationId, err := strconv.ParseInt(verifiableHeaders[utils.ApplicationIdHeader], 10, 32)
	if err != nil {
		log.Error(log_code.ErrorAuthenticate, errors.WithMessagef(err, "parse appId from redis"))
		return createError("internal Server error", codes.Internal)
	}
	//nolint
	if v.appId != -1 && int32(applicationId) != v.appId {
		return createError("unauthorized", codes.Unauthenticated, "received unexpected application identity")
	}

	v.appId = int32(applicationId)
	v.headers = verifiableHeaders
	return nil
}

func (v *verifiable) verifyUserToken() error {
	userToken := string(v.getParam(utils.UserTokenHeader))
	if userToken != "" {
		userId, err := verifyToken.User(userToken)
		if err != nil {
			return createError("unauthorized", codes.Unauthenticated, "received invalid user token")
		}
		v.headers[utils.UserIdHeader] = userId
	}
	v.headers[utils.UserTokenHeader] = userToken

	v.headers[utils.DeviceTokenHeader] = string(v.getParam(utils.DeviceTokenHeader))

	var err error
	v.headers, err = auth.verify.Identity(v.headers, v.path)
	if err != nil {
		switch err {
		case veritification.ErrorInvalidUserId:
			return createError("unauthorized", codes.Unauthenticated, err.Error())
		case veritification.ErrorPermittedToCallUser, veritification.ErrorPermittedToCallApplication:
			return createError("forbidden", codes.PermissionDenied, err.Error())
		default:
			return createError("internal server error", codes.Internal)
		}
	}
	return nil
}

func (v *verifiable) setHeaders() {
	for key, value := range v.headers {
		if value != "" {
			v.ctx.Request.Header.Set(key, value)
		}
	}
}

func (v *verifiable) checkInnerMethods() error {
	_, ok := routing.InnerMethods[v.path]
	if ok {
		adminToken := string(v.getParam(AdminTokenHeader))
		if adminToken == "" {
			return createError("unauthorized", codes.Unauthenticated, "admin token required")
		}
		if verifyToken.Admin(adminToken) != nil {
			return createError("unauthorized", codes.Unauthenticated, "received invalid admin token")
		}
	}
	return nil
}

func (v *verifiable) checkUserMethods() error {
	if v.headers[utils.UserIdHeader] == "" {
		if _, ok := routing.AuthUserMethods[v.path]; ok {
			return createError("unauthorized", codes.Unauthenticated, "user token required")
		}
	}
	return nil
}

func (v *verifiable) getParam(key string) []byte {
	val := v.ctx.Request.Header.Peek(key)
	if len(val) != 0 {
		return val
	}
	return v.ctx.Request.URI().QueryArgs().Peek(key)
}
