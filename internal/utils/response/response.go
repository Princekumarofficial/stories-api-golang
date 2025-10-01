package response

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
)

type Response struct {
	Status  string      `json:"status"`
	Error   string      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

const (
	StatusSuccess = "success"
	StatusError   = "error"
)

func WriteJSON(w http.ResponseWriter, status int, data interface{}) error {

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	return json.NewEncoder(w).Encode(data)
}

func GeneralError(err error) Response {
	return Response{
		Status: StatusError,
		Error:  err.Error(),
	}
}

func ValidationError(errs validator.ValidationErrors) Response {
	var errorMessages string
	for _, err := range errs {
		errorMessages += err.Field() + ": " + err.Tag() + "; "
	}

	return Response{
		Status: StatusError,
		Error:  errorMessages,
	}
}

func RequestOK(message string, data interface{}) Response {
	return Response{
		Status:  StatusSuccess,
		Message: message,
		Data:    data,
	}
}
