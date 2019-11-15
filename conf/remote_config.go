package conf

import (
	"github.com/integration-system/isp-journal/rx"
	"github.com/integration-system/isp-lib/structure"
	"time"
)

const (
	KB = int64(1024)
	MB = int64(1 << 20)

	defaultSyncTimeout   = 30 * time.Second
	defaultStreamTimeout = 60 * time.Second

	defaultBufferSize          = 4 * KB
	defaultMaxRequestBodySize  = 512 * MB
	DefaultMaxResponseBodySize = 32 * MB
)

type (
	RemoteConfig struct {
		Secrets           SecretSetting                 `schema:"Настройка секретов"`
		ServerSetting     HttpSetting                   `schema:"Настройка сервера"`
		GrpcSetting       GrpcSetting                   `schema:"Настройка grpc соединения"`
		Metrics           structure.MetricConfiguration `schema:"Настройка метрик"`
		JournalSetting    Journal                       `schema:"Настройка журалирования"`
		Redis             structure.RedisConfiguration  `schema:"Настройка Redis" valid:"required~Required"`
		AccountingSetting Accounting                    `schema:"Настройка учета запросов"`
		AuthCacheSetting  Cache                         `schema:"Настройка кеширования данных аутентификации приложений" valid:"required~Required"`
	}

	Cache struct {
		EnableCash   bool   `schema:"Кеширование,включает кеширования для токена приложения"`
		EvictTimeout string `schema:"Время жизни записи в кеше"`
	}

	Journal struct {
		Journal         rx.Config `schema:"Настройка конфигурации"`
		MethodsPatterns []string  `schema:"Список методов для логирования,список строк вида: 'module/group/method'(* - для частичного совпадения). При обработке запроса, если вызываемый метод совпадает со строкой из списка, тела запроса и ответа записываются в лог"`
	}

	SecretSetting struct {
		Admin          string `schema:"Секрет для проверки токена администратора"`
		Application    string `schema:"Секрет для проверки токена приложений"`
		VerifyAppToken bool   `schema:"Проверка подписи токена приложений"`
	}

	Accounting struct {
		Enable  bool                `schema:"Статус работы учета,включает/отключает учет запросов"`
		Setting []AccountingSetting `schema:"Настройка учета для приложений"`
	}

	AccountingSetting struct {
		ApplicationId int32          `valid:"required~Required" schema:"Идентификатор приложения"`
		Limits        []LimitSetting `schema:"Настройка ограничений на запросы"`
	}

	LimitSetting struct {
		Pattern  string `valid:"required~Required" schema:"Шаблон пути,указывается путь для которого будут применяется ограничения; поддерживается '/*' для неполного совпадения"`
		MaxCount int    `schema:"Количество запросов"`
		Timeout  string `schema:"Время жизни одного запроса"`
	}

	HttpSetting struct {
		MaxRequestBodySizeBytes int64 `schema:"Максимальный размер тела запроса,в байтайх, по умолчанию: 512 MB"`
	}

	GrpcSetting struct {
		EnableOriginalProtoErrors            bool  `schema:"Проксирование ошибок в протобаф,включение/отключение проксирования, по умолчанию отключено"`
		ProxyGrpcErrorDetails                bool  `schema:"Проксирование первого элемента из details GRPC ошибки,включение/отключение проксирования, по умолчанию отключено"`
		MultipartDataTransferBufferSizeBytes int64 `schema:"Размер буфера для передачи бинарных файлов,по умолчанию 4 KB"`
		SyncInvokeMethodTimeoutMs            int64 `schema:"Время ожидания вызова метода,значение в миллисекундах, по умолчанию: 30000"`
		StreamInvokeMethodTimeoutMs          int64 `schema:"Время ожидания передачи и обработки файла,значение в миллисекундах, по умолчанию: 60000"`
	}
)

func (cfg GrpcSetting) GetSyncInvokeTimeout() time.Duration {
	if cfg.SyncInvokeMethodTimeoutMs <= 0 {
		return defaultSyncTimeout
	}
	return time.Duration(cfg.SyncInvokeMethodTimeoutMs) * time.Millisecond
}

func (cfg GrpcSetting) GetStreamInvokeTimeout() time.Duration {
	if cfg.StreamInvokeMethodTimeoutMs <= 0 {
		return defaultStreamTimeout
	}
	return time.Duration(cfg.StreamInvokeMethodTimeoutMs) * time.Millisecond
}

func (cfg GrpcSetting) GetTransferFileBufferSize() int64 {
	if cfg.MultipartDataTransferBufferSizeBytes <= 0 {
		return defaultBufferSize
	}
	return cfg.MultipartDataTransferBufferSizeBytes
}

func (cfg HttpSetting) GetMaxRequestBodySize() int64 {
	if cfg.MaxRequestBodySizeBytes <= 0 {
		return defaultMaxRequestBodySize
	}
	return cfg.MaxRequestBodySizeBytes
}
