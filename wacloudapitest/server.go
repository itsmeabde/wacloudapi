package wacloudapitest

import (
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/itsmeabde/wacloudapi"
)

const (
	DefaultPhoneNumberID = "10001"
	DefaultWABAID        = "20002"
	DefaultAPIVersion    = "v21.0"
	DefaultAppSecret     = "test_app_secret"
	DefaultAccessToken   = "mock_access_token"
)

// inMemoryTransport executes HTTP requests directly against an http.Handler
// without requiring local TCP socket roundtrips.
type inMemoryTransport struct {
	handler http.Handler
}

func (t *inMemoryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	t.handler.ServeHTTP(rec, req)
	return rec.Result(), nil
}

// Server is an in-memory Meta WhatsApp Cloud API mock test server.
type Server struct {
	mu            sync.RWMutex
	httpServer    *httptest.Server
	httpClient    *http.Client
	mux           *http.ServeMux
	phoneNumberID string
	wabaID        string
	apiVersion    string
	appSecret     string
	accessToken   string
	state         *State

	customHandlers map[string]http.HandlerFunc
}

// NewServer creates and starts a new mock WhatsApp Cloud API test server.
func NewServer(opts ...Option) *Server {
	s := &Server{
		phoneNumberID:  DefaultPhoneNumberID,
		wabaID:         DefaultWABAID,
		apiVersion:     DefaultAPIVersion,
		appSecret:      DefaultAppSecret,
		accessToken:    DefaultAccessToken,
		state:          newState(),
		mux:            http.NewServeMux(),
		customHandlers: make(map[string]http.HandlerFunc),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}

	s.registerRoutes()
	s.httpServer = httptest.NewServer(s)

	// Configure an in-memory client that routes requests directly to ServeHTTP
	// while still maintaining the real httptest server for URL binding.
	s.httpClient = &http.Client{
		Transport: &inMemoryTransport{handler: s},
	}

	return s
}

// ServeHTTP handles incoming HTTP requests to the mock server.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	customHandler, exists := s.customHandlers[r.URL.Path]
	s.mu.RUnlock()

	if exists && customHandler != nil {
		customHandler(w, r)
		return
	}

	s.mux.ServeHTTP(w, r)
}

// registerRoutes registers standard routes on the internal mux.
func (s *Server) registerRoutes() {
	// Root or healthcheck route
	s.mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
}

// URL returns the base URL of the running mock HTTP server.
func (s *Server) URL() string {
	if s.httpServer == nil {
		return ""
	}
	return s.httpServer.URL
}

// Close shuts down the mock HTTP server.
func (s *Server) Close() {
	if s.httpServer != nil {
		s.httpServer.Close()
	}
}

// HTTPClient returns the configured *http.Client for communicating with the mock server.
func (s *Server) HTTPClient() *http.Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.httpClient
}

// Client returns a *wacloudapi.Client configured to target this mock server.
func (s *Server) Client() *wacloudapi.Client {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return wacloudapi.New(
		s.accessToken,
		s.phoneNumberID,
		wacloudapi.WithBaseURL(s.URL()),
		wacloudapi.WithAPIVersion(s.apiVersion),
		wacloudapi.WithWABAID(s.wabaID),
		wacloudapi.WithHTTPClient(s.httpClient),
	)
}

// Reset resets the server's in-memory mock state and custom handlers.
func (s *Server) Reset() {
	s.mu.Lock()
	s.customHandlers = make(map[string]http.HandlerFunc)
	s.mu.Unlock()

	s.state.Reset()
}

// PhoneNumberID returns the configured Phone Number ID.
func (s *Server) PhoneNumberID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.phoneNumberID
}

// WABAID returns the configured WABA ID.
func (s *Server) WABAID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.wabaID
}

// APIVersion returns the configured API Version.
func (s *Server) APIVersion() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.apiVersion
}

// AppSecret returns the configured App Secret.
func (s *Server) AppSecret() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.appSecret
}

// AccessToken returns the configured Access Token.
func (s *Server) AccessToken() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.accessToken
}
