package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Details string `json:"details,omitempty"`
	Time    string `json:"time"`
}

// RecoveryMiddleware recovers from panics and returns a 500 error
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("PANIC: %v\n%s", err, debug.Stack())

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)

				response := ErrorResponse{
					Error:   "Internal server error",
					Code:    http.StatusInternalServerError,
					Details: "An unexpected error occurred",
					Time:    time.Now().Format(time.RFC3339),
				}

				json.NewEncoder(w).Encode(response)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// RequestLoggerMiddleware logs all requests
func RequestLoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Log request
		log.Printf("[%s] %s %s", r.Method, r.URL.Path, r.RemoteAddr)

		// Wrap response writer to capture status
		recorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(recorder, r)

		// Log response
		duration := time.Since(start)
		log.Printf("[%d] %s %s (%v)", recorder.statusCode, r.Method, r.URL.Path, duration)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// SecurityHeaders adds security headers
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		next.ServeHTTP(w, r)
	})
}

// ErrorHandler creates a JSON error response
func ErrorHandler(w http.ResponseWriter, err string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	response := ErrorResponse{
		Error: err,
		Code:  code,
		Time:  time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}
