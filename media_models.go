package wacloudapi

type UploadMediaResponse struct {
	ID string `json:"id"`
}

type MediaMetadata struct {
	ID               string `json:"id"`
	URL              string `json:"url"`
	MimeType         string `json:"mime_type"`
	SHA256           string `json:"sha256"`
	FileSize         int64  `json:"file_size"`
	MessagingProduct string `json:"messaging_product"`
}
