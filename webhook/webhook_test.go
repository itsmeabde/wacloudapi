package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVerifyChallenge(t *testing.T) {
	verifyToken := "my-secret-token"

	// Valid challenge
	req := httptest.NewRequest(http.MethodGet, "/webhook?hub.mode=subscribe&hub.verify_token=my-secret-token&hub.challenge=challenge123", nil)
	w := httptest.NewRecorder()
	if !VerifyChallenge(w, req, verifyToken) {
		t.Errorf("expected VerifyChallenge to succeed")
	}
	if w.Code != http.StatusOK || w.Body.String() != "challenge123" {
		t.Errorf("unexpected response: code %d, body: %s", w.Code, w.Body.String())
	}

	// Invalid token
	reqInvalid := httptest.NewRequest(http.MethodGet, "/webhook?hub.mode=subscribe&hub.verify_token=wrong-token&hub.challenge=challenge123", nil)
	wInvalid := httptest.NewRecorder()
	if VerifyChallenge(wInvalid, reqInvalid, verifyToken) {
		t.Errorf("expected VerifyChallenge to fail on invalid token")
	}
	if wInvalid.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", wInvalid.Code)
	}

	// Invalid mode
	reqWrongMode := httptest.NewRequest(http.MethodGet, "/webhook?hub.mode=unsubscribe&hub.verify_token=my-secret-token&hub.challenge=challenge123", nil)
	wWrongMode := httptest.NewRecorder()
	if VerifyChallenge(wWrongMode, reqWrongMode, verifyToken) {
		t.Errorf("expected VerifyChallenge to fail on wrong mode")
	}
	if wWrongMode.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", wWrongMode.Code)
	}
}

func TestVerifySignature(t *testing.T) {
	appSecret := "meta-app-secret-xyz"
	body := []byte(`{"object":"whatsapp_business_account"}`)

	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write(body)
	validSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if err := VerifySignature(body, validSig, appSecret); err != nil {
		t.Fatalf("expected valid signature, got error: %v", err)
	}

	// Invalid hex signature
	if err := VerifySignature(body, "sha256=invalidhex!", appSecret); err == nil {
		t.Errorf("expected error on invalid signature hex")
	}

	// Signature mismatch (valid hex but wrong hash)
	fakeSig := "sha256=" + hex.EncodeToString([]byte("12345678901234567890123456789012"))
	if err := VerifySignature(body, fakeSig, appSecret); err != ErrInvalidSignature {
		t.Errorf("expected ErrInvalidSignature on signature mismatch, got %v", err)
	}

	// Missing prefix / wrong format
	if err := VerifySignature(body, "sha1=abcdef", appSecret); err != ErrInvalidSignature {
		t.Errorf("expected ErrInvalidSignature on non-sha256 prefix, got %v", err)
	}

	// Empty signature header
	if err := VerifySignature(body, "", appSecret); err != ErrMissingSignature {
		t.Errorf("expected ErrMissingSignature on empty signature header, got %v", err)
	}
}

func TestParsePayloadTextMessage(t *testing.T) {
	rawJSON := []byte(`{
		"object": "whatsapp_business_account",
		"entry": [{
			"id": "WABA_ID_123",
			"changes": [{
				"field": "messages",
				"value": {
					"messaging_product": "whatsapp",
					"metadata": {
						"display_phone_number": "15550234567",
						"phone_number_id": "1234567890"
					},
					"contacts": [{
						"profile": { "name": "Alice" },
						"wa_id": "62811111111"
					}],
					"messages": [{
						"from": "62811111111",
						"id": "wamid.HBgM...",
						"timestamp": "1700000000",
						"type": "text",
						"text": { "body": "Halo, saya butuh bantuan" }
					}]
				}
			}]
		}]
	}`)

	payload, err := ParsePayload(rawJSON)
	if err != nil {
		t.Fatalf("unexpected error parsing payload: %v", err)
	}

	if payload.Object != "whatsapp_business_account" || len(payload.Entry) == 0 {
		t.Fatalf("unexpected payload structure: %+v", payload)
	}

	msg := payload.Entry[0].Changes[0].Value.Messages[0]
	if msg.From != "62811111111" || msg.Type != "text" || msg.Text == nil || msg.Text.Body != "Halo, saya butuh bantuan" {
		t.Errorf("unexpected message content: %+v", msg)
	}

	if msg.MediaID() != "" {
		t.Errorf("expected empty MediaID for text message, got: %s", msg.MediaID())
	}
}

func TestParsePayloadMediaAndOtherTypes(t *testing.T) {
	rawJSON := []byte(`{
		"object": "whatsapp_business_account",
		"entry": [{
			"id": "WABA_ID_123",
			"changes": [{
				"field": "messages",
				"value": {
					"messaging_product": "whatsapp",
					"metadata": {
						"display_phone_number": "15550234567",
						"phone_number_id": "1234567890"
					},
					"messages": [
						{
							"from": "62811111111",
							"id": "wamid.img1",
							"timestamp": "1700000001",
							"type": "image",
							"image": {
								"id": "media_img_123",
								"mime_type": "image/jpeg",
								"sha256": "hash123",
								"caption": "Photo caption"
							},
							"context": {
								"from": "62899999999",
								"id": "wamid.prev",
								"forwarded": true,
								"frequently_forwarded": false
							}
						},
						{
							"from": "62811111111",
							"id": "wamid.aud1",
							"timestamp": "1700000002",
							"type": "audio",
							"audio": { "id": "media_aud_123", "mime_type": "audio/ogg" }
						},
						{
							"from": "62811111111",
							"id": "wamid.vid1",
							"timestamp": "1700000003",
							"type": "video",
							"video": { "id": "media_vid_123", "mime_type": "video/mp4" }
						},
						{
							"from": "62811111111",
							"id": "wamid.doc1",
							"timestamp": "1700000004",
							"type": "document",
							"document": { "id": "media_doc_123", "mime_type": "application/pdf", "filename": "doc.pdf" }
						},
						{
							"from": "62811111111",
							"id": "wamid.stk1",
							"timestamp": "1700000005",
							"type": "sticker",
							"sticker": { "id": "media_stk_123", "mime_type": "image/webp" }
						},
						{
							"from": "62811111111",
							"id": "wamid.loc1",
							"timestamp": "1700000006",
							"type": "location",
							"location": {
								"latitude": -6.2,
								"longitude": 106.8,
								"name": "Monas",
								"address": "Jakarta"
							}
						},
						{
							"from": "62811111111",
							"id": "wamid.react1",
							"timestamp": "1700000007",
							"type": "reaction",
							"reaction": {
								"message_id": "wamid.orig",
								"emoji": "👍"
							}
						},
						{
							"from": "62811111111",
							"id": "wamid.btn1",
							"timestamp": "1700000008",
							"type": "interactive",
							"interactive": {
								"type": "button_reply",
								"button_reply": {
									"id": "btn_yes",
									"title": "Yes"
								}
							}
						},
						{
							"from": "62811111111",
							"id": "wamid.list1",
							"timestamp": "1700000009",
							"type": "interactive",
							"interactive": {
								"type": "list_reply",
								"list_reply": {
									"id": "row_1",
									"title": "Option 1",
									"description": "First option"
								}
							}
						},
						{
							"from": "62811111111",
							"id": "wamid.sys1",
							"timestamp": "1700000010",
							"type": "system",
							"system": {
								"body": "User changed number",
								"identity": "ident_1",
								"wa_id": "62811111111",
								"type": "customer_changed_number",
								"customer": "62811111111"
							}
						}
					]
				}
			}]
		}]
	}`)

	payload, err := ParsePayload(rawJSON)
	if err != nil {
		t.Fatalf("unexpected error parsing media payload: %v", err)
	}

	msgs := payload.Entry[0].Changes[0].Value.Messages
	if len(msgs) != 10 {
		t.Fatalf("expected 10 messages, got %d", len(msgs))
	}

	// Verify MediaID helper
	if msgs[0].MediaID() != "media_img_123" {
		t.Errorf("expected media_img_123, got %s", msgs[0].MediaID())
	}
	if msgs[0].Context == nil || !msgs[0].Context.Forwarded {
		t.Errorf("expected context forwarded=true, got %+v", msgs[0].Context)
	}
	if msgs[1].MediaID() != "media_aud_123" {
		t.Errorf("expected media_aud_123, got %s", msgs[1].MediaID())
	}
	if msgs[2].MediaID() != "media_vid_123" {
		t.Errorf("expected media_vid_123, got %s", msgs[2].MediaID())
	}
	if msgs[3].MediaID() != "media_doc_123" {
		t.Errorf("expected media_doc_123, got %s", msgs[3].MediaID())
	}
	if msgs[4].MediaID() != "media_stk_123" {
		t.Errorf("expected media_stk_123, got %s", msgs[4].MediaID())
	}

	// Location
	if msgs[5].Location == nil || msgs[5].Location.Name != "Monas" {
		t.Errorf("unexpected location: %+v", msgs[5].Location)
	}

	// Reaction
	if msgs[6].Reaction == nil || msgs[6].Reaction.Emoji != "👍" {
		t.Errorf("unexpected reaction: %+v", msgs[6].Reaction)
	}

	// Button reply
	if msgs[7].Interactive == nil || msgs[7].Interactive.ButtonReply == nil || msgs[7].Interactive.ButtonReply.ID != "btn_yes" {
		t.Errorf("unexpected button reply: %+v", msgs[7].Interactive)
	}

	// List reply
	if msgs[8].Interactive == nil || msgs[8].Interactive.ListReply == nil || msgs[8].Interactive.ListReply.ID != "row_1" {
		t.Errorf("unexpected list reply: %+v", msgs[8].Interactive)
	}

	// System message
	if msgs[9].System == nil || msgs[9].System.Customer != "62811111111" {
		t.Errorf("unexpected system message: %+v", msgs[9].System)
	}
}

func TestParsePayloadStatusesAndErrors(t *testing.T) {
	rawJSON := []byte(`{
		"object": "whatsapp_business_account",
		"entry": [{
			"id": "WABA_ID_123",
			"changes": [{
				"field": "messages",
				"value": {
					"messaging_product": "whatsapp",
					"metadata": {
						"display_phone_number": "15550234567",
						"phone_number_id": "1234567890"
					},
					"statuses": [{
						"id": "wamid.msg1",
						"status": "delivered",
						"timestamp": "1700000020",
						"recipient_id": "62811111111",
						"conversation": {
							"id": "conv_123",
							"expiration_timestamp": "1700086400",
							"origin": { "type": "user_initiated" }
						},
						"pricing": {
							"billable": true,
							"pricing_model": "CBP",
							"category": "service"
						},
						"errors": [{
							"code": 131051,
							"title": "Message unsupported",
							"message": "Message type unsupported",
							"error_data": { "details": "Some details" }
						}]
					}],
					"errors": [{
						"code": 131056,
						"title": "Account rate limit reached",
						"message": "Rate limit exceeded"
					}]
				}
			}]
		}]
	}`)

	payload, err := ParsePayload(rawJSON)
	if err != nil {
		t.Fatalf("unexpected error parsing status payload: %v", err)
	}

	val := payload.Entry[0].Changes[0].Value
	if len(val.Statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(val.Statuses))
	}
	st := val.Statuses[0]
	if st.Status != "delivered" || st.Conversation.Origin.Type != "user_initiated" || !st.Pricing.Billable {
		t.Errorf("unexpected status content: %+v", st)
	}
	if len(st.Errors) != 1 || st.Errors[0].Code != 131051 {
		t.Errorf("unexpected status errors: %+v", st.Errors)
	}
	if len(val.Errors) != 1 || val.Errors[0].Code != 131056 {
		t.Errorf("unexpected value errors: %+v", val.Errors)
	}
}

func TestParsePayloadInvalidJSON(t *testing.T) {
	_, err := ParsePayload([]byte(`{invalid-json`))
	if err == nil {
		t.Fatalf("expected error on invalid JSON")
	}
}
