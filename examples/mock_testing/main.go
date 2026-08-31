package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/itsmeabde/wacloudapi"
	"github.com/itsmeabde/wacloudapi/wacloudapitest"
	"github.com/itsmeabde/wacloudapi/webhook"
)

func main() {
	ctx := context.Background()

	fmt.Println("================================================================")
	fmt.Println("🚀 WhatsApp Cloud API Mock Testing Demo (`wacloudapitest`)")
	fmt.Println("================================================================")

	// -------------------------------------------------------------------------
	// 1. Initialize In-Memory Mock Server
	// -------------------------------------------------------------------------
	const appSecret = "demo_meta_app_secret_123"
	srv := wacloudapitest.NewServer(
		wacloudapitest.WithPhoneNumberID("10001"),
		wacloudapitest.WithWABAID("20002"),
		wacloudapitest.WithAppSecret(appSecret),
	)
	defer srv.Close()

	fmt.Printf("[1] Mock server running at: %s\n", srv.URL())
	fmt.Printf("    - Phone Number ID: %s\n", srv.PhoneNumberID())
	fmt.Printf("    - WABA ID:         %s\n\n", srv.WABAID())

	// Obtain a pre-configured SDK client that routes to the mock server in-memory
	client := srv.Client()

	// -------------------------------------------------------------------------
	// 2. Scenario 1: Outbound Messaging & Verification Assertions
	// -------------------------------------------------------------------------
	fmt.Println("[2] Testing Outbound Messages & Request Assertions...")

	recipient := "628123456789"

	// Send Text Message
	resText, err := client.Messages.SendText(ctx, recipient, "Hello from mock testing!")
	if err != nil {
		log.Fatalf("Failed to send text: %v", err)
	}
	fmt.Printf("    ✓ Sent text message (ID: %s)\n", resText.Messages[0].ID)

	// Send Interactive Button Message
	interactive := &wacloudapi.InteractiveMessage{
		Type: wacloudapi.InteractiveTypeButton,
		Body: wacloudapi.InteractiveBody{Text: "Please choose an option:"},
		Action: wacloudapi.InteractiveAction{
			Buttons: []wacloudapi.ButtonAction{
				{Type: "reply", Reply: wacloudapi.ButtonReply{ID: "opt_yes", Title: "Accept"}},
				{Type: "reply", Reply: wacloudapi.ButtonReply{ID: "opt_no", Title: "Decline"}},
			},
		},
	}
	resBtn, err := client.Messages.SendInteractive(ctx, recipient, interactive)
	if err != nil {
		log.Fatalf("Failed to send interactive message: %v", err)
	}
	fmt.Printf("    ✓ Sent interactive buttons (ID: %s)\n", resBtn.Messages[0].ID)

	// Assertions on recorded server state
	allSent := srv.SentMessages()
	lastSent := srv.LastSentMessage()
	sentToRecipient := srv.SentMessagesTo(recipient)

	if len(allSent) != 2 {
		log.Fatalf("Assertion failed: expected 2 sent messages, got %d", len(allSent))
	}
	if lastSent.Interactive == nil || lastSent.Interactive.Action.Buttons[0].Reply.ID != "opt_yes" {
		log.Fatalf("Assertion failed: last sent message button ID mismatch")
	}
	if len(sentToRecipient) != 2 {
		log.Fatalf("Assertion failed: expected 2 messages sent to %s, got %d", recipient, len(sentToRecipient))
	}
	fmt.Printf("    ✓ Assertions passed: %d messages recorded, last type is %s\n\n", len(allSent), lastSent.Type)

	// -------------------------------------------------------------------------
	// 3. Scenario 2: Chaos Injection (Rate Limits & Auto-Recovery)
	// -------------------------------------------------------------------------
	fmt.Println("[3] Testing Chaos Injection & Automatic Retry Recovery...")

	// Create client with retry enabled
	retryClient := wacloudapi.New(
		srv.AccessToken(),
		srv.PhoneNumberID(),
		wacloudapi.WithBaseURL(srv.URL()),
		wacloudapi.WithHTTPClient(srv.HTTPClient()),
		wacloudapi.WithRetry(3, 10*time.Millisecond),
	)

	// Inject 1 Rate Limit error (HTTP 429)
	srv.InjectRateLimit(1)
	fmt.Println("    - Injected 1x HTTP 429 Rate Limit error into queue")

	// Message should succeed transparently after SDK retries
	retryRes, err := retryClient.Messages.SendText(ctx, recipient, "Resilient message after rate limit")
	if err != nil {
		log.Fatalf("Expected retry to succeed, got: %v", err)
	}
	fmt.Printf("    ✓ Recovered from rate limit via retry! Message ID: %s\n", retryRes.Messages[0].ID)

	// Inject an unrecoverable Auth Error (HTTP 401, code 190)
	srv.InjectMetaError(190, "Invalid OAuth access token - Cannot parse access token", 1)
	_, err = retryClient.Messages.SendText(ctx, recipient, "This will fail with 401")
	if err == nil {
		log.Fatal("Expected auth error, got nil")
	}

	var apiErr *wacloudapi.APIError
	if errors.As(err, &apiErr) && apiErr.IsAuthError() {
		fmt.Printf("    ✓ Successfully trapped Auth Error (Code: %d, Message: %s)\n\n", apiErr.Code, apiErr.Message)
	} else {
		log.Fatalf("Unexpected error type: %v", err)
	}

	// -------------------------------------------------------------------------
	// 4. Scenario 3: Inbound Webhook Event Simulation
	// -------------------------------------------------------------------------
	fmt.Println("[4] Testing Inbound Webhook Simulation (HMAC-SHA256 Signed)...")

	// Setup webhook handler
	h := webhook.NewHandler("test_verify_token", appSecret)

	var (
		wgText       sync.WaitGroup
		wgStatus     sync.WaitGroup
		textReceived string
		statusReceived string
	)

	h.OnTextMessage(func(ctx context.Context, msg webhook.Message, meta webhook.Metadata) error {
		textReceived = msg.Text.Body
		fmt.Printf("    [Webhook Event] Received text from %s: %q\n", msg.From, msg.Text.Body)
		wgText.Done()
		return nil
	})

	h.OnStatus(func(ctx context.Context, status webhook.Status, meta webhook.Metadata) error {
		statusReceived = status.Status
		fmt.Printf("    [Webhook Event] Received status update for %s: %q\n", status.ID, status.Status)
		wgStatus.Done()
		return nil
	})

	// Simulate inbound customer message
	wgText.Add(1)
	err = srv.SimulateInboundText(ctx, h, "628999888777", "Help! I need customer support.")
	if err != nil {
		log.Fatalf("SimulateInboundText failed: %v", err)
	}
	wgText.Wait()

	if textReceived != "Help! I need customer support." {
		log.Fatalf("Webhook handler did not receive expected text: %q", textReceived)
	}

	// Simulate delivery status update
	wgStatus.Add(1)
	err = srv.SimulateStatusDelivered(ctx, h, "wamid.mock.98765", "628999888777")
	if err != nil {
		log.Fatalf("SimulateStatusDelivered failed: %v", err)
	}
	wgStatus.Wait()

	if statusReceived != "delivered" {
		log.Fatalf("Webhook handler did not receive expected status: %q", statusReceived)
	}
	fmt.Println("    ✓ Webhook simulation and HMAC-SHA256 signature verification verified!")

	// -------------------------------------------------------------------------
	// 5. Scenario 4: Media & Templates In-Memory Store
	// -------------------------------------------------------------------------
	fmt.Println("\n[5] Testing Media & Template Mock Operations...")

	// Upload mock media
	fileData := []byte("PDF-MOCK-INVOICE-DATA")
	mediaRes, err := client.Media.Upload(ctx, "invoice.pdf", bytes.NewReader(fileData), "application/pdf")
	if err != nil {
		log.Fatalf("Failed to upload mock media: %v", err)
	}
	fmt.Printf("    ✓ Uploaded mock media: ID = %s\n", mediaRes.ID)

	// Download mock media stream
	stream, meta, err := client.Media.Download(ctx, mediaRes.ID)
	if err != nil {
		log.Fatalf("Failed to download mock media: %v", err)
	}
	defer stream.Close()
	fmt.Printf("    ✓ Downloaded mock media: MIME = %s, SHA256 = %s\n", meta.MimeType, meta.SHA256)

	// Create and list templates
	tplRes, err := client.Templates.Create(ctx, &wacloudapi.CreateTemplateRequest{
		Name:     "demo_shipping_update",
		Category: wacloudapi.TemplateCategoryUtility,
		Language: "en_US",
		Components: []wacloudapi.TemplateComponent{
			{Type: "BODY", Text: "Your package {{1}} is on its way!"},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create template: %v", err)
	}
	fmt.Printf("    ✓ Created template: ID = %s (Status: %s)\n", tplRes.ID, tplRes.Status)

	tplList, err := client.Templates.List(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to list templates: %v", err)
	}
	fmt.Printf("    ✓ Listed %d template(s) in mock store\n", len(tplList.Data))

	// -------------------------------------------------------------------------
	// 6. Reset Mock State
	// -------------------------------------------------------------------------
	srv.Reset()
	if len(srv.SentMessages()) != 0 {
		log.Fatalf("Expected 0 sent messages after Reset(), got %d", len(srv.SentMessages()))
	}
	fmt.Println("\n[6] Mock state reset successfully.")

	fmt.Println(strings.Repeat("=", 64))
	fmt.Println("🎉 All wacloudapitest mock testing scenarios completed successfully!")
	fmt.Println(strings.Repeat("=", 64))
}
