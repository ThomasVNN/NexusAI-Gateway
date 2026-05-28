package router

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// responseWriterWrapper wraps standard http.ResponseWriter to capture status code
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriterWrapper) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	return rw.ResponseWriter.Write(b)
}

// generateCorrelationID generates a secure random correlation ID string
func generateCorrelationID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "fallback-correlation-id"
	}
	return hex.EncodeToString(bytes)
}

// WithCorrelationID injects a unique request tracer header if not present
func WithCorrelationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corrID := r.Header.Get("X-Correlation-ID")
		if corrID == "" {
			corrID = generateCorrelationID()
		}
		
		// Ingress tracing setup
		r.Header.Set("X-Correlation-ID", corrID)
		w.Header().Set("X-Correlation-ID", corrID)
		
		next.ServeHTTP(w, r)
	})
}

// StructuredLog represents the JSON log layout for request logs
type StructuredLog struct {
	Time          string `json:"time"`
	Level         string `json:"level"`
	Message       string `json:"msg"`
	Service       string `json:"service"`
	Method        string `json:"method"`
	Path          string `json:"path"`
	CorrelationID string `json:"correlation_id"`
	StatusCode    int    `json:"status_code"`
	LatencyMS     int64  `json:"latency_ms"`
}

// WithStructuredLogging logs request endpoints, duration, and status codes in JSON format
func WithStructuredLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		corrID := r.Header.Get("X-Correlation-ID")
		
		wrappedWriter := &responseWriterWrapper{ResponseWriter: w}
		
		next.ServeHTTP(wrappedWriter, r)
		
		duration := time.Since(startTime).Milliseconds()
		
		logEntry := StructuredLog{
			Time:          time.Now().UTC().Format(time.RFC3339),
			Level:         "INFO",
			Message:       "Request completed",
			Service:       "nexusai-gateway",
			Method:        r.Method,
			Path:          r.URL.Path,
			CorrelationID: corrID,
			StatusCode:    wrappedWriter.statusCode,
			LatencyMS:     duration,
		}
		
		jsonBytes, err := json.Marshal(logEntry)
		if err == nil {
			log.Println(string(jsonBytes))
		} else {
			log.Printf("Error logging request context: %v", err)
		}
	})
}
