package httpclient

import (
	"net/http"
	"time"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	headers    map[string]string
}

type Config struct {
	Timeout    time.Duration
	BaseURL    string
	Headers    map[string]string
	RetryCount int
}

func New(cfg Config) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		baseURL: cfg.BaseURL,
		headers: cfg.Headers,
	}
}
