package wacloudapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMessagesServiceSendText(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v21.0/12345/messages" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req SendMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode req body: %v", err)
		}

		if req.MessagingProduct != "whatsapp" {
			t.Errorf("expected messaging_product 'whatsapp', got '%s'", req.MessagingProduct)
		}
		if req.To != "628123456789" || req.Type != "text" || req.Text == nil || req.Text.Body != "Hello WhatsApp" || !req.Text.PreviewURL {
			t.Errorf("unexpected request payload: %+v", req)
		}
		if req.Context == nil || req.Context.MessageID != "wamid.reply123" {
			t.Errorf("unexpected context payload: %+v", req.Context)
		}

		_ = json.NewEncoder(w).Encode(SendMessageResponse{
			MessagingProduct: "whatsapp",
			Contacts: []struct {
				Input string `json:"input"`
				WaID  string `json:"wa_id"`
			}{
				{Input: "628123456789", WaID: "628123456789"},
			},
			Messages: []struct {
				ID            string `json:"id"`
				MessageStatus string `json:"message_status,omitempty"`
			}{
				{ID: "wamid.test123", MessageStatus: "accepted"},
			},
		})
	}))
	defer ts.Close()

	c := New("test-token", "12345", WithBaseURL(ts.URL))
	res, err := c.Messages.SendText(context.Background(), "628123456789", "Hello WhatsApp", WithPreviewURL(true), WithReplyTo("wamid.reply123"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Messages) == 0 || res.Messages[0].ID != "wamid.test123" {
		t.Errorf("unexpected response: %+v", res)
	}
	if len(res.Contacts) == 0 || res.Contacts[0].WaID != "628123456789" {
		t.Errorf("unexpected contacts in response: %+v", res.Contacts)
	}
}

func TestMessagesServiceSendImageWithMediaID(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req SendMessageRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.Type != "image" || req.Image == nil || req.Image.ID != "media-id-99" || req.Image.Caption != "Test caption" {
			t.Errorf("unexpected image payload: %+v", req)
		}
		if req.Context == nil || req.Context.MessageID != "wamid.reply456" {
			t.Errorf("unexpected reply context: %+v", req.Context)
		}

		_ = json.NewEncoder(w).Encode(SendMessageResponse{
			MessagingProduct: "whatsapp",
			Messages: []struct {
				ID            string `json:"id"`
				MessageStatus string `json:"message_status,omitempty"`
			}{
				{ID: "wamid.media123"},
			},
		})
	}))
	defer ts.Close()

	c := New("test-token", "12345", WithBaseURL(ts.URL))
	res, err := c.Messages.SendImage(context.Background(), "628123456789", MediaByID("media-id-99"), WithCaption("Test caption"), WithReplyTo("wamid.reply456"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Messages[0].ID != "wamid.media123" {
		t.Errorf("unexpected message id: %s", res.Messages[0].ID)
	}
}

func TestMessagesServiceSendImageWithMediaURL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req SendMessageRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.Type != "image" || req.Image == nil || req.Image.Link != "https://example.com/test.jpg" {
			t.Errorf("unexpected image payload: %+v", req)
		}

		_ = json.NewEncoder(w).Encode(SendMessageResponse{
			MessagingProduct: "whatsapp",
			Messages: []struct {
				ID            string `json:"id"`
				MessageStatus string `json:"message_status,omitempty"`
			}{
				{ID: "wamid.imgurl123"},
			},
		})
	}))
	defer ts.Close()

	c := New("test-token", "12345", WithBaseURL(ts.URL))
	res, err := c.Messages.SendImage(context.Background(), "628123456789", MediaByURL("https://example.com/test.jpg"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Messages[0].ID != "wamid.imgurl123" {
		t.Errorf("unexpected message id: %s", res.Messages[0].ID)
	}
}

func TestMessagesServiceSendAudio(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req SendMessageRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.Type != "audio" || req.Audio == nil || req.Audio.ID != "audio-id-1" {
			t.Errorf("unexpected audio payload: %+v", req)
		}

		_ = json.NewEncoder(w).Encode(SendMessageResponse{
			MessagingProduct: "whatsapp",
			Messages: []struct {
				ID            string `json:"id"`
				MessageStatus string `json:"message_status,omitempty"`
			}{
				{ID: "wamid.audio123"},
			},
		})
	}))
	defer ts.Close()

	c := New("test-token", "12345", WithBaseURL(ts.URL))
	res, err := c.Messages.SendAudio(context.Background(), "628123456789", MediaByID("audio-id-1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Messages[0].ID != "wamid.audio123" {
		t.Errorf("unexpected response: %+v", res)
	}
}

func TestMessagesServiceSendVideo(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req SendMessageRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.Type != "video" || req.Video == nil || req.Video.Link != "https://example.com/video.mp4" || req.Video.Caption != "Video Caption" {
			t.Errorf("unexpected video payload: %+v", req)
		}
		if req.Context == nil || req.Context.MessageID != "wamid.rep" {
			t.Errorf("unexpected reply context: %+v", req.Context)
		}

		_ = json.NewEncoder(w).Encode(SendMessageResponse{
			MessagingProduct: "whatsapp",
			Messages: []struct {
				ID            string `json:"id"`
				MessageStatus string `json:"message_status,omitempty"`
			}{
				{ID: "wamid.video123"},
			},
		})
	}))
	defer ts.Close()

	c := New("test-token", "12345", WithBaseURL(ts.URL))
	res, err := c.Messages.SendVideo(context.Background(), "628123456789", MediaByURL("https://example.com/video.mp4"), WithCaption("Video Caption"), WithReplyTo("wamid.rep"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Messages[0].ID != "wamid.video123" {
		t.Errorf("unexpected response: %+v", res)
	}
}

func TestMessagesServiceSendDocument(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req SendMessageRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.Type != "document" || req.Document == nil || req.Document.ID != "doc-123" || req.Document.Filename != "invoice.pdf" || req.Document.Caption != "Invoice" {
			t.Errorf("unexpected document payload: %+v", req)
		}
		if req.Context == nil || req.Context.MessageID != "wamid.rep" {
			t.Errorf("unexpected reply context: %+v", req.Context)
		}

		_ = json.NewEncoder(w).Encode(SendMessageResponse{
			MessagingProduct: "whatsapp",
			Messages: []struct {
				ID            string `json:"id"`
				MessageStatus string `json:"message_status,omitempty"`
			}{
				{ID: "wamid.doc123"},
			},
		})
	}))
	defer ts.Close()

	c := New("test-token", "12345", WithBaseURL(ts.URL))
	res, err := c.Messages.SendDocument(context.Background(), "628123456789", MediaByID("doc-123"), WithFilename("invoice.pdf"), WithCaption("Invoice"), WithReplyTo("wamid.rep"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Messages[0].ID != "wamid.doc123" {
		t.Errorf("unexpected response: %+v", res)
	}
}

func TestMessagesServiceSendSticker(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req SendMessageRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.Type != "sticker" || req.Sticker == nil || req.Sticker.ID != "sticker-123" {
			t.Errorf("unexpected sticker payload: %+v", req)
		}

		_ = json.NewEncoder(w).Encode(SendMessageResponse{
			MessagingProduct: "whatsapp",
			Messages: []struct {
				ID            string `json:"id"`
				MessageStatus string `json:"message_status,omitempty"`
			}{
				{ID: "wamid.sticker123"},
			},
		})
	}))
	defer ts.Close()

	c := New("test-token", "12345", WithBaseURL(ts.URL))
	res, err := c.Messages.SendSticker(context.Background(), "628123456789", MediaByID("sticker-123"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Messages[0].ID != "wamid.sticker123" {
		t.Errorf("unexpected response: %+v", res)
	}
}

func TestMessagesServiceSendLocation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req SendMessageRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.Type != "location" || req.Location == nil || req.Location.Latitude != -6.2088 || req.Location.Longitude != 106.8456 || req.Location.Name != "Jakarta" || req.Location.Address != "Indonesia" {
			t.Errorf("unexpected location payload: %+v", req)
		}
		if req.Context == nil || req.Context.MessageID != "wamid.rep" {
			t.Errorf("unexpected reply context: %+v", req.Context)
		}

		_ = json.NewEncoder(w).Encode(SendMessageResponse{
			MessagingProduct: "whatsapp",
			Messages: []struct {
				ID            string `json:"id"`
				MessageStatus string `json:"message_status,omitempty"`
			}{
				{ID: "wamid.loc123"},
			},
		})
	}))
	defer ts.Close()

	c := New("test-token", "12345", WithBaseURL(ts.URL))
	loc := Location{
		Latitude:  -6.2088,
		Longitude: 106.8456,
		Name:      "Jakarta",
		Address:   "Indonesia",
	}
	res, err := c.Messages.SendLocation(context.Background(), "628123456789", loc, WithReplyTo("wamid.rep"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Messages[0].ID != "wamid.loc123" {
		t.Errorf("unexpected response: %+v", res)
	}
}

func TestMessagesServiceSendContacts(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req SendMessageRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.Type != "contacts" || len(req.Contacts) != 1 {
			t.Fatalf("unexpected contacts payload: %+v", req)
		}
		c0 := req.Contacts[0]
		if c0.Name.FormattedName != "John Doe" || len(c0.Phones) != 1 || c0.Phones[0].Phone != "+1234567890" {
			t.Errorf("unexpected contact details: %+v", c0)
		}
		if c0.Org == nil || c0.Org.Company != "Acme Inc" {
			t.Errorf("unexpected org details: %+v", c0.Org)
		}
		if len(c0.Emails) != 1 || c0.Emails[0].Email != "john@example.com" {
			t.Errorf("unexpected email details: %+v", c0.Emails)
		}
		if len(c0.Addresses) != 1 || c0.Addresses[0].City != "City" {
			t.Errorf("unexpected address details: %+v", c0.Addresses)
		}

		_ = json.NewEncoder(w).Encode(SendMessageResponse{
			MessagingProduct: "whatsapp",
			Messages: []struct {
				ID            string `json:"id"`
				MessageStatus string `json:"message_status,omitempty"`
			}{
				{ID: "wamid.contacts123"},
			},
		})
	}))
	defer ts.Close()

	c := New("test-token", "12345", WithBaseURL(ts.URL))
	contacts := []Contact{
		{
			Name: ContactName{
				FormattedName: "John Doe",
				FirstName:     "John",
				LastName:      "Doe",
			},
			Phones: []ContactPhone{
				{Phone: "+1234567890", Type: "WORK", WaID: "1234567890"},
			},
			Emails: []ContactEmail{
				{Email: "john@example.com", Type: "WORK"},
			},
			Addresses: []ContactAddress{
				{Street: "123 St", City: "City", State: "State", Zip: "12345", Country: "US", CountryCode: "US", Type: "WORK"},
			},
			Org: &ContactOrg{
				Company:    "Acme Inc",
				Department: "Engineering",
				Title:      "Engineer",
			},
		},
	}
	res, err := c.Messages.SendContacts(context.Background(), "628123456789", contacts, WithReplyTo("wamid.rep"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Messages[0].ID != "wamid.contacts123" {
		t.Errorf("unexpected response: %+v", res)
	}
}

func TestMessagesServiceSendReaction(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req SendMessageRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.Type != "reaction" || req.Reaction == nil || req.Reaction.MessageID != "wamid.123" || req.Reaction.Emoji != "👍" {
			t.Errorf("unexpected reaction payload: %+v", req)
		}

		_ = json.NewEncoder(w).Encode(SendMessageResponse{
			MessagingProduct: "whatsapp",
			Messages: []struct {
				ID            string `json:"id"`
				MessageStatus string `json:"message_status,omitempty"`
			}{
				{ID: "wamid.reaction123"},
			},
		})
	}))
	defer ts.Close()

	c := New("test-token", "12345", WithBaseURL(ts.URL))
	res, err := c.Messages.SendReaction(context.Background(), "628123456789", "wamid.123", "👍")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Messages[0].ID != "wamid.reaction123" {
		t.Errorf("unexpected response: %+v", res)
	}
}

func TestMessagesServiceSendTemplate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req SendMessageRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.Type != "template" || req.Template == nil || req.Template.Name != "sample_template" || req.Template.Language.Code != "en_US" {
			t.Fatalf("unexpected template payload: %+v", req)
		}
		if len(req.Template.Components) != 1 {
			t.Fatalf("expected 1 component, got %d", len(req.Template.Components))
		}
		comp := req.Template.Components[0]
		if comp.Type != "body" || len(comp.Parameters) != 3 {
			t.Fatalf("unexpected component: %+v", comp)
		}
		if comp.Parameters[0].Type != "text" || comp.Parameters[0].Text != "John" {
			t.Errorf("unexpected parameter 0: %+v", comp.Parameters[0])
		}
		if comp.Parameters[1].Type != "currency" || comp.Parameters[1].Currency.Amount1000 != 100000 {
			t.Errorf("unexpected parameter 1: %+v", comp.Parameters[1])
		}
		if comp.Parameters[2].Type != "date_time" || comp.Parameters[2].DateTime.FallbackValue != "tomorrow" {
			t.Errorf("unexpected parameter 2: %+v", comp.Parameters[2])
		}

		_ = json.NewEncoder(w).Encode(SendMessageResponse{
			MessagingProduct: "whatsapp",
			Messages: []struct {
				ID            string `json:"id"`
				MessageStatus string `json:"message_status,omitempty"`
			}{
				{ID: "wamid.template123"},
			},
		})
	}))
	defer ts.Close()

	c := New("test-token", "12345", WithBaseURL(ts.URL))
	tpl := &TemplateMessage{
		Name:     "sample_template",
		Language: TemplateLanguage{Code: "en_US"},
		Components: []TemplateComponent{
			{
				Type: "body",
				Parameters: []TemplateParameter{
					{Type: "text", Text: "John"},
					{Type: "currency", Currency: &Currency{FallbackValue: "$100", Code: "USD", Amount1000: 100000}},
					{Type: "date_time", DateTime: &DateTime{FallbackValue: "tomorrow"}},
				},
			},
		},
	}
	res, err := c.Messages.SendTemplate(context.Background(), "628123456789", tpl, WithReplyTo("wamid.rep"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Messages[0].ID != "wamid.template123" {
		t.Errorf("unexpected response: %+v", res)
	}
}

func TestMessagesServiceSendInteractiveButton(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req SendMessageRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.Type != "interactive" || req.Interactive == nil || req.Interactive.Type != InteractiveTypeButton {
			t.Fatalf("unexpected interactive payload: %+v", req)
		}
		if req.Interactive.Body.Text != "Choose an option" {
			t.Errorf("unexpected body: %+v", req.Interactive.Body)
		}
		if len(req.Interactive.Action.Buttons) != 2 {
			t.Fatalf("expected 2 buttons, got %d", len(req.Interactive.Action.Buttons))
		}
		if req.Interactive.Action.Buttons[0].Reply.ID != "btn1" || req.Interactive.Action.Buttons[0].Reply.Title != "Yes" {
			t.Errorf("unexpected button 0: %+v", req.Interactive.Action.Buttons[0])
		}

		_ = json.NewEncoder(w).Encode(SendMessageResponse{
			MessagingProduct: "whatsapp",
			Messages: []struct {
				ID            string `json:"id"`
				MessageStatus string `json:"message_status,omitempty"`
			}{
				{ID: "wamid.interactive123"},
			},
		})
	}))
	defer ts.Close()

	c := New("test-token", "12345", WithBaseURL(ts.URL))
	interactive := &InteractiveMessage{
		Type: InteractiveTypeButton,
		Header: &InteractiveHeader{
			Type: "text",
			Text: "Header Text",
		},
		Body: InteractiveBody{
			Text: "Choose an option",
		},
		Footer: &InteractiveFooter{
			Text: "Footer Text",
		},
		Action: InteractiveAction{
			Buttons: []ButtonAction{
				{Type: "reply", Reply: ButtonReply{ID: "btn1", Title: "Yes"}},
				{Type: "reply", Reply: ButtonReply{ID: "btn2", Title: "No"}},
			},
		},
	}
	res, err := c.Messages.SendInteractive(context.Background(), "628123456789", interactive, WithReplyTo("wamid.rep"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Messages[0].ID != "wamid.interactive123" {
		t.Errorf("unexpected response: %+v", res)
	}
}

func TestMessagesServiceSendInteractiveList(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req SendMessageRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.Type != "interactive" || req.Interactive == nil || req.Interactive.Type != InteractiveTypeList {
			t.Fatalf("unexpected interactive payload: %+v", req)
		}
		if req.Interactive.Action.Button != "Select Option" || len(req.Interactive.Action.Sections) != 1 {
			t.Fatalf("unexpected action: %+v", req.Interactive.Action)
		}
		sec := req.Interactive.Action.Sections[0]
		if sec.Title != "Section 1" || len(sec.Rows) != 2 || sec.Rows[0].ID != "row1" {
			t.Errorf("unexpected section: %+v", sec)
		}

		_ = json.NewEncoder(w).Encode(SendMessageResponse{
			MessagingProduct: "whatsapp",
			Messages: []struct {
				ID            string `json:"id"`
				MessageStatus string `json:"message_status,omitempty"`
			}{
				{ID: "wamid.list123"},
			},
		})
	}))
	defer ts.Close()

	c := New("test-token", "12345", WithBaseURL(ts.URL))
	interactive := &InteractiveMessage{
		Type: InteractiveTypeList,
		Body: InteractiveBody{Text: "Please select an option"},
		Action: InteractiveAction{
			Button: "Select Option",
			Sections: []ListSection{
				{
					Title: "Section 1",
					Rows: []ListRow{
						{ID: "row1", Title: "Row 1", Description: "Desc 1"},
						{ID: "row2", Title: "Row 2"},
					},
				},
			},
		},
	}
	res, err := c.Messages.SendInteractive(context.Background(), "628123456789", interactive)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Messages[0].ID != "wamid.list123" {
		t.Errorf("unexpected response: %+v", res)
	}
}

func TestMessagesServiceSendNilRequest(t *testing.T) {
	c := New("test-token", "12345")
	_, err := c.Messages.Send(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected error when sending nil request, got nil")
	}
}

func TestMessagesServiceMarkAsRead(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)

		if body["messaging_product"] != "whatsapp" || body["status"] != "read" || body["message_id"] != "wamid.msg123" {
			t.Errorf("unexpected mark as read body: %+v", body)
		}

		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}))
	defer ts.Close()

	c := New("test-token", "12345", WithBaseURL(ts.URL))
	err := c.Messages.MarkAsRead(context.Background(), "wamid.msg123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
