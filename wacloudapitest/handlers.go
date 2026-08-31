package wacloudapitest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"unicode"

	"github.com/itsmeabde/wacloudapi"
)

func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	b, _ := json.Marshal(data)
	_, _ = w.Write(b)
}

func writeAPIError(w http.ResponseWriter, statusCode int, message string, code int) {
	writeJSON(w, statusCode, map[string]interface{}{
		"error": map[string]interface{}{
			"message": message,
			"type":    "OAuthException",
			"code":    code,
		},
	})
}

func (s *Server) mediaURL(mediaID string) string {
	if s.apiVersion != "" {
		return fmt.Sprintf("%s/%s/%s/download", s.URL(), s.apiVersion, mediaID)
	}
	return fmt.Sprintf("%s/%s/download", s.URL(), mediaID)
}

func (s *Server) qrImageURL(code string) string {
	if s.apiVersion != "" {
		return fmt.Sprintf("%s/%s/mock_qr_image/%s", s.URL(), s.apiVersion, code)
	}
	return fmt.Sprintf("%s/mock_qr_image/%s", s.URL(), code)
}

// handleGraphAPI processes incoming Graph API requests according to URL segments and methods.
func (s *Server) handleGraphAPI(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	if path == "" || path == "health" {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	parts := strings.Split(path, "/")
	// Strip API version prefix if present (e.g. "v21.0")
	if len(parts) > 0 {
		seg := parts[0]
		if seg == s.apiVersion || (len(seg) >= 2 && seg[0] == 'v' && unicode.IsDigit(rune(seg[1]))) {
			parts = parts[1:]
		}
	}

	if len(parts) == 0 {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	switch len(parts) {
	case 1:
		s.handleSingleSegment(w, r, parts[0])
	case 2:
		s.handleTwoSegments(w, r, parts[0], parts[1])
	case 3:
		s.handleThreeSegments(w, r, parts[0], parts[1], parts[2])
	default:
		writeAPIError(w, http.StatusNotFound, fmt.Sprintf("Unknown route /%s", path), 100)
	}
}

func (s *Server) handleSingleSegment(w http.ResponseWriter, r *http.Request, id string) {
	switch r.Method {
	case http.MethodGet:
		s.state.mu.RLock()
		defer s.state.mu.RUnlock()

		// 1. Check Media store
		if media, ok := s.state.mediaStore[id]; ok && media != nil {
			writeJSON(w, http.StatusOK, &wacloudapi.MediaMetadata{
				ID:               media.ID,
				URL:              s.mediaURL(media.ID),
				MimeType:         media.MIMEType,
				SHA256:           media.SHA256,
				FileSize:         media.FileSize,
				MessagingProduct: "whatsapp",
			})
			return
		}

		// 2. Check Templates
		if tpl, ok := s.state.templates[id]; ok && tpl != nil {
			writeJSON(w, http.StatusOK, tpl)
			return
		}

		// 3. Check QR codes
		if qr, ok := s.state.qrCodes[id]; ok && qr != nil {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"data": []wacloudapi.QRCodeDetails{*qr},
			})
			return
		}

		// 4. Check Phone Numbers
		for _, phone := range s.state.phoneNumbers {
			if phone.ID == id {
				writeJSON(w, http.StatusOK, phone)
				return
			}
		}
		if id == s.phoneNumberID {
			writeJSON(w, http.StatusOK, &wacloudapi.PhoneNumberDetails{
				ID:                 id,
				DisplayPhoneNumber: "+1 555-0100",
				VerifiedName:       "Test Business",
				QualityRating:      "GREEN",
			})
			return
		}

		writeAPIError(w, http.StatusNotFound, fmt.Sprintf("Node %s not found", id), 100)

	case http.MethodPost:
		bodyBytes, _ := io.ReadAll(r.Body)

		s.state.mu.Lock()
		defer s.state.mu.Unlock()

		// Check if it's a template update (has components)
		if tpl, ok := s.state.templates[id]; ok && tpl != nil {
			var updateReq wacloudapi.UpdateTemplateRequest
			if err := json.Unmarshal(bodyBytes, &updateReq); err == nil && len(updateReq.Components) > 0 {
				tpl.Components = updateReq.Components
				writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
				return
			}
		}

		// Set Two-Step PIN or generic POST
		writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})

	case http.MethodDelete:
		s.state.mu.Lock()
		defer s.state.mu.Unlock()

		if _, ok := s.state.mediaStore[id]; ok {
			delete(s.state.mediaStore, id)
			writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
			return
		}

		if _, ok := s.state.templates[id]; ok {
			delete(s.state.templates, id)
			writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
			return
		}

		if _, ok := s.state.qrCodes[id]; ok {
			delete(s.state.qrCodes, id)
			writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
			return
		}

		writeAPIError(w, http.StatusNotFound, fmt.Sprintf("Node %s not found", id), 100)

	default:
		writeAPIError(w, http.StatusMethodNotAllowed, "Method not allowed", 100)
	}
}

func (s *Server) handleTwoSegments(w http.ResponseWriter, r *http.Request, nodeID, action string) {
	if nodeID == "mock_qr_image" {
		s.serveQRImage(w, r, action)
		return
	}

	switch action {
	case "messages":
		if r.Method != http.MethodPost {
			writeAPIError(w, http.StatusMethodNotAllowed, "Method not allowed", 100)
			return
		}
		s.handleMessages(w, r, nodeID)

	case "media":
		if r.Method != http.MethodPost {
			writeAPIError(w, http.StatusMethodNotAllowed, "Method not allowed", 100)
			return
		}
		s.handleMediaUpload(w, r, nodeID)

	case "download":
		if r.Method != http.MethodGet {
			writeAPIError(w, http.StatusMethodNotAllowed, "Method not allowed", 100)
			return
		}
		s.handleMediaDownload(w, r, nodeID)

	case "whatsapp_business_profile":
		if r.Method == http.MethodGet {
			s.handleGetBusinessProfile(w, r, nodeID)
		} else if r.Method == http.MethodPost {
			s.handleUpdateBusinessProfile(w, r, nodeID)
		} else {
			writeAPIError(w, http.StatusMethodNotAllowed, "Method not allowed", 100)
		}

	case "message_templates":
		if r.Method == http.MethodGet {
			s.handleListTemplates(w, r, nodeID)
		} else if r.Method == http.MethodPost {
			s.handleCreateTemplate(w, r, nodeID)
		} else if r.Method == http.MethodDelete {
			s.handleDeleteTemplateByName(w, r, nodeID)
		} else {
			writeAPIError(w, http.StatusMethodNotAllowed, "Method not allowed", 100)
		}

	case "phone_numbers":
		if r.Method == http.MethodGet {
			s.handleListPhoneNumbers(w, r, nodeID)
		} else {
			writeAPIError(w, http.StatusMethodNotAllowed, "Method not allowed", 100)
		}

	case "request_code", "verify_code", "register", "deregister":
		if r.Method == http.MethodPost {
			writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
		} else {
			writeAPIError(w, http.StatusMethodNotAllowed, "Method not allowed", 100)
		}

	case "message_qrdls":
		if r.Method == http.MethodGet {
			s.handleListQRCodes(w, r, nodeID)
		} else if r.Method == http.MethodPost {
			s.handleCreateQRCode(w, r, nodeID)
		} else {
			writeAPIError(w, http.StatusMethodNotAllowed, "Method not allowed", 100)
		}

	default:
		writeAPIError(w, http.StatusNotFound, fmt.Sprintf("Unknown endpoint /%s/%s", nodeID, action), 100)
	}
}

func (s *Server) handleThreeSegments(w http.ResponseWriter, r *http.Request, nodeID, resource, subID string) {
	if resource == "message_qrdls" {
		switch r.Method {
		case http.MethodGet:
			s.handleGetQRCode(w, r, nodeID, subID)
		case http.MethodPost:
			s.handleUpdateQRCode(w, r, nodeID, subID)
		case http.MethodDelete:
			s.handleDeleteQRCode(w, r, nodeID, subID)
		default:
			writeAPIError(w, http.StatusMethodNotAllowed, "Method not allowed", 100)
		}
		return
	}

	if resource == "mock_qr_image" {
		s.serveQRImage(w, r, subID)
		return
	}

	writeAPIError(w, http.StatusNotFound, fmt.Sprintf("Unknown endpoint /%s/%s/%s", nodeID, resource, subID), 100)
}

func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request, nodeID string) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "Failed to read request body", 100)
		return
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &raw); err != nil {
		writeAPIError(w, http.StatusBadRequest, "Invalid JSON body", 100)
		return
	}

	// If status is "read" (MarkAsRead)
	if status, ok := raw["status"].(string); ok && status == "read" {
		writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
		return
	}

	var sendReq wacloudapi.SendMessageRequest
	if err := json.Unmarshal(bodyBytes, &sendReq); err != nil {
		writeAPIError(w, http.StatusBadRequest, "Invalid SendMessageRequest", 100)
		return
	}

	s.state.mu.Lock()
	s.state.msgCounter++
	msgID := fmt.Sprintf("wamid.mock.%d", s.state.msgCounter)
	s.state.sentMessages = append(s.state.sentMessages, &sendReq)
	s.state.mu.Unlock()

	resp := map[string]interface{}{
		"messaging_product": "whatsapp",
		"contacts": []map[string]string{
			{
				"input": sendReq.To,
				"wa_id": sendReq.To,
			},
		},
		"messages": []map[string]string{
			{
				"id":             msgID,
				"message_status": "accepted",
			},
		},
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleMediaUpload(w http.ResponseWriter, r *http.Request, nodeID string) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeAPIError(w, http.StatusBadRequest, "Failed to parse multipart form", 100)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "Missing file parameter", 100)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Failed to read uploaded file", 100)
		return
	}

	mimeType := r.FormValue("type")
	if mimeType == "" {
		mimeType = header.Header.Get("Content-Type")
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	sum := sha256.Sum256(data)
	sha256Hex := hex.EncodeToString(sum[:])

	s.state.mu.Lock()
	s.state.mediaCounter++
	mediaID := fmt.Sprintf("mock_media_%d", s.state.mediaCounter)
	mockMedia := &MockMedia{
		ID:       mediaID,
		MIMEType: mimeType,
		Filename: header.Filename,
		Data:     data,
		FileSize: int64(len(data)),
		SHA256:   sha256Hex,
	}
	s.state.mediaStore[mediaID] = mockMedia
	s.state.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]string{
		"id": mediaID,
	})
}

func (s *Server) handleMediaDownload(w http.ResponseWriter, r *http.Request, mediaID string) {
	s.state.mu.RLock()
	media, ok := s.state.mediaStore[mediaID]
	s.state.mu.RUnlock()

	if !ok || media == nil {
		writeAPIError(w, http.StatusNotFound, fmt.Sprintf("Media %s not found", mediaID), 100)
		return
	}

	w.Header().Set("Content-Type", media.MIMEType)
	w.Header().Set("Content-Length", strconv.Itoa(len(media.Data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(media.Data)
}

func (s *Server) handleGetBusinessProfile(w http.ResponseWriter, r *http.Request, nodeID string) {
	s.state.mu.RLock()
	prof := *s.state.profile
	s.state.mu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": []wacloudapi.BusinessProfile{prof},
	})
}

func (s *Server) handleUpdateBusinessProfile(w http.ResponseWriter, r *http.Request, nodeID string) {
	var req wacloudapi.UpdateBusinessProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "Invalid request body", 100)
		return
	}

	s.state.mu.Lock()
	if req.About != "" {
		s.state.profile.About = req.About
	}
	if req.Address != "" {
		s.state.profile.Address = req.Address
	}
	if req.Description != "" {
		s.state.profile.Description = req.Description
	}
	if req.Email != "" {
		s.state.profile.Email = req.Email
	}
	if len(req.Websites) > 0 {
		s.state.profile.Websites = req.Websites
	}
	if req.Vertical != "" {
		s.state.profile.Vertical = req.Vertical
	}
	s.state.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func (s *Server) handleCreateTemplate(w http.ResponseWriter, r *http.Request, nodeID string) {
	var req wacloudapi.CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "Invalid request body", 100)
		return
	}

	s.state.mu.Lock()
	s.state.tplCounter++
	tplID := fmt.Sprintf("mock_tpl_%d", s.state.tplCounter)
	category := req.Category
	if category == "" {
		category = wacloudapi.TemplateCategoryMarketing
	}
	tpl := &wacloudapi.TemplateDetails{
		ID:         tplID,
		Name:       req.Name,
		Status:     wacloudapi.TemplateStatusApproved,
		Category:   category,
		Language:   req.Language,
		Components: req.Components,
	}
	s.state.templates[tplID] = tpl
	s.state.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":       tplID,
		"status":   wacloudapi.TemplateStatusApproved,
		"category": category,
	})
}

func (s *Server) handleListTemplates(w http.ResponseWriter, r *http.Request, nodeID string) {
	category := r.URL.Query().Get("category")
	status := r.URL.Query().Get("status")
	name := r.URL.Query().Get("name")

	s.state.mu.RLock()
	var list []wacloudapi.TemplateDetails
	for _, tpl := range s.state.templates {
		if category != "" && string(tpl.Category) != category {
			continue
		}
		if status != "" && string(tpl.Status) != status {
			continue
		}
		if name != "" && tpl.Name != name {
			continue
		}
		list = append(list, *tpl)
	}
	s.state.mu.RUnlock()

	if list == nil {
		list = make([]wacloudapi.TemplateDetails, 0)
	}

	resp := map[string]interface{}{
		"data": list,
		"paging": map[string]interface{}{
			"cursors": map[string]string{
				"before": "mock_before_cursor",
				"after":  "mock_after_cursor",
			},
		},
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleDeleteTemplateByName(w http.ResponseWriter, r *http.Request, nodeID string) {
	name := r.URL.Query().Get("name")
	if name == "" {
		writeAPIError(w, http.StatusBadRequest, "Template name parameter is required", 100)
		return
	}

	s.state.mu.Lock()
	for id, tpl := range s.state.templates {
		if tpl.Name == name {
			delete(s.state.templates, id)
		}
	}
	s.state.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func (s *Server) handleListPhoneNumbers(w http.ResponseWriter, r *http.Request, nodeID string) {
	s.state.mu.RLock()
	phones := make([]wacloudapi.PhoneNumberDetails, len(s.state.phoneNumbers))
	copy(phones, s.state.phoneNumbers)
	s.state.mu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": phones,
	})
}

func (s *Server) handleCreateQRCode(w http.ResponseWriter, r *http.Request, nodeID string) {
	var req wacloudapi.CreateQRCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "Invalid request body", 100)
		return
	}

	s.state.mu.Lock()
	s.state.qrCounter++
	code := fmt.Sprintf("mock_qr_%d", s.state.qrCounter)
	deepLink := fmt.Sprintf("https://wa.me/message/%s", code)
	imageURL := s.qrImageURL(code)

	qr := &wacloudapi.QRCodeDetails{
		Code:             code,
		PrefilledMessage: req.PrefilledMessage,
		DeepLinkURL:      deepLink,
		QRImageURL:       imageURL,
	}
	s.state.qrCodes[code] = qr
	s.state.mu.Unlock()

	writeJSON(w, http.StatusOK, qr)
}

func (s *Server) handleListQRCodes(w http.ResponseWriter, r *http.Request, nodeID string) {
	s.state.mu.RLock()
	var list []wacloudapi.QRCodeDetails
	for _, qr := range s.state.qrCodes {
		list = append(list, *qr)
	}
	s.state.mu.RUnlock()

	if list == nil {
		list = make([]wacloudapi.QRCodeDetails, 0)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": list,
	})
}

func (s *Server) handleGetQRCode(w http.ResponseWriter, r *http.Request, nodeID, subID string) {
	s.state.mu.RLock()
	qr, ok := s.state.qrCodes[subID]
	s.state.mu.RUnlock()

	if !ok || qr == nil {
		writeAPIError(w, http.StatusNotFound, fmt.Sprintf("QR code %s not found", subID), 100)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": []wacloudapi.QRCodeDetails{*qr},
	})
}

func (s *Server) handleUpdateQRCode(w http.ResponseWriter, r *http.Request, nodeID, subID string) {
	var body struct {
		PrefilledMessage string `json:"prefilled_message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "Invalid request body", 100)
		return
	}

	s.state.mu.Lock()
	qr, ok := s.state.qrCodes[subID]
	if !ok || qr == nil {
		s.state.mu.Unlock()
		writeAPIError(w, http.StatusNotFound, fmt.Sprintf("QR code %s not found", subID), 100)
		return
	}
	qr.PrefilledMessage = body.PrefilledMessage
	updated := *qr
	s.state.mu.Unlock()

	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDeleteQRCode(w http.ResponseWriter, r *http.Request, nodeID, subID string) {
	s.state.mu.Lock()
	_, ok := s.state.qrCodes[subID]
	if !ok {
		s.state.mu.Unlock()
		writeAPIError(w, http.StatusNotFound, fmt.Sprintf("QR code %s not found", subID), 100)
		return
	}
	delete(s.state.qrCodes, subID)
	s.state.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func (s *Server) serveQRImage(w http.ResponseWriter, r *http.Request, code string) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" width="200" height="200"><rect width="200" height="200" fill="#000"/><text x="20" y="100" fill="#fff" font-size="14">Mock QR Code</text></svg>`
	w.Header().Set("Content-Type", "image/svg+xml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(svg))
}
