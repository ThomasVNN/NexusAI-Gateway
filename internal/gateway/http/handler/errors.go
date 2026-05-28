package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
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

type StructuredLog struct {
	Time          string `json:"time"`
	Level         string `json:"level"`
	Message       string `json:"msg"`
	Service       string `json:"service"`
	CorrelationID string `json:"correlation_id,omitempty"`
	Event         string `json:"event,omitempty"`
	Reason        string `json:"reason,omitempty"`
	Path          string `json:"path,omitempty"`
	Method        string `json:"method,omitempty"`
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
	if r != nil {
		corrID = r.Header.Get("X-Correlation-ID")
	}

	logEntry := StructuredLog{
		Time:          time.Now().UTC().Format(time.RFC3339),
		Level:         level,
		Message:       msg,
		Service:       "nexusai-gateway",
		CorrelationID: corrID,
		Event:         event,
		Reason:        reason,
	}
	if r != nil {
		logEntry.Path = r.URL.Path
		logEntry.Method = r.Method
	}

	jsonBytes, err := json.Marshal(logEntry)
	if err == nil {
		log.Println(string(jsonBytes))
	} else {
		log.Printf("Error marshal log: %v", err)
	}
}
