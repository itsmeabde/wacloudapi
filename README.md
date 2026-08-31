# Go WhatsApp Cloud API SDK (`wacloudapi`)

[![Go Reference](https://pkg.go.dev/badge/github.com/itsmeabde/wacloudapi.svg)](https://pkg.go.dev/github.com/itsmeabde/wacloudapi)
[![Go Report Card](https://goreportcard.com/badge/github.com/itsmeabde/wacloudapi)](https://goreportcard.com/report/github.com/itsmeabde/wacloudapi)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

An idiomatic, modular, thread-safe, and **zero external dependencies** Go package for interacting with the **Meta WhatsApp Cloud API (Graph API v21.0+)**.

---

## ✨ Key Features

- **Zero External Dependencies**: Built 100% using the Go Standard Library (`net/http`, `crypto/rsa`, `crypto/aes`, `crypto/hmac`, `encoding/json`, etc.).
- **Messages API (Comprehensive & Type-Safe)**: Supports all WhatsApp message types (Text, Media, Templates, Interactive Buttons & Lists, Location, Contacts, Reactions, Quoted Replies).
- **Commerce & Catalog Messages**: Send Single Product messages, Multi-Product Messages (MPM), and Order Details Messages.
- **WhatsApp Flows Outbound & Data Endpoint (`wacloudapi/flows`)**: Send interactive WhatsApp Flows forms and a complete data endpoint server featuring RSA-OAEP (SHA-256) + AES-128-GCM decryption, inverted IV response encryption, typed screen/action routing, and health checks (`ping`).
- **Media API**: Memory-efficient multipart uploads and streaming downloads (`io.Reader` / `io.ReadCloser`).
- **Business Profile Management**: Retrieve and update WhatsApp business profile details (About, Address, Description, Email, Websites, Vertical).
- **Message Templates Management**: Complete CRUD for WhatsApp message templates, status validation, and cursor pagination helpers (`NextPage`, `PrevPage`).
- **Phone Numbers & 2FA Management**: Manage business phone numbers, two-step verification (PIN) setup, OTP request & verification (SMS/Voice), and deregistration.
- **Message QR Codes API**: Generate, list, update, and delete QR codes with prefilled messages, plus streaming download helpers for QR code images (PNG / SVG).
- **Dual Webhook Processing**: Ready-to-use event dispatcher (`http.Handler`) with typed callbacks (Text, Media, Interactive, Order, Flow Response, Status Update) and standalone HMAC-SHA256 signature verification.
- **Mock Testing & Webhook Simulator Kit (`wacloudapi/wacloudapitest`)**: Complete in-memory mock server with zero external dependencies for unit & integration testing, outbound message recording/assertions (`SentMessages`), chaos engineering (`InjectRateLimit`, `InjectServerError`), custom route mocking, and inbound webhook simulation (Text, Media, Order, Flow Response, Delivery/Read/Failed status updates) signed with HMAC-SHA256.
- **Resilient & Structured Errors**: Structured error handling (`*APIError`), error classification (`IsRateLimit()`, `IsAuthError()`), and automatic exponential backoff retries.

---

## 📦 Installation

```bash
go get github.com/itsmeabde/wacloudapi
```

---

## 🚀 Quick Start

### Client Initialization

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
		wacloudapi.WithWABAID("WABA_ACCOUNT_ID"), // Required for Templates & Phone Numbers List
		wacloudapi.WithRetry(3),
		wacloudapi.WithTimeout(15*time.Second),
	)

	// Send a simple text message
	res, err := client.Messages.SendText(context.Background(), "628123456789", "Hello from wacloudapi!")
	if err != nil {
		panic(err)
	}
	fmt.Println("Message sent with ID:", res.Messages[0].ID)
}
```

---

## 📖 Usage Guide

### 1. Sending Messages (Messages API)

#### Interactive Messages (Button Reply)
```go
interactive := &wacloudapi.InteractiveMessage{
	Type: wacloudapi.InteractiveTypeButton,
	Body: wacloudapi.InteractiveBody{Text: "Would you like to proceed with your order confirmation?"},
	Action: wacloudapi.InteractiveAction{
		Buttons: []wacloudapi.ButtonAction{
			{Type: "reply", Reply: wacloudapi.ButtonReply{ID: "btn_yes", Title: "Yes, Confirm"}},
			{Type: "reply", Reply: wacloudapi.ButtonReply{ID: "btn_no", Title: "Cancel"}},
		},
	},
}
res, err := client.Messages.SendInteractive(ctx, "628123456789", interactive)
```

#### Media Messages (Image, Document, Audio, Video)
```go
res, err := client.Messages.SendDocument(ctx, "628123456789",
	wacloudapi.MediaByURL("https://example.com/invoice.pdf"),
	wacloudapi.WithCaption("Here is your invoice"),
	wacloudapi.WithFilename("Invoice-2026.pdf"),
)
```

#### Commerce Messages: Single Product
```go
res, err := client.Messages.SendSingleProduct(ctx, "628123456789", "CATALOG_ID", "PRODUCT_RETAILER_ID", "Check out our featured product offer!")
```

#### Commerce Messages: Multi-Product Message (MPM)
```go
res, err := client.Messages.SendMultiProduct(ctx, "628123456789", &wacloudapi.MultiProductRequest{
	CatalogID:   "CATALOG_ID",
	HeaderTitle: "Special Promo Catalog",
	BodyText:    "Select items from our catalog:",
	Sections: []wacloudapi.ProductSection{
		{
			Title: "Electronics",
			ProductItems: []wacloudapi.ProductItem{
				{ProductRetailerID: "prod_phone_1"},
				{ProductRetailerID: "prod_laptop_2"},
			},
		},
	},
})
```

#### Outbound WhatsApp Flow Messages
```go
res, err := client.Messages.SendFlow(ctx, "628123456789", &wacloudapi.FlowMessageRequest{
	FlowID:             "FLOW_ID",
	FlowToken:          "token_survey_123",
	FlowCTA:            "Start Survey",
	FlowAction:         "navigate",
	FlowMode:           "published",
	FlowMessageVersion: "3",
	BodyText:           "Please complete our customer satisfaction survey using the button below.",
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

The `wacloudapi/flows` sub-package provides a ready-to-use HTTP handler for managing dynamic WhatsApp Flows data endpoints, automatically handling RSA-OAEP SHA-256 and AES-128-GCM encryption/decryption:

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
	// Initialize handler from RSA Private Key (PEM format)
	pemBytes, _ := os.ReadFile("private_key.pem")
	h, err := flows.NewHandlerFromPEM(pemBytes)
	if err != nil {
		log.Fatalf("Failed to create flows handler: %v", err)
	}

	// 1. Register handler per screen
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

	// 2. Health check endpoint (ping from Meta)
	h.OnPing(func(ctx context.Context, req *flows.Request) (*flows.Response, error) {
		return &flows.Response{
			Data: map[string]interface{}{
				"status": "active",
			},
		}, nil
	})

	// 3. Handle decryption or internal processing errors
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
// Retrieve business profile
profile, err := client.BusinessProfile.Get(ctx)
if err != nil {
	log.Fatal(err)
}
fmt.Printf("About: %s, Vertical: %s\n", profile.About, profile.Vertical)

// Update business profile
err = client.BusinessProfile.Update(ctx, &wacloudapi.UpdateBusinessProfileRequest{
	About:       "Official Customer Support",
	Description: "Contact us for questions and support regarding our services.",
	Vertical:    wacloudapi.VerticalRetail,
	Websites:    []string{"https://example.com"},
	Email:       "support@example.com",
})
```

---

### 4. Message Templates Management

```go
// Create New Template
created, err := client.Templates.Create(ctx, &wacloudapi.CreateTemplateRequest{
	Name:     "order_update_v1",
	Category: wacloudapi.TemplateCategoryUtility,
	Language: "en_US",
	Components: []wacloudapi.TemplateComponent{
		{
			Type: "BODY",
			Text: "Hello {{1}}, your order {{2}} is on its way.",
		},
	},
})

// List Templates with Pagination
list, err := client.Templates.List(ctx, &wacloudapi.ListTemplatesRequest{
	Limit: 10,
})
for _, tpl := range list.Data {
	fmt.Printf("- %s [%s] (%s)\n", tpl.Name, tpl.Category, tpl.Status)
}

// Navigate to the next page
if list.Paging.Cursors.After != "" {
	nextList, err := client.Templates.NextPage(ctx, list)
	// ...
}
```

---

### 5. Phone Numbers & Two-Step Verification (2FA)

```go
// List all phone numbers in WABA
numbers, err := client.PhoneNumbers.List(ctx)

// Request verification code (SMS or Voice)
err = client.PhoneNumbers.RequestCode(ctx, "PHONE_NUMBER_ID", &wacloudapi.RequestVerificationCodeRequest{
	CodeMethod: wacloudapi.CodeMethodSMS,
	Language:   "en_US",
})

// Verify OTP code
err = client.PhoneNumbers.VerifyCode(ctx, "PHONE_NUMBER_ID", &wacloudapi.VerifyCodeRequest{
	Code: "123456",
})

// Set 6-digit PIN for Two-Step Verification
err = client.PhoneNumbers.SetTwoStepVerification(ctx, "PHONE_NUMBER_ID", &wacloudapi.SetTwoStepVerificationRequest{
	PIN: "654321",
})
```

---

### 6. Message QR Codes API

```go
// Create QR Code with a pre-filled message
qr, err := client.QRCodes.Create(ctx, &wacloudapi.CreateQRCodeRequest{
	PrefilledMessage: "Hi, I'm interested in today's special offer!",
	ImageFormat:      wacloudapi.QRCodeImagePNG,
})
fmt.Printf("Deep Link: %s\n", qr.DeepLinkURL)

// Stream download QR Code image
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

// Download file via streaming
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
		fmt.Printf("Message from %s (%s): %s\n", meta.DisplayPhoneNumber, msg.From, msg.Text.Body)
		return nil
	})

	// 2. Inbound Order Callback (Incoming Catalog Orders)
	h.OnOrderMessage(func(ctx context.Context, order webhook.Order, msg webhook.Message, meta webhook.Metadata) error {
		fmt.Printf("New order from %s, Catalog: %s, Items: %d\n", msg.From, order.CatalogID, len(order.ProductItems))
		for _, item := range order.ProductItems {
			fmt.Printf("  - Item: %s, Qty: %d, Price: %d %s\n", item.ProductRetailerID, item.Quantity, item.ItemPrice, item.Currency)
		}
		return nil
	})

	// 3. WhatsApp Flow Response Callback (nfm_reply)
	h.OnFlowResponseMessage(func(ctx context.Context, reply webhook.NFMReply, msg webhook.Message, meta webhook.Metadata) error {
		fmt.Printf("Flow response from %s (Name: %s): %s\n", msg.From, reply.Name, reply.ResponseJSON)
		return nil
	})

	// 4. Status Update Callback
	h.OnStatus(func(ctx context.Context, status webhook.Status, meta webhook.Metadata) error {
		fmt.Printf("Message status %s: %s\n", status.ID, status.Status)
		return nil
	})

	http.Handle("/webhook", h)
	http.ListenAndServe(":8080", nil)
}
```

---

### 9. Structured Error Handling

```go
res, err := client.Messages.SendText(ctx, "628123456789", "Test")
if err != nil {
	if apiErr, ok := err.(*wacloudapi.APIError); ok {
		fmt.Printf("API Error Code: %d, Message: %s\n", apiErr.Code, apiErr.Message)
		if apiErr.IsRateLimit() {
			fmt.Println("Rate limit exceeded, please retry after backoff.")
		}
		if apiErr.IsAuthError() {
			fmt.Println("Access token is invalid or expired.")
		}
	}
}
```

---

### 10. Mock Testing & Webhook Simulator (`wacloudapi/wacloudapitest`)

The `wacloudapi/wacloudapitest` sub-package provides a comprehensive, in-memory mock test server with zero external dependencies to test WhatsApp bots, webhook handlers, and integration flows locally and deterministically without requiring an internet connection or real Meta credentials.

#### A. Creating Mock Server & Client

```go
package main_test

import (
	"context"
	"testing"

	"github.com/itsmeabde/wacloudapi/wacloudapitest"
)

func TestBot_SendMessage(t *testing.T) {
	// 1. Initialize mock server
	srv := wacloudapitest.NewServer(
		wacloudapitest.WithPhoneNumberID("10001"),
		wacloudapitest.WithWABAID("20002"),
		wacloudapitest.WithAppSecret("test_app_secret"),
	)
	defer srv.Close()

	// 2. Create client connected to mock server
	client := srv.Client()

	// 3. Execute message sending
	res, err := client.Messages.SendText(context.Background(), "628123456789", "Hello from mock!")
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// 4. Assert recorded outbound messages on server
	if len(srv.SentMessages()) != 1 {
		t.Fatalf("Expected 1 sent message")
	}
	last := srv.LastSentMessage()
	if last.Text.Body != "Hello from mock!" {
		t.Errorf("Message text mismatch: %s", last.Text.Body)
	}
}
```

#### B. Assertions & Outbound Message Verification

The mock server records all incoming message delivery request payloads in memory:

```go
// Retrieve all messages sent to the mock server
allMessages := srv.SentMessages()

// Retrieve the last sent message
lastMessage := srv.LastSentMessage()

// Retrieve all messages sent to a specific recipient phone number
userMessages := srv.SentMessagesTo("628123456789")

// Retrieve stored uploaded media on mock server
mediaMap := srv.UploadedMedia()
```

#### C. Chaos Testing & Error Injection

Test bot resilience or retry logic by injecting Meta API errors or HTTP server errors:

```go
// Inject N HTTP 429 Rate Limit responses (Code 130429)
srv.InjectRateLimit(2)

// Inject N HTTP 500 Internal Server Error responses
srv.InjectServerError(1)

// Inject custom Meta API error (e.g. Token Expired 190, Invalid Template 132000)
srv.InjectMetaError(190, "OAuthException: Error validating access token", 1)

// Register custom mock route
srv.SetCustomHandler("GET /custom_endpoint", func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success":true}`))
})

// Reset all state, queued errors, and custom handlers
srv.Reset()
```

#### D. Inbound Webhook Simulation (HMAC-SHA256 Signed)

Simulate incoming messages or status updates from WhatsApp to your `http.Handler` or HTTP server URL complete with a valid `X-Hub-Signature-256` header:

```go
h := webhook.NewHandler("VERIFY_TOKEN", srv.AppSecret())

// Simulate inbound text message from customer
err := srv.SimulateInboundText(ctx, h, "628123456789", "Please provide product info")

// Simulate inbound media (image, audio, video, document, sticker)
err := srv.SimulateInboundMedia(ctx, h, "628123456789", "image", "mock_media_123")

// Simulate inbound catalog order
err := srv.SimulateInboundOrder(ctx, h, "628123456789", "CATALOG_1", []webhook.OrderItem{
	{ProductRetailerID: "PROD_1", Quantity: 2, ItemPrice: 50000, Currency: "IDR"},
})

// Simulate interactive Flow response (nfm_reply)
err := srv.SimulateInboundFlowResponse(ctx, h, "628123456789", "FLOW_TOK", `{"screen":"APPOINTMENT","date":"2026-09-01"}`)

// Simulate status updates (delivered, read, failed)
err := srv.SimulateStatusDelivered(ctx, h, "wamid.mock.123", "628123456789")
err := srv.SimulateStatusRead(ctx, h, "wamid.mock.123", "628123456789")
err := srv.SimulateStatusFailed(ctx, h, "wamid.mock.123", "628123456789", 131056, "Recipient not registered")
```

---

## 📂 Examples

Ready-to-use code examples are available in the [`examples/`](./examples) directory:
- [`examples/send_messages/`](./examples/send_messages) - Sending text, interactive, and template messages.
- [`examples/commerce_messages/`](./examples/commerce_messages) - Sending catalog messages (Single Product, Multi-Product MPM) & WhatsApp Flows.
- [`examples/flows_endpoint/`](./examples/flows_endpoint) - Implementing an encrypted WhatsApp Flows Data Endpoint server (RSA/AES-GCM).
- [`examples/media_upload/`](./examples/media_upload) - Uploading media and sending documents/images.
- [`examples/business_profile/`](./examples/business_profile) - Retrieving and updating WhatsApp Business Profile.
- [`examples/templates_management/`](./examples/templates_management) - Managing WABA message templates.
- [`examples/qr_codes/`](./examples/qr_codes) - Creating and managing WhatsApp QR Codes.
- [`examples/webhook_server/`](./examples/webhook_server) - HTTP server for processing WhatsApp Cloud API webhooks (Text, Media, Order, Flow).
- [`examples/mock_testing/`](./examples/mock_testing) - Unit testing WhatsApp bots with mock server, outbound message assertions, chaos engineering, and inbound webhook simulation (`wacloudapitest`).

---

## 🧪 Testing

Run unit and integration tests:

```bash
# Run entire test suite with race detector and coverage
go test -v -race -cover ./...

# Run static analysis
go vet ./...

# Ensure all example applications compile successfully
go build ./examples/...
```

---

## 📚 License

Distributed under the MIT License. See `LICENSE` for more information.
