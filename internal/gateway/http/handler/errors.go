package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/apierror"
)

type ErrorPayload = apierror.ErrorPayload
type ErrorDetail = apierror.ErrorDetail
type APIErrorResponse = apierror.APIErrorResponse

func WriteError(w http.ResponseWriter, statusCode int, errorCode string, message string) {
	apierror.WriteError(w, statusCode, errorCode, message)
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
