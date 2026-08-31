package flows

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
)

// ScreenHandler handles a WhatsApp Flows request for a specific screen.
type ScreenHandler func(ctx context.Context, req *Request) (*Response, error)

// ActionHandler handles WhatsApp Flows requests when no screen-specific handler matches.
type ActionHandler func(ctx context.Context, req *Request) (*Response, error)

// PingHandler handles WhatsApp Flows health check ("ping") requests.
type PingHandler func(ctx context.Context, req *Request) (*Response, error)

// ErrorHandler handles internal errors encountered during request processing, decryption, or callbacks.
type ErrorHandler func(ctx context.Context, err error)

// Handler implements http.Handler for the WhatsApp Flows Data Endpoint.
// It decrypts incoming requests, routes them to registered screen, ping, or action handlers,
// and encrypts responses back using the session's AES key with an inverted initial vector.
type Handler struct {
	privateKey *rsa.PrivateKey

	mu             sync.RWMutex
	screenHandlers map[string]ScreenHandler
	onAction       ActionHandler
	onPing         PingHandler
	onError        []ErrorHandler
}

// NewHandler creates a new WhatsApp Flows Handler using the provided RSA private key.
func NewHandler(key *rsa.PrivateKey) *Handler {
	return &Handler{
		privateKey:     key,
		screenHandlers: make(map[string]ScreenHandler),
	}
}

// NewHandlerFromPEM creates a new WhatsApp Flows Handler by parsing RSA private key PEM bytes.
// It supports PKCS#1, PKCS#8, and encrypted PEM blocks with an optional passphrase.
func NewHandlerFromPEM(pemBytes []byte, passphrase ...string) (*Handler, error) {
	key, err := ParsePrivateKey(pemBytes, passphrase...)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key from PEM: %w", err)
	}
	return NewHandler(key), nil
}

// NewHandlerFromKeyFile creates a new WhatsApp Flows Handler by reading an RSA private key file from disk.
// It supports PKCS#1, PKCS#8, and encrypted PEM blocks with an optional passphrase.
func NewHandlerFromKeyFile(path string, passphrase ...string) (*Handler, error) {
	key, err := LoadPrivateKeyFile(path, passphrase...)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key from file: %w", err)
	}
	return NewHandler(key), nil
}

// HandleScreen registers a handler for a specific screen ID.
func (h *Handler) HandleScreen(screenID string, fn ScreenHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.screenHandlers == nil {
		h.screenHandlers = make(map[string]ScreenHandler)
	}
	h.screenHandlers[screenID] = fn
}

// OnAction registers a fallback or generic handler for flow actions.
func (h *Handler) OnAction(fn ActionHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onAction = fn
}

// OnPing registers a custom handler for health check ("ping") requests.
func (h *Handler) OnPing(fn PingHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onPing = fn
}

// OnError registers an error callback to handle internal decryption, decoding, or execution errors.
func (h *Handler) OnError(fn ErrorHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onError = append(h.onError, fn)
}

// ServeHTTP handles incoming HTTP requests from WhatsApp Flows.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	if h.privateKey == nil {
		err := errors.New("flows handler: private key is not configured")
		h.dispatchError(ctx, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.dispatchError(ctx, fmt.Errorf("failed to read request body: %w", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var payload EncryptedPayload
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		h.dispatchError(ctx, fmt.Errorf("failed to parse encrypted payload: %w", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	req, session, err := DecryptRequest(&payload, h.privateKey)
	if err != nil {
		h.dispatchError(ctx, fmt.Errorf("failed to decrypt flow request: %w", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var resp *Response

	if req.Action == "ping" {
		h.mu.RLock()
		pingFn := h.onPing
		h.mu.RUnlock()

		if pingFn != nil {
			resp, err = pingFn(ctx, req)
		} else {
			resp = &Response{
				Version: req.Version,
				Data: map[string]interface{}{
					"status": "active",
				},
			}
		}
	} else {
		var screenFn ScreenHandler
		if req.Screen != "" {
			h.mu.RLock()
			screenFn = h.screenHandlers[req.Screen]
			h.mu.RUnlock()
		}

		if screenFn != nil {
			resp, err = screenFn(ctx, req)
		} else {
			h.mu.RLock()
			actionFn := h.onAction
			h.mu.RUnlock()

			if actionFn != nil {
				resp, err = actionFn(ctx, req)
			} else {
				err = fmt.Errorf("unhandled flow request: screen=%q, action=%q", req.Screen, req.Action)
			}
		}
	}

	if err != nil {
		h.dispatchError(ctx, err)
		if resp == nil {
			resp = &Response{
				Version: req.Version,
				Error: &FlowError{
					Message: err.Error(),
				},
			}
		}
	}

	if resp == nil {
		resp = &Response{
			Version: req.Version,
			Data:    map[string]interface{}{},
		}
	} else if resp.Version == "" && req.Version != "" {
		resp.Version = req.Version
	}

	encRespB64, err := EncryptResponse(resp, session)
	if err != nil {
		h.dispatchError(ctx, fmt.Errorf("failed to encrypt flow response: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(encRespB64))
}

func (h *Handler) dispatchError(ctx context.Context, err error) {
	h.mu.RLock()
	onError := append([]ErrorHandler(nil), h.onError...)
	h.mu.RUnlock()
	for _, fn := range onError {
		fn(ctx, err)
	}
}
