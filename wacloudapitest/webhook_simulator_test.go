package wacloudapitest_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/itsmeabde/wacloudapi/wacloudapitest"
	"github.com/itsmeabde/wacloudapi/webhook"
)

func TestSimulateWebhook_DirectHandler_InboundText(t *testing.T) {
	appSecret := "sim_secret_123"
	srv := wacloudapitest.NewServer(
		wacloudapitest.WithAppSecret(appSecret),
		wacloudapitest.WithPhoneNumberID("phone_sim_1"),
		wacloudapitest.WithWABAID("waba_sim_1"),
	)
	defer srv.Close()

	h := webhook.NewHandler("verify_tok", appSecret)

	var receivedText string
	var receivedFrom string
	var receivedPhoneID string
	var wg sync.WaitGroup
	wg.Add(1)

	h.OnTextMessage(func(ctx context.Context, msg webhook.Message, meta webhook.Metadata) error {
		if msg.Text != nil {
			receivedText = msg.Text.Body
		}
		receivedFrom = msg.From
		receivedPhoneID = meta.PhoneNumberID
		wg.Done()
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := srv.SimulateInboundText(ctx, h, "628123456789", "Hello from simulator!")
	if err != nil {
		t.Fatalf("SimulateInboundText failed: %v", err)
	}

	wg.Wait()

	if receivedText != "Hello from simulator!" {
		t.Errorf("expected text %q, got %q", "Hello from simulator!", receivedText)
	}
	if receivedFrom != "628123456789" {
		t.Errorf("expected from %q, got %q", "628123456789", receivedFrom)
	}
	if receivedPhoneID != "phone_sim_1" {
		t.Errorf("expected phone_number_id %q, got %q", "phone_sim_1", receivedPhoneID)
	}
}

func TestSimulateWebhook_HTTPServerURL(t *testing.T) {
	appSecret := "sim_secret_url"
	srv := wacloudapitest.NewServer(
		wacloudapitest.WithAppSecret(appSecret),
	)
	defer srv.Close()

	h := webhook.NewHandler("verify_tok", appSecret)

	var receivedCount int32
	var wg sync.WaitGroup
	wg.Add(1)

	h.OnTextMessage(func(ctx context.Context, msg webhook.Message, meta webhook.Metadata) error {
		if msg.Text != nil && msg.Text.Body == "HTTP server test" {
			atomic.AddInt32(&receivedCount, 1)
		}
		wg.Done()
		return nil
	})

	ts := httptest.NewServer(h)
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := srv.SimulateInboundText(ctx, ts.URL, "628999", "HTTP server test")
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("skipping TCP dial test in restricted sandbox environment: %v", err)
		}
		t.Fatalf("SimulateInboundText via URL failed: %v", err)
	}

	wg.Wait()

	if atomic.LoadInt32(&receivedCount) != 1 {
		t.Errorf("expected 1 message received, got %d", receivedCount)
	}
}

func TestSimulateWebhook_HTTPTestServerTarget(t *testing.T) {
	appSecret := "sim_secret_ts"
	srv := wacloudapitest.NewServer(
		wacloudapitest.WithAppSecret(appSecret),
	)
	defer srv.Close()

	h := webhook.NewHandler("verify_tok", appSecret)

	var receivedCount int32
	var wg sync.WaitGroup
	wg.Add(1)

	h.OnTextMessage(func(ctx context.Context, msg webhook.Message, meta webhook.Metadata) error {
		if msg.Text != nil && msg.Text.Body == "HTTPTestServer target test" {
			atomic.AddInt32(&receivedCount, 1)
		}
		wg.Done()
		return nil
	})

	ts := httptest.NewServer(h)
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := srv.SimulateInboundText(ctx, ts, "628999", "HTTPTestServer target test")
	if err != nil {
		t.Fatalf("SimulateInboundText with *httptest.Server failed: %v", err)
	}

	wg.Wait()

	if atomic.LoadInt32(&receivedCount) != 1 {
		t.Errorf("expected 1 message received, got %d", receivedCount)
	}
}

func TestSimulateWebhook_InboundMedia_AllTypes(t *testing.T) {
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	mediaCases := []struct {
		mediaType string
		mediaID   string
	}{
		{"image", "img_test_123"},
		{"audio", "audio_test_456"},
		{"video", "video_test_789"},
		{"document", "doc_test_101"},
		{"sticker", "stk_test_202"},
	}

	for _, tc := range mediaCases {
		t.Run(tc.mediaType, func(t *testing.T) {
			h := webhook.NewHandler("verify_tok", srv.AppSecret())

			var capturedMsg webhook.Message
			var wg sync.WaitGroup
			wg.Add(1)

			h.OnMediaMessage(func(ctx context.Context, msg webhook.Message, meta webhook.Metadata) error {
				capturedMsg = msg
				wg.Done()
				return nil
			})

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			err := srv.SimulateInboundMedia(ctx, h, "628111", tc.mediaType, tc.mediaID)
			if err != nil {
				t.Fatalf("SimulateInboundMedia(%s) failed: %v", tc.mediaType, err)
			}

			wg.Wait()

			if capturedMsg.Type != tc.mediaType {
				t.Errorf("expected msg type %q, got %q", tc.mediaType, capturedMsg.Type)
			}
			if capturedMsg.MediaID() != tc.mediaID {
				t.Errorf("expected media ID %q, got %q", tc.mediaID, capturedMsg.MediaID())
			}
		})
	}
}

func TestSimulateWebhook_InboundOrder(t *testing.T) {
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	h := webhook.NewHandler("verify_tok", srv.AppSecret())

	var capturedOrder webhook.Order
	var capturedFrom string
	var wg sync.WaitGroup
	wg.Add(1)

	h.OnOrderMessage(func(ctx context.Context, order webhook.Order, msg webhook.Message, meta webhook.Metadata) error {
		capturedOrder = order
		capturedFrom = msg.From
		wg.Done()
		return nil
	})

	items := []webhook.OrderItem{
		{
			ProductRetailerID: "SKU-990",
			Quantity:          "2",
			ItemPrice:         150.75,
			Currency:          "USD",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := srv.SimulateInboundOrder(ctx, h, "628555", "catalog_food_1", items)
	if err != nil {
		t.Fatalf("SimulateInboundOrder failed: %v", err)
	}

	wg.Wait()

	if capturedFrom != "628555" {
		t.Errorf("expected from 628555, got %q", capturedFrom)
	}
	if capturedOrder.CatalogID != "catalog_food_1" {
		t.Errorf("expected catalog ID 'catalog_food_1', got %q", capturedOrder.CatalogID)
	}
	if len(capturedOrder.ProductItems) != 1 || capturedOrder.ProductItems[0].ProductRetailerID != "SKU-990" {
		t.Errorf("unexpected product items: %+v", capturedOrder.ProductItems)
	}
}

func TestSimulateWebhook_InboundFlowResponse(t *testing.T) {
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	h := webhook.NewHandler("verify_tok", srv.AppSecret())

	var capturedReply webhook.NFMReply
	var capturedFrom string
	var wg sync.WaitGroup
	wg.Add(1)

	h.OnFlowResponseMessage(func(ctx context.Context, reply webhook.NFMReply, msg webhook.Message, meta webhook.Metadata) error {
		capturedReply = reply
		capturedFrom = msg.From
		wg.Done()
		return nil
	})

	flowResponseJSON := `{"screen":"APPOINTMENT_SUMMARY","booking_id":"BK-404","status":"confirmed"}`
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := srv.SimulateInboundFlowResponse(ctx, h, "628777", "token_flow_abc", flowResponseJSON)
	if err != nil {
		t.Fatalf("SimulateInboundFlowResponse failed: %v", err)
	}

	wg.Wait()

	if capturedFrom != "628777" {
		t.Errorf("expected from 628777, got %q", capturedFrom)
	}
	if capturedReply.Name != "flow" {
		t.Errorf("expected reply name 'flow', got %q", capturedReply.Name)
	}
	if capturedReply.ResponseJSON != flowResponseJSON {
		t.Errorf("expected response JSON %q, got %q", flowResponseJSON, capturedReply.ResponseJSON)
	}
}

func TestSimulateWebhook_Statuses(t *testing.T) {
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	t.Run("Delivered", func(t *testing.T) {
		h := webhook.NewHandler("verify_tok", srv.AppSecret())

		var capturedStatus webhook.Status
		var wg sync.WaitGroup
		wg.Add(1)

		h.OnStatus(func(ctx context.Context, status webhook.Status, meta webhook.Metadata) error {
			capturedStatus = status
			wg.Done()
			return nil
		})

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := srv.SimulateStatusDelivered(ctx, h, "wamid.deliv.1", "628111222")
		if err != nil {
			t.Fatalf("SimulateStatusDelivered failed: %v", err)
		}

		wg.Wait()

		if capturedStatus.ID != "wamid.deliv.1" {
			t.Errorf("expected status ID 'wamid.deliv.1', got %q", capturedStatus.ID)
		}
		if capturedStatus.Status != "delivered" {
			t.Errorf("expected status 'delivered', got %q", capturedStatus.Status)
		}
		if capturedStatus.RecipientID != "628111222" {
			t.Errorf("expected recipient '628111222', got %q", capturedStatus.RecipientID)
		}
	})

	t.Run("Read", func(t *testing.T) {
		h := webhook.NewHandler("verify_tok", srv.AppSecret())

		var capturedStatus webhook.Status
		var wg sync.WaitGroup
		wg.Add(1)

		h.OnStatus(func(ctx context.Context, status webhook.Status, meta webhook.Metadata) error {
			capturedStatus = status
			wg.Done()
			return nil
		})

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := srv.SimulateStatusRead(ctx, h, "wamid.read.1", "628111333")
		if err != nil {
			t.Fatalf("SimulateStatusRead failed: %v", err)
		}

		wg.Wait()

		if capturedStatus.ID != "wamid.read.1" {
			t.Errorf("expected status ID 'wamid.read.1', got %q", capturedStatus.ID)
		}
		if capturedStatus.Status != "read" {
			t.Errorf("expected status 'read', got %q", capturedStatus.Status)
		}
		if capturedStatus.RecipientID != "628111333" {
			t.Errorf("expected recipient '628111333', got %q", capturedStatus.RecipientID)
		}
	})

	t.Run("Failed", func(t *testing.T) {
		h := webhook.NewHandler("verify_tok", srv.AppSecret())

		var capturedStatus webhook.Status
		var wg sync.WaitGroup
		wg.Add(1)

		h.OnStatus(func(ctx context.Context, status webhook.Status, meta webhook.Metadata) error {
			capturedStatus = status
			wg.Done()
			return nil
		})

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := srv.SimulateStatusFailed(ctx, h, "wamid.fail.1", "628111444", 131026, "Message Undeliverable")
		if err != nil {
			t.Fatalf("SimulateStatusFailed failed: %v", err)
		}

		wg.Wait()

		if capturedStatus.ID != "wamid.fail.1" {
			t.Errorf("expected status ID 'wamid.fail.1', got %q", capturedStatus.ID)
		}
		if capturedStatus.Status != "failed" {
			t.Errorf("expected status 'failed', got %q", capturedStatus.Status)
		}
		if capturedStatus.RecipientID != "628111444" {
			t.Errorf("expected recipient '628111444', got %q", capturedStatus.RecipientID)
		}
		if len(capturedStatus.Errors) != 1 || capturedStatus.Errors[0].Code != 131026 || capturedStatus.Errors[0].Title != "Message Undeliverable" {
			t.Errorf("unexpected status errors: %+v", capturedStatus.Errors)
		}
	})
}

func TestSimulateWebhook_CustomPayload(t *testing.T) {
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	h := webhook.NewHandler("verify_tok", srv.AppSecret())

	var capturedCount int32
	var wg sync.WaitGroup
	wg.Add(1)

	h.OnTextMessage(func(ctx context.Context, msg webhook.Message, meta webhook.Metadata) error {
		if msg.Text != nil && msg.Text.Body == "Custom raw payload" {
			atomic.AddInt32(&capturedCount, 1)
		}
		wg.Done()
		return nil
	})

	customPayload := &webhook.Payload{
		Object: "whatsapp_business_account",
		Entry: []webhook.Entry{
			{
				ID: srv.WABAID(),
				Changes: []webhook.Change{
					{
						Field: "messages",
						Value: webhook.Value{
							MessagingProduct: "whatsapp",
							Metadata: webhook.Metadata{
								PhoneNumberID:      srv.PhoneNumberID(),
								DisplayPhoneNumber: "+1 555-0100",
							},
							Messages: []webhook.Message{
								{
									From: "62819999",
									ID:   "wamid.custom.1",
									Type: "text",
									Text: &webhook.Text{Body: "Custom raw payload"},
								},
							},
						},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := srv.SimulateWebhook(ctx, h, customPayload)
	if err != nil {
		t.Fatalf("SimulateWebhook failed with custom payload: %v", err)
	}

	wg.Wait()

	if atomic.LoadInt32(&capturedCount) != 1 {
		t.Errorf("expected 1 message received from custom payload, got %d", capturedCount)
	}
}

func TestSimulateWebhook_SignatureMismatch(t *testing.T) {
	serverSecret := "secret_server"
	handlerSecret := "secret_handler_mismatch"

	srv := wacloudapitest.NewServer(wacloudapitest.WithAppSecret(serverSecret))
	defer srv.Close()

	h := webhook.NewHandler("verify_tok", handlerSecret)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := srv.SimulateInboundText(ctx, h, "6281", "Will fail signature")
	if err == nil {
		t.Fatal("expected error due to signature mismatch, got nil")
	}
}

func TestSimulateWebhook_InvalidTargetsAndPayloads(t *testing.T) {
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	ctx := context.Background()

	// 1. Nil target
	if err := srv.SimulateInboundText(ctx, nil, "123", "test"); err == nil {
		t.Error("expected error with nil target, got nil")
	}

	// 2. Nil payload
	if err := srv.SimulateWebhook(ctx, "http://localhost:1234", nil); err == nil {
		t.Error("expected error with nil payload, got nil")
	}

	// 3. Unsupported target type
	if err := srv.SimulateInboundText(ctx, 12345, "123", "test"); err == nil {
		t.Error("expected error with unsupported target type, got nil")
	}

	// 4. HTTP target URL that returns 404
	badTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer badTS.Close()

	if err := srv.SimulateInboundText(ctx, badTS.URL, "123", "test"); err == nil {
		t.Error("expected error when target returns HTTP 404, got nil")
	}

	// 5. http.Handler returning HTTP 500 error code
	errHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	if err := srv.SimulateInboundText(ctx, errHandler, "123", "test"); err == nil {
		t.Error("expected error when http.Handler returns 500, got nil")
	}

	// 6. Invalid URL string
	if err := srv.SimulateInboundText(ctx, "http://\x7f/bad-url", "123", "test"); err == nil {
		t.Error("expected error with invalid URL string, got nil")
	}

	// 7. Unknown media type fallback
	if err := srv.SimulateInboundMedia(ctx, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), "123", "unknown_media_type", "med_custom"); err != nil {
		t.Errorf("expected success for unknown media type fallback, got: %v", err)
	}

	// 8. *httptest.Server with nil Config.Handler
	bareTS := &httptest.Server{URL: "http://invalid-fake-host-999.test"}
	if err := srv.SimulateInboundText(ctx, bareTS, "123", "test"); err == nil {
		t.Error("expected error for bare httptest.Server with invalid URL, got nil")
	}
}

func TestSimulateWebhook_Concurrency(t *testing.T) {
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	h := webhook.NewHandler("verify_tok", srv.AppSecret())

	var counter int64
	h.OnTextMessage(func(ctx context.Context, msg webhook.Message, meta webhook.Metadata) error {
		atomic.AddInt64(&counter, 1)
		return nil
	})

	var wg sync.WaitGroup
	numRoutines := 30

	for i := 0; i < numRoutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ctx := context.Background()
			_ = srv.SimulateInboundText(ctx, h, "628000", "Concurrent ping")
		}(i)
	}

	wg.Wait()

	// Wait briefly for asynchronous callbacks to finish
	time.Sleep(100 * time.Millisecond)

	if val := atomic.LoadInt64(&counter); val != int64(numRoutines) {
		t.Errorf("expected %d messages received, got %d", numRoutines, val)
	}
}
