package wacloudapitest

import (
	"net/http"
)

// InjectedError represents an injected error response in the chaos error queue.
type InjectedError struct {
	StatusCode int
	Code       int
	Message    string
}

// InjectRateLimit queues count HTTP 429 Rate Limit responses.
func (s *Server) InjectRateLimit(count int) {
	if count <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := 0; i < count; i++ {
		s.errorQueue = append(s.errorQueue, &InjectedError{
			StatusCode: http.StatusTooManyRequests,
			Code:       130429,
			Message:    "Rate limit hit",
		})
	}
}

// InjectServerError queues count HTTP 500 Internal Server Error responses.
func (s *Server) InjectServerError(count int) {
	if count <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := 0; i < count; i++ {
		s.errorQueue = append(s.errorQueue, &InjectedError{
			StatusCode: http.StatusInternalServerError,
			Code:       2,
			Message:    "Internal server error",
		})
	}
}

// InjectMetaError queues count Meta API error responses with the specified code and message.
func (s *Server) InjectMetaError(code int, message string, count int) {
	if count <= 0 {
		return
	}
	statusCode := http.StatusBadRequest
	switch {
	case code == 130429 || code == 80007 || code == 613:
		statusCode = http.StatusTooManyRequests
	case code == 190 || code == 102:
		statusCode = http.StatusUnauthorized
	case code == 1 || code == 2:
		statusCode = http.StatusInternalServerError
	case code >= 132000 && code <= 132999:
		statusCode = http.StatusBadRequest
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for i := 0; i < count; i++ {
		s.errorQueue = append(s.errorQueue, &InjectedError{
			StatusCode: statusCode,
			Code:       code,
			Message:    message,
		})
	}
}

// SetCustomHandler registers a custom HTTP handler for a specific route pattern.
func (s *Server) SetCustomHandler(pattern string, handler http.HandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.customHandlers == nil {
		s.customHandlers = make(map[string]http.HandlerFunc)
	}
	s.customHandlers[pattern] = handler
}
