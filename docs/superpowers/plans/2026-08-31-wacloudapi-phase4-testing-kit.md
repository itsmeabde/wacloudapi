# Mock Testing Kit & Simulator Server (Phase 4) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Membangun sub-package `wacloudapitest` sebagai in-memory mock simulator Meta WhatsApp Cloud API dan Webhook event generator untuk testing end-to-end tanpa kredensial sungguhan.

**Architecture:** Membangun `wacloudapitest.Server` yang membungkus `httptest.Server`, mengelola in-memory state seluruh modul WhatsApp Cloud API v21.0+, menyediakan error injection queue, dan menyertakan webhook event dispatcher bertanda tangan HMAC-SHA256.

**Tech Stack:** Go 1.22+ (Zero external dependencies, 100% Go Standard Library: `net/http/httptest`, `crypto/hmac`, `crypto/sha256`, `crypto/rand`, `sync`, `encoding/json`, `encoding/hex`, `testing`).

## Global Constraints
- Go Version: `go 1.22+`
- External Dependencies: 0 (Zero external dependencies, only Go standard library)
- Module Name: `github.com/itsmeabde/wacloudapi/wacloudapitest`
- Default API Version: `v21.0`
- Concurrency: All exported mock server methods and state access must be guarded by mutex for concurrent testing
- Test Coverage: >85% statement coverage across new package with zero race conditions

---

### Task 1: `wacloudapitest` Core Server, State & Options

**Files:**
- Create: `wacloudapitest/options.go`
- Create: `wacloudapitest/state.go`
- Create: `wacloudapitest/server.go`
- Test: `wacloudapitest/server_test.go`

**Interfaces:**
- Produces:
  - `Server`, `Option`, `WithPhoneNumberID`, `WithWABAID`, `WithAPIVersion`, `WithAppSecret`, `WithAccessToken`
  - `NewServer(opts ...Option) *Server`
  - `(s *Server) URL() string`
  - `(s *Server) Close()`
  - `(s *Server) Client() *wacloudapi.Client`
  - `(s *Server) Reset()`

- [ ] **Step 1: Write failing tests for Server lifecycle and Client creation in `server_test.go`**

Add tests for creating server, getting configured client, verifying base URL and tokens, and closing server.

- [ ] **Step 2: Run test to verify failure**

Run: `GOWORK=off go test -v ./wacloudapitest/...`
Expected: FAIL.

- [ ] **Step 3: Implement options, state, and server core**

Implement `options.go`, `state.go`, and `server.go`.

- [ ] **Step 4: Run test to verify it passes**

Run: `GOWORK=off go test -v ./wacloudapitest/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add wacloudapitest/options.go wacloudapitest/state.go wacloudapitest/server.go wacloudapitest/server_test.go
git commit -m "feat: implement wacloudapitest core Server, in-memory state, and configuration options"
```

---

### Task 2: In-Memory API Handlers (Messages, Media, Profile, Templates, Phones, QRCodes)

**Files:**
- Create: `wacloudapitest/handlers.go`
- Modify: `wacloudapitest/server.go`
- Test: `wacloudapitest/handlers_test.go`

**Interfaces:**
- Produces:
  - Handlers for:
    - `POST /{phone_number_id}/messages` (records to `s.sentMessages`, returns `wamid.mock.xxx`)
    - `POST /{phone_number_id}/media` (records binary data to `s.mediaStore`, returns `mock_media_xxx`)
    - `GET /{media_id}` and `GET /{media_id}/download` (serves stored media bytes)
    - `GET/POST /{phone_number_id}/whatsapp_business_profile`
    - `GET/POST/DELETE /{waba_id}/message_templates`
    - `GET /{waba_id}/phone_numbers`, `POST /{phone_id}/register`, `verify_code`, `request_code`, `deregister`
    - `GET/POST/DELETE /{phone_number_id}/message_qrdls` (and QR image serving)
  - Assertions:
    - `(s *Server) SentMessages() []*wacloudapi.SendMessageRequest`
    - `(s *Server) LastSentMessage() *wacloudapi.SendMessageRequest`
    - `(s *Server) SentMessagesTo(phone string) []*wacloudapi.SendMessageRequest`
    - `(s *Server) UploadedMedia() map[string]*MockMedia`

- [ ] **Step 1: Write failing tests in `handlers_test.go`**

Test sending messages, uploading/downloading media, creating/listing templates, managing profile, and querying assertions.

- [ ] **Step 2: Run test to verify failure**

Run: `GOWORK=off go test -v ./wacloudapitest/...`
Expected: FAIL.

- [ ] **Step 3: Implement in-memory API handlers in `handlers.go`**

Implement routing and request processing for all Graph API endpoints.

- [ ] **Step 4: Run test to verify it passes**

Run: `GOWORK=off go test -v ./wacloudapitest/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add wacloudapitest/handlers.go wacloudapitest/server.go wacloudapitest/handlers_test.go
git commit -m "feat: implement in-memory mock handlers for Messages, Media, Profile, Templates, PhoneNumbers, and QRCodes"
```

---

### Task 3: Error Injection & Chaos Testing

**Files:**
- Create: `wacloudapitest/chaos.go`
- Modify: `wacloudapitest/server.go`
- Test: `wacloudapitest/chaos_test.go`

**Interfaces:**
- Produces:
  - `(s *Server) InjectRateLimit(count int)`
  - `(s *Server) InjectServerError(count int)`
  - `(s *Server) InjectMetaError(code int, message string, count int)`
  - `(s *Server) SetCustomHandler(pattern string, handler http.HandlerFunc)`

- [ ] **Step 1: Write failing tests in `chaos_test.go`**

Test rate limit injection, server error injection, and retry recovery on caller side.

- [ ] **Step 2: Run test to verify failure**

Run: `GOWORK=off go test -v ./wacloudapitest/...`
Expected: FAIL.

- [ ] **Step 3: Implement chaos engine and error injection in `chaos.go`**

Implement FIFO error injection queue, status overrides, and custom handler map.

- [ ] **Step 4: Run test to verify it passes**

Run: `GOWORK=off go test -v ./wacloudapitest/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add wacloudapitest/chaos.go wacloudapitest/server.go wacloudapitest/chaos_test.go
git commit -m "feat: implement Chaos Engine and Error Injection in wacloudapitest"
```

---

### Task 4: Webhook Event Simulator & HMAC Generator

**Files:**
- Create: `wacloudapitest/webhook_simulator.go`
- Test: `wacloudapitest/webhook_simulator_test.go`

**Interfaces:**
- Produces:
  - `(s *Server) SimulateWebhook(ctx context.Context, target interface{}, payload *webhook.Payload) error`
  - `(s *Server) SimulateInboundText(ctx context.Context, target interface{}, from, text string) error`
  - `(s *Server) SimulateInboundMedia(ctx context.Context, target interface{}, from, mediaType, mediaID string) error`
  - `(s *Server) SimulateInboundOrder(ctx context.Context, target interface{}, from, catalogID string, items []webhook.OrderItem) error`
  - `(s *Server) SimulateInboundFlowResponse(ctx context.Context, target interface{}, from, flowToken, responseJSON string) error`
  - `(s *Server) SimulateStatusDelivered(ctx context.Context, target interface{}, messageID, recipientID string) error`
  - `(s *Server) SimulateStatusRead(ctx context.Context, target interface{}, messageID, recipientID string) error`
  - `(s *Server) SimulateStatusFailed(ctx context.Context, target interface{}, messageID, recipientID string, code int, title string) error`

- [ ] **Step 1: Write failing tests in `webhook_simulator_test.go`**

Test sending simulated webhook events against `*webhook.Handler` and HTTP test server URL.

- [ ] **Step 2: Run test to verify failure**

Run: `GOWORK=off go test -v ./wacloudapitest/...`
Expected: FAIL.

- [ ] **Step 3: Implement webhook event generator and HMAC signer in `webhook_simulator.go`**

Implement event construction, signature computation (`sha256=<hex>`), and dispatching to URL string or `http.Handler`.

- [ ] **Step 4: Run test to verify it passes**

Run: `GOWORK=off go test -v ./wacloudapitest/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add wacloudapitest/webhook_simulator.go wacloudapitest/webhook_simulator_test.go
git commit -m "feat: implement Webhook Simulator with HMAC-SHA256 signing in wacloudapitest"
```

---

### Task 5: Examples, Documentation & Full Verification

**Files:**
- Create: `examples/mock_testing/main.go`
- Modify: `README.md`

**Interfaces:**
- Consumes: All features from Tasks 1-4

- [ ] **Step 1: Implement example and update README**

Create `examples/mock_testing/main.go` demonstrating how to write unit tests for a WhatsApp bot with `wacloudapitest`. Update `README.md` with complete documentation for `wacloudapitest`.

- [ ] **Step 2: Run full test suite with -race, vet, and build examples**

Run: `GOWORK=off go test -v -race -cover ./...`
Run: `GOWORK=off go vet ./...`
Run: `GOWORK=off go build ./examples/...`
Expected: ALL PASS.

- [ ] **Step 3: Commit**

```bash
git add examples/ README.md
git commit -m "docs: add mock_testing example, update README with wacloudapitest guide, and verify full test suite"
```
