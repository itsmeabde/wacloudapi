package wacloudapi

type QRCodeImageFormat string

const (
	QRCodeImageSVG QRCodeImageFormat = "SVG"
	QRCodeImagePNG QRCodeImageFormat = "PNG"
)

type CreateQRCodeRequest struct {
	PrefilledMessage string            `json:"prefilled_message"`
	ImageFormat      QRCodeImageFormat `json:"generate_qr_image,omitempty"`
}

type QRCodeDetails struct {
	Code             string `json:"code"`
	PrefilledMessage string `json:"prefilled_message"`
	DeepLinkURL      string `json:"deep_link_url"`
	QRImageURL       string `json:"qr_image_url"`
}

type ListQRCodesResponse struct {
	Data []QRCodeDetails `json:"data"`
}
