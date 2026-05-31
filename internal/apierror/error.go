package apierror

import (
	"encoding/json"
	"net/http"
)

// ErrorDetail represents a detailed error field
type ErrorDetail struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

// ErrorPayload represents the error structure in API responses
type ErrorPayload struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details []ErrorDetail `json:"details"`
}

// APIErrorResponse represents a standard API error response
type APIErrorResponse struct {
	Success bool          `json:"success"`
	Data    interface{}   `json:"data"`
	Meta    interface{}   `json:"meta"`
	Error   *ErrorPayload `json:"error"`
}

// WriteError writes a standardized JSON error response
func WriteError(w http.ResponseWriter, statusCode int, errorCode string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	resp := APIErrorResponse{
		Success: false,
		Data:    nil,
		Meta:    map[string]interface{}{},
		Error: &ErrorPayload{
			Code:    errorCode,
			Message: message,
			Details: []ErrorDetail{},
		},
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// WriteErrorWithDetails writes a JSON error response with additional details
func WriteErrorWithDetails(w http.ResponseWriter, statusCode int, errorCode string, message string, details []ErrorDetail) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	resp := APIErrorResponse{
		Success: false,
		Data:    nil,
		Meta:    map[string]interface{}{},
		Error: &ErrorPayload{
			Code:    errorCode,
			Message: message,
			Details: details,
		},
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// NewErrorPayload creates a new error payload
func NewErrorPayload(code, message string) *ErrorPayload {
	return &ErrorPayload{
		Code:    code,
		Message: message,
		Details: []ErrorDetail{},
	}
}

// NewErrorDetail creates a new error detail
func NewErrorDetail(field, message string) ErrorDetail {
	return ErrorDetail{
		Field:   field,
		Message: message,
	}
}
