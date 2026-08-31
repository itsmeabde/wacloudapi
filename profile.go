package wacloudapi

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type BusinessProfileService struct {
	client *Client
}

func newBusinessProfileService(client *Client) *BusinessProfileService {
	return &BusinessProfileService{client: client}
}

func (s *BusinessProfileService) Get(ctx context.Context, fields ...string) (*BusinessProfile, error) {
	fieldQuery := "about,address,description,email,profile_picture_url,websites,vertical"
	if len(fields) > 0 && strings.TrimSpace(fields[0]) != "" {
		fieldQuery = strings.Join(fields, ",")
	}

	endpoint := fmt.Sprintf("%s/whatsapp_business_profile?fields=%s",
		s.client.config.PhoneNumberID,
		url.QueryEscape(fieldQuery),
	)

	var wrapper struct {
		Data []BusinessProfile `json:"data"`
	}
	if err := s.client.sendJSON(ctx, http.MethodGet, endpoint, nil, &wrapper); err != nil {
		return nil, err
	}

	if len(wrapper.Data) == 0 {
		return nil, fmt.Errorf("wacloudapi: no business profile data found")
	}

	return &wrapper.Data[0], nil
}

func (s *BusinessProfileService) Update(ctx context.Context, req *UpdateBusinessProfileRequest) error {
	if req == nil {
		return fmt.Errorf("wacloudapi: update request cannot be nil")
	}
	req.MessagingProduct = "whatsapp"

	endpoint := fmt.Sprintf("%s/whatsapp_business_profile", s.client.config.PhoneNumberID)
	return s.client.sendJSON(ctx, http.MethodPost, endpoint, req, nil)
}
