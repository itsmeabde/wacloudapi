# Product Requirement Document (PRD) & Technical Design: Mock Testing Kit & Simulator Server (Phase 4)

**Module**: `github.com/itsmeabde/wacloudapi/wacloudapitest`  
**Go Version**: `go 1.22+`  
**Status**: Approved Spec  
**Author**: @itsmeabde & Antigravity  
**Date**: 2026-08-31  

---

## 1. Executive Summary & Goals

Untuk mempermudah pengembang membangun dan menguji bot, webhook backend, dan integrasi WhatsApp Cloud API tanpa memerlukan akun Meta Developer aktif, kredensial sungguhan, atau koneksi internet, **Fase 4** menghadirkan **`wacloudapitest`**:
1. **In-Memory Meta Cloud API Simulator Server (`wacloudapitest.Server`)**: Mensimulasikan endpoint Graph API v21.0+ untuk Messages, Media, Business Profile, Templates, Phone Numbers, dan QR Codes.
2. **Recorded Requests & State Assertions**: Menyediakan method inspeksi riwayat request (`SentMessages()`, `LastSentMessage()`, `SentMessagesTo()`, `UploadedMedia()`).
3. **Chaos Testing & Error Injection**: Memungkinkan simulasi skenario kegagalan (`InjectRateLimit()`, `InjectServerError()`, `InjectMetaError()`, `SetCustomHandler()`).
4. **Webhook Event Generator with HMAC-SHA256**: Mengirim event webhook simulasi (Text, Media, Order, Flow Reply, Message Status) lengkap dengan signature kriptografis yang valid ke target `http.Handler` atau URL server.

Prinsip inti:
- **Zero External Dependencies** (100% Go Standard Library: `net/http/httptest`, `crypto/hmac`, `crypto/sha256`, `sync`, `encoding/json`, `testing`).
- **Idiomatic Go & Type-Safe**.
- **Thread-Safe & Race-Clean**.

---

## 2. API Surface & Architecture

### 2.1 Inisialisasi Server Mock
```go
// Membuat mock server baru dengan opsi default atau kustom
srv := wacloudapitest.NewServer(
    wacloudapitest.WithPhoneNumberID("10001"),
    wacloudapitest.WithWABAID("waba_20002"),
    wacloudapitest.WithAppSecret("test_app_secret"),
)
defer srv.Close()

// Mendapatkan client wacloudapi yang sudah terkonfigurasi ke mock server
client := srv.Client()
```

### 2.2 Endpoint yang Disimulasikan
- `POST /{phone_number_id}/messages`: Menerima pesan (teks, media, template, interactive, SPM, MPM, Flow, order) -> mencatat ke memory -> return `{"messaging_product": "whatsapp", "messages": [{"id": "wamid.mock.xxx"}]}`.
- `POST /{phone_number_id}/media`: Menerima upload multipart -> menyimpan file bytes di memory -> return `{"id": "mock_media_xxx"}`.
- `GET /{media_id}` & Download: Mengembalikan metadata dan melayani unduhan byte file asli.
- `GET/POST /{phone_number_id}/whatsapp_business_profile`: Mengambil & memperbarui profil bisnis in-memory.
- `GET/POST/DELETE /{waba_id}/message_templates`: Simulasi template CRUD dengan cursor pagination.
- `GET /{waba_id}/phone_numbers` & `POST /{phone_id}/register`, `verify_code`, `request_code`: Simulasi registrasi nomor & 2FA PIN.
- `GET/POST/DELETE /{phone_number_id}/message_qrdls`: Simulasi QR code CRUD & penyajian gambar QR SVG/PNG.

### 2.3 Error Injection & Chaos Testing
```go
// Simulasi HTTP 429 Rate Limit untuk 2 request pertama
srv.InjectRateLimit(2)

// Simulasi Meta API Error khusus (misal error code 130429 rate limit) untuk 1 request
srv.InjectMetaError(130429, "Rate limit reached", 1)

// Simulasi Internal Server Error (HTTP 500)
srv.InjectServerError(1)

// Custom handler override
srv.SetCustomHandler("/v21.0/custom-endpoint", customHandlerFunc)
```

### 2.4 Webhook Event Simulator
```go
// Kirim simulasi pesan teks masuk ke handler / server webhook
err := srv.SimulateInboundText(ctx, webhookTargetURL, "628123456789", "Halo bot!")

// Kirim simulasi status pesan (delivered, read, failed)
err := srv.SimulateStatusDelivered(ctx, webhookTargetURL, "wamid.mock.123", "628123456789")
err := srv.SimulateStatusRead(ctx, webhookTargetURL, "wamid.mock.123", "628123456789")

// Kirim simulasi event order belanja
err := srv.SimulateInboundOrder(ctx, webhookTargetURL, "628123456789", "cat_123", items)

// Kirim simulasi event flow response
err := srv.SimulateInboundFlowResponse(ctx, webhookTargetURL, "628123456789", "token_abc", responseJSON)
```

---

## 3. Struktur Package

```text
github.com/itsmeabde/wacloudapi/
├── wacloudapitest/
│   ├── server.go            # Server struct, httptest.Server wrapper, routing multiplexer
│   ├── options.go           # Server options (WithPhoneNumberID, WithWABAID, WithAppSecret)
│   ├── handlers.go          # In-memory handlers untuk Messages, Media, Templates, Profile, dll.
│   ├── state.go             # In-memory data store (messages history, media map, templates list)
│   ├── chaos.go             # Error injection queue & rate limit simulator
│   ├── webhook_simulator.go # Webhook event generator dengan HMAC-SHA256 signer
│   └── server_test.go       # Unit & integration tests untuk testing kit
└── examples/
    └── mock_testing/        # Contoh cara unit-testing bot menggunakan wacloudapitest
```
