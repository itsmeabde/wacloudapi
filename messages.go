package wacloudapi

import (
	"context"
	"fmt"
	"net/http"
)

type MessagesService struct {
	client *Client
}

func newMessagesService(client *Client) *MessagesService {
	return &MessagesService{client: client}
}

func (s *MessagesService) Send(ctx context.Context, req *SendMessageRequest) (*SendMessageResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("wacloudapi: request cannot be nil")
	}
	req.MessagingProduct = "whatsapp"

	endpoint := fmt.Sprintf("%s/messages", s.client.config.PhoneNumberID)
	var resp SendMessageResponse
	if err := s.client.sendJSON(ctx, http.MethodPost, endpoint, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *MessagesService) SendText(ctx context.Context, to, text string, opts ...MessageOption) (*SendMessageResponse, error) {
	opt := applyMessageOptions(opts...)
	req := &SendMessageRequest{
		To:   to,
		Type: "text",
		Text: &TextMessage{
			Body:       text,
			PreviewURL: opt.PreviewURL,
		},
	}
	if opt.ReplyTo != "" {
		req.Context = &MessageContext{MessageID: opt.ReplyTo}
	}
	return s.Send(ctx, req)
}

func (s *MessagesService) SendImage(ctx context.Context, to string, media MediaSource, opts ...MessageOption) (*SendMessageResponse, error) {
	opt := applyMessageOptions(opts...)
	msg := &MediaMessage{
		ID:      media.ID,
		Link:    media.Link,
		Caption: opt.Caption,
	}
	req := &SendMessageRequest{
		To:    to,
		Type:  "image",
		Image: msg,
	}
	if opt.ReplyTo != "" {
		req.Context = &MessageContext{MessageID: opt.ReplyTo}
	}
	return s.Send(ctx, req)
}

func (s *MessagesService) SendAudio(ctx context.Context, to string, media MediaSource) (*SendMessageResponse, error) {
	req := &SendMessageRequest{
		To:   to,
		Type: "audio",
		Audio: &MediaMessage{
			ID:   media.ID,
			Link: media.Link,
		},
	}
	return s.Send(ctx, req)
}

func (s *MessagesService) SendVideo(ctx context.Context, to string, media MediaSource, opts ...MessageOption) (*SendMessageResponse, error) {
	opt := applyMessageOptions(opts...)
	req := &SendMessageRequest{
		To:   to,
		Type: "video",
		Video: &MediaMessage{
			ID:      media.ID,
			Link:    media.Link,
			Caption: opt.Caption,
		},
	}
	if opt.ReplyTo != "" {
		req.Context = &MessageContext{MessageID: opt.ReplyTo}
	}
	return s.Send(ctx, req)
}

func (s *MessagesService) SendDocument(ctx context.Context, to string, media MediaSource, opts ...MessageOption) (*SendMessageResponse, error) {
	opt := applyMessageOptions(opts...)
	req := &SendMessageRequest{
		To:   to,
		Type: "document",
		Document: &MediaMessage{
			ID:       media.ID,
			Link:     media.Link,
			Caption:  opt.Caption,
			Filename: opt.Filename,
		},
	}
	if opt.ReplyTo != "" {
		req.Context = &MessageContext{MessageID: opt.ReplyTo}
	}
	return s.Send(ctx, req)
}

func (s *MessagesService) SendSticker(ctx context.Context, to string, media MediaSource) (*SendMessageResponse, error) {
	req := &SendMessageRequest{
		To:   to,
		Type: "sticker",
		Sticker: &MediaMessage{
			ID:   media.ID,
			Link: media.Link,
		},
	}
	return s.Send(ctx, req)
}

func (s *MessagesService) SendLocation(ctx context.Context, to string, loc Location, opts ...MessageOption) (*SendMessageResponse, error) {
	opt := applyMessageOptions(opts...)
	req := &SendMessageRequest{
		To:       to,
		Type:     "location",
		Location: &loc,
	}
	if opt.ReplyTo != "" {
		req.Context = &MessageContext{MessageID: opt.ReplyTo}
	}
	return s.Send(ctx, req)
}

func (s *MessagesService) SendContacts(ctx context.Context, to string, contacts []Contact, opts ...MessageOption) (*SendMessageResponse, error) {
	opt := applyMessageOptions(opts...)
	req := &SendMessageRequest{
		To:       to,
		Type:     "contacts",
		Contacts: contacts,
	}
	if opt.ReplyTo != "" {
		req.Context = &MessageContext{MessageID: opt.ReplyTo}
	}
	return s.Send(ctx, req)
}

func (s *MessagesService) SendReaction(ctx context.Context, to, messageID, emoji string) (*SendMessageResponse, error) {
	req := &SendMessageRequest{
		To:   to,
		Type: "reaction",
		Reaction: &Reaction{
			MessageID: messageID,
			Emoji:     emoji,
		},
	}
	return s.Send(ctx, req)
}

func (s *MessagesService) SendTemplate(ctx context.Context, to string, tpl *TemplateMessage, opts ...MessageOption) (*SendMessageResponse, error) {
	opt := applyMessageOptions(opts...)
	req := &SendMessageRequest{
		To:       to,
		Type:     "template",
		Template: tpl,
	}
	if opt.ReplyTo != "" {
		req.Context = &MessageContext{MessageID: opt.ReplyTo}
	}
	return s.Send(ctx, req)
}

func (s *MessagesService) SendInteractive(ctx context.Context, to string, interactive *InteractiveMessage, opts ...MessageOption) (*SendMessageResponse, error) {
	opt := applyMessageOptions(opts...)
	req := &SendMessageRequest{
		To:          to,
		Type:        "interactive",
		Interactive: interactive,
	}
	if opt.ReplyTo != "" {
		req.Context = &MessageContext{MessageID: opt.ReplyTo}
	}
	return s.Send(ctx, req)
}

func (s *MessagesService) MarkAsRead(ctx context.Context, messageID string) error {
	endpoint := fmt.Sprintf("%s/messages", s.client.config.PhoneNumberID)
	body := map[string]string{
		"messaging_product": "whatsapp",
		"status":            "read",
		"message_id":        messageID,
	}
	return s.client.sendJSON(ctx, http.MethodPost, endpoint, body, nil)
}

func applyMessageOptions(opts ...MessageOption) *messageOptions {
	o := &messageOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}
