# WhatsApp Flows & Commerce Messages (Phase 3) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Membangun WhatsApp Flows Data Endpoint sub-package (`wacloudapi/flows`) dan dukungan Outbound/Inbound Commerce & Catalog Messages (Single/Multi Product, Orders, Flows).

**Architecture:** Memperluas `MessagesService` dengan helper outbound commerce/flows, memperluas `webhook` dengan order & flow event callbacks, serta membangun sub-package `flows` yang mengimplementasikan protokol enkripsi RSA-OAEP + AES-GCM dan `http.Handler` screen dispatcher.

**Tech Stack:** Go 1.22+ (Zero external dependencies, 100% Go Standard Library: `crypto/rsa`, `crypto/aes`, `crypto/cipher`, `crypto/rand`, `crypto/sha256`, `crypto/x509`, `encoding/pem`, `encoding/base64`, `encoding/json`, `net/http`).

## Global Constraints
- Go Version: `go 1.22+`
- External Dependencies: 0 (Zero external dependencies, only Go standard library)
- Module Name: `github.com/itsmeabde/wacloudapi`
- Default API Version: `v21.0`
- Default Base URL: `https://graph.facebook.com`
- Concurrency: All exported client, handler, and crypto functions must be safe for concurrent goroutine access
- Context: All API network calls must accept and respect `context.Context`

---

### Task 1: Outbound Commerce & Flow Message Models and Methods

**Files:**
- Modify: `message_models.go`
- Modify: `messages.go`
- Modify: `messages_test.go`

**Interfaces:**
- Produces:
  - Models: `ProductSection`, `ProductItem`, `MultiProductRequest`, `OrderDetailsMessage`, `OrderItem`, `OrderPaymentSettings`, `FlowMessageRequest`, `FlowActionPayload`
  - Methods on `MessagesService`:
    - `SendSingleProduct(ctx context.Context, to, catalogID, productRetailerID, body string, opts ...MessageOption) (*SendMessageResponse, error)`
    - `SendMultiProduct(ctx context.Context, to string, req *MultiProductRequest, opts ...MessageOption) (*SendMessageResponse, error)`
    - `SendOrderDetails(ctx context.Context, to string, order *OrderDetailsMessage, opts ...MessageOption) (*SendMessageResponse, error)`
    - `SendFlow(ctx context.Context, to string, req *FlowMessageRequest, opts ...MessageOption) (*SendMessageResponse, error)`

- [ ] **Step 1: Write failing tests for Commerce & Flow outbound messaging in `messages_test.go`**

Add to `messages_test.go`:
```go
func TestMessagesServiceSendSingleProduct(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req SendMessageRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.Type != "interactive" || req.Interactive == nil || req.Interactive.Type != InteractiveTypeProduct {
			t.Errorf("unexpected interactive product type: %+v", req.Interactive)
		}
		if req.Interactive.Action.CatalogID != "cat_123" || req.Interactive.Action.ProductRetailerID != "prod_456" {
			t.Errorf("unexpected product action: %+v", req.Interactive.Action)
		}

		_ = json.NewEncoder(w).Encode(SendMessageResponse{
			MessagingProduct: "whatsapp",
			Messages: []struct {
				ID            string `json:"id"`
				MessageStatus string `json:"message_status,omitempty"`
			}{
				{ID: "wamid.prod123"},
			},
		})
	}))
	defer ts.Close()

	c := New("test-token", "phone-1", WithBaseURL(ts.URL))
	res, err := c.Messages.SendSingleProduct(context.Background(), "628123456789", "cat_123", "prod_456", "Lihat produk pilihan kami")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Messages[0].ID != "wamid.prod123" {
		t.Errorf("unexpected message id: %s", res.Messages[0].ID)
	}
}

func TestMessagesServiceSendMultiProduct(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req SendMessageRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.Interactive.Type != InteractiveTypeProductList {
			t.Errorf("unexpected type: %s", req.Interactive.Type)
		}
		if len(req.Interactive.Action.Sections) != 1 || len(req.Interactive.Action.Sections[0].ProductItems) != 2 {
			t.Errorf("unexpected sections: %+v", req.Interactive.Action.Sections)
		}

		_ = json.NewEncoder(w).Encode(SendMessageResponse{
			MessagingProduct: "whatsapp",
			Messages: []struct {
				ID            string `json:"id"`
				MessageStatus string `json:"message_status,omitempty"`
			}{
				{ID: "wamid.mpm123"},
			},
		})
	}))
	defer ts.Close()

	c := New("test-token", "phone-1", WithBaseURL(ts.URL))
	res, err := c.Messages.SendMultiProduct(context.Background(), "628123456789", &MultiProductRequest{
		CatalogID:   "cat_123",
		HeaderTitle: "Katalog Promo",
		BodyText:    "Pilih produk favorit Anda:",
		Sections: []ProductSection{
			{
				Title: "Kategori Elektronik",
				ProductItems: []ProductItem{
					{ProductRetailerID: "item_1"},
					{ProductRetailerID: "item_2"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Messages[0].ID != "wamid.mpm123" {
		t.Errorf("unexpected message id: %s", res.Messages[0].ID)
	}
}

func TestMessagesServiceSendFlow(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req SendMessageRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.Interactive.Type != InteractiveTypeFlow || req.Interactive.Action.Name != "flow" {
			t.Errorf("unexpected flow action: %+v", req.Interactive)
		}

		_ = json.NewEncoder(w).Encode(SendMessageResponse{
			MessagingProduct: "whatsapp",
			Messages: []struct {
				ID            string `json:"id"`
				MessageStatus string `json:"message_status,omitempty"`
			}{
				{ID: "wamid.flow123"},
			},
		})
	}))
	defer ts.Close()

	c := New("test-token", "phone-1", WithBaseURL(ts.URL))
	res, err := c.Messages.SendFlow(context.Background(), "628123456789", &FlowMessageRequest{
		FlowID:    "flow_999",
		FlowToken: "token_abc",
		FlowCTA:   "Mulai Survey",
		BodyText:  "Silakan isi survey singkat ini",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Messages[0].ID != "wamid.flow123" {
		t.Errorf("unexpected id: %s", res.Messages[0].ID)
	}
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `GOWORK=off go test -v -run "TestMessagesServiceSend(SingleProduct|MultiProduct|Flow)" ./...`
Expected: FAIL.

- [ ] **Step 3: Implement Commerce & Flow message models and methods**

In `message_models.go`, add:
```go
const (
	InteractiveTypeProduct     InteractiveType = "product"
	InteractiveTypeProductList InteractiveType = "product_list"
	InteractiveTypeFlow        InteractiveType = "flow"
	InteractiveTypeOrderDetails InteractiveType = "order_details"
	InteractiveTypeOrderStatus  InteractiveType = "order_status"
)

type ProductItem struct {
	ProductRetailerID string `json:"product_retailer_id"`
}

type ProductSection struct {
	Title        string        `json:"title"`
	ProductItems []ProductItem `json:"product_items"`
}

type MultiProductRequest struct {
	CatalogID   string           `json:"catalog_id"`
	HeaderTitle string           `json:"header_title,omitempty"`
	BodyText    string           `json:"body_text"`
	FooterText  string           `json:"footer_text,omitempty"`
	Sections    []ProductSection `json:"sections"`
}

type FlowMessageRequest struct {
	FlowID             string             `json:"flow_id"`
	FlowToken          string             `json:"flow_token"`
	FlowCTA            string             `json:"flow_cta"`
	FlowAction         string             `json:"flow_action,omitempty"` // Default: "navigate"
	FlowMode           string             `json:"flow_mode,omitempty"`   // "draft" or "published"
	FlowMessageVersion string             `json:"flow_message_version,omitempty"` // Default: "3"
	Header             *InteractiveHeader `json:"header,omitempty"`
	BodyText           string             `json:"body_text"`
	FooterText         string             `json:"footer_text,omitempty"`
	ActionPayload      *FlowActionPayload `json:"action_payload,omitempty"`
}

type FlowActionPayload struct {
	Screen string      `json:"screen,omitempty"`
	Data   interface{} `json:"data,omitempty"`
}

type FlowParameters struct {
	Mode               string             `json:"mode,omitempty"`
	FlowMessageVersion string             `json:"flow_message_version,omitempty"`
	FlowToken          string             `json:"flow_token"`
	FlowID             string             `json:"flow_id"`
	FlowCTA            string             `json:"flow_cta"`
	FlowAction         string             `json:"flow_action,omitempty"`
	FlowActionPayload  *FlowActionPayload `json:"flow_action_payload,omitempty"`
}

// Extend InteractiveAction to include CatalogID, ProductRetailerID, and ProductSections
```

In `messages.go`, implement:
- `SendSingleProduct(ctx, to, catalogID, productRetailerID, body, opts ...MessageOption)`
- `SendMultiProduct(ctx, to, req *MultiProductRequest, opts ...MessageOption)`
- `SendOrderDetails(ctx, to, order *OrderDetailsMessage, opts ...MessageOption)`
- `SendFlow(ctx, to, req *FlowMessageRequest, opts ...MessageOption)`

- [ ] **Step 4: Run test to verify it passes**

Run: `GOWORK=off go test -v ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add message_models.go messages.go messages_test.go
git commit -m "feat: add outbound Commerce (Single/Multi Product, Order) and WhatsApp Flows message helpers"
```

---

### Task 2: Webhook Commerce & Flow Response Models and Callbacks

**Files:**
- Modify: `webhook/models.go`
- Modify: `webhook/handler.go`
- Modify: `webhook/webhook_test.go`
- Modify: `webhook/handler_test.go`

**Interfaces:**
- Produces:
  - Models: `Order`, `OrderItem`, `NFMReply` in `webhook`
  - Callback Registrations on `*webhook.Handler`:
    - `OnOrderMessage(fn func(ctx context.Context, order Order, msg Message, meta Metadata) error)`
    - `OnFlowResponseMessage(fn func(ctx context.Context, reply NFMReply, msg Message, meta Metadata) error)`

- [ ] **Step 1: Write failing tests for Webhook Inbound Order & NFMReply parsing and dispatching**

Add tests to `webhook/webhook_test.go` and `webhook/handler_test.go` covering order and flow responses.

- [ ] **Step 2: Run test to verify failure**

Run: `GOWORK=off go test -v ./webhook/...`
Expected: FAIL.

- [ ] **Step 3: Implement Webhook models and handlers for Order & Flow response**

In `webhook/models.go`:
- Add `Order` struct (`CatalogID`, `Text`, `ProductItems: []OrderItem{ ProductRetailerID, Quantity, ItemPrice, Currency }`).
- Add `NFMReply` struct (`Name`, `Body`, `ResponseJSON`).
- Attach `Order *Order` to `Message` and `NFMReply *NFMReply` to `Interactive`.

In `webhook/handler.go`:
- Add `onOrderMessage []OrderHandler` and `onFlowReply []FlowReplyHandler`.
- Implement `OnOrderMessage(fn)` and `OnFlowResponseMessage(fn)`.
- Dispatch in `dispatchEvents` thread-safely with error propagation.

- [ ] **Step 4: Run test to verify it passes**

Run: `GOWORK=off go test -v ./webhook/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add webhook/models.go webhook/handler.go webhook/webhook_test.go webhook/handler_test.go
git commit -m "feat: add webhook event models and handlers for Inbound Orders and Flow NFM replies"
```

---

### Task 3: WhatsApp Flows Cryptography & Key Loader

**Files:**
- Create: `flows/models.go`
- Create: `flows/crypto.go`
- Test: `flows/crypto_test.go`

**Interfaces:**
- Produces:
  - Models: `EncryptedPayload`, `Request`, `Response`, `CryptoSession`
  - Functions:
    - `ParsePrivateKey(pemBytes []byte, passphrase ...string) (*rsa.PrivateKey, error)`
    - `LoadPrivateKeyFile(path string, passphrase ...string) (*rsa.PrivateKey, error)`
    - `DecryptRequest(payload *EncryptedPayload, key *rsa.PrivateKey) (*Request, *CryptoSession, error)`
    - `EncryptResponse(resp *Response, session *CryptoSession) (string, error)`

- [ ] **Step 1: Write failing tests for RSA-OAEP + AES-GCM Flow decryption & response encryption**

Create `flows/crypto_test.go` testing generating test RSA keypair, encrypting mock payload, decrypting with `DecryptRequest`, and verifying `EncryptResponse` with inverted IV.

- [ ] **Step 2: Run test to verify failure**

Run: `GOWORK=off go test -v ./flows/...`
Expected: FAIL.

- [ ] **Step 3: Implement Flow models, RSA/AES-GCM crypto, and key parsers**

Create `flows/models.go` and `flows/crypto.go` with complete PKCS#1 / PKCS#8 parser, RSA-OAEP SHA-256 decryption, AES-128-GCM decryption, byte-inversion IV (`iv[i] ^ 0xFF`), and AES-GCM response encryption.

- [ ] **Step 4: Run test to verify it passes**

Run: `GOWORK=off go test -v ./flows/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add flows/models.go flows/crypto.go flows/crypto_test.go
git commit -m "feat: implement WhatsApp Flows RSA-OAEP and AES-GCM cryptography and key loader"
```

---

### Task 4: WhatsApp Flows HTTP Handler & Screen/Action Dispatcher

**Files:**
- Create: `flows/handler.go`
- Test: `flows/handler_test.go`

**Interfaces:**
- Produces:
  - `Handler`, `NewHandler(key *rsa.PrivateKey) *Handler`
  - `NewHandlerFromPEM(pemBytes []byte, passphrase ...string) (*Handler, error)`
  - `NewHandlerFromKeyFile(path string, passphrase ...string) (*Handler, error)`
  - `(h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request)`
  - `(h *Handler) HandleScreen(screenID string, fn ScreenHandler)`
  - `(h *Handler) OnAction(fn ActionHandler)`
  - `(h *Handler) OnPing(fn PingHandler)`
  - `(h *Handler) OnError(fn ErrorHandler)`

- [ ] **Step 1: Write failing tests for Flows HTTP Handler**

Create `flows/handler_test.go` testing POST encrypted request, screen routing (`INIT`, `data_exchange`), ping handling, error handling, and thread safety.

- [ ] **Step 2: Run test to verify failure**

Run: `GOWORK=off go test -v ./flows/...`
Expected: FAIL.

- [ ] **Step 3: Implement WhatsApp Flows HTTP Handler**

Create `flows/handler.go` implementing `http.Handler`, decrypting incoming requests, invoking matching screen or general action callbacks, encrypting responses, and writing base64 strings with HTTP 200.

- [ ] **Step 4: Run test to verify it passes**

Run: `GOWORK=off go test -v ./flows/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add flows/handler.go flows/handler_test.go
git commit -m "feat: implement WhatsApp Flows HTTP handler and screen/action dispatcher"
```

---

### Task 5: Examples, Documentation & Full Verification

**Files:**
- Create: `examples/commerce_messages/main.go`
- Create: `examples/flows_endpoint/main.go`
- Modify: `README.md`

**Interfaces:**
- Consumes: All features from Tasks 1-4

- [ ] **Step 1: Create runnable examples for Commerce & Flows**

Create `examples/commerce_messages/main.go` and `examples/flows_endpoint/main.go`.
Update `README.md` with complete documentation on Commerce messages and WhatsApp Flows data endpoint setup.

- [ ] **Step 2: Run full test suite with -race, vet, and build examples**

Run: `GOWORK=off go test -v -race -cover ./...`
Run: `GOWORK=off go vet ./...`
Run: `GOWORK=off go build ./examples/...`
Expected: ALL PASS.

- [ ] **Step 3: Commit**

```bash
git add examples/ README.md
git commit -m "docs: add Phase 3 examples, update README with WhatsApp Flows and Commerce guides, and verify full test suite"
```
