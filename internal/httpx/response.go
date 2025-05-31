package httpx

import "net/http"

type Response[T any] struct {
	Success    bool   `json:"success"`
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
	Data       *T     `json:"data,omitempty"`
}

func OK[T any](msg string, data T) Response[T] {
	return Response[T]{true, http.StatusOK, msg, &data}
}
func Fail(msg string, code int) Response[struct{}] {
	return Response[struct{}]{false, code, msg, nil}
}
