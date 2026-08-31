package wacloudapitest_test

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/itsmeabde/wacloudapi"
	"github.com/itsmeabde/wacloudapi/wacloudapitest"
)

func TestHandlers_Messages(t *testing.T) {
	ctx := context.Background()
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	client := srv.Client()

	// Initial assertions should be empty
	if len(srv.SentMessages()) != 0 {
		t.Fatalf("expected 0 sent messages, got %d", len(srv.SentMessages()))
	}
	if srv.LastSentMessage() != nil {
		t.Fatalf("expected nil last sent message, got %+v", srv.LastSentMessage())
	}

	// 1. Send text message
	resp, err := client.Messages.SendText(ctx, "+1234567890", "Hello World")
	if err != nil {
		t.Fatalf("SendText failed: %v", err)
	}
	if resp.MessagingProduct != "whatsapp" {
		t.Errorf("expected messaging_product whatsapp, got %q", resp.MessagingProduct)
	}
	if len(resp.Messages) == 0 || !strings.HasPrefix(resp.Messages[0].ID, "wamid.mock.") {
		t.Fatalf("expected wamid.mock.xxx message ID, got %+v", resp.Messages)
	}

	// Assertions
	if len(srv.SentMessages()) != 1 {
		t.Fatalf("expected 1 sent message, got %d", len(srv.SentMessages()))
	}
	last := srv.LastSentMessage()
	if last == nil || last.To != "+1234567890" || last.Text == nil || last.Text.Body != "Hello World" {
		t.Fatalf("unexpected last sent message: %+v", last)
	}

	// 2. Send image message
	respImg, err := client.Messages.SendImage(ctx, "+9876543210", wacloudapi.MediaByID("mock_media_123"), wacloudapi.WithCaption("Test Image"))
	if err != nil {
		t.Fatalf("SendImage failed: %v", err)
	}
	if len(respImg.Messages) == 0 {
		t.Fatalf("expected image message response with ID")
	}

	// 3. Mark as read
	if err := client.Messages.MarkAsRead(ctx, resp.Messages[0].ID); err != nil {
		t.Fatalf("MarkAsRead failed: %v", err)
	}

	// Assertions by phone
	msgs1 := srv.SentMessagesTo("+1234567890")
	if len(msgs1) != 1 || msgs1[0].Text.Body != "Hello World" {
		t.Fatalf("expected 1 message to +1234567890, got %d", len(msgs1))
	}
	msgs2 := srv.SentMessagesTo("+9876543210")
	if len(msgs2) != 1 || msgs2[0].Image == nil || msgs2[0].Image.Caption != "Test Image" {
		t.Fatalf("expected 1 message to +9876543210, got %d", len(msgs2))
	}
	msgs3 := srv.SentMessagesTo("+0000000000")
	if len(msgs3) != 0 {
		t.Fatalf("expected 0 messages to non-existent recipient, got %d", len(msgs3))
	}
}

func TestHandlers_AllMessageTypes(t *testing.T) {
	ctx := context.Background()
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	client := srv.Client()

	// Audio
	_, err := client.Messages.SendAudio(ctx, "+111", wacloudapi.MediaByID("m_audio"))
	if err != nil {
		t.Fatalf("SendAudio failed: %v", err)
	}

	// Video
	_, err = client.Messages.SendVideo(ctx, "+111", wacloudapi.MediaByURL("http://example.com/video.mp4"), wacloudapi.WithCaption("Vid"))
	if err != nil {
		t.Fatalf("SendVideo failed: %v", err)
	}

	// Document
	_, err = client.Messages.SendDocument(ctx, "+111", wacloudapi.MediaByID("m_doc"), wacloudapi.WithFilename("doc.pdf"))
	if err != nil {
		t.Fatalf("SendDocument failed: %v", err)
	}

	// Sticker
	_, err = client.Messages.SendSticker(ctx, "+111", wacloudapi.MediaByID("m_sticker"))
	if err != nil {
		t.Fatalf("SendSticker failed: %v", err)
	}

	// Location
	_, err = client.Messages.SendLocation(ctx, "+111", wacloudapi.Location{Latitude: 37.7749, Longitude: -122.4194, Name: "SF"})
	if err != nil {
		t.Fatalf("SendLocation failed: %v", err)
	}

	// Contacts
	_, err = client.Messages.SendContacts(ctx, "+111", []wacloudapi.Contact{{Name: wacloudapi.ContactName{FormattedName: "Alice"}}})
	if err != nil {
		t.Fatalf("SendContacts failed: %v", err)
	}

	// Reaction
	_, err = client.Messages.SendReaction(ctx, "+111", "wamid.mock.1", "👍")
	if err != nil {
		t.Fatalf("SendReaction failed: %v", err)
	}

	// Template
	_, err = client.Messages.SendTemplate(ctx, "+111", &wacloudapi.TemplateMessage{
		Name:     "sample_template",
		Language: wacloudapi.TemplateLanguage{Code: "en_US"},
	})
	if err != nil {
		t.Fatalf("SendTemplate failed: %v", err)
	}

	// Single Product
	_, err = client.Messages.SendSingleProduct(ctx, "+111", "catalog_1", "sku_100", "Buy this product")
	if err != nil {
		t.Fatalf("SendSingleProduct failed: %v", err)
	}

	// Multi Product
	_, err = client.Messages.SendMultiProduct(ctx, "+111", &wacloudapi.MultiProductRequest{
		CatalogID: "catalog_1",
		BodyText:  "Choose products",
		Sections: []wacloudapi.ProductSection{
			{
				Title: "Section 1",
				ProductItems: []wacloudapi.ProductItem{
					{ProductRetailerID: "sku_1"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("SendMultiProduct failed: %v", err)
	}

	// Flow
	_, err = client.Messages.SendFlow(ctx, "+111", &wacloudapi.FlowMessageRequest{
		FlowID:    "flow_1",
		FlowToken: "token_1",
		FlowCTA:   "Open Flow",
		BodyText:  "Please fill the flow",
	})
	if err != nil {
		t.Fatalf("SendFlow failed: %v", err)
	}

	// Order Details
	_, err = client.Messages.SendOrderDetails(ctx, "+111", &wacloudapi.OrderDetailsMessage{
		ReferenceID: "ref_1",
		Currency:    "USD",
		TotalAmount: wacloudapi.OrderAmount{Value: 1000, Offset: 100},
		Order: wacloudapi.OrderInfo{
			Items: []wacloudapi.OrderItem{
				{Name: "Item 1", Quantity: 1, Amount: wacloudapi.OrderAmount{Value: 1000, Offset: 100}},
			},
			Subtotal: wacloudapi.OrderAmount{Value: 1000, Offset: 100},
		},
	})
	if err != nil {
		t.Fatalf("SendOrderDetails failed: %v", err)
	}

	// Total messages recorded should be 12
	sent := srv.SentMessagesTo("+111")
	if len(sent) != 12 {
		t.Fatalf("expected 12 messages to +111, got %d", len(sent))
	}
}

func TestHandlers_Media(t *testing.T) {
	ctx := context.Background()
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	client := srv.Client()

	// Initial uploaded media should be empty
	if len(srv.UploadedMedia()) != 0 {
		t.Fatalf("expected 0 uploaded media, got %d", len(srv.UploadedMedia()))
	}

	// 1. Upload media
	content := []byte("fake image content binary data 12345")
	uploadResp, err := client.Media.Upload(ctx, "test.png", bytes.NewReader(content), "image/png")
	if err != nil {
		t.Fatalf("Media.Upload failed: %v", err)
	}
	if uploadResp.ID == "" || !strings.HasPrefix(uploadResp.ID, "mock_media_") {
		t.Fatalf("expected mock_media_xxx ID, got %q", uploadResp.ID)
	}

	// Assertion check
	mediaMap := srv.UploadedMedia()
	if len(mediaMap) != 1 {
		t.Fatalf("expected 1 media in store, got %d", len(mediaMap))
	}
	stored, ok := mediaMap[uploadResp.ID]
	if !ok || stored == nil {
		t.Fatalf("media %q not found in UploadedMedia()", uploadResp.ID)
	}
	if string(stored.Data) != string(content) {
		t.Fatalf("stored media data mismatch")
	}
	if stored.MIMEType != "image/png" {
		t.Errorf("expected MIMEType image/png, got %q", stored.MIMEType)
	}
	if stored.Filename != "test.png" {
		t.Errorf("expected Filename test.png, got %q", stored.Filename)
	}

	// 2. Get media metadata
	meta, err := client.Media.Get(ctx, uploadResp.ID)
	if err != nil {
		t.Fatalf("Media.Get failed: %v", err)
	}
	if meta.ID != uploadResp.ID {
		t.Errorf("expected meta ID %q, got %q", uploadResp.ID, meta.ID)
	}
	if meta.FileSize != int64(len(content)) {
		t.Errorf("expected FileSize %d, got %d", len(content), meta.FileSize)
	}
	if meta.MimeType != "image/png" {
		t.Errorf("expected MimeType image/png, got %q", meta.MimeType)
	}

	// 3. Download bytes
	downloaded, dlMeta, err := client.Media.DownloadBytes(ctx, uploadResp.ID)
	if err != nil {
		t.Fatalf("Media.DownloadBytes failed: %v", err)
	}
	if dlMeta == nil || dlMeta.ID != uploadResp.ID {
		t.Errorf("expected dlMeta ID %q, got %+v", uploadResp.ID, dlMeta)
	}
	if !bytes.Equal(downloaded, content) {
		t.Fatalf("downloaded bytes do not match original content")
	}

	// 4. Delete media
	if err := client.Media.Delete(ctx, uploadResp.ID); err != nil {
		t.Fatalf("Media.Delete failed: %v", err)
	}

	// 5. Subsequent Get should fail
	_, err = client.Media.Get(ctx, uploadResp.ID)
	if err == nil {
		t.Fatalf("expected error getting deleted media, got nil")
	}
}

func TestHandlers_BusinessProfile(t *testing.T) {
	ctx := context.Background()
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	client := srv.Client()

	// 1. Get initial profile
	prof, err := client.BusinessProfile.Get(ctx)
	if err != nil {
		t.Fatalf("BusinessProfile.Get failed: %v", err)
	}
	if prof.MessagingProduct != "whatsapp" {
		t.Errorf("expected messaging_product whatsapp, got %q", prof.MessagingProduct)
	}

	// 2. Update profile
	updateReq := &wacloudapi.UpdateBusinessProfileRequest{
		About:       "About our business",
		Address:     "123 Tech Lane",
		Description: "A great tech business",
		Email:       "contact@example.com",
		Websites:    []string{"https://example.com"},
		Vertical:    wacloudapi.VerticalProfServices,
	}
	if err := client.BusinessProfile.Update(ctx, updateReq); err != nil {
		t.Fatalf("BusinessProfile.Update failed: %v", err)
	}

	// 3. Get profile again to verify updates
	updatedProf, err := client.BusinessProfile.Get(ctx)
	if err != nil {
		t.Fatalf("BusinessProfile.Get after update failed: %v", err)
	}
	if updatedProf.About != updateReq.About {
		t.Errorf("expected About %q, got %q", updateReq.About, updatedProf.About)
	}
	if updatedProf.Address != updateReq.Address {
		t.Errorf("expected Address %q, got %q", updateReq.Address, updatedProf.Address)
	}
	if updatedProf.Description != updateReq.Description {
		t.Errorf("expected Description %q, got %q", updateReq.Description, updatedProf.Description)
	}
	if updatedProf.Email != updateReq.Email {
		t.Errorf("expected Email %q, got %q", updateReq.Email, updatedProf.Email)
	}
	if len(updatedProf.Websites) != 1 || updatedProf.Websites[0] != "https://example.com" {
		t.Errorf("expected Websites %v, got %v", updateReq.Websites, updatedProf.Websites)
	}
	if updatedProf.Vertical != wacloudapi.VerticalProfServices {
		t.Errorf("expected Vertical %q, got %q", wacloudapi.VerticalProfServices, updatedProf.Vertical)
	}
}

func TestHandlers_Templates(t *testing.T) {
	ctx := context.Background()
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	client := srv.Client()

	// 1. Create template
	createReq := &wacloudapi.CreateTemplateRequest{
		Name:     "order_confirmation",
		Category: wacloudapi.TemplateCategoryUtility,
		Language: "en_US",
		Components: []wacloudapi.TemplateComponent{
			{
				Type: "BODY",
				Text: "Your order {{1}} has been confirmed.",
			},
		},
	}

	createResp, err := client.Templates.Create(ctx, createReq)
	if err != nil {
		t.Fatalf("Templates.Create failed: %v", err)
	}
	if createResp.ID == "" || !strings.HasPrefix(createResp.ID, "mock_tpl_") {
		t.Fatalf("expected mock_tpl_xxx ID, got %q", createResp.ID)
	}
	if createResp.Status != wacloudapi.TemplateStatusApproved {
		t.Errorf("expected status APPROVED, got %q", createResp.Status)
	}
	if createResp.Category != wacloudapi.TemplateCategoryUtility {
		t.Errorf("expected category UTILITY, got %q", createResp.Category)
	}

	// 2. Get template by ID
	tpl, err := client.Templates.Get(ctx, createResp.ID)
	if err != nil {
		t.Fatalf("Templates.Get failed: %v", err)
	}
	if tpl.Name != "order_confirmation" {
		t.Errorf("expected Name order_confirmation, got %q", tpl.Name)
	}
	if len(tpl.Components) != 1 || tpl.Components[0].Text != "Your order {{1}} has been confirmed." {
		t.Errorf("unexpected template components: %+v", tpl.Components)
	}

	// 3. List templates
	listResp, err := client.Templates.List(ctx, &wacloudapi.ListTemplatesRequest{
		Category: wacloudapi.TemplateCategoryUtility,
	})
	if err != nil {
		t.Fatalf("Templates.List failed: %v", err)
	}
	if len(listResp.Data) != 1 {
		t.Fatalf("expected 1 template in list, got %d", len(listResp.Data))
	}

	// 4. Update template
	updateReq := &wacloudapi.UpdateTemplateRequest{
		Components: []wacloudapi.TemplateComponent{
			{
				Type: "BODY",
				Text: "Updated body: your order {{1}} is confirmed.",
			},
		},
	}
	updateResp, err := client.Templates.Update(ctx, createResp.ID, updateReq)
	if err != nil {
		t.Fatalf("Templates.Update failed: %v", err)
	}
	if !updateResp.Success {
		t.Errorf("expected update success true")
	}

	// Verify update
	updatedTpl, err := client.Templates.Get(ctx, createResp.ID)
	if err != nil {
		t.Fatalf("Templates.Get after update failed: %v", err)
	}
	if updatedTpl.Components[0].Text != "Updated body: your order {{1}} is confirmed." {
		t.Errorf("template component text not updated: %+v", updatedTpl.Components)
	}

	// 5. Delete by ID
	if err := client.Templates.DeleteByID(ctx, createResp.ID); err != nil {
		t.Fatalf("Templates.DeleteByID failed: %v", err)
	}

	// Subsequent Get should fail
	_, err = client.Templates.Get(ctx, createResp.ID)
	if err == nil {
		t.Fatalf("expected error getting deleted template, got nil")
	}

	// Create another template to test DeleteByName
	createResp2, err := client.Templates.Create(ctx, &wacloudapi.CreateTemplateRequest{
		Name:     "welcome_promo",
		Category: wacloudapi.TemplateCategoryMarketing,
		Language: "en_US",
		Components: []wacloudapi.TemplateComponent{
			{Type: "BODY", Text: "Welcome to our store!"},
		},
	})
	if err != nil {
		t.Fatalf("Templates.Create 2 failed: %v", err)
	}
	if err := client.Templates.DeleteByName(ctx, "welcome_promo"); err != nil {
		t.Fatalf("Templates.DeleteByName failed: %v", err)
	}
	_, err = client.Templates.Get(ctx, createResp2.ID)
	if err == nil {
		t.Fatalf("expected deleted template by name to not exist")
	}
}

func TestHandlers_PhoneNumbers(t *testing.T) {
	ctx := context.Background()
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	client := srv.Client()

	// 1. List phone numbers
	listResp, err := client.PhoneNumbers.List(ctx)
	if err != nil {
		t.Fatalf("PhoneNumbers.List failed: %v", err)
	}
	if len(listResp.Data) == 0 {
		t.Fatalf("expected at least 1 default phone number")
	}

	// 2. Get phone number
	phoneDetails, err := client.PhoneNumbers.Get(ctx, srv.PhoneNumberID())
	if err != nil {
		t.Fatalf("PhoneNumbers.Get failed: %v", err)
	}
	if phoneDetails.ID != srv.PhoneNumberID() {
		t.Errorf("expected phone ID %q, got %q", srv.PhoneNumberID(), phoneDetails.ID)
	}

	// 3. Request verification code
	if err := client.PhoneNumbers.RequestCode(ctx, srv.PhoneNumberID(), wacloudapi.CodeMethodSMS, "en_US"); err != nil {
		t.Fatalf("PhoneNumbers.RequestCode failed: %v", err)
	}

	// 4. Verify code
	if err := client.PhoneNumbers.VerifyCode(ctx, srv.PhoneNumberID(), "123456"); err != nil {
		t.Fatalf("PhoneNumbers.VerifyCode failed: %v", err)
	}

	// 5. Register
	if err := client.PhoneNumbers.Register(ctx, srv.PhoneNumberID(), "123456"); err != nil {
		t.Fatalf("PhoneNumbers.Register failed: %v", err)
	}

	// 6. Set 2-step PIN
	if err := client.PhoneNumbers.SetTwoStepPIN(ctx, srv.PhoneNumberID(), "654321"); err != nil {
		t.Fatalf("PhoneNumbers.SetTwoStepPIN failed: %v", err)
	}

	// 7. Deregister
	if err := client.PhoneNumbers.Deregister(ctx, srv.PhoneNumberID()); err != nil {
		t.Fatalf("PhoneNumbers.Deregister failed: %v", err)
	}
}

func TestHandlers_QRCodes(t *testing.T) {
	ctx := context.Background()
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	client := srv.Client()

	// 1. Create QR code
	createReq := &wacloudapi.CreateQRCodeRequest{
		PrefilledMessage: "Hello from QR code!",
		ImageFormat:      wacloudapi.QRCodeImagePNG,
	}
	qrDetails, err := client.QRCodes.Create(ctx, createReq)
	if err != nil {
		t.Fatalf("QRCodes.Create failed: %v", err)
	}
	if qrDetails.Code == "" || !strings.HasPrefix(qrDetails.Code, "mock_qr_") {
		t.Fatalf("expected mock_qr_xxx code, got %q", qrDetails.Code)
	}
	if qrDetails.PrefilledMessage != "Hello from QR code!" {
		t.Errorf("expected PrefilledMessage 'Hello from QR code!', got %q", qrDetails.PrefilledMessage)
	}
	if qrDetails.DeepLinkURL == "" || qrDetails.QRImageURL == "" {
		t.Fatalf("expected non-empty DeepLinkURL and QRImageURL: %+v", qrDetails)
	}

	// 2. List QR codes
	listResp, err := client.QRCodes.List(ctx)
	if err != nil {
		t.Fatalf("QRCodes.List failed: %v", err)
	}
	if len(listResp.Data) != 1 {
		t.Fatalf("expected 1 QR code in list, got %d", len(listResp.Data))
	}

	// 3. Get QR code
	gotQR, err := client.QRCodes.Get(ctx, qrDetails.Code)
	if err != nil {
		t.Fatalf("QRCodes.Get failed: %v", err)
	}
	if gotQR.Code != qrDetails.Code {
		t.Errorf("expected code %q, got %q", qrDetails.Code, gotQR.Code)
	}

	// 4. Update QR code
	updatedQR, err := client.QRCodes.Update(ctx, qrDetails.Code, "Updated QR message")
	if err != nil {
		t.Fatalf("QRCodes.Update failed: %v", err)
	}
	if updatedQR.PrefilledMessage != "Updated QR message" {
		t.Errorf("expected updated message 'Updated QR message', got %q", updatedQR.PrefilledMessage)
	}

	// 5. Download QR image
	imgBytes, err := client.QRCodes.DownloadImage(ctx, qrDetails.Code)
	if err != nil {
		t.Fatalf("QRCodes.DownloadImage failed: %v", err)
	}
	if len(imgBytes) == 0 {
		t.Fatalf("expected non-empty QR image bytes")
	}

	// 6. Delete QR code
	if err := client.QRCodes.Delete(ctx, qrDetails.Code); err != nil {
		t.Fatalf("QRCodes.Delete failed: %v", err)
	}

	// Subsequent Get should fail
	_, err = client.QRCodes.Get(ctx, qrDetails.Code)
	if err == nil {
		t.Fatalf("expected error getting deleted QR code, got nil")
	}
}

func TestHandlers_Reset(t *testing.T) {
	ctx := context.Background()
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	client := srv.Client()

	// Send message
	_, err := client.Messages.SendText(ctx, "+1234567890", "Test message")
	if err != nil {
		t.Fatalf("SendText failed: %v", err)
	}
	// Upload media
	_, err = client.Media.Upload(ctx, "test.txt", strings.NewReader("hello"), "text/plain")
	if err != nil {
		t.Fatalf("Media.Upload failed: %v", err)
	}

	if len(srv.SentMessages()) != 1 {
		t.Fatalf("expected 1 sent message before reset")
	}
	if len(srv.UploadedMedia()) != 1 {
		t.Fatalf("expected 1 uploaded media before reset")
	}

	// Reset
	srv.Reset()

	if len(srv.SentMessages()) != 0 {
		t.Fatalf("expected 0 sent messages after reset, got %d", len(srv.SentMessages()))
	}
	if len(srv.UploadedMedia()) != 0 {
		t.Fatalf("expected 0 uploaded media after reset, got %d", len(srv.UploadedMedia()))
	}
}

func TestHandlers_Concurrency(t *testing.T) {
	ctx := context.Background()
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	client := srv.Client()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, _ = client.Messages.SendText(ctx, "+1234567890", "Concurrent msg")
			_, _ = client.Media.Upload(ctx, "test.txt", strings.NewReader("sample"), "text/plain")
			_ = srv.SentMessages()
			_ = srv.LastSentMessage()
			_ = srv.SentMessagesTo("+1234567890")
			_ = srv.UploadedMedia()
		}(i)
	}
	wg.Wait()

	if len(srv.SentMessages()) != 20 {
		t.Errorf("expected 20 sent messages, got %d", len(srv.SentMessages()))
	}
	if len(srv.UploadedMedia()) != 20 {
		t.Errorf("expected 20 uploaded media, got %d", len(srv.UploadedMedia()))
	}
}

func TestHandlers_NotFoundErrors(t *testing.T) {
	ctx := context.Background()
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	client := srv.Client()

	// Non-existent media
	if _, err := client.Media.Get(ctx, "non_existent_media_id"); err == nil {
		t.Errorf("expected error for non-existent media")
	}

	// Non-existent template
	if _, err := client.Templates.Get(ctx, "non_existent_tpl_id"); err == nil {
		t.Errorf("expected error for non-existent template")
	}

	// Non-existent QR code
	if _, err := client.QRCodes.Get(ctx, "non_existent_qr_id"); err == nil {
		t.Errorf("expected error for non-existent QR code")
	}
}

func TestHandlers_CustomHandlerOverride(t *testing.T) {
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	// Direct check on health route
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL()+"/health", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	resp, err := srv.HTTPClient().Do(req)
	if err != nil {
		t.Fatalf("failed request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

