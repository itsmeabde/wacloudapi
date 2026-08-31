package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestHandlerServeHTTP(t *testing.T) {
	verifyToken := "test-verify-token"
	appSecret := "test-secret"
	h := NewHandler(verifyToken, appSecret)

	var (
		msgReceived   int32
		textReceived  int32
		mediaReceived int32
		intReceived   int32
		statusCount   int32
	)

	var wg sync.WaitGroup
	// We expect 4 messages (text, image, interactive, reaction) + 1 status update in our POST test payload below
	// OnMessage: 4 times
	// OnTextMessage: 1 time
	// OnMediaMessage: 1 time
	// OnInteractiveMessage: 1 time
	// OnStatus: 1 time
	wg.Add(8)

	h.OnMessage(func(ctx context.Context, msg Message, meta Metadata) error {
		atomic.AddInt32(&msgReceived, 1)
		wg.Done()
		return nil
	})

	h.OnTextMessage(func(ctx context.Context, msg Message, meta Metadata) error {
		if msg.Text != nil && msg.Text.Body == "Hello Bot" && meta.PhoneNumberID == "123456" {
			atomic.AddInt32(&textReceived, 1)
		}
		wg.Done()
		return nil
	})

	h.OnMediaMessage(func(ctx context.Context, msg Message, meta Metadata) error {
		if msg.Image != nil && msg.Image.ID == "img_123" {
			atomic.AddInt32(&mediaReceived, 1)
		}
		wg.Done()
		return nil
	})

	h.OnInteractiveMessage(func(ctx context.Context, msg Message, meta Metadata) error {
		if msg.Interactive != nil && msg.Interactive.ButtonReply != nil && msg.Interactive.ButtonReply.ID == "btn_yes" {
			atomic.AddInt32(&intReceived, 1)
		}
		wg.Done()
		return nil
	})

	h.OnStatus(func(ctx context.Context, status Status, meta Metadata) error {
		if status.Status == "delivered" && meta.PhoneNumberID == "123456" {
			atomic.AddInt32(&statusCount, 1)
		}
		wg.Done()
		return nil
	})

	// 1. Test GET Challenge Verification
	reqGet := httptest.NewRequest(http.MethodGet, "/webhook?hub.mode=subscribe&hub.verify_token="+verifyToken+"&hub.challenge=test_chall", nil)
	wGet := httptest.NewRecorder()
	h.ServeHTTP(wGet, reqGet)
	if wGet.Code != http.StatusOK || wGet.Body.String() != "test_chall" {
		t.Errorf("unexpected GET challenge response: code %d, body %s", wGet.Code, wGet.Body.String())
	}

	// GET with invalid token
	reqGetInvalid := httptest.NewRequest(http.MethodGet, "/webhook?hub.mode=subscribe&hub.verify_token=wrong&hub.challenge=test_chall", nil)
	wGetInvalid := httptest.NewRecorder()
	h.ServeHTTP(wGetInvalid, reqGetInvalid)
	if wGetInvalid.Code != http.StatusForbidden {
		t.Errorf("expected 403 on invalid GET token, got %d", wGetInvalid.Code)
	}

	// 2. Test Invalid Method (PUT)
	reqPut := httptest.NewRequest(http.MethodPut, "/webhook", nil)
	wPut := httptest.NewRecorder()
	h.ServeHTTP(wPut, reqPut)
	if wPut.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 Method Not Allowed, got %d", wPut.Code)
	}

	// 3. Test POST Event Dispatching
	rawPayload := []byte(`{
		"object": "whatsapp_business_account",
		"entry": [{
			"id": "1",
			"changes": [{
				"field": "messages",
				"value": {
					"messaging_product": "whatsapp",
					"metadata": { "phone_number_id": "123456", "display_phone_number": "123" },
					"messages": [
						{ "from": "6281", "id": "wamid.1", "type": "text", "text": { "body": "Hello Bot" } },
						{ "from": "6281", "id": "wamid.2", "type": "image", "image": { "id": "img_123", "mime_type": "image/jpeg" } },
						{ "from": "6281", "id": "wamid.3", "type": "interactive", "interactive": { "type": "button_reply", "button_reply": { "id": "btn_yes", "title": "Yes" } } },
						{ "from": "6281", "id": "wamid.4", "type": "reaction", "reaction": { "message_id": "wamid.1", "emoji": "👍" } }
					],
					"statuses": [{ "id": "wamid.0", "status": "delivered", "recipient_id": "6281" }]
				}
			}]
		}]
	}`)

	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write(rawPayload)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	reqPost := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(rawPayload))
	reqPost.Header.Set("X-Hub-Signature-256", sig)
	wPost := httptest.NewRecorder()

	h.ServeHTTP(wPost, reqPost)
	if wPost.Code != http.StatusOK {
		t.Errorf("expected POST status 200, got %d", wPost.Code)
	}

	// Wait for async dispatch
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event dispatching callbacks")
	}

	if atomic.LoadInt32(&msgReceived) != 4 {
		t.Errorf("expected 4 OnMessage calls, got %d", msgReceived)
	}
	if atomic.LoadInt32(&textReceived) != 1 {
		t.Errorf("expected 1 OnTextMessage call, got %d", textReceived)
	}
	if atomic.LoadInt32(&mediaReceived) != 1 {
		t.Errorf("expected 1 OnMediaMessage call, got %d", mediaReceived)
	}
	if atomic.LoadInt32(&intReceived) != 1 {
		t.Errorf("expected 1 OnInteractiveMessage call, got %d", intReceived)
	}
	if atomic.LoadInt32(&statusCount) != 1 {
		t.Errorf("expected 1 OnStatus call, got %d", statusCount)
	}
}

func TestHandlerErrors(t *testing.T) {
	verifyToken := "test-verify-token"
	appSecret := "test-secret"
	h := NewHandler(verifyToken, appSecret)

	var errCount int32
	var lastErr error
	var errMu sync.Mutex

	h.OnError(func(ctx context.Context, err error) {
		atomic.AddInt32(&errCount, 1)
		errMu.Lock()
		lastErr = err
		errMu.Unlock()
	})

	// 1. Missing signature
	reqNoSig := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader([]byte(`{}`)))
	wNoSig := httptest.NewRecorder()
	h.ServeHTTP(wNoSig, reqNoSig)
	if wNoSig.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 on missing signature, got %d", wNoSig.Code)
	}
	if atomic.LoadInt32(&errCount) != 1 {
		t.Errorf("expected OnError triggered once, got %d", errCount)
	}
	if !errors.Is(lastErr, ErrMissingSignature) {
		t.Errorf("expected ErrMissingSignature, got %v", lastErr)
	}

	// 2. Invalid signature
	reqBadSig := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader([]byte(`{}`)))
	reqBadSig.Header.Set("X-Hub-Signature-256", "sha256=badhex")
	wBadSig := httptest.NewRecorder()
	h.ServeHTTP(wBadSig, reqBadSig)
	if wBadSig.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 on bad signature, got %d", wBadSig.Code)
	}
	if atomic.LoadInt32(&errCount) != 2 {
		t.Errorf("expected OnError triggered twice, got %d", errCount)
	}

	// 3. Invalid JSON payload
	badJSON := []byte(`{invalid`)
	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write(badJSON)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	reqBadJSON := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(badJSON))
	reqBadJSON.Header.Set("X-Hub-Signature-256", sig)
	wBadJSON := httptest.NewRecorder()
	h.ServeHTTP(wBadJSON, reqBadJSON)
	if wBadJSON.Code != http.StatusBadRequest {
		t.Errorf("expected 400 on bad JSON, got %d", wBadJSON.Code)
	}
	if atomic.LoadInt32(&errCount) != 3 {
		t.Errorf("expected OnError triggered 3 times, got %d", errCount)
	}
}

func TestHandlerNoAppSecret(t *testing.T) {
	// If appSecret is empty, signature verification should be skipped
	h := NewHandler("verify-tok", "")

	var called int32
	var wg sync.WaitGroup
	wg.Add(1)

	h.OnTextMessage(func(ctx context.Context, msg Message, meta Metadata) error {
		atomic.AddInt32(&called, 1)
		wg.Done()
		return nil
	})

	rawPayload := []byte(`{
		"object": "whatsapp_business_account",
		"entry": [{
			"id": "1",
			"changes": [{
				"field": "messages",
				"value": {
					"messaging_product": "whatsapp",
					"metadata": { "phone_number_id": "123456", "display_phone_number": "123" },
					"messages": [{ "from": "6281", "id": "wamid.1", "type": "text", "text": { "body": "No secret test" } }]
				}
			}]
		}]
	}`)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(rawPayload))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK without appSecret, got %d", w.Code)
	}

	wg.Wait()
	if atomic.LoadInt32(&called) != 1 {
		t.Errorf("expected 1 OnTextMessage call, got %d", called)
	}
}

func TestIsMediaType(t *testing.T) {
	mediaTypes := []string{"image", "audio", "video", "document", "sticker"}
	for _, mt := range mediaTypes {
		if !isMediaType(mt) {
			t.Errorf("expected %s to be recognized as media type", mt)
		}
	}

	nonMedia := []string{"text", "interactive", "location", "contacts", "reaction", "system", "unknown"}
	for _, nmt := range nonMedia {
		if isMediaType(nmt) {
			t.Errorf("expected %s NOT to be recognized as media type", nmt)
		}
	}
}

func TestHandlerCallbackErrorPropagation(t *testing.T) {
	h := NewHandler("tok", "")

	expectedErr := errors.New("callback failed")
	var errCount int32
	var wg sync.WaitGroup
	wg.Add(2) // 1 from OnTextMessage, 1 from OnStatus

	h.OnError(func(ctx context.Context, err error) {
		if errors.Is(err, expectedErr) {
			atomic.AddInt32(&errCount, 1)
		}
	})

	h.OnTextMessage(func(ctx context.Context, msg Message, meta Metadata) error {
		defer wg.Done()
		return expectedErr
	})

	h.OnStatus(func(ctx context.Context, status Status, meta Metadata) error {
		defer wg.Done()
		return expectedErr
	})

	payload := &Payload{
		Entry: []Entry{
			{
				Changes: []Change{
					{
						Value: Value{
							Messages: []Message{
								{Type: "text", Text: &Text{Body: "hi"}},
							},
							Statuses: []Status{
								{ID: "status-1", Status: "delivered"},
							},
						},
					},
				},
			},
		},
	}

	h.dispatchEvents(context.Background(), payload)
	wg.Wait()

	if atomic.LoadInt32(&errCount) != 2 {
		t.Errorf("expected 2 errors propagated to OnError, got %d", errCount)
	}
}

func TestHandlerNoDeadlockWhenRegisteringInCallback(t *testing.T) {
	h := NewHandler("tok", "")

	var wg sync.WaitGroup
	wg.Add(1)

	// In callback, register a new handler on h (which acquires h.mu.Lock)
	h.OnTextMessage(func(ctx context.Context, msg Message, meta Metadata) error {
		defer wg.Done()
		h.OnMessage(func(ctx context.Context, msg Message, meta Metadata) error {
			return nil
		})
		return nil
	})

	payload := &Payload{
		Entry: []Entry{
			{
				Changes: []Change{
					{
						Value: Value{
							Messages: []Message{
								{Type: "text", Text: &Text{Body: "hi"}},
							},
						},
					},
				},
			},
		},
	}

	done := make(chan struct{})
	go func() {
		h.dispatchEvents(context.Background(), payload)
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("deadlock occurred when registering callback inside a handler")
	}
}

func TestHandlerOrderAndFlowCallbacks(t *testing.T) {
	verifyToken := "test-verify-token"
	appSecret := "test-secret"
	h := NewHandler(verifyToken, appSecret)

	var (
		msgReceived   int32
		orderReceived int32
		flowReceived  int32
	)

	var (
		capturedOrder Order
		capturedFlow  NFMReply
		capturedMeta  Metadata
		capturedMsg   Message
	)

	var wg sync.WaitGroup
	// We will send 2 messages: 1 order message and 1 flow interactive response message.
	// OnMessage: 2 times
	// OnOrderMessage: 1 time
	// OnFlowResponseMessage: 1 time
	wg.Add(4)

	h.OnMessage(func(ctx context.Context, msg Message, meta Metadata) error {
		atomic.AddInt32(&msgReceived, 1)
		wg.Done()
		return nil
	})

	h.OnOrderMessage(func(ctx context.Context, order Order, msg Message, meta Metadata) error {
		capturedOrder = order
		capturedMsg = msg
		capturedMeta = meta
		atomic.AddInt32(&orderReceived, 1)
		wg.Done()
		return nil
	})

	h.OnFlowResponseMessage(func(ctx context.Context, reply NFMReply, msg Message, meta Metadata) error {
		capturedFlow = reply
		atomic.AddInt32(&flowReceived, 1)
		wg.Done()
		return nil
	})

	rawPayload := []byte(`{
		"object": "whatsapp_business_account",
		"entry": [{
			"id": "1",
			"changes": [{
				"field": "messages",
				"value": {
					"messaging_product": "whatsapp",
					"metadata": { "phone_number_id": "123456", "display_phone_number": "123" },
					"messages": [
						{
							"from": "6281",
							"id": "wamid.order1",
							"type": "order",
							"order": {
								"catalog_id": "cat_test_1",
								"text": "Please deliver soon",
								"product_items": [
									{
										"product_retailer_id": "sku_1",
										"quantity": "3",
										"item_price": 99.5,
										"currency": "USD"
									}
								]
							}
						},
						{
							"from": "6281",
							"id": "wamid.flow1",
							"type": "interactive",
							"interactive": {
								"type": "nfm_reply",
								"nfm_reply": {
									"name": "flow",
									"body": "Sent",
									"response_json": "{\"screen\":\"SUCCESS\"}"
								}
							}
						}
					]
				}
			}]
		}]
	}`)

	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write(rawPayload)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	reqPost := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(rawPayload))
	reqPost.Header.Set("X-Hub-Signature-256", sig)
	wPost := httptest.NewRecorder()

	h.ServeHTTP(wPost, reqPost)
	if wPost.Code != http.StatusOK {
		t.Errorf("expected POST status 200, got %d", wPost.Code)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for order and flow callbacks")
	}

	if atomic.LoadInt32(&msgReceived) != 2 {
		t.Errorf("expected 2 OnMessage calls, got %d", msgReceived)
	}
	if atomic.LoadInt32(&orderReceived) != 1 {
		t.Errorf("expected 1 OnOrderMessage call, got %d", orderReceived)
	}
	if atomic.LoadInt32(&flowReceived) != 1 {
		t.Errorf("expected 1 OnFlowResponseMessage call, got %d", flowReceived)
	}

	if capturedOrder.CatalogID != "cat_test_1" || len(capturedOrder.ProductItems) != 1 || capturedOrder.ProductItems[0].ProductRetailerID != "sku_1" {
		t.Errorf("unexpected captured order: %+v", capturedOrder)
	}
	if capturedMsg.ID != "wamid.order1" {
		t.Errorf("unexpected captured msg: %+v", capturedMsg)
	}
	if capturedFlow.Name != "flow" || capturedFlow.ResponseJSON != `{"screen":"SUCCESS"}` {
		t.Errorf("unexpected captured flow reply: %+v", capturedFlow)
	}
	if capturedMeta.PhoneNumberID != "123456" {
		t.Errorf("unexpected captured meta: %+v", capturedMeta)
	}
}

func TestHandlerOrderAndFlowCallbackErrorPropagation(t *testing.T) {
	h := NewHandler("tok", "")

	orderErr := errors.New("order handling failed")
	flowErr := errors.New("flow handling failed")

	var orderErrCount int32
	var flowErrCount int32
	var wg sync.WaitGroup
	wg.Add(2)

	h.OnError(func(ctx context.Context, err error) {
		if errors.Is(err, orderErr) {
			atomic.AddInt32(&orderErrCount, 1)
		}
		if errors.Is(err, flowErr) {
			atomic.AddInt32(&flowErrCount, 1)
		}
	})

	h.OnOrderMessage(func(ctx context.Context, order Order, msg Message, meta Metadata) error {
		defer wg.Done()
		return orderErr
	})

	h.OnFlowResponseMessage(func(ctx context.Context, reply NFMReply, msg Message, meta Metadata) error {
		defer wg.Done()
		return flowErr
	})

	payload := &Payload{
		Entry: []Entry{
			{
				Changes: []Change{
					{
						Value: Value{
							Messages: []Message{
								{
									Type: "order",
									Order: &Order{
										CatalogID: "cat_1",
									},
								},
								{
									Type: "interactive",
									Interactive: &Interactive{
										Type: InteractiveTypeNFMReply,
										NFMReply: &NFMReply{
											Name: "flow",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	h.dispatchEvents(context.Background(), payload)
	wg.Wait()

	if atomic.LoadInt32(&orderErrCount) != 1 {
		t.Errorf("expected 1 order error propagated, got %d", orderErrCount)
	}
	if atomic.LoadInt32(&flowErrCount) != 1 {
		t.Errorf("expected 1 flow error propagated, got %d", flowErrCount)
	}
}


