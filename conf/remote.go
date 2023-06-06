package conf

import (
	"reflect"

	"github.com/integration-system/isp-kit/log"
	"github.com/integration-system/isp-kit/rc/schema"
	"github.com/integration-system/jsonschema"
	"github.com/pkg/errors"
)

func init() {
	schema.CustomGenerators.Register("logLevel", func(field reflect.StructField, t *jsonschema.Type) {
		t.Type = "string"
		t.Enum = []interface{}{"debug", "info", "error", "fatal"}
	})
}

type Remote struct {
	Redis       *Redis       `schema:"Настройки Redis,обязательно, если используется механизм суточных ограничений или ограничений пропускной способности"`
	Http        Http         `schema:"Настройки HTTP"`
	Logging     Logging      `schema:"Настройки логирования"`
	Caching     Caching      `schema:"Настройки кеширования"`
	DailyLimits []DailyLimit `schema:"Настройки суточных ограничений,сбрасываются раз в сутки в 00:00"`
	Throttling  []Throttling `schema:"Настройки пропускной способности"`
}

type Http struct {
	MaxRequestBodySizeInMb int64 `valid:"required" schema:"Максимальная длинна тела запроса,в мегабайтах"`
	ProxyTimeoutInSec      int   `valid:"required" schema:"Таймаут на проксирование,в секундах"`
}

type Logging struct {
	LogLevel         log.Level `schemaGen:"logLevel" schema:"Уровень логирования,логирование запросов осуществляется на уровне debug"`
	RequestLogEnable bool      `schema:"Включить логирование запросов"`
	BodyLogEnable    bool      `schema:"Включить логирование тел запросов и ответов,должно быть включено логирование запросов"`
	Skip             []string  `valid:"omitempty" schema:"регулярные выражения для отключения логирования"`
}

type Caching struct {
	AuthenticationDataInSec int `valid:"required" schema:"Время кеширования данных аутентификации,в секундах"`
	AuthorizationDataInSec  int `valid:"required" schema:"Время кеширования данных авторизации,в секундах"`
}

type DailyLimit struct {
	ApplicationId  int   `valid:"required" schema:"ID приложения"`
	RequestsPerDay int64 `valid:"required" schema:"Запросов в сутки"`
}

type Throttling struct {
	ApplicationId      int `valid:"required" schema:"ID приложения"`
	RequestsPerSeconds int `valid:"required,range(1|1000)" schema:"Запросов в секунду,не конфликтует с суточными ограничениями, алгоритм не работает на значениях больше 1000"`
}

type Redis struct {
	Address  string         `schema:"Адрес,обязателено, если sentinel не указан"`
	Username string         `schema:"Имя пользовтаеля"`
	Password string         `schema:"Пароль"`
	Sentinel *RedisSentinel `schema:"Настройки sentinel,обязательно, если address не указан"`
}

type RedisSentinel struct {
	Addresses  []string `valid:"required" schema:"Адреса нод в кластере"`
	MasterName string   `valid:"required" schema:"Имя мастера"`
	Username   string   `schema:"Имя пользовтаеля в sentinel"`
	Password   string   `schema:"Пароль в sentinel"`
}

func (r Remote) Validate() error {
	if (len(r.Throttling) > 0 || len(r.DailyLimits) > 0) && r.Redis == nil {
		return errors.New("redis is required if dailyLimits or throttling were specified")
	}
	if r.Redis != nil && r.Redis.Sentinel == nil && r.Redis.Address == "" {
		return errors.New("invalid redis config. sentinel or address are required")
	}
	return nil
}
