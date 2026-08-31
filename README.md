# Go WhatsApp Cloud API SDK (`wacloudapi`)

[![Go Reference](https://pkg.go.dev/badge/github.com/itsmeabde/wacloudapi.svg)](https://pkg.go.dev/github.com/itsmeabde/wacloudapi)
[![Go Report Card](https://goreportcard.com/badge/github.com/itsmeabde/wacloudapi)](https://goreportcard.com/report/github.com/itsmeabde/wacloudapi)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Package Go idiomatik, modular, aman secara konkuren (*thread-safe*), dan **zero external dependencies** untuk berinteraksi dengan **Meta WhatsApp Cloud API (Graph API v21.0+)**.

---

## ✨ Fitur Utama

- **Zero External Dependencies**: Dibangun 100% menggunakan Go Standard Library (`net/http`, `crypto/hmac`, `encoding/json`, dll.).
- **Messages API (Lengkap & Type-Safe)**: Mendukung semua jenis pesan WhatsApp (Teks, Media, Template, Interaktif Tombol & List, Lokasi, Kontak, Reaksi, Quoted Reply).
- **Media API**: Upload multipart hemat memori dan download streaming (`io.Reader` / `io.ReadCloser`).
- **Business Profile Management**: Ambil dan perbarui profil bisnis WhatsApp (About, Address, Description, Email, Websites, Vertical).
- **Message Templates Management**: CRUD lengkap untuk template pesan WhatsApp, validasi status, dan helper cursor pagination (`NextPage`, `PrevPage`).
- **Phone Numbers & 2FA Management**: Manajemen nomor telepon bisnis, registrasi verifikasi 2 langkah (PIN), request & verify OTP (SMS/Voice), dan deregistrasi.
- **Message QR Codes API**: Generate, list, update, hapus QR code dengan pesan prefilled, serta helper streaming download gambar QR code (PNG / SVG).
- **Dual Webhook Processing**: Event dispatcher siap pakai (`http.Handler`) dengan typed callbacks serta standalone validator signature HMAC-SHA256.
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

---

### 2. Business Profile Management

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

### 3. Message Templates Management

```go
// Membuat Template Baru
created, err := client.Templates.Create(ctx, &wacloudapi.CreateTemplateRequest{
	Name:     "order_update_v1",
	Category: wacloudapi.TemplateCategoryUtility,
	Language: "id",
	Components: []wacloudapi.TemplateComponent{
		{
			Type: wacloudapi.ComponentTypeBody,
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

### 4. Phone Numbers & Two-Step Verification (2FA)

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

### 5. Message QR Codes API

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

### 6. Media Management (Upload & Download)

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

### 7. Webhook Handling

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

	h.OnTextMessage(func(ctx context.Context, msg webhook.Message, meta webhook.Metadata) error {
		fmt.Printf("Pesan dari %s (%s): %s\n", meta.DisplayPhoneNumber, msg.From, msg.Text.Body)
		return nil
	})

	h.OnStatus(func(ctx context.Context, status webhook.Status, meta webhook.Metadata) error {
		fmt.Printf("Status pesan %s: %s\n", status.ID, status.Status)
		return nil
	})

	http.Handle("/webhook", h)
	http.ListenAndServe(":8080", nil)
}
```

---

### 8. Penanganan Error Terstruktur

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

## 📂 Contoh Lengkap (Examples)

Contoh kode siap pakai tersedia di direktori [`examples/`](./examples):
- [`examples/send_messages/`](./examples/send_messages) - Mengirim pesan teks, interaktif, dan template.
- [`examples/media_upload/`](./examples/media_upload) - Upload media dan mengirim dokumen/gambar.
- [`examples/business_profile/`](./examples/business_profile) - Mengambil dan memperbarui WhatsApp Business Profile.
- [`examples/templates_management/`](./examples/templates_management) - Manajemen template pesan WABA.
- [`examples/qr_codes/`](./examples/qr_codes) - Pembuatan dan manajemen WhatsApp QR Codes.
- [`examples/webhook_server/`](./examples/webhook_server) - Server HTTP penanganan webhook WhatsApp Cloud API.

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
