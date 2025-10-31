package httpx

import (
	"encoding/json"
	"net/http"
)

// JSON writes the provided payload as JSON with the supplied status code.
func JSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(payload)
}

// Error writes an error response with a standard envelope.
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, map[string]any{"error": message})
}
