// nolint:gochecknoinits,lll
package conf

import (
	"reflect"

	"github.com/txix-open/isp-kit/log"
	"github.com/txix-open/isp-kit/rc/schema"
	"github.com/txix-open/jsonschema"
)

func init() {
	schema.CustomGenerators.Register("logLevel", func(field reflect.StructField, t *jsonschema.Schema) {
		t.Type = "string"
		t.Enum = []interface{}{"debug", "info", "warn", "error", "fatal"}
	})
}

type Remote struct {
	Http                            Http         `schema:"Настройки HTTP"`
	Logging                         Logging      `schema:"Настройки логирования"`
	Caching                         Caching      `schema:"Настройки кеширования"`
	DailyLimits                     []DailyLimit `schema:"Настройки суточных ограничений,сбрасываются раз в сутки в 00:00"`
	Throttling                      []Throttling `schema:"Настройки пропускной способности"`
	EnableClientRequestIdForwarding bool         `schema:"Включить проброс requestId из заголовка запроса"`
}

type Http struct {
	MaxRequestBodySizeInMb int64 `validate:"required" schema:"Максимальная длинна тела запроса,в мегабайтах"`
	ProxyTimeoutInSec      int   `validate:"required" schema:"Таймаут на проксирование,в секундах"`
}

type Logging struct {
	LogLevel                        log.Level `schemaGen:"logLevel" schema:"Уровень логирования,логирование запросов осуществляется на уровне debug"`
	RequestLogEnable                bool      `schema:"Включить логирование запросов"`
	BodyLogEnable                   bool      `schema:"Включить логирование тел запросов и ответов,должно быть включено логирование запросов"`
	SkipBodyLoggingEndpointPrefixes []string  `schema:"регулярные выражения для отключения логирования"`
	EnableForceUnescapingUnicode    bool      `schema:"Включить перевод тел запроса из unicode в utf-8, должно быть включено логирование тел запросов и ответов"`
}

type Caching struct {
	AuthenticationDataInSec int `validate:"required" schema:"Время кеширования данных аутентификации,в секундах"`
	AuthorizationDataInSec  int `validate:"required" schema:"Время кеширования данных авторизации,в секундах"`
}

type DailyLimit struct {
	ApplicationId  int   `validate:"required" schema:"ID приложения"`
	RequestsPerDay int64 `validate:"required" schema:"Запросов в сутки"`
}

type Throttling struct {
	ApplicationId      int `validate:"required" schema:"ID приложения"`
	RequestsPerSeconds int `validate:"required,min=1,max=1000" schema:"Запросов в секунду,не конфликтует с суточными ограничениями, алгоритм не работает на значениях больше 1000"`
}
