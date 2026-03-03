package response

import (
	"encoding/json"
	"net/http"
)

type Envelope[T any] struct {
	Success bool      `json:"success"`
	Data    *T        `json:"data"`
	Error   *string   `json:"error"`
	Meta    *PageMeta `json:"meta"`
}

type PageMeta struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
}

func JSON[T any](w http.ResponseWriter, status int, data *T, meta *PageMeta) {
	w.Header().Set("Content-Type", "application/json")
	payload := Envelope[T]{
		Success: status < http.StatusBadRequest,
		Data:    data,
		Error:   nil,
		Meta:    meta,
	}
	writeEnvelope(w, status, payload)
}

func Error(w http.ResponseWriter, status int, message string) {
	payload := Envelope[map[string]string]{
		Success: false,
		Data:    nil,
		Error:   &message,
		Meta:    nil,
	}
	writeEnvelope(w, status, payload)
}

func writeEnvelope[T any](w http.ResponseWriter, status int, payload Envelope[T]) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		w.Write([]byte(`{"success":false,"data":null,"error":"internal encoding error","meta":null}`))
	}
}
