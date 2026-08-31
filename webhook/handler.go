package webhook

import (
	"context"
	"io"
	"net/http"
	"sync"
)

type MessageHandler func(ctx context.Context, msg Message, meta Metadata) error
type StatusHandler func(ctx context.Context, status Status, meta Metadata) error
type ErrorHandler func(ctx context.Context, err error)

type Handler struct {
	verifyToken string
	appSecret   string

	mu               sync.RWMutex
	onMessage        []MessageHandler
	onTextMessage    []MessageHandler
	onMediaMessage   []MessageHandler
	onInteractiveMsg []MessageHandler
	onStatus         []StatusHandler
	onError          []ErrorHandler
}

func NewHandler(verifyToken, appSecret string) *Handler {
	return &Handler{
		verifyToken: verifyToken,
		appSecret:   appSecret,
	}
}

func (h *Handler) OnMessage(fn MessageHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onMessage = append(h.onMessage, fn)
}

func (h *Handler) OnTextMessage(fn MessageHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onTextMessage = append(h.onTextMessage, fn)
}

func (h *Handler) OnMediaMessage(fn MessageHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onMediaMessage = append(h.onMediaMessage, fn)
}

func (h *Handler) OnInteractiveMessage(fn MessageHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onInteractiveMsg = append(h.onInteractiveMsg, fn)
}

func (h *Handler) OnStatus(fn StatusHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onStatus = append(h.onStatus, fn)
}

func (h *Handler) OnError(fn ErrorHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onError = append(h.onError, fn)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		VerifyChallenge(w, r, h.verifyToken)
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.dispatchError(r.Context(), err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if h.appSecret != "" {
		sig := r.Header.Get("X-Hub-Signature-256")
		if err := VerifySignature(bodyBytes, sig, h.appSecret); err != nil {
			h.dispatchError(r.Context(), err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	payload, err := ParsePayload(bodyBytes)
	if err != nil {
		h.dispatchError(r.Context(), err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)

	// Dispatch events
	go h.dispatchEvents(context.Background(), payload)
}

func (h *Handler) dispatchEvents(ctx context.Context, payload *Payload) {
	h.mu.RLock()
	onMsg := append([]MessageHandler(nil), h.onMessage...)
	onText := append([]MessageHandler(nil), h.onTextMessage...)
	onMedia := append([]MessageHandler(nil), h.onMediaMessage...)
	onInteractive := append([]MessageHandler(nil), h.onInteractiveMsg...)
	onStatus := append([]StatusHandler(nil), h.onStatus...)
	h.mu.RUnlock()

	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			val := change.Value
			meta := val.Metadata

			// Process Messages
			for _, msg := range val.Messages {
				for _, fn := range onMsg {
					if err := fn(ctx, msg, meta); err != nil {
						h.dispatchError(ctx, err)
					}
				}

				if msg.Type == "text" {
					for _, fn := range onText {
						if err := fn(ctx, msg, meta); err != nil {
							h.dispatchError(ctx, err)
						}
					}
				}

				if isMediaType(msg.Type) {
					for _, fn := range onMedia {
						if err := fn(ctx, msg, meta); err != nil {
							h.dispatchError(ctx, err)
						}
					}
				}

				if msg.Type == "interactive" {
					for _, fn := range onInteractive {
						if err := fn(ctx, msg, meta); err != nil {
							h.dispatchError(ctx, err)
						}
					}
				}
			}

			// Process Statuses
			for _, status := range val.Statuses {
				for _, fn := range onStatus {
					if err := fn(ctx, status, meta); err != nil {
						h.dispatchError(ctx, err)
					}
				}
			}
		}
	}
}

func (h *Handler) dispatchError(ctx context.Context, err error) {
	h.mu.RLock()
	onError := append([]ErrorHandler(nil), h.onError...)
	h.mu.RUnlock()
	for _, fn := range onError {
		fn(ctx, err)
	}
}

func isMediaType(t string) bool {
	switch t {
	case "image", "audio", "video", "document", "sticker":
		return true
	default:
		return false
	}
}
