package wacloudapitest

// Option defines a functional configuration option for Server.
type Option func(*Server)

// WithPhoneNumberID sets the simulated Phone Number ID for the mock server.
func WithPhoneNumberID(phoneNumberID string) Option {
	return func(s *Server) {
		if phoneNumberID != "" {
			s.phoneNumberID = phoneNumberID
		}
	}
}

// WithWABAID sets the simulated WhatsApp Business Account (WABA) ID for the mock server.
func WithWABAID(wabaID string) Option {
	return func(s *Server) {
		if wabaID != "" {
			s.wabaID = wabaID
		}
	}
}

// WithAPIVersion sets the Meta Graph API version for the mock server (e.g. "v21.0").
func WithAPIVersion(apiVersion string) Option {
	return func(s *Server) {
		if apiVersion != "" {
			s.apiVersion = apiVersion
		}
	}
}

// WithAppSecret sets the Meta App Secret used for HMAC-SHA256 signature generation in webhook simulation.
func WithAppSecret(appSecret string) Option {
	return func(s *Server) {
		if appSecret != "" {
			s.appSecret = appSecret
		}
	}
}

// WithAccessToken sets the simulated access token accepted/provided by the mock server.
func WithAccessToken(accessToken string) Option {
	return func(s *Server) {
		if accessToken != "" {
			s.accessToken = accessToken
		}
	}
}
