package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"context"
)

type ErrorDetail struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

type ErrorPayload struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details []ErrorDetail `json:"details"`
}

type APIErrorResponse struct {
	Success bool          `json:"success"`
	Data    interface{}   `json:"data"`
	Meta    interface{}   `json:"meta"`
	Error   *ErrorPayload `json:"error"`
}

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

func LogSecurityEvent(r *http.Request, level string, msg string, event string, reason string) {
	corrID := ""
	path := ""
	method := ""
	if r != nil {
		corrID = r.Header.Get("X-Correlation-ID")
		path = r.URL.Path
		method = r.Method
	}

	attrs := []slog.Attr{
		slog.String("service", "nexusai-gateway"),
		slog.String("event", event),
		slog.String("reason", reason),
	}
	if corrID != "" {
		attrs = append(attrs, slog.String("correlation_id", corrID))
	}
	if path != "" {
		attrs = append(attrs, slog.String("path", path), slog.String("method", method))
	}

	switch strings.ToUpper(level) {
	case "ERROR":
		slog.LogAttrs(context.Background(), slog.LevelError, msg, attrs...)
	case "WARN", "WARNING":
		slog.LogAttrs(context.Background(), slog.LevelWarn, msg, attrs...)
	default:
		slog.LogAttrs(context.Background(), slog.LevelInfo, msg, attrs...)
	}
}
