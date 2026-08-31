package wacloudapitest

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/itsmeabde/wacloudapi/webhook"
)

var mockMsgIDCounter uint64

func generateMockWAMID() string {
	val := atomic.AddUint64(&mockMsgIDCounter, 1)
	return fmt.Sprintf("wamid.mock.%d.%d", time.Now().UnixNano(), val)
}

// SimulateWebhook sends a custom webhook.Payload to the specified target (URL string or http.Handler),
// signed with the mock server's configured Meta App Secret via HMAC-SHA256 (X-Hub-Signature-256 header).
func (s *Server) SimulateWebhook(ctx context.Context, target interface{}, payload *webhook.Payload) error {
	if target == nil {
		return errors.New("webhook simulator: target cannot be nil")
	}
	if payload == nil {
		return errors.New("webhook simulator: payload cannot be nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("webhook simulator: failed to marshal payload: %w", err)
	}

	mac := hmac.New(sha256.New, []byte(s.AppSecret()))
	mac.Write(bodyBytes)
	sigHeader := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	switch tgt := target.(type) {
	case string:
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, tgt, bytes.NewReader(bodyBytes))
		if err != nil {
			return fmt.Errorf("webhook simulator: failed to create HTTP request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Hub-Signature-256", sigHeader)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("webhook simulator: request failed: %w", err)
		}
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, resp.Body)

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("webhook simulator: target URL returned status %d", resp.StatusCode)
		}
		return nil

	case *httptest.Server:
		if tgt.Config != nil && tgt.Config.Handler != nil {
			return s.SimulateWebhook(ctx, tgt.Config.Handler, payload)
		}
		return s.SimulateWebhook(ctx, tgt.URL, payload)

	case http.Handler:
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "/webhook", bytes.NewReader(bodyBytes))
		if err != nil {
			return fmt.Errorf("webhook simulator: failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Hub-Signature-256", sigHeader)

		rec := httptest.NewRecorder()
		tgt.ServeHTTP(rec, req)

		if rec.Code < 200 || rec.Code >= 300 {
			return fmt.Errorf("webhook simulator: handler returned status %d", rec.Code)
		}
		return nil

	default:
		return fmt.Errorf("webhook simulator: unsupported target type %T, expected string URL or http.Handler", target)
	}
}

// SimulateInboundText simulates an incoming text message from a user to the business.
func (s *Server) SimulateInboundText(ctx context.Context, target interface{}, from, text string) error {
	msg := webhook.Message{
		From:      from,
		ID:        generateMockWAMID(),
		Timestamp: strconv.FormatInt(time.Now().Unix(), 10),
		Type:      "text",
		Text:      &webhook.Text{Body: text},
	}
	contact := webhook.Contact{
		Profile: webhook.ContactProfile{Name: from},
		WaID:    from,
	}
	return s.SimulateWebhook(ctx, target, s.buildMessagePayload(msg, contact))
}

// SimulateInboundMedia simulates an incoming media message (image, audio, video, document, sticker) from a user.
func (s *Server) SimulateInboundMedia(ctx context.Context, target interface{}, from, mediaType, mediaID string) error {
	mediaObj := &webhook.Media{
		ID: mediaID,
	}
	switch mediaType {
	case "image":
		mediaObj.MimeType = "image/jpeg"
	case "audio":
		mediaObj.MimeType = "audio/ogg"
	case "video":
		mediaObj.MimeType = "video/mp4"
	case "document":
		mediaObj.MimeType = "application/pdf"
		mediaObj.Filename = "document.pdf"
	case "sticker":
		mediaObj.MimeType = "image/webp"
	default:
		mediaObj.MimeType = "application/octet-stream"
	}

	msg := webhook.Message{
		From:      from,
		ID:        generateMockWAMID(),
		Timestamp: strconv.FormatInt(time.Now().Unix(), 10),
		Type:      mediaType,
	}
	switch mediaType {
	case "image":
		msg.Image = mediaObj
	case "audio":
		msg.Audio = mediaObj
	case "video":
		msg.Video = mediaObj
	case "document":
		msg.Document = mediaObj
	case "sticker":
		msg.Sticker = mediaObj
	}

	contact := webhook.Contact{
		Profile: webhook.ContactProfile{Name: from},
		WaID:    from,
	}
	return s.SimulateWebhook(ctx, target, s.buildMessagePayload(msg, contact))
}

// SimulateInboundOrder simulates an incoming customer order message.
func (s *Server) SimulateInboundOrder(ctx context.Context, target interface{}, from, catalogID string, items []webhook.OrderItem) error {
	msg := webhook.Message{
		From:      from,
		ID:        generateMockWAMID(),
		Timestamp: strconv.FormatInt(time.Now().Unix(), 10),
		Type:      "order",
		Order: &webhook.Order{
			CatalogID:    catalogID,
			ProductItems: items,
		},
	}
	contact := webhook.Contact{
		Profile: webhook.ContactProfile{Name: from},
		WaID:    from,
	}
	return s.SimulateWebhook(ctx, target, s.buildMessagePayload(msg, contact))
}

// SimulateInboundFlowResponse simulates an incoming WhatsApp Flow completion response (NFM reply).
func (s *Server) SimulateInboundFlowResponse(ctx context.Context, target interface{}, from, flowToken, responseJSON string) error {
	msg := webhook.Message{
		From:      from,
		ID:        generateMockWAMID(),
		Timestamp: strconv.FormatInt(time.Now().Unix(), 10),
		Type:      "interactive",
		Interactive: &webhook.Interactive{
			Type: webhook.InteractiveTypeNFMReply,
			NFMReply: &webhook.NFMReply{
				Name:         "flow",
				Body:         "Sent",
				ResponseJSON: responseJSON,
			},
		},
	}
	contact := webhook.Contact{
		Profile: webhook.ContactProfile{Name: from},
		WaID:    from,
	}
	return s.SimulateWebhook(ctx, target, s.buildMessagePayload(msg, contact))
}

// SimulateStatusDelivered simulates a message delivery status update event.
func (s *Server) SimulateStatusDelivered(ctx context.Context, target interface{}, messageID, recipientID string) error {
	status := webhook.Status{
		ID:          messageID,
		Status:      "delivered",
		Timestamp:   strconv.FormatInt(time.Now().Unix(), 10),
		RecipientID: recipientID,
		Conversation: &webhook.Conversation{
			ID: "conv_" + messageID,
			Origin: webhook.Origin{
				Type: "business_initiated",
			},
		},
		Pricing: &webhook.Pricing{
			Billable:     true,
			PricingModel: "CBP",
			Category:     "service",
		},
	}
	return s.simulateStatus(ctx, target, status)
}

// SimulateStatusRead simulates a message read status update event.
func (s *Server) SimulateStatusRead(ctx context.Context, target interface{}, messageID, recipientID string) error {
	status := webhook.Status{
		ID:          messageID,
		Status:      "read",
		Timestamp:   strconv.FormatInt(time.Now().Unix(), 10),
		RecipientID: recipientID,
	}
	return s.simulateStatus(ctx, target, status)
}

// SimulateStatusFailed simulates a message delivery failure status update event.
func (s *Server) SimulateStatusFailed(ctx context.Context, target interface{}, messageID, recipientID string, code int, title string) error {
	status := webhook.Status{
		ID:          messageID,
		Status:      "failed",
		Timestamp:   strconv.FormatInt(time.Now().Unix(), 10),
		RecipientID: recipientID,
		Errors: []webhook.APIError{
			{
				Code:    code,
				Title:   title,
				Message: title,
			},
		},
	}
	return s.simulateStatus(ctx, target, status)
}

func (s *Server) buildWebhookPayload(changes ...webhook.Change) *webhook.Payload {
	return &webhook.Payload{
		Object: "whatsapp_business_account",
		Entry: []webhook.Entry{
			{
				ID:      s.WABAID(),
				Changes: changes,
			},
		},
	}
}

func (s *Server) buildMessagePayload(msg webhook.Message, contact webhook.Contact) *webhook.Payload {
	return s.buildWebhookPayload(webhook.Change{
		Field: "messages",
		Value: webhook.Value{
			MessagingProduct: "whatsapp",
			Metadata: webhook.Metadata{
				PhoneNumberID:      s.PhoneNumberID(),
				DisplayPhoneNumber: "+1 555-0100",
			},
			Contacts: []webhook.Contact{contact},
			Messages: []webhook.Message{msg},
		},
	})
}

func (s *Server) simulateStatus(ctx context.Context, target interface{}, status webhook.Status) error {
	payload := s.buildWebhookPayload(webhook.Change{
		Field: "messages",
		Value: webhook.Value{
			MessagingProduct: "whatsapp",
			Metadata: webhook.Metadata{
				PhoneNumberID:      s.PhoneNumberID(),
				DisplayPhoneNumber: "+1 555-0100",
			},
			Statuses: []webhook.Status{status},
		},
	})
	return s.SimulateWebhook(ctx, target, payload)
}
