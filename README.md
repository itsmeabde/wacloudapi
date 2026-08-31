# Go WhatsApp Cloud API SDK (`wacloudapi`)

[![Go Reference](https://pkg.go.dev/badge/github.com/itsmeabde/wacloudapi.svg)](https://pkg.go.dev/github.com/itsmeabde/wacloudapi)
[![Go Report Card](https://goreportcard.com/badge/github.com/itsmeabde/wacloudapi)](https://goreportcard.com/report/github.com/itsmeabde/wacloudapi)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Package Go idiomatik, modular, aman secara konkuren (*thread-safe*), dan **zero external dependencies** untuk berinteraksi dengan **Meta WhatsApp Cloud API (Graph API v21.0+)**.

## ✨ Fitur Utama
- **Zero External Dependencies**: Dibangun 100% menggunakan Go Standard Library (`net/http`, `crypto/hmac`, dll.).
- **Lengkap & Type-Safe**: Mendukung semua jenis pesan WhatsApp (Teks, Media, Template, Interaktif/Tombol/List, Lokasi, Kontak, Reaksi, Quoted Reply).
- **Media Streaming**: Upload multipart dan download streaming hemat memori (`io.Reader` / `io.ReadCloser`).
- **Dual Webhook Processing**: Event dispatcher siap pakai (`http.Handler`) dengan callback bertipe serta standalone validator signature HMAC-SHA256.
- **Resilient & Structured Errors**: Penanganan error terstruktur (`*APIError`), klasifikasi error (`IsRateLimit()`, `IsAuthError()`), serta retry exponential backoff otomatis.

## 📦 Instalasi

```bash
go get github.com/itsmeabde/wacloudapi
```

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
        wacloudapi.WithRetry(3),
        wacloudapi.WithTimeout(15 * time.Second),
    )

    // Kirim pesan teks sederhana
    res, err := client.Messages.SendText(context.Background(), "628123456789", "Halo dari wacloudapi!")
    if err != nil {
        panic(err)
    }
    fmt.Println("Pesan terkirim dengan ID:", res.Messages[0].ID)
}
```

### Menangani Webhook
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
        fmt.Printf("Pesan dari %s: %s\n", msg.From, msg.Text.Body)
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

## 📚 Lisensi
MIT License.
