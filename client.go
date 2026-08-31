package wacloudapi

type Client struct {
	config   *Config
	Messages *MessagesService
}

func New(accessToken, phoneNumberID string, opts ...Option) *Client {
	cfg := defaultConfig(accessToken, phoneNumberID)
	for _, opt := range opts {
		opt(cfg)
	}

	c := &Client{
		config: cfg,
	}
	c.Messages = newMessagesService(c)

	return c
}
