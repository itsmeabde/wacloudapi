# Go WhatsApp Cloud API SDK (`wacloudapi`)

[![Go Reference](https://pkg.go.dev/badge/github.com/itsmeabde/wacloudapi.svg)](https://pkg.go.dev/github.com/itsmeabde/wacloudapi)
[![Go Report Card](https://goreportcard.com/badge/github.com/itsmeabde/wacloudapi)](https://goreportcard.com/report/github.com/itsmeabde/wacloudapi)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Package Go idiomatik, modular, aman secara konkuren (*thread-safe*), dan **zero external dependencies** untuk berinteraksi dengan **Meta WhatsApp Cloud API (Graph API v21.0+)**.

---

## ✨ Fitur Utama

- **Zero External Dependencies**: Dibangun 100% menggunakan Go Standard Library (`net/http`, `crypto/rsa`, `crypto/aes`, `crypto/hmac`, `encoding/json`, dll.).
- **Messages API (Lengkap & Type-Safe)**: Mendukung semua jenis pesan WhatsApp (Teks, Media, Template, Interaktif Tombol & List, Lokasi, Kontak, Reaksi, Quoted Reply).
- **Commerce & Catalog Messages**: Pengiriman pesan katalog Single Product, Multi-Product Messages (MPM), dan Order Details Messages.
- **WhatsApp Flows Outbound & Data Endpoint (`wacloudapi/flows`)**: Pengiriman form interaktif WhatsApp Flows serta server data endpoint lengkap dengan dekripsi RSA-OAEP (SHA-256) + AES-128-GCM, enkripsi response terbalik (*inverted IV*), routing screen/action bertipe, dan *health check* (`ping`).
- **Media API**: Upload multipart hemat memori dan download streaming (`io.Reader` / `io.ReadCloser`).
- **Business Profile Management**: Ambil dan perbarui profil bisnis WhatsApp (About, Address, Description, Email, Websites, Vertical).
- **Message Templates Management**: CRUD lengkap untuk template pesan WhatsApp, validasi status, dan helper cursor pagination (`NextPage`, `PrevPage`).
- **Phone Numbers & 2FA Management**: Manajemen nomor telepon bisnis, registrasi verifikasi 2 langkah (PIN), request & verify OTP (SMS/Voice), dan deregistrasi.
- **Message QR Codes API**: Generate, list, update, hapus QR code dengan pesan prefilled, serta helper streaming download gambar QR code (PNG / SVG).
- **Dual Webhook Processing**: Event dispatcher siap pakai (`http.Handler`) dengan typed callbacks (Teks, Media, Interaktif, Order, Flow Response, Status Update) serta standalone validator signature HMAC-SHA256.
- **Mock Testing & Webhook Simulator Kit (`wacloudapi/wacloudapitest`)**: Server mock in-memory lengkap tanpa external dependencies untuk unit & integration testing, perekaman/asersi pesan keluar (`SentMessages`), chaos engineering (`InjectRateLimit`, `InjectServerError`), custom route mocking, dan simulasi webhook inbound (Teks, Media, Order, Flow Response, Status Delivery/Read/Failed) bertanda tangan HMAC-SHA256.
- **Resilient & Structured Errors**: Penanganan error terstruktur (`*APIError`), klasifikasi error (`IsRateLimit()`, `IsAuthError()`), serta retry exponential backoff otomatis.

---

## 📦 Instalasi

```bash
go get github.com/itsmeabde/wacloudapi
```

---

## 🚀 Quick Start

### Inisialisasi Client

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/itsmeabde/wacloudapi"
)

func main() {
	client := wacloudapi.New("ACCESS_TOKEN", "PHONE_NUMBER_ID",
		wacloudapi.WithWABAID("WABA_ACCOUNT_ID"), // Dibutuhkan untuk Templates & Phone Numbers List
		wacloudapi.WithRetry(3),
		wacloudapi.WithTimeout(15*time.Second),
	)

	// Kirim pesan teks sederhana
	res, err := client.Messages.SendText(context.Background(), "628123456789", "Halo dari wacloudapi!")
	if err != nil {
		panic(err)
	}
	fmt.Println("Pesan terkirim dengan ID:", res.Messages[0].ID)
}
```

---

## 📖 Panduan Penggunaan

### 1. Mengirim Pesan (Messages API)

#### Pesan Interaktif (Tombol / Button Reply)
```go
interactive := &wacloudapi.InteractiveMessage{
	Type: wacloudapi.InteractiveTypeButton,
	Body: wacloudapi.InteractiveBody{Text: "Apakah Anda ingin melanjutkan konfirmasi pesanan?"},
	Action: wacloudapi.InteractiveAction{
		Buttons: []wacloudapi.ButtonAction{
			{Type: "reply", Reply: wacloudapi.ButtonReply{ID: "btn_yes", Title: "Ya, Setuju"}},
			{Type: "reply", Reply: wacloudapi.ButtonReply{ID: "btn_no", Title: "Batalkan"}},
		},
	},
}
res, err := client.Messages.SendInteractive(ctx, "628123456789", interactive)
```

#### Pesan Media (Gambar, Dokumen, Audio, Video)
```go
res, err := client.Messages.SendDocument(ctx, "628123456789",
	wacloudapi.MediaByURL("https://example.com/invoice.pdf"),
	wacloudapi.WithCaption("Berikut tagihan Anda"),
	wacloudapi.WithFilename("Invoice-2026.pdf"),
)
```

#### Pesan Commerce: Single Product
```go
res, err := client.Messages.SendSingleProduct(ctx, "628123456789", "CATALOG_ID", "PRODUCT_RETAILER_ID", "Lihat penawaran produk unggulan kami!")
```

#### Pesan Commerce: Multi-Product Message (MPM)
```go
res, err := client.Messages.SendMultiProduct(ctx, "628123456789", &wacloudapi.MultiProductRequest{
	CatalogID:   "CATALOG_ID",
	HeaderTitle: "Katalog Promo Spesial",
	BodyText:    "Pilih produk kebutuhan Anda:",
	Sections: []wacloudapi.ProductSection{
		{
			Title: "Elektronik",
			ProductItems: []wacloudapi.ProductItem{
				{ProductRetailerID: "prod_phone_1"},
				{ProductRetailerID: "prod_laptop_2"},
			},
		},
	},
})
```

#### Pesan WhatsApp Flow Outbound
```go
res, err := client.Messages.SendFlow(ctx, "628123456789", &wacloudapi.FlowMessageRequest{
	FlowID:             "FLOW_ID",
	FlowToken:          "token_survey_123",
	FlowCTA:            "Mulai Survey",
	FlowAction:         "navigate",
	FlowMode:           "published",
	FlowMessageVersion: "3",
	BodyText:           "Silakan isi survey kepuasan pelanggan melalui tombol di bawah.",
	ActionPayload: &wacloudapi.FlowActionPayload{
		Screen: "SURVEY_START",
		Data: map[string]interface{}{
			"customer_id": "cust_12345",
		},
	},
})
```

---

### 2. WhatsApp Flows Data Endpoint (`wacloudapi/flows`)

Sub-package `wacloudapi/flows` menyediakan HTTP handler siap pakai untuk mengelola endpoint data dinamis WhatsApp Flows yang otomatis menangani enkripsi/dekripsi RSA-OAEP SHA-256 dan AES-128-GCM:

```go
package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/itsmeabde/wacloudapi/flows"
)

func main() {
	// Inisialisasi handler dari RSA Private Key (PEM format)
	pemBytes, _ := os.ReadFile("private_key.pem")
	h, err := flows.NewHandlerFromPEM(pemBytes)
	if err != nil {
		log.Fatalf("Failed to create flows handler: %v", err)
	}

	// 1. Daftarkan handler per screen
	h.HandleScreen("APPOINTMENT", func(ctx context.Context, req *flows.Request) (*flows.Response, error) {
		log.Printf("Received screen request for %s with data: %+v", req.Screen, req.Data)

		return &flows.Response{
			Screen: "CONFIRMATION",
			Data: map[string]interface{}{
				"appointment_id": "APT-12345",
				"date":           "2026-09-01",
				"time":           "14:00",
				"status":         "confirmed",
			},
		}, nil
	})

	// 2. Health check endpoint (ping dari Meta)
	h.OnPing(func(ctx context.Context, req *flows.Request) (*flows.Response, error) {
		return &flows.Response{
			Data: map[string]interface{}{
				"status": "active",
			},
		}, nil
	})

	// 3. Tangani error dekripsi atau pemrosesan internal
	h.OnError(func(ctx context.Context, err error) {
		log.Printf("[Flow Handler Error]: %v", err)
	})

	http.Handle("/flows", h)
	http.ListenAndServe(":8080", nil)
}
```

---

### 3. Business Profile Management

```go
// Mengambil profil bisnis
profile, err := client.BusinessProfile.Get(ctx)
if err != nil {
	log.Fatal(err)
}
fmt.Printf("About: %s, Vertical: %s\n", profile.About, profile.Vertical)

// Memperbarui profil bisnis
err = client.BusinessProfile.Update(ctx, &wacloudapi.UpdateBusinessProfileRequest{
	About:       "Official Customer Support",
	Description: "Hubungi kami untuk pertanyaan dan bantuan seputar layanan.",
	Vertical:    wacloudapi.VerticalRetail,
	Websites:    []string{"https://example.com"},
	Email:       "support@example.com",
})
```

---

### 4. Message Templates Management

```go
// Membuat Template Baru
created, err := client.Templates.Create(ctx, &wacloudapi.CreateTemplateRequest{
	Name:     "order_update_v1",
	Category: wacloudapi.TemplateCategoryUtility,
	Language: "id",
	Components: []wacloudapi.TemplateComponent{
		{
			Type: "BODY",
			Text: "Halo {{1}}, pesanan {{2}} sedang dalam pengiriman.",
		},
	},
})

// Menampilkan Daftar Template dengan Pagination
list, err := client.Templates.List(ctx, &wacloudapi.ListTemplatesRequest{
	Limit: 10,
})
for _, tpl := range list.Data {
	fmt.Printf("- %s [%s] (%s)\n", tpl.Name, tpl.Category, tpl.Status)
}

// Navigasi halaman berikutnya
if list.Paging.Cursors.After != "" {
	nextList, err := client.Templates.NextPage(ctx, list)
	// ...
}
```

---

### 5. Phone Numbers & Two-Step Verification (2FA)

```go
// List semua nomor telepon di WABA
numbers, err := client.PhoneNumbers.List(ctx)

// Request kode verifikasi (SMS atau Voice)
err = client.PhoneNumbers.RequestCode(ctx, "PHONE_NUMBER_ID", &wacloudapi.RequestVerificationCodeRequest{
	CodeMethod: wacloudapi.CodeMethodSMS,
	Language:   "id",
})

// Verifikasi kode OTP
err = client.PhoneNumbers.VerifyCode(ctx, "PHONE_NUMBER_ID", &wacloudapi.VerifyCodeRequest{
	Code: "123456",
})

// Set 6-digit PIN Two-Step Verification
err = client.PhoneNumbers.SetTwoStepVerification(ctx, "PHONE_NUMBER_ID", &wacloudapi.SetTwoStepVerificationRequest{
	PIN: "654321",
})
```

---

### 6. Message QR Codes API

```go
// Buat QR Code dengan pesan pre-filled
qr, err := client.QRCodes.Create(ctx, &wacloudapi.CreateQRCodeRequest{
	PrefilledMessage: "Halo, saya tertarik dengan promo spesial hari ini!",
	ImageFormat:      wacloudapi.QRCodeImagePNG,
})
fmt.Printf("Deep Link: %s\n", qr.DeepLinkURL)

// Download gambar QR Code secara streaming
stream, err := client.QRCodes.DownloadImage(ctx, qr.QRImageURL)
if err != nil {
	log.Fatal(err)
}
defer stream.Close()

out, _ := os.Create("qrcode.png")
defer out.Close()
io.Copy(out, stream)
```

---

### 7. Media Management (Upload & Download)

```go
// Upload file
file, _ := os.Open("invoice.pdf")
defer file.Close()

media, err := client.Media.Upload(ctx, "invoice.pdf", file, "application/pdf")
fmt.Println("Uploaded Media ID:", media.ID)

// Download file streaming
stream, meta, err := client.Media.Download(ctx, media.ID)
if err != nil {
	log.Fatal(err)
}
defer stream.Close()
```

---

### 8. Webhook Handling (Inbound Messages, Orders & Flows)

```go
package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/itsmeabde/wacloudapi/webhook"
)

func main() {
	h := webhook.NewHandler("VERIFY_TOKEN", "APP_SECRET")

	// 1. Text Message Callback
	h.OnTextMessage(func(ctx context.Context, msg webhook.Message, meta webhook.Metadata) error {
		fmt.Printf("Pesan dari %s (%s): %s\n", meta.DisplayPhoneNumber, msg.From, msg.Text.Body)
		return nil
	})

	// 2. Inbound Order Callback (Katalog Pesanan Masuk)
	h.OnOrderMessage(func(ctx context.Context, order webhook.Order, msg webhook.Message, meta webhook.Metadata) error {
		fmt.Printf("Pesanan baru dari %s, Catalog: %s, Items: %d\n", msg.From, order.CatalogID, len(order.ProductItems))
		for _, item := range order.ProductItems {
			fmt.Printf("  - Item: %s, Qty: %d, Harga: %d %s\n", item.ProductRetailerID, item.Quantity, item.ItemPrice, item.Currency)
		}
		return nil
	})

	// 3. WhatsApp Flow Response Callback (nfm_reply)
	h.OnFlowResponseMessage(func(ctx context.Context, reply webhook.NFMReply, msg webhook.Message, meta webhook.Metadata) error {
		fmt.Printf("Respons Flow dari %s (Name: %s): %s\n", msg.From, reply.Name, reply.ResponseJSON)
		return nil
	})

	// 4. Status Update Callback
	h.OnStatus(func(ctx context.Context, status webhook.Status, meta webhook.Metadata) error {
		fmt.Printf("Status pesan %s: %s\n", status.ID, status.Status)
		return nil
	})

	http.Handle("/webhook", h)
	http.ListenAndServe(":8080", nil)
}
```

---

### 9. Penanganan Error Terstruktur

```go
res, err := client.Messages.SendText(ctx, "628123456789", "Test")
if err != nil {
	if apiErr, ok := err.(*wacloudapi.APIError); ok {
		fmt.Printf("API Error Code: %d, Message: %s\n", apiErr.Code, apiErr.Message)
		if apiErr.IsRateLimit() {
			fmt.Println("Terkena rate limit, tunggu beberapa saat.")
		}
		if apiErr.IsAuthError() {
			fmt.Println("Token akses tidak valid atau kedaluwarsa.")
		}
	}
}
```

---

### 10. Mock Testing & Webhook Simulator (`wacloudapi/wacloudapitest`)

Sub-package `wacloudapi/wacloudapitest` menyediakan mock test server in-memory lengkap tanpa external dependencies untuk menguji bot WhatsApp, webhook handler, dan alur integrasi aplikasi Anda secara lokal dan deterministik tanpa memerlukan koneksi internet atau kredensial riil Meta.

#### A. Membuat Mock Server & Client

```go
package main_test

import (
	"context"
	"testing"

	"github.com/itsmeabde/wacloudapi/wacloudapitest"
)

func TestBot_SendMessage(t *testing.T) {
	// 1. Inisialisasi mock server
	srv := wacloudapitest.NewServer(
		wacloudapitest.WithPhoneNumberID("10001"),
		wacloudapitest.WithWABAID("20002"),
		wacloudapitest.WithAppSecret("test_app_secret"),
	)
	defer srv.Close()

	// 2. Buat client yang terhubung ke server mock
	client := srv.Client()

	// 3. Eksekusi pengiriman pesan
	res, err := client.Messages.SendText(context.Background(), "628123456789", "Halo dari mock!")
	if err != nil {
		t.Fatalf("Gagal mengirim pesan: %v", err)
	}

	// 4. Asersi pesan keluar yang tercatat di server
	if len(srv.SentMessages()) != 1 {
		t.Fatalf("Harus tercatat 1 pesan terkirim")
	}
	last := srv.LastSentMessage()
	if last.Text.Body != "Halo dari mock!" {
		t.Errorf("Teks pesan tidak sesuai: %s", last.Text.Body)
	}
}
```

#### B. Asersi & Verifikasi Pesan Keluar

Server mock menyimpan seluruh payload permintaan pengiriman pesan yang masuk ke memory:

```go
// Mengambil seluruh pesan yang terkirim ke mock server
allMessages := srv.SentMessages()

// Mengambil pesan terakhir yang dikirim
lastMessage := srv.LastSentMessage()

// Mengambil seluruh pesan yang dikirim ke nomor penerima tertentu
userMessages := srv.SentMessagesTo("628123456789")

// Mengambil media yang tersimpan di mock server
mediaMap := srv.UploadedMedia()
```

#### C. Chaos Testing & Error Injection

Uji ketahanan bot atau retry logic dengan menginjeksi error Meta API atau HTTP error:

```go
// Injeksi N respons HTTP 429 Rate Limit (Kode 130429)
srv.InjectRateLimit(2)

// Injeksi N respons HTTP 500 Internal Server Error
srv.InjectServerError(1)

// Injeksi error Meta kustom (misal Token Expired 190, Invalid Template 132000)
srv.InjectMetaError(190, "OAuthException: Error validating access token", 1)

// Mendaftarkan mock route kustom
srv.SetCustomHandler("GET /custom_endpoint", func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success":true}`))
})

// Reset seluruh state, queue error, dan custom handler
srv.Reset()
```

#### D. Simulasi Webhook Inbound (HMAC-SHA256 Signed)

Simulasikan pesan atau status update dari WhatsApp ke `http.Handler` atau HTTP server URL Anda lengkap dengan signature `X-Hub-Signature-256` yang valid:

```go
h := webhook.NewHandler("VERIFY_TOKEN", srv.AppSecret())

// Simulasi pesan teks masuk dari pelanggan
err := srv.SimulateInboundText(ctx, h, "628123456789", "Tolong info produk")

// Simulasi media masuk (image, audio, video, document, sticker)
err := srv.SimulateInboundMedia(ctx, h, "628123456789", "image", "mock_media_123")

// Simulasi pesanan katalog masuk (inbound order)
err := srv.SimulateInboundOrder(ctx, h, "628123456789", "CATALOG_1", []webhook.OrderItem{
	{ProductRetailerID: "PROD_1", Quantity: 2, ItemPrice: 50000, Currency: "IDR"},
})

// Simulasi respons Flow interaktif (nfm_reply)
err := srv.SimulateInboundFlowResponse(ctx, h, "628123456789", "FLOW_TOK", `{"screen":"APPOINTMENT","date":"2026-09-01"}`)

// Simulasi status update (delivered, read, failed)
err := srv.SimulateStatusDelivered(ctx, h, "wamid.mock.123", "628123456789")
err := srv.SimulateStatusRead(ctx, h, "wamid.mock.123", "628123456789")
err := srv.SimulateStatusFailed(ctx, h, "wamid.mock.123", "628123456789", 131056, "Recipient not registered")
```

---

## 📂 Contoh Lengkap (Examples)

Contoh kode siap pakai tersedia di direktori [`examples/`](./examples):
- [`examples/send_messages/`](./examples/send_messages) - Mengirim pesan teks, interaktif, dan template.
- [`examples/commerce_messages/`](./examples/commerce_messages) - Mengirim pesan katalog (Single Product, Multi-Product MPM) & WhatsApp Flows.
- [`examples/flows_endpoint/`](./examples/flows_endpoint) - Implementasi server WhatsApp Flows Data Endpoint terenkripsi (RSA/AES-GCM).
- [`examples/media_upload/`](./examples/media_upload) - Upload media dan mengirim dokumen/gambar.
- [`examples/business_profile/`](./examples/business_profile) - Mengambil dan memperbarui WhatsApp Business Profile.
- [`examples/templates_management/`](./examples/templates_management) - Manajemen template pesan WABA.
- [`examples/qr_codes/`](./examples/qr_codes) - Pembuatan dan manajemen WhatsApp QR Codes.
- [`examples/webhook_server/`](./examples/webhook_server) - Server HTTP penanganan webhook WhatsApp Cloud API (Teks, Media, Order, Flow).
- [`examples/mock_testing/`](./examples/mock_testing) - Pengujian unit bot WhatsApp dengan mock server, asersi pesan keluar, chaos engineering, dan simulasi webhook inbound (`wacloudapitest`).

---

## 🧪 Testing

Jalankan pengujian unit dan integration:

```bash
# Menjalankan seluruh test suite dengan race detector dan coverage
go test -v -race -cover ./...

# Menjalankan static analysis
go vet ./...

# Memastikan seluruh contoh aplikasi dapat dikompilasi
go build ./examples/...
```

---

## 📚 Lisensi

Didistribusikan di bawah lisensi MIT. Lihat `LICENSE` untuk informasi lebih lanjut.
