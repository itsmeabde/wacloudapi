# Product Requirement Document (PRD) & Technical Design: Go WhatsApp Cloud API SDK

**Module**: `github.com/itsmeabde/wacloudapi`  
**Go Version**: `go 1.22+`  
**Status**: Approved Spec  
**Author**: @itsmeabde & Antigravity  
**Date**: 2026-08-31  

---

## 1. Executive Summary & Goals

### 1.1 Overview
`wacloudapi` adalah library/SDK Go (*Golang*) modern, terstruktur (*modular service-based*), idiomatik, dan *zero external dependencies* untuk berinteraksi dengan **Meta WhatsApp Cloud API (Graph API v21.0+)**.

### 1.2 Key Objectives
- **Developer-Friendly (DX)**: Menyediakan interface yang bersih, autocomplete-friendly, berbasis *functional options*, serta mendukung `context.Context` di setiap pemanggilan.
- **Zero External Dependencies**: Dibangun murni menggunakan Go Standard Library (`net/http`, `crypto/hmac`, `crypto/sha256`, `encoding/json`, `io`, `time`, dll.).
- **Reliable & Resilient**: Memiliki penanganan error terstruktur (`*APIError`), verifikasi signature kriptografis, dan konfigurasi retry otomatis (*exponential backoff*) untuk transient failure.
- **Dual Webhook Processing**: Menyediakan `http.Handler` siap pakai untuk event-driven architecture serta parser mandiri (*standalone parser*) untuk integrasi ke custom HTTP framework (Gin, Fiber, Echo, Chi, dll.).
- **Memory Efficient**: Mendukung streaming upload & download (`io.Reader` / `io.ReadCloser`) untuk file media berukuran besar.

---

## 2. Scope & Feature Requirements (v1.0)

### 2.1 Outbound Messaging (`client.Messages`)
Mendukung pengiriman semua tipe pesan WhatsApp Cloud API:
1. **Text**: Teks biasa dengan opsi URL preview toggle.
2. **Media**: Image, Audio, Video, Document (dengan custom filename), dan Sticker. Mendukung pengiriman melalui **Media ID** (hasil upload) atau **Link URL**.
3. **Template**: Marketing, Utility, dan Authentication templates dengan dynamic parameter replacement (text, currency, date_time) dan interactive button payload.
4. **Interactive Messages**:
   - **Quick Reply Buttons**: Hingga 3 tombol aksi cepat.
   - **List Messages**: Menu dengan multiple sections dan baris pilihan (hingga 10 items).
   - **Call-to-Action (CTA) URL**: Tombol tautan ke website eksternal.
5. **Location**: Koordinat latitude, longitude, nama lokasi, dan alamat.
6. **Contacts**: vCard/Contact struct lengkap (nama, telepon, email, organisasi).
7. **Reaction**: Mengirim reaksi emoji ke pesan tertentu atau menghapus reaksi.
8. **Context / Reply-to**: Mengirim pesan sebagai balasan (*quoted message*) ke pesan tertentu (`wamid`).
9. **Mark as Read**: Menandai pesan masuk sebagai telah dibaca.

### 2.2 Media Management (`client.Media`)
1. **Upload**: Upload file media via `multipart/form-data` dari `io.Reader` atau path lokal, mengembalikan Media ID.
2. **Metadata & URL Lookup**: Mengambil informasi metadata media (MIME type, file size, SHA256 hash, temporary download URL).
3. **Streaming Download**: Mengunduh binary media langsung sebagai `io.ReadCloser` dengan header autentikasi Bearer token.
4. **Buffer Download**: Helper untuk mengunduh media langsung ke `[]byte` untuk file kecil.
5. **Delete**: Menghapus file media dari server WhatsApp Cloud API.

### 2.3 Webhook Handling (`wacloudapi/webhook`)
1. **Challenge Verification (HTTP GET)**: Verifikasi otomatis query string `hub.mode`, `hub.verify_token`, dan mengembalikan `hub.challenge`.
2. **Cryptographic Signature Verification (HTTP POST)**: Validasi `X-Hub-Signature-256` menggunakan HMAC-SHA256 dengan `App Secret`.
3. **Payload Parsing**: Pemetaan JSON webhook Meta ke Go struct bertipe kuat (*strongly-typed*).
4. **Event-Driven Dispatcher (`http.Handler`)**: Hook callback terdaftar:
   - `OnMessage(func(ctx context.Context, msg Message, meta Metadata) error)`
   - `OnTextMessage(func(ctx context.Context, msg Message, meta Metadata) error)`
   - `OnMediaMessage(func(ctx context.Context, msg Message, meta Metadata) error)`
   - `OnInteractiveMessage(func(ctx context.Context, msg Message, meta Metadata) error)`
   - `OnStatus(func(ctx context.Context, status Status, meta Metadata) error)`
   - `OnError(func(ctx context.Context, err error))`
5. **Standalone Parser & Verifier**: Fungsi utilitas `VerifySignature()` dan `ParsePayload()` untuk framework kustom.

### 2.4 Error Handling & Resilience
1. **Structured `APIError`**: Menangkap kode error Meta Graph API (`Code`, `Subcode`, `Message`, `Type`, `FBTraceID`, `HTTPStatusCode`).
2. **Error Classification Helpers**: `apiErr.IsRateLimit()`, `apiErr.IsAuthError()`, `apiErr.IsTemplateError()`.
3. **Exponential Backoff Retry**: Built-in retry untuk HTTP status 429 (Rate Limit) dan 5xx (Server Error) dengan batas maksimal retry yang dapat dikonfigurasi.

---

## 3. Package Structure & Architecture

```text
github.com/itsmeabde/wacloudapi/
├── go.mod
├── client.go              # Inisialisasi Client, Config, & Service references
├── options.go             # Functional Options (WithHTTPClient, WithRetry, WithAPIVersion, dll.)
├── error.go               # APIError, parsing response error Meta Graph API, helper predicates
├── request.go             # Internal HTTP runner, retry backoff logic, header injection
├── messages.go            # MessagesService implementation (Send, SendText, SendImage, dll.)
├── message_models.go      # Request & Response structs untuk messaging
├── media.go               # MediaService implementation (Upload, Get, Download, Delete)
├── media_models.go        # Structs untuk media upload, metadata, dan response
├── webhook/               # Sub-package khusus webhook
│   ├── webhook.go         # Core verification (challenge & signature) & ParsePayload()
│   ├── handler.go         # Standard http.Handler implementation & event dispatcher
│   └── models.go          # Struct event webhook (Payload, Message, Status, Metadata, etc.)
└── examples/              # Runnable examples
    ├── send_messages/     # Contoh kirim text, image, template, interactive buttons
    ├── media_upload/      # Contoh upload dan download media
    └── webhook_server/    # Contoh HTTP server webhook dengan Gin/net-http
```

---

## 4. Detailed Technical Specification

### 4.1 Client & Configuration

```go
package wacloudapi

import (
    "net/http"
    "time"
)

type Config struct {
    AccessToken   string
    PhoneNumberID string
    APIVersion    string        // Default: "v21.0"
    BaseURL       string        // Default: "https://graph.facebook.com"
    HTTPClient    *http.Client  // Default: &http.Client{Timeout: 30 * time.Second}
    MaxRetries    int           // Default: 0
    RetryWaitMin  time.Duration // Default: 500ms
    RetryWaitMax  time.Duration // Default: 5s
}

type Option func(*Config)

func WithAPIVersion(version string) Option
func WithBaseURL(url string) Option
func WithHTTPClient(client *http.Client) Option
func WithTimeout(timeout time.Duration) Option
func WithRetry(maxRetries int, waitMin ...time.Duration) Option

type Client struct {
    config   *Config
    Messages *MessagesService
    Media    *MediaService
}

func New(accessToken, phoneNumberID string, opts ...Option) *Client
```

### 4.2 Error Handling (`error.go`)

```go
type APIError struct {
    HTTPStatusCode int    `json:"-"`
    Message        string `json:"message"`
    Type           string `json:"type"`
    Code           int    `json:"code"`
    ErrorSubcode   int    `json:"error_subcode"`
    ErrorUserTitle string `json:"error_user_title,omitempty"`
    ErrorUserMsg   string `json:"error_user_msg,omitempty"`
    FBTraceID      string `json:"fbtrace_id,omitempty"`
}

func (e *APIError) Error() string
func (e *APIError) IsRateLimit() bool
func (e *APIError) IsAuthError() bool
func (e *APIError) IsTemplateError() bool
```

### 4.3 Messages Service (`messages.go` & `message_models.go`)

#### Message Options
- `WithPreviewURL(preview bool)`
- `WithReplyTo(messageID string)`
- `WithCaption(caption string)`
- `WithFilename(filename string)`

#### Core Interface
```go
type MessagesService struct { /* unexported fields */ }

func (s *MessagesService) Send(ctx context.Context, req *SendMessageRequest) (*SendMessageResponse, error)
func (s *MessagesService) SendText(ctx context.Context, to, text string, opts ...MessageOption) (*SendMessageResponse, error)
func (s *MessagesService) SendImage(ctx context.Context, to string, media MediaSource, opts ...MessageOption) (*SendMessageResponse, error)
func (s *MessagesService) SendAudio(ctx context.Context, to string, media MediaSource) (*SendMessageResponse, error)
func (s *MessagesService) SendVideo(ctx context.Context, to string, media MediaSource, opts ...MessageOption) (*SendMessageResponse, error)
func (s *MessagesService) SendDocument(ctx context.Context, to string, media MediaSource, opts ...MessageOption) (*SendMessageResponse, error)
func (s *MessagesService) SendSticker(ctx context.Context, to string, media MediaSource) (*SendMessageResponse, error)
func (s *MessagesService) SendLocation(ctx context.Context, to string, loc Location, opts ...MessageOption) (*SendMessageResponse, error)
func (s *MessagesService) SendContacts(ctx context.Context, to string, contacts []Contact, opts ...MessageOption) (*SendMessageResponse, error)
func (s *MessagesService) SendReaction(ctx context.Context, to, messageID, emoji string) (*SendMessageResponse, error)
func (s *MessagesService) SendTemplate(ctx context.Context, to string, tpl *TemplateMessage, opts ...MessageOption) (*SendMessageResponse, error)
func (s *MessagesService) SendInteractive(ctx context.Context, to string, interactive *InteractiveMessage, opts ...MessageOption) (*SendMessageResponse, error)
func (s *MessagesService) MarkAsRead(ctx context.Context, messageID string) error
```

#### Media Source Abstraction
```go
type MediaSource struct {
    ID   string
    Link string
}

func MediaByID(id string) MediaSource
func MediaByURL(url string) MediaSource
```

### 4.4 Media Service (`media.go` & `media_models.go`)

```go
type MediaService struct { /* unexported fields */ }

func (s *MediaService) Upload(ctx context.Context, filename string, r io.Reader, mimeType string) (*UploadMediaResponse, error)
func (s *MediaService) UploadFile(ctx context.Context, filePath, mimeType string) (*UploadMediaResponse, error)
func (s *MediaService) Get(ctx context.Context, mediaID string) (*MediaMetadata, error)
func (s *MediaService) Download(ctx context.Context, mediaID string) (io.ReadCloser, *MediaMetadata, error)
func (s *MediaService) DownloadBytes(ctx context.Context, mediaID string) ([]byte, *MediaMetadata, error)
func (s *MediaService) Delete(ctx context.Context, mediaID string) error
```

### 4.5 Webhook Subpackage (`webhook/`)

#### Verification & Standalone Parsers (`webhook.go`)
```go
func VerifyChallenge(w http.ResponseWriter, r *http.Request, verifyToken string) bool
func VerifySignature(body []byte, signatureHeader, appSecret string) error
func ParsePayload(body []byte) (*Payload, error)
```

#### Event Handler (`handler.go`)
```go
type Handler struct { /* unexported fields */ }

func NewHandler(verifyToken, appSecret string) *Handler
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request)

func (h *Handler) OnMessage(fn func(ctx context.Context, msg Message, meta Metadata) error)
func (h *Handler) OnTextMessage(fn func(ctx context.Context, msg Message, meta Metadata) error)
func (h *Handler) OnMediaMessage(fn func(ctx context.Context, msg Message, meta Metadata) error)
func (h *Handler) OnInteractiveMessage(fn func(ctx context.Context, msg Message, meta Metadata) error)
func (h *Handler) OnStatus(fn func(ctx context.Context, status Status, meta Metadata) error)
func (h *Handler) OnError(fn func(ctx context.Context, err error))
```

---

## 5. Non-Functional Requirements & Testing Strategy

### 5.1 Concurrency & Safety
- Semua instance `*Client` dan `*webhook.Handler` aman digunakan secara concurrent (*thread-safe*).
- Menghormati pembatalan (*cancellation*) dan deadline timeout via `context.Context`.

### 5.2 Unit & Integration Testing
- Menggunakan `httptest.Server` untuk mocking respons Meta Graph API.
- Test coverage minimal 90% mencakup:
  - Error parsing dan predicate checks (`IsRateLimit`, `IsAuthError`, dll.).
  - Retry mechanism dengan jitter/exponential backoff.
  - Multipart upload parsing dan streaming download verification.
  - HMAC-SHA256 webhook signature validation (valid, invalid signature, corrupted payload).
  - Webhook GET challenge validation.

---

## 6. Future Roadmap (v2.0+)
- WhatsApp Business Profile API (Update About, Address, Description, Email, Websites, Profile Picture).
- WhatsApp Template Management API (Create, Update, Delete message templates via API).
- Phone Number Registration & Two-Step Verification API.
- QR Code Management API.
