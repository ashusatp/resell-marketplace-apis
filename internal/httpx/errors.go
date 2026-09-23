package httpx

import (
	"encoding/json"
	"net/http"
)

type Code string

const (
	CodeInvalidID          Code = "invalid_id"          // 400 http.StatusBadRequest
	CodeInternalError      Code = "internal_error"      // 500 http.StatusInternalServerError
	CodeBadRequest         Code = "bad_request"         // 400 http.StatusBadRequest
	CodeNotFound           Code = "not_found"           // 404 http.StatusNotFound
	CodeUnauthorized       Code = "unauthorized"        // 401 http.StatusUnauthorized
	CodeForbidden          Code = "forbidden"           // 403 http.StatusForbidden
	CodeTooManyRequests    Code = "too_many_requests"   // 429 http.StatusTooManyRequests
	CodeServiceUnavailable Code = "service_unavailable" // 503 http.StatusServiceUnavailable
	CodeConflict           Code = "conflict"            // 409 http.StatusConflict
	CodeValidationFailed   Code = "validation_failed"   // 422 http.StatusUnprocessableEntity
	CodeMalformedJSON      Code = "malformed_json"      // 400 http.StatusBadRequest

)

type AppError struct {
	Error errorPayload `json:"error"`
}

type errorPayload struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

func Error(w http.ResponseWriter, statusCode int, message string, code Code) {

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(AppError{
		Error: errorPayload{
			Code:    code,
			Message: message,
		},
	})
}

func ValidationError(w http.ResponseWriter, statusCode int, message string, code Code, field string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(AppError{
		Error: errorPayload{
			Code:    code,
			Message: message,
			Field:   field,
		},
	})
}
