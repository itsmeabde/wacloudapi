# Product Requirement Document (PRD) & Technical Design: WhatsApp Flows & Commerce (Phase 3)

**Module**: `github.com/itsmeabde/wacloudapi`  
**Go Version**: `go 1.22+`  
**Status**: Approved Spec  
**Author**: @itsmeabde & Antigravity  
**Date**: 2026-08-31  

---

## 1. Executive Summary & Goals

Melanjutkan kapabilitas v1.0 (Messaging, Media, Webhooks) dan v2.0 (Business Management APIs), **Fase 3** melengkapi ekosistem `wacloudapi` dengan fitur **Interactive Commerce & WhatsApp Flows**:
1. **Outbound Commerce & Catalog Messages**: Single Product Messages, Multi-Product Messages (MPM), Order Details Messages, dan Outbound Flow Messages.
2. **Inbound Webhook Events**: Parsing event pesanan masuk (*inbound order*) dan respons Flow (*nfm_reply*), dilengkapi handler callback bertipe.
3. **WhatsApp Flows Data Endpoint Sub-package (`wacloudapi/flows`)**: Server endpoint siap pakai untuk mendekripsi payload enkripsi Meta (RSA-OAEP SHA-256 + AES-128-GCM), routing action/screen, dan mengenkripsi kembali respons Flow.

Prinsip inti yang tetap dipertahankan:
- **Zero External Dependencies** (100% Go Standard Library: `crypto/rsa`, `crypto/aes`, `crypto/cipher`, `crypto/x509`, `encoding/pem`, `net/http`, dll.).
- **Idiomatic Go & Type Safety**.
- **Resilient & Thread-Safe**.

---

## 2. Scope & Feature Requirements

### 2.1 Outbound Commerce & Flow Messages (`client.Messages`)
- **Single Product Message**:
  `SendSingleProduct(ctx, to, catalogID, productRetailerID, body, opts ...MessageOption)`
- **Multi-Product Message (MPM)**:
  `SendMultiProduct(ctx, to, req *MultiProductRequest, opts ...MessageOption)`
- **Order Details Message**:
  `SendOrderDetails(ctx, to, order *OrderDetailsMessage, opts ...MessageOption)`
- **Flow Message**:
  `SendFlow(ctx, to, req *FlowMessageRequest, opts ...MessageOption)`

### 2.2 Inbound Webhook Events (`wacloudapi/webhook`)
- **Order Event**:
  Mendukung pesan tipe `order` (`msg.Order` dengan `CatalogID`, `Text`, `ProductItems` [ProductRetailerID, Quantity, ItemPrice, Currency]).
- **Flow Response Event**:
  Mendukung pesan interaktif tipe `nfm_reply` (`msg.Interactive.NFMReply` dengan `Name`, `Body`, `ResponseJSON`).
- **Callback Registration**:
  - `OnOrderMessage(func(ctx context.Context, order Order, msg Message, meta Metadata) error)`
  - `OnFlowResponseMessage(func(ctx context.Context, reply NFMReply, msg Message, meta Metadata) error)`

### 2.3 WhatsApp Flows Data Endpoint (`wacloudapi/flows`)
- **Kriptografi RSA & AES-GCM**:
  - Dekripsi `encrypted_aes_key` dengan RSA-OAEP SHA-256 menggunakan RSA Private Key bisnis.
  - Dekripsi `encrypted_flow_data` dengan AES-128-GCM menggunakan `initial_vector`.
  - Enkripsi response payload dengan AES-128-GCM menggunakan `flipped_iv` (inversi byte per byte `iv[i] ^ 0xFF`).
- **Handler Initialization**:
  - `NewHandler(key *rsa.PrivateKey) *Handler`
  - `NewHandlerFromPEM(pemBytes []byte, passphrase ...string) (*Handler, error)`
  - `NewHandlerFromKeyFile(path string, passphrase ...string) (*Handler, error)`
- **Screen & Action Routing**:
  - `HandleScreen(screenID string, fn func(ctx context.Context, req *Request) (*Response, error))`
  - `OnAction(fn func(ctx context.Context, req *Request) (*Response, error))`
  - `OnPing(fn func(ctx context.Context, req *Request) (*Response, error))`
  - `OnError(fn func(ctx context.Context, err error))`
- **Standalone Helpers**:
  - `DecryptRequest(encrypted *EncryptedPayload, key *rsa.PrivateKey) (*Request, *CryptoSession, error)`
  - `EncryptResponse(resp *Response, session *CryptoSession) (string, error)`

---

## 3. Package Structure

```text
github.com/itsmeabde/wacloudapi/
├── message_models.go       # Extended dengan Commerce & Flow outbound models
├── messages.go             # Extended dengan SendSingleProduct, SendMultiProduct, SendFlow, dll.
├── webhook/
│   ├── models.go           # Extended dengan Order & NFMReply structs
│   └── handler.go          # Extended dengan OnOrderMessage & OnFlowResponseMessage
├── flows/                  # Sub-package khusus WhatsApp Flows Data Endpoint
│   ├── crypto.go           # RSA-OAEP + AES-GCM decrypt/encrypt & key loader
│   ├── handler.go          # http.Handler implementation, action/screen router
│   ├── models.go           # Request & Response structs for WhatsApp Flows
│   └── flows_test.go       # Unit & integration tests untuk Flows encryption & routing
└── examples/
    ├── commerce_messages/  # Contoh kirim single/multi product & order details
    └── flows_endpoint/     # Contoh HTTP endpoint server WhatsApp Flows
```
