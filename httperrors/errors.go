package httperrors

import (
	"net/http"

	"github.com/integration-system/isp-kit/json"
)

type HttpError struct {
	statusCode  int
	userMessage string
	details     []interface{}
	err         error
}

func New(statusCode int, userMessage string, internalError error) HttpError {
	return HttpError{
		statusCode:  statusCode,
		userMessage: userMessage,
		err:         internalError,
	}
}

func (e HttpError) Error() string {
	return e.err.Error()
}

func (e HttpError) WriteError(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.statusCode)
	data := map[string]interface{}{
		"errorCode":    http.StatusText(e.statusCode),
		"errorMessage": e.userMessage,
		"details":      e.details,
	}
	return json.NewEncoder(w).Encode(data)
}

func (e *HttpError) WithDetails(details ...interface{}) {
	e.details = details
}
