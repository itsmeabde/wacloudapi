package flows

// EncryptedPayload represents the raw encrypted JSON body received from the WhatsApp Flows data endpoint.
type EncryptedPayload struct {
	EncryptedAESKey   string `json:"encrypted_aes_key"`
	EncryptedFlowData string `json:"encrypted_flow_data"`
	InitialVector     string `json:"initial_vector"`
}

// Request represents the decrypted WhatsApp Flows request payload.
type Request struct {
	Version      string                 `json:"version"`
	Action       string                 `json:"action"`
	Screen       string                 `json:"screen,omitempty"`
	Data         map[string]interface{} `json:"data,omitempty"`
	FlowToken    string                 `json:"flow_token,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	ErrorCode    string                 `json:"error_code,omitempty"`
}

// Response represents the unencrypted response payload to be sent back to WhatsApp Flows.
type Response struct {
	Version string                 `json:"version,omitempty"`
	Screen  string                 `json:"screen,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Error   *FlowError             `json:"error,omitempty"`
}

// FlowError represents error details inside a WhatsApp Flows response.
type FlowError struct {
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

// CryptoSession stores the decrypted AES key and initial vector for encrypting the corresponding response.
type CryptoSession struct {
	AESKey        []byte
	InitialVector []byte
}
