### v4.2.0
* новый параметр в remote_config: `skipBodyLoggingEndpointPrefixes`. Массив re. Если они есть в пути, то запрос не будет логироваться
### v4.1.2
* исправлена опция логирования тела запроса (при настройке ws могла отключаться)
### v4.1.1
* исправлена маршрутизация запросов
### v4.1.0
* обновлены зависимости
* добавлена поддержка проксирования WebSocket `ws` 
### v4.0.0
* реализована аутентификация токена администратора через msp-admin-service
### v3.0.3
* исправлено тело ответа grpc ошибок валидации
* обновлены зависимости
### v3.0.2
* добавлен charset=utf-8 в заголовок ответа для grpc
### v3.0.1
* исправлено проксирование http с GET параметрами
* исправлено логирование пути при проксировании HTTP
### v3.0.0
* реализация новой схемы аутентификации/авторизации
### v2.8.3
* use isp-kit logger instead local logger
* use log string instead log any (common log type)
* add replace grpc (isp-lib and isp-kit conflict)
### v2.8.2
* add file rotation
### v2.8.1
* add xml format support for new logger
### v2.8.0
* add new logger
* remove journal
* add identities to requests log
### v2.7.6
* updated dependencies
* migrated to common local config
### v2.7.5
* updated dependencies
### v2.7.4
* updated isp-lib
* updated isp-lib-test
### v2.7.3
* updated isp-lib
* updated isp-event-lib
## 2.7.2
* fix accounting
## 2.7.1
* fix default local config
## 2.7.0
* add support Redis Sentinel
* update grpc client
* code cleanup
## 2.6.3
* fix search from locations
* fix proxying URI in http and ws
## 2.6.2
* fix panic if journal enable and error logging
* add content-type, content-length header to response
* close journal
* journal module is not more required
* add missing journaling
## 2.6.0
* migrate to go mod
## 2.5.4
* fix auth cache
## 2.5.3
* fix grpc multipart error response
## 2.5.2
* fix init http
## 2.5.1
* fix code response when doesn't match user id
* fix config reload
## 2.5.0
* add skip for check exist method 
* add handler for user methods
* fix user id replacing
## 2.4.0
* add access checkout by user id
## 2.3.0
* add support for tokens in get params
## 2.2.0
* add websocket proxy
## 2.1.0
* fix proxy path
* add skip authenticate
* add snapshot version
## 2.0.0
* add integration with isp-config-service 2.0
## 1.3.0
* add unload requests
## 1.2.0
* add snapshot account
## 1.1.0
* add token verification via JWT
* add accounting for applications
* add optional cache for application tokens
