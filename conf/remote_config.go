//nolint
package conf

import (
	"time"

	"github.com/integration-system/isp-lib/v2/structure"
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
		Database          structure.DBConfiguration     `schema:"Настройка подключения к базе данных"`
		TokensSetting     TokensSetting                 `schema:"Настройка секретов"`
		HttpSetting       HttpSetting                   `schema:"Настройка сервера"`
		GrpcSetting       GrpcSetting                   `schema:"Настройка grpc соединения"`
		Metrics           structure.MetricConfiguration `schema:"Настройка метрик"`
		JournalSetting    Journal                       `schema:"Настройка журалирования"`
		Redis             structure.RedisConfiguration  `schema:"Настройка Redis" valid:"required~Required"`
		AccountingSetting Accounting                    `schema:"Настройка учета запросов"`
		AuthCacheSetting  Cache                         `schema:"Настройка кеширования данных аутентификации приложений" valid:"required~Required"`
	}

	Cache struct {
		EnableCache  bool   `schema:"Кеширование,включает кеширования для токена приложения"`
		EvictTimeout string `schema:"Время жизни записи в кеше"`
	}

	Journal struct {
		Journal         JorunalConfig `schema:"Настройка конфигурации"`
		MethodsPatterns []string      `schema:"Список методов для логирования,список строк вида: 'module/group/method'(* - для частичного совпадения). При обработке запроса, если вызываемый метод совпадает со строкой из списка, тела запроса и ответа записываются в лог"`
	}

	JorunalConfig struct {
		Enable    bool   `schema:"Включение/отключение журналирования"`
		Filename  string `schema:"Имя файла,путь до файла в который будут записываться логи"`
		MaxSizeMb int    `schema:"Максимальный размер файла,ограничение по размеру файла после достижения которого логи будут записываться в новый файл"`
		Compress  bool   `schema:"Сжатие логов,архивирует файлы в gzip"`
	}

	TokensSetting struct {
		AdminSecret       string `schema:"Секрет для проверки токена администратора"`
		ApplicationSecret string `schema:"Секрет для проверки токена приложений"`
		ApplicationVerify bool   `schema:"Проверка подписи токена приложений"`
		UserSecret        string `schema:"Секрет для проверки токена пользователя"`
	}

	Accounting struct {
		Enable          bool                `schema:"Учет запросов,включает/отключает учет запросов"`
		SnapshotTimeout string              `schema:"Частота выгрузки ограничений для приложений" valid:"required~Required"`
		Setting         []AccountingSetting `schema:"Настройка учета для приложений"`
		Storing         StoringSetting      `schema:"Настройка хранения всех запросов"`
	}

	AccountingSetting struct {
		ApplicationId int32          `valid:"required~Required" schema:"Идентификатор приложения"`
		EnableStoring bool           `schema:"Хранение запросов,если включено, каждый факт обработки запроса будет фиксироваться в БД"`
		Limits        []LimitSetting `schema:"Настройка ограничений на запросы"`
	}

	LimitSetting struct {
		Pattern  string `valid:"required~Required" schema:"Шаблон пути,указывается путь для которого будут применяется ограничения; поддерживается '/*' для неполного совпадения"`
		MaxCount int    `schema:"Количество запросов"`
		Timeout  string `schema:"Время жизни одного запроса"`
	}

	StoringSetting struct {
		Size    int    `schema:"Размер буфера,размер буфера,при достижении которого данные сбрасываются в БД" valid:"required,range(1|1000000)"`
		Timeout string `schema:"Частота выгрузки всех учтенных запросов,периодичность сброса данных в БД" valid:"required~Required"`
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
