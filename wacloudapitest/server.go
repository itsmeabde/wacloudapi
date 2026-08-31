package wacloudapitest

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"unicode"

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
	phoneNumberID string
	wabaID        string
	apiVersion    string
	appSecret     string
	accessToken   string
	state         *State

	customHandlers map[string]http.HandlerFunc
	errorQueue     []*InjectedError
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
		customHandlers: make(map[string]http.HandlerFunc),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}

	// Ensure phone numbers in state reflect the configured phone number
	if s.phoneNumberID != DefaultPhoneNumberID {
		s.state.mu.Lock()
		s.state.phoneNumbers = append(s.state.phoneNumbers, wacloudapi.PhoneNumberDetails{
			ID:                 s.phoneNumberID,
			DisplayPhoneNumber: "+1 555-0100",
			VerifiedName:       "Test Business",
			QualityRating:      "GREEN",
			CodeVerificationStatus: "VERIFIED",
		})
		s.state.mu.Unlock()
	}

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
	// 1. Check chaos error injection queue
	s.mu.Lock()
	if len(s.errorQueue) > 0 {
		errItem := s.errorQueue[0]
		s.errorQueue = s.errorQueue[1:]
		s.mu.Unlock()

		writeAPIError(w, errItem.StatusCode, errItem.Message, errItem.Code)
		return
	}
	s.mu.Unlock()

	// 2. Check custom handlers
	s.mu.RLock()
	customHandler, exists := s.customHandlers[r.URL.Path]
	if !exists {
		customHandler, exists = s.customHandlers[r.Method+" "+r.URL.Path]
	}
	if !exists {
		// Also check stripped path if API version prefix was present (e.g. /custom_endpoint if request was /v21.0/custom_endpoint)
		path := strings.Trim(r.URL.Path, "/")
		parts := strings.Split(path, "/")
		if len(parts) > 0 && (parts[0] == s.apiVersion || (len(parts[0]) >= 2 && parts[0][0] == 'v' && unicode.IsDigit(rune(parts[0][1])))) {
			stripped := "/" + strings.Join(parts[1:], "/")
			customHandler, exists = s.customHandlers[stripped]
			if !exists {
				customHandler, exists = s.customHandlers[r.Method+" "+stripped]
			}
		}
	}
	s.mu.RUnlock()

	if exists && customHandler != nil {
		customHandler(w, r)
		return
	}

	s.handleGraphAPI(w, r)
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

// Reset resets the server's in-memory mock state, chaos error queue, and custom handlers.
func (s *Server) Reset() {
	s.mu.Lock()
	s.customHandlers = make(map[string]http.HandlerFunc)
	s.errorQueue = nil
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

// SentMessages returns a copy of all SendMessageRequest payloads recorded by the mock server.
func (s *Server) SentMessages() []*wacloudapi.SendMessageRequest {
	s.state.mu.RLock()
	defer s.state.mu.RUnlock()

	result := make([]*wacloudapi.SendMessageRequest, len(s.state.sentMessages))
	copy(result, s.state.sentMessages)
	return result
}

// LastSentMessage returns the most recent SendMessageRequest payload recorded, or nil if none.
func (s *Server) LastSentMessage() *wacloudapi.SendMessageRequest {
	s.state.mu.RLock()
	defer s.state.mu.RUnlock()

	if len(s.state.sentMessages) == 0 {
		return nil
	}
	return s.state.sentMessages[len(s.state.sentMessages)-1]
}

// SentMessagesTo returns all SendMessageRequest payloads sent to the specified recipient phone number.
func (s *Server) SentMessagesTo(phone string) []*wacloudapi.SendMessageRequest {
	s.state.mu.RLock()
	defer s.state.mu.RUnlock()

	var result []*wacloudapi.SendMessageRequest
	for _, msg := range s.state.sentMessages {
		if msg != nil && msg.To == phone {
			result = append(result, msg)
		}
	}
	return result
}

// UploadedMedia returns a copy of the mock server's in-memory media store mapped by media ID.
func (s *Server) UploadedMedia() map[string]*MockMedia {
	s.state.mu.RLock()
	defer s.state.mu.RUnlock()

	result := make(map[string]*MockMedia, len(s.state.mediaStore))
	for k, v := range s.state.mediaStore {
		result[k] = v
	}
	return result
}
